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
type RateLimitError struct {
	SSIError
	RetryAfter *float64
}

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

func NewRateLimitError(message string, retryAfter *float64) *RateLimitError {
	return &RateLimitError{
		SSIError:   SSIError{Message: message, Code: "RATE_LIMITED"},
		RetryAfter: retryAfter,
	}
}

func RequireNonEmpty(value, fieldName string) error {
	if value == "" {
		return NewValidationError(fmt.Sprintf("%s is required and cannot be empty", fieldName))
	}
	return nil
}

func RequirePositive(value float64, fieldName string) error {
	if value <= 0 {
		return NewValidationError(fmt.Sprintf("%s must be positive, got %v", fieldName, value))
	}
	return nil
}

func RequireNonNegative(value float64, fieldName string) error {
	if value < 0 {
		return NewValidationError(fmt.Sprintf("%s must be non-negative, got %v", fieldName, value))
	}
	return nil
}
