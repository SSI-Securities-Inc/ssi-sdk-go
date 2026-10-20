package tradingws

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	ssisdk "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

func b64(i *big.Int) string { return base64.StdEncoding.EncodeToString(i.Bytes()) }

// testKey generates a throw-away key in the SDK's base64(XML) format.
func testKey(t *testing.T) (string, *rsa.PrivateKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	x := "<RSAKeyValue><Modulus>" + b64(priv.N) + "</Modulus><Exponent>" + b64(big.NewInt(int64(priv.E))) +
		"</Exponent><D>" + b64(priv.D) + "</D></RSAKeyValue>"
	return base64.StdEncoding.EncodeToString([]byte(x)), priv
}

func verify(t *testing.T, pub *rsa.PublicKey, text, sig, encoding string) {
	t.Helper()
	var raw []byte
	var err error
	if encoding == "hex" {
		raw, err = hex.DecodeString(sig)
	} else {
		raw, err = base64.StdEncoding.DecodeString(sig)
	}
	if err != nil {
		t.Fatalf("signature is not %s: %v", encoding, err)
	}
	h := sha256.Sum256([]byte(text))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], raw); err != nil {
		t.Fatalf("signature does not verify over the params text: %v", err)
	}
}

func TestBuildCommandSignsTheExactParamsText(t *testing.T) {
	key, priv := testKey(t)
	params := `{"accountNo":"X","price":25.5,"note":"<&>"}`
	raw, err := BuildCommand("order.place", params, key, "abc", 1700000000123, "base64")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(raw, `"params":`+params+`,"signature":`) {
		t.Fatalf("params must be inserted verbatim: %s", raw)
	}
	var frame struct {
		ID        string          `json:"id"`
		Method    string          `json:"method"`
		Params    json.RawMessage `json:"params"`
		Signature string          `json:"signature"`
		Timestamp int64           `json:"timestamp"`
	}
	if err := json.Unmarshal([]byte(raw), &frame); err != nil {
		t.Fatal(err)
	}
	if frame.ID != "abc" || frame.Method != "order.place" || frame.Timestamp != 1700000000123 {
		t.Fatalf("bad frame: %+v", frame)
	}
	verify(t, &priv.PublicKey, params, frame.Signature, "base64")
}

func TestBuildCommandQueryIsNotSignedAndNeedsNoKey(t *testing.T) {
	raw, err := BuildCommand("query.position", `{"accountNo":"X"}`, "", "i1", 1, "base64")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "signature") {
		t.Fatalf("queries are not signed: %s", raw)
	}
}

func TestBuildCommandFailsClosed(t *testing.T) {
	if _, err := BuildCommand("ORDER.PLACE", `{}`, "", "i1", 1, "base64"); err == nil {
		t.Fatal("a state-changing command without a private key must not be built (case-insensitive)")
	}
	if _, err := BuildCommand("query.position", `{}`, "", "", 1, "base64"); err == nil {
		t.Fatal("an empty id must be refused")
	}
	if _, err := BuildCommand("query.position", `{bad`, "", "i1", 1, "base64"); err == nil {
		t.Fatal("invalid params JSON must be refused")
	}
}

func TestUnwrap(t *testing.T) {
	ok, err := Unwrap(map[string]interface{}{"id": "1", "status": float64(200)}, "")
	if err != nil || ok == nil {
		t.Fatalf("success frame: %v", err)
	}
	_, err = Unwrap(map[string]interface{}{"status": float64(409), "error": map[string]interface{}{"code": "DUP", "message": "dup"}}, "")
	var twe *ssisdk.TradingWSError
	if !errors.As(err, &twe) || twe.Status != 409 || twe.ErrorCode != "DUP" || twe.ServerMessage != "dup" {
		t.Fatalf("error frame: %v", err)
	}
	if twe.Error() != "409 DUP: dup" {
		t.Fatalf("message: %q", twe.Error())
	}
	_, err = Unwrap(map[string]interface{}{"status": "error", "error": map[string]interface{}{"code": "FCO_ERROR", "message": "m"}}, "")
	if !errors.As(err, &twe) || twe.Status != 0 || twe.ErrorCode != "FCO_ERROR" {
		t.Fatalf("status \"error\": %v", err)
	}
	_, err = Unwrap(map[string]interface{}{"status": float64(401), "error": map[string]interface{}{"code": "401013", "message": "bad sig"}}, "base64")
	if !errors.As(err, &twe) || !strings.Contains(twe.ServerMessage, "signed as base64") || !strings.Contains(twe.ServerMessage, `"hex"`) {
		t.Fatalf("signature hint missing: %v", err)
	}
}

func TestRejectionFrames(t *testing.T) {
	f := map[string]interface{}{"code": float64(403), "msg": "no scope"}
	if !IsRejection(f) {
		t.Fatal("a code+msg frame without id is a rejection")
	}
	if IsRejection(map[string]interface{}{"id": "1", "code": float64(403), "msg": "x"}) {
		t.Fatal("a frame with an id is not a rejection")
	}
	e := RejectionError(f)
	if e.Status != 403 || e.ServerMessage != "no scope" {
		t.Fatalf("rejection error: %+v", e)
	}
}

func TestNewClientRequiresWSS(t *testing.T) {
	_, err := NewClient("ws://example.test/ws/v3/trading", func() (string, error) { return "t", nil }, "")
	if err == nil {
		t.Fatal("ws:// must be refused")
	}
	if _, err := NewClient("wss://example.test/x", func() (string, error) { return "t", nil }, "", WithSignatureEncoding("rot13")); err == nil {
		t.Fatal("unknown signature encoding must be refused")
	}
}

// ── fake socket ──────────────────────────────────────────────────────────────

type fakeConn struct {
	in      chan []byte
	closed  chan struct{}
	once    sync.Once
	mu      sync.Mutex
	writes  []string
	onWrite func(string)
}

func newFakeConn() *fakeConn {
	return &fakeConn{in: make(chan []byte, 64), closed: make(chan struct{})}
}

func (f *fakeConn) ReadMessage() (int, []byte, error) {
	select {
	case m := <-f.in:
		return 1, m, nil
	case <-f.closed:
		return 0, nil, errors.New("closed")
	}
}

func (f *fakeConn) WriteMessage(_ int, data []byte) error {
	f.mu.Lock()
	f.writes = append(f.writes, string(data))
	cb := f.onWrite
	f.mu.Unlock()
	if cb != nil {
		cb(string(data))
	}
	return nil
}

func (f *fakeConn) Close() error { f.once.Do(func() { close(f.closed) }); return nil }

func (f *fakeConn) written() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.writes...)
}

func (f *fakeConn) push(v interface{}) {
	b, _ := json.Marshal(v)
	f.in <- b
}

type sentFrame struct {
	ID        string          `json:"id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params"`
	Signature string          `json:"signature"`
}

func parseSent(t *testing.T, raw string) sentFrame {
	t.Helper()
	var s sentFrame
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("sent frame is not JSON: %v", err)
	}
	return s
}

func newTestClient(t *testing.T, fc *fakeConn, key string, opts ...Option) *Client {
	t.Helper()
	dial := func(_ context.Context, _ string, _ http.Header, _ time.Duration) (conn, error) { return fc, nil }
	all := append([]Option{
		WithHandshakeGraceMs(0), WithPingIntervalMs(0), WithRequestTimeoutMs(300),
		func(o *options) { o.dial = dial },
	}, opts...)
	c, err := NewClient("wss://example.test/ws/v3/trading", func() (string, error) { return "tok", nil }, key, all...)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

func TestRequestReturnsFirstFrameAndSignsParams(t *testing.T) {
	key, priv := testKey(t)
	fc := newFakeConn()
	fc.onWrite = func(raw string) {
		var s sentFrame
		_ = json.Unmarshal([]byte(raw), &s)
		fc.push(map[string]interface{}{"id": s.ID, "method": s.Method, "status": 200, "result": map[string]interface{}{"orderStatus": "PD"},
			"rateLimit": map[string]interface{}{"remaining": 49}})
	}
	c := newTestClient(t, fc, key)
	resp, err := c.Request(context.Background(), "order.place", map[string]interface{}{"accountNo": "A1", "quantity": 100})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != 200 || resp.Result.(map[string]interface{})["orderStatus"] != "PD" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if c.RateLimit()["remaining"] != float64(49) {
		t.Fatalf("rate limit not recorded: %v", c.RateLimit())
	}
	sent := parseSent(t, fc.written()[0])
	verify(t, &priv.PublicKey, string(sent.Params), sent.Signature, "base64")
}

func TestAutoModeRetriesOnceWithHexWhenSignatureRefused(t *testing.T) {
	key, priv := testKey(t)
	fc := newFakeConn()
	fc.onWrite = func(raw string) {
		var s sentFrame
		_ = json.Unmarshal([]byte(raw), &s)
		if _, err := base64.StdEncoding.DecodeString(s.Signature); err == nil && len(fc.written()) == 1 {
			fc.push(map[string]interface{}{"id": s.ID, "status": 401, "error": map[string]interface{}{"code": "401013", "message": "bad signature"}})
			return
		}
		fc.push(map[string]interface{}{"id": s.ID, "status": 200, "result": map[string]interface{}{}})
	}
	c := newTestClient(t, fc, key)
	if _, err := c.Request(context.Background(), "order.place", map[string]interface{}{"a": 1}); err != nil {
		t.Fatal(err)
	}
	w := fc.written()
	if len(w) != 2 {
		t.Fatalf("expected exactly one retry, got %d writes", len(w))
	}
	first, second := parseSent(t, w[0]), parseSent(t, w[1])
	if first.ID == second.ID {
		t.Fatal("the retry must use a fresh id")
	}
	verify(t, &priv.PublicKey, string(second.Params), second.Signature, "hex")

	// the working encoding is remembered
	if _, err := c.Request(context.Background(), "order.place", map[string]interface{}{"a": 2}); err != nil {
		t.Fatal(err)
	}
	third := parseSent(t, fc.written()[len(fc.written())-1])
	verify(t, &priv.PublicKey, string(third.Params), third.Signature, "hex")
}

func TestBothEncodingsRefused(t *testing.T) {
	key, _ := testKey(t)
	fc := newFakeConn()
	fc.onWrite = func(raw string) {
		var s sentFrame
		_ = json.Unmarshal([]byte(raw), &s)
		fc.push(map[string]interface{}{"id": s.ID, "status": 401, "error": map[string]interface{}{"code": "401008", "message": "nope"}})
	}
	c := newTestClient(t, fc, key)
	_, err := c.Request(context.Background(), "order.cancel", map[string]interface{}{"a": 1})
	var twe *ssisdk.TradingWSError
	if !errors.As(err, &twe) || !strings.Contains(twe.ServerMessage, "refused with a base64 and a hex signature") {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := len(fc.written()); n != 2 {
		t.Fatalf("expected 2 attempts, got %d", n)
	}
}

func TestFixedEncodingNeverRetries(t *testing.T) {
	key, _ := testKey(t)
	fc := newFakeConn()
	fc.onWrite = func(raw string) {
		var s sentFrame
		_ = json.Unmarshal([]byte(raw), &s)
		fc.push(map[string]interface{}{"id": s.ID, "status": 401, "error": map[string]interface{}{"code": "401013", "message": "x"}})
	}
	c := newTestClient(t, fc, key, WithSignatureEncoding(EncodingHex))
	if _, err := c.Request(context.Background(), "order.place", map[string]interface{}{"a": 1}); err == nil {
		t.Fatal("expected an error")
	}
	if n := len(fc.written()); n != 1 {
		t.Fatalf("a fixed encoding must not retry, got %d writes", n)
	}
}

func TestTimeoutIsNeverRetried(t *testing.T) {
	key, _ := testKey(t)
	fc := newFakeConn() // never answers
	c := newTestClient(t, fc, key, WithRequestTimeoutMs(50))
	_, err := c.Request(context.Background(), "order.place", map[string]interface{}{"a": 1})
	var wse *ssisdk.WebSocketError
	if !errors.As(err, &wse) || !strings.Contains(err.Error(), "not retried") {
		t.Fatalf("expected a timeout WebSocketError, got %v", err)
	}
	if n := len(fc.written()); n != 1 {
		t.Fatalf("a timed-out command must be sent once, got %d", n)
	}
}

func TestRateLimitZeroRemainingDelaysNextCommand(t *testing.T) {
	fc := newFakeConn()
	fc.onWrite = func(raw string) {
		var s sentFrame
		_ = json.Unmarshal([]byte(raw), &s)
		fc.push(map[string]interface{}{"id": s.ID, "status": 200, "result": map[string]interface{}{},
			"rateLimit": map[string]interface{}{"remaining": 0, "resetMs": 60000}})
	}
	var slept []time.Duration
	base := time.Unix(1700000000, 0)
	c := newTestClient(t, fc, "", func(o *options) {
		o.now = func() time.Time { return base }
		o.sleep = func(_ context.Context, d time.Duration) error { slept = append(slept, d); return nil }
	})
	for i := 0; i < 2; i++ {
		if _, err := c.Request(context.Background(), "query.position", map[string]interface{}{"accountNo": "A"}); err != nil {
			t.Fatal(err)
		}
	}
	if len(slept) != 1 || slept[0] != 2*time.Second {
		t.Fatalf("the wait must be capped at MaxRateLimitWait (2s): %v", slept)
	}
}

func TestPostUpgradeRefusalFailsConnect(t *testing.T) {
	fc := newFakeConn()
	fc.push(map[string]interface{}{"code": 403, "msg": "missing scope"})
	dial := func(_ context.Context, _ string, _ http.Header, _ time.Duration) (conn, error) { return fc, nil }
	c, err := NewClient("wss://example.test/ws/v3/trading", func() (string, error) { return "tok", nil }, "",
		WithPingIntervalMs(0), WithHandshakeGraceMs(1000), func(o *options) { o.dial = dial })
	if err != nil {
		t.Fatal(err)
	}
	err = c.Connect(context.Background())
	var twe *ssisdk.TradingWSError
	if !errors.As(err, &twe) || twe.Status != 403 {
		t.Fatalf("expected the 403 refusal, got %v", err)
	}
	if c.IsConnected() {
		t.Fatal("a refused connection must be closed")
	}
}

func TestDialErrorDoesNotLeakToken(t *testing.T) {
	dial := func(_ context.Context, _ string, h http.Header, _ time.Duration) (conn, error) {
		return nil, errors.New("handshake refused with HTTP 403")
	}
	c, _ := NewClient("wss://example.test/ws/v3/trading", func() (string, error) { return "SECRET-TOKEN-VALUE", nil }, "",
		func(o *options) { o.dial = dial })
	err := c.Connect(context.Background())
	if err == nil || strings.Contains(err.Error(), "SECRET-TOKEN-VALUE") {
		t.Fatalf("bad error: %v", err)
	}
}

func TestEventsWithoutIDGoToOnEvent(t *testing.T) {
	fc := newFakeConn()
	c := newTestClient(t, fc, "")
	got := make(chan map[string]interface{}, 1)
	c.SetOnEvent(func(m map[string]interface{}) { got <- m })
	fc.push(map[string]interface{}{"channel": "TRADING", "topic": "order.X", "data": map[string]interface{}{"eventType": "orderEvent"}})
	select {
	case m := <-got:
		if m["topic"] != "order.X" {
			t.Fatalf("unexpected event: %v", m)
		}
	case <-time.After(time.Second):
		t.Fatal("event not delivered")
	}
}

func TestSubscribeFramesAndNotConnected(t *testing.T) {
	fc := newFakeConn()
	c := newTestClient(t, fc, "")
	if err := c.Subscribe("TRADING", []string{"order.A"}); err != nil {
		t.Fatal(err)
	}
	if err := c.Unsubscribe("TRADING", nil); err != nil {
		t.Fatal(err)
	}
	w := fc.written()
	if w[0] != `{"method":"SUBSCRIBE","channel":"TRADING","topics":["order.A"]}` {
		t.Fatalf("subscribe frame: %s", w[0])
	}
	if w[1] != `{"method":"UNSUBSCRIBE","channel":"TRADING","topics":[]}` {
		t.Fatalf("unsubscribe-all frame must carry an empty list: %s", w[1])
	}
	c.Close()
	if err := c.Ping(); err == nil {
		t.Fatal("ping after Close must fail")
	}
}

func TestCloseWakesWaiter(t *testing.T) {
	fc := newFakeConn()
	c := newTestClient(t, fc, "", WithRequestTimeoutMs(5000))
	errc := make(chan error, 1)
	go func() {
		_, err := c.Request(context.Background(), "query.position", map[string]interface{}{"accountNo": "A"})
		errc <- err
	}()
	for len(fc.written()) == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	c.Close()
	select {
	case err := <-errc:
		if err == nil || !strings.Contains(err.Error(), "connection closed") {
			t.Fatalf("unexpected: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("waiter was not woken")
	}
}

func TestQueryOrderBookValidatesFromBeforeTo(t *testing.T) {
	s := &Service{}
	cases := []struct{ from, to, want string }{
		{"bad", "also-bad", "from"},
		{"2099/01/01", "bad", "to"},
		{"", "bad", "to"},
		{"bad", "", "from"},
	}
	for _, c := range cases {
		for i := 0; i < 20; i++ {
			_, err := s.QueryOrderBook(context.Background(), OrderBookQuery{AccountNo: "A1", FromDate: c.from, ToDate: c.to})
			var ve *ssisdk.ValidationError
			if !errors.As(err, &ve) || !strings.Contains(ve.Error(), c.want) {
				t.Fatalf("from=%q to=%q: want ValidationError naming %q, got %v", c.from, c.to, c.want, err)
			}
		}
	}
}
