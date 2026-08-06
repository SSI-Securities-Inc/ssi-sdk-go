package auth

import (
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

// Token represents an OAuth2-style access token.
type Token struct {
	AccessToken           string `json:"accessToken"`
	TokenType             string `json:"tokenType"`
	ExpiresAt             int64  `json:"expiresAt"`
	RefreshToken          string `json:"refreshToken"`
	RefreshTokenExpiresAt int64  `json:"refreshExpiresAt"`
}

func TokenFromMap(data map[string]interface{}) *Token {
	t := &Token{
		TokenType: "Bearer",
	}
	if v, ok := data["accessToken"].(string); ok {
		t.AccessToken = v
	}
	if v, ok := data["tokenType"].(string); ok {
		t.TokenType = v
	}
	if v, ok := data["expiresAt"]; ok {
		t.ExpiresAt = util.ToInt64(v)
	}
	if v, ok := data["refreshToken"].(string); ok {
		t.RefreshToken = v
	}
	if v, ok := data["refreshExpiresAt"]; ok {
		t.RefreshTokenExpiresAt = util.ToInt64(v)
	}
	return t
}

// TokenRequest is the request to obtain an access token.
type TokenRequest struct {
	APIKey        string `json:"apiKey"`
	APISecret     string `json:"apiSecret"`
	OTP           string `json:"otp,omitempty"`
	TransactionID string `json:"transactionId,omitempty"`
}

// OTPRequest is the request to get OTP for trading operations.
type OTPRequest struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
}

// RefreshTokenRequest is the request to refresh an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}
