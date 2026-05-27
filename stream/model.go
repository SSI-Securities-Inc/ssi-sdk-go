package stream

import (
	"strings"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

// RequestMessage is the message format for subscribing/unsubscribing to streaming channels.
type RequestMessage struct {
	Method  StreamingMethod  `json:"method"`
	Channel StreamingChannel `json:"channel"`
	Topics  []string         `json:"topics"`
}

func (r *RequestMessage) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"method":  string(r.Method),
		"channel": string(r.Channel),
		"topics":  r.Topics,
	}
}

// HeartbeatMessage is the heartbeat message from the server.
type HeartbeatMessage struct {
	Method  StreamingMethod  `json:"method"`
	Channel StreamingChannel `json:"channel"`
	Status  string           `json:"status"`
	Message string           `json:"message"`
}

func HeartbeatMessageFromMap(data map[string]interface{}) *HeartbeatMessage {
	hb := &HeartbeatMessage{
		Status:  util.ToStr(data["status"]),
		Message: util.ToStr(data["message"]),
	}
	if m := util.ToStr(data["method"]); m != "" {
		hb.Method = StreamingMethod(m)
	}
	if c := util.ToStr(data["channel"]); c != "" {
		hb.Channel = StreamingChannel(c)
	}
	return hb
}

// TradeMessage is a real-time trade tick.
type TradeMessage struct {
	Type        DataType `json:"type"`
	TradingTime string   `json:"t"`
	Symbol      string   `json:"s"`
	Price       float64  `json:"p"`
	Quantity    int      `json:"q"`
	Side        string   `json:"si"`
	TotalVolume int      `json:"v"`
}

func TradeMessageFromMap(data map[string]interface{}) *TradeMessage {
	side := util.ToStr(data["si"])
	if side == "" {
		side = "U"
	}
	return &TradeMessage{
		Type:        DataTypeTrade,
		TradingTime: util.ToStr(data["t"]),
		Symbol:      util.ToStr(data["s"]),
		Price:       util.ToFloat64(data["p"]),
		Quantity:    util.ToInt(data["q"]),
		Side:        side,
		TotalVolume: util.ToInt(data["v"]),
	}
}

// IntervalMessage is a real-time trade interval (OHLCV).
type IntervalMessage struct {
	Type         DataType `json:"type"`
	IntervalTime string   `json:"st"`
	TradingTime  string   `json:"t"`
	Symbol       string   `json:"s"`
	Open         float64  `json:"o"`
	High         float64  `json:"h"`
	Low          float64  `json:"l"`
	Close        float64  `json:"c"`
	Volume       int      `json:"v"`
}

func IntervalMessageFromMap(data map[string]interface{}) *IntervalMessage {
	return &IntervalMessage{
		Type:         DataTypeTrade,
		IntervalTime: util.ToStr(data["st"]),
		TradingTime:  util.ToStr(data["t"]),
		Symbol:       util.ToStr(data["s"]),
		Open:         util.ToFloat64(data["o"]),
		High:         util.ToFloat64(data["h"]),
		Low:          util.ToFloat64(data["l"]),
		Close:        util.ToFloat64(data["c"]),
		Volume:       util.ToInt(data["v"]),
	}
}

// QuoteMessage is a real-time bid/ask quote update.
type QuoteMessage struct {
	Type        DataType  `json:"type"`
	TradingTime string    `json:"t"`
	Symbol      string    `json:"s"`
	BidPrices   []float64 `json:"bidPrices"`
	BidVolumes  []int     `json:"bidVolumes"`
	AskPrices   []float64 `json:"askPrices"`
	AskVolumes  []int     `json:"askVolumes"`
}

func QuoteMessageFromMap(data map[string]interface{}) *QuoteMessage {
	qm := &QuoteMessage{
		Type:        DataTypeQuote,
		TradingTime: util.ToStr(data["t"]),
		Symbol:      util.ToStr(data["s"]),
	}
	if bids, ok := data["bids"].([]interface{}); ok {
		for _, b := range bids {
			if pair, ok := b.([]interface{}); ok && len(pair) >= 2 {
				qm.BidPrices = append(qm.BidPrices, util.ToFloat64(pair[0]))
				qm.BidVolumes = append(qm.BidVolumes, util.ToInt(pair[1]))
			}
		}
	}
	if asks, ok := data["asks"].([]interface{}); ok {
		for _, a := range asks {
			if pair, ok := a.([]interface{}); ok && len(pair) >= 2 {
				qm.AskPrices = append(qm.AskPrices, util.ToFloat64(pair[0]))
				qm.AskVolumes = append(qm.AskVolumes, util.ToInt(pair[1]))
			}
		}
	}
	return qm
}

// MarketStatusMessage is a market status update.
type MarketStatusMessage struct {
	Market      string `json:"market"`
	Status      string `json:"status"`
	TradingDate string `json:"tradingDate"`
}

func MarketStatusMessageFromMap(data map[string]interface{}) *MarketStatusMessage {
	return &MarketStatusMessage{
		Market:      util.ToStr(data["market"]),
		Status:      util.ToStr(data["status"]),
		TradingDate: util.ToStr(data["tradingDate"]),
	}
}

// ForeignRoomMessage is foreign room (foreign investor activity) data.
type ForeignRoomMessage struct {
	Type         DataType `json:"type"`
	TradingTime  string   `json:"t"`
	Symbol       string   `json:"s"`
	TotalRoom    int      `json:"tr"`
	CurrentRoom  int      `json:"cr"`
	BuyQuantity  int      `json:"bq"`
	BuyValue     int      `json:"bv"`
	SellQuantity int      `json:"sq"`
	SellValue    int      `json:"sv"`
}

func ForeignRoomMessageFromMap(data map[string]interface{}) *ForeignRoomMessage {
	return &ForeignRoomMessage{
		Type:         DataTypeRoom,
		TradingTime:  util.ToStr(data["t"]),
		Symbol:       util.ToStr(data["s"]),
		TotalRoom:    util.ToInt(data["tr"]),
		CurrentRoom:  util.ToInt(data["cr"]),
		BuyQuantity:  util.ToInt(data["bq"]),
		BuyValue:     util.ToInt(data["bv"]),
		SellQuantity: util.ToInt(data["sq"]),
		SellValue:    util.ToInt(data["sv"]),
	}
}

// PutMessage is put-through (deal) data.
type PutMessage struct {
	Type          DataType `json:"type"`
	TradingTime   string   `json:"t"`
	Symbol        string   `json:"s"`
	Price         float64  `json:"p"`
	Quantity      int      `json:"q"`
	TotalQuantity int      `json:"tq"`
	TotalValue    int      `json:"tv"`
}

func PutMessageFromMap(data map[string]interface{}) *PutMessage {
	return &PutMessage{
		Type:          DataTypePut,
		TradingTime:   util.ToStr(data["t"]),
		Symbol:        util.ToStr(data["s"]),
		Price:         util.ToFloat64(data["p"]),
		Quantity:      util.ToInt(data["q"]),
		TotalQuantity: util.ToInt(data["tq"]),
		TotalValue:    util.ToInt(data["tv"]),
	}
}

// OddLotMessage is odd-lot trade data.
type OddLotMessage struct {
	Type        DataType  `json:"type"`
	TradingTime string    `json:"t"`
	Symbol      string    `json:"s"`
	Price       float64   `json:"p"`
	Quantity    int       `json:"q"`
	BidPrices   []float64 `json:"bidPrices"`
	BidVolumes  []int     `json:"bidVolumes"`
	AskPrices   []float64 `json:"askPrices"`
	AskVolumes  []int     `json:"askVolumes"`
}

func OddLotMessageFromMap(data map[string]interface{}) *OddLotMessage {
	ol := &OddLotMessage{
		Type:        DataTypeOddLot,
		TradingTime: util.ToStr(data["t"]),
		Symbol:      util.ToStr(data["s"]),
		Price:       util.ToFloat64(data["p"]),
		Quantity:    util.ToInt(data["q"]),
	}
	if bids, ok := data["bids"].([]interface{}); ok {
		for _, b := range bids {
			if pair, ok := b.([]interface{}); ok && len(pair) >= 2 {
				ol.BidPrices = append(ol.BidPrices, util.ToFloat64(pair[0]))
				ol.BidVolumes = append(ol.BidVolumes, util.ToInt(pair[1]))
			}
		}
	}
	if asks, ok := data["asks"].([]interface{}); ok {
		for _, a := range asks {
			if pair, ok := a.([]interface{}); ok && len(pair) >= 2 {
				ol.AskPrices = append(ol.AskPrices, util.ToFloat64(pair[0]))
				ol.AskVolumes = append(ol.AskVolumes, util.ToInt(pair[1]))
			}
		}
	}
	return ol
}

// OrderStatusMessage is a real-time order status update.
type OrderStatusMessage struct {
	Type            StreamingType       `json:"type"`
	AccountNo       string              `json:"accountNo"`
	ClientRequestID string              `json:"clientRequestId"`
	OrderID         string              `json:"orderId"`
	Symbol          string              `json:"symbol"`
	Side            trading.OrderSide   `json:"side,omitempty"`
	OrderType       trading.OrderType   `json:"orderType,omitempty"`
	Price           float64             `json:"price"`
	Quantity        int                 `json:"quantity"`
	OSQuantity      int                 `json:"osQty"`
	FilledQuantity  int                 `json:"filledQty"`
	CancelQuantity  int                 `json:"cancelQty"`
	Status          trading.OrderStatus `json:"orderStatus,omitempty"`
	InputTime       string              `json:"inputTime"`
	ModifyTime      string              `json:"modifyTime"`
	Message         string              `json:"rejectReason"`
}

func OrderStatusMessageFromMap(data map[string]interface{}) *OrderStatusMessage {
	osm := &OrderStatusMessage{
		Type:            StreamingTypeOrder,
		AccountNo:       util.ToStr(data["accountNo"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		OrderID:         util.ToStr(data["orderId"]),
		Symbol:          util.ToStr(data["symbol"]),
		Price:           util.ToFloat64(data["price"]),
		Quantity:        util.ToInt(data["quantity"]),
		OSQuantity:      util.ToInt(data["osQty"]),
		FilledQuantity:  util.ToInt(data["filledQty"]),
		CancelQuantity:  util.ToInt(data["cancelQty"]),
		InputTime:       util.ToStr(data["inputTime"]),
		ModifyTime:      util.ToStr(data["modifyTime"]),
		Message:         util.ToStr(data["rejectReason"]),
	}
	if s := util.ToStr(data["side"]); s != "" {
		osm.Side = trading.OrderSide(s)
	}
	if ot := util.ToStr(data["orderType"]); ot != "" {
		osm.OrderType = trading.OrderType(ot)
	}
	if os := util.ToStr(data["orderStatus"]); os != "" {
		osm.Status = trading.OrderStatus(os)
	}
	return osm
}

// PortfolioMessage is a real-time portfolio update.
type PortfolioMessage struct {
	Type        StreamingType `json:"type"`
	AccountNo   string        `json:"accountNo"`
	TotalAsset  float64       `json:"totalAsset"`
	CashBalance float64       `json:"cashBalance"`
	StockValue  float64       `json:"stockValue"`
}

func PortfolioMessageFromMap(data map[string]interface{}) *PortfolioMessage {
	return &PortfolioMessage{
		Type:        StreamingTypePortfolio,
		AccountNo:   util.ToStr(data["accountNo"]),
		TotalAsset:  util.ToFloat64(data["totalAsset"]),
		CashBalance: util.ToFloat64(data["cashBalance"]),
		StockValue:  util.ToFloat64(data["stockValue"]),
	}
}

// ParseStreamingDataMessage parses a streaming data message based on topic.
func ParseStreamingDataMessage(topic string, data map[string]interface{}) interface{} {
	switch {
	case strings.HasPrefix(topic, string(DataTopicTrade)):
		if _, ok := data["st"]; ok {
			return IntervalMessageFromMap(data)
		}
		return TradeMessageFromMap(data)
	case strings.HasPrefix(topic, string(DataTopicQuote)):
		return QuoteMessageFromMap(data)
	case strings.HasPrefix(topic, string(DataTopicRoom)):
		return ForeignRoomMessageFromMap(data)
	case strings.HasPrefix(topic, string(DataTopicMarket)):
		return MarketStatusMessageFromMap(data)
	case strings.HasPrefix(topic, string(DataTopicPut)):
		return PutMessageFromMap(data)
	case strings.HasPrefix(topic, string(DataTopicOddLot)):
		return OddLotMessageFromMap(data)
	default:
		return data
	}
}

// ParseStreamingTradingMessage parses a streaming trading message based on topic.
func ParseStreamingTradingMessage(topic string, data map[string]interface{}) interface{} {
	switch {
	case strings.HasPrefix(topic, "order."):
		return OrderStatusMessageFromMap(data)
	case strings.HasPrefix(topic, "portfolio."):
		return PortfolioMessageFromMap(data)
	default:
		return data
	}
}
