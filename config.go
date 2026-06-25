package ssisdk

const (
	DefaultAPIURL             = "https://api.ssi.com.vn"
	DefaultStreamingURL       = "wss://stream.ssi.com.vn/ws/v3"
	DefaultTimeout            = 60
	DefaultMaxRetries         = 5
	DefaultRetryDelay         = 2
	DefaultRateLimitPerSecond = 10
)

// Config holds configuration for the SSI FastConnect SDK.
type Config struct {
	ClientID           string
	APIURL             string
	StreamingURL       string
	APIKey             string
	APISecret          string
	PrivateKey         string
	Timeout            int
	MaxRetries         int
	RetryDelay         float64
	RateLimitPerSecond int
	LogLevel           string
	Proxy              string
}

// NewConfig creates a new Config with default values.
func NewConfig(clientID string) *Config {
	return &Config{
		ClientID:           clientID,
		APIURL:             DefaultAPIURL,
		StreamingURL:       DefaultStreamingURL,
		Timeout:            DefaultTimeout,
		MaxRetries:         DefaultMaxRetries,
		RetryDelay:         float64(DefaultRetryDelay),
		RateLimitPerSecond: DefaultRateLimitPerSecond,
		LogLevel:           "INFO",
	}
}
