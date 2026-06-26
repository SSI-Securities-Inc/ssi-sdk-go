package market

import (
	"fmt"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

// DownloadDataRequest is a bulk download data request.
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
type DownloadData struct {
	Data       []map[string]interface{} `json:"data"`
	TotalCount int                      `json:"totalCount"`
}

func DownloadDataFromMap(data map[string]interface{}) *DownloadData {
	d := &DownloadData{}
	if items, ok := data["data"].([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				d.Data = append(d.Data, m)
			}
		}
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
	Symbol      string  `json:"symbol"`
	TradingDate string  `json:"tradingDate"`
	OpenPrice   float64 `json:"open"`
	HighPrice   float64 `json:"high"`
	LowPrice    float64 `json:"low"`
	ClosePrice  float64 `json:"close"`
	Volume      int     `json:"volume"`
	Value       float64 `json:"value"`
}

func OHLCDataFromList(data []interface{}) []OHLCData {
	var result []OHLCData
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
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
	return map[string]string{
		"board": string(*r.Board),
	}
}

// MarketIndexes is market indices information.
type MarketIndexes struct {
	Index     string `json:"index"`
	IndexName string `json:"indexName"`
	Board     *Board `json:"board,omitempty"`
}

func MarketIndexesFromList(data []interface{}) []MarketIndexes {
	var result []MarketIndexes
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		mi := MarketIndexes{
			Index:     util.ToStr(m["index"]),
			IndexName: util.ToStr(m["indexName"]),
		}
		if boardStr := util.ToStr(m["board"]); boardStr != "" {
			b := Board(boardStr)
			mi.Board = &b
		}
		result = append(result, mi)
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

// MarketIndexSummary is market index summary information.
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
}

func MarketIndexSummaryFromList(data []interface{}) []MarketIndexSummary {
	var result []MarketIndexSummary
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
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

// SecuritiesInfo is securities information.
type SecuritiesInfo struct {
	Symbol             string  `json:"symbol"`
	Board              *Board  `json:"board,omitempty"`
	Index              string  `json:"index,omitempty"`
	SymbolNameVi       string  `json:"symbolNameVi,omitempty"`
	SymbolNameEn       string  `json:"symbolNameEn,omitempty"`
	LotSize            int     `json:"lotSize,omitempty"`
	MaturityDate       string  `json:"maturityDate,omitempty"`
	FirstTradingDate   string  `json:"firstTradingDate,omitempty"`
	LastTradingDate    string  `json:"lastTradingDate,omitempty"`
	CWUnderlyingSymbol string  `json:"cwUnderlyingSymbol,omitempty"`
	CWExercisePrice    float64 `json:"cwExercisePrice,omitempty"`
	CWExecutionRatio   float64 `json:"cwExecutionRatio,omitempty"`
	ListedShares       int     `json:"listedShare,omitempty"`
	ICBCode            string  `json:"icbCode,omitempty"`
	ICBName            string  `json:"icbName,omitempty"`
	IIndex             float64 `json:"iIndex,omitempty"`
	INAV               float64 `json:"iNav,omitempty"`
}

func SecuritiesInfoFromList(data []interface{}) []SecuritiesInfo {
	var result []SecuritiesInfo
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		si := SecuritiesInfo{
			Symbol:             util.ToStr(m["symbol"]),
			Index:              util.ToStr(m["index"]),
			SymbolNameVi:       util.ToStr(m["symbolNameVi"]),
			SymbolNameEn:       util.ToStr(m["symbolNameEn"]),
			LotSize:            util.ToInt(m["lotSize"]),
			MaturityDate:       util.ToStr(m["maturityDate"]),
			FirstTradingDate:   util.ToStr(m["firstTradingDate"]),
			LastTradingDate:    util.ToStr(m["lastTradingDate"]),
			CWUnderlyingSymbol: util.ToStr(m["cwUnderlyingSymbol"]),
			CWExercisePrice:    util.ToFloat64(m["cwExercisePrice"]),
			CWExecutionRatio:   util.ToFloat64(m["cwExecutionRatio"]),
			ListedShares:       util.ToInt(m["listedShare"]),
			ICBCode:            util.ToStr(m["icbCode"]),
			ICBName:            util.ToStr(m["icbName"]),
			IIndex:             util.ToFloat64(m["iIndex"]),
			INAV:               util.ToFloat64(m["iNav"]),
		}
		if boardStr := util.ToStr(m["board"]); boardStr != "" {
			b := Board(boardStr)
			si.Board = &b
		}
		result = append(result, si)
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

// SecuritiesSummary is securities summary information.
type SecuritiesSummary struct {
	Symbol             string  `json:"symbol"`
	TradingDate        string  `json:"tradingDate"`
	PriceChange        float64 `json:"priceChange"`
	PriceChangePercent float64 `json:"priceChangePercentage"`
	OpenPrice          float64 `json:"open"`
	HighPrice          float64 `json:"high"`
	LowPrice           float64 `json:"low"`
	ClosePrice         float64 `json:"close"`
	AveragePrice       float64 `json:"average"`
	TotalMatch         int     `json:"totalMatch"`
	TotalMatchValue    float64 `json:"totalMatchValue"`
	TotalBuy           int     `json:"totalBuy"`
	TotalTradeBuy      float64 `json:"totalTradeBuy"`
	TotalSell          int     `json:"totalSell"`
	TotalTradeSell     float64 `json:"totalTradeSell"`
}

func SecuritiesSummaryFromList(data []interface{}) []SecuritiesSummary {
	var result []SecuritiesSummary
	for _, item := range data {
		m, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, SecuritiesSummary{
			Symbol:             util.ToStr(m["symbol"]),
			TradingDate:        util.ToStr(m["tradingDate"]),
			PriceChange:        util.ToFloat64(m["priceChange"]),
			PriceChangePercent: util.ToFloat64(m["priceChangePercentage"]),
			OpenPrice:          util.ToFloat64(m["open"]),
			HighPrice:          util.ToFloat64(m["high"]),
			LowPrice:           util.ToFloat64(m["low"]),
			ClosePrice:         util.ToFloat64(m["close"]),
			AveragePrice:       util.ToFloat64(m["average"]),
			TotalMatch:         util.ToInt(m["totalMatch"]),
			TotalMatchValue:    util.ToFloat64(m["totalMatchValue"]),
			TotalBuy:           util.ToInt(m["totalBuy"]),
			TotalTradeBuy:      util.ToFloat64(m["totalTradeBuy"]),
			TotalSell:          util.ToInt(m["totalSell"]),
			TotalTradeSell:     util.ToFloat64(m["totalTradeSell"]),
		})
	}
	return result
}
