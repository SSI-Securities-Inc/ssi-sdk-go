package market

import (
	"fmt"
	"strings"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

// normalizeBoard parses a board leniently: a known value becomes its Board, an unknown one is
// kept as the raw string. ok is false for an empty value.
func normalizeBoard(raw interface{}) (Board, bool) {
	text := strings.TrimSpace(util.ToStr(raw))
	if text == "" {
		return "", false
	}
	if b, ok := BoardFromValue(text); ok {
		return b, true
	}
	return Board(text), true
}

func boardPtr(raw interface{}) *Board {
	if b, ok := normalizeBoard(raw); ok {
		return &b
	}
	return nil
}

// blankToNil maps the server's empty-string "no value" marker (and a missing value) to nil.
func blankToNil(v interface{}) *string {
	if v == nil {
		return nil
	}
	s := util.ToStr(v)
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return &s
}

func rowMaps(data []interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(data))
	for _, item := range data {
		if m, ok := item.(map[string]interface{}); ok {
			out = append(out, m)
		}
	}
	return out
}

// DownloadDataRequest is a bulk download data request.
//
// Deprecated: the server has no data/file endpoint yet and its final response shape is
// undecided, so nothing uses this model.
type DownloadDataRequest struct {
	Symbol    string    `json:"symbol"`
	Timeframe Timeframe `json:"timeFrame"`
}

func (r *DownloadDataRequest) ToMap() map[string]string {
	return map[string]string{
		"symbol":    r.Symbol,
		"timeFrame": string(r.Timeframe),
	}
}

// DownloadData is a bulk download data result.
//
// Deprecated: see DownloadDataRequest.
type DownloadData struct {
	Data       []map[string]interface{} `json:"data"`
	TotalCount int                      `json:"totalCount"`
}

func DownloadDataFromMap(data map[string]interface{}) *DownloadData {
	d := &DownloadData{}
	if items, ok := data["data"].([]interface{}); ok {
		d.Data = rowMaps(items)
		d.TotalCount = len(d.Data)
	}
	if v, ok := data["totalCount"]; ok {
		d.TotalCount = util.ToInt(v)
	}
	return d
}

// OHLCRequest is an OHLC data request.
type OHLCRequest struct {
	Symbol    string    `json:"symbol"`
	FromDate  string    `json:"from"`
	ToDate    string    `json:"to"`
	Timeframe Timeframe `json:"timeFrame"`
	Page      int       `json:"pageIndex"`
	Size      int       `json:"pageSize"`
}

func (r *OHLCRequest) ToMap() map[string]string {
	return map[string]string{
		"symbol":    r.Symbol,
		"from":      r.FromDate,
		"to":        r.ToDate,
		"timeFrame": string(r.Timeframe),
		"pageIndex": fmt.Sprintf("%d", r.Page),
		"pageSize":  fmt.Sprintf("%d", r.Size),
	}
}

// OHLCData is an OHLC data result.
type OHLCData struct {
	Symbol string `json:"symbol"`
	// TradingDate is the date as the server formats it (varies by endpoint).
	TradingDate string  `json:"tradingDate"`
	OpenPrice   float64 `json:"open"`
	HighPrice   float64 `json:"high"`
	LowPrice    float64 `json:"low"`
	ClosePrice  float64 `json:"close"`
	Volume      int     `json:"volume"`
	Value       float64 `json:"value"`
}

func OHLCDataFromList(data []interface{}) []OHLCData {
	result := make([]OHLCData, 0, len(data))
	for _, m := range rowMaps(data) {
		result = append(result, OHLCData{
			Symbol:      util.ToStr(m["symbol"]),
			TradingDate: util.ToStr(m["tradingDate"]),
			OpenPrice:   util.ToFloat64(m["open"]),
			HighPrice:   util.ToFloat64(m["high"]),
			LowPrice:    util.ToFloat64(m["low"]),
			ClosePrice:  util.ToFloat64(m["close"]),
			Volume:      util.ToInt(m["volume"]),
			Value:       util.ToFloat64(m["value"]),
		})
	}
	return result
}

// MarketIndexesRequest is a market index information request.
type MarketIndexesRequest struct {
	Board *Board `json:"board,omitempty"`
}

func (r *MarketIndexesRequest) ToMap() map[string]string {
	if r.Board == nil {
		return map[string]string{}
	}
	return map[string]string{"board": string(*r.Board)}
}

// MarketIndexes is market indices information.
type MarketIndexes struct {
	Index     string `json:"index"`
	IndexName string `json:"indexName"`
	// Board is the matching board, or the raw string for a board this SDK does not know.
	Board *Board `json:"board,omitempty"`
	// BoardRaw is the board exactly as the server sent it.
	BoardRaw *string `json:"-"`
}

func MarketIndexesFromList(data []interface{}) []MarketIndexes {
	result := make([]MarketIndexes, 0, len(data))
	for _, m := range rowMaps(data) {
		result = append(result, MarketIndexes{
			Index:     util.ToStr(m["index"]),
			IndexName: util.ToStr(m["indexName"]),
			Board:     boardPtr(m["board"]),
			BoardRaw:  blankToNil(m["board"]),
		})
	}
	return result
}

// MarketIndexSummaryRequest is a market index summary request.
type MarketIndexSummaryRequest struct {
	Index       string `json:"index,omitempty"`
	Board       *Board `json:"board,omitempty"`
	TradingDate string `json:"tradingDate,omitempty"`
}

func (r *MarketIndexSummaryRequest) ToMap() map[string]string {
	data := map[string]string{}
	if r.Index != "" {
		data["index"] = r.Index
	}
	if r.Board != nil {
		data["board"] = string(*r.Board)
	}
	if r.TradingDate != "" {
		data["tradingDate"] = r.TradingDate
	}
	return data
}

// MarketIndexSummary is market index summary information. The foreign-trading fields are nil
// when the server omits them.
type MarketIndexSummary struct {
	TradingDate        string  `json:"tradingDate"`
	TotalTrade         int     `json:"totalTrade"`
	TotalTradeValue    float64 `json:"totalTradeValue"`
	TotalMatch         int     `json:"totalMatch"`
	TotalMatchValue    float64 `json:"totalMatchValue"`
	TotalDeal          int     `json:"totalDeal"`
	TotalDealValue     float64 `json:"totalDealValue"`
	IndexChange        float64 `json:"indexChange"`
	IndexChangePercent float64 `json:"indexChangePercentage"`
	IndexValue         float64 `json:"indexValue"`
	TotalAdvanceStock  int     `json:"totalAdvanceStock"`
	TotalDeclineStock  int     `json:"totalDeclineStock"`
	TotalSteadyStock   int     `json:"totalNoChangeStock"`
	TotalCeilingStock  int     `json:"totalCeilingStock"`
	TotalFloorStock    int     `json:"totalFloorStock"`
	TotalPropBuy       int     `json:"totalPropBuy"`
	TotalPropBuyValue  float64 `json:"totalPropBuyValue"`
	TotalPropSell      int     `json:"totalPropSell"`
	TotalPropSellValue float64 `json:"totalPropSellValue"`

	TotalBuyForeign           *float64 `json:"totalBuyForeign,omitempty"`
	TotalBuyForeignValue      *float64 `json:"totalBuyForeignValue,omitempty"`
	TotalSellForeign          *float64 `json:"totalSellForeign,omitempty"`
	TotalSellForeignValue     *float64 `json:"totalSellForeignValue,omitempty"`
	NetPurchasesForeignVolume *float64 `json:"netPurchasesForeignVolume,omitempty"`
	NetPurchasesForeignValue  *float64 `json:"netPurchasesForeignValue,omitempty"`
}

func MarketIndexSummaryFromList(data []interface{}) []MarketIndexSummary {
	result := make([]MarketIndexSummary, 0, len(data))
	for _, m := range rowMaps(data) {
		result = append(result, MarketIndexSummary{
			TradingDate:        util.ToStr(m["tradingDate"]),
			TotalTrade:         util.ToInt(m["totalTrade"]),
			TotalTradeValue:    util.ToFloat64(m["totalTradeValue"]),
			TotalMatch:         util.ToInt(m["totalMatch"]),
			TotalMatchValue:    util.ToFloat64(m["totalMatchValue"]),
			TotalDeal:          util.ToInt(m["totalDeal"]),
			TotalDealValue:     util.ToFloat64(m["totalDealValue"]),
			IndexChange:        util.ToFloat64(m["indexChange"]),
			IndexChangePercent: util.ToFloat64(m["indexChangePercentage"]),
			IndexValue:         util.ToFloat64(m["indexValue"]),
			TotalAdvanceStock:  util.ToInt(m["totalAdvanceStock"]),
			TotalDeclineStock:  util.ToInt(m["totalDeclineStock"]),
			TotalSteadyStock:   util.ToInt(m["totalNoChangeStock"]),
			TotalCeilingStock:  util.ToInt(m["totalCeilingStock"]),
			TotalFloorStock:    util.ToInt(m["totalFloorStock"]),
			TotalPropBuy:       util.ToInt(m["totalPropBuy"]),
			TotalPropBuyValue:  util.ToFloat64(m["totalPropBuyValue"]),
			TotalPropSell:      util.ToInt(m["totalPropSell"]),
			TotalPropSellValue: util.ToFloat64(m["totalPropSellValue"]),

			TotalBuyForeign:           util.ToOptFloat64(m["totalBuyForeign"]),
			TotalBuyForeignValue:      util.ToOptFloat64(m["totalBuyForeignValue"]),
			TotalSellForeign:          util.ToOptFloat64(m["totalSellForeign"]),
			TotalSellForeignValue:     util.ToOptFloat64(m["totalSellForeignValue"]),
			NetPurchasesForeignVolume: util.ToOptFloat64(m["netPurchasesForeignVolume"]),
			NetPurchasesForeignValue:  util.ToOptFloat64(m["netPurchasesForeignValue"]),
		})
	}
	return result
}

// SecuritiesInfoRequest is a securities information request.
type SecuritiesInfoRequest struct {
	Symbol string `json:"symbol,omitempty"`
	Board  *Board `json:"board,omitempty"`
	Index  string `json:"index,omitempty"`
}

func (r *SecuritiesInfoRequest) ToMap() map[string]string {
	data := map[string]string{}
	if r.Symbol != "" {
		data["symbol"] = r.Symbol
	}
	if r.Board != nil {
		data["board"] = string(*r.Board)
	}
	if r.Index != "" {
		data["index"] = r.Index
	}
	return data
}

// SecuritiesInfo is securities information. A numeric field the server omits or leaves empty
// is nil (not 0), and an empty text/date field is nil.
type SecuritiesInfo struct {
	Symbol             string   `json:"symbol"`
	Board              *Board   `json:"board,omitempty"`
	Index              string   `json:"index,omitempty"`
	SymbolNameVi       string   `json:"symbolNameVi,omitempty"`
	SymbolNameEn       string   `json:"symbolNameEn,omitempty"`
	LotSize            *int     `json:"lotSize,omitempty"`
	MaturityDate       *string  `json:"maturityDate,omitempty"`
	FirstTradingDate   *string  `json:"firstTradingDate,omitempty"`
	LastTradingDate    *string  `json:"lastTradingDate,omitempty"`
	CWUnderlyingSymbol *string  `json:"cwUnderlyingSymbol,omitempty"`
	CWExercisePrice    *float64 `json:"cwExercisePrice,omitempty"`
	CWExecutionRatio   *float64 `json:"cwExecutionRatio,omitempty"`
	ListedShares       *int     `json:"listedShare,omitempty"`
	ICBCode            *string  `json:"icbCode,omitempty"`
	ICBName            *string  `json:"icbName,omitempty"`
	IIndex             *float64 `json:"iIndex,omitempty"`
	INAV               *float64 `json:"iNav,omitempty"`
	OpenInterest       *float64 `json:"openInterest,omitempty"`
	SettlementPrice    *float64 `json:"settlementPrice,omitempty"`
	StockType          *string  `json:"stockType,omitempty"`
}

func SecuritiesInfoFromList(data []interface{}) []SecuritiesInfo {
	result := make([]SecuritiesInfo, 0, len(data))
	for _, m := range rowMaps(data) {
		result = append(result, SecuritiesInfo{
			Symbol:             util.ToStr(m["symbol"]),
			Board:              boardPtr(m["board"]),
			Index:              util.ToStr(m["index"]),
			SymbolNameVi:       util.ToStr(m["symbolNameVi"]),
			SymbolNameEn:       util.ToStr(m["symbolNameEn"]),
			LotSize:            util.ToOptInt(m["lotSize"]),
			MaturityDate:       blankToNil(m["maturityDate"]),
			FirstTradingDate:   blankToNil(m["firstTradingDate"]),
			LastTradingDate:    blankToNil(m["lastTradingDate"]),
			CWUnderlyingSymbol: blankToNil(m["cwUnderlyingSymbol"]),
			CWExercisePrice:    util.ToOptFloat64(m["cwExercisePrice"]),
			CWExecutionRatio:   util.ToOptFloat64(m["cwExecutionRatio"]),
			ListedShares:       util.ToOptInt(m["listedShare"]),
			ICBCode:            blankToNil(m["icbCode"]),
			ICBName:            blankToNil(m["icbName"]),
			IIndex:             util.ToOptFloat64(m["iIndex"]),
			INAV:               util.ToOptFloat64(m["iNav"]),
			OpenInterest:       util.ToOptFloat64(m["openInterest"]),
			SettlementPrice:    util.ToOptFloat64(m["settlementPrice"]),
			StockType:          blankToNil(m["stockType"]),
		})
	}
	return result
}

// SecuritiesSummaryRequest is a securities summary request.
type SecuritiesSummaryRequest struct {
	FromDate string `json:"from"`
	ToDate   string `json:"to"`
	Symbol   string `json:"symbol,omitempty"`
	Index    string `json:"index,omitempty"`
	Page     int    `json:"pageIndex"`
	Size     int    `json:"pageSize"`
}

func (r *SecuritiesSummaryRequest) ToMap() map[string]string {
	data := map[string]string{
		"from":      r.FromDate,
		"to":        r.ToDate,
		"pageIndex": fmt.Sprintf("%d", r.Page),
		"pageSize":  fmt.Sprintf("%d", r.Size),
	}
	if r.Symbol != "" {
		data["symbol"] = r.Symbol
	}
	if r.Index != "" {
		data["index"] = r.Index
	}
	return data
}

// SecuritiesSummary is securities summary information. Ceiling, Floor and RefPrice are nil
// unless the server sends them (securitiesSummary does not: use GetMasterData).
type SecuritiesSummary struct {
	Symbol                string   `json:"symbol"`
	TradingDate           string   `json:"tradingDate"`
	PriceChange           float64  `json:"priceChange"`
	PriceChangePercent    float64  `json:"priceChangePercentage"`
	OpenPrice             float64  `json:"open"`
	HighPrice             float64  `json:"high"`
	LowPrice              float64  `json:"low"`
	ClosePrice            float64  `json:"close"`
	AveragePrice          float64  `json:"average"`
	TotalMatch            int      `json:"totalMatch"`
	TotalMatchValue       float64  `json:"totalMatchValue"`
	TotalBuy              int      `json:"totalBuy"`
	TotalTradeBuy         float64  `json:"totalTradeBuy"`
	TotalSell             int      `json:"totalSell"`
	TotalTradeSell        float64  `json:"totalTradeSell"`
	TotalForeignBuy       int      `json:"totalForeignBuy"`
	TotalForeignBuyValue  float64  `json:"totalForeignBuyValue"`
	TotalForeignSell      int      `json:"totalForeignSell"`
	TotalForeignSellValue float64  `json:"totalForeignSellValue"`
	RemainForeignRoom     int      `json:"remainForeignRoom"`
	TotalForeignRoom      int      `json:"totalForeignRoom"`
	TotalDeal             int      `json:"totalDeal"`
	TotalDealValue        float64  `json:"totalDealValue"`
	OpenInterest          float64  `json:"openInterest"`
	SettlementPrice       float64  `json:"settlementPrice"`
	Ceiling               *float64 `json:"ceiling,omitempty"`
	Floor                 *float64 `json:"floor,omitempty"`
	RefPrice              *float64 `json:"refPrice,omitempty"`
}

func SecuritiesSummaryFromList(data []interface{}) []SecuritiesSummary {
	result := make([]SecuritiesSummary, 0, len(data))
	for _, m := range rowMaps(data) {
		result = append(result, SecuritiesSummary{
			Symbol:                util.ToStr(m["symbol"]),
			TradingDate:           util.ToStr(m["tradingDate"]),
			PriceChange:           util.ToFloat64(m["priceChange"]),
			PriceChangePercent:    util.ToFloat64(m["priceChangePercentage"]),
			OpenPrice:             util.ToFloat64(m["open"]),
			HighPrice:             util.ToFloat64(m["high"]),
			LowPrice:              util.ToFloat64(m["low"]),
			ClosePrice:            util.ToFloat64(m["close"]),
			AveragePrice:          util.ToFloat64(m["average"]),
			TotalMatch:            util.ToInt(m["totalMatch"]),
			TotalMatchValue:       util.ToFloat64(m["totalMatchValue"]),
			TotalBuy:              util.ToInt(m["totalBuy"]),
			TotalTradeBuy:         util.ToFloat64(m["totalTradeBuy"]),
			TotalSell:             util.ToInt(m["totalSell"]),
			TotalTradeSell:        util.ToFloat64(m["totalTradeSell"]),
			TotalForeignBuy:       util.ToInt(m["totalForeignBuy"]),
			TotalForeignBuyValue:  util.ToFloat64(m["totalForeignBuyValue"]),
			TotalForeignSell:      util.ToInt(m["totalForeignSell"]),
			TotalForeignSellValue: util.ToFloat64(m["totalForeignSellValue"]),
			RemainForeignRoom:     util.ToInt(m["remainForeignRoom"]),
			TotalForeignRoom:      util.ToInt(m["totalForeignRoom"]),
			TotalDeal:             util.ToInt(m["totalDeal"]),
			TotalDealValue:        util.ToFloat64(m["totalDealValue"]),
			OpenInterest:          util.ToFloat64(m["openInterest"]),
			SettlementPrice:       util.ToFloat64(m["settlementPrice"]),
			Ceiling:               util.ToOptFloat64(m["ceiling"]),
			Floor:                 util.ToOptFloat64(m["floor"]),
			RefPrice:              util.ToOptFloat64(m["refPrice"]),
		})
	}
	return result
}

// MasterDataRequest is a master data (reference price) request.
type MasterDataRequest struct {
	FromDate string
	ToDate   string
	Page     int
	Size     int
}

// ToMap uses the endpoint's lower-case from/to keys; only set bounds are sent.
func (r *MasterDataRequest) ToMap() map[string]string {
	data := map[string]string{}
	if r.FromDate != "" {
		data["from"] = r.FromDate
	}
	if r.ToDate != "" {
		data["to"] = r.ToDate
	}
	data["pageIndex"] = fmt.Sprintf("%d", r.Page)
	data["pageSize"] = fmt.Sprintf("%d", r.Size)
	return data
}

// MasterData represents ceiling, floor, and reference prices for a symbol on a trading date.
type MasterData struct {
	// Board is the matching board, or the raw string for a board this SDK does not know.
	Board       Board   `json:"board"`
	Symbol      string  `json:"symbol"`
	TradingDate string  `json:"tradingDate"`
	Ceiling     float64 `json:"ceiling"`
	Floor       float64 `json:"floor"`
	RefPrice    float64 `json:"refPrice"`
}

func MasterDataFromList(data []interface{}) []MasterData {
	result := make([]MasterData, 0, len(data))
	for _, m := range rowMaps(data) {
		board, _ := normalizeBoard(m["board"])
		result = append(result, MasterData{
			Board:       board,
			Symbol:      util.ToStr(m["symbol"]),
			TradingDate: util.ToStr(m["tradingDate"]),
			Ceiling:     util.ToFloat64(m["ceiling"]),
			Floor:       util.ToFloat64(m["floor"]),
			RefPrice:    util.ToFloat64(util.Pick(m, "refPrice", "referencePrice")),
		})
	}
	return result
}

// CorporateActionsRequest is a corporate actions request. A nil Symbol asks for every security.
type CorporateActionsRequest struct {
	FromDate string
	ToDate   string
	Symbol   *string
	Language string
	Page     int
	Size     int
}

// ToMap uses the endpoint's query keys; symbol is sent only when set.
func (r *CorporateActionsRequest) ToMap() map[string]string {
	data := map[string]string{
		"from":      r.FromDate,
		"to":        r.ToDate,
		"language":  r.Language,
		"pageIndex": fmt.Sprintf("%d", r.Page),
		"pageSize":  fmt.Sprintf("%d", r.Size),
	}
	if r.Symbol != nil {
		data["symbol"] = *r.Symbol
	}
	return data
}

// CorporateAction is one corporate action (dividend, bonus shares, rights issue, ...) of a
// security. Every field is the string the server sends: dates stay YYYY/MM/DD text and Ratio is
// not parsed (its meaning depends on the event type). A field the server leaves blank (a newly
// announced event has no ex-right date or ratio yet), null or omits is "".
type CorporateAction struct {
	Symbol      string `json:"symbol"`
	EventTitle  string `json:"eventTitle"`
	ExrightDate string `json:"exrightDate"`
	RecordDate  string `json:"recordDate"`
	IssueDate   string `json:"issueDate"`
	Exchange    string `json:"exchange"`
	Ratio       string `json:"ratio"`
}

func CorporateActionsFromList(data []interface{}) []CorporateAction {
	result := make([]CorporateAction, 0, len(data))
	for _, m := range rowMaps(data) {
		result = append(result, CorporateAction{
			Symbol:      util.ToStr(m["symbol"]),
			EventTitle:  util.ToStr(m["eventTitle"]),
			ExrightDate: util.ToStr(m["exrightDate"]),
			RecordDate:  util.ToStr(m["recordDate"]),
			IssueDate:   util.ToStr(m["issueDate"]),
			Exchange:    util.ToStr(m["exchange"]),
			Ratio:       util.ToStr(m["ratio"]),
		})
	}
	return result
}
