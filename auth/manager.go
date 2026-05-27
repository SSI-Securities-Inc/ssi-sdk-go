package auth

import (
	"fmt"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var authLog = logger.New("ssi_sdk.auth")

const (
	epAccessToken  = "/api/v3/auth/token"
	epRefreshToken = "/api/v3/auth/refresh"
	epRequestOTP   = "/api/v3/auth/requestOtp"
)

// TokenManager handles authentication tokens and OTP verification.
type TokenManager struct {
	rest      *transport.RestClient
	apiKey    string
	apiSecret string
	token     *Token
}

func NewTokenManager(rest *transport.RestClient, apiKey, apiSecret string) *TokenManager {
	return &TokenManager{
		rest:      rest,
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}
}

func (tm *TokenManager) Token() *Token {
	return tm.token
}

func (tm *TokenManager) AccessToken() string {
	if tm.token == nil {
		return ""
	}
	return tm.token.AccessToken
}

func (tm *TokenManager) IsTokenExpired() bool {
	if tm.token == nil {
		return true
	}
	if tm.token.ExpiresAt <= 0 {
		return false
	}
	return time.Now().Unix() >= tm.token.ExpiresAt
}

func (tm *TokenManager) HasRefreshToken() bool {
	return tm.token != nil && tm.token.RefreshToken != ""
}

func (tm *TokenManager) IsRefreshTokenExpired() bool {
	if tm.token == nil || tm.token.RefreshToken == "" {
		return true
	}
	if tm.token.RefreshTokenExpiresAt <= 0 {
		return false
	}
	return time.Now().Unix() >= tm.token.RefreshTokenExpiresAt
}

func (tm *TokenManager) Refresh() (*Token, error) {
	if tm.apiKey == "" || tm.apiSecret == "" {
		return nil, ssi.NewAuthenticationError("api_key and api_secret are required for token refresh", "", 0, nil)
	}
	if !tm.HasRefreshToken() {
		return nil, ssi.NewAuthenticationError("No refresh token available — authenticate first", "", 0, nil)
	}

	body := map[string]interface{}{
		"refreshToken": tm.token.RefreshToken,
	}

	data, err := tm.rest.Post(epRefreshToken, body, nil, nil)
	if err != nil {
		authLog.Error("Failed to refresh token for endpoint %s: %v", epRefreshToken, err)
		return nil, err
	}

	payload := extractPayload(data)
	if payload == nil {
		return nil, ssi.NewAPIError("Unexpected response format while refreshing token", "", 0, nil)
	}

	tm.token = TokenFromMap(payload)
	if tm.token.AccessToken == "" {
		return nil, ssi.NewAPIError("Refreshed token payload is missing access token", "", 0, nil)
	}
	tm.rest.SetAuthHeader(tm.token.AccessToken)

	authLog.Info("Token refreshed successfully")
	return tm.token, nil
}

func (tm *TokenManager) Authenticate(otp string) (*Token, error) {
	if tm.apiKey == "" || tm.apiSecret == "" {
		return nil, ssi.NewAuthenticationError("api_key and api_secret are required for authentication", "", 0, nil)
	}

	body := map[string]interface{}{
		"apiKey":    tm.apiKey,
		"apiSecret": tm.apiSecret,
	}
	if otp != "" {
		body["otp"] = otp
	}

	data, err := tm.rest.Post(epAccessToken, body, nil, nil)
	if err != nil {
		authLog.Error("Failed to authenticate for endpoint %s: %v", epAccessToken, err)
		return nil, err
	}

	payload := extractPayload(data)
	if payload == nil {
		return nil, ssi.NewAPIError("Unexpected response format while authenticating", "", 0, nil)
	}

	tm.token = TokenFromMap(payload)
	if tm.token.AccessToken == "" {
		return nil, ssi.NewAPIError("Authentication payload is missing access token", "", 0, nil)
	}
	tm.rest.SetAuthHeader(tm.token.AccessToken)

	authLog.Info("Authentication successful")
	return tm.token, nil
}

func (tm *TokenManager) SetToken(token *Token) {
	tm.token = token
	tm.rest.SetAuthHeader(token.AccessToken)
	authLog.Info("Access token set manually")
}

func (tm *TokenManager) RequestOTP() (map[string]interface{}, error) {
	if tm.apiKey == "" || tm.apiSecret == "" {
		return nil, ssi.NewAuthenticationError("api_key and api_secret are required for OTP request", "", 0, nil)
	}

	body := map[string]interface{}{
		"apiKey":    tm.apiKey,
		"apiSecret": tm.apiSecret,
	}

	data, err := tm.rest.Post(epRequestOTP, body, nil, nil)
	if err != nil {
		authLog.Error("Failed to request OTP for endpoint %s: %v", epRequestOTP, err)
		return nil, err
	}

	return data, nil
}

func extractPayload(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	if payload, ok := data["data"].(map[string]interface{}); ok {
		return payload
	}
	if _, ok := data["accessToken"]; ok {
		return data
	}
	return nil
}

// Deprecated: ConnectRestOnly is deprecated, use Authenticate instead.
func (tm *TokenManager) ConnectRestOnly() {
	fmt.Println("WARNING: connect_rest_only is deprecated, use Authenticate(otp) instead")
}
