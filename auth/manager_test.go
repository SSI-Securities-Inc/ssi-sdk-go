package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

func newManager(t *testing.T, h http.HandlerFunc) (*TokenManager, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	cfg := ssi.NewConfig("c")
	cfg.APIURL = srv.URL
	cfg.LogLevel = "ERROR"
	cfg.RateLimitPerSecond = 0
	return NewTokenManager(transport.NewRestClient(cfg), "key", "secret"), srv
}

func tokenBody(access string, expiresIn int64) string {
	now := time.Now().Unix()
	return fmt.Sprintf(`{"accessToken":%q,"tokenType":"Bearer","expiresAt":%d,"refreshToken":"r-%s","refreshExpiresAt":%d}`,
		access, now+expiresIn, access, now+86400)
}

func TestTokenExpiredSkewAndUnknown(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	if !tokenExpired(nil, now) {
		t.Fatal("no token is expired")
	}
	if !tokenExpired(&Token{AccessToken: "a", ExpiresAt: 0}, now) {
		t.Fatal("unknown expiry must count as expired")
	}
	if tokenExpired(&Token{AccessToken: "a", ExpiresAt: 1_000_000 + ssi.TokenExpirySkewMs/1000 + 1}, now) {
		t.Fatal("valid beyond the skew")
	}
	if !tokenExpired(&Token{AccessToken: "a", ExpiresAt: 1_000_000 + ssi.TokenExpirySkewMs/1000}, now) {
		t.Fatal("inside the 30s skew counts as expired")
	}
}

func TestTokenFromMapNormalizesMilliseconds(t *testing.T) {
	tk := TokenFromMap(map[string]interface{}{"accessToken": "a", "expiresAt": float64(1_700_000_000_000), "refreshExpiresAt": "garbage"})
	if tk.ExpiresAt != 1_700_000_000 || tk.RefreshTokenExpiresAt != 0 || tk.TokenType != "Bearer" {
		t.Fatalf("%+v", tk)
	}
	if (&Token{ExpiresAt: 50, RefreshTokenExpiresAt: 20}).EffectiveExpiresAt() != 20 {
		t.Fatal("effective expiry is the earlier known one")
	}
	if (&Token{ExpiresAt: 50}).EffectiveExpiresAt() != 50 || (&Token{}).EffectiveExpiresAt() != 0 {
		t.Fatal("unknown values are ignored")
	}
}

func TestTokenStringNeverPrintsSecrets(t *testing.T) {
	tk := Token{AccessToken: "ACCESS-SECRET-VALUE", RefreshToken: "REFRESH-SECRET-VALUE", TokenType: "Bearer"}
	for _, s := range []string{tk.String(), fmt.Sprintf("%v", tk), fmt.Sprintf("%+v", &tk), fmt.Sprintf("%#v", tk)} {
		if strings.Contains(s, "SECRET-VALUE") {
			t.Fatalf("leak: %s", s)
		}
	}
	req := TokenRequest{APIKey: "KEY-SECRET-VALUE", APISecret: "API-SECRET-VALUE", OTP: "123456"}
	if s := fmt.Sprintf("%v %#v", req, req); strings.Contains(s, "SECRET-VALUE") || strings.Contains(s, "123456") {
		t.Fatalf("request leak: %s", s)
	}
}

func TestAuthenticateIsASingleCall(t *testing.T) {
	var hits int32
	var body map[string]interface{}
	tm, _ := newManager(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`{"code":401114,"msg":"waiting"}`))
	})
	_, err := tm.Authenticate("", "tx-1")
	var pe *ssi.SmartOTPPendingError
	if !errors.As(err, &pe) {
		t.Fatalf("want pending, got %v", err)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("Authenticate must not poll, hits=%d", hits)
	}
	if body["transactionId"] != "tx-1" || body["otp"] != nil {
		t.Fatalf("body: %v", body)
	}
}

func TestAuthenticateRejectsOTPAndTransactionID(t *testing.T) {
	tm, _ := newManager(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	if _, err := tm.Authenticate("123456", "tx"); err == nil {
		t.Fatal("both otp and transaction id must be refused")
	}
	tm2 := NewTokenManager(transport.NewRestClient(ssi.NewConfig("c")), "", "")
	if _, err := tm2.Authenticate("1"); err == nil {
		t.Fatal("missing credentials must fail without a request")
	}
}

func TestEnsureAuthenticatedUsesValidTokenWithoutRequest(t *testing.T) {
	tm, _ := newManager(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	tm.SetToken(&Token{AccessToken: "ok", ExpiresAt: time.Now().Unix() + 3600})
	got, err := tm.EnsureAuthenticated("")
	if err != nil || got != "ok" {
		t.Fatalf("%q %v", got, err)
	}
}

func TestEnsureAuthenticatedFailsClosedWithoutOTP(t *testing.T) {
	tm, _ := newManager(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })
	if _, err := tm.EnsureAuthenticated(""); err == nil {
		t.Fatal("no token and no OTP must be an error")
	}
	tm.SetToken(&Token{AccessToken: "old", ExpiresAt: time.Now().Unix() - 10}) // expired, no refresh token
	_, err := tm.EnsureAuthenticated("")
	var re *ssi.ReauthenticationRequiredError
	if !errors.As(err, &re) {
		t.Fatalf("want ReauthenticationRequiredError, got %v", err)
	}
}

func TestConcurrentRefreshMakesOneServerCall(t *testing.T) {
	var refreshes int32
	tm, _ := newManager(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/auth/refresh" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		atomic.AddInt32(&refreshes, 1)
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write([]byte(tokenBody("fresh", 3600)))
	})
	now := time.Now().Unix()
	tm.SetToken(&Token{AccessToken: "old", ExpiresAt: now - 5, RefreshToken: "r-old", RefreshTokenExpiresAt: now + 3600})

	var wg sync.WaitGroup
	results := make([]string, 8)
	for i := range results {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tok, err := tm.EnsureAuthenticated("")
			if err != nil {
				t.Errorf("caller %d: %v", i, err)
			}
			results[i] = tok
		}(i)
	}
	wg.Wait()
	if n := atomic.LoadInt32(&refreshes); n != 1 {
		t.Fatalf("expected a single refresh call, got %d", n)
	}
	for i, r := range results {
		if r != "fresh" {
			t.Fatalf("caller %d got %q", i, r)
		}
	}
}

func TestRefreshRejectedClearsSession(t *testing.T) {
	tm, _ := newManager(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = w.Write([]byte(`{"code":401103,"msg":"refresh token reused"}`))
	})
	now := time.Now().Unix()
	tm.SetToken(&Token{AccessToken: "old", ExpiresAt: now - 5, RefreshToken: "r-old", RefreshTokenExpiresAt: now + 3600})
	_, err := tm.Refresh()
	var re *ssi.ReauthenticationRequiredError
	if !errors.As(err, &re) || re.Code != "401103" {
		t.Fatalf("want ReauthenticationRequiredError, got %v", err)
	}
	if tm.Token() != nil || tm.AccessToken() != "" {
		t.Fatal("the stored token must be cleared")
	}
	if _, err := tm.Refresh(); err == nil {
		t.Fatal("a cleared session must not call the server again")
	}
}

func TestParseTokenValidation(t *testing.T) {
	if _, err := parseToken(map[string]interface{}{"data": "nope"}, "x"); err == nil {
		t.Fatal("non-object data")
	}
	if _, err := parseToken(map[string]interface{}{"data": map[string]interface{}{}}, "x"); err == nil {
		t.Fatal("missing access token")
	}
	tk, err := parseToken(map[string]interface{}{"data": map[string]interface{}{"accessToken": "a", "expiresAt": float64(5)}}, "x")
	if err != nil || tk.AccessToken != "a" {
		t.Fatalf("%v %v", tk, err)
	}
}

func TestRefreshChecksRefreshTokenBeforeCredentials(t *testing.T) {
	cfg := ssi.NewConfig("c")
	cfg.LogLevel = "ERROR"
	tm := NewTokenManager(transport.NewRestClient(cfg), "", "")
	_, err := tm.Refresh()
	if err == nil || !strings.Contains(err.Error(), "No refresh token") {
		t.Fatalf("want the missing refresh token error first, got %v", err)
	}
	tm.SetToken(&Token{AccessToken: "a", RefreshToken: "r", RefreshTokenExpiresAt: time.Now().Unix() + 3600})
	_, err = tm.Refresh()
	if err == nil || !strings.Contains(err.Error(), "api_key and api_secret are required for token refresh") {
		t.Fatalf("want the credentials error, got %v", err)
	}
}

func TestPollAttemptsUsesMilliseconds(t *testing.T) {
	cfg := ssi.NewConfig("c")
	wait, n := pollAttempts(cfg, nil, nil)
	if wait != 5000 || n != 12 {
		t.Fatalf("defaults: %d ms x %d", wait, n)
	}
	iv, retries := 1000, 3
	if wait, n = pollAttempts(cfg, &iv, &retries); wait != 1000 || n != 3 {
		t.Fatalf("explicit: %d ms x %d", wait, n)
	}
	if wait, n = pollAttempts(cfg, &iv, nil); wait != 1000 || n != 60 {
		t.Fatalf("interval only: %d ms x %d", wait, n)
	}
}
