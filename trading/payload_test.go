package trading

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

func testBuilder() *PayloadBuilder {
	cfg := ssi.NewConfig("c")
	cfg.MaxBatchOrders = 3
	cfg.UserAgent = "test-agent"
	return NewPayloadBuilder(cfg)
}

func decode(t *testing.T, text string) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(text), &m); err != nil {
		t.Fatalf("not JSON: %v: %s", err, text)
	}
	return m
}

func isValidation(err error) bool {
	var ve *ssi.ValidationError
	return errors.As(err, &ve)
}

func TestPlaceOrderPayload(t *testing.T) {
	text, err := testBuilder().PlaceOrder("A1", "VNM", OrderSideBuy, 100, 25.5, OrderTypeLO, "my-key-1")
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, text)
	if m["price"] != "25.5" || m["clientRequestId"] != "my-key-1" || m["orderType"] != "LO" || m["side"] != "B" {
		t.Fatalf("payload: %v", m)
	}
	if m["deviceId"] == "" || m["deviceId"] == nil || m["userAgent"] != "test-agent" {
		t.Fatalf("identity not filled: %v", m)
	}
}

func TestPlaceOrderGeneratesIdempotencyKey(t *testing.T) {
	b := testBuilder()
	a, _ := b.PlaceOrder("A1", "VNM", OrderSideBuy, 100, 25.5, OrderTypeLO, "")
	c, _ := b.PlaceOrder("A1", "VNM", OrderSideBuy, 100, 25.5, OrderTypeLO, "")
	ida, idc := decode(t, a)["clientRequestId"].(string), decode(t, c)["clientRequestId"].(string)
	if ida == "" || ida == idc {
		t.Fatalf("each order needs its own non-empty key: %q %q", ida, idc)
	}
	if len(ida) > ssi.MaxClientRequestIDLength {
		t.Fatalf("generated key too long: %q", ida)
	}
}

func TestPlaceOrderPriceIsPlainDecimal(t *testing.T) {
	for _, p := range []float64{0.00001, 1234567890.5, 25.1} {
		text, err := testBuilder().PlaceOrder("A1", "VNM", OrderSideSell, 1, p, OrderTypeLO, "k")
		if err != nil {
			t.Fatal(err)
		}
		price := decode(t, text)["price"].(string)
		if strings.ContainsAny(price, "eE") {
			t.Fatalf("exponent notation for %v: %s", p, price)
		}
	}
}

func TestPlaceOrderMarketTypesSendZeroPrice(t *testing.T) {
	text, err := testBuilder().PlaceOrder("A1", "VNM", OrderSideBuy, 100, 0, OrderTypeATO, "k")
	if err != nil || decode(t, text)["price"] != "0" {
		t.Fatalf("%v %s", err, text)
	}
}

func TestPlaceOrderValidation(t *testing.T) {
	b := testBuilder()
	bad := []struct {
		name  string
		price float64
		qty   int
		ot    OrderType
		acc   string
		key   string
	}{
		{"LO without price", 0, 100, OrderTypeLO, "A1", "k"},
		{"PLO negative price", -1, 100, OrderTypePLO, "A1", "k"},
		{"negative price on ATO", -1, 100, OrderTypeATO, "A1", "k"},
		{"zero qty", 10, 0, OrderTypeLO, "A1", "k"},
		{"no account", 10, 1, OrderTypeLO, " ", "k"},
		{"key too long", 10, 1, OrderTypeLO, "A1", strings.Repeat("x", ssi.MaxClientRequestIDLength+1)},
	}
	for _, c := range bad {
		_, err := b.PlaceOrder(c.acc, "VNM", OrderSideBuy, c.qty, c.price, c.ot, c.key)
		if !isValidation(err) {
			t.Errorf("%s: want ValidationError, got %v", c.name, err)
		}
	}
}

func TestModifyAndCancelValidation(t *testing.T) {
	b := testBuilder()
	p, q := 10.0, 5
	if _, err := b.ModifyOrder("A1", "o1", "", &p, &q); !isValidation(err) {
		t.Fatal("price and quantity together must be refused")
	}
	if _, err := b.ModifyOrder("A1", "o1", "", nil, nil); !isValidation(err) {
		t.Fatal("nothing to modify must be refused")
	}
	if _, err := b.ModifyOrder("A1", "o1", "c1", &p, nil); !isValidation(err) {
		t.Fatal("both orderId and clientRequestId must be refused")
	}
	zero := 0.0
	if _, err := b.ModifyOrder("A1", "o1", "", &zero, nil); !isValidation(err) {
		t.Fatal("zero price must be refused")
	}
	text, err := b.ModifyOrder("A1", "o1", "", &p, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, text)
	if m["price"] != "10" || m["orderId"] != "o1" || m["clientModifyId"] == "" {
		t.Fatalf("modify payload: %v", m)
	}
	if _, ok := m["quantity"]; ok {
		t.Fatal("quantity must be omitted when only the price changes")
	}

	if _, err := b.CancelOrder("A1", "", ""); !isValidation(err) {
		t.Fatal("cancel needs one id")
	}
	if _, err := b.CancelOrder("A1", "o1", "c1"); !isValidation(err) {
		t.Fatal("cancel takes exactly one id")
	}
	text, err = b.CancelOrder("A1", "", "c1")
	if err != nil || decode(t, text)["clientCancelId"] == "" {
		t.Fatalf("%v %s", err, text)
	}
}

func TestBatchPlace(t *testing.T) {
	b := testBuilder()
	item := BatchPlaceOrderItem{AccountNo: "A1", Symbol: "VNM", Side: OrderSideBuy, Quantity: 100, Price: 25.5, OrderType: OrderTypeLO}
	before := time.Now().UnixMilli()
	text, err := b.BatchPlace([]BatchPlaceOrderItem{item, item})
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, text)
	orders := m["orders"].([]interface{})
	if len(orders) != 2 || m["batchRequestId"] == "" {
		t.Fatalf("batch: %v", m)
	}
	if ts := int64(m["batchRequestTime"].(float64)); ts < before {
		t.Fatalf("batchRequestTime must be epoch milliseconds now: %d", ts)
	}
	k1 := orders[0].(map[string]interface{})["clientRequestId"]
	k2 := orders[1].(map[string]interface{})["clientRequestId"]
	if k1 == "" || k1 == k2 {
		t.Fatalf("each order needs its own idempotency key: %v %v", k1, k2)
	}
}

func TestBatchLimitsAndAllOrNothing(t *testing.T) {
	b := testBuilder() // limit 3
	good := BatchPlaceOrderItem{AccountNo: "A1", Symbol: "VNM", Side: OrderSideBuy, Quantity: 1, Price: 10, OrderType: OrderTypeLO}
	if _, err := b.BatchPlace(nil); !isValidation(err) {
		t.Fatal("empty batch")
	}
	if _, err := b.BatchPlace([]BatchPlaceOrderItem{good, good, good, good}); !isValidation(err) {
		t.Fatal("batch over MaxBatchOrders must be refused locally")
	}
	bad := good
	bad.Price = 0
	if _, err := b.BatchPlace([]BatchPlaceOrderItem{good, bad}); !isValidation(err) {
		t.Fatal("one bad order must reject the whole batch")
	}
	if _, err := b.BatchCancel([]BatchCancelOrderItem{{AccountNo: "A1", OrderID: "o1"}, {AccountNo: "A1"}}); !isValidation(err) {
		t.Fatal("a cancel item without an id must reject the batch")
	}
	text, err := b.BatchCancel([]BatchCancelOrderItem{{AccountNo: "A1", OrderID: "o1"}, {AccountNo: "A1", ClientRequestID: "c2"}})
	if err != nil || len(decode(t, text)["orders"].([]interface{})) != 2 {
		t.Fatalf("%v %s", err, text)
	}
}

func TestFCOPlaceWindow(t *testing.T) {
	b := testBuilder()
	mk := func(from, to string) *GTDParams {
		return &GTDParams{AccountNo: "A1", Symbol: "VNM", Side: OrderSideBuy, Price: 25.5, PriceSlip: 0.1, Quantity: 100, FromDate: from, ToDate: to}
	}
	text, err := b.FCOPlace(mk("2099/01/01 09:00:00", "2099/01/10 15:00:00"))
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, text)
	if m["price"] != "25.5" || m["deviceId"] == "" || m["userAgent"] != "test-agent" {
		t.Fatalf("fco payload: %v", m)
	}
	if _, has := m["code"]; has {
		t.Fatal("empty OTP code must be omitted")
	}
	for name, p := range map[string]*GTDParams{
		"reversed":      mk("2099/01/10 09:00:00", "2099/01/01 09:00:00"),
		"over 31 days":  mk("2099/01/01 09:00:00", "2099/03/01 09:00:00"),
		"bad format":    mk("2099-01-01", "2099-01-02"),
		"missing dates": mk("", ""),
	} {
		if _, err := b.FCOPlace(p); !isValidation(err) {
			t.Errorf("%s: want ValidationError, got %v", name, err)
		}
	}
}

func TestFCOPlaceWordPriceHasNoSlip(t *testing.T) {
	p := &GTDParams{AccountNo: "A1", Symbol: "VNM", Side: OrderSideBuy, Price: OrderTypeMTL, PriceSlip: 5, Quantity: 1,
		FromDate: "2099/01/01 09:00:00", ToDate: "2099/01/02 09:00:00"}
	text, err := testBuilder().FCOPlace(p)
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, text)
	if m["price"] != "MTL" || m["priceSlip"] != float64(0) {
		t.Fatalf("%v", m)
	}
}

func TestFCOCancel(t *testing.T) {
	b := testBuilder()
	if _, err := b.FCOCancel(" ", ""); !isValidation(err) {
		t.Fatal("empty fcoId")
	}
	text, err := b.FCOCancel("f1", "123456")
	if err != nil {
		t.Fatal(err)
	}
	if m := decode(t, text); m["fcoId"] != "f1" || m["code"] != "123456" {
		t.Fatalf("%v", m)
	}
	text, _ = b.FCOCancel("f1", "")
	if _, has := decode(t, text)["code"]; has {
		t.Fatal("empty code must be omitted")
	}
}

func TestFCOListParams(t *testing.T) {
	b := testBuilder()
	if _, err := b.FCOListParams(&FCOListRequest{}); !isValidation(err) {
		t.Fatal("accountNo is required")
	}
	if _, err := b.FCOListParams(&FCOListRequest{AccountNo: "A1", ProcessStatus: []FCOStatus{"a", "b"}}); !isValidation(err) {
		t.Fatal("only one process status is allowed")
	}
	p, err := b.FCOListParams(&FCOListRequest{AccountNo: "A1", PageIndex: 2, PageSize: 50})
	if err != nil {
		t.Fatal(err)
	}
	if p["accountNo"] != "A1" || p["pageIndex"] != 2 || p["pageSize"] != 50 {
		t.Fatalf("%v", p)
	}
	p, _ = b.FCOListParams(&FCOListRequest{AccountNo: "A1"})
	if _, has := p["pageIndex"]; has {
		t.Fatal("page numbers only when > 0")
	}
}

func TestFCOSlipForWords(t *testing.T) {
	mk := func(price interface{}) *GTDParams {
		return &GTDParams{AccountNo: "A1", Symbol: "VNM", Side: OrderSideBuy, Price: price, PriceSlip: 5, Quantity: 1,
			FromDate: "2099/01/01 09:00:00", ToDate: "2099/01/02 09:00:00"}
	}
	cases := []struct {
		price interface{}
		slip  float64
	}{
		{"MP", 0}, {"mp", 0}, {" Ato ", 0}, {"ATC", 0}, {"MTL", 0}, {"MOK", 0}, {"MAK", 0},
		{OrderTypeMTL, 0}, {OrderTypeATO, 0},
		{OrderTypeLO, 5}, {OrderTypePLO, 5}, {"LO", 5}, {"PLO", 5}, {25.5, 5},
	}
	for _, c := range cases {
		text, err := testBuilder().FCOPlace(mk(c.price))
		if err != nil {
			t.Fatalf("%v: %v", c.price, err)
		}
		if got := decode(t, text)["priceSlip"]; got != c.slip {
			t.Errorf("price %v: slip %v, want %v", c.price, got, c.slip)
		}
	}
}

func TestFCOCancelUserAgentOnlyWhenConfigured(t *testing.T) {
	text, err := testBuilder().FCOCancel("f1", "")
	if err != nil {
		t.Fatal(err)
	}
	if decode(t, text)["userAgent"] != "test-agent" {
		t.Fatalf("configured user agent must be sent: %s", text)
	}
	cfg := ssi.NewConfig("c")
	text, err = NewPayloadBuilder(cfg).FCOCancel("f1", "")
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, text)
	if _, has := m["userAgent"]; has {
		t.Fatalf("userAgent must be omitted when Config.UserAgent is empty: %s", text)
	}
	if m["deviceId"] == nil || m["deviceId"] == "" {
		t.Fatalf("deviceId is still sent: %s", text)
	}
}
