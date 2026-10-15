package trading

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

// marshal encodes a request body. The returned text is exactly what gets signed and sent.
func marshal(v interface{}) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", ssi.NewValidationError("request body cannot be encoded as JSON (a number is not finite)")
	}
	return string(b), nil
}

// defaultPayloadUserAgent is put in order payloads when Config.UserAgent is empty.
func defaultPayloadUserAgent() string { return "SSI Go SDK/" + ssi.Version }

// ---------------------------------------------------------------------------
// Orders
// ---------------------------------------------------------------------------

// PlaceOrderRequest is the payload for placing a new order.
//
// Price is a float64 and goes out as a plain decimal string (never exponent notation). Market-priced
// order types (ATO/ATC/MTL/MP/MOK/MAK) send "0"; LO/PLO need a positive price.
type PlaceOrderRequest struct {
	AccountNo       string
	Symbol          string
	Side            OrderSide
	Quantity        int
	Price           float64
	OrderType       OrderType
	ClientRequestID string
	DeviceID        string
	UserAgent       string
}

type placeOrderWire struct {
	AccountNo       string `json:"accountNo"`
	Symbol          string `json:"symbol"`
	Side            string `json:"side"`
	Quantity        int    `json:"quantity"`
	Price           string `json:"price"`
	OrderType       string `json:"orderType"`
	ClientRequestID string `json:"clientRequestId"`
	DeviceID        string `json:"deviceId"`
	UserAgent       string `json:"userAgent"`
}

func (r *PlaceOrderRequest) priceText() (string, error) {
	if pricedOrderType(r.OrderType) && !(r.Price > 0) {
		return "", ssi.NewValidationError(fmt.Sprintf("price is required for %s orders", r.OrderType))
	}
	text, err := util.FormatPrice(r.Price)
	if err != nil {
		return "", ssi.NewValidationError(err.Error())
	}
	return text, nil
}

func (r *PlaceOrderRequest) wire() (placeOrderWire, error) {
	price, err := r.priceText()
	if err != nil {
		return placeOrderWire{}, err
	}
	return placeOrderWire{
		AccountNo: r.AccountNo, Symbol: r.Symbol, Side: string(r.Side), Quantity: r.Quantity,
		Price: price, OrderType: string(r.OrderType), ClientRequestID: r.ClientRequestID,
		DeviceID: r.DeviceID, UserAgent: r.UserAgent,
	}, nil
}

// ToJSON is the exact body that is signed and sent.
func (r *PlaceOrderRequest) ToJSON() (string, error) {
	w, err := r.wire()
	if err != nil {
		return "", err
	}
	return marshal(w)
}

// PlaceOrderResponse is the response for placing a new order.
type PlaceOrderResponse struct {
	OrderID         string
	ClientRequestID string
	// Status is an OrderStatus constant, or the raw string for a value this SDK does not know.
	Status OrderStatus
}

func PlaceOrderResponseFromMap(data map[string]interface{}) *PlaceOrderResponse {
	return &PlaceOrderResponse{
		OrderID:         util.ToStr(data["orderId"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		Status:          normOrderStatus(util.ToStr(data["orderStatus"])),
	}
}

// ModifyOrderRequest is the modify order payload. A call changes either the price or the
// quantity, never both.
type ModifyOrderRequest struct {
	AccountNo       string
	Quantity        *int
	Price           *float64
	OrderID         string
	ClientModifyID  string
	ClientRequestID string
	DeviceID        string
	UserAgent       string
}

type modifyOrderWire struct {
	AccountNo       string `json:"accountNo"`
	ClientModifyID  string `json:"clientModifyId"`
	DeviceID        string `json:"deviceId"`
	UserAgent       string `json:"userAgent"`
	OrderID         string `json:"orderId,omitempty"`
	ClientRequestID string `json:"clientRequestId,omitempty"`
	Quantity        *int   `json:"quantity,omitempty"`
	Price           string `json:"price,omitempty"`
}

// ToJSON is the exact body that is signed and sent.
func (r *ModifyOrderRequest) ToJSON() (string, error) {
	w := modifyOrderWire{
		AccountNo: r.AccountNo, ClientModifyID: r.ClientModifyID, DeviceID: r.DeviceID,
		UserAgent: r.UserAgent, OrderID: r.OrderID, ClientRequestID: r.ClientRequestID,
		Quantity: r.Quantity,
	}
	if r.Price != nil {
		text, err := util.FormatPrice(*r.Price)
		if err != nil {
			return "", ssi.NewValidationError(err.Error())
		}
		w.Price = text
	}
	return marshal(w)
}

// ModifyOrderResponse is the response for modifying an order.
type ModifyOrderResponse struct {
	ClientModifyID  string
	OrderID         string
	ClientRequestID string
	Status          OrderStatus
}

func ModifyOrderResponseFromMap(data map[string]interface{}) *ModifyOrderResponse {
	return &ModifyOrderResponse{
		ClientModifyID:  util.ToStr(data["clientModifyId"]),
		OrderID:         util.ToStr(data["orderId"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		Status:          normOrderStatus(util.ToStr(data["orderStatus"])),
	}
}

// CancelOrderRequest is the cancel order payload.
type CancelOrderRequest struct {
	AccountNo       string
	OrderID         string
	ClientRequestID string
	ClientCancelID  string
	DeviceID        string
	UserAgent       string
}

type cancelOrderWire struct {
	AccountNo       string `json:"accountNo"`
	ClientCancelID  string `json:"clientCancelId"`
	DeviceID        string `json:"deviceId"`
	UserAgent       string `json:"userAgent"`
	OrderID         string `json:"orderId,omitempty"`
	ClientRequestID string `json:"clientRequestId,omitempty"`
}

func (r *CancelOrderRequest) wire() cancelOrderWire {
	return cancelOrderWire{
		AccountNo: r.AccountNo, ClientCancelID: r.ClientCancelID, DeviceID: r.DeviceID,
		UserAgent: r.UserAgent, OrderID: r.OrderID, ClientRequestID: r.ClientRequestID,
	}
}

// ToJSON is the exact body that is signed and sent.
func (r *CancelOrderRequest) ToJSON() (string, error) { return marshal(r.wire()) }

// CancelOrderResponse is the response for canceling an order.
type CancelOrderResponse struct {
	ClientCancelID  string
	OrderID         string
	ClientRequestID string
	Status          OrderStatus
}

func CancelOrderResponseFromMap(data map[string]interface{}) *CancelOrderResponse {
	return &CancelOrderResponse{
		ClientCancelID:  util.ToStr(data["clientCancelId"]),
		OrderID:         util.ToStr(data["orderId"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		Status:          normOrderStatus(util.ToStr(data["orderStatus"])),
	}
}

// MaxBuySellRequest is the query for the max buy/sell endpoint. Price is optional (nil = market
// price).
type MaxBuySellRequest struct {
	AccountNo string
	Symbol    string
	Price     *float64
}

// ToMap builds the query parameters; the symbol is upper-cased and price is sent only when set.
func (r *MaxBuySellRequest) ToMap() (map[string]string, error) {
	result := map[string]string{
		"accountNo": r.AccountNo,
		"symbol":    strings.ToUpper(r.Symbol),
	}
	if r.Price != nil {
		text, err := util.FormatPrice(*r.Price)
		if err != nil {
			return nil, ssi.NewValidationError(err.Error())
		}
		result["price"] = text
	}
	return result, nil
}

// MaxBuySellResponse is the response for max buy/sell request. Derivative accounts get only the
// two quantities: PurchasingPower and MarginRatio stay nil there, which is not the same as 0.
type MaxBuySellResponse struct {
	AccountNo       string
	Symbol          string
	MaxBuyQuantity  int
	MaxSellQuantity int
	// MarginRatio is the server's string; nil on derivative accounts.
	MarginRatio *string
	// PurchasingPower is nil on derivative accounts.
	PurchasingPower *int
}

// PurchasePower is the old string form of PurchasingPower ("" when absent).
//
// Deprecated: use PurchasingPower.
func (r *MaxBuySellResponse) PurchasePower() string {
	if r.PurchasingPower == nil {
		return ""
	}
	return strconv.Itoa(*r.PurchasingPower)
}

func MaxBuySellResponseFromMap(data map[string]interface{}, symbol string) *MaxBuySellResponse {
	res := &MaxBuySellResponse{
		AccountNo:       util.ToStr(data["accountNo"]),
		Symbol:          strings.ToUpper(symbol),
		MaxBuyQuantity:  util.ToInt(data["maxBuyQty"]),
		MaxSellQuantity: util.ToInt(data["maxSellQty"]),
		PurchasingPower: util.ToOptInt(util.Pick(data, "purchasingPower", "purchasePower")),
	}
	if ratio := util.ToStr(data["marginRatio"]); ratio != "" {
		res.MarginRatio = &ratio
	}
	return res
}

// ---------------------------------------------------------------------------
// Batch
// ---------------------------------------------------------------------------

// BatchOrderRequest is the body of POST/DELETE /api/v3/trading/order/batch (and the WebSocket
// batch params).
//
// BatchRequestID must be unused today for this apiKey (a repeat is a 409) and BatchRequestTime
// must be now in epoch milliseconds (outside the server's replay window it is a 400111). The
// whole batch is validated first: one bad order rejects all.
type BatchOrderRequest struct {
	BatchRequestID   string
	BatchRequestTime int64
	// Orders are the per-order wire payloads, built by the service.
	Orders []interface{}
}

type batchWire struct {
	BatchRequestID   string        `json:"batchRequestId"`
	BatchRequestTime int64         `json:"batchRequestTime"`
	Orders           []interface{} `json:"orders"`
}

// ToJSON is the exact body that is signed and sent.
func (r *BatchOrderRequest) ToJSON() (string, error) {
	orders := r.Orders
	if orders == nil {
		orders = []interface{}{}
	}
	return marshal(batchWire{r.BatchRequestID, r.BatchRequestTime, orders})
}

// BatchOrderResult is the outcome of one order of a batch.
type BatchOrderResult struct {
	ClientRequestID string
	OrderID         string
	Status          OrderStatus
	Success         bool
	ErrorCode       string
	ErrorMessage    string
	ClientCancelID  string
}

func batchResultFromMap(data map[string]interface{}) BatchOrderResult {
	return BatchOrderResult{
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		OrderID:         util.ToStr(data["orderId"]),
		Status:          normOrderStatus(util.ToStr(data["orderStatus"])),
		Success:         util.ToBool(data["success"]),
		ErrorCode:       util.ToStr(data["errorCode"]),
		ErrorMessage:    util.ToStr(data["errorMessage"]),
		ClientCancelID:  util.ToStr(data["clientCancelId"]),
	}
}

// BatchOrderResponse is the response of a batch request.
type BatchOrderResponse struct {
	BatchRequestID string
	// Results holds one result per order, in request order.
	Results []BatchOrderResult
}

// BatchOrderResponseFromMap parses the response; a missing or non-list results is an empty list.
func BatchOrderResponseFromMap(data map[string]interface{}) *BatchOrderResponse {
	res := &BatchOrderResponse{BatchRequestID: util.ToStr(data["batchRequestId"]), Results: []BatchOrderResult{}}
	raw, _ := data["results"].([]interface{})
	for _, item := range raw {
		if m, ok := item.(map[string]interface{}); ok {
			res.Results = append(res.Results, batchResultFromMap(m))
		}
	}
	return res
}

// BatchPlaceOrderItem is one order of PlaceBatchOrders (the same arguments as PlaceOrder).
// ClientRequestID is your idempotency key (<= 20 characters) and is generated when empty.
type BatchPlaceOrderItem struct {
	AccountNo       string
	Symbol          string
	Side            OrderSide
	Quantity        int
	Price           float64
	OrderType       OrderType
	ClientRequestID string
}

// BatchCancelOrderItem is one order of CancelBatchOrders: AccountNo plus exactly one of OrderID or
// ClientRequestID.
type BatchCancelOrderItem struct {
	AccountNo       string
	OrderID         string
	ClientRequestID string
}

// ---------------------------------------------------------------------------
// FCO shared parsing
// ---------------------------------------------------------------------------

// FCOCommon holds the fields every FCO payload carries (REST list/order book and stream events
// alike). One parser, so the REST models and the stream message cannot drift apart.
type FCOCommon struct {
	FCOID           string
	AccountNo       string
	Symbol          string
	Quantity        int
	MatchedQuantity int
	IsPlaceOrder    bool
	Type            FCOType
}

// ParseFCOCommon reads camelCase first, with the spec's casing as a fallback.
func ParseFCOCommon(data map[string]interface{}) FCOCommon {
	return FCOCommon{
		FCOID:           util.ToStr(util.Pick(data, "fcoId", "FcoId")),
		AccountNo:       util.ToStr(util.Pick(data, "accountNo", "AccountNo")),
		Symbol:          util.ToStr(util.Pick(data, "symbol", "Symbol")),
		Quantity:        util.ToInt(util.Pick(data, "quantity", "Quantity")),
		MatchedQuantity: util.ToInt(util.Pick(data, "matchedQuantity", "matchedQuality")),
		IsPlaceOrder:    util.ToBool(util.Pick(data, "isPlaceOrder", "IsPlaceOrder")),
		Type:            normFCOType(util.ToStr(data["type"])),
	}
}

func optFloat(v interface{}) *float64 { return util.ToOptFloat64(v) }

// FCOParams holds the detailed trigger parameters of an FCO order; unset fields are nil.
type FCOParams struct {
	StopPrice      *float64
	Side           OrderSide
	ActivePrice    *float64
	TrailingAmount *float64
	TPActivePrice  *float64
	SLActivePrice  *float64
	TPPrice        string
	SLPrice        string
	TPSlip         *float64
	SLSlip         *float64
	Operator       FCOOperator
}

// FCOParamsFromMap returns nil for a nil or empty map. The operator may be a name or the
// server's number 0..4.
func FCOParamsFromMap(data map[string]interface{}) *FCOParams {
	if len(data) == 0 {
		return nil
	}
	p := &FCOParams{
		StopPrice:      optFloat(data["stopPrice"]),
		ActivePrice:    optFloat(data["activePrice"]),
		TrailingAmount: optFloat(data["trailingAmount"]),
		TPActivePrice:  optFloat(data["tpActivePrice"]),
		SLActivePrice:  optFloat(data["slActivePrice"]),
		TPPrice:        util.ToStr(data["tpPrice"]),
		SLPrice:        util.ToStr(data["slPrice"]),
		TPSlip:         optFloat(data["tpSlip"]),
		SLSlip:         optFloat(data["slSlip"]),
		Operator:       ParseFCOOperator(util.ToStr(util.Pick(data, "operator", "price_operator"))),
	}
	if side := util.ToStr(data["side"]); side != "" {
		p.Side = normOrderSide(side)
	}
	return p
}

func asMap(v interface{}) map[string]interface{} {
	m, _ := v.(map[string]interface{})
	return m
}

// FCOInfo is a conditional order item in the FCO list.
type FCOInfo struct {
	FCOID           string
	ClientID        string
	AccountNo       string
	Quantity        int
	Price           string
	PriceSlip       *float64
	Symbol          string
	Type            FCOType
	FromDate        string
	ToDate          string
	MatchedQuantity int
	IsPlaceOrder    bool
	Status          FCOStatus
	Detail          string
	Params          *FCOParams
	OrderType       OrderType
}

func FCOInfoFromMap(data map[string]interface{}) *FCOInfo {
	common := ParseFCOCommon(data)
	return &FCOInfo{
		FCOID:           common.FCOID,
		ClientID:        util.ToStr(util.Pick(data, "username", "Username")),
		AccountNo:       common.AccountNo,
		Quantity:        common.Quantity,
		Price:           util.ToStr(data["price"]),
		PriceSlip:       optFloat(data["priceSlip"]),
		Symbol:          common.Symbol,
		Type:            common.Type,
		FromDate:        util.ToStr(data["from"]),
		ToDate:          util.ToStr(data["to"]),
		MatchedQuantity: common.MatchedQuantity,
		IsPlaceOrder:    common.IsPlaceOrder,
		Status:          normFCOStatus(util.ToStr(data["status"])),
		Detail:          util.ToStr(data["detail"]),
		Params:          FCOParamsFromMap(asMap(util.Pick(data, "params", "fco_params", "fcoParams"))),
		OrderType:       normOrderType(util.ToStr(data["orderType"])),
	}
}

func FCOInfoSliceFromList(raw interface{}) []*FCOInfo {
	list, _ := raw.([]interface{})
	result := make([]*FCOInfo, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, FCOInfoFromMap(m))
		}
	}
	return result
}

// FCOListRequest is the query for the FCO list. ProcessStatus may hold several statuses: they go
// out as a repeated processStatus query key. PageIndex/PageSize are sent only when > 0. From/To
// are "YYYY/MM/DD HH:MM:SS" (a bare "YYYY/MM/DD" is accepted too).
type FCOListRequest struct {
	AccountNo     string
	FCOID         string
	Type          FCOType
	ProcessStatus []FCOStatus
	Symbol        string
	Side          OrderSide
	FromDate      string
	ToDate        string
	PageIndex     int
	PageSize      int
}

// ToQuery builds the query string values; type, statuses and side are resolved
// case-insensitively and an unknown value is sent as given.
func (r *FCOListRequest) ToQuery() url.Values {
	q := url.Values{}
	q.Set("accountNo", r.AccountNo)
	if r.FCOID != "" {
		q.Set("fcoId", r.FCOID)
	}
	if r.Type != "" {
		q.Set("type", string(normFCOType(string(r.Type))))
	}
	for _, s := range r.ProcessStatus {
		q.Add("processStatus", string(normFCOStatus(string(s))))
	}
	if r.Symbol != "" {
		q.Set("symbol", r.Symbol)
	}
	if r.Side != "" {
		q.Set("side", string(normOrderSide(string(r.Side))))
	}
	if r.FromDate != "" {
		q.Set("from", r.FromDate)
	}
	if r.ToDate != "" {
		q.Set("to", r.ToDate)
	}
	if r.PageIndex > 0 {
		q.Set("pageIndex", strconv.Itoa(r.PageIndex))
	}
	if r.PageSize > 0 {
		q.Set("pageSize", strconv.Itoa(r.PageSize))
	}
	return q
}

// FCOListResponse is a page of FCOs. PagesCount is the server's count and can miss the last
// partial page: page until a page comes back empty, or use ItemsCount.
type FCOListResponse struct {
	PageIndex  int
	PageSize   int
	ItemsCount int
	PagesCount int
	FCOList    []*FCOInfo
}

func pagedDefaults(data map[string]interface{}) (map[string]interface{}, int, int, int, int) {
	if data == nil || util.IsNoContent(data) {
		data = map[string]interface{}{}
	}
	pageIndex, pageSize := 1, 10
	if v := util.ToOptInt(data["pageIndex"]); v != nil {
		pageIndex = *v
	}
	if v := util.ToOptInt(data["pageSize"]); v != nil {
		pageSize = *v
	}
	return data, pageIndex, pageSize, util.ToInt(data["itemsCount"]), util.ToInt(util.Pick(data, "pagesCount", "pageCount"))
}

func FCOListResponseFromMap(data map[string]interface{}) *FCOListResponse {
	data, pageIndex, pageSize, items, pages := pagedDefaults(data)
	return &FCOListResponse{
		PageIndex:  pageIndex,
		PageSize:   pageSize,
		ItemsCount: items,
		PagesCount: pages,
		FCOList:    FCOInfoSliceFromList(util.Pick(data, "data", "fcoList", "fco_list")),
	}
}

// FCOOrderBookRequest is the query for the FCO order book.
type FCOOrderBookRequest struct {
	FCOID     string
	PageIndex int
	PageSize  int
}

func (r *FCOOrderBookRequest) ToMap() map[string]string {
	params := map[string]string{"fcoId": r.FCOID}
	if r.PageIndex > 0 {
		params["pageIndex"] = strconv.Itoa(r.PageIndex)
	}
	if r.PageSize > 0 {
		params["pageSize"] = strconv.Itoa(r.PageSize)
	}
	return params
}

// FCOOrder is an entry in the FCO order book / execution log.
type FCOOrder struct {
	FCOID           string
	AccountNo       string
	Quantity        int
	Price           string
	Symbol          string
	Side            OrderSide
	OrderType       OrderType
	IsMainOrder     bool
	IsAttachedOrder bool
	CreatedTime     string
	UpdatedTime     string
	UniqueID        string
	OrderID         string
	MatchedQuantity int
	OSQuantity      int
	AvgPrice        float64
	Status          OrderStatus
	Detail          string
	Type            FCOType
}

func FCOOrderFromMap(data map[string]interface{}) *FCOOrder {
	common := ParseFCOCommon(data)
	return &FCOOrder{
		FCOID:           common.FCOID,
		AccountNo:       common.AccountNo,
		Quantity:        common.Quantity,
		Price:           util.ToStr(data["price"]),
		Symbol:          common.Symbol,
		Side:            normOrderSide(util.ToStr(data["side"])),
		OrderType:       normOrderType(util.ToStr(data["orderType"])),
		IsMainOrder:     util.ToBool(util.Pick(data, "isMainOrder", "IsMainOrder")),
		IsAttachedOrder: util.ToBool(data["isAttachedOrder"]),
		CreatedTime:     util.ToStr(data["createdTime"]),
		UpdatedTime:     util.ToStr(data["updatedTime"]),
		UniqueID:        util.ToStr(util.Pick(data, "uniqueId", "uniquedId", "UniqueId")),
		OrderID:         util.ToStr(data["orderId"]),
		MatchedQuantity: common.MatchedQuantity,
		OSQuantity:      util.ToInt(data["osQuantity"]),
		AvgPrice:        util.ToFloat64(data["avgPrice"]),
		Status:          normOrderStatus(util.ToStr(data["status"])),
		Detail:          util.ToStr(data["detail"]),
		Type:            common.Type,
	}
}

func FCOOrderSliceFromList(raw interface{}) []*FCOOrder {
	list, _ := raw.([]interface{})
	result := make([]*FCOOrder, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, FCOOrderFromMap(m))
		}
	}
	return result
}

// FCOOrderBookResponse is a page of order book entries. PagesCount is the server's count and
// can miss the last partial page: page until a page comes back empty, or use ItemsCount.
type FCOOrderBookResponse struct {
	PageIndex  int
	PageSize   int
	ItemsCount int
	PagesCount int
	OrderBook  []*FCOOrder
}

func FCOOrderBookResponseFromMap(data map[string]interface{}) *FCOOrderBookResponse {
	data, pageIndex, pageSize, items, pages := pagedDefaults(data)
	return &FCOOrderBookResponse{
		PageIndex:  pageIndex,
		PageSize:   pageSize,
		ItemsCount: items,
		PagesCount: pages,
		OrderBook:  FCOOrderSliceFromList(util.Pick(data, "data", "orderBook", "order_book")),
	}
}

// ---------------------------------------------------------------------------
// FCO place requests
// ---------------------------------------------------------------------------

// FCOParamsRequest is implemented by the five FCO place payloads.
type FCOParamsRequest interface {
	// ToJSON is the exact body that is signed and sent.
	ToJSON() (string, error)
	setIdentity(deviceID, userAgent string)
	window() (from, to string)
}

// priceAndSlip returns the price text and slip for an FCO price that may be a number or a word.
// A numeric price keeps its slip; an order-type word (MP/MTL/...) has no slip.
func priceAndSlip(price interface{}, slip float64) (string, float64, error) {
	switch v := price.(type) {
	case nil:
		return "", 0, nil
	case OrderType:
		return string(v), slipForWord(string(v), slip), nil
	case string:
		return v, slipForWord(v, slip), nil
	case float64:
		text, err := util.FormatPrice(v)
		return text, slip, err
	case float32:
		text, err := util.FormatPrice(float64(v))
		return text, slip, err
	case int:
		return strconv.Itoa(v), slip, nil
	case int64:
		return strconv.FormatInt(v, 10), slip, nil
	default:
		return fmt.Sprint(v), 0, nil
	}
}

// marketWords are the order-type words that carry no slip (a limit word such as LO/PLO keeps it).
var marketWords = map[string]bool{"ATO": true, "ATC": true, "MTL": true, "MP": true, "MOK": true, "MAK": true}

func slipForWord(word string, slip float64) float64 {
	if marketWords[strings.ToUpper(strings.TrimSpace(word))] {
		return 0
	}
	return slip
}

func priceOnly(price interface{}) (string, error) {
	text, _, err := priceAndSlip(price, 0)
	return text, err
}

func wrapPriceErr(err error) error {
	if err == nil {
		return nil
	}
	return ssi.NewValidationError(err.Error())
}

// GTDParams is a good-till-date order. Price may be a number or an OrderType word (the slip is
// then 0).
type GTDParams struct {
	AccountNo string
	Symbol    string
	Side      OrderSide
	Price     interface{}
	PriceSlip float64
	Quantity  int
	FromDate  string
	ToDate    string
	DeviceID  string
	UserAgent string
	// Code is the one-time code (OTP) when the account requires it; omitted if empty.
	Code string
}

func (p *GTDParams) setIdentity(d, ua string) { p.DeviceID, p.UserAgent = d, ua }
func (p *GTDParams) window() (string, string) { return p.FromDate, p.ToDate }

type gtdWire struct {
	AccountNo string  `json:"accountNo"`
	Type      string  `json:"type"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     string  `json:"price"`
	PriceSlip float64 `json:"priceSlip"`
	Quantity  int     `json:"quantity"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	DeviceID  string  `json:"deviceId"`
	UserAgent string  `json:"userAgent"`
	Code      string  `json:"code,omitempty"`
}

func (p *GTDParams) ToJSON() (string, error) {
	price, slip, err := priceAndSlip(p.Price, p.PriceSlip)
	if err != nil {
		return "", wrapPriceErr(err)
	}
	return marshal(gtdWire{
		p.AccountNo, string(FCOTypeGTD), p.Symbol, string(p.Side), price, slip, p.Quantity,
		p.FromDate, p.ToDate, p.DeviceID, p.UserAgent, p.Code,
	})
}

// StopParams is a stop or stop-limit order. FCOType defaults to stop; for a plain stop the price
// is sent as MTL with no slip.
type StopParams struct {
	AccountNo string
	Symbol    string
	Side      OrderSide
	StopPrice float64
	Operator  FCOOperator
	Quantity  int
	FromDate  string
	ToDate    string
	Price     interface{}
	PriceSlip float64
	FCOType   FCOType
	DeviceID  string
	UserAgent string
	Code      string
}

func (p *StopParams) setIdentity(d, ua string) { p.DeviceID, p.UserAgent = d, ua }
func (p *StopParams) window() (string, string) { return p.FromDate, p.ToDate }

type stopWire struct {
	AccountNo string  `json:"accountNo"`
	Type      string  `json:"type"`
	Symbol    string  `json:"symbol"`
	Side      string  `json:"side"`
	Price     string  `json:"price"`
	PriceSlip float64 `json:"priceSlip"`
	Quantity  int     `json:"quantity"`
	From      string  `json:"from"`
	To        string  `json:"to"`
	StopPrice float64 `json:"stopPrice"`
	Operator  string  `json:"operator"`
	DeviceID  string  `json:"deviceId"`
	UserAgent string  `json:"userAgent"`
	Code      string  `json:"code,omitempty"`
}

func (p *StopParams) ToJSON() (string, error) {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeStop
	}
	price, slip := string(OrderTypeMTL), 0.0
	if fcoType != FCOTypeStop {
		var err error
		if price, err = priceOnly(p.Price); err != nil {
			return "", wrapPriceErr(err)
		}
		slip = p.PriceSlip
	}
	return marshal(stopWire{
		p.AccountNo, string(fcoType), p.Symbol, string(p.Side), price, slip, p.Quantity,
		p.FromDate, p.ToDate, p.StopPrice, string(p.Operator), p.DeviceID, p.UserAgent, p.Code,
	})
}

// TrailingStopParams is a trailing stop or trailing stop-limit order. FCOType defaults to
// trailing_stop, which sends price MTL with no slip.
type TrailingStopParams struct {
	AccountNo      string
	Symbol         string
	Side           OrderSide
	Quantity       int
	ActivePrice    float64
	TrailingAmount float64
	PriceSlip      float64
	FromDate       string
	ToDate         string
	FCOType        FCOType
	DeviceID       string
	UserAgent      string
	Code           string
}

func (p *TrailingStopParams) setIdentity(d, ua string) { p.DeviceID, p.UserAgent = d, ua }
func (p *TrailingStopParams) window() (string, string) { return p.FromDate, p.ToDate }

type trailingWire struct {
	AccountNo      string  `json:"accountNo"`
	Type           string  `json:"type"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	Quantity       int     `json:"quantity"`
	From           string  `json:"from"`
	To             string  `json:"to"`
	ActivePrice    float64 `json:"activePrice"`
	TrailingAmount float64 `json:"trailingAmount"`
	UserAgent      string  `json:"userAgent"`
	DeviceID       string  `json:"deviceId"`
	Price          string  `json:"price,omitempty"`
	PriceSlip      float64 `json:"priceSlip"`
	Code           string  `json:"code,omitempty"`
}

func (p *TrailingStopParams) ToJSON() (string, error) {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeTrailingStop
	}
	w := trailingWire{
		AccountNo: p.AccountNo, Type: string(fcoType), Symbol: p.Symbol, Side: string(p.Side),
		Quantity: p.Quantity, From: p.FromDate, To: p.ToDate, ActivePrice: p.ActivePrice,
		TrailingAmount: p.TrailingAmount, UserAgent: p.UserAgent, DeviceID: p.DeviceID,
		PriceSlip: p.PriceSlip, Code: p.Code,
	}
	if fcoType == FCOTypeTrailingStop {
		w.Price, w.PriceSlip = string(OrderTypeMTL), 0
	}
	return marshal(w)
}

// OCOParams is a one-cancels-the-other order with a take-profit and a stop-loss leg. TPPrice and
// SLPrice may be a number or an OrderType word.
type OCOParams struct {
	AccountNo     string
	Symbol        string
	Quantity      int
	Side          OrderSide
	TPActivePrice float64
	SLActivePrice float64
	TPPrice       interface{}
	SLPrice       interface{}
	TPSlip        float64
	SLSlip        float64
	FromDate      string
	ToDate        string
	FCOType       FCOType
	DeviceID      string
	UserAgent     string
	Code          string
}

func (p *OCOParams) setIdentity(d, ua string) { p.DeviceID, p.UserAgent = d, ua }
func (p *OCOParams) window() (string, string) { return p.FromDate, p.ToDate }

type ocoWire struct {
	AccountNo      string  `json:"accountNo"`
	Type           string  `json:"type"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	Quantity       int     `json:"quantity"`
	From           string  `json:"from"`
	To             string  `json:"to"`
	TPActivePrice  float64 `json:"tpActivePrice"`
	SLActivePrice  float64 `json:"slActivePrice"`
	TPPrice        string  `json:"tpPrice"`
	SLPrice        string  `json:"slPrice"`
	TPSlip         float64 `json:"tpSlip"`
	SLSlip         float64 `json:"slSlip"`
	UserAgent      string  `json:"userAgent"`
	DeviceID       string  `json:"deviceId"`
	Price          string  `json:"price"`
	PriceSlip      int     `json:"priceSlip"`
	StopPrice      int     `json:"stopPrice"`
	ActivePrice    int     `json:"activePrice"`
	TrailingAmount int     `json:"trailingAmount"`
	Operator       string  `json:"operator"`
	Code           string  `json:"code"`
}

func (p *OCOParams) ToJSON() (string, error) {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeOCO
	}
	tp, err := priceOnly(p.TPPrice)
	if err != nil {
		return "", wrapPriceErr(err)
	}
	sl, err := priceOnly(p.SLPrice)
	if err != nil {
		return "", wrapPriceErr(err)
	}
	return marshal(ocoWire{
		AccountNo: p.AccountNo, Type: string(fcoType), Symbol: p.Symbol, Side: string(p.Side),
		Quantity: p.Quantity, From: p.FromDate, To: p.ToDate,
		TPActivePrice: p.TPActivePrice, SLActivePrice: p.SLActivePrice, TPPrice: tp, SLPrice: sl,
		TPSlip: p.TPSlip, SLSlip: p.SLSlip, UserAgent: p.UserAgent, DeviceID: p.DeviceID,
		Price: "MP", Code: p.Code,
	})
}

// BullBearParams is an entry order with take-profit and stop-loss legs. Price, TPPrice and
// SLPrice may be a number or an OrderType word (that leg's slip is then 0).
type BullBearParams struct {
	AccountNo     string
	Symbol        string
	Quantity      int
	Side          OrderSide
	Price         interface{}
	PriceSlip     float64
	TPActivePrice float64
	SLActivePrice float64
	TPPrice       interface{}
	SLPrice       interface{}
	TPSlip        float64
	SLSlip        float64
	FromDate      string
	ToDate        string
	FCOType       FCOType
	DeviceID      string
	UserAgent     string
	Code          string
}

func (p *BullBearParams) setIdentity(d, ua string) { p.DeviceID, p.UserAgent = d, ua }
func (p *BullBearParams) window() (string, string) { return p.FromDate, p.ToDate }

type bullBearWire struct {
	AccountNo     string  `json:"accountNo"`
	Type          string  `json:"type"`
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"`
	Quantity      int     `json:"quantity"`
	Price         string  `json:"price"`
	PriceSlip     float64 `json:"priceSlip"`
	From          string  `json:"from"`
	To            string  `json:"to"`
	TPActivePrice float64 `json:"tpActivePrice"`
	SLActivePrice float64 `json:"slActivePrice"`
	TPPrice       string  `json:"tpPrice"`
	SLPrice       string  `json:"slPrice"`
	TPSlip        float64 `json:"tpSlip"`
	SLSlip        float64 `json:"slSlip"`
	UserAgent     string  `json:"userAgent"`
	DeviceID      string  `json:"deviceId"`
	Code          string  `json:"code,omitempty"`
}

func (p *BullBearParams) ToJSON() (string, error) {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeBullBear
	}
	price, slip, err := priceAndSlip(p.Price, p.PriceSlip)
	if err != nil {
		return "", wrapPriceErr(err)
	}
	tp, tpSlip, err := priceAndSlip(p.TPPrice, p.TPSlip)
	if err != nil {
		return "", wrapPriceErr(err)
	}
	sl, slSlip, err := priceAndSlip(p.SLPrice, p.SLSlip)
	if err != nil {
		return "", wrapPriceErr(err)
	}
	return marshal(bullBearWire{
		AccountNo: p.AccountNo, Type: string(fcoType), Symbol: p.Symbol, Side: string(p.Side),
		Quantity: p.Quantity, Price: price, PriceSlip: slip, From: p.FromDate, To: p.ToDate,
		TPActivePrice: p.TPActivePrice, SLActivePrice: p.SLActivePrice, TPPrice: tp, SLPrice: sl,
		TPSlip: tpSlip, SLSlip: slSlip, UserAgent: p.UserAgent, DeviceID: p.DeviceID, Code: p.Code,
	})
}

// FCOPlaceResponse is the response to placing an FCO.
type FCOPlaceResponse struct {
	FCOID string
}

func fcoIDFromMap(data map[string]interface{}) string {
	if data == nil {
		return ""
	}
	if nested := asMap(data["data"]); nested != nil {
		data = nested
	}
	return util.ToStr(data["fcoId"])
}

func FCOPlaceResponseFromMap(data map[string]interface{}) *FCOPlaceResponse {
	return &FCOPlaceResponse{FCOID: fcoIDFromMap(data)}
}

// FCOCancelRequest is the payload for cancelling an FCO. DeviceID is required by the server (the
// service fills it in); the server records its own IP, so no ipAddress is sent.
type FCOCancelRequest struct {
	FCOID     string
	DeviceID  string
	UserAgent string
	// Code is the one-time code (OTP) when the account requires it; omitted if empty.
	Code string
}

type fcoCancelWire struct {
	FCOID     string `json:"fcoId"`
	DeviceID  string `json:"deviceId"`
	UserAgent string `json:"userAgent,omitempty"`
	Code      string `json:"code,omitempty"`
}

// ToJSON is the exact body that is signed and sent.
func (r *FCOCancelRequest) ToJSON() (string, error) {
	return marshal(fcoCancelWire{r.FCOID, r.DeviceID, r.UserAgent, r.Code})
}

// FCOCancelResponse is the response to cancelling an FCO.
type FCOCancelResponse struct {
	FCOID string
}

func FCOCancelResponseFromMap(data map[string]interface{}) *FCOCancelResponse {
	return &FCOCancelResponse{FCOID: fcoIDFromMap(data)}
}

// FCOStatusHistoryItem is one status transition of an FCO.
type FCOStatusHistoryItem struct {
	State  string
	Time   string
	Code   *string
	Detail string
}

// FCOStatusHistoryFromMap parses the statusHistory response ({"data": [...]}); the server's 204
// marker gives an empty list.
func FCOStatusHistoryFromMap(data map[string]interface{}) []FCOStatusHistoryItem {
	items := []FCOStatusHistoryItem{}
	if data == nil || util.IsNoContent(data) {
		return items
	}
	raw, _ := data["data"].([]interface{})
	for _, r := range raw {
		m, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		item := FCOStatusHistoryItem{
			State:  util.ToStr(m["state"]),
			Time:   util.ToStr(m["time"]),
			Detail: util.ToStr(m["detail"]),
		}
		if code := util.ToStr(m["code"]); code != "" {
			item.Code = &code
		}
		items = append(items, item)
	}
	return items
}
