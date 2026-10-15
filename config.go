package ssisdk

import (
	"fmt"
	"strings"
)

// Config holds configuration for the SSI FastConnect SDK.
//
// APIURL, StreamingURL, TradingAPIDomain and TradingWSDomain fall back to their defaults when
// left blank. Clients never mutate the Config they are given: they work on a normalized copy
// (see Normalized), so one Config can be shared between clients.
type Config struct {
	ClientID     string
	APIURL       string
	StreamingURL string
	// TradingAPIDomain is the host of the trading REST API (orders, FCO, account, portfolio).
	// While it stays at the default it follows APIURL. A bare host[:port] gets https://.
	// Auth and market data always use APIURL.
	TradingAPIDomain string
	// TradingWSDomain is the experimental trading WebSocket (default
	// wss://api.ssi.com.vn/ws/v3/trading). A bare host[:port] gets wss:// and /ws/v3/trading.
	TradingWSDomain string
	APIKey          string
	APISecret       string
	PrivateKey      string
	// TimeoutMs is the HTTP request timeout and the WebSocket handshake timeout, in milliseconds.
	TimeoutMs  int
	MaxRetries int
	// RetryDelayMs is the first retry delay in milliseconds; it doubles on every further retry.
	RetryDelayMs int
	// RateLimitPerSecond is the caller's own ceiling (0 = unlimited). The server's advertised
	// limit can only lower it.
	RateLimitPerSecond int
	LogLevel           string
	// MaxBatchOrders is the most orders in one batch request (server default 20). Bigger
	// batches are refused locally.
	MaxBatchOrders int
	Proxy          string
	// UserAgent is an optional User-Agent for HTTP requests and order payloads.
	UserAgent string
	// AutoReconnect re-opens a dropped streaming socket automatically. NewConfig sets it true.
	AutoReconnect bool
	// OTPPollIntervalMs is the time between Smart OTP approval polls, in milliseconds.
	OTPPollIntervalMs int
	// OTPPollMaxWaitMs is the total time to keep polling for Smart OTP approval, in milliseconds.
	OTPPollMaxWaitMs int
}

// NewConfig creates a new Config with default values.
func NewConfig(clientID string) *Config {
	return &Config{
		ClientID:           clientID,
		APIURL:             DefaultAPIURL,
		StreamingURL:       DefaultStreamingURL,
		TradingAPIDomain:   DefaultTradingAPIDomain,
		TradingWSDomain:    DefaultTradingWSDomain,
		TimeoutMs:          DefaultTimeoutMs,
		MaxRetries:         DefaultMaxRetries,
		RetryDelayMs:       DefaultRetryDelayMs,
		RateLimitPerSecond: DefaultRateLimitPerSecond,
		LogLevel:           "INFO",
		MaxBatchOrders:     MaxBatchOrders,
		AutoReconnect:      true,
		OTPPollIntervalMs:  DefaultOTPPollIntervalMs,
		OTPPollMaxWaitMs:   DefaultOTPPollMaxWaitMs,
	}
}

// Normalized returns a copy of c with blank endpoints and invalid numeric settings replaced by
// their defaults. c itself is never modified.
func (c *Config) Normalized() *Config {
	n := *c
	if strings.TrimSpace(n.APIURL) == "" {
		n.APIURL = DefaultAPIURL
	}
	if strings.TrimSpace(n.StreamingURL) == "" {
		n.StreamingURL = DefaultStreamingURL
	}
	if strings.TrimSpace(n.TradingAPIDomain) == "" {
		n.TradingAPIDomain = DefaultTradingAPIDomain
	}
	if strings.TrimSpace(n.TradingWSDomain) == "" {
		n.TradingWSDomain = DefaultTradingWSDomain
	}
	if n.MaxBatchOrders < 1 {
		n.MaxBatchOrders = MaxBatchOrders
	}
	if n.OTPPollIntervalMs <= 0 {
		n.OTPPollIntervalMs = DefaultOTPPollIntervalMs
	}
	if n.OTPPollMaxWaitMs <= 0 {
		n.OTPPollMaxWaitMs = DefaultOTPPollMaxWaitMs
	}
	return &n
}

// TradingAPIURL is the base URL of the trading REST API: TradingAPIDomain, or APIURL when that
// is empty or still the default (so it follows APIURL).
//
// A bare host or host:port becomes https://host; a value with http:// or https:// keeps its
// scheme. Any path or trailing slash is dropped. Another scheme is a ValidationError.
func (c *Config) TradingAPIURL() (string, error) {
	value := strings.TrimSpace(c.TradingAPIDomain)
	apiURL := strings.TrimRight(c.APIURL, "/")
	if apiURL == "" {
		apiURL = DefaultAPIURL
	}
	if value == "" || strings.TrimRight(value, "/") == DefaultTradingAPIDomain {
		return apiURL, nil
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	parts := strings.SplitN(value, "://", 2)
	scheme := strings.ToLower(parts[0])
	if scheme != "http" && scheme != "https" {
		return "", NewValidationError(fmt.Sprintf("TradingAPIDomain must be http(s), got '%s://'", parts[0]))
	}
	host := strings.SplitN(parts[1], "/", 2)[0]
	return scheme + "://" + host, nil
}

// TradingWSURL is the trading WebSocket URL built from TradingWSDomain (always wss:// for a
// bare host). A value that already has a scheme is used as is; the path is appended only when
// the value has none.
func (c *Config) TradingWSURL() string {
	value := strings.TrimRight(strings.TrimSpace(c.TradingWSDomain), "/")
	if value == "" {
		value = DefaultTradingWSDomain
	}
	if !strings.Contains(value, "://") {
		value = "wss://" + value
	}
	rest := value[strings.Index(value, "://")+3:]
	if !strings.Contains(rest, "/") {
		value += TradingWSPath
	}
	return value
}

func maskSecret(s string) string {
	if s == "" {
		return `""`
	}
	return "***"
}

// String never prints APISecret or PrivateKey (and shows APIKey only as set/unset).
func (c Config) String() string {
	return fmt.Sprintf(
		"Config{ClientID:%q APIURL:%q StreamingURL:%q TradingAPIDomain:%q TradingWSDomain:%q "+
			"APIKey:%s APISecret:%s PrivateKey:%s TimeoutMs:%d MaxRetries:%d RetryDelayMs:%d "+
			"RateLimitPerSecond:%d LogLevel:%q MaxBatchOrders:%d Proxy:%s UserAgent:%q "+
			"AutoReconnect:%v OTPPollIntervalMs:%d OTPPollMaxWaitMs:%d}",
		c.ClientID, c.APIURL, c.StreamingURL, c.TradingAPIDomain, c.TradingWSDomain,
		maskSecret(c.APIKey), maskSecret(c.APISecret), maskSecret(c.PrivateKey),
		c.TimeoutMs, c.MaxRetries, c.RetryDelayMs, c.RateLimitPerSecond, c.LogLevel,
		c.MaxBatchOrders, maskSecret(c.Proxy), c.UserAgent, c.AutoReconnect,
		c.OTPPollIntervalMs, c.OTPPollMaxWaitMs,
	)
}

// GoString keeps %#v from leaking secrets as well.
func (c Config) GoString() string { return c.String() }
