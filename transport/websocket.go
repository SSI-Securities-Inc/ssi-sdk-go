package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/redact"
	"github.com/gorilla/websocket"
)

var wsLog = logger.New("ssi_sdk.transport.websocket")

const (
	wsThreadJoinTimeout = 5 * time.Second
	// A connection that lasted at least this long counts as healthy: the backoff starts over.
	wsStableMs = 10000
)

// MessageHandler is a callback for WebSocket messages.
type MessageHandler func(msg map[string]interface{})

// TokenProvider returns the bearer token for the next connect. forceRefresh is true after the
// server rejected the token and when cycling for expiry. An *ssi.AuthenticationError it returns
// is fatal (the credentials need a new OTP); any other error counts as a failed attempt.
type TokenProvider func(forceRefresh bool) (string, error)

// wsConn is the part of *websocket.Conn the client uses (tests inject a fake).
type wsConn interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(messageType int, data []byte) error
	Close() error
}

// dialFunc opens one socket. status is the HTTP status of a refused handshake (0 when unknown).
type dialFunc func(rawURL string, header http.Header) (conn wsConn, status int, err error)

// WSOption tunes a WebSocketClient.
type WSOption func(*WebSocketClient)

// WithTokenProvider sets the provider called before every connect. Without it the token from
// SetToken is used.
func WithTokenProvider(p TokenProvider) WSOption {
	return func(c *WebSocketClient) { c.tokenProvider = p }
}

// WithExpiryProvider sets the function returning the epoch second when the credentials stop
// working (the earlier of access and refresh token); 0 means unknown.
func WithExpiryProvider(p func() float64) WSOption {
	return func(c *WebSocketClient) { c.expiryProvider = p }
}

// WithAutoReconnect overrides Config.AutoReconnect for this client.
func WithAutoReconnect(on bool) WSOption {
	return func(c *WebSocketClient) { c.autoReconnect = on }
}

// session is one socket and what the server said on it.
type session struct {
	conn      wsConn
	rejected  chan struct{}
	mu        sync.Mutex
	rejection map[string]interface{}
}

func (s *session) rejectionFrame() map[string]interface{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rejection
}

// WebSocketClient is the WebSocket client for SSI real-time streaming.
//
// Connect starts a supervisor goroutine that keeps one socket open: it re-opens the socket after
// a drop (exponential backoff with jitter, capped), always with a fresh token from the
// TokenProvider, re-opens it ahead of the token's expiry (the server never closes a socket whose
// JWT expired, it only checks at connect), and re-sends every subscription. The old socket is
// always closed before a new one opens: the server allows only 10 per client and a leak would
// exhaust that quota.
type WebSocketClient struct {
	config         *ssi.Config
	tokenProvider  TokenProvider
	expiryProvider func() float64
	autoReconnect  bool

	dial   dialFunc
	random func() float64
	clock  func() int64 // epoch milliseconds
	sleep  func(ms int, stop <-chan struct{}) bool

	mu         sync.Mutex
	token      string
	sess       *session
	stop       chan struct{}
	done       chan struct{}
	lastErr    error
	reconnects int
	onState    func(ConnectionEvent)

	writeMu sync.Mutex

	handlerMu sync.RWMutex
	handlers  map[string][]MessageHandler

	book *SubscriptionBook
}

// NewWebSocketClient builds a client. It never mutates config.
func NewWebSocketClient(config *ssi.Config, opts ...WSOption) *WebSocketClient {
	c := &WebSocketClient{
		config:        config,
		autoReconnect: config.AutoReconnect,
		handlers:      make(map[string][]MessageHandler),
		book:          NewSubscriptionBook(),
		random:        defaultRandom,
		clock:         func() int64 { return time.Now().UnixMilli() },
		sleep:         sleepOrStop,
	}
	c.dial = c.defaultDial
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func sleepOrStop(ms int, stop <-chan struct{}) bool {
	t := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer t.Stop()
	select {
	case <-t.C:
		return true
	case <-stop:
		return false
	}
}

func (c *WebSocketClient) defaultDial(rawURL string, header http.Header) (wsConn, int, error) {
	dialer := &websocket.Dialer{HandshakeTimeout: time.Duration(c.config.TimeoutMs) * time.Millisecond}
	if c.config.Proxy != "" {
		proxyURL, err := url.Parse(c.config.Proxy)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid proxy URL")
		}
		dialer.Proxy = http.ProxyURL(proxyURL)
	}
	conn, resp, err := dialer.Dial(rawURL, header)
	status := 0
	if resp != nil {
		status = resp.StatusCode
	}
	if err != nil {
		return nil, status, err
	}
	return conn, status, nil
}

// SetToken sets a static bearer token (used when there is no TokenProvider).
func (c *WebSocketClient) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

// SetOnState registers the connection-state callback (nil clears it). It runs on the SDK's own
// goroutine: keep it short and do not call Disconnect from it.
func (c *WebSocketClient) SetOnState(fn func(ConnectionEvent)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onState = fn
}

// LastError is the error that ended the supervisor (a connection that could not be recovered).
func (c *WebSocketClient) LastError() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastErr
}

// ReconnectCount is how many times the socket was re-opened after the first connect.
func (c *WebSocketClient) ReconnectCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reconnects
}

func (c *WebSocketClient) emit(state string, err error) {
	c.mu.Lock()
	handler := c.onState
	count := c.reconnects
	c.mu.Unlock()
	if handler == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			wsLog.Error("Error in the on_connection handler: %s", redact.MaskBearer(fmt.Sprint(r)))
		}
	}()
	handler(ConnectionEvent{State: state, ReconnectCount: count, Err: err})
}

// requireWSS refuses to open a socket that is not TLS-protected: the bearer token travels in it.
func requireWSS(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || strings.ToLower(u.Scheme) != "wss" {
		return ssi.NewWebSocketError(fmt.Sprintf("streaming URL must use wss://, got %q", redact.SafeURL(rawURL)))
	}
	return nil
}

func authFailure(status int) error {
	return ssi.NewAuthenticationError(
		fmt.Sprintf("WebSocket authentication failed after refreshing the token (%d)", status),
		fmt.Sprint(status), status, nil)
}

// Connect opens the WebSocket and keeps it open. It returns once the first connection is
// established; later drops are handled in the background.
func (c *WebSocketClient) Connect() error {
	return c.ConnectContext(context.Background())
}

// ConnectContext is Connect that also stops waiting when ctx ends (the socket is then closed).
func (c *WebSocketClient) ConnectContext(ctx context.Context) error {
	c.mu.Lock()
	if c.done != nil {
		select {
		case <-c.done:
		default:
			c.mu.Unlock()
			return nil
		}
	}
	if err := requireWSS(c.config.StreamingURL); err != nil {
		c.mu.Unlock()
		return err
	}
	stop := make(chan struct{})
	done := make(chan struct{})
	ready := make(chan error, 1)
	c.stop, c.done, c.lastErr = stop, done, nil
	c.mu.Unlock()

	go c.supervise(stop, done, ready)

	select {
	case err := <-ready:
		if err != nil {
			c.Disconnect()
			return err
		}
		return nil
	case <-ctx.Done():
		c.Disconnect()
		return ssi.NewWebSocketError("Connect cancelled: " + ctx.Err().Error())
	}
}

// Disconnect stops the supervisor, closes the socket and forgets the subscriptions.
func (c *WebSocketClient) Disconnect() {
	c.mu.Lock()
	stop, done := c.stop, c.done
	c.stop = nil
	sess := c.sess
	c.mu.Unlock()

	if stop != nil {
		close(stop)
	}
	if sess != nil {
		_ = sess.conn.Close()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(wsThreadJoinTimeout):
		}
	}
	c.mu.Lock()
	c.sess = nil
	c.done = nil
	c.mu.Unlock()
	c.book.Clear()
	wsLog.Info("WebSocket disconnected")
}

// On registers the handler for a channel (replacing any earlier one); nil clears it.
func (c *WebSocketClient) On(channel string, handler MessageHandler) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	if handler == nil {
		delete(c.handlers, channel)
		return
	}
	c.handlers[channel] = []MessageHandler{handler}
}

// Off clears the handler of a channel (only one is kept per channel).
func (c *WebSocketClient) Off(channel string, _ MessageHandler) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	delete(c.handlers, channel)
}

// Send sends a message (safe from any goroutine). A successful subscribe/unsubscribe is
// remembered so it is re-sent after a reconnect. A request that names many topics is split so
// each frame stays under the server's size limit.
func (c *WebSocketClient) Send(data map[string]interface{}) error {
	c.mu.Lock()
	sess := c.sess
	c.mu.Unlock()
	if sess == nil {
		return ssi.NewWebSocketError("Not connected. Call Connect() first.")
	}
	if err := c.writeParts(sess.conn, data); err != nil {
		return ssi.NewWebSocketError(fmt.Sprintf("Failed to send message: %v", err))
	}
	c.book.Record(data)
	return nil
}

func (c *WebSocketClient) writeParts(conn wsConn, data map[string]interface{}) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	for _, part := range SplitRequest(data, ssi.WSMaxFrameBytes) {
		b, err := json.Marshal(part)
		if err != nil {
			return err
		}
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			return err
		}
	}
	return nil
}

// ── supervisor ────────────────────────────────────────────

func (c *WebSocketClient) currentToken(forceRefresh bool) (string, error) {
	if c.tokenProvider != nil {
		return c.tokenProvider(forceRefresh)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.token, nil
}

func (c *WebSocketClient) backoff(attempt int) int {
	return BackoffDelayMs(attempt, c.config.RetryDelayMs, ssi.WSReconnectMaxDelayMs, c.random())
}

// open opens one socket with a freshly obtained token.
func (c *WebSocketClient) open(forceRefresh bool) (*session, int, error) {
	token, err := c.currentToken(forceRefresh)
	if err != nil {
		return nil, 0, err
	}
	header := http.Header{}
	header.Set(HeaderContentType, ContentTypeJSON)
	header.Set(HeaderAccept, ContentTypeJSON)
	if token != "" {
		header.Set(HeaderAuthorization, AuthSchemeBearer+token)
	}
	wsLog.Debug("Connecting to WebSocket %s", redact.SafeURL(c.config.StreamingURL))
	conn, status, err := c.dial(c.config.StreamingURL, header)
	if err != nil {
		return nil, status, err
	}
	sess := &session{conn: conn, rejected: make(chan struct{}, 1)}
	c.mu.Lock()
	c.sess = sess
	c.mu.Unlock()
	wsLog.Info("WebSocket connected to %s", redact.SafeURL(c.config.StreamingURL))
	return sess, 0, nil
}

func (c *WebSocketClient) closeSession(sess *session) {
	c.mu.Lock()
	if c.sess == sess {
		c.sess = nil
	}
	c.mu.Unlock()
	if sess != nil {
		_ = sess.conn.Close()
	}
}

func (c *WebSocketClient) resubscribe(conn wsConn) error {
	for _, request := range c.book.Requests() {
		if err := c.writeParts(conn, request); err != nil {
			return err
		}
	}
	return nil
}

// serve runs one socket until it closes, the server rejects it, or it must be cycled. It returns
// "closed", "rejected", "refresh" or "stop".
func (c *WebSocketClient) serve(sess *session, stop <-chan struct{}) string {
	listenDone := make(chan struct{})
	go func() {
		defer close(listenDone)
		c.listen(sess)
	}()
	var refresh <-chan time.Time
	var expires float64
	if c.expiryProvider != nil {
		expires = c.expiryProvider()
	}
	if ms, ok := MsUntilRefresh(int64(expires*1000), c.clock(), ssi.WSTokenRefreshLeadMs); ok {
		timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
		defer timer.Stop()
		refresh = timer.C
	}
	reason := ""
	select {
	case <-listenDone:
		reason = "closed"
	case <-sess.rejected:
		reason = "rejected"
	case <-refresh:
		reason = "refresh"
	case <-stop:
		reason = "stop"
	}
	// A refusal frame is followed by a close; report the cause, not the symptom.
	if reason == "closed" && sess.rejectionFrame() != nil {
		reason = "rejected"
	}
	return reason
}

func (c *WebSocketClient) fail(err error, ready chan<- error, pending *bool) {
	c.mu.Lock()
	c.lastErr = err
	c.mu.Unlock()
	wsLog.Error("WebSocket stopped: %T", err)
	c.emit("failed", err)
	if *pending {
		*pending = false
		ready <- err
	}
}

func (c *WebSocketClient) supervise(stop <-chan struct{}, done chan<- struct{}, ready chan<- error) {
	defer close(done)
	pending := true
	attempt := 0
	authRetried := false
	forceRefresh := false
	stopped := func() bool {
		select {
		case <-stop:
			return true
		default:
			return false
		}
	}
	var current *session
	defer func() { c.closeSession(current) }()

	for !stopped() {
		sess, status, err := c.open(forceRefresh)
		if err != nil {
			var authErr *ssi.AuthenticationError
			if errors.As(err, &authErr) {
				c.fail(err, ready, &pending) // cannot be fixed by retrying: needs a new OTP
				return
			}
			if status == 401 || status == 403 {
				if authRetried {
					c.fail(authFailure(status), ready, &pending)
					return
				}
				authRetried, forceRefresh = true, true
				continue
			}
			wait := c.backoff(attempt)
			if status == 429 {
				wait = RateLimitWaitMs(nil)
			}
			attempt++
			if pending && attempt >= maxInt(1, c.config.MaxRetries) {
				c.fail(ssi.NewWebSocketError(fmt.Sprintf("Failed to connect to %s after %d attempts: %T",
					redact.SafeURL(c.config.StreamingURL), attempt, err)), ready, &pending)
				return
			}
			wsLog.Info("WebSocket connect attempt %d failed (%T); retrying in %d ms", attempt, err, wait)
			if !c.sleep(wait, stop) {
				return
			}
			continue
		}
		current = sess
		openedAt := c.clock()
		forceRefresh = false
		if err := c.resubscribe(sess.conn); err != nil {
			wsLog.Info("Could not restore subscriptions")
		}
		if pending {
			pending = false
			ready <- nil
			c.emit("connected", nil)
		} else {
			c.mu.Lock()
			c.reconnects++
			n := c.reconnects
			c.mu.Unlock()
			wsLog.Info("WebSocket reconnected (#%d)", n)
			c.emit("reconnected", nil)
		}

		reason := c.serve(sess, stop)
		c.closeSession(sess)
		current = nil
		if reason == "stop" || stopped() {
			return
		}
		c.emit("disconnected", nil)
		stable := c.clock()-openedAt > wsStableMs
		if stable {
			attempt = 0
		} else {
			attempt++
		}

		var wait int
		switch reason {
		case "refresh":
			forceRefresh = true // the old token is about to expire: get a new one
			continue
		case "rejected":
			kind, retryAfter := ClassifyRejection(sess.rejectionFrame())
			switch kind {
			case RejectionAuth:
				if authRetried {
					c.fail(authFailure(401), ready, &pending)
					return
				}
				authRetried, forceRefresh = true, true
				continue
			case RejectionRateLimited:
				wait = RateLimitWaitMs(retryAfter)
			default:
				wait = c.backoff(attempt)
			}
		default:
			authRetried = false
			if !c.autoReconnect {
				return
			}
			wait = c.backoff(attempt)
		}
		wsLog.Info("WebSocket closed; reconnecting in %d ms", wait)
		if !c.sleep(wait, stop) {
			return
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── receive ───────────────────────────────────────────────

func (c *WebSocketClient) listen(sess *session) {
	for {
		_, raw, err := sess.conn.ReadMessage()
		if err != nil {
			wsLog.Info("WebSocket connection closed")
			return
		}
		message := DecodeMessage(raw)
		if message == nil {
			continue
		}
		if hasRoute(RoutesOf(message), ssi.WSRouteError) {
			sess.mu.Lock()
			sess.rejection = message
			sess.mu.Unlock()
			select {
			case sess.rejected <- struct{}{}:
			default:
			}
		}
		c.dispatch(message)
	}
}

// dispatch hands a decoded frame to every handler on its routes. A handler that panics is
// logged and skipped: it must not end the listen loop.
func (c *WebSocketClient) dispatch(message map[string]interface{}) {
	if category := AckErrorCategory(message); category != "" {
		wsLog.Info("Server refused a %v request on %v: %s", message["method"], message["channel"], category)
	}
	for _, route := range RoutesOf(message) {
		c.handlerMu.RLock()
		handlers := append([]MessageHandler(nil), c.handlers[route]...)
		c.handlerMu.RUnlock()
		for _, handler := range handlers {
			func() {
				defer func() {
					if r := recover(); r != nil {
						wsLog.Error("Error in handler for route '%s': %s", route, redact.MaskBearer(fmt.Sprint(r)))
					}
				}()
				handler(message)
			}()
		}
	}
}

// Wait blocks until the supervisor ends (Disconnect or an unrecoverable error) or timeoutMs
// milliseconds pass (nil waits forever).
func (c *WebSocketClient) Wait(timeoutMs *int) {
	c.mu.Lock()
	done := c.done
	c.mu.Unlock()
	if done == nil {
		return
	}
	if timeoutMs == nil {
		<-done
		return
	}
	select {
	case <-done:
	case <-time.After(time.Duration(*timeoutMs) * time.Millisecond):
	}
}

// IsConnected reports whether a socket is currently open and the supervisor is running.
func (c *WebSocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sess == nil || c.done == nil {
		return false
	}
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

// defaultRandom is jitter for reconnect delays; it is not security-relevant.
func defaultRandom() float64 { return rand.Float64() }
