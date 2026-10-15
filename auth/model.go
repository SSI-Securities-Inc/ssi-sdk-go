package auth

import (
	"fmt"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

// Epoch values at or above this are milliseconds (10^10 s is the year 2286).
const msThreshold = int64(10_000_000_000)

// toSeconds normalizes an epoch to seconds. The server sends seconds; a millisecond value
// (>= 1e10) is divided by 1000 defensively. nil, garbage and negatives become 0
// (meaning "unknown/expired").
func toSeconds(v interface{}) int64 {
	s := util.ToInt64(v)
	if s >= msThreshold {
		s /= 1000
	}
	if s < 0 {
		return 0
	}
	return s
}

// Token represents an OAuth2-style access token. Its String/GoString never print the tokens.
type Token struct {
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	// ExpiresAt is the access token expiry, epoch seconds (0 = unknown, treated as expired).
	ExpiresAt    int64  `json:"expiresAt"`
	RefreshToken string `json:"refreshToken"`
	// RefreshTokenExpiresAt is the refresh token expiry, epoch seconds.
	RefreshTokenExpiresAt int64 `json:"refreshExpiresAt"`
}

func (t Token) String() string {
	return fmt.Sprintf("Token{TokenType:%q ExpiresAt:%d HasRefreshToken:%v RefreshTokenExpiresAt:%d}",
		t.TokenType, t.ExpiresAt, t.RefreshToken != "", t.RefreshTokenExpiresAt)
}

func (t Token) GoString() string { return t.String() }

// EffectiveExpiresAt is the epoch second after which this token set can no longer be used
// without a new OTP: the earlier of the access-token and refresh-token expiries, ignoring a
// value that is unknown (<= 0). 0 when both are unknown.
func (t *Token) EffectiveExpiresAt() int64 {
	var known int64
	for _, v := range []int64{t.ExpiresAt, t.RefreshTokenExpiresAt} {
		if v > 0 && (known == 0 || v < known) {
			known = v
		}
	}
	return known
}

// CanRefresh reports whether the refresh token is present and not (about to be) expired. An
// unknown refresh expiry (<= 0) gets the benefit of the doubt: the server rejects it if dead.
func (t *Token) CanRefresh() bool {
	if t.RefreshToken == "" {
		return false
	}
	if t.RefreshTokenExpiresAt <= 0 {
		return true
	}
	return time.Now().Unix() < t.RefreshTokenExpiresAt-ssi.TokenExpirySkewMs/1000
}

func TokenFromMap(data map[string]interface{}) *Token {
	t := &Token{TokenType: "Bearer"}
	if v, ok := data["accessToken"].(string); ok {
		t.AccessToken = v
	}
	if v, ok := data["tokenType"].(string); ok && v != "" {
		t.TokenType = v
	}
	t.ExpiresAt = toSeconds(data["expiresAt"])
	if v, ok := data["refreshToken"].(string); ok {
		t.RefreshToken = v
	}
	t.RefreshTokenExpiresAt = toSeconds(data["refreshExpiresAt"])
	return t
}

// TokenRequest is the request to obtain an access token.
type TokenRequest struct {
	APIKey        string `json:"apiKey"`
	APISecret     string `json:"apiSecret"`
	OTP           string `json:"otp,omitempty"`
	TransactionID string `json:"transactionId,omitempty"`
}

func (r TokenRequest) String() string {
	return "TokenRequest{APIKey:*** APISecret:*** OTP:*** TransactionID:***}"
}
func (r TokenRequest) GoString() string { return r.String() }

// OTPRequest is the request to get OTP for trading operations.
type OTPRequest struct {
	APIKey    string `json:"apiKey"`
	APISecret string `json:"apiSecret"`
}

func (r OTPRequest) String() string   { return "OTPRequest{APIKey:*** APISecret:***}" }
func (r OTPRequest) GoString() string { return r.String() }

// RefreshTokenRequest is the request to refresh an access token.
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (r RefreshTokenRequest) String() string   { return "RefreshTokenRequest{RefreshToken:***}" }
func (r RefreshTokenRequest) GoString() string { return r.String() }

// OTPResponse is the response to an OTP request. TransactionID is only present for Smart OTP
// (push-approval); SMS/email OTP responses carry just a message.
type OTPResponse struct {
	Message       string
	TransactionID string
}

func (r OTPResponse) String() string {
	return fmt.Sprintf("OTPResponse{Message:%q HasTransactionID:%v}", r.Message, r.TransactionID != "")
}

func (r OTPResponse) GoString() string { return r.String() }

// OTPResponseFromMap builds an OTPResponse from the requestOtp response body.
func OTPResponseFromMap(data map[string]interface{}) OTPResponse {
	payload := data
	if inner, ok := data["data"].(map[string]interface{}); ok {
		payload = inner
	}
	msg := util.ToStr(util.Pick(payload, "message", "msg"))
	return OTPResponse{Message: msg, TransactionID: util.ToStr(payload["transactionId"])}
}
