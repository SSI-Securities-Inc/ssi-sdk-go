package market

import (
	"fmt"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var marketLog = logger.New("ssi_sdk.services.market")

const (
	defaultSize = 1000
	defaultPage = 1

	epDataOHLC              = "/api/v3/data/ohlc"
	epDataIndexList         = "/api/v3/data/indexList"
	epDataIndexSummary      = "/api/v3/data/indexSummary"
	epDataSecuritiesByBoard = "/api/v3/data/securitiesByBoard"
	epDataSecuritiesSummary = "/api/v3/data/securitiesSummary"
)

// Service provides market data operations.
type Service struct {
	rest *transport.RestClient
}

func NewService(rest *transport.RestClient) *Service {
	return &Service{rest: rest}
}

func (s *Service) getOHLC(symbol, fromDate, toDate string, timeframe Timeframe, page, size int) ([]OHLCData, error) {
	params := (&OHLCRequest{
		Symbol:    symbol,
		FromDate:  fromDate,
		ToDate:    toDate,
		Timeframe: timeframe,
		Page:      page,
		Size:      size,
	}).ToMap()

	data, err := s.rest.Get(epDataOHLC, params, nil)
	if err != nil {
		marketLog.Error("Error fetching OHLC data for symbol %s, timeframe %s: %v", symbol, string(timeframe), err)
		return nil, err
	}

	dataList, _ := data["data"].([]interface{})
	return OHLCDataFromList(dataList), nil
}

func (s *Service) GetOHLC1Minute(symbol string) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, util.BeginningOfDay(), util.EndOfDay(), TimeframeMinute1, defaultPage, defaultSize)
}

func (s *Service) GetOHLC1MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeMinute1, page, size)
}

func (s *Service) GetOHLC3Minute(symbol string) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, util.BeginningOfDay(), util.EndOfDay(), TimeframeMinute3, defaultPage, defaultSize)
}

func (s *Service) GetOHLC3MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeMinute3, page, size)
}

func (s *Service) GetOHLC5Minute(symbol string) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, util.BeginningOfDay(), util.EndOfDay(), TimeframeMinute5, defaultPage, defaultSize)
}

func (s *Service) GetOHLC5MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeMinute5, page, size)
}

func (s *Service) GetOHLC15Minute(symbol string) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, util.BeginningOfDay(), util.EndOfDay(), TimeframeMinute15, defaultPage, defaultSize)
}

func (s *Service) GetOHLC15MinuteHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeMinute15, page, size)
}

func (s *Service) GetOHLC1Hour(symbol string) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, util.BeginningOfDay(), util.EndOfDay(), TimeframeHour1, defaultPage, defaultSize)
}

func (s *Service) GetOHLC1HourHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeHour1, page, size)
}

func (s *Service) GetOHLC1DayHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeDay1, page, size)
}

func (s *Service) GetOHLC1WeekHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeWeek1, page, size)
}

func (s *Service) GetOHLC1MonthHistorical(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getOHLC(symbol, fromDate, toDate, TimeframeMonth1, page, size)
}

// Index methods

func (s *Service) getIndexes(board *Board) ([]MarketIndexes, error) {
	params := (&MarketIndexesRequest{Board: board}).ToMap()
	data, err := s.rest.Get(epDataIndexList, params, nil)
	if err != nil {
		marketLog.Error("Error fetching market indexes: %v", err)
		return nil, err
	}

	dataList, _ := data["data"].([]interface{})
	if dataList == nil {
		return nil, nil
	}
	return MarketIndexesFromList(dataList), nil
}

func (s *Service) GetIndexes() ([]MarketIndexes, error) {
	return s.getIndexes(nil)
}

func (s *Service) GetIndexesByBoard(board Board) ([]MarketIndexes, error) {
	if err := ssi.RequireNonEmpty(string(board), "Board must be provided"); err != nil {
		return nil, err
	}
	return s.getIndexes(&board)
}

// Index summary methods

func (s *Service) getIndexSummary(index string, board *Board, tradingDate string) ([]MarketIndexSummary, error) {
	params := (&MarketIndexSummaryRequest{
		Index:       index,
		Board:       board,
		TradingDate: tradingDate,
	}).ToMap()

	data, err := s.rest.Get(epDataIndexSummary, params, nil)
	if err != nil {
		marketLog.Error("Error fetching index summary: %v", err)
		return nil, err
	}

	dataList, _ := data["data"].([]interface{})
	if dataList == nil {
		return nil, nil
	}
	return MarketIndexSummaryFromList(dataList), nil
}

func (s *Service) GetIndexSummary(index string) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(index, "Index code must be provided"); err != nil {
		return nil, err
	}
	summaries, err := s.getIndexSummary(index, nil, "")
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, fmt.Errorf("no summary information found for index: %s", index)
	}
	return &summaries[0], nil
}

func (s *Service) GetIndexSummaryHistorical(index, tradingDate string) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(index, "Index code must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(tradingDate, "Trading date must be provided"); err != nil {
		return nil, err
	}
	summaries, err := s.getIndexSummary(index, nil, tradingDate)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, fmt.Errorf("no summary information found for index: %s on trading date: %s", index, tradingDate)
	}
	return &summaries[0], nil
}

func (s *Service) GetBoardSummary(board Board) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(string(board), "Board must be provided"); err != nil {
		return nil, err
	}
	summaries, err := s.getIndexSummary("", &board, "")
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, fmt.Errorf("no summary information found for board: %s", string(board))
	}
	return &summaries[0], nil
}

func (s *Service) GetBoardSummaryHistorical(board Board, tradingDate string) (*MarketIndexSummary, error) {
	if err := ssi.RequireNonEmpty(string(board), "Board must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(tradingDate, "Trading date must be provided"); err != nil {
		return nil, err
	}
	summaries, err := s.getIndexSummary("", &board, tradingDate)
	if err != nil {
		return nil, err
	}
	if len(summaries) == 0 {
		return nil, fmt.Errorf("no summary information found for board: %s on trading date: %s", string(board), tradingDate)
	}
	return &summaries[0], nil
}

// Securities info methods

func (s *Service) getSecuritiesInfo(index string, board *Board, symbol string) ([]SecuritiesInfo, error) {
	params := (&SecuritiesInfoRequest{
		Index:  index,
		Board:  board,
		Symbol: symbol,
	}).ToMap()

	data, err := s.rest.Get(epDataSecuritiesByBoard, params, nil)
	if err != nil {
		marketLog.Error("Error fetching securities information: %v", err)
		return nil, err
	}

	dataList, _ := data["data"].([]interface{})
	if dataList == nil {
		return nil, nil
	}
	return SecuritiesInfoFromList(dataList), nil
}

func (s *Service) GetSecuritiesInfo(symbol string) (*SecuritiesInfo, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	infos, err := s.getSecuritiesInfo("", nil, symbol)
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("no securities information found for symbol: %s", symbol)
	}
	return &infos[0], nil
}

func (s *Service) GetSecuritiesInfoByIndex(index string) ([]SecuritiesInfo, error) {
	if err := ssi.RequireNonEmpty(index, "Index code must be provided"); err != nil {
		return nil, err
	}
	infos, err := s.getSecuritiesInfo(index, nil, "")
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("no securities information found for index: %s", index)
	}
	return infos, nil
}

func (s *Service) GetSecuritiesInfoByBoard(board Board) ([]SecuritiesInfo, error) {
	if err := ssi.RequireNonEmpty(string(board), "Board must be provided"); err != nil {
		return nil, err
	}
	infos, err := s.getSecuritiesInfo("", &board, "")
	if err != nil {
		return nil, err
	}
	if len(infos) == 0 {
		return nil, fmt.Errorf("no securities information found for board: %s", string(board))
	}
	return infos, nil
}

// Securities summary methods

func (s *Service) getSecuritiesSummary(fromDate, toDate, symbol, index string, page, size int) ([]SecuritiesSummary, error) {
	params := (&SecuritiesSummaryRequest{
		FromDate: fromDate,
		ToDate:   toDate,
		Symbol:   symbol,
		Index:    index,
		Page:     page,
		Size:     size,
	}).ToMap()

	data, err := s.rest.Get(epDataSecuritiesSummary, params, nil)
	if err != nil {
		marketLog.Error("Error fetching securities summary: %v", err)
		return nil, err
	}

	dataList, _ := data["data"].([]interface{})
	return SecuritiesSummaryFromList(dataList), nil
}

func (s *Service) GetSecuritiesSummary(symbol string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	today := util.TodayDateStr()
	return s.getSecuritiesSummary(today, today, symbol, "", defaultPage, defaultSize)
}

func (s *Service) GetSecuritiesSummaryHistorical(symbol, fromDate, toDate string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getSecuritiesSummary(fromDate, toDate, symbol, "", defaultPage, defaultSize)
}

func (s *Service) GetSecuritiesSummaryByIndex(index string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(index, "Index code must be provided"); err != nil {
		return nil, err
	}
	today := util.TodayDateStr()
	return s.getSecuritiesSummary(today, today, "", index, defaultPage, defaultSize)
}

func (s *Service) GetSecuritiesSummaryByIndexHistorical(index, fromDate, toDate string) ([]SecuritiesSummary, error) {
	if err := ssi.RequireNonEmpty(index, "Index code must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(fromDate, "From date must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(toDate, "To date must be provided"); err != nil {
		return nil, err
	}
	return s.getSecuritiesSummary(fromDate, toDate, "", index, defaultPage, defaultSize)
}
