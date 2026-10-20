// Package tradingws is a client for the trading WebSocket (/ws/v3/trading): orders over a
// socket.
//
// It is an explicit opt-in: the Trading / Stream facades never connect to it. Open it with
// TradingWS (from an authenticated Auth) or build a Client and a Service yourself.
//
// Frame format:
//
//	{"id":"<non-empty>","method":"order.place","params":{...},"signature":"<base64 or hex>","timestamp":<epoch ms>}
//
// Rules enforced here:
//
//   - params is serialized once; that exact text is signed (RSA-SHA256, base64 by default, see
//     WithSignatureEncoding) and inserted verbatim into the frame. Re-serializing it after
//     signing changes the bytes and the server answers 401013.
//   - Commands that change state (orders, FCO place/cancel) need a private key; without one
//     nothing is sent (fail closed). Queries are not signed.
//   - One id can receive several frames (an immediate PD ack, later a status 200 frame with the
//     order event), so responses are delivered through a per-id channel (Call).
//   - A command that times out is never retried: it may already have been accepted. Reconcile
//     through the order.<account> event stream or query.orderBook.
//   - Only wss://. The bearer token goes in the header; params, signatures, tokens and account
//     numbers are never logged and never carried in an error.
//   - The server accepts the socket first (HTTP 101) and refuses afterwards with a
//     {"code": 401|403|429|500, "msg": ...} frame, so Connect waits a short grace period for it.
package tradingws

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/redact"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/signature"
)

var wsLog = logger.New("ssi_sdk.transport.trading_ws")

// SignatureEncoding is what the caller may ask for.
type SignatureEncoding string

const (
	// EncodingAuto signs as base64 (what the public documentation states) and, if the server
	// refuses a command's signature (401006/401008/401013: the command was not executed), sends
	// it once more as hex (what REST uses) and keeps whichever works.
	EncodingAuto   SignatureEncoding = "auto"
	EncodingBase64 SignatureEncoding = "base64"
	EncodingHex    SignatureEncoding = "hex"
)

// mutatingMethods need a signature (matched case-insensitively).
var mutatingMethods = map[string]bool{
	"order.place": true, "order.amend": true, "order.cancel": true,
	"order.batchnew": true, "order.batchcancel": true,
	"fco.place": true, "fco.cancel": true,
}

// IsMutating reports whether the server requires a signature for method.
func IsMutating(method string) bool { return mutatingMethods[strings.ToLower(method)] }

var signatureCodes = map[string]bool{"401006": true, "401008": true, "401013": true}

// Response is a success frame: {"id","method","status":200,"result","rateLimit",...}.
type Response struct {
	ID           string
	Method       string
	Status       int
	Result       interface{}
	RateLimit    map[string]interface{}
	ReceivedTime interface{}
}

// ResponseFromFrame builds a response from a decoded frame; every key is optional.
func ResponseFromFrame(frame map[string]interface{}) *Response {
	r := &Response{
		ID: str(frame["id"]), Method: str(frame["method"]), Status: statusOf(frame, 200),
		Result: frame["result"], ReceivedTime: frame["receivedTime"], RateLimit: map[string]interface{}{},
	}
	if rl, ok := frame["rateLimit"].(map[string]interface{}); ok {
		r.RateLimit = rl
	}
	return r
}

func str(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	}
	return fmt.Sprint(v)
}

// statusOf reads frame["status"]: absent means def, anything unparsable (e.g. "error") means 0.
func statusOf(frame map[string]interface{}, def int) int {
	v, ok := frame["status"]
	if !ok {
		return def
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
			return n
		}
	}
	return 0
}

// marshalParams serializes params once, compactly and without HTML escaping. A json.RawMessage is
// used as is (it must already be valid JSON).
func marshalParams(params interface{}) (string, error) {
	if params == nil {
		return "{}", nil
	}
	if raw, ok := params.(json.RawMessage); ok {
		if !json.Valid(raw) {
			return "", ssi.NewValidationError("params is not valid JSON")
		}
		return string(raw), nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(params); err != nil {
		return "", ssi.NewValidationError("params cannot be encoded as JSON")
	}
	return strings.TrimSuffix(buf.String(), "\n"), nil
}

func jsonString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(buf.String(), "\n")
}

// BuildCommand builds the text of one command frame. paramsText is inserted verbatim and, for a
// mutating method, signed as is.
func BuildCommand(method, paramsText, privateKey, requestID string, timestampMs int64, encoding string) (string, error) {
	if requestID == "" {
		return "", ssi.NewWebSocketError("a command needs a non-empty id")
	}
	if !json.Valid([]byte(paramsText)) {
		return "", ssi.NewValidationError("params is not valid JSON")
	}
	sigPart := ""
	if IsMutating(method) {
		if privateKey == "" {
			return "", ssi.NewWebSocketError("private_key is required for trading commands that change state")
		}
		sig, err := signature.SignWithEncoding(paramsText, privateKey, encoding)
		if err != nil {
			return "", err
		}
		sigPart = `,"signature":` + jsonString(sig)
	}
	return `{"id":` + jsonString(requestID) + `,"method":` + jsonString(method) + `,"params":` +
		paramsText + sigPart + `,"timestamp":` + strconv.FormatInt(timestampMs, 10) + `}`, nil
}

// Unwrap returns a success frame, or a *ssi.TradingWSError for an error frame. Error frames look
// like {"id","status": 4xx | "error","error":{"code","message"}} (fco.* answers status "error"
// with error.code "FCO_ERROR"). encodingHint ("" for none) adds a hint to signature refusals.
func Unwrap(frame map[string]interface{}, encodingHint string) (map[string]interface{}, error) {
	errVal := frame["error"]
	status := statusOf(frame, 200)
	hasErr := false
	switch t := errVal.(type) {
	case nil:
	case string:
		hasErr = t != ""
	case map[string]interface{}:
		hasErr = len(t) > 0
	case bool:
		hasErr = t
	default:
		hasErr = true
	}
	if !hasErr && status < 400 {
		return frame, nil
	}
	detail, _ := errVal.(map[string]interface{})
	code, message := str(detail["code"]), str(detail["message"])
	if encodingHint != "" && signatureCodes[code] {
		other := "base64"
		if encodingHint == "base64" {
			other = "hex"
		}
		message += fmt.Sprintf(" (signature rejected: signed as %s; if every signed command fails this way pass WithSignatureEncoding(%q) to the trading WS client)", encodingHint, other)
	}
	return nil, ssi.NewTradingWSError(status, code, message)
}

func bothRefused(e *ssi.TradingWSError) *ssi.TradingWSError {
	return ssi.NewTradingWSError(e.Status, e.ErrorCode,
		e.ServerMessage+" (refused with a base64 and a hex signature: check that Config.PrivateKey is the key registered for this API key)")
}

// IsRejection reports a post-upgrade refusal: {"code": 401|403|429|500, "msg": ...} with no id.
func IsRejection(frame map[string]interface{}) bool {
	_, hasID := frame["id"]
	_, hasCode := frame["code"]
	_, hasMsg := frame["msg"]
	_, hasMessage := frame["message"]
	return !hasID && hasCode && (hasMsg || hasMessage)
}

// RejectionError is the error for a refusal frame (401 token, 403 missing tradingws:*:*, 429).
func RejectionError(frame map[string]interface{}) *ssi.TradingWSError {
	status := 0
	switch t := frame["code"].(type) {
	case float64:
		status = int(t)
	case string:
		status, _ = strconv.Atoi(strings.TrimSpace(t))
	}
	message := str(frame["msg"])
	if message == "" {
		message = str(frame["message"])
	}
	return ssi.NewTradingWSError(status, str(frame["code"]), message)
}

func decodeFrame(raw []byte) map[string]interface{} {
	var frame map[string]interface{}
	if err := json.Unmarshal(raw, &frame); err != nil || frame == nil {
		wsLog.Error("Dropped a non-JSON trading frame (%d bytes)", len(raw))
		return nil
	}
	return frame
}

// ── options ──────────────────────────────────────────────────────────────────

type conn interface {
	WriteMessage(messageType int, data []byte) error
	ReadMessage() (int, []byte, error)
	Close() error
}

type dialFunc func(ctx context.Context, url string, header http.Header, timeout time.Duration) (conn, error)

type options struct {
	encoding         SignatureEncoding
	timeout          time.Duration
	userAgent        string
	handshakeGrace   time.Duration
	respectRateLimit bool
	maxRateLimitWait time.Duration
	pingInterval     time.Duration
	dial             dialFunc
	now              func() time.Time
	sleep            func(ctx context.Context, d time.Duration) error
}

// Option tunes the client.
type Option func(*options)

// WithSignatureEncoding fixes the encoding ("base64" / "hex") or leaves it on "auto" (default).
func WithSignatureEncoding(e SignatureEncoding) Option { return func(o *options) { o.encoding = e } }

// WithRequestTimeoutMs is how long to wait for a response before giving up (no retry), in
// milliseconds. Default 10000.
func WithRequestTimeoutMs(ms int) Option {
	return func(o *options) { o.timeout = time.Duration(ms) * time.Millisecond }
}

// WithUserAgent sets the User-Agent of the upgrade request (default: the SDK's browser-like one;
// the gateway's firewall answers 403 to a bare library one).
func WithUserAgent(ua string) Option { return func(o *options) { o.userAgent = ua } }

// WithHandshakeGrace is how long Connect waits after the upgrade for the refusal frame the server
// sends after HTTP 101, in milliseconds. 0 skips. Default 500.
func WithHandshakeGraceMs(ms int) Option {
	return func(o *options) { o.handshakeGrace = time.Duration(ms) * time.Millisecond }
}

// WithRespectRateLimit makes the client wait for a response's resetMs (at most
// WithMaxRateLimitWaitMs) once remaining hits 0, instead of getting a 429. Default true.
func WithRespectRateLimit(on bool) Option { return func(o *options) { o.respectRateLimit = on } }

// WithMaxRateLimitWaitMs bounds that wait, in milliseconds. Default 2000.
func WithMaxRateLimitWaitMs(ms int) Option {
	return func(o *options) { o.maxRateLimitWait = time.Duration(ms) * time.Millisecond }
}

// WithPingIntervalMs sends ping_pong this often (milliseconds) to notice a hung connection
// early. 0 disables. Default 30000.
func WithPingIntervalMs(ms int) Option {
	return func(o *options) { o.pingInterval = time.Duration(ms) * time.Millisecond }
}

func defaultOptions() options {
	return options{
		encoding: EncodingAuto, timeout: 10 * time.Second, handshakeGrace: 500 * time.Millisecond,
		respectRateLimit: true, maxRateLimitWait: 2 * time.Second, pingInterval: 30 * time.Second,
		dial: realDial, now: time.Now, sleep: realSleep,
	}
}

func realSleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func realDial(ctx context.Context, rawURL string, header http.Header, timeout time.Duration) (conn, error) {
	d := &websocket.Dialer{HandshakeTimeout: timeout, Proxy: http.ProxyFromEnvironment}
	c, resp, err := d.DialContext(ctx, rawURL, header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("handshake refused with HTTP %d", resp.StatusCode)
		}
		return nil, errors.New("could not open the socket")
	}
	return c, nil
}

// ── client ───────────────────────────────────────────────────────────────────

type item struct {
	frame  map[string]interface{}
	err    error
	closed bool
}

// Client is the low-level trading WebSocket client. It is safe for concurrent use.
type Client struct {
	url           string
	tokenProvider func() (string, error)
	privateKey    string
	o             options

	mu           sync.Mutex
	sendMu       sync.Mutex
	conn         conn
	done         chan struct{}
	stop         chan struct{}
	pending      map[string]chan item
	encoding     string // "base64" or "hex": what signatures use right now
	rejection    *ssi.TradingWSError
	blockedUntil time.Time
	rateLimit    map[string]interface{}
	onEvent      func(map[string]interface{})
}

// NewClient creates the client; rawURL must be wss://<domain>/ws/v3/trading
// (Config.TradingWSURL()). tokenProvider is called on every Connect. privateKey signs
// state-changing commands (Config.PrivateKey).
func NewClient(rawURL string, tokenProvider func() (string, error), privateKey string, opts ...Option) (*Client, error) {
	if err := requireWSS(rawURL); err != nil {
		return nil, err
	}
	o := defaultOptions()
	for _, fn := range opts {
		fn(&o)
	}
	switch o.encoding {
	case EncodingAuto, EncodingBase64, EncodingHex:
	default:
		return nil, ssi.NewValidationError(fmt.Sprintf("signature encoding must be auto, base64 or hex, got %q", o.encoding))
	}
	if o.timeout <= 0 {
		o.timeout = 10 * time.Second
	}
	if o.userAgent == "" {
		o.userAgent = ssi.DefaultUserAgent
	}
	enc := string(o.encoding)
	if o.encoding == EncodingAuto {
		enc = "base64"
	}
	return &Client{
		url: rawURL, tokenProvider: tokenProvider, privateKey: privateKey, o: o,
		pending: map[string]chan item{}, encoding: enc, rateLimit: map[string]interface{}{},
	}, nil
}

// requireWSS refuses a socket that is not TLS-protected: the bearer token travels in it.
func requireWSS(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil || strings.ToLower(u.Scheme) != "wss" {
		return ssi.NewWebSocketError(fmt.Sprintf("trading WebSocket URL must use wss://, got %q", redact.SafeURL(rawURL)))
	}
	return nil
}

// SetOnEvent sets the handler for frames that carry no command id (subscription acks, order and
// portfolio events). Pass nil to clear it. A panic in the handler is recovered and logged.
func (c *Client) SetOnEvent(fn func(map[string]interface{})) {
	c.mu.Lock()
	c.onEvent = fn
	c.mu.Unlock()
}

// IsConnected reports whether the socket is open and the reader is running.
func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectedLocked()
}

func (c *Client) connectedLocked() bool {
	if c.conn == nil || c.done == nil {
		return false
	}
	select {
	case <-c.done:
		return false
	default:
		return true
	}
}

// Rejection is the refusal the server sent after the upgrade (401/403/429/500), if any.
func (c *Client) Rejection() *ssi.TradingWSError {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rejection
}

// RateLimit is a copy of the latest quota the server reported ("remaining", "resetMs", ...).
func (c *Client) RateLimit() map[string]interface{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make(map[string]interface{}, len(c.rateLimit))
	for k, v := range c.rateLimit {
		out[k] = v
	}
	return out
}

// Connect opens the socket with the bearer token in the header and waits the handshake grace for a
// post-upgrade refusal.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connectedLocked() {
		c.mu.Unlock()
		return nil
	}
	stale := c.conn
	c.conn = nil
	c.mu.Unlock()
	if stale != nil {
		_ = stale.Close()
	}

	token, err := c.tokenProvider()
	if err != nil {
		return err
	}
	header := http.Header{}
	header.Set(ssi.HeaderAuthorization, ssi.AuthSchemeBearer+token)
	header.Set(ssi.HeaderUserAgent, c.o.userAgent)
	nc, err := c.o.dial(ctx, c.url, header, c.o.timeout)
	if err != nil {
		return ssi.NewWebSocketError("failed to connect to the trading WebSocket: " + err.Error())
	}

	done := make(chan struct{})
	stop := make(chan struct{})
	c.mu.Lock()
	c.conn, c.done, c.stop, c.rejection = nc, done, stop, nil
	c.mu.Unlock()
	go c.readLoop(nc, done)

	var waited time.Duration
	const step = 50 * time.Millisecond
	for waited < c.o.handshakeGrace {
		if rej := c.Rejection(); rej != nil {
			break
		}
		select {
		case <-done:
			waited = c.o.handshakeGrace
			continue
		default:
		}
		if err := c.o.sleep(ctx, step); err != nil {
			c.Close()
			return err
		}
		waited += step
	}
	if rej := c.Rejection(); rej != nil {
		c.Close()
		return rej
	}
	if c.o.pingInterval > 0 {
		go c.pingLoop(stop)
	}
	return nil
}

func (c *Client) pingLoop(stop <-chan struct{}) {
	t := time.NewTicker(c.o.pingInterval)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-t.C:
			if err := c.Ping(); err != nil {
				return
			}
		}
	}
}

// Close closes the socket, stops the reader and wakes every waiter. It is safe to call twice.
func (c *Client) Close() {
	c.mu.Lock()
	nc, done, stop := c.conn, c.done, c.stop
	c.conn, c.stop = nil, nil
	c.mu.Unlock()
	if stop != nil {
		close(stop)
	}
	if nc != nil {
		_ = nc.Close()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
	c.failPending(nil)
}

// failPending wakes every waiter: with err when the server refused the connection, or as closed.
func (c *Client) failPending(err *ssi.TradingWSError) {
	c.mu.Lock()
	waiting := c.pending
	c.pending = map[string]chan item{}
	c.mu.Unlock()
	for _, ch := range waiting {
		it := item{closed: true}
		if err != nil {
			it = item{err: err}
		}
		select {
		case ch <- it:
		default:
		}
	}
}

func (c *Client) readLoop(nc conn, done chan struct{}) {
	defer close(done)
	defer c.failPending(nil)
	for {
		_, raw, err := nc.ReadMessage()
		if err != nil {
			return
		}
		frame := decodeFrame(raw)
		if frame == nil {
			continue
		}
		if IsRejection(frame) {
			rej := RejectionError(frame)
			c.mu.Lock()
			c.rejection = rej
			c.mu.Unlock()
			c.failPending(rej)
			continue
		}
		var ch chan item
		var handler func(map[string]interface{})
		c.mu.Lock()
		if id, ok := frame["id"].(string); ok {
			ch = c.pending[id]
		}
		handler = c.onEvent
		c.mu.Unlock()
		if ch != nil {
			select {
			case ch <- item{frame: frame}:
			default:
				wsLog.Error("Dropped a trading response frame: the per-command buffer is full")
			}
		} else if handler != nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						wsLog.Error("Error in the OnEvent handler: %s", redact.MaskBearer(fmt.Sprint(r)))
					}
				}()
				handler(frame)
			}()
		}
	}
}

func (c *Client) newID() (string, error) {
	for {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return "", ssi.NewWebSocketError("could not generate a command id")
		}
		id := hex.EncodeToString(b)
		c.mu.Lock()
		_, used := c.pending[id]
		c.mu.Unlock()
		if !used {
			return id, nil
		}
	}
}

// Call is one in-flight command: every frame the server sends with its id arrives through Next.
type Call struct {
	c  *Client
	id string
	ch chan item
}

// Close stops routing frames to the call; later frames with that id go to the OnEvent handler.
func (k *Call) Close() {
	k.c.mu.Lock()
	if k.c.pending[k.id] == k.ch {
		delete(k.c.pending, k.id)
	}
	k.c.mu.Unlock()
}

// Next takes the next frame of the call. timeoutMs (milliseconds) <= 0 uses the client's request
// timeout.
//
// Errors: *ssi.TradingWSError for an error frame or a refusal; *ssi.WebSocketError on timeout (the
// command is NOT retried) or a closed connection.
func (k *Call) Next(ctx context.Context, timeoutMs int) (*Response, error) {
	return k.c.next(ctx, k, time.Duration(timeoutMs)*time.Millisecond, k.c.hint())
}

func (c *Client) hint() string {
	if c.o.encoding == EncodingAuto {
		return ""
	}
	return string(c.o.encoding)
}

func (c *Client) next(ctx context.Context, k *Call, timeout time.Duration, hint string) (*Response, error) {
	if timeout <= 0 {
		timeout = c.o.timeout
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case it := <-k.ch:
		if it.err != nil {
			return nil, it.err
		}
		if it.closed {
			return nil, ssi.NewWebSocketError("connection closed before the response arrived")
		}
		frame, err := Unwrap(it.frame, hint)
		if err != nil {
			return nil, err
		}
		resp := ResponseFromFrame(frame)
		c.noteRateLimit(resp.RateLimit)
		return resp, nil
	case <-t.C:
		return nil, ssi.NewWebSocketError("timed out waiting for a response (not retried)")
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) noteRateLimit(rl map[string]interface{}) {
	if len(rl) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rateLimit = rl
	if rem, ok := rl["remaining"].(float64); ok && rem == 0 {
		reset, _ := rl["resetMs"].(float64)
		wait := time.Duration(reset * float64(time.Millisecond))
		if wait > c.o.maxRateLimitWait {
			wait = c.o.maxRateLimitWait
		}
		c.blockedUntil = c.o.now().Add(wait)
	}
}

// RequestStream sends one command and returns its Call; the caller releases it with Call.Close.
// params is serialized once (pass a json.RawMessage for text you already serialized).
func (c *Client) RequestStream(ctx context.Context, method string, params interface{}) (*Call, error) {
	text, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	enc := c.encoding
	c.mu.Unlock()
	return c.send(ctx, method, text, enc)
}

func (c *Client) send(ctx context.Context, method, paramsText, enc string) (*Call, error) {
	c.mu.Lock()
	rej, nc, blocked := c.rejection, c.conn, c.blockedUntil
	c.mu.Unlock()
	if rej != nil {
		return nil, rej
	}
	if nc == nil {
		return nil, ssi.NewWebSocketError("not connected")
	}
	if c.o.respectRateLimit {
		if wait := blocked.Sub(c.o.now()); wait > 0 {
			if err := c.o.sleep(ctx, wait); err != nil {
				return nil, err
			}
		}
	}
	id, err := c.newID()
	if err != nil {
		return nil, err
	}
	raw, err := BuildCommand(method, paramsText, c.privateKey, id, c.o.now().UnixMilli(), enc)
	if err != nil {
		return nil, err
	}
	ch := make(chan item, 16)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	call := &Call{c: c, id: id, ch: ch}
	c.sendMu.Lock()
	err = nc.WriteMessage(websocket.TextMessage, []byte(raw))
	c.sendMu.Unlock()
	if err != nil {
		call.Close()
		return nil, ssi.NewWebSocketError(fmt.Sprintf("failed to send %s: %T", method, err))
	}
	return call, nil
}

func (c *Client) otherEncoding(method string, err *ssi.TradingWSError, used string) string {
	if c.o.encoding != EncodingAuto || !IsMutating(method) || !signatureCodes[err.ErrorCode] {
		return ""
	}
	if used == "base64" {
		return "hex"
	}
	return "base64"
}

func (c *Client) once(ctx context.Context, method, text, enc string) (*Response, error) {
	call, err := c.send(ctx, method, text, enc)
	if err != nil {
		return nil, err
	}
	defer call.Close()
	hint := ""
	if c.o.encoding != EncodingAuto {
		hint = enc
	}
	return c.next(ctx, call, 0, hint)
}

// Request sends one command and returns its first response frame (the ack). It is never
// retried, with one exception: in "auto" signature mode a command whose signature the server
// refused (it was NOT executed) is sent once more with the other encoding.
//
// For order.* the ack carries orderStatus "PD"; the final result arrives later on the
// order.<account> events. Use RequestStream to also see the second frame with the same id.
func (c *Client) Request(ctx context.Context, method string, params interface{}) (*Response, error) {
	text, err := marshalParams(params)
	if err != nil {
		return nil, err
	}
	return c.request(ctx, method, text)
}

func (c *Client) request(ctx context.Context, method, text string) (*Response, error) {
	c.mu.Lock()
	enc := c.encoding
	c.mu.Unlock()
	resp, err := c.once(ctx, method, text, enc)
	if err == nil {
		return resp, nil
	}
	var first *ssi.TradingWSError
	if !errors.As(err, &first) {
		return nil, err
	}
	other := c.otherEncoding(method, first, enc)
	if other == "" {
		return nil, err
	}
	resp, err = c.once(ctx, method, text, other)
	if err == nil {
		c.mu.Lock()
		c.encoding = other
		c.mu.Unlock()
		return resp, nil
	}
	var second *ssi.TradingWSError
	if errors.As(err, &second) {
		return nil, bothRefused(second)
	}
	return nil, err
}

type subscriptionFrame struct {
	Method  string   `json:"method"`
	Channel string   `json:"channel"`
	Topics  []string `json:"topics"`
}

// Subscribe sends a subscription frame (no id, so the server treats it as a subscription). The ack
// and the later events carry no command id, so they arrive on the OnEvent handler.
func (c *Client) Subscribe(channel string, topics []string) error {
	return c.sendSubscription("SUBSCRIBE", channel, topics)
}

// Unsubscribe removes topics subscribed with Subscribe; no topics drops every one on the channel.
func (c *Client) Unsubscribe(channel string, topics []string) error {
	return c.sendSubscription("UNSUBSCRIBE", channel, topics)
}

// Ping sends PING_PONG; the ack arrives on the OnEvent handler.
func (c *Client) Ping() error {
	return c.sendFrame([]byte(`{"method":"PING_PONG"}`))
}

func (c *Client) sendSubscription(method, channel string, topics []string) error {
	if topics == nil {
		topics = []string{}
	}
	b, err := json.Marshal(subscriptionFrame{Method: method, Channel: channel, Topics: topics})
	if err != nil {
		return ssi.NewValidationError("topics cannot be encoded as JSON")
	}
	return c.sendFrame(b)
}

func (c *Client) sendFrame(b []byte) error {
	c.mu.Lock()
	nc := c.conn
	c.mu.Unlock()
	if nc == nil {
		return ssi.NewWebSocketError("not connected")
	}
	c.sendMu.Lock()
	err := nc.WriteMessage(websocket.TextMessage, b)
	c.sendMu.Unlock()
	if err != nil {
		return ssi.NewWebSocketError(fmt.Sprintf("failed to send: %T", err))
	}
	return nil
}
