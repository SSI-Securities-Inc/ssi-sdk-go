// Package ssisdk provides shared types for the SSI FastConnect Go SDK.
//
// The main entry point for users is the ssi/ sub-package.
// This root package contains shared error types, configuration, and version information.
package ssisdk

import "fmt"

// SSIError is the base error type for all SDK errors.
type SSIError struct {
	Message      string
	Code         string
	StatusCode   int
	ResponseBody map[string]interface{}
	Headers      map[string]string
}

func (e *SSIError) Error() string {
	return e.Message
}

// AuthenticationError is returned when authentication fails (401/403).
type AuthenticationError struct {
	SSIError
}

// APIError is returned for general API errors (4xx/5xx).
type APIError struct {
	SSIError
}

// WebSocketError is returned for WebSocket connection/message errors.
type WebSocketError struct {
	SSIError
}

// ValidationError is returned when request parameters are invalid.
type ValidationError struct {
	SSIError
}

// RateLimitError is returned when the API rate limit is exceeded (429).
//
// The server sends X-RateLimit-Limit/X-RateLimit-Remaining and, on 429, Retry-After. It does
// not send a reset time, so Reset stays nil unless a caller supplies one.
type RateLimitError struct {
	SSIError
	// RetryAfterMs is how long to wait before retrying, in milliseconds, converted from the
	// Retry-After header (which is in seconds). Nil when the API did not say.
	RetryAfterMs *int
	// Limit is the request quota from X-RateLimit-Limit (requests per second), if sent.
	Limit *int
	// Remaining is the remaining quota from X-RateLimit-Remaining, if sent.
	Remaining *int
	// Reset is the quota reset time exactly as the server sent it (the REST API sends none, so
	// it is nil unless a caller supplies one). It is not converted; its unit is the server's.
	Reset *float64
}

// SmartOTPPendingError: Smart OTP push-approval has not been confirmed on the device yet
// (HTTP 202 / code 401114). Poll again with the same transaction id.
type SmartOTPPendingError struct {
	AuthenticationError
}

// SmartOTPRejectedError: Smart OTP approval can no longer succeed (rejected, expired or unknown
// transaction: codes 401113/401115/401116).
type SmartOTPRejectedError struct {
	AuthenticationError
}

// ReauthenticationRequiredError: the session cannot be refreshed; the user must authenticate
// again with a new OTP. The stored token has been cleared.
type ReauthenticationRequiredError struct {
	AuthenticationError
}

// DuplicateRequestError: HTTP 409. The clientRequestId / batchRequestId was already used today.
// The SDK never retries the order under a new id: the original order may be live.
type DuplicateRequestError struct {
	APIError
}

// TradingWSError is an error frame from the trading WebSocket:
// {"id", "status", "error": {...}}. It never carries the signed params (they hold account numbers).
type TradingWSError struct {
	WebSocketError
	Status        int
	ErrorCode     string
	ServerMessage string
}

// Unwrap lets errors.As match the parent error types.
func (e *AuthenticationError) Unwrap() error { return &e.SSIError }
func (e *APIError) Unwrap() error            { return &e.SSIError }
func (e *WebSocketError) Unwrap() error      { return &e.SSIError }
func (e *ValidationError) Unwrap() error     { return &e.SSIError }
func (e *RateLimitError) Unwrap() error      { return &e.SSIError }

func (e *SmartOTPPendingError) Unwrap() error          { return &e.AuthenticationError }
func (e *SmartOTPRejectedError) Unwrap() error         { return &e.AuthenticationError }
func (e *ReauthenticationRequiredError) Unwrap() error { return &e.AuthenticationError }
func (e *DuplicateRequestError) Unwrap() error         { return &e.APIError }
func (e *TradingWSError) Unwrap() error                { return &e.WebSocketError }

func NewSSIError(message, code string, statusCode int, responseBody map[string]interface{}) *SSIError {
	return &SSIError{
		Message:      message,
		Code:         code,
		StatusCode:   statusCode,
		ResponseBody: responseBody,
	}
}

func NewAuthenticationError(message, code string, statusCode int, responseBody map[string]interface{}) *AuthenticationError {
	return &AuthenticationError{
		SSIError: SSIError{
			Message:      message,
			Code:         code,
			StatusCode:   statusCode,
			ResponseBody: responseBody,
		},
	}
}

func NewAPIError(message, code string, statusCode int, responseBody map[string]interface{}) *APIError {
	return &APIError{
		SSIError: SSIError{
			Message:      message,
			Code:         code,
			StatusCode:   statusCode,
			ResponseBody: responseBody,
		},
	}
}

func NewWebSocketError(message string) *WebSocketError {
	return &WebSocketError{
		SSIError: SSIError{Message: message},
	}
}

func NewValidationError(message string) *ValidationError {
	return &ValidationError{
		SSIError: SSIError{Message: message},
	}
}

func NewRateLimitError(message string, retryAfterMs *int) *RateLimitError {
	return &RateLimitError{
		SSIError:     SSIError{Message: message, Code: "RATE_LIMITED", StatusCode: 429},
		RetryAfterMs: retryAfterMs,
	}
}

func NewSmartOTPPendingError(message, code string, statusCode int, body map[string]interface{}) *SmartOTPPendingError {
	return &SmartOTPPendingError{AuthenticationError: *NewAuthenticationError(message, code, statusCode, body)}
}

func NewSmartOTPRejectedError(message, code string, statusCode int, body map[string]interface{}) *SmartOTPRejectedError {
	return &SmartOTPRejectedError{AuthenticationError: *NewAuthenticationError(message, code, statusCode, body)}
}

func NewReauthenticationRequiredError(message, code string, statusCode int, body map[string]interface{}) *ReauthenticationRequiredError {
	return &ReauthenticationRequiredError{AuthenticationError: *NewAuthenticationError(message, code, statusCode, body)}
}

func NewDuplicateRequestError(message, code string, body map[string]interface{}) *DuplicateRequestError {
	return &DuplicateRequestError{APIError: *NewAPIError(message, code, HTTPStatusConflict, body)}
}

func NewTradingWSError(status int, code, message string) *TradingWSError {
	return &TradingWSError{
		WebSocketError: WebSocketError{SSIError: SSIError{
			Message:    fmt.Sprintf("%d %s: %s", status, code, message),
			Code:       code,
			StatusCode: status,
		}},
		Status:        status,
		ErrorCode:     code,
		ServerMessage: message,
	}
}
