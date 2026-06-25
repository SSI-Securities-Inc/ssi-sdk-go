package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/ratelimit"
)

var restLog = logger.New("ssi_sdk.transport.rest")

const (
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderAuthorization = "Authorization"
	HeaderRetryAfter    = "Retry-After"
	HeaderSignature     = "X-Signature"
	ContentTypeJSON     = "application/json"
	AuthSchemeBearer    = "Bearer "

	httpStatusUnauthorized   = 401
	httpStatusForbidden      = 403
	httpStatusRateLimit      = 429
	httpStatusNoContent      = 204
	httpStatusErrorThreshold = 400
)

// RestClient is the synchronous HTTP client for SSI REST API.
type RestClient struct {
	config      *ssi.Config
	rateLimiter *ratelimit.Limiter
	client      *http.Client
	headers     map[string]string
	mu          sync.Mutex
}

func NewRestClient(config *ssi.Config) *RestClient {
	logger.SetLevelFromString(config.LogLevel)
	transport := &http.Transport{}
	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &RestClient{
		config:      config,
		rateLimiter: ratelimit.New(config.RateLimitPerSecond),
		client: &http.Client{
			Timeout:   time.Duration(config.Timeout) * time.Second,
			Transport: transport,
		},
		headers: map[string]string{
			HeaderContentType: ContentTypeJSON,
			HeaderAccept:      ContentTypeJSON,
		},
	}
}

func (c *RestClient) SetAuthHeader(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[HeaderAuthorization] = AuthSchemeBearer + token
}

func (c *RestClient) handleResponse(resp *http.Response) (map[string]interface{}, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	restLog.Debug("<-- Status: %s | Body: %s", resp.Status, string(body))

	if resp.StatusCode == httpStatusUnauthorized || resp.StatusCode == httpStatusForbidden {
		var respBody map[string]interface{}
		_ = json.Unmarshal(body, &respBody)
		return nil, ssi.NewAuthenticationError(
			fmt.Sprintf("Authentication failed: %d", resp.StatusCode),
			strconv.Itoa(resp.StatusCode),
			resp.StatusCode,
			respBody,
		)
	}

	if resp.StatusCode == httpStatusRateLimit {
		var retryAfter *float64
		if ra := resp.Header.Get(HeaderRetryAfter); ra != "" {
			if v, err := strconv.ParseFloat(ra, 64); err == nil {
				retryAfter = &v
			}
		}
		return nil, ssi.NewRateLimitError("Rate limit exceeded", retryAfter)
	}

	if resp.StatusCode >= httpStatusErrorThreshold {
		var respBody map[string]interface{}
		_ = json.Unmarshal(body, &respBody)
		return nil, ssi.NewAPIError(
			fmt.Sprintf("API error: %d", resp.StatusCode),
			strconv.Itoa(resp.StatusCode),
			resp.StatusCode,
			respBody,
		)
	}

	if resp.StatusCode == httpStatusNoContent {
		return map[string]interface{}{}, nil
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Body may be a JSON array; wrap it so callers receive {"data": [...]}
		var arr []interface{}
		if err2 := json.Unmarshal(body, &arr); err2 != nil {
			return nil, fmt.Errorf("failed to parse JSON response: %w", err)
		}
		return map[string]interface{}{"data": arr}, nil
	}
	return result, nil
}

func (c *RestClient) Request(method, path string, params map[string]string, jsonBody map[string]interface{}, content []byte, headers map[string]string) (map[string]interface{}, error) {
	c.rateLimiter.Acquire()

	var doRequest func() (map[string]interface{}, error)
	doRequest = func() (map[string]interface{}, error) {
		reqURL := c.config.APIURL + path
		if len(params) > 0 {
			q := url.Values{}
			for k, v := range params {
				q.Set(k, v)
			}
			reqURL += "?" + q.Encode()
		}

		var bodyBytes []byte
		if content != nil {
			bodyBytes = content
		} else if jsonBody != nil {
			b, err := json.Marshal(jsonBody)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
			}
			bodyBytes = b
		}

		var bodyReader io.Reader
		if bodyBytes != nil {
			bodyReader = bytes.NewReader(bodyBytes)
		}

		req, err := http.NewRequest(method, reqURL, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		c.mu.Lock()
		for k, v := range c.headers {
			if k == HeaderAuthorization && path == "/api/v3/auth/refresh" {
				continue
			}
			req.Header.Set(k, v)
		}
		c.mu.Unlock()

		// Per-request header overrides (if any).
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		restLog.Debug("[%s] %s | params: %v | body: %s", method, path, params, string(bodyBytes))

		resp, err := c.client.Do(req)
		if err != nil {
			restLog.Error("[%s] %s -> error: %v", method, path, err)
			return nil, err
		}
		restLog.Debug("Received response: %s", resp.Status)
		return c.handleResponse(resp)
	}

	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		result, err := doRequest()
		if err != nil {
			if isTimeoutError(err) {
				lastErr = err
				if attempt < c.config.MaxRetries {
					sleepDuration := c.config.RetryDelay * math.Pow(2, float64(attempt))
					time.Sleep(time.Duration(sleepDuration * float64(time.Second)))
					continue
				}
			}
			return nil, err
		}
		return result, nil
	}
	return nil, lastErr
}

func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	if netErr, ok := err.(interface{ Timeout() bool }); ok {
		return netErr.Timeout()
	}
	return false
}

func (c *RestClient) Get(path string, params map[string]string, headers map[string]string) (map[string]interface{}, error) {
	return c.Request("GET", path, params, nil, nil, headers)
}

func (c *RestClient) Post(path string, jsonBody map[string]interface{}, content []byte, headers map[string]string) (map[string]interface{}, error) {
	return c.Request("POST", path, nil, jsonBody, content, headers)
}

func (c *RestClient) Put(path string, jsonBody map[string]interface{}, content []byte, headers map[string]string) (map[string]interface{}, error) {
	return c.Request("PUT", path, nil, jsonBody, content, headers)
}

func (c *RestClient) Delete(path string, jsonBody map[string]interface{}, content []byte, headers map[string]string) (map[string]interface{}, error) {
	return c.Request("DELETE", path, nil, jsonBody, content, headers)
}

func (c *RestClient) Close() {
	c.client.CloseIdleConnections()
	restLog.Info("REST client closed")
}
