package util

import (
	"math"
	"testing"
)

func TestToFloat64(t *testing.T) {
	cases := []struct {
		in   interface{}
		want float64
	}{
		{"2,493,089.5", 2493089.5},
		{" 12 ", 12},
		{"", 0},
		{nil, 0},
		{true, 0},
		{"abc", 0},
		{math.Inf(1), 0},
		{math.NaN(), 0},
		{"NaN", 0},
		{int64(7), 7},
		{float64(1.5), 1.5},
	}
	for _, c := range cases {
		if got := ToFloat64(c.in); got != c.want {
			t.Errorf("ToFloat64(%#v)=%v want %v", c.in, got, c.want)
		}
	}
}

func TestToIntTruncatesTowardZero(t *testing.T) {
	cases := map[string]int{"1234.0": 1234, "2,493,089": 2493089, "-3": -3, "-3.9": -3, "x": 0}
	for in, want := range cases {
		if got := ToInt(in); got != want {
			t.Errorf("ToInt(%q)=%d want %d", in, got, want)
		}
	}
}

func TestOptionalNumbersDistinguishMissingFromZero(t *testing.T) {
	if ToOptInt(nil) != nil || ToOptInt("") != nil || ToOptInt("x") != nil {
		t.Fatal("missing/invalid must be nil")
	}
	if p := ToOptInt("0"); p == nil || *p != 0 {
		t.Fatal("0 must stay 0, not nil")
	}
	if p := ToOptFloat64("1,5"); p == nil || *p != 15 {
		t.Fatalf("got %v", p)
	}
}

func TestFormatPriceNeverUsesExponent(t *testing.T) {
	cases := map[float64]string{100: "100", 0.1: "0.1", 1e-7: "0.0000001", 12345678901234: "12345678901234", 25.5: "25.5"}
	for in, want := range cases {
		got, err := FormatPrice(in)
		if err != nil || got != want {
			t.Errorf("FormatPrice(%v)=%q,%v want %q", in, got, err, want)
		}
	}
	if _, err := FormatPrice(math.NaN()); err == nil {
		t.Error("NaN must be rejected")
	}
	if _, err := FormatPrice(math.Inf(-1)); err == nil {
		t.Error("Inf must be rejected")
	}
}

func TestIsNoContentAndPick(t *testing.T) {
	if !IsNoContent(map[string]interface{}{"code": float64(204), "msg": "No Content"}) {
		t.Fatal("204 marker")
	}
	if !IsNoContent(map[string]interface{}{"code": "204"}) {
		t.Fatal("string 204 marker")
	}
	if IsNoContent(map[string]interface{}{"code": float64(200)}) || IsNoContent("x") || IsNoContent(nil) {
		t.Fatal("not a marker")
	}
	m := map[string]interface{}{"a": nil, "b": 0, "c": 3}
	if Pick(m, "a", "b", "c") != 0 {
		t.Fatal("Pick skips nil but keeps zero")
	}
	if Pick(m, "x") != nil {
		t.Fatal("missing -> nil")
	}
}

func TestParseDate(t *testing.T) {
	for _, s := range []string{"2025/01/02", "2025-01-02", "2025/01/02 10:11:12", "01/02/2025", "25/12/2025"} {
		if _, ok := ParseDate(s); !ok {
			t.Errorf("%q should parse", s)
		}
	}
	d, _ := ParseDate("01/02/2025")
	if d.Month() != 1 || d.Day() != 2 {
		t.Errorf("ambiguous date must read MM/DD/YYYY, got %v", d)
	}
	for _, s := range []string{"", "  ", "garbage"} {
		if _, ok := ParseDate(s); ok {
			t.Errorf("%q should not parse", s)
		}
	}
}

func TestGenerateRequestID(t *testing.T) {
	a, b := GenerateRequestID(), GenerateRequestID()
	if len(a) != 20 || a == b {
		t.Fatalf("bad ids %q %q", a, b)
	}
}
