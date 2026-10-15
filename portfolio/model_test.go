package portfolio

import (
	"testing"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

func TestParsePrice(t *testing.T) {
	cases := []struct {
		in   interface{}
		f    float64
		text string
	}{
		{nil, 0, ""},
		{" ", 0, ""},
		{"12.5", 12.5, ""},
		{float64(3), 3, ""},
		{"ATC", 0, "ATC"},
	}
	for _, c := range cases {
		f, text := parsePrice(c.in)
		if f != c.f || text != c.text {
			t.Errorf("parsePrice(%v) = %v,%q want %v,%q", c.in, f, text, c.f, c.text)
		}
	}
}

func TestEquityPositionTBuyQuantity(t *testing.T) {
	rows := EquityPositionFromList([]interface{}{
		map[string]interface{}{"symbol": "FPT", "t1BuyQuantity": float64(100), "t2BuyQuantity": float64(200)},
		map[string]interface{}{"symbol": "VNM"},
		map[string]interface{}{"symbol": "HPG", "t1BuyQuantity": "300", "t2BuyQuantity": "400"},
	})
	if len(rows) != 3 {
		t.Fatalf("rows = %d", len(rows))
	}
	want := [][2]int{{100, 200}, {0, 0}, {300, 400}}
	for i, w := range want {
		if rows[i].T1BuyQuantity != w[0] || rows[i].T2BuyQuantity != w[1] {
			t.Errorf("%s: got %d/%d want %d/%d", rows[i].Symbol, rows[i].T1BuyQuantity, rows[i].T2BuyQuantity, w[0], w[1])
		}
	}
}

func TestOrderFromMapNormalizesCase(t *testing.T) {
	o := OrderFromMap(map[string]interface{}{
		"side": "b", "orderType": "lo", "orderStatus": "ff", "price": "ato",
	}, "A1")
	if o.Side != trading.OrderSideBuy || o.OrderType != trading.OrderTypeLO || o.Status != "FF" {
		t.Fatalf("not normalized: %+v", o)
	}
	if o.Price != 0 || o.PriceText != string(trading.OrderTypeATO) {
		t.Fatalf("price word: %v %q", o.Price, o.PriceText)
	}
	o = OrderFromMap(map[string]interface{}{"side": "X", "orderStatus": "NEW-THING", "price": "weird"}, "A1")
	if o.Side != "X" || o.Status != "NEW-THING" || o.PriceText != "weird" {
		t.Fatalf("unknown values must be kept as sent: %+v", o)
	}
}
