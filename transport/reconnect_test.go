package transport

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

func TestBackoffDelayMs(t *testing.T) {
	if got := BackoffDelayMs(0, 1000, 30000, 1); got != 1000 {
		t.Fatalf("attempt 0, full jitter: %v", got)
	}
	if got := BackoffDelayMs(0, 1000, 30000, 0); got != 500 {
		t.Fatalf("attempt 0, zero jitter is half: %v", got)
	}
	if got := BackoffDelayMs(3, 1000, 30000, 1); got != 8000 {
		t.Fatalf("exponential: %v", got)
	}
	if got := BackoffDelayMs(50, 1000, 30000, 1); got != 30000 {
		t.Fatalf("capped: %v", got)
	}
	if got := BackoffDelayMs(0, 0, 30000, 0); got < 50 {
		t.Fatalf("floor: %v", got)
	}
	if got := BackoffDelayMs(-5, 1000, 0, 2); got != 1000 {
		t.Fatalf("negative attempt / out-of-range jitter are clamped: %v", got)
	}
}

func TestClassifyRejection(t *testing.T) {
	cases := []struct {
		in   map[string]interface{}
		kind RejectionKind
		wait *int
	}{
		{map[string]interface{}{"code": float64(401), "msg": "x"}, RejectionAuth, nil},
		{map[string]interface{}{"code": "403"}, RejectionAuth, nil},
		{map[string]interface{}{"code": float64(429), "retryAfter": float64(7)}, RejectionRateLimited, ptr(7000)},
		{map[string]interface{}{"code": float64(429), "retry_after": "3"}, RejectionRateLimited, ptr(3000)},
		{map[string]interface{}{"code": float64(503)}, RejectionTransient, nil},
		{map[string]interface{}{"code": float64(429), "retryAfter": float64(-1)}, RejectionRateLimited, nil},
		{map[string]interface{}{}, RejectionTransient, nil},
	}
	for i, c := range cases {
		kind, wait := ClassifyRejection(c.in)
		if kind != c.kind || !reflect.DeepEqual(wait, c.wait) {
			t.Errorf("case %d: got (%v, %v), want (%v, %v)", i, kind, wait, c.kind, c.wait)
		}
	}
}

func ptr(n int) *int { return &n }

func TestRateLimitWaitMs(t *testing.T) {
	if RateLimitWaitMs(nil) != ssi.WSRateLimitBackoffMs {
		t.Fatal("default floor")
	}
	if RateLimitWaitMs(ptr(1000)) != ssi.WSRateLimitBackoffMs {
		t.Fatal("a shorter Retry-After must not go under the floor")
	}
	if got := RateLimitWaitMs(ptr(ssi.WSRateLimitBackoffMs + 100000)); got != ssi.WSRateLimitBackoffMs+100000 {
		t.Fatalf("a longer Retry-After is honoured: %v", got)
	}
}

func TestMsUntilRefresh(t *testing.T) {
	if _, ok := MsUntilRefresh(0, 1_000_000, 60000); ok {
		t.Fatal("unknown expiry must never refresh proactively")
	}
	if ms, ok := MsUntilRefresh(2_000_000, 1_000_000, 60000); !ok || ms != 940_000 {
		t.Fatalf("got %v %v", ms, ok)
	}
	if ms, ok := MsUntilRefresh(1_030_000, 1_000_000, 60000); !ok || ms != 0 {
		t.Fatalf("already inside the lead window: %v %v", ms, ok)
	}
	if ms, _ := MsUntilRefresh(1_000_000_000_000, 0, 0); ms != 1_000_000_000_000-int64(ssi.WSTokenRefreshLeadMs) {
		t.Fatalf("default lead: %v", ms)
	}
}

func TestSubscriptionBook(t *testing.T) {
	b := NewSubscriptionBook()
	b.Record(map[string]interface{}{"method": "SUBSCRIBE", "channel": "DATA", "topics": []string{"trade.VNM", "trade.FPT"}})
	b.Record(map[string]interface{}{"method": "subscribe", "channel": "DATA", "topics": []interface{}{"trade.VNM", "quote.VNM"}})
	b.Record(map[string]interface{}{"method": "subscribe", "channel": "TRADING", "topics": []string{"order.A1"}})
	b.Record(map[string]interface{}{"method": "ping"})
	b.Record(map[string]interface{}{"method": "subscribe", "channel": "DATA", "topics": "bad"})

	want := []map[string]interface{}{
		{"method": "subscribe", "channel": "DATA", "topics": []string{"trade.VNM", "trade.FPT", "quote.VNM"}},
		{"method": "subscribe", "channel": "TRADING", "topics": []string{"order.A1"}},
	}
	if got := b.Requests(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v", got)
	}

	b.Record(map[string]interface{}{"method": "unsubscribe", "channel": "DATA", "topics": []string{"trade.VNM"}})
	if got := b.Requests()[0]["topics"]; !reflect.DeepEqual(got, []string{"trade.FPT", "quote.VNM"}) {
		t.Fatalf("after partial unsubscribe: %v", got)
	}
	b.Record(map[string]interface{}{"method": "unsubscribe", "channel": "DATA", "topics": []string{"trade.FPT", "quote.VNM"}})
	if got := b.Requests(); len(got) != 1 || got[0]["channel"] != "TRADING" {
		t.Fatalf("an emptied channel must disappear: %v", got)
	}
	b.Clear()
	if len(b.Requests()) != 0 {
		t.Fatal("Clear must forget everything")
	}
}

func TestSubscriptionBookRequestsAreCopies(t *testing.T) {
	b := NewSubscriptionBook()
	b.Record(map[string]interface{}{"method": "subscribe", "channel": "DATA", "topics": []string{"a"}})
	r := b.Requests()
	r[0]["topics"].([]string)[0] = "mutated"
	if b.Requests()[0]["topics"].([]string)[0] != "a" {
		t.Fatal("callers must not be able to edit the book")
	}
}

func TestSplitRequestKeepsEveryFrameUnderLimit(t *testing.T) {
	var topics []string
	for i := 0; i < 400; i++ {
		topics = append(topics, "trade.SYM"+strings.Repeat("X", 3)+string(rune('A'+i%26)))
	}
	req := map[string]interface{}{"method": "subscribe", "channel": "DATA", "topics": topics}
	parts := SplitRequest(req, 0)
	if len(parts) < 2 {
		t.Fatalf("expected a split, got %d part(s)", len(parts))
	}
	total := 0
	for _, p := range parts {
		b, _ := json.Marshal(p)
		if len(b) > ssi.WSMaxFrameBytes {
			t.Fatalf("part is %d bytes", len(b))
		}
		if p["method"] != "subscribe" || p["channel"] != "DATA" {
			t.Fatal("method/channel must be preserved")
		}
		total += len(p["topics"].([]string))
	}
	if total != len(topics) {
		t.Fatalf("topics lost: %d != %d", total, len(topics))
	}
}

func TestSplitRequestUnchanged(t *testing.T) {
	small := map[string]interface{}{"method": "subscribe", "topics": []string{"a"}}
	if got := SplitRequest(small, 0); len(got) != 1 {
		t.Fatal("small request must not split")
	}
	noTopics := map[string]interface{}{"method": "ping"}
	if got := SplitRequest(noTopics, 1); len(got) != 1 {
		t.Fatal("a request without topics is returned as is")
	}
	if got := SplitRequest(map[string]interface{}{"topics": []string{"aaaa", "bbbb"}}, 10); len(got) != 2 {
		t.Fatalf("a single topic bigger than the limit still goes out alone: %v", got)
	}
}

func TestSplitSubscriptions(t *testing.T) {
	got := SplitSubscriptions(map[string]interface{}{"trading": "order.A;portfolio.A;", "data": ""})
	if !reflect.DeepEqual(got["trading"], []string{"order.A", "portfolio.A"}) || len(got["data"]) != 0 {
		t.Fatalf("got %v", got)
	}
	if got := SplitSubscriptions(map[string]interface{}{}); got["trading"] == nil || got["data"] == nil {
		t.Fatal("missing keys must give empty (non-nil) lists")
	}
}

func TestDecodeMessage(t *testing.T) {
	if DecodeMessage([]byte(`[1,2]`)) != nil || DecodeMessage([]byte(`nope`)) != nil || DecodeMessage([]byte(`null`)) != nil {
		t.Fatal("non-object frames must decode to nil")
	}
	if DecodeMessage([]byte(`{"a":1}`))["a"] != float64(1) {
		t.Fatal("object frame")
	}
}

func TestRoutesOf(t *testing.T) {
	cases := []struct {
		name string
		in   map[string]interface{}
		want []string
	}{
		{"data", map[string]interface{}{"channel": "DATA", "topic": "x"}, []string{"DATA"}},
		{"ack on channel", map[string]interface{}{"channel": "DATA", "method": "subscribe", "status": "ok"}, []string{"DATA", ssi.WSRouteAck}},
		{"id", map[string]interface{}{"id": "1", "status": 200}, []string{ssi.WSRouteID}},
		{"rejection", map[string]interface{}{"code": 403, "msg": "x"}, []string{ssi.WSRouteError}},
		{"list", map[string]interface{}{"trading": "a;b", "data": ""}, []string{ssi.WSRouteListSubscription}},
		{"bare ack", map[string]interface{}{"method": "subscribe", "status": "ok"}, []string{ssi.WSRouteAck}},
		{"unknown", map[string]interface{}{"foo": 1}, nil},
	}
	for _, c := range cases {
		if got := RoutesOf(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestAckErrorCategoryNeverEchoesServerText(t *testing.T) {
	m := map[string]interface{}{"method": "subscribe", "status": "error", "message": "Denied (account): order.0001234567"}
	got := AckErrorCategory(m)
	if got != "Denied (account)" {
		t.Fatalf("got %q", got)
	}
	if AckErrorCategory(map[string]interface{}{"method": "subscribe", "status": "error", "message": "boom"}) != "error" {
		t.Fatal("unknown text maps to plain error")
	}
	if AckErrorCategory(map[string]interface{}{"method": "subscribe", "status": "ok"}) != "" {
		t.Fatal("successful ack has no category")
	}
}
