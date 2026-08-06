// Package ssi provides specialized entry points for the SSI FastConnect SDK.
//
// Clients overview:
//
//	Auth     — Credentials + token management. Entry point for all other clients.
//	Data     — Market data. Pass an authenticated Auth (no OTP needed).
//	Trading  — Trading + portfolio + account. Pass an Auth after calling Authenticate(otp).
//	Stream   — Real-time WebSocket. Pass an Auth after calling Authenticate(otp).
//
// Quick start:
//
//	// Market data (no OTP)
//	auth := ssi.NewAuth(config)
//	defer auth.Close()
//	auth.Authenticate("")
//	data := ssi.NewData(auth)
//	ohlc, _ := data.MarketData.GetOHLC1Minute("VNM")
//
//	// Trading + streaming (OTP required)
//	auth := ssi.NewAuth(config)
//	defer auth.Close()
//	auth.Authenticate("123456")
//	t := ssi.NewTrading(auth)
//	order, _ := t.Trading.PlaceLimitOrder(...)
//
//	s := ssi.NewStream(auth)
//	defer s.Disconnect()
//	s.Streaming.SetOnData(func(msg interface{}) { fmt.Println(msg) })
//	s.Connect()
//	s.Streaming.SubscribeSymbolTrade([]string{"VNM"}, nil)
//	s.Wait(nil)
package ssi

import (
	"time"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/account"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/auth"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/portfolio"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/stream"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var clientLog = logger.New("ssi_sdk.client")

// Re-export Config and NewConfig so users only need to import ssi/.
type Config = ssisdk.Config

// NewConfig creates a new Config with default values.
func NewConfig(clientID string) *Config {
	return ssisdk.NewConfig(clientID)
}

// ── Auth ──────────────────────────────────────────────────────────────────────

// Auth manages credentials and access tokens.
// It is the required entry point for all other specialized clients.
//
//	auth := ssi.NewAuth(config)
//	defer auth.Close()
//	token, err := auth.Authenticate("123456") // pass "" if no OTP needed
type Auth struct {
	config       *ssisdk.Config
	restClient   *transport.RestClient
	TokenManager *auth.TokenManager
}

// NewAuth creates an Auth client from the given Config.
func NewAuth(config *Config) *Auth {
	rest := transport.NewRestClient(config)
	return &Auth{
		config:       config,
		restClient:   rest,
		TokenManager: auth.NewTokenManager(rest, config.APIKey, config.APISecret),
	}
}

// Authenticate obtains an access token using OTP code or Smart OTP transactionId.
func (a *Auth) Authenticate(otp string, transactionID ...string) (*auth.Token, error) {
	return a.TokenManager.Authenticate(otp, transactionID...)
}

// Refresh exchanges the current refresh token for a new access token.
func (a *Auth) Refresh() (*auth.Token, error) {
	return a.TokenManager.Refresh()
}

// RequestOTP triggers an OTP SMS/email to the registered number.
func (a *Auth) RequestOTP() (map[string]interface{}, error) {
	return a.TokenManager.RequestOTP()
}

// EnsureAuthenticated ensures a valid access token is available, renewing, authenticating with OTP, or polling Smart OTP as needed.
func (a *Auth) EnsureAuthenticated(otp string, transactionID ...string) (string, error) {
	return a.TokenManager.EnsureAuthenticated(otp, transactionID...)
}

// AccessToken returns the current bearer token string, or "" if not authenticated.
func (a *Auth) AccessToken() string {
	return a.TokenManager.AccessToken()
}

// Close releases the underlying HTTP connection pool.
func (a *Auth) Close() {
	a.restClient.Close()
}

// ── Data ──────────────────────────────────────────────────────────────────────

// Data is the market data client.
// Requires auth.Authenticate("") (no OTP) before use.
//
//	auth := ssi.NewAuth(config)
//	defer auth.Close()
//	auth.Authenticate("")
//	data := ssi.NewData(auth)
//	ohlc, err := data.MarketData.GetOHLC1Minute("VNM")
type Data struct {
	auth       *Auth
	MarketData *market.Service
}

// NewData creates a market data client backed by the given Auth.
func NewData(auth *Auth) *Data {
	return &Data{
		auth:       auth,
		MarketData: market.NewService(auth.restClient),
	}
}

// ── Trading ───────────────────────────────────────────────────────────────────

// Trading is the client for orders, portfolio, and account management.
// Requires auth.Authenticate(otp) with a valid OTP before use.
//
//	auth := ssi.NewAuth(config)
//	defer auth.Close()
//	auth.Authenticate("123456")
//	t := ssi.NewTrading(auth)
//	order, err := t.Trading.PlaceLimitOrder(accountNo, "VNM", trading.SideBuy, 100, 50000)
type Trading struct {
	auth      *Auth
	Trading   *trading.Service
	Account   *account.Service
	Portfolio *portfolio.Service
}

// NewTrading creates a trading/portfolio/account client backed by the given Auth.
func NewTrading(auth *Auth) *Trading {
	return &Trading{
		auth:      auth,
		Trading:   trading.NewService(auth.restClient, auth.config.PrivateKey),
		Account:   account.NewService(auth.restClient),
		Portfolio: portfolio.NewService(auth.restClient, auth.config.ClientID),
	}
}

// ── Stream ────────────────────────────────────────────────────────────────────

// Stream is the real-time WebSocket client.
// Requires auth.Authenticate(otp) with a valid OTP before use.
//
//	auth := ssi.NewAuth(config)
//	defer auth.Close()
//	auth.Authenticate("123456")
//	s := ssi.NewStream(auth)
//	defer s.Disconnect()
//	s.Streaming.SetOnData(func(msg interface{}) { fmt.Println(msg) })
//	s.Connect()
//	s.Streaming.SubscribeSymbolTrade([]string{"VNM"}, nil)
//	s.Wait(nil) // block until disconnected
type Stream struct {
	auth      *Auth
	wsClient  *transport.WebSocketClient
	Streaming *stream.Service
}

// NewStream creates a streaming client backed by the given Auth.
// The current access token is applied to the WebSocket immediately.
func NewStream(auth *Auth) *Stream {
	ws := transport.NewWebSocketClient(auth.config)
	if token := auth.AccessToken(); token != "" {
		ws.SetToken(token)
	}
	return &Stream{
		auth:      auth,
		wsClient:  ws,
		Streaming: stream.NewService(ws),
	}
}

// Connect establishes the WebSocket connection. Call after NewStream.
func (s *Stream) Connect() error {
	return s.wsClient.Connect()
}

// Disconnect closes the WebSocket connection and releases resources.
func (s *Stream) Disconnect() {
	s.Streaming.StopPingLoop()
	s.wsClient.Disconnect()
}

// IsConnected reports whether the WebSocket is currently connected.
func (s *Stream) IsConnected() bool {
	return s.wsClient.IsConnected()
}

// Wait blocks until the WebSocket is disconnected or timeout elapses.
// Pass nil to block indefinitely.
func (s *Stream) Wait(timeout *time.Duration) {
	s.wsClient.Wait(timeout)
}

// ── Client (deprecated unified client) ───────────────────────────────────────

// Client is the legacy unified client that bundles all services in one struct.
//
// Deprecated: Use the specialized Auth / Data / Trading / Stream clients instead.
// This type is retained for backward compatibility only and may be removed in a future version.
type Client struct {
	config       *ssisdk.Config
	restClient   *transport.RestClient
	wsClient     *transport.WebSocketClient
	TokenManager *auth.TokenManager
	Account      *account.Service
	Portfolio    *portfolio.Service
	MarketData   *market.Service
	Trading      *trading.Service
	Streaming    *stream.Service
}

// NewClient creates the legacy unified client.
//
// Deprecated: Use NewAuth + NewData / NewTrading / NewStream instead.
func NewClient(config *ssisdk.Config) *Client {
	restClient := transport.NewRestClient(config)
	wsClient := transport.NewWebSocketClient(config)
	return &Client{
		config:       config,
		restClient:   restClient,
		wsClient:     wsClient,
		TokenManager: auth.NewTokenManager(restClient, config.APIKey, config.APISecret),
		Account:      account.NewService(restClient),
		Portfolio:    portfolio.NewService(restClient, config.ClientID),
		MarketData:   market.NewService(restClient),
		Trading:      trading.NewService(restClient, config.PrivateKey),
		Streaming:    stream.NewService(wsClient),
	}
}

// Authenticate obtains an access token and sets it on the WebSocket client.
//
// Deprecated: Use Auth.Authenticate instead.
func (c *Client) Authenticate(otp string) error {
	token, err := c.TokenManager.Authenticate(otp)
	if err != nil {
		return err
	}
	c.wsClient.SetToken(token.AccessToken)
	clientLog.Info("Client authenticated")
	return nil
}

// Connect establishes the WebSocket connection.
//
// Deprecated: Use Stream.Connect instead.
func (c *Client) Connect() error {
	if err := c.wsClient.Connect(); err != nil {
		return err
	}
	clientLog.Info("Client WebSocket connected")
	return nil
}

// Disconnect closes all connections and releases resources.
//
// Deprecated: Use Auth.Close + Stream.Disconnect instead.
func (c *Client) Disconnect() {
	c.wsClient.Disconnect()
	c.restClient.Close()
	clientLog.Info("Client disconnected")
}
