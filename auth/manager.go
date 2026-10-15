package auth

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/redact"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var authLog = logger.New("ssi_sdk.auth")

const (
	epAccessToken  = "/api/v3/auth/token"
	epRefreshToken = "/api/v3/auth/refresh"
	epRequestOTP   = "/api/v3/auth/requestOtp"

	SmartOTPPendingStatus  = ssi.SmartOTPPendingStatus
	SmartOTPPendingCode    = ssi.SmartOTPPendingCode
	SmartOTPPollMaxRetries = 5
	SmartOTPPollIntervalMs = 5000 // milliseconds
)

// TokenManager handles authentication tokens and OTP verification. It is safe for concurrent
// use; concurrent refreshes make a single server call.
type TokenManager struct {
	rest      *transport.RestClient
	apiKey    string
	apiSecret string

	mu        sync.RWMutex // guards token
	token     *Token
	refreshMu sync.Mutex // single-flight refresh
}

func NewTokenManager(rest *transport.RestClient, apiKey, apiSecret string) *TokenManager {
	return &TokenManager{rest: rest, apiKey: apiKey, apiSecret: apiSecret}
}

func (tm *TokenManager) Token() *Token {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.token
}

func (tm *TokenManager) setToken(t *Token) {
	tm.mu.Lock()
	tm.token = t
	tm.mu.Unlock()
}

func (tm *TokenManager) AccessToken() string {
	if t := tm.Token(); t != nil {
		return t.AccessToken
	}
	return ""
}

// IsTokenExpired reports whether the access token is missing or (nearly) expired. A 30-second
// skew is applied so a token is never sent seconds before it dies; a token with an unknown
// expiry (<= 0) is treated as expired.
func (tm *TokenManager) IsTokenExpired() bool {
	return tokenExpired(tm.Token(), time.Now())
}

func tokenExpired(t *Token, now time.Time) bool {
	if t == nil {
		return true
	}
	if t.ExpiresAt <= 0 {
		return true
	}
	return now.Unix() >= t.ExpiresAt-ssi.TokenExpirySkewMs/1000
}

func (tm *TokenManager) HasRefreshToken() bool {
	t := tm.Token()
	return t != nil && t.RefreshToken != ""
}

// CanRefresh reports whether the current token can still be refreshed.
func (tm *TokenManager) CanRefresh() bool {
	t := tm.Token()
	return t != nil && t.CanRefresh()
}

// IsRefreshTokenExpired is true when there is no usable refresh token.
func (tm *TokenManager) IsRefreshTokenExpired() bool {
	return !tm.CanRefresh()
}

func (tm *TokenManager) requireCredentials(purpose string) error {
	if tm.apiKey == "" || tm.apiSecret == "" {
		return ssi.NewAuthenticationError("api_key and api_secret are required for "+purpose, "", 0, nil)
	}
	return nil
}

func (tm *TokenManager) checkRefreshable() error {
	t := tm.Token()
	if t == nil || t.RefreshToken == "" {
		return ssi.NewAuthenticationError("No refresh token available — authenticate first", "", 0, nil)
	}
	if !t.CanRefresh() {
		return ssi.NewReauthenticationRequiredError("Refresh token expired — authenticate again with a new OTP", "", 0, nil)
	}
	return nil
}

// clearToken forgets the token and stops sending it; the session is unusable.
func (tm *TokenManager) clearToken() {
	tm.setToken(nil)
	tm.rest.ClearAuthHeader()
}

// coalesced reports whether another caller already refreshed while this one waited for the lock.
func (tm *TokenManager) coalesced(observed string) bool {
	t := tm.Token()
	return t != nil && t.AccessToken != observed && !tm.IsTokenExpired()
}

func errCode(err error) (string, int, map[string]interface{}) {
	var ae *ssi.AuthenticationError
	if errors.As(err, &ae) {
		return ae.Code, ae.StatusCode, ae.ResponseBody
	}
	var pe *ssi.APIError
	if errors.As(err, &pe) {
		return pe.Code, pe.StatusCode, pe.ResponseBody
	}
	return "", 0, nil
}

func isAuthOrAPIError(err error) bool {
	var ae *ssi.AuthenticationError
	var pe *ssi.APIError
	return errors.As(err, &ae) || errors.As(err, &pe)
}

// isReauthError reports whether a refresh failure means the refresh token is dead (401101-
// 401104, 401106-401108). The server treats a reused refresh token as a leak and revokes the
// apiKey's whole session, so the only way forward is a new OTP login: never retry.
func isReauthError(err error) bool {
	code, _, _ := errCode(err)
	n, convErr := strconv.Atoi(code)
	return convErr == nil && ssi.RefreshReauthCodes[n]
}

// isSmartOTPPending reports whether an auth error means the Smart OTP approval is still
// pending (HTTP 202 with body {"code": 401114}).
func isSmartOTPPending(err error) bool {
	var pending *ssi.SmartOTPPendingError
	if errors.As(err, &pending) {
		return true
	}
	_, _, body := errCode(err)
	if body == nil || body["code"] == nil {
		return false
	}
	return fmt.Sprint(body["code"]) == strconv.Itoa(SmartOTPPendingCode) ||
		fmt.Sprintf("%.0f", toFloat(body["code"])) == strconv.Itoa(SmartOTPPendingCode)
}

func toFloat(v interface{}) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return math.NaN()
}

// parseToken parses and validates a token from a raw response payload.
func parseToken(data map[string]interface{}, what string) (*Token, error) {
	if data == nil {
		return nil, ssi.NewAPIError("Unexpected response format while "+what, "", 0, nil)
	}
	payload := data
	if inner, present := data["data"]; present {
		m, ok := inner.(map[string]interface{})
		if !ok {
			return nil, ssi.NewAPIError("Unexpected token payload format while "+what, "", 0, redact.RedactMap(data))
		}
		payload = m
	}
	t := TokenFromMap(payload)
	if t.AccessToken == "" {
		return nil, ssi.NewAPIError("Token payload is missing access token while "+what, "", 0, redact.RedactMap(data))
	}
	return t, nil
}

// Refresh obtains a new access token using the current refresh token. Concurrent callers are
// coalesced: only one request reaches the server, the others reuse its result. The lock is per
// process: the server keeps one active session per apiKey, so a login/refresh in another
// process invalidates this token.
//
// Returns a ReauthenticationRequiredError (and clears the stored token) when the refresh token
// expired or the server rejected it (401101-401104, 401106-401108).
func (tm *TokenManager) Refresh() (*Token, error) {
	return tm.refresh(true)
}

// refresh runs under the single-flight lock. With force=false (the EnsureAuthenticated path)
// the refresh is skipped if another caller already renewed the token while this one waited.
func (tm *TokenManager) refresh(force bool) (*Token, error) {
	if err := tm.checkRefreshable(); err != nil {
		return nil, err
	}
	observed := tm.AccessToken()

	tm.refreshMu.Lock()
	defer tm.refreshMu.Unlock()

	if tm.coalesced(observed) || (!force && !tm.IsTokenExpired()) {
		return tm.Token(), nil
	}
	if err := tm.checkRefreshable(); err != nil {
		return nil, err
	}
	if err := tm.requireCredentials("token refresh"); err != nil {
		return nil, err
	}
	body := map[string]interface{}{"refreshToken": tm.Token().RefreshToken}
	data, err := tm.rest.Post(epRefreshToken, body, nil, nil)
	if err != nil {
		if isAuthOrAPIError(err) && isReauthError(err) {
			code, status, respBody := errCode(err)
			tm.clearToken()
			return nil, ssi.NewReauthenticationRequiredError(
				"Refresh token rejected by the server — authenticate again with a new OTP", code, status, respBody)
		}
		authLog.Error("Failed to refresh token: %v", err)
		return nil, err
	}
	tok, err := parseToken(data, "refreshing")
	if err != nil {
		return nil, err
	}
	tm.setToken(tok)
	tm.rest.SetAuthHeader(tok.AccessToken)
	authLog.Info("Token refreshed successfully")
	return tok, nil
}

// Authenticate obtains an access token with one request, using either an OTP code or a Smart
// OTP transaction id (never both). For Smart OTP this makes a single attempt and returns a
// SmartOTPPendingError while approval is pending; use EnsureAuthenticated to poll.
func (tm *TokenManager) Authenticate(otp string, transactionID ...string) (*Token, error) {
	txID := ""
	if len(transactionID) > 0 {
		txID = transactionID[0]
	}
	return tm.authenticateOnce(otp, txID)
}

func (tm *TokenManager) authenticateOnce(otp, transactionID string) (*Token, error) {
	if err := tm.requireCredentials("authentication"); err != nil {
		return nil, err
	}
	if otp != "" && transactionID != "" {
		return nil, ssi.NewAuthenticationError("Pass only one of otp or transaction_id, not both", "", 0, nil)
	}
	body := map[string]interface{}{"apiKey": tm.apiKey, "apiSecret": tm.apiSecret}
	if otp != "" {
		body["otp"] = otp
	}
	if transactionID != "" {
		body["transactionId"] = transactionID
	}
	data, err := tm.rest.Post(epAccessToken, body, nil, nil)
	if err != nil {
		if !isSmartOTPPending(err) {
			authLog.Error("Failed to authenticate: %v", err)
		}
		return nil, err
	}
	tok, err := parseToken(data, "authenticating")
	if err != nil {
		return nil, err
	}
	tm.setToken(tok)
	tm.rest.SetAuthHeader(tok.AccessToken)
	authLog.Info("Authentication successful")
	return tok, nil
}

func (tm *TokenManager) SetToken(token *Token) {
	tm.setToken(token)
	tm.rest.SetAuthHeader(token.AccessToken)
	authLog.Info("Access token set manually")
}

// RequestOTP requests an OTP to be sent (normal OTP) or pushed for approval (Smart OTP). Both
// account types share the endpoint: the server decides how to deliver the OTP. Use
// RequestOTPTyped for a model.
func (tm *TokenManager) RequestOTP() (map[string]interface{}, error) {
	if err := tm.requireCredentials("OTP request"); err != nil {
		return nil, err
	}
	body := map[string]interface{}{"apiKey": tm.apiKey, "apiSecret": tm.apiSecret}
	data, err := tm.rest.Post(epRequestOTP, body, nil, nil)
	if err != nil {
		authLog.Error("Failed to request OTP: %v", err)
		return nil, err
	}
	return data, nil
}

// RequestOTPTyped is RequestOTP returning an OTPResponse; TransactionID is set only for Smart OTP.
func (tm *TokenManager) RequestOTPTyped() (*OTPResponse, error) {
	data, err := tm.RequestOTP()
	if err != nil {
		return nil, err
	}
	r := OTPResponseFromMap(data)
	return &r, nil
}

// pollAttempts resolves the Smart OTP poll (interval in ms, attempts); explicit args win over config.
func pollAttempts(cfg *ssi.Config, intervalMs *int, maxRetries *int) (int, int) {
	wait := cfg.OTPPollIntervalMs
	if intervalMs != nil {
		wait = *intervalMs
	}
	if maxRetries != nil {
		if *maxRetries < 1 {
			return wait, 1
		}
		return wait, *maxRetries
	}
	if wait <= 0 {
		return wait, 1
	}
	n := (cfg.OTPPollMaxWaitMs + wait - 1) / wait
	if n < 1 {
		n = 1
	}
	return wait, n
}

// PollSmartOTP polls Authenticate with the transaction id every intervalMs milliseconds for up
// to maxRetries attempts. A pending approval (202/401114) keeps polling; any other error,
// including the rejected/expired/unknown-transaction codes, stops immediately.
func (tm *TokenManager) PollSmartOTP(transactionID string, intervalMs int, maxRetries int) (*Token, error) {
	return tm.pollSmartOTP(context.Background(), transactionID, &intervalMs, &maxRetries)
}

func (tm *TokenManager) pollSmartOTP(ctx context.Context, transactionID string, intervalMs *int, maxRetries *int) (*Token, error) {
	waitMs, attempts := pollAttempts(tm.rest.Config(), intervalMs, maxRetries)
	for attempt := 1; attempt <= attempts; attempt++ {
		tok, err := tm.authenticateOnce("", transactionID)
		if err == nil {
			return tok, nil
		}
		if !isSmartOTPPending(err) {
			return nil, err
		}
		if attempt >= attempts {
			code, status, _ := errCode(err)
			return nil, ssi.NewSmartOTPPendingError(
				fmt.Sprintf("Smart OTP approval not confirmed after %d attempts — ask the user to approve the request on their device, then call again with the same transaction_id", attempts),
				code, status, nil)
		}
		authLog.Info("Smart OTP approval still pending (attempt %d/%d), retrying in %d ms...", attempt, attempts, waitMs)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(waitMs) * time.Millisecond):
		}
	}
	return nil, ssi.NewAuthenticationError("Smart OTP polling failed", "", 0, nil)
}

// EnsureOptions are the inputs of EnsureAuthenticatedWith.
type EnsureOptions struct {
	// OTP is the normal OTP code typed by the user: verified in a single call.
	OTP string
	// TransactionID is the Smart OTP transaction id (from RequestOTP). Approval on the device is
	// asynchronous, so this polls Authenticate until the user approves. If polling runs out
	// while the transaction is still valid, call again with the same TransactionID.
	TransactionID string
	// PollIntervalMs overrides Config.OTPPollIntervalMs (milliseconds).
	PollIntervalMs *int
	// PollMaxRetries overrides Config.OTPPollMaxWaitMs divided by the interval.
	PollMaxRetries *int
}

// EnsureAuthenticated ensures a valid access token is available: a still-valid token is
// returned as is; otherwise it refreshes when the refresh token is usable, else logs in with
// the OTP / Smart OTP transaction id.
//
// Errors: ReauthenticationRequiredError when the session cannot be refreshed and no OTP was
// given; AuthenticationError when there is no token and no OTP/transaction id;
// SmartOTPPendingError / SmartOTPRejectedError for Smart OTP; APIError otherwise.
func (tm *TokenManager) EnsureAuthenticated(otp string, transactionID ...string) (string, error) {
	opts := EnsureOptions{OTP: otp}
	if len(transactionID) > 0 {
		opts.TransactionID = transactionID[0]
	}
	return tm.EnsureAuthenticatedWith(context.Background(), opts)
}

// EnsureAuthenticatedWith is EnsureAuthenticated with explicit options and a context that
// cancels Smart OTP polling.
func (tm *TokenManager) EnsureAuthenticatedWith(ctx context.Context, opts EnsureOptions) (string, error) {
	if t := tm.Token(); t != nil && !tm.IsTokenExpired() {
		return t.AccessToken, nil
	}
	switch {
	case tm.CanRefresh():
		if _, err := tm.refresh(false); err != nil {
			return "", err
		}
	case opts.OTP != "":
		if _, err := tm.authenticateOnce(opts.OTP, ""); err != nil {
			return "", err
		}
	case opts.TransactionID != "":
		if _, err := tm.pollSmartOTP(ctx, opts.TransactionID, opts.PollIntervalMs, opts.PollMaxRetries); err != nil {
			return "", err
		}
	case tm.Token() != nil:
		return "", ssi.NewReauthenticationRequiredError(
			"Session expired and cannot be refreshed — authenticate again with a new OTP", "", 0, nil)
	default:
		return "", ssi.NewAuthenticationError(
			"OTP or Smart OTP transaction_id is required to authenticate — no token available", "", 0, nil)
	}
	return tm.AccessToken(), nil
}

// ConnectRestOnly is a no-op retained for backward compatibility.
//
// Deprecated: Use Authenticate("") for market-data-only (no-OTP) access.
func (tm *TokenManager) ConnectRestOnly() {
	authLog.Error("ConnectRestOnly is deprecated, use Authenticate(otp) instead")
}
