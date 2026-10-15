package ssisdk

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestConfigStringNeverPrintsSecrets(t *testing.T) {
	c := NewConfig("client")
	c.APIKey = "API-KEY-SECRET-VALUE"
	c.APISecret = "API-SECRET-SECRET-VALUE"
	c.PrivateKey = "PRIVATE-KEY-SECRET-VALUE"
	c.Proxy = "http://user:PROXY-SECRET-VALUE@proxy:8080"
	for _, s := range []string{c.String(), fmt.Sprintf("%v", c), fmt.Sprintf("%+v", *c), fmt.Sprintf("%#v", c), fmt.Sprint(*c)} {
		if strings.Contains(s, "SECRET-VALUE") {
			t.Fatalf("secret leaked: %s", s)
		}
	}
}

func TestNormalizedDoesNotMutate(t *testing.T) {
	c := &Config{}
	n := c.Normalized()
	if c.APIURL != "" || c.MaxBatchOrders != 0 {
		t.Fatal("the caller's Config must not change")
	}
	if n.APIURL != DefaultAPIURL || n.StreamingURL != DefaultStreamingURL || n.MaxBatchOrders != MaxBatchOrders ||
		n.OTPPollIntervalMs <= 0 || n.OTPPollMaxWaitMs <= 0 {
		t.Fatalf("defaults not applied: %v", n)
	}
}

func TestTradingAPIURL(t *testing.T) {
	cases := []struct{ domain, api, want string }{
		{"", "https://api.example", "https://api.example"},
		{DefaultTradingAPIDomain, "https://api.example/", "https://api.example"},
		{"trade.example:8443", "https://api.example", "https://trade.example:8443"},
		{"http://127.0.0.1:9/some/path", "https://api.example", "http://127.0.0.1:9"},
		{"HTTPS://Trade.Example/", "https://api.example", "https://Trade.Example"},
	}
	for _, c := range cases {
		cfg := &Config{TradingAPIDomain: c.domain, APIURL: c.api}
		got, err := cfg.TradingAPIURL()
		if err != nil || got != c.want {
			t.Errorf("(%q,%q) = %q, %v; want %q", c.domain, c.api, got, err, c.want)
		}
	}
	_, err := (&Config{TradingAPIDomain: "ftp://x", APIURL: "https://a"}).TradingAPIURL()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("a non-http scheme must be a ValidationError, got %v", err)
	}
}

func TestTradingWSURL(t *testing.T) {
	cases := map[string]string{
		"":                           DefaultTradingWSDomain,
		"host.example":               "wss://host.example" + TradingWSPath,
		"host.example:8443/":         "wss://host.example:8443" + TradingWSPath,
		"wss://host.example/custom/": "wss://host.example/custom",
	}
	for domain, want := range cases {
		got := (&Config{TradingWSDomain: domain}).TradingWSURL()
		if domain == "" {
			if !strings.HasPrefix(got, "wss://") {
				t.Errorf("default must be wss://: %q", got)
			}
			continue
		}
		if got != want {
			t.Errorf("TradingWSURL(%q) = %q, want %q", domain, got, want)
		}
	}
}

func TestRequireHelpers(t *testing.T) {
	if RequireNonEmpty("  ", "x") == nil || RequireNonEmpty("a", "x") != nil {
		t.Fatal("RequireNonEmpty")
	}
	if RequirePositive(0, "x") == nil || RequirePositive(-1, "x") == nil || RequirePositive(0.1, "x") != nil {
		t.Fatal("RequirePositive")
	}
	nan := 0.0
	nan = nan / nan
	if RequirePositive(nan, "x") == nil || RequireNonNegative(nan, "x") == nil {
		t.Fatal("NaN must never pass a range check")
	}
	if RequireNonNegative(0, "x") != nil || RequireNonNegative(-0.01, "x") == nil {
		t.Fatal("RequireNonNegative")
	}
	if RequireSymbol("VNM", "s") != nil || RequireSymbol("VNM,FPT", "s") == nil || RequireSymbol("V N", "s") == nil {
		t.Fatal("RequireSymbol")
	}
	if RequireExactlyOne("a", "", "a", "b") != nil || RequireExactlyOne("", "", "a", "b") == nil || RequireExactlyOne("a", "b", "a", "b") == nil {
		t.Fatal("RequireExactlyOne")
	}
	if RequireDateRange("2026/01/01 00:00:00", "2026/03/01 00:00:00", false, true, 31) == nil {
		t.Fatal("span cap")
	}
	if RequireDateRange("2026/01/01 00:00:00", "2026/01/31 00:00:00", false, true, 31) != nil {
		t.Fatal("30 days is within the cap")
	}
}

func TestErrorsUnwrapToParents(t *testing.T) {
	var err error = NewDuplicateRequestError("dup", "409", nil)
	var api *APIError
	if !errors.As(err, &api) {
		t.Fatal("DuplicateRequestError must be an APIError")
	}
	err = NewSmartOTPPendingError("p", "401114", 202, nil)
	var au *AuthenticationError
	if !errors.As(err, &au) {
		t.Fatal("SmartOTPPendingError must be an AuthenticationError")
	}
	err = NewTradingWSError(409, "DUP", "dup")
	var ws *WebSocketError
	if !errors.As(err, &ws) {
		t.Fatal("TradingWSError must be a WebSocketError")
	}
}
