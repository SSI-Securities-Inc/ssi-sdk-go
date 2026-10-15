package stream

import (
	"errors"
	"testing"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
)

func TestBuildTopic(t *testing.T) {
	cases := []struct {
		typ, filter, interval, want string
	}{
		{"trade", "vnm", "", "trade.VNM"},
		{"trade", "vnm", "1m", "trade.VNM@1m"},
		{"trade", "vn30", "*", "trade.VN30@*"},
		{"quote", "HOSE", "*", "quote.HOSE@*"},
		{"room", "hnx", "*", "room.HNX@*"},
		{"trade", "upcomindex", "*", "trade.HNXUPCOMINDEX@*"},
		{"trade", "SSI", "TICK", "trade.SSI@tick"},
		{"trade", "*", "5m", "trade.*@5m"},
		{"quote", "A-B", "", "quote.A-B"},
		{"trade.index", "VN30", "1m", "trade.index.VN30@1m"},
		{"trade.index", "upcomindex", "", "trade.index.HNXUPCOMINDEX"},
		{"order", "abc123", "", "order.abc123"},
		{"portfolio", "AbC", "", "portfolio.AbC"},
		{"indexsummary", "*", "", "indexsummary.*"},
	}
	for _, c := range cases {
		got, err := BuildTopic(c.typ, c.filter, tf(c.interval))
		if err != nil || got != c.want {
			t.Errorf("BuildTopic(%q,%q,%q) = %q, %v; want %q", c.typ, c.filter, c.interval, got, err, c.want)
		}
	}
}

func TestBuildTopicRejects(t *testing.T) {
	bad := []struct{ typ, filter, interval string }{
		{"asset", "A", ""},
		{"trade", "", ""},
		{"trade", "VN M", ""},
		{"trade", "A,,B", ""},
		{"trade", "VNM;FPT", ""},
		{"quote", "VNM", "1m"},
		{"trade", "VNM", "15m"},
		{"trade.index", "*", "1m"},
		{"trade.index", "A,B", "1m"},
		{"order", "A\nB", ""},
		{"trade", "VNM,FPT", ""},
		{"trade", "VNM,FPT", "tick"},
		{"trade", "*", "*"},
		{"trade", "A-B", "*"},
		{"trade.index", "VN30", "*"},
		{"market", "HOSE", "*"},
		{"order", "ACC1", "*"},
	}
	for _, c := range bad {
		_, err := BuildTopic(c.typ, c.filter, tf(c.interval))
		var ve *ssi.ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("BuildTopic(%q,%q,%q) should be a ValidationError, got %v", c.typ, c.filter, c.interval, err)
		}
	}
}

func TestBuildTopics(t *testing.T) {
	got, err := BuildTopics("trade", []string{"vnm", "fpt"}, "")
	if err != nil || len(got) != 2 || got[0] != "trade.VNM" || got[1] != "trade.FPT" {
		t.Fatalf("got %v %v", got, err)
	}
	if _, err := BuildTopics("trade", nil, ""); err == nil {
		t.Fatal("empty list must fail")
	}
	if _, err := BuildTopics("trade", []string{"ok", "bad one"}, ""); err == nil {
		t.Fatal("one bad filter must fail the whole call")
	}
}

func TestGroupRequestsUseAllInterval(t *testing.T) {
	s := &Service{}
	for _, method := range []StreamingMethod{StreamingMethodSubscribe, StreamingMethodUnsubscribe} {
		reqs, err := s.groupRequests(method, boardNames([]market.Board{market.BoardHOSE, market.BoardHNX}))
		if err != nil || len(reqs) != 3 {
			t.Fatalf("method %v: got %d requests, %v", method, len(reqs), err)
		}
		want := [][]string{
			{"trade.HOSE@*", "trade.HNX@*"},
			{"quote.HOSE@*", "quote.HNX@*"},
			{"room.HOSE@*", "room.HNX@*"},
		}
		for i, req := range reqs {
			if req.Method != method || req.Channel != StreamingChannelData {
				t.Errorf("request %d: method=%v channel=%v", i, req.Method, req.Channel)
			}
			if len(req.Topics) != len(want[i]) || req.Topics[0] != want[i][0] || req.Topics[1] != want[i][1] {
				t.Errorf("request %d topics = %v; want %v", i, req.Topics, want[i])
			}
		}
	}

	reqs, err := s.groupRequests(StreamingMethodSubscribe, []string{"vn30"})
	if err != nil || reqs[0].Topics[0] != "trade.VN30@*" {
		t.Fatalf("index: %v %v", reqs, err)
	}
	if _, err := s.groupRequests(StreamingMethodSubscribe, []string{"VN30", "A,B"}); err == nil {
		t.Fatal("a comma list must fail the whole group before anything is sent")
	}
	if _, err := s.groupRequests(StreamingMethodSubscribe, nil); err == nil {
		t.Fatal("empty list must fail")
	}
}

func tf(s string) market.Timeframe { return market.Timeframe(s) }
