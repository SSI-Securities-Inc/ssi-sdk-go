package stream

import (
	"strconv"
	"strings"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

// RequestMessage is the message format for subscribing/unsubscribing to streaming channels.
type RequestMessage struct {
	Method  StreamingMethod
	Channel StreamingChannel
	Topics  []string
}

func (r *RequestMessage) ToMap() map[string]interface{} {
	topics := r.Topics
	if topics == nil {
		topics = []string{}
	}
	return map[string]interface{}{
		"method":  string(r.Method),
		"channel": string(r.Channel),
		"topics":  topics,
	}
}

// HeartbeatMessage is the heartbeat message from the server. An unknown method or channel is
// kept as its raw string.
type HeartbeatMessage struct {
	Method  StreamingMethod
	Channel StreamingChannel
	Status  string
	Message string
}

// HeartbeatMessageFromMap never fails: a missing key yields an empty field.
func HeartbeatMessageFromMap(data map[string]interface{}) *HeartbeatMessage {
	return &HeartbeatMessage{
		Method:  StreamingMethod(strings.ToLower(util.ToStr(data["method"]))),
		Channel: StreamingChannel(strings.ToUpper(util.ToStr(data["channel"]))),
		Status:  util.ToStr(data["status"]),
		Message: util.ToStr(data["message"]),
	}
}

// levels splits [[price, volume], ...] into parallel slices, skipping malformed entries. The
// server's order (best first) is kept.
func levels(items interface{}) ([]float64, []int) {
	prices := []float64{}
	volumes := []int{}
	list, ok := items.([]interface{})
	if !ok {
		return prices, volumes
	}
	for _, item := range list {
		if pair, ok := item.([]interface{}); ok && len(pair) >= 2 {
			prices = append(prices, util.ToFloat64(pair[0]))
			volumes = append(volumes, util.ToInt(pair[1]))
		}
	}
	return prices, volumes
}

// parsePrice returns the numeric price, or 0 plus the word the server sent for a market-priced
// order (ATO/ATC/MP).
func parsePrice(v interface{}) (float64, string) {
	if v == nil {
		return 0, ""
	}
	if s, ok := v.(string); ok {
		s = strings.TrimSpace(s)
		if s == "" {
			return 0, ""
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f, ""
		}
		return 0, s
	}
	return util.ToFloat64(v), ""
}

// TradeMessage is a real-time trade tick. Open/High/Low/Avg are nil when the server omitted them.
type TradeMessage struct {
	Type        DataType
	TradingTime string
	Symbol      string
	Price       float64
	Quantity    int
	Side        string
	TotalVolume int
	Open        *float64
	High        *float64
	Low         *float64
	Avg         *float64
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
		Open:        util.ToOptFloat64(data["o"]),
		High:        util.ToOptFloat64(data["h"]),
		Low:         util.ToOptFloat64(data["l"]),
		Avg:         util.ToOptFloat64(data["a"]),
	}
}

// IndexTickMessage is the tick of an index itself, from trade.index.<code>. The server sends
// abbreviations only; the meaning of each field was inferred from the abbreviation, so Raw keeps
// the original object. A key the server omitted is nil.
type IndexTickMessage struct {
	Type               DataType
	Symbol             string
	Board              string
	Status             string
	TradingTime        string
	Value              *float64
	Change             *float64
	ChangePercent      *float64
	Advances           *float64
	Declines           *float64
	NoChange           *float64
	Ceilings           *float64
	Floors             *float64
	TotalMatchValue    *float64
	TotalMatchQuantity *float64
	TotalPutValue      *float64
	TotalPutQuantity   *float64
	TotalQuantity      *float64
	TotalValue         *float64
	Raw                map[string]interface{}
}

func IndexTickMessageFromMap(data map[string]interface{}) *IndexTickMessage {
	raw := make(map[string]interface{}, len(data))
	for k, v := range data {
		raw[k] = v
	}
	return &IndexTickMessage{
		Type:               DataTypeTrade,
		Symbol:             util.ToStr(data["s"]),
		Board:              util.ToStr(data["b"]),
		Status:             util.ToStr(data["ss"]),
		TradingTime:        util.ToStr(data["t"]),
		Value:              util.ToOptFloat64(data["v"]),
		Change:             util.ToOptFloat64(data["c"]),
		ChangePercent:      util.ToOptFloat64(data["cr"]),
		Advances:           util.ToOptFloat64(data["adv"]),
		Declines:           util.ToOptFloat64(data["dec"]),
		NoChange:           util.ToOptFloat64(data["nc"]),
		Ceilings:           util.ToOptFloat64(data["ce"]),
		Floors:             util.ToOptFloat64(data["fl"]),
		TotalMatchValue:    util.ToOptFloat64(data["tmv"]),
		TotalMatchQuantity: util.ToOptFloat64(data["tmq"]),
		TotalPutValue:      util.ToOptFloat64(data["tvp"]),
		TotalPutQuantity:   util.ToOptFloat64(data["tqp"]),
		TotalQuantity:      util.ToOptFloat64(data["tq"]),
		TotalValue:         util.ToOptFloat64(data["tv"]),
		Raw:                raw,
	}
}

// IntervalMessage is a real-time trade interval (OHLCV candle).
type IntervalMessage struct {
	Type         DataType
	IntervalTime string
	TradingTime  string
	Symbol       string
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       int
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

// QuoteMessage is a real-time bid/ask quote update; prices and volumes are parallel slices.
type QuoteMessage struct {
	Type        DataType
	TradingTime string
	Symbol      string
	BidPrices   []float64
	BidVolumes  []int
	AskPrices   []float64
	AskVolumes  []int
}

func QuoteMessageFromMap(data map[string]interface{}) *QuoteMessage {
	qm := &QuoteMessage{
		Type:        DataTypeQuote,
		TradingTime: util.ToStr(data["t"]),
		Symbol:      util.ToStr(data["s"]),
	}
	qm.BidPrices, qm.BidVolumes = levels(data["bids"])
	qm.AskPrices, qm.AskVolumes = levels(data["asks"])
	return qm
}

// MarketStatusMessage is a market status update (the legacy market.* shape).
type MarketStatusMessage struct {
	Market      string
	Status      string
	TradingDate string
}

func MarketStatusMessageFromMap(data map[string]interface{}) *MarketStatusMessage {
	return &MarketStatusMessage{
		Market:      util.ToStr(data["market"]),
		Status:      util.ToStr(data["status"]),
		TradingDate: util.ToStr(data["tradingDate"]),
	}
}

// MarketFlagMessage is a market session flag (ATO, LO, ATC, ...) from the market.flag topic.
// Board is exactly what the server sends: HOSE, HNX, UPCOM or DER (derivatives).
type MarketFlagMessage struct {
	Board       string
	TradingTime string
	Flag        string
}

func MarketFlagMessageFromMap(data map[string]interface{}) *MarketFlagMessage {
	return &MarketFlagMessage{
		Board:       util.ToStr(data["b"]),
		TradingTime: util.ToStr(data["t"]),
		Flag:        util.ToStr(data["f"]),
	}
}

// IndexSummaryMessage is the full summary of one index, from indexsummary.<code>. The public
// documentation does not list its fields, so none are guessed: Index comes from the topic and
// Data keeps what the server sent, untouched. Read values with Get.
type IndexSummaryMessage struct {
	Index string
	Data  map[string]interface{}
}

// IndexSummaryMessageFromMap takes the frame's data object and topic.
func IndexSummaryMessageFromMap(data map[string]interface{}, topic string) *IndexSummaryMessage {
	const prefix = "indexsummary."
	code := ""
	if len(topic) >= len(prefix) && strings.EqualFold(topic[:len(prefix)], prefix) {
		code = topic[len(prefix):]
	}
	copied := make(map[string]interface{}, len(data))
	for k, v := range data {
		copied[k] = v
	}
	return &IndexSummaryMessage{Index: strings.ToUpper(code), Data: copied}
}

// Get returns the first of keys present in Data, or nil if none is.
func (m *IndexSummaryMessage) Get(keys ...string) interface{} {
	for _, key := range keys {
		if v, ok := m.Data[key]; ok {
			return v
		}
	}
	return nil
}

// MarketDataMessage is master data (ceiling / floor / reference price) from market.<symbol>. A
// price the server omitted is nil.
type MarketDataMessage struct {
	Symbol      string
	Board       string
	TradingTime string
	Ceiling     *float64
	Floor       *float64
	RefPrice    *float64
}

func MarketDataMessageFromMap(data map[string]interface{}) *MarketDataMessage {
	return &MarketDataMessage{
		Symbol:      util.ToStr(data["s"]),
		Board:       util.ToStr(data["b"]),
		TradingTime: util.ToStr(data["t"]),
		Ceiling:     util.ToOptFloat64(data["ce"]),
		Floor:       util.ToOptFloat64(data["fl"]),
		RefPrice:    util.ToOptFloat64(data["ref"]),
	}
}

// ForeignRoomMessage is foreign room (foreign investor activity) data.
type ForeignRoomMessage struct {
	Type         DataType
	TradingTime  string
	Symbol       string
	TotalRoom    int
	CurrentRoom  int
	BuyQuantity  int
	BuyValue     int
	SellQuantity int
	SellValue    int
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
	Type          DataType
	TradingTime   string
	Symbol        string
	Price         float64
	Quantity      int
	TotalQuantity int
	TotalValue    int
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
	Type        DataType
	TradingTime string
	Symbol      string
	Price       float64
	Quantity    int
	BidPrices   []float64
	BidVolumes  []int
	AskPrices   []float64
	AskVolumes  []int
}

func OddLotMessageFromMap(data map[string]interface{}) *OddLotMessage {
	ol := &OddLotMessage{
		Type:        DataTypeOddLot,
		TradingTime: util.ToStr(data["t"]),
		Symbol:      util.ToStr(data["s"]),
		Price:       util.ToFloat64(data["p"]),
		Quantity:    util.ToInt(data["q"]),
	}
	ol.BidPrices, ol.BidVolumes = levels(data["bids"])
	ol.AskPrices, ol.AskVolumes = levels(data["asks"])
	return ol
}

func sideOf(v interface{}) trading.OrderSide {
	raw := util.ToStr(v)
	if s, ok := trading.OrderSideFromValue(raw); ok {
		return s
	}
	return trading.OrderSide(raw)
}

func orderTypeOf(v interface{}) trading.OrderType {
	raw := util.ToStr(v)
	if s, ok := trading.OrderTypeFromValue(raw); ok {
		return s
	}
	return trading.OrderType(raw)
}

func orderStatusOf(v interface{}) trading.OrderStatus {
	raw := util.ToStr(v)
	if s, ok := trading.OrderStatusFromValue(raw); ok {
		return s
	}
	return trading.OrderStatus(raw)
}

// OrderStatusMessage is a real-time order status update (orderEvent on order.<account>).
//
// The price of an ATO/ATC/MP order is a word, so Price is 0 and PriceText holds the word then
// (same for AvgPrice). Besides the usual cycle an order can report PD (accepted, not yet at the
// exchange) and RJ (rejected by the server itself, with ErrorCode). An unknown Side, OrderType or
// Status is kept as its raw string; an absent one is "".
type OrderStatusMessage struct {
	Type            StreamingType
	AccountNo       string
	ClientRequestID string
	OrderID         string
	Symbol          string
	Side            trading.OrderSide
	OrderType       trading.OrderType
	Price           float64
	PriceText       string
	Quantity        int
	OSQuantity      int
	FilledQuantity  int
	CancelQuantity  int
	Status          trading.OrderStatus
	InputTime       string
	ModifiedTime    string
	Message         string
	AvgPrice        float64
	AvgPriceText    string
	RejectReason    string
	SystemRequestID string
	ErrorCode       string
	ErrorMessage    string
	NotifyID        string
	ConnectionID    string
}

func OrderStatusMessageFromMap(data map[string]interface{}) *OrderStatusMessage {
	osm := &OrderStatusMessage{
		Type:            StreamingTypeOrder,
		AccountNo:       util.ToStr(data["accountNo"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		OrderID:         util.ToStr(data["orderId"]),
		Symbol:          util.ToStr(data["symbol"]),
		Side:            sideOf(data["side"]),
		OrderType:       orderTypeOf(data["orderType"]),
		Quantity:        util.ToInt(data["quantity"]),
		OSQuantity:      util.ToInt(data["osQty"]),
		FilledQuantity:  util.ToInt(data["filledQty"]),
		CancelQuantity:  util.ToInt(data["cancelQty"]),
		Status:          orderStatusOf(data["orderStatus"]),
		InputTime:       util.ToStr(data["inputTime"]),
		ModifiedTime:    util.ToStr(util.Pick(data, "modifiedTime", "modifyTime")),
		Message:         util.ToStr(data["rejectReason"]),
		RejectReason:    util.ToStr(data["rejectReason"]),
		SystemRequestID: util.ToStr(data["systemRequestId"]),
		ErrorCode:       util.ToStr(data["errorCode"]),
		ErrorMessage:    util.ToStr(data["errorMessage"]),
		NotifyID:        util.ToStr(data["notifyId"]),
		ConnectionID:    util.ToStr(data["connectionId"]),
	}
	osm.Price, osm.PriceText = parsePrice(data["price"])
	osm.AvgPrice, osm.AvgPriceText = parsePrice(data["avgPrice"])
	return osm
}

// OrderMatchMessage is a fill of an order (orderMatchEvent on order.<account>).
type OrderMatchMessage struct {
	Type            StreamingType
	NotifyID        string
	ConnectionID    string
	ClientRequestID string
	OrderID         string
	Symbol          string
	Side            trading.OrderSide
	AccountNo       string
	MatchPrice      float64
	MatchPriceText  string
	MatchQty        int
	MatchTime       string
}

func OrderMatchMessageFromMap(data map[string]interface{}) *OrderMatchMessage {
	m := &OrderMatchMessage{
		Type:            StreamingTypeOrderMatch,
		NotifyID:        util.ToStr(data["notifyId"]),
		ConnectionID:    util.ToStr(data["connectionId"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		OrderID:         util.ToStr(data["orderId"]),
		Symbol:          util.ToStr(data["symbol"]),
		Side:            sideOf(data["side"]),
		AccountNo:       util.ToStr(data["accountNo"]),
		MatchQty:        util.ToInt(data["matchQty"]),
		MatchTime:       util.ToStr(data["matchTime"]),
	}
	m.MatchPrice, m.MatchPriceText = parsePrice(data["matchPrice"])
	return m
}

// PortfolioMessage is a real-time position update (clientPortfolioEvent, derivative accounts
// only). A price the server omitted is nil. TotalAsset, CashBalance and StockValue are always
// nil: the server never sends them.
type PortfolioMessage struct {
	Type         StreamingType
	AccountNo    string
	TotalAsset   *float64
	CashBalance  *float64
	StockValue   *float64
	Symbol       string
	LongQty      int
	ShortQty     int
	Net          int
	BidAvgPrice  *float64
	AskAvgPrice  *float64
	TradePrice   *float64
	MarketPrice  *float64
	FloatingPL   *float64
	TradingPL    *float64
	PositionType string
}

func PortfolioMessageFromMap(data map[string]interface{}) *PortfolioMessage {
	return &PortfolioMessage{
		Type:         StreamingTypePortfolio,
		AccountNo:    util.ToStr(data["accountNo"]),
		TotalAsset:   util.ToOptFloat64(data["totalAsset"]),
		CashBalance:  util.ToOptFloat64(data["cashBalance"]),
		StockValue:   util.ToOptFloat64(data["stockValue"]),
		Symbol:       util.ToStr(data["symbol"]),
		LongQty:      util.ToInt(data["longQty"]),
		ShortQty:     util.ToInt(data["shortQty"]),
		Net:          util.ToInt(data["net"]),
		BidAvgPrice:  util.ToOptFloat64(data["bidAvgPrice"]),
		AskAvgPrice:  util.ToOptFloat64(data["askAvgPrice"]),
		TradePrice:   util.ToOptFloat64(data["tradePrice"]),
		MarketPrice:  util.ToOptFloat64(data["marketPrice"]),
		FloatingPL:   util.ToOptFloat64(data["floatingPL"]),
		TradingPL:    util.ToOptFloat64(data["tradingPL"]),
		PositionType: util.ToStr(data["positionType"]),
	}
}

// FCOOrderUpdateMessage is a real-time FCO (conditional order) update (eventType fcoEvent).
type FCOOrderUpdateMessage struct {
	FCOID           string
	ProcessStatus   trading.FCOStatus
	MatchedQuantity int
	IsPlaceOrder    bool
	Symbol          string
	Quantity        int
	Price           string
	AccountNo       string
	UpdatedTime     string
	Status          string
	Message         string
	Username        string
	EventType       string
	Type            trading.FCOType
}

func FCOOrderUpdateMessageFromMap(data map[string]interface{}) *FCOOrderUpdateMessage {
	common := trading.ParseFCOCommon(data)
	status := util.ToStr(data["processStatus"])
	if s, ok := trading.FCOStatusFromValue(status); ok {
		status = string(s)
	}
	return &FCOOrderUpdateMessage{
		FCOID:           common.FCOID,
		ProcessStatus:   trading.FCOStatus(status),
		MatchedQuantity: common.MatchedQuantity,
		IsPlaceOrder:    common.IsPlaceOrder,
		Symbol:          common.Symbol,
		Quantity:        common.Quantity,
		Price:           util.ToStr(data["price"]),
		AccountNo:       common.AccountNo,
		UpdatedTime:     util.ToStr(data["updatedTime"]),
		Status:          util.ToStr(data["status"]),
		Message:         util.ToStr(data["message"]),
		Username:        util.ToStr(data["username"]),
		EventType:       util.ToStr(data["eventType"]),
		Type:            common.Type,
	}
}

func payload(msg map[string]interface{}) map[string]interface{} {
	data, _ := msg["data"].(map[string]interface{})
	if data == nil {
		return map[string]interface{}{}
	}
	return data
}

// ParseDataMessage deserialises a DATA-channel frame into its message type by topic, matched
// case-insensitively (the server upper-cases the part after the type). A frame whose topic has
// no model here (including acks) is returned as the raw map.
func ParseDataMessage(msg map[string]interface{}) interface{} {
	topic, _ := msg["topic"].(string)
	data := payload(msg)
	kind := strings.ToLower(topic)
	switch {
	case strings.HasPrefix(kind, "trade.index."):
		if strings.HasSuffix(kind, "@1m") || strings.HasSuffix(kind, "@5m") {
			return IntervalMessageFromMap(data)
		}
		return IndexTickMessageFromMap(data)
	case strings.HasPrefix(kind, string(DataTopicTrade)):
		if strings.HasSuffix(kind, "@1m") || strings.HasSuffix(kind, "@5m") {
			return IntervalMessageFromMap(data)
		}
		return TradeMessageFromMap(data)
	case strings.HasPrefix(kind, string(DataTopicQuote)):
		return QuoteMessageFromMap(data)
	case strings.HasPrefix(kind, string(DataTopicRoom)):
		return ForeignRoomMessageFromMap(data)
	case kind == TopicMarketFlag:
		return MarketFlagMessageFromMap(data)
	case strings.HasPrefix(kind, string(DataTopicIndexSummary)):
		return IndexSummaryMessageFromMap(data, topic)
	case strings.HasPrefix(kind, string(DataTopicMarket)):
		for _, key := range []string{"s", "ce", "fl", "ref"} {
			if _, ok := data[key]; ok {
				return MarketDataMessageFromMap(data)
			}
		}
		return MarketStatusMessageFromMap(data)
	case strings.HasPrefix(kind, string(DataTopicPut)):
		return PutMessageFromMap(data)
	case strings.HasPrefix(kind, string(DataTopicOddLot)):
		return OddLotMessageFromMap(data)
	}
	return msg
}

// ParseTradingMessage deserialises a TRADING-channel frame. The type is chosen by
// data.eventType; with none the topic decides (order.* / portfolio.*). An event type this SDK
// does not know is passed on as the raw map.
func ParseTradingMessage(msg map[string]interface{}) interface{} {
	topic, _ := msg["topic"].(string)
	data := payload(msg)
	event, _ := data["eventType"].(string)
	switch event {
	case "fcoEvent":
		return FCOOrderUpdateMessageFromMap(data)
	case string(StreamingTypeOrderMatch):
		return OrderMatchMessageFromMap(data)
	case string(StreamingTypePortfolio):
		return PortfolioMessageFromMap(data)
	case string(StreamingTypeOrder):
		return OrderStatusMessageFromMap(data)
	case "":
		if strings.HasPrefix(topic, "order.") {
			return OrderStatusMessageFromMap(data)
		}
		if strings.HasPrefix(topic, "portfolio.") {
			return PortfolioMessageFromMap(data)
		}
	}
	return msg
}
