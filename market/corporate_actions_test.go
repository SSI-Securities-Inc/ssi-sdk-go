package market

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

// caServer answers each request with the next canned body and records the query of every call.
type caServer struct {
	mu      sync.Mutex
	queries []url.Values
	paths   []string
	bodies  []string
}

func newCAServer(t *testing.T, bodies ...string) (*Service, *caServer) {
	t.Helper()
	rec := &caServer{bodies: bodies}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		i := len(rec.queries)
		rec.queries = append(rec.queries, r.URL.Query())
		rec.paths = append(rec.paths, r.URL.Path)
		if i >= len(rec.bodies) {
			w.WriteHeader(500)
			return
		}
		_, _ = w.Write([]byte(rec.bodies[i]))
	}))
	t.Cleanup(srv.Close)
	cfg := ssi.NewConfig("test-client")
	cfg.APIURL = srv.URL
	cfg.LogLevel = "ERROR"
	cfg.RateLimitPerSecond = 0
	cfg.MaxRetries = 0
	return NewService(transport.NewRestClient(cfg)), rec
}

const caPage1 = `{"pageSize":100,"pageIndex":1,"pagesCount":2,"itemsCount":3,"data":[
 {"symbol":"AAA","eventTitle":"Cash dividend","exrightDate":"2026/10/20","recordDate":"2026/10/21","issueDate":"2026/11/05","exchange":"HOSE","ratio":"0.9388"},
 {"symbol":"BBB","eventTitle":"Bonus shares","exrightDate":"","recordDate":"2026/10/22","issueDate":"2026/11/06","exchange":"HNX","ratio":""}]}`

const caPage2 = `{"pageSize":100,"pageIndex":2,"pagesCount":2,"itemsCount":3,"data":[
 {"symbol":"CCC","eventTitle":"Rights issue","exrightDate":"2026/10/23","recordDate":"2026/10/24","issueDate":"2026/11/07","exchange":"UPCOM","ratio":"10:1"}]}`

func TestGetCorporateActionsHistoricalFetchesAllPages(t *testing.T) {
	s, rec := newCAServer(t, caPage1, caPage2)
	got, err := s.GetCorporateActionsHistorical("2026/10/19", "2026/10/24",
		WithCorporateActionsSymbol("AAA"), WithCorporateActionsLanguage("EN"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.queries) != 2 {
		t.Fatalf("want 2 requests, got %d", len(rec.queries))
	}
	for i, q := range rec.queries {
		if rec.paths[i] != "/api/v3/data/corporateActions" {
			t.Fatalf("path %q", rec.paths[i])
		}
		want := map[string]string{
			"from": "2026/10/19", "to": "2026/10/24", "symbol": "AAA", "language": "en",
			"pageSize": "100", "pageIndex": []string{"1", "2"}[i],
		}
		for k, v := range want {
			if q.Get(k) != v {
				t.Errorf("request %d: %s=%q, want %q", i+1, k, q.Get(k), v)
			}
		}
	}
	if len(got) != 3 {
		t.Fatalf("want 3 items, got %d", len(got))
	}
	first := CorporateAction{
		Symbol: "AAA", EventTitle: "Cash dividend", ExrightDate: "2026/10/20", RecordDate: "2026/10/21",
		IssueDate: "2026/11/05", Exchange: "HOSE", Ratio: "0.9388",
	}
	if got[0] != first {
		t.Fatalf("got %+v", got[0])
	}
	if got[1].ExrightDate != "" || got[1].Ratio != "" {
		t.Fatalf("blank fields must stay empty: %+v", got[1])
	}
	if got[2].Symbol != "CCC" || got[2].Ratio != "10:1" {
		t.Fatalf("page 2 item: %+v", got[2])
	}
}

func TestGetCorporateActionsTodayUsesSameFromAndTo(t *testing.T) {
	s, rec := newCAServer(t, `{"pageSize":100,"pageIndex":1,"pagesCount":1,"itemsCount":0,"data":[]}`)
	before := util.TodayDateStr()
	if _, err := s.GetCorporateActions(); err != nil {
		t.Fatal(err)
	}
	after := util.TodayDateStr()
	q := rec.queries[0]
	if q.Get("from") != q.Get("to") || (q.Get("from") != before && q.Get("from") != after) {
		t.Fatalf("today must send from==to==today: %v", q)
	}
	if q.Get("language") != "vn" || q.Get("pageIndex") != "1" || q.Get("pageSize") != "100" {
		t.Fatalf("defaults: %v", q)
	}
	if _, has := q["symbol"]; has {
		t.Fatalf("symbol must be omitted when not given: %v", q)
	}
}

func TestGetCorporateActionsNoContentIsEmptyList(t *testing.T) {
	s, rec := newCAServer(t, `{"code":204}`)
	got, err := s.GetCorporateActions()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("want empty list, got %v", got)
	}
	if len(rec.queries) != 1 {
		t.Fatalf("a 204 marker ends the loop: %d requests", len(rec.queries))
	}
}

func TestGetCorporateActionsNullAndMissingFieldsBecomeEmpty(t *testing.T) {
	s, _ := newCAServer(t, `{"pagesCount":1,"data":[{"symbol":"AAA","eventTitle":null,"ratio":null,"exchange":"HOSE"}]}`)
	got, err := s.GetCorporateActions()
	if err != nil {
		t.Fatal(err)
	}
	want := CorporateAction{Symbol: "AAA", Exchange: "HOSE"}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("got %+v", got)
	}
}

func TestGetCorporateActionsHistoricalAcceptsFiveDaySpan(t *testing.T) {
	s, rec := newCAServer(t, `{"code":204}`)
	if _, err := s.GetCorporateActionsHistorical("2026/10/01", "2026/10/06"); err != nil {
		t.Fatal(err)
	}
	if len(rec.queries) != 1 {
		t.Fatalf("a span of exactly 5 days is valid: %d requests", len(rec.queries))
	}
}

func TestGetCorporateActionsValidationSendsNoRequest(t *testing.T) {
	long21 := strings.Repeat("A", 21)
	cases := []struct {
		name     string
		from, to string
		opts     []CorporateActionsOption
	}{
		{"range over 5 days", "2026/10/01", "2026/10/07", nil},
		{"from after to", "2026/10/05", "2026/10/04", nil},
		{"dash date format", "2026-10-01", "2026-10-02", nil},
		{"bad from only", "2026/10/01x", "2026/10/02", nil},
		{"empty date", "", "2026/10/02", nil},
		{"symbol with separator", "2026/10/01", "2026/10/02", []CorporateActionsOption{WithCorporateActionsSymbol("AAA,BBB")}},
		{"symbol with punctuation", "2026/10/01", "2026/10/02", []CorporateActionsOption{WithCorporateActionsSymbol("AA-A")}},
		{"empty symbol", "2026/10/01", "2026/10/02", []CorporateActionsOption{WithCorporateActionsSymbol("")}},
		{"symbol over 20 chars", "2026/10/01", "2026/10/02", []CorporateActionsOption{WithCorporateActionsSymbol(long21)}},
		{"bad language", "2026/10/01", "2026/10/02", []CorporateActionsOption{WithCorporateActionsLanguage("fr")}},
		{"empty language", "2026/10/01", "2026/10/02", []CorporateActionsOption{WithCorporateActionsLanguage("")}},
	}
	for _, c := range cases {
		s, rec := newCAServer(t, `{"code":204}`)
		_, err := s.GetCorporateActionsHistorical(c.from, c.to, c.opts...)
		if !isValidation(err) {
			t.Errorf("%s: want ValidationError, got %v", c.name, err)
		}
		if len(rec.queries) != 0 {
			t.Errorf("%s: %d request(s) sent, want none", c.name, len(rec.queries))
		}
	}

	// the today method validates symbol / language before any request too
	s, rec := newCAServer(t, `{"code":204}`)
	if _, err := s.GetCorporateActions(WithCorporateActionsSymbol("AA A")); !isValidation(err) {
		t.Errorf("today with bad symbol: want ValidationError, got %v", err)
	}
	if _, err := s.GetCorporateActions(WithCorporateActionsLanguage("jp")); !isValidation(err) {
		t.Errorf("today with bad language: want ValidationError, got %v", err)
	}
	if len(rec.queries) != 0 {
		t.Errorf("%d request(s) sent, want none", len(rec.queries))
	}
}

func TestBuildCorporateActionsParamsPagingRules(t *testing.T) {
	if _, err := buildCorporateActionsParams("2026/10/01", "2026/10/02", nil, "vn", 0, 100); !isValidation(err) {
		t.Errorf("page 0: want ValidationError, got %v", err)
	}
	for _, size := range []int{0, 5, 11, 1000} {
		if _, err := buildCorporateActionsParams("2026/10/01", "2026/10/02", nil, "vn", 1, size); !isValidation(err) {
			t.Errorf("size %d: want ValidationError, got %v", size, err)
		}
	}
	for _, size := range []int{10, 20, 50, 100} {
		if _, err := buildCorporateActionsParams("2026/10/01", "2026/10/02", nil, "vn", 1, size); err != nil {
			t.Errorf("size %d: %v", size, err)
		}
	}
	sym := strings.Repeat("A", 20)
	if _, err := buildCorporateActionsParams("2026/10/01", "2026/10/02", &sym, " VN ", 1, 10); err != nil {
		t.Errorf("20-char symbol and a padded upper-case language are valid: %v", err)
	}
}
