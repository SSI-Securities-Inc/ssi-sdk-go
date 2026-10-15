package market

import (
	"errors"
	"testing"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

func isValidation(err error) bool {
	var ve *ssi.ValidationError
	return errors.As(err, &ve)
}

var fixedNow = time.Date(2026, 3, 10, 10, 30, 0, 0, time.Local)

func TestOHLCParamsDefaultsToToday(t *testing.T) {
	p, err := buildOHLCParams("VNM", TimeframeMinute1, "", "", 1, 100, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if p["from"] != "2026/03/10 00:00:00" || p["to"] != "2026/03/10 23:59:59" || p["timeFrame"] != "1m" || p["pageSize"] != "100" {
		t.Fatalf("%v", p)
	}
}

func TestOHLCParamsCompletesBareDates(t *testing.T) {
	p, err := buildOHLCParams("VNM", TimeframeDay1, "2026/01/01", "2026/01/31", 1, 10, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if p["from"] != "2026/01/01 00:00:00" || p["to"] != "2026/01/31 23:59:59" {
		t.Fatalf("a bare date makes the server answer 400213: %v", p)
	}
}

func TestOHLCParamsClampsFutureEndForIntraday(t *testing.T) {
	p, err := buildOHLCParams("VNM", TimeframeMinute5, "2026/03/09", "2026/12/31", 1, 10, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	if p["to"] != "2026/03/10 23:59:59" {
		t.Fatalf("future end must be cut to today for minute bars: %v", p["to"])
	}
	p, _ = buildOHLCParams("VNM", TimeframeDay1, "2026/03/09", "2026/12/31", 1, 10, fixedNow)
	if p["to"] != "2026/12/31 23:59:59" {
		t.Fatalf("daily bars are not clamped: %v", p["to"])
	}
	p, _ = buildOHLCParams("VNM", TimeframeMinute5, "2026/11/01", "2026/12/31", 1, 10, fixedNow)
	if p["to"] != "2026/12/31 23:59:59" {
		t.Fatalf("a range that starts in the future is left alone: %v", p["to"])
	}
}

func TestOHLCParamsValidation(t *testing.T) {
	bad := []struct {
		name       string
		symbol     string
		tf         Timeframe
		from, to   string
		page, size int
	}{
		{"week", "VNM", TimeframeWeek1, "", "", 1, 10},
		{"month", "VNM", TimeframeMonth1, "", "", 1, 10},
		{"unknown tf", "VNM", Timeframe("2h"), "", "", 1, 10},
		{"symbol list", "VNM,FPT", TimeframeDay1, "", "", 1, 10},
		{"empty symbol", "", TimeframeDay1, "", "", 1, 10},
		{"page zero", "VNM", TimeframeDay1, "", "", 0, 10},
		{"size zero", "VNM", TimeframeDay1, "", "", 1, 0},
		{"reversed", "VNM", TimeframeDay1, "2026/02/01", "2026/01/01", 1, 10},
		{"bad date", "VNM", TimeframeDay1, "01-02-2026", "2026/01/01", 1, 10},
	}
	for _, c := range bad {
		if _, err := buildOHLCParams(c.symbol, c.tf, c.from, c.to, c.page, c.size, fixedNow); !isValidation(err) {
			t.Errorf("%s: want ValidationError, got %v", c.name, err)
		}
	}
}

func TestRestBoard(t *testing.T) {
	b, err := restBoard(Board("hose"))
	if err != nil || *b != "HOSE" {
		t.Fatalf("%v %v", b, err)
	}
	if _, err := restBoard(Board("BOGUS")); !isValidation(err) {
		t.Fatal("unknown board")
	}
}

func TestRowsOf(t *testing.T) {
	if rows, err := rowsOf(map[string]interface{}{"code": float64(204)}, "x"); rows != nil || err != nil {
		t.Fatalf("a {code:204} marker is no rows, not an error: %v %v", rows, err)
	}
	if rows, err := rowsOf(nil, "x"); rows != nil || err != nil {
		t.Fatal("nil is no rows")
	}
	if _, err := rowsOf(map[string]interface{}{"data": "oops"}, "x"); err == nil {
		t.Fatal("a non-list payload is an error")
	}
	rows, err := rowsOf(map[string]interface{}{"data": []interface{}{map[string]interface{}{"a": 1}}}, "x")
	if err != nil || len(rows) != 1 {
		t.Fatalf("%v %v", rows, err)
	}
}

func TestPagesCount(t *testing.T) {
	if pagesCount(map[string]interface{}{}) != 1 {
		t.Fatal("default is one page")
	}
	if pagesCount(map[string]interface{}{"pageCount": float64(4)}) != 4 {
		t.Fatal("pageCount fallback")
	}
	if pagesCount(map[string]interface{}{"pagesCount": float64(7), "pageCount": float64(4)}) != 7 {
		t.Fatal("pagesCount wins")
	}
}
