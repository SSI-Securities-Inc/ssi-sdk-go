package redact

import (
	"strings"
	"testing"
)

func TestMask(t *testing.T) {
	cases := map[string]string{
		"":                "***",
		"short":           "***",
		"12345678":        "***",
		"123456789":       "***6789",
		"abcdefghijklmno": "***lmno",
	}
	for in, want := range cases {
		if got := Mask(in); got != want {
			t.Errorf("Mask(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMaskBearer(t *testing.T) {
	got := MaskBearer("Authorization failed for Bearer abcdefghijklmnop.qrst now")
	if strings.Contains(got, "abcdefghijkl") || !strings.Contains(got, "Bearer ***qrst") {
		t.Fatalf("token not masked: %q", got)
	}
	if MaskBearer("no secret here") != "no secret here" {
		t.Fatal("text without a token must be unchanged")
	}
}

func TestRedactDeepAndNonMutating(t *testing.T) {
	in := map[string]interface{}{
		"AccessToken": "very-long-access-token-value",
		"symbol":      "VNM",
		"nested": map[string]interface{}{
			"otp":       "123456",
			"accountNo": "0001234567",
			"items":     []interface{}{map[string]interface{}{"signature": "sigsigsigsigsig"}, "Bearer tokentokentoken1"},
		},
	}
	out := Redact(in).(map[string]interface{})
	if out["symbol"] != "VNM" {
		t.Fatal("non-sensitive value changed")
	}
	if out["AccessToken"] != "***alue" {
		t.Fatalf("case-insensitive key match failed: %v", out["AccessToken"])
	}
	nested := out["nested"].(map[string]interface{})
	if nested["otp"] != "***" || nested["accountNo"] != "***4567" {
		t.Fatalf("nested not masked: %v", nested)
	}
	items := nested["items"].([]interface{})
	if items[0].(map[string]interface{})["signature"] != "***sig" && !strings.HasPrefix(items[0].(map[string]interface{})["signature"].(string), "***") {
		t.Fatalf("slice item not masked: %v", items[0])
	}
	if strings.Contains(items[1].(string), "tokentokentoken1") {
		t.Fatalf("bearer inside a string not masked: %v", items[1])
	}
	if in["AccessToken"] != "very-long-access-token-value" || in["nested"].(map[string]interface{})["otp"] != "123456" {
		t.Fatal("input was mutated")
	}
}

func TestRedactMapNil(t *testing.T) {
	if RedactMap(nil) != nil {
		t.Fatal("nil in, nil out")
	}
}

func TestSafeURL(t *testing.T) {
	got := SafeURL("wss://user:pw@host.example:8443/ws/v3?token=SECRET#frag")
	if got != "wss://host.example:8443/ws/v3" {
		t.Fatalf("got %q", got)
	}
}
