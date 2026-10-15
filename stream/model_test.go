package stream

import "testing"

func TestParsePrice(t *testing.T) {
	cases := []struct {
		in   interface{}
		f    float64
		text string
	}{
		{nil, 0, ""},
		{"", 0, ""},
		{"  ", 0, ""},
		{"25.5", 25.5, ""},
		{float64(26), 26, ""},
		{"ATO", 0, "ATO"},
		{" MP ", 0, "MP"},
	}
	for _, c := range cases {
		f, text := parsePrice(c.in)
		if f != c.f || text != c.text {
			t.Errorf("parsePrice(%v) = %v,%q want %v,%q", c.in, f, text, c.f, c.text)
		}
	}
}

func TestLevelsSkipsMalformed(t *testing.T) {
	p, v := levels([]interface{}{
		[]interface{}{10.5, float64(100)},
		[]interface{}{"x"},
		"bad",
		[]interface{}{11.0, float64(5)},
	})
	if len(p) != 2 || p[0] != 10.5 || v[1] != 5 {
		t.Fatalf("%v %v", p, v)
	}
	p, v = levels(nil)
	if p == nil || v == nil || len(p) != 0 {
		t.Fatal("non-list input gives empty, non-nil slices")
	}
}

func TestParseDataMessageTrade(t *testing.T) {
	got := ParseDataMessage(map[string]interface{}{
		"channel": "DATA", "topic": "trade.VNM",
		"data": map[string]interface{}{"s": "VNM", "p": 25.5, "q": float64(100), "v": float64(1000), "o": 25.0},
	})
	tm, ok := got.(*TradeMessage)
	if !ok {
		t.Fatalf("got %T", got)
	}
	if tm.Symbol != "VNM" || tm.Price != 25.5 || tm.Quantity != 100 || tm.Side != "U" {
		t.Fatalf("%+v", tm)
	}
	if tm.Open == nil || *tm.Open != 25 || tm.High != nil {
		t.Fatalf("omitted optional values must be nil: %+v", tm)
	}
}

func TestParseDataMessageRoutingIsCaseInsensitive(t *testing.T) {
	data := map[string]interface{}{"data": map[string]interface{}{"s": "VN30"}}
	mk := func(topic string) map[string]interface{} {
		return map[string]interface{}{"topic": topic, "data": data["data"]}
	}
	if _, ok := ParseDataMessage(mk("TRADE.INDEX.VN30")).(*IndexTickMessage); !ok {
		t.Fatal("trade.index tick")
	}
	if _, ok := ParseDataMessage(mk("trade.index.VN30@1m")).(*IntervalMessage); !ok {
		t.Fatal("trade.index interval")
	}
	if _, ok := ParseDataMessage(mk("trade.VNM@5m")).(*IntervalMessage); !ok {
		t.Fatal("trade interval")
	}
	if _, ok := ParseDataMessage(mk("quote.VNM")).(*QuoteMessage); !ok {
		t.Fatal("quote")
	}
}

func TestParseDataMessageUnknownIsRaw(t *testing.T) {
	ack := map[string]interface{}{"method": "subscribe", "status": "ok"}
	got, ok := ParseDataMessage(ack).(map[string]interface{})
	if !ok || got["method"] != "subscribe" {
		t.Fatalf("a frame without a known topic is returned raw: %T", got)
	}
}

func TestParseTradingMessageOrder(t *testing.T) {
	got := ParseTradingMessage(map[string]interface{}{
		"topic": "order.A1",
		"data": map[string]interface{}{
			"eventType": "orderEvent", "orderId": "o1", "clientRequestId": "c1", "symbol": "VNM",
			"side": "B", "orderType": "ATO", "price": "ATO", "avgPrice": "25.5", "quantity": float64(100),
			"orderStatus": "PD", "osQty": float64(100),
		},
	})
	o, ok := got.(*OrderStatusMessage)
	if !ok {
		t.Fatalf("got %T", got)
	}
	if o.Price != 0 || o.PriceText != "ATO" || o.AvgPrice != 25.5 || o.AvgPriceText != "" {
		t.Fatalf("price handling: %+v", o)
	}
	if string(o.Status) != "PD" || o.Quantity != 100 {
		t.Fatalf("%+v", o)
	}
}

func TestParseTradingMessageUnknownStatusKeptRaw(t *testing.T) {
	got := ParseTradingMessage(map[string]interface{}{
		"topic": "order.A1",
		"data":  map[string]interface{}{"eventType": "orderEvent", "orderStatus": "NEWSTATUS", "side": "Z"},
	}).(*OrderStatusMessage)
	if string(got.Status) != "NEWSTATUS" || string(got.Side) != "Z" {
		t.Fatalf("unknown values must survive as raw strings: %+v", got)
	}
	empty := ParseTradingMessage(map[string]interface{}{"topic": "order.A1", "data": map[string]interface{}{}}).(*OrderStatusMessage)
	if empty.Status != "" || empty.Side != "" {
		t.Fatalf("absent values are empty: %+v", empty)
	}
}

func TestParseTradingMessageMatchPortfolioAndFallbacks(t *testing.T) {
	m := ParseTradingMessage(map[string]interface{}{
		"topic": "order.A1",
		"data":  map[string]interface{}{"eventType": "orderMatchEvent", "matchPrice": "25.5", "matchQty": float64(10)},
	})
	om, ok := m.(*OrderMatchMessage)
	if !ok || om.MatchPrice != 25.5 || om.MatchQty != 10 {
		t.Fatalf("%T %+v", m, m)
	}

	p := ParseTradingMessage(map[string]interface{}{
		"topic": "portfolio.A1",
		"data":  map[string]interface{}{"eventType": "clientPortfolioEvent", "symbol": "VN30F", "net": float64(-2), "marketPrice": 1300.5},
	})
	pm, ok := p.(*PortfolioMessage)
	if !ok || pm.Net != -2 || pm.MarketPrice == nil || *pm.MarketPrice != 1300.5 || pm.TradePrice != nil || pm.TotalAsset != nil {
		t.Fatalf("%T %+v", p, p)
	}

	if _, ok := ParseTradingMessage(map[string]interface{}{"topic": "portfolio.A1", "data": map[string]interface{}{"symbol": "X"}}).(*PortfolioMessage); !ok {
		t.Fatal("no eventType: the topic decides")
	}
	if _, ok := ParseTradingMessage(map[string]interface{}{"topic": "order.A1", "data": map[string]interface{}{"x": 1}}).(*OrderStatusMessage); !ok {
		t.Fatal("no eventType: the topic decides")
	}
	unknown := map[string]interface{}{"topic": "order.A1", "data": map[string]interface{}{"eventType": "somethingNew"}}
	if raw, ok := ParseTradingMessage(unknown).(map[string]interface{}); !ok || raw["topic"] != "order.A1" {
		t.Fatal("an unknown eventType is passed on as the raw frame")
	}
	if _, ok := ParseTradingMessage(map[string]interface{}{"data": "not-an-object"}).(map[string]interface{}); !ok {
		t.Fatal("a malformed payload must not panic")
	}
}

func TestParseTradingMessageFCO(t *testing.T) {
	got := ParseTradingMessage(map[string]interface{}{
		"topic": "order.A1",
		"data":  map[string]interface{}{"eventType": "fcoEvent", "fcoId": "f1", "processStatus": "ACTIVE", "symbol": "VNM"},
	})
	f, ok := got.(*FCOOrderUpdateMessage)
	if !ok || f.FCOID != "f1" || f.Symbol != "VNM" {
		t.Fatalf("%T %+v", got, got)
	}
}
