package transport

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
)

var wsLog = logger.New("ssi_sdk.transport.websocket")

const wsThreadJoinTimeout = 5

// MessageHandler is a callback for WebSocket messages.
type MessageHandler func(msg map[string]interface{})

// WebSocketClient is the synchronous WebSocket client for SSI real-time streaming.
type WebSocketClient struct {
	config    *ssi.Config
	conn      *websocket.Conn
	handlers  map[string][]MessageHandler
	token     string
	running   bool
	mu        sync.RWMutex
	handlerMu sync.RWMutex
	done      chan struct{}
}

func NewWebSocketClient(config *ssi.Config) *WebSocketClient {
	return &WebSocketClient{
		config:   config,
		handlers: make(map[string][]MessageHandler),
		done:     make(chan struct{}),
	}
}

func (c *WebSocketClient) SetToken(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.token = token
}

func (c *WebSocketClient) Connect() error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	wsURL := c.config.StreamingURL
	header := http.Header{}
	header.Set(HeaderContentType, ContentTypeJSON)
	header.Set(HeaderAccept, ContentTypeJSON)

	c.mu.RLock()
	if c.token != "" {
		header.Set(HeaderAuthorization, AuthSchemeBearer+c.token)
	}
	c.mu.RUnlock()

	var dialer *websocket.Dialer
	if c.config.Proxy != "" {
		proxyURL, err := url.Parse(c.config.Proxy)
		if err != nil {
			return fmt.Errorf("invalid proxy URL: %w", err)
		}
		dialer = &websocket.Dialer{
			Proxy:            http.ProxyURL(proxyURL),
			HandshakeTimeout: time.Duration(c.config.Timeout) * time.Second,
		}
	} else {
		dialer = &websocket.Dialer{
			HandshakeTimeout: time.Duration(c.config.Timeout) * time.Second,
		}
	}

	maxRetries := c.config.MaxRetries
	delay := c.config.RetryDelay
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		conn, _, err := dialer.Dial(wsURL, header)
		if err == nil {
			c.mu.Lock()
			c.conn = conn
			c.running = true
			c.done = make(chan struct{})
			c.mu.Unlock()

			go c.listen()
			wsLog.Info("WebSocket connected to %s", wsURL)
			return nil
		}
		lastErr = err
		if attempt < maxRetries {
			wait := delay * math.Pow(2, float64(attempt))
			wsLog.Error("WebSocket connect attempt %d/%d failed: %v. Retrying in %.1fs",
				attempt+1, maxRetries+1, err, wait)
			time.Sleep(time.Duration(wait * float64(time.Second)))
		}
	}

	return ssi.NewWebSocketError(
		fmt.Sprintf("Failed to connect to %s after %d attempts: %v", wsURL, maxRetries+1, lastErr),
	)
}

func (c *WebSocketClient) Disconnect() {
	c.mu.Lock()
	c.running = false
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()

	if conn != nil {
		conn.Close()
	}

	select {
	case <-c.done:
	case <-time.After(time.Duration(wsThreadJoinTimeout) * time.Second):
	}
	wsLog.Info("WebSocket disconnected")
}

func (c *WebSocketClient) On(channel string, handler MessageHandler) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	if handler == nil {
		delete(c.handlers, channel)
		return
	}
	c.handlers[channel] = []MessageHandler{handler}
}

func (c *WebSocketClient) Off(channel string, _ MessageHandler) {
	c.handlerMu.Lock()
	defer c.handlerMu.Unlock()
	// Only one handler is registered per channel, so any Off clears it.
	delete(c.handlers, channel)
}

func (c *WebSocketClient) Send(data map[string]interface{}) error {
	c.mu.RLock()
	conn := c.conn
	running := c.running
	c.mu.RUnlock()

	if conn == nil || !running {
		return ssi.NewWebSocketError("Not connected. Call Connect() first.")
	}

	b, err := json.Marshal(data)
	if err != nil {
		return ssi.NewWebSocketError(fmt.Sprintf("Failed to marshal message: %v", err))
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn == nil {
		return ssi.NewWebSocketError("Connection closed while sending message.")
	}
	if err := c.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return ssi.NewWebSocketError(fmt.Sprintf("Failed to send message: %v", err))
	}
	return nil
}

func (c *WebSocketClient) listen() {
	defer func() {
		close(c.done)
		c.mu.Lock()
		c.running = false
		c.conn = nil
		c.mu.Unlock()
	}()

	for {
		c.mu.RLock()
		conn := c.conn
		running := c.running
		c.mu.RUnlock()

		if !running || conn == nil {
			return
		}

		_, rawMessage, err := conn.ReadMessage()
		if err != nil {
			c.mu.RLock()
			stillRunning := c.running
			c.mu.RUnlock()
			if !stillRunning {
				return
			}
			wsLog.Error("WebSocket read error: %v", err)
			return
		}

		var message map[string]interface{}
		if err := json.Unmarshal(rawMessage, &message); err != nil {
			wsLog.Debug("Received non-JSON message: %s", string(rawMessage))
			continue
		}

		channel, _ := message["channel"].(string)
		c.handlerMu.RLock()
		handlers := c.handlers[channel]
		c.handlerMu.RUnlock()

		for _, handler := range handlers {
			func() {
				defer func() {
					if r := recover(); r != nil {
						wsLog.Error("Error in handler for channel '%s': %v", channel, r)
					}
				}()
				handler(message)
			}()
		}
	}
}

func (c *WebSocketClient) Wait(timeout *time.Duration) {
	if timeout == nil {
		<-c.done
		return
	}
	select {
	case <-c.done:
	case <-time.After(*timeout):
	}
}

func (c *WebSocketClient) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.running && c.conn != nil
}
