package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/ratelimit"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/redact"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

var restLog = logger.New("ssi_sdk.transport.rest")

const (
	HeaderContentType   = ssi.HeaderContentType
	HeaderAccept        = ssi.HeaderAccept
	HeaderAuthorization = ssi.HeaderAuthorization
	HeaderRetryAfter    = ssi.HeaderRetryAfter
	HeaderSignature     = ssi.HeaderSignature
	HeaderUserAgent     = ssi.HeaderUserAgent
	DefaultUserAgent    = ssi.DefaultUserAgent
	ContentTypeJSON     = ssi.ContentTypeJSON
	AuthSchemeBearer    = ssi.AuthSchemeBearer

	httpStatusUnauthorized   = 401
	httpStatusForbidden      = 403
	httpStatusRateLimit      = 429
	httpStatusNoContent      = 204
	httpStatusErrorThreshold = 400
)

// RateLimitInfo holds the last X-RateLimit-Limit / X-RateLimit-Remaining the server sent
// (nil when the header was absent).
type RateLimitInfo struct {
	Limit     *int
	Remaining *int
}

// Request describes one REST call.
type Request struct {
	Method string
	Path   string
	Params map[string]string
	// Query carries repeated query keys (for example several processStatus values); it is merged
	// with Params.
	Query   url.Values
	JSON    map[string]interface{}
	Content []byte
	Headers map[string]string
	// Retry controls replay after a timeout. nil means GET only: a timed-out POST/PUT/DELETE
	// may already have been processed (an order, an OTP), so replaying it is not safe.
	Retry *bool
}

// RestClient is the HTTP client for SSI REST API. It is safe for concurrent use.
type RestClient struct {
	config      *ssi.Config
	rateLimiter *ratelimit.Limiter
	client      *http.Client
	headers     map[string]string
	initErr     error
	mu          sync.Mutex
	rateInfo    RateLimitInfo
}

// NewRestClient builds a client from a normalized copy of config (the caller's Config is never
// mutated).
func NewRestClient(config *ssi.Config) *RestClient {
	cfg := config.Normalized()
	logger.SetLevelFromString(cfg.LogLevel)
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		ForceAttemptHTTP2:   true,
		DisableCompression:  true,
	}
	var initErr error
	if cfg.Proxy != "" {
		proxyURL, err := url.Parse(cfg.Proxy)
		if err != nil || proxyURL.Host == "" {
			// Fail closed: never fall back to a direct connection when a proxy was requested.
			initErr = ssi.NewValidationError("Proxy is not a valid URL")
		} else {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}
	ua := cfg.UserAgent
	if ua == "" {
		ua = DefaultUserAgent
	}
	return &RestClient{
		config:      cfg,
		rateLimiter: ratelimit.New(cfg.RateLimitPerSecond),
		initErr:     initErr,
		client: &http.Client{
			Timeout:   time.Duration(cfg.TimeoutMs) * time.Millisecond,
			Transport: transport,
		},
		headers: map[string]string{
			HeaderContentType: ContentTypeJSON,
			HeaderAccept:      ContentTypeJSON,
			HeaderUserAgent:   ua,
		},
	}
}

// Config returns the normalized configuration this client was built with.
func (c *RestClient) Config() *ssi.Config { return c.config }

// PrivateKey returns the configured private key used for signing requests.
func (c *RestClient) PrivateKey() string { return c.config.PrivateKey }

// RateLimit returns the last quota numbers the server reported.
func (c *RestClient) RateLimit() RateLimitInfo {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rateInfo
}

func (c *RestClient) SetAuthHeader(token string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.headers[HeaderAuthorization] = AuthSchemeBearer + token
}

// ClearAuthHeader drops the bearer token so a dead token is no longer sent.
func (c *RestClient) ClearAuthHeader() {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.headers, HeaderAuthorization)
}

// route returns the base URL for path: the trading host for trading/account paths (it equals
// APIURL unless Config.TradingAPIDomain points elsewhere), otherwise APIURL.
func route(cfg *ssi.Config, path string) (string, error) {
	for _, prefix := range ssi.TradingAPIPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return cfg.TradingAPIURL()
		}
	}
	return strings.TrimRight(cfg.APIURL, "/"), nil
}

func headerInt(h http.Header, name string) *int {
	v := strings.TrimSpace(h.Get(name))
	if v == "" {
		return nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return nil
	}
	return &n
}

// retryAfterMs parses Retry-After (delta-seconds on the wire) into milliseconds; an HTTP-date
// or garbage yields nil.
func retryAfterMs(v string) *int {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f < 0 || math.IsNaN(f) || math.IsInf(f, 0) {
		return nil
	}
	ms := int(math.Round(math.Min(f, 1e6) * 1000))
	return &ms
}

// serverCode returns the server's "code" from a {code, msg} body.
func serverCode(body map[string]interface{}) (num int, text string, ok bool) {
	if body == nil || body["code"] == nil {
		return 0, "", false
	}
	raw := body["code"]
	if n := util.ToOptInt(raw); n != nil {
		return *n, strconv.Itoa(*n), true
	}
	return 0, util.ToStr(raw), true
}

// serverMessage returns the server's msg/message. A raw body that is not a {code, msg} object
// is never echoed: it could carry anything.
func serverMessage(body map[string]interface{}) string {
	for _, key := range []string{"msg", "message"} {
		if v, ok := body[key]; ok && v != nil {
			if s := util.ToStr(v); s != "" {
				return s
			}
		}
	}
	return ""
}

func flatHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, v := range h {
		if len(v) > 0 {
			out[k] = v[0]
		}
	}
	return redact.Redact(out).(map[string]string)
}

// errorFor builds the error for a non-2xx response (or a 2xx {code, msg} failure). The message
// carries only the server's code and msg, never the raw body or URL.
func errorFor(resp *http.Response, body map[string]interface{}) error {
	status := resp.StatusCode
	num, codeText, hasCode := serverCode(body)
	detail := serverMessage(body)
	message := fmt.Sprintf("API error %d", status)
	if hasCode {
		message += " [" + codeText + "]"
	}
	if detail != "" {
		message += ": " + detail
	}
	shown := strconv.Itoa(status)
	if hasCode {
		shown = codeText
	}
	var safeBody map[string]interface{}
	if body != nil {
		safeBody = redact.RedactMap(body)
	}
	headers := flatHeaders(resp.Header)

	switch {
	case status == httpStatusRateLimit:
		rl := ssi.NewRateLimitError(message, retryAfterMs(resp.Header.Get(HeaderRetryAfter)))
		rl.Code = shown
		rl.ResponseBody = safeBody
		rl.Headers = headers
		rl.Limit = headerInt(resp.Header, ssi.HeaderRateLimitLimit)
		rl.Remaining = headerInt(resp.Header, ssi.HeaderRateLimitRemaining)
		return rl
	case status == httpStatusUnauthorized || status == httpStatusForbidden:
		if hasCode && ssi.SmartOTPTerminalCodes[num] {
			e := ssi.NewSmartOTPRejectedError(message, shown, status, safeBody)
			e.Headers = headers
			return e
		}
		e := ssi.NewAuthenticationError(message, shown, status, safeBody)
		e.Headers = headers
		return e
	case status == ssi.HTTPStatusConflict:
		e := ssi.NewDuplicateRequestError(message, shown, safeBody)
		e.Headers = headers
		return e
	}
	e := ssi.NewAPIError(message, shown, status, safeBody)
	e.Headers = headers
	return e
}

// handleResponse parses and validates an HTTP response. Order matters: a pending Smart OTP
// arrives as HTTP 202 (not an error status), and a failure can arrive as HTTP 200 with a
// {code, msg} body. An empty result is {"code": 204, ...} (see util.IsNoContent).
func (c *RestClient) handleResponse(resp *http.Response) (map[string]interface{}, error) {
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ssi.NewAPIError("failed to read response body", strconv.Itoa(resp.StatusCode), resp.StatusCode, nil)
	}

	c.observeRateLimit(resp)
	restLog.Debug("<-- Status: %s", resp.Status)

	var body map[string]interface{}
	var arr []interface{}
	if len(bytes.TrimSpace(raw)) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			body = nil
			if err2 := json.Unmarshal(raw, &arr); err2 != nil {
				arr = nil
			}
		}
	}

	if resp.StatusCode >= httpStatusErrorThreshold {
		return nil, errorFor(resp, body)
	}
	if resp.StatusCode == httpStatusNoContent {
		return map[string]interface{}{}, nil
	}

	if resp.StatusCode == ssi.SmartOTPPendingStatus && body != nil {
		if num, _, ok := serverCode(body); ok && num == ssi.SmartOTPPendingCode {
			e := ssi.NewSmartOTPPendingError(
				fmt.Sprintf("Smart OTP approval pending [%d]: %s", ssi.SmartOTPPendingCode, serverMessage(body)),
				strconv.Itoa(ssi.SmartOTPPendingCode), resp.StatusCode, redact.RedactMap(body))
			e.Headers = flatHeaders(resp.Header)
			return nil, e
		}
	}
	if body == nil {
		if arr != nil {
			// A JSON array body is wrapped so callers receive {"data": [...]}.
			return map[string]interface{}{"data": arr}, nil
		}
		e := ssi.NewAPIError(fmt.Sprintf("API returned a non-JSON body (HTTP %d)", resp.StatusCode),
			strconv.Itoa(resp.StatusCode), resp.StatusCode, nil)
		e.Headers = flatHeaders(resp.Header)
		return nil, e
	}
	_, hasMsg := body["msg"]
	_, hasMessage := body["message"]
	if hasMsg || hasMessage {
		if num := util.ToOptInt(body["code"]); num != nil && *num != 200 && *num != 204 {
			return nil, errorFor(resp, body)
		}
	}
	return body, nil
}

// observeRateLimit follows the server's quota headers: never exceed X-RateLimit-Limit per
// second, and rest for a second when X-RateLimit-Remaining hits 0.
func (c *RestClient) observeRateLimit(resp *http.Response) {
	info := RateLimitInfo{
		Limit:     headerInt(resp.Header, ssi.HeaderRateLimitLimit),
		Remaining: headerInt(resp.Header, ssi.HeaderRateLimitRemaining),
	}
	c.mu.Lock()
	c.rateInfo = info
	c.mu.Unlock()
	if info.Limit != nil {
		c.rateLimiter.Cap(*info.Limit)
	}
	if info.Remaining != nil && *info.Remaining == 0 {
		c.rateLimiter.PauseMs(1000)
	}
}

func (c *RestClient) once(ctx context.Context, req Request, baseURL string) (map[string]interface{}, error) {
	reqURL := baseURL + req.Path
	if len(req.Params) > 0 || len(req.Query) > 0 {
		q := url.Values{}
		for k, vs := range req.Query {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		for k, v := range req.Params {
			q.Set(k, v)
		}
		reqURL += "?" + q.Encode()
	}

	var bodyBytes []byte
	if req.Content != nil {
		bodyBytes = req.Content
	} else if req.JSON != nil {
		b, err := json.Marshal(req.JSON)
		if err != nil {
			return nil, ssi.NewValidationError("request body cannot be encoded as JSON")
		}
		bodyBytes = b
	}
	var bodyReader io.Reader
	if bodyBytes != nil {
		bodyReader = bytes.NewReader(bodyBytes)
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, reqURL, bodyReader)
	if err != nil {
		return nil, ssi.NewValidationError("invalid request")
	}

	c.mu.Lock()
	for k, v := range c.headers {
		httpReq.Header.Set(k, v)
	}
	c.mu.Unlock()
	for k, v := range req.Headers {
		httpReq.Header.Set(k, v)
	}

	restLog.Debug("[%s] %s", req.Method, req.Path)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		// url.Error embeds the full URL (query string included): log the cause only.
		var ue *url.Error
		if errors.As(err, &ue) {
			restLog.Error("[%s] %s -> error: %v", req.Method, req.Path, ue.Err)
		} else {
			restLog.Error("[%s] %s -> error: %v", req.Method, req.Path, err)
		}
		return nil, err
	}
	return c.handleResponse(resp)
}

func isTimeoutError(err error) bool {
	var ne net.Error
	return errors.As(err, &ne) && ne.Timeout()
}

// rateWait is how long to wait after a 429 in milliseconds (Retry-After, else 1s doubling), or
// ok=false to give up.
func rateWait(e *ssi.RateLimitError, attempt int) (waitMs int, ok bool) {
	wait := ssi.RateLimitDefaultWaitMs << uint(attempt)
	if e.RetryAfterMs != nil && *e.RetryAfterMs > 0 {
		wait = *e.RetryAfterMs
	}
	return wait, wait <= ssi.RateLimitMaxWaitMs
}

// Do sends a request: it honours the server's rate-limit headers, waits out a 429 on a GET (at
// most RateLimitMaxRetries times, never longer than RateLimitMaxWait), and retries timeouts
// with exponential backoff only when the request is replayable (GET unless forced).
func (c *RestClient) Do(ctx context.Context, req Request) (map[string]interface{}, error) {
	if c.initErr != nil {
		return nil, c.initErr
	}
	req.Method = strings.ToUpper(req.Method)
	replayable := req.Method == "GET"
	if req.Retry != nil {
		replayable = *req.Retry
	}
	baseURL, err := route(c.config, req.Path)
	if err != nil {
		return nil, err
	}

	attempt := 0
	for {
		if err := c.rateLimiter.AcquireCtx(ctx); err != nil {
			return nil, err
		}
		result, err := c.withTimeoutRetry(ctx, req, baseURL, replayable)
		if err == nil {
			return result, nil
		}
		var rl *ssi.RateLimitError
		if errors.As(err, &rl) {
			waitMs, ok := rateWait(rl, attempt)
			if !replayable || !ok || attempt >= ssi.RateLimitMaxRetries {
				return nil, err
			}
			restLog.Info("429 on %s: waiting %d ms (attempt %d)", req.Path, waitMs, attempt+1)
			c.rateLimiter.PauseMs(waitMs)
			attempt++
			continue
		}
		return nil, err
	}
}

func (c *RestClient) withTimeoutRetry(ctx context.Context, req Request, baseURL string, replayable bool) (map[string]interface{}, error) {
	maxRetries := c.config.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	for attempt := 0; ; attempt++ {
		result, err := c.once(ctx, req, baseURL)
		if err == nil {
			return result, nil
		}
		if !replayable || !isTimeoutError(err) || ctx.Err() != nil || attempt >= maxRetries {
			return nil, err
		}
		delay := time.Duration(c.config.RetryDelayMs) * time.Millisecond << uint(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
		}
	}
}

// Request is the positional form of Do, with context.Background().
func (c *RestClient) Request(method, path string, params map[string]string, jsonBody map[string]interface{}, content []byte, headers map[string]string) (map[string]interface{}, error) {
	return c.Do(context.Background(), Request{
		Method: method, Path: path, Params: params, JSON: jsonBody, Content: content, Headers: headers,
	})
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
