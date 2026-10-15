package experimental

import (
	"context"
	"encoding/json"
	"errors"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"
)

// jsonNumber marks already-formatted decimal text to be emitted as a JSON number (not a string).
type jsonNumber string

func (n jsonNumber) MarshalJSON() ([]byte, error) {
	b := []byte(n)
	if !json.Valid(b) {
		return nil, errors.New("not a JSON number")
	}
	return b, nil
}

// TradingWS opens the trading WebSocket for an authenticated Auth and returns the service. The
// socket renews nothing by itself: if the connection drops, call Client().Connect again (the
// token provider asks the TokenManager for a valid token every time).
//
// Defaults come from the Config (request timeout = Config.TimeoutMs or 10000, User-Agent =
// Config.UserAgent); opts override them. The caller must Close the returned service.
//
// EXPERIMENTAL: see the package documentation.
func TradingWS(ctx context.Context, auth *ssi.Auth, opts ...Option) (*Service, error) {
	cfg := auth.Config()
	timeoutMs := cfg.TimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 10000
	}
	all := append([]Option{WithRequestTimeoutMs(timeoutMs), WithUserAgent(cfg.UserAgent)}, opts...)
	client, err := NewClient(cfg.TradingWSURL(), func() (string, error) {
		return auth.EnsureAuthenticated("")
	}, cfg.PrivateKey, all...)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		return nil, err
	}
	return NewService(client, cfg), nil
}
