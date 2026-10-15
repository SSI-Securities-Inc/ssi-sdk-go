package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
)

func testConfig(apiURL string) *ssi.Config {
	cfg := ssi.NewConfig("test-client")
	cfg.APIURL = apiURL
	cfg.LogLevel = "ERROR"
	cfg.RateLimitPerSecond = 0
	cfg.MaxRetries = 0
	return cfg
}

func TestAPIErrorCodeIsServerCode(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"code":400123,"msg":"bad symbol","accessToken":"SHOULD-NOT-LEAK-12345"}`))
	}))
	defer srv.Close()
	_, err := NewRestClient(testConfig(srv.URL)).Get("/api/v3/market/x", nil, nil)
	var ae *ssi.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("want APIError, got %T %v", err, err)
	}
	if ae.Code != "400123" || ae.StatusCode != 400 {
		t.Fatalf("code=%q status=%d", ae.Code, ae.StatusCode)
	}
	if !strings.Contains(ae.Message, "bad symbol") {
		t.Fatalf("server msg missing: %s", ae.Message)
	}
	if ae.ResponseBody["accessToken"] == "SHOULD-NOT-LEAK-12345" {
		t.Fatal("response body kept on the error must be redacted")
	}
}

func TestErrorDoesNotEchoRawBodyOrURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		_, _ = w.Write([]byte(`<html>stack trace /srv/app/secret.go SECRET-RAW</html>`))
	}))
	defer srv.Close()
	_, err := NewRestClient(testConfig(srv.URL)).Get("/api/v3/market/x", map[string]string{"token": "QUERY-SECRET"}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, leak := range []string{"SECRET-RAW", "secret.go", "QUERY-SECRET", srv.URL} {
		if strings.Contains(err.Error(), leak) {
			t.Fatalf("error leaks %q: %v", leak, err)
		}
	}
}

func TestRateLimitGETIsWaitedOut(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&hits, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(429)
			_, _ = w.Write([]byte(`{"code":429,"msg":"slow down"}`))
			return
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	c := NewRestClient(testConfig(srv.URL))
	out, err := c.Get("/api/v3/market/x", nil, nil)
	if err != nil || out["ok"] != true {
		t.Fatalf("GET after 429 should succeed: %v %v", out, err)
	}
	if atomic.LoadInt32(&hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", hits)
	}
}

func TestRateLimitPOSTIsNeverReplayed(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Retry-After", "1")
		w.Header().Set("X-RateLimit-Limit", "10")
		w.Header().Set("X-RateLimit-Remaining", "3")
		w.WriteHeader(429)
		_, _ = w.Write([]byte(`{"code":429,"msg":"slow down"}`))
	}))
	defer srv.Close()
	c := NewRestClient(testConfig(srv.URL))
	_, err := c.Post("/api/v3/trading/order", map[string]interface{}{"a": 1}, nil, nil)
	var rl *ssi.RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("want RateLimitError, got %v", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("a POST must be sent once, got %d", hits)
	}
	if rl.RetryAfterMs == nil || *rl.RetryAfterMs != 1000 || rl.Limit == nil || *rl.Limit != 10 || rl.Remaining == nil || *rl.Remaining != 3 {
		t.Fatalf("rate limit fields: %+v", rl)
	}
	if info := c.RateLimit(); info.Limit == nil || *info.Limit != 10 {
		t.Fatalf("RateLimit() not recorded: %+v", info)
	}
}

func TestRateLimitLongWaitIsReturned(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(429)
	}))
	defer srv.Close()
	_, err := NewRestClient(testConfig(srv.URL)).Get("/api/v3/market/x", nil, nil)
	var rl *ssi.RateLimitError
	if !errors.As(err, &rl) || atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("a wait above the cap must not be slept: err=%v hits=%d", err, hits)
	}
}

func TestPendingSmartOTPAndDuplicate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/pending") {
			w.WriteHeader(202)
			_, _ = w.Write([]byte(`{"code":401114,"msg":"waiting"}`))
			return
		}
		w.WriteHeader(409)
		_, _ = w.Write([]byte(`{"code":409001,"msg":"duplicate clientRequestId"}`))
	}))
	defer srv.Close()
	c := NewRestClient(testConfig(srv.URL))

	_, err := c.Post("/api/v3/auth/pending", map[string]interface{}{}, nil, nil)
	var pe *ssi.SmartOTPPendingError
	if !errors.As(err, &pe) || pe.Code != "401114" {
		t.Fatalf("want SmartOTPPendingError, got %T %v", err, err)
	}

	_, err = c.Post("/api/v3/trading/order", map[string]interface{}{}, nil, nil)
	var de *ssi.DuplicateRequestError
	if !errors.As(err, &de) || de.StatusCode != 409 {
		t.Fatalf("want DuplicateRequestError, got %T %v", err, err)
	}
}

func TestSmartOTPTerminalCodeIsRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"code":401115,"msg":"expired"}`))
	}))
	defer srv.Close()
	_, err := NewRestClient(testConfig(srv.URL)).Post("/api/v3/auth/token", map[string]interface{}{}, nil, nil)
	var re *ssi.SmartOTPRejectedError
	if !errors.As(err, &re) {
		t.Fatalf("want SmartOTPRejectedError, got %T %v", err, err)
	}
}

func TestHTTP200WithFailureBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"code":500100,"msg":"internal"}`))
	}))
	defer srv.Close()
	_, err := NewRestClient(testConfig(srv.URL)).Get("/api/v3/market/x", nil, nil)
	var ae *ssi.APIError
	if !errors.As(err, &ae) || ae.Code != "500100" {
		t.Fatalf("a {code,msg} failure on HTTP 200 must be an error, got %v", err)
	}
}

func TestArrayBodyIsWrappedAndNoContentIsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/empty") {
			w.WriteHeader(204)
			return
		}
		_, _ = w.Write([]byte(`[{"a":1}]`))
	}))
	defer srv.Close()
	c := NewRestClient(testConfig(srv.URL))
	out, err := c.Get("/api/v3/market/list", nil, nil)
	if err != nil || len(out["data"].([]interface{})) != 1 {
		t.Fatalf("array wrap: %v %v", out, err)
	}
	out, err = c.Get("/api/v3/market/empty", nil, nil)
	if err != nil || len(out) != 0 {
		t.Fatalf("204: %v %v", out, err)
	}
}

func TestTradingHostRouting(t *testing.T) {
	var apiHits, tradingHits int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiHits, 1)
		_, _ = w.Write([]byte(`{"host":"api"}`))
	}))
	defer api.Close()
	trd := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tradingHits, 1)
		_, _ = w.Write([]byte(`{"host":"trading"}`))
	}))
	defer trd.Close()

	cfg := testConfig(api.URL)
	cfg.TradingAPIDomain = trd.URL
	c := NewRestClient(cfg)
	for path, want := range map[string]string{
		"/api/v3/trading/order":   "trading",
		"/api/v3/account/balance": "trading",
		"/api/v3/market/ohlc":     "api",
		"/api/v3/auth/token":      "api",
	} {
		out, err := c.Get(path, nil, nil)
		if err != nil || out["host"] != want {
			t.Errorf("%s -> %v %v, want host %s", path, out, err, want)
		}
	}
}

func TestTradingHostFollowsAPIURLByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"host":"api"}`))
	}))
	defer srv.Close()
	out, err := NewRestClient(testConfig(srv.URL)).Get("/api/v3/trading/order", nil, nil)
	if err != nil || out["host"] != "api" {
		t.Fatalf("%v %v", out, err)
	}
}

func TestBadProxyFailsClosed(t *testing.T) {
	var hits int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	cfg := testConfig(srv.URL)
	cfg.Proxy = "://not a url"
	_, err := NewRestClient(cfg).Get("/api/v3/market/x", nil, nil)
	var ve *ssi.ValidationError
	if !errors.As(err, &ve) || atomic.LoadInt32(&hits) != 0 {
		t.Fatalf("a bad proxy must not fall back to a direct connection: err=%v hits=%d", err, hits)
	}
}

func TestConfigNotMutated(t *testing.T) {
	cfg := testConfig("")
	cfg.TradingAPIDomain = ""
	_ = NewRestClient(cfg)
	if cfg.APIURL != "" || cfg.TradingAPIDomain != "" {
		t.Fatal("NewRestClient must work on a copy of the Config")
	}
}

func TestRefreshPathSendsStoredAuthorizationLikePython(t *testing.T) {
	var gotAuth atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth.Store(r.Header.Get("Authorization"))
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := NewRestClient(testConfig(srv.URL))
	c.SetAuthHeader("tok")
	if _, err := c.Post("/api/v3/auth/refresh", map[string]interface{}{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth.Load() != "Bearer tok" {
		t.Fatalf("refresh carries the stored token, as in Python: %v", gotAuth.Load())
	}
	if _, err := c.Get("/api/v3/market/x", nil, nil); err != nil {
		t.Fatal(err)
	}
	if gotAuth.Load() != "Bearer tok" {
		t.Fatalf("normal calls carry the token: %v", gotAuth.Load())
	}
	c.ClearAuthHeader()
	_, _ = c.Get("/api/v3/market/x", nil, nil)
	if gotAuth.Load() != "" {
		t.Fatal("ClearAuthHeader must stop sending the token")
	}
}

func TestDebugLogCarriesOnlyMethodAndPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"accountNo":"ACC-123456","accessToken":"TOKEN-SECRET-999"}}`))
	}))
	defer srv.Close()
	var buf strings.Builder
	logger.SetOutput(&buf)
	defer func() {
		logger.SetOutput(os.Stderr)
		logger.SetLevel(logger.LevelError)
	}()
	body := map[string]interface{}{"accountNo": "ACC-123456", "price": "25.5"}
	cfg := testConfig(srv.URL)
	cfg.LogLevel = "DEBUG"
	if _, err := NewRestClient(cfg).Post("/api/v3/trading/order", body, nil, nil); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "/api/v3/trading/order") {
		t.Fatalf("method/path missing: %q", out)
	}
	for _, leak := range []string{"ACC-123456", "TOKEN-SECRET-999", "25.5", "accountNo"} {
		if strings.Contains(out, leak) {
			t.Fatalf("debug log leaked %q: %s", leak, out)
		}
	}
}

func TestRetryAfterMs(t *testing.T) {
	cases := []struct {
		in   string
		want int
		ok   bool
	}{{"1", 1000, true}, {"0.5", 500, true}, {"3600", 3600000, true}, {"", 0, false}, {"-1", 0, false}, {"Wed, 21 Oct 2026 07:28:00 GMT", 0, false}}
	for _, c := range cases {
		got := retryAfterMs(c.in)
		if (got != nil) != c.ok || (c.ok && *got != c.want) {
			t.Errorf("%q: got %v, want %v (ok=%v)", c.in, got, c.want, c.ok)
		}
	}
}

func TestRateWaitMs(t *testing.T) {
	if w, ok := rateWait(&ssi.RateLimitError{}, 2); w != 4000 || !ok {
		t.Fatalf("default doubles from 1000 ms: %d %v", w, ok)
	}
	ms := 31000
	if _, ok := rateWait(&ssi.RateLimitError{RetryAfterMs: &ms}, 0); ok {
		t.Fatal("a wait above RateLimitMaxWaitMs must be refused")
	}
}
