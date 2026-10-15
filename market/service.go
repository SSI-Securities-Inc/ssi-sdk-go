package market

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var marketLog = logger.New("ssi_sdk.services.market")

const (
	defaultSize = ssi.DefaultSize
	defaultPage = ssi.DefaultPage

	epDataOHLC              = "/api/v3/data/ohlc"
	epDataIndexList         = "/api/v3/data/indexList"
	epDataIndexSummary      = "/api/v3/data/indexSummary"
	epDataSecuritiesByBoard = "/api/v3/data/securitiesByBoard"
	epDataSecuritiesSummary = "/api/v3/data/securitiesSummary"
	epDataMasterData        = "/api/v3/data/masterdata"

	weekMonthRejected = "The server does not support the %s timeframe (error 400210); " +
		"request 1d bars and aggregate them client-side instead"
)

// ErrNotImplemented is returned by the bulk OHLC download methods: the server has no
// data/file endpoint yet (its final response shape is undecided).
var ErrNotImplemented = errors.New("OHLC download is not implemented yet")

// Service provides market data operations.
type Service struct {
	rest *transport.RestClient
}

func NewService(rest *transport.RestClient) *Service {
	return &Service{rest: rest}
}

// rowsOf returns the rows of a list response. The REST client wraps a bare JSON array as
// {"data": [...]}; the server answers an empty result with HTTP 200 and {"code": 204} (an
// object, not []), which maps to no rows rather than a parse error.
func rowsOf(data map[string]interface{}, what string) ([]interface{}, error) {
	if data == nil || util.IsNoContent(data) {
		return nil, nil
	}
	rows, present := data["data"]
	if !present || rows == nil {
		return nil, nil
	}
	list, ok := rows.([]interface{})
	if !ok {
		return nil, ssi.NewAPIError("Unexpected response format while reading "+what, "", 0, nil)
	}
	return list, nil
}

// pagesCount is the total pages of a paginated response (pagesCount, falling back to pageCount).
func pagesCount(data map[string]interface{}) int {
	n := util.ToOptInt(util.Pick(data, "pagesCount", "pageCount"))
	if n == nil {
		return 1
	}
	return *n
}

// restBoard validates a board for a REST data endpoint (HOSE/HNX/UPCOM only).
func restBoard(board Board) (*Board, error) {
	resolved, ok := BoardFromValue(string(board))
	allowed := false
	for _, b := range restBoards {
		if b == resolved {
			allowed = true
		}
	}
	if !ok || !allowed {
		names := make([]string, 0, len(restBoards))
		for _, b := range restBoards {
			names = append(names, string(b))
		}
		return nil, ssi.NewValidationError(fmt.Sprintf("board must be one of %s, got '%s'", strings.Join(names, ", "), board))
	}
	return &resolved, nil
}

func checkTimeframe(tf Timeframe) error {
	if AllowedTimeframes[tf] {
		return nil
	}
	allowed := make([]string, 0, len(AllowedTimeframes))
	for t := range AllowedTimeframes {
		allowed = append(allowed, string(t))
	}
	sort.Strings(allowed)
	return ssi.NewValidationError(fmt.Sprintf("timeframe %s is not supported; use one of %s", tf, strings.Join(allowed, ", ")))
}

// checkPaging: the server validates neither pageIndex nor pageSize; 0 or negative values give
// undefined results, so refuse them here.
func checkPaging(page, size int) error {
	if err := ssi.RequirePositive(float64(page), "page"); err != nil {
		return err
	}
	return ssi.RequirePositive(float64(size), "size")
}

// clampFutureEnd: for minute/hour bars a "to" later than today makes the server answer nothing
// at all, so cut it to the end of today. Daily bars are not affected, and a range that starts
// in the future is left alone (it is empty anyway).
func clampFutureEnd(tf Timeframe, fromDate, toDate string, now time.Time) (string, error) {
	if tf == TimeframeDay1 {
		return toDate, nil
	}
	to, err := ssi.ParseDateArg(toDate, "to", true, false)
	if err != nil {
		return "", err
	}
	from, err := ssi.ParseDateArg(fromDate, "from", true, false)
	if err != nil {
		return "", err
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayOf := func(t time.Time) time.Time { return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()) }
	if dayOf(to).After(today) && !dayOf(from).After(today) {
		return now.Format(ssi.DateLayout) + " 23:59:59", nil
	}
	return toDate, nil
}

var dateOnly = regexp.MustCompile(`^\d{4}[/-]\d{2}[/-]\d{2}$`)

// withTime completes YYYY/MM/DD to "YYYY/MM/DD <timeText>": the OHLC endpoint answers 400213
// (Invalid Date/Timestamp Format) to a bare date.
func withTime(value, timeText string) string {
	text := strings.TrimSpace(value)
	if dateOnly.MatchString(text) {
		return strings.ReplaceAll(text, "-", "/") + " " + timeText
	}
	return value
}

func buildOHLCParams(symbol string, tf Timeframe, fromDate, toDate string, page, size int, now time.Time) (map[string]string, error) {
	if err := ssi.RequireSymbol(symbol, "symbol"); err != nil {
		return nil, err
	}
	if err := checkTimeframe(tf); err != nil {
		return nil, err
	}
	if err := checkPaging(page, size); err != nil {
		return nil, err
	}
	if fromDate == "" {
		fromDate = now.Format(ssi.DateLayout) + " 00:00:00"
	}
	if toDate == "" {
		toDate = now.Format(ssi.DateLayout) + " 23:59:59"
	}
	if err := ssi.RequireDateRange(fromDate, toDate, true, false, 0); err != nil {
		return nil, err
	}
	fromDate = withTime(fromDate, "00:00:00")
	toDate = withTime(toDate, "23:59:59")
	toDate, err := clampFutureEnd(tf, fromDate, toDate, now)
	if err != nil {
		return nil, err
	}
	return (&OHLCRequest{
		Symbol: symbol, FromDate: fromDate, ToDate: toDate, Timeframe: tf, Page: page, Size: size,
	}).ToMap(), nil
}

func (s *Service) getOHLC(symbol string, tf Timeframe, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	params, err := buildOHLCParams(symbol, tf, fromDate, toDate, page, size, time.Now())
	if err != nil {
		return nil, err
	}
	data, err := s.rest.Get(epDataOHLC, params, nil)
	if err != nil {
		marketLog.Error("Error fetching OHLC data for symbol %s, timeframe %s: %v", symbol, string(tf), err)
		return nil, err
	}
	rows, err := rowsOf(data, "OHLC")
	if err != nil {
		return nil, err
	}
	return OHLCDataFromList(rows), nil
}

// OHLCQuery selects the range and paging of IterOHLC / GetOHLCAll.
type OHLCQuery struct {
	// FromDate is the range start; defaults to the start of today. YYYY/MM/DD is completed to
	// 00:00:00.
	FromDate string
	// ToDate is the range end; defaults to the end of today. YYYY/MM/DD is completed to 23:59:59.
	ToDate string
	// Size is the bars requested per page (> 0); defaults to 1000.
	Size int
	// Ascending (GetOHLCAll only) returns the oldest bar first; the server sends newest first.
	Ascending bool
}

func (q OHLCQuery) size() int {
	if q.Size == 0 {
		return defaultSize
	}
	return q.Size
}

// IterOHLC calls yield with every OHLC bar in a range, fetching page after page. The OHLC
// endpoint reports no page count, so paging stops at the first empty or short page. Bars arrive
// in server order: newest first. Return false from yield to stop early.
func (s *Service) IterOHLC(symbol string, tf Timeframe, q OHLCQuery, yield func(OHLCData) bool) error {
	size := q.size()
	if err := ssi.RequirePositive(float64(size), "size"); err != nil {
		return err
	}
	for page := defaultPage; ; page++ {
		bars, err := s.getOHLC(symbol, tf, q.FromDate, q.ToDate, page, size)
		if err != nil {
			return err
		}
		for _, bar := range bars {
			if !yield(bar) {
				return nil
			}
		}
		if len(bars) < size {
			return nil
		}
	}
}

// GetOHLCAll fetches every OHLC bar in a range across all pages.
func (s *Service) GetOHLCAll(symbol string, tf Timeframe, q OHLCQuery) ([]OHLCData, error) {
	var bars []OHLCData
	err := s.IterOHLC(symbol, tf, q, func(b OHLCData) bool {
		bars = append(bars, b)
		return true
	})
	if err != nil {
		return nil, err
	}
	if q.Ascending {
		for i, j := 0, len(bars)-1; i < j; i, j = i+1, j-1 {
			bars[i], bars[j] = bars[j], bars[i]
		}
	}
	return bars, nil
}

// DownloadOHLC1Minute is not implemented: the server has no data/file endpoint yet.
//
// Deprecated: use GetOHLCAll or IterOHLC.
func (s *Service) DownloadOHLC1Minute(symbol string) ([]OHLCData, error) {
	return nil, ErrNotImplemented
}

// DownloadOHLC1Day is not implemented: the server has no data/file endpoint yet.
//
// Deprecated: use GetOHLCAll or IterOHLC.
func (s *Service) DownloadOHLC1Day(symbol string) ([]OHLCData, error) {
	return nil, ErrNotImplemented
}

func (s *Service) historical(symbol, fromDate, toDate string, tf Timeframe, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(fromDate, "fromDate"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "toDate"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, tf, fromDate, toDate, page, size)
}

func (s *Service) GetOHLC1Minute(symbol string) ([]OHLCData, error) {
	return s.getOHLC(symbol, TimeframeMinute1, "", "", defaultPage, defaultSize)
}

func (s *Service) GetOHLC1MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	return s.historical(symbol, fromDate, toDate, TimeframeMinute1, page, size)
}

func (s *Service) GetOHLC3Minute(symbol string) ([]OHLCData, error) {
	return s.getOHLC(symbol, TimeframeMinute3, "", "", defaultPage, defaultSize)
}

func (s *Service) GetOHLC3MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	return s.historical(symbol, fromDate, toDate, TimeframeMinute3, page, size)
}

func (s *Service) GetOHLC5Minute(symbol string) ([]OHLCData, error) {
	return s.getOHLC(symbol, TimeframeMinute5, "", "", defaultPage, defaultSize)
}

func (s *Service) GetOHLC5MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	return s.historical(symbol, fromDate, toDate, TimeframeMinute5, page, size)
}

func (s *Service) GetOHLC15Minute(symbol string) ([]OHLCData, error) {
	return s.getOHLC(symbol, TimeframeMinute15, "", "", defaultPage, defaultSize)
}

func (s *Service) GetOHLC15MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	return s.historical(symbol, fromDate, toDate, TimeframeMinute15, page, size)
}

func (s *Service) GetOHLC1Hour(symbol string) ([]OHLCData, error) {
	return s.getOHLC(symbol, TimeframeHour1, "", "", defaultPage, defaultSize)
}

func (s *Service) GetOHLC1HourHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	return s.historical(symbol, fromDate, toDate, TimeframeHour1, page, size)
}

func (s *Service) GetOHLC1DayHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	return s.historical(symbol, fromDate, toDate, TimeframeDay1, page, size)
}

// GetOHLC1WeekHistorical always returns a ValidationError: the server does not serve 1-week
// bars (error 400210). Request 1-day bars and aggregate them client-side.
//
// Deprecated: the server rejects the 1w timeframe.
func (s *Service) GetOHLC1WeekHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := s.requireRange(fromDate, toDate); err != nil {
		return nil, err
	}
	return nil, ssi.NewValidationError(fmt.Sprintf(weekMonthRejected, "1w"))
}

// GetOHLC1MonthHistorical always returns a ValidationError: the server does not serve 1-month
// bars (error 400210). Request 1-day bars and aggregate them client-side.
//
// Deprecated: the server rejects the 1M timeframe.
func (s *Service) GetOHLC1MonthHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := s.requireRange(fromDate, toDate); err != nil {
		return nil, err
	}
	return nil, ssi.NewValidationError(fmt.Sprintf(weekMonthRejected, "1M"))
}

func (s *Service) requireRange(fromDate, toDate string) error {
	if err := ssi.RequireNonEmpty(fromDate, "fromDate"); err != nil {
		return err
	}
	return ssi.RequireNonEmpty(toDate, "toDate")
}

// Index methods

func (s *Service) getIndexes(board *Board) ([]MarketIndexes, error) {
	if board != nil {
		b, err := restBoard(*board)
		if err != nil {
			return nil, err
		}
		board = b
	}
	data, err := s.rest.Get(epDataIndexList, (&MarketIndexesRequest{Board: board}).ToMap(), nil)
	if err != nil {
		marketLog.Error("Error fetching market indexes: %v", err)
		return nil, err
	}
	rows, err := rowsOf(data, "indexes")
	if err != nil {
		return nil, err
	}
	return MarketIndexesFromList(rows), nil
}

func (s *Service) GetIndexes() ([]MarketIndexes, error) {
	return s.getIndexes(nil)
}

func (s *Service) GetIndexesByBoard(board Board) ([]MarketIndexes, error) {
	if err := ssi.RequireNonEmpty(string(board), "board"); err != nil {
		return nil, err
	}
	return s.getIndexes(&board)
}

// Index summary methods

func (s *Service) getIndexSummary(index string, board *Board, tradingDate string) ([]MarketIndexSummary, error) {
	if board != nil {
		b, err := restBoard(*board)
		if err != nil {
			return nil, err
		}
		board = b
	}
	params := (&MarketIndexSummaryRequest{Index: index, Board: board, TradingDate: tradingDate}).ToMap()
	data, err := s.rest.Get(epDataIndexSummary, params, nil)
	if err != nil {
		marketLog.Error("Error fetching index summary: %v", err)
		return nil, err
	}
	rows, err := rowsOf(data, "index summary")
	if err != nil {
		return nil, err
	}
	return MarketIndexSummaryFromList(rows), nil
}

func firstSummary(list []MarketIndexSummary) *MarketIndexSummary {
	if len(list) == 0 {
		return nil
	}
	return &list[0]
}

// GetIndexSummary returns the latest summary for an index, or nil when none is found.
func (s *Service) GetIndexSummary(index string) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(index, "index"); err != nil {
		return nil, err
	}
	list, err := s.getIndexSummary(index, nil, "")
	if err != nil {
		return nil, err
	}
	return firstSummary(list), nil
}

// GetIndexSummaryHistorical returns the index summary for a trading date, or nil when none is found.
func (s *Service) GetIndexSummaryHistorical(index, tradingDate string) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(index, "index"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(tradingDate, "tradingDate"); err != nil {
		return nil, err
	}
	list, err := s.getIndexSummary(index, nil, tradingDate)
	if err != nil {
		return nil, err
	}
	return firstSummary(list), nil
}

// GetBoardSummary returns the latest summary for a board, or nil when none is found.
func (s *Service) GetBoardSummary(board Board) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(string(board), "board"); err != nil {
		return nil, err
	}
	list, err := s.getIndexSummary("", &board, "")
	if err != nil {
		return nil, err
	}
	return firstSummary(list), nil
}

// GetBoardSummaryHistorical returns the board summary for a trading date, or nil when none is found.
func (s *Service) GetBoardSummaryHistorical(board Board, tradingDate string) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(string(board), "board"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(tradingDate, "tradingDate"); err != nil {
		return nil, err
	}
	list, err := s.getIndexSummary("", &board, tradingDate)
	if err != nil {
		return nil, err
	}
	return firstSummary(list), nil
}

// Securities info methods

func (s *Service) getSecuritiesInfo(index string, board *Board, symbol string) ([]SecuritiesInfo, error) {
	if board != nil {
		b, err := restBoard(*board)
		if err != nil {
			return nil, err
		}
		board = b
	}
	params := (&SecuritiesInfoRequest{Index: index, Board: board, Symbol: symbol}).ToMap()
	data, err := s.rest.Get(epDataSecuritiesByBoard, params, nil)
	if err != nil {
		marketLog.Error("Error fetching securities information: %v", err)
		return nil, err
	}
	rows, err := rowsOf(data, "securities info")
	if err != nil {
		return nil, err
	}
	return SecuritiesInfoFromList(rows), nil
}

// GetSecuritiesInfo returns the info record for a single security, or nil when none is found.
func (s *Service) GetSecuritiesInfo(symbol string) (*SecuritiesInfo, error) {
	if err := ssi.RequireNonEmpty(symbol, "symbol"); err != nil {
		return nil, err
	}
	infos, err := s.getSecuritiesInfo("", nil, symbol)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, nil
	}
	return &infos[0], nil
}

func (s *Service) GetSecuritiesInfoByIndex(index string) ([]SecuritiesInfo, error) {
	if err := ssi.RequireNonEmpty(index, "index"); err != nil {
		return nil, err
	}
	return s.getSecuritiesInfo(index, nil, "")
}

func (s *Service) GetSecuritiesInfoByBoard(board Board) ([]SecuritiesInfo, error) {
	if err := ssi.RequireNonEmpty(string(board), "board"); err != nil {
		return nil, err
	}
	return s.getSecuritiesInfo("", &board, "")
}

// Securities summary methods

func buildSecuritiesSummaryParams(fromDate, toDate, symbol, index string, page, size int) (map[string]string, error) {
	if err := ssi.RequireExactlyOne(symbol, index, "symbol", "index"); err != nil {
		return nil, err
	}
	if err := ssi.RequireDateRange(fromDate, toDate, false, false, 0); err != nil {
		return nil, err
	}
	if err := checkPaging(page, size); err != nil {
		return nil, err
	}
	return (&SecuritiesSummaryRequest{
		FromDate: fromDate, ToDate: toDate, Symbol: symbol, Index: index, Page: page, Size: size,
	}).ToMap(), nil
}

// getSecuritiesSummary fetches every page (by the response's pagesCount).
func (s *Service) getSecuritiesSummary(fromDate, toDate, symbol, index string) ([]SecuritiesSummary, error) {
	var items []SecuritiesSummary
	pages := 1
	for page := defaultPage; page <= pages; page++ {
		params, err := buildSecuritiesSummaryParams(fromDate, toDate, symbol, index, page, defaultSize)
		if err != nil {
			return nil, err
		}
		data, err := s.rest.Get(epDataSecuritiesSummary, params, nil)
		if err != nil {
			marketLog.Error("Error fetching securities summary: %v", err)
			return nil, err
		}
		rows, err := rowsOf(data, "securities summary")
		if err != nil {
			return nil, err
		}
		more := SecuritiesSummaryFromList(rows)
		if len(more) == 0 {
			break
		}
		items = append(items, more...)
		pages = pagesCount(data)
	}
	return items, nil
}

func (s *Service) GetSecuritiesSummary(symbol string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(symbol, "symbol"); err != nil {
		return nil, err
	}
	today := util.TodayDateStr()
	return s.getSecuritiesSummary(today, today, symbol, "")
}

func (s *Service) GetSecuritiesSummaryHistorical(symbol, fromDate, toDate string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(symbol, "symbol"); err != nil {
		return nil, err
	}
	if err := s.requireRange(fromDate, toDate); err != nil {
		return nil, err
	}
	return s.getSecuritiesSummary(fromDate, toDate, symbol, "")
}

func (s *Service) GetSecuritiesSummaryByIndex(index string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(index, "index"); err != nil {
		return nil, err
	}
	today := util.TodayDateStr()
	return s.getSecuritiesSummary(today, today, "", index)
}

func (s *Service) GetSecuritiesSummaryByIndexHistorical(index, fromDate, toDate string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(index, "index"); err != nil {
		return nil, err
	}
	if err := s.requireRange(fromDate, toDate); err != nil {
		return nil, err
	}
	return s.getSecuritiesSummary(fromDate, toDate, "", index)
}

// Master data

// GetMasterData returns today's reference prices (ceiling/floor/reference) for all symbols.
// The API paginates; every page is fetched and combined.
func (s *Service) GetMasterData() ([]MasterData, error) {
	today := util.TodayDateStr()
	return s.getMasterData(today, today)
}

func (s *Service) GetMasterDataHistorical(fromDate, toDate string) ([]MasterData, error) {
	if err := s.requireRange(fromDate, toDate); err != nil {
		return nil, err
	}
	return s.getMasterData(fromDate, toDate)
}

func (s *Service) getMasterData(fromDate, toDate string) ([]MasterData, error) {
	var items []MasterData
	for page := defaultPage; ; page++ {
		if err := ssi.RequireDateRange(fromDate, toDate, false, false, 0); err != nil {
			return nil, err
		}
		if err := checkPaging(page, defaultSize); err != nil {
			return nil, err
		}
		params := (&MasterDataRequest{FromDate: fromDate, ToDate: toDate, Page: page, Size: defaultSize}).ToMap()
		data, err := s.rest.Get(epDataMasterData, params, nil)
		if err != nil {
			marketLog.Error("Error fetching master data: %v", err)
			return nil, err
		}
		rows, err := rowsOf(data, "master data")
		if err != nil {
			return nil, err
		}
		items = append(items, MasterDataFromList(rows)...)
		if page >= pagesCount(data) {
			break
		}
	}
	return items, nil
}
