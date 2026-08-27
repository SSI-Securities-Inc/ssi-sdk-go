package trading

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
)

// PlaceOrderRequest is the payload for placing a new order.
type PlaceOrderRequest struct {
	AccountNo       string    `json:"accountNo"`
	Symbol          string    `json:"symbol"`
	Side            OrderSide `json:"side"`
	Quantity        int       `json:"quantity"`
	Price           float64   `json:"price"`
	OrderType       OrderType `json:"orderType"`
	ClientRequestID string    `json:"clientRequestId"`
	DeviceID        string    `json:"deviceId"`
	UserAgent       string    `json:"userAgent"`
}

func (r *PlaceOrderRequest) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"accountNo":       r.AccountNo,
		"symbol":          r.Symbol,
		"side":            string(r.Side),
		"quantity":        r.Quantity,
		"price":           fmt.Sprintf("%v", r.Price),
		"orderType":       string(r.OrderType),
		"clientRequestId": r.ClientRequestID,
		"deviceId":        r.DeviceID,
		"userAgent":       r.UserAgent,
	}
}

func (r *PlaceOrderRequest) ToJSON() string {
	b, _ := json.Marshal(r.ToMap())
	return string(b)
}

// PlaceOrderResponse is the response for placing a new order.
type PlaceOrderResponse struct {
	OrderID         string      `json:"orderId"`
	ClientRequestID string      `json:"clientRequestId"`
	Status          OrderStatus `json:"orderStatus"`
}

func PlaceOrderResponseFromMap(data map[string]interface{}) *PlaceOrderResponse {
	return &PlaceOrderResponse{
		OrderID:         util.ToStr(data["orderId"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		Status:          OrderStatus(util.ToStr(data["orderStatus"])),
	}
}

// ModifyOrderRequest is the modify order payload.
type ModifyOrderRequest struct {
	AccountNo       string   `json:"accountNo"`
	Quantity        *int     `json:"quantity,omitempty"`
	Price           *float64 `json:"price,omitempty"`
	OrderID         string   `json:"orderId,omitempty"`
	ClientModifyID  string   `json:"clientModifyId"`
	ClientRequestID string   `json:"clientRequestId,omitempty"`
	DeviceID        string   `json:"deviceId"`
	UserAgent       string   `json:"userAgent"`
}

func (r *ModifyOrderRequest) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"accountNo":      r.AccountNo,
		"clientModifyId": r.ClientModifyID,
		"deviceId":       r.DeviceID,
		"userAgent":      r.UserAgent,
	}
	if r.OrderID != "" {
		result["orderId"] = r.OrderID
	}
	if r.ClientRequestID != "" {
		result["clientRequestId"] = r.ClientRequestID
	}
	if r.Quantity != nil {
		result["quantity"] = *r.Quantity
	}
	if r.Price != nil {
		result["price"] = fmt.Sprintf("%v", *r.Price)
	}
	return result
}

func (r *ModifyOrderRequest) ToJSON() string {
	b, _ := json.Marshal(r.ToMap())
	return string(b)
}

// ModifyOrderResponse is the response for modifying an order.
type ModifyOrderResponse struct {
	ClientModifyID  string      `json:"clientModifyId"`
	OrderID         string      `json:"orderId"`
	ClientRequestID string      `json:"clientRequestId"`
	Status          OrderStatus `json:"orderStatus"`
}

func ModifyOrderResponseFromMap(data map[string]interface{}) *ModifyOrderResponse {
	return &ModifyOrderResponse{
		ClientModifyID:  util.ToStr(data["clientModifyId"]),
		OrderID:         util.ToStr(data["orderId"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		Status:          OrderStatus(util.ToStr(data["orderStatus"])),
	}
}

// CancelOrderRequest is the cancel order payload.
type CancelOrderRequest struct {
	AccountNo       string `json:"accountNo"`
	OrderID         string `json:"orderId,omitempty"`
	ClientRequestID string `json:"clientRequestId,omitempty"`
	ClientCancelID  string `json:"clientCancelId"`
	DeviceID        string `json:"deviceId"`
	UserAgent       string `json:"userAgent"`
}

func (r *CancelOrderRequest) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"accountNo":      r.AccountNo,
		"clientCancelId": r.ClientCancelID,
		"deviceId":       r.DeviceID,
		"userAgent":      r.UserAgent,
	}
	if r.OrderID != "" {
		result["orderId"] = r.OrderID
	}
	if r.ClientRequestID != "" {
		result["clientRequestId"] = r.ClientRequestID
	}
	return result
}

func (r *CancelOrderRequest) ToJSON() string {
	b, _ := json.Marshal(r.ToMap())
	return string(b)
}

// CancelOrderResponse is the response for canceling an order.
type CancelOrderResponse struct {
	ClientCancelID  string      `json:"clientCancelId"`
	OrderID         string      `json:"orderId"`
	ClientRequestID string      `json:"clientRequestId"`
	Status          OrderStatus `json:"orderStatus"`
}

func CancelOrderResponseFromMap(data map[string]interface{}) *CancelOrderResponse {
	return &CancelOrderResponse{
		ClientCancelID:  util.ToStr(data["clientCancelId"]),
		OrderID:         util.ToStr(data["orderId"]),
		ClientRequestID: util.ToStr(data["clientRequestId"]),
		Status:          OrderStatus(util.ToStr(data["orderStatus"])),
	}
}

// MaxBuySellRequest is the payload for max buy/sell request.
type MaxBuySellRequest struct {
	AccountNo string  `json:"accountNo"`
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price,omitempty"`
}

func (r *MaxBuySellRequest) ToMap() map[string]string {
	result := map[string]string{
		"accountNo": r.AccountNo,
		"symbol":    strings.ToUpper(r.Symbol),
	}
	if r.Price > 0 {
		result["price"] = fmt.Sprintf("%v", r.Price)
	}
	return result
}

// MaxBuySellResponse is the response for max buy/sell request.
type MaxBuySellResponse struct {
	AccountNo       string `json:"accountNo"`
	Symbol          string `json:"symbol"`
	MaxBuyQuantity  int    `json:"maxBuyQty"`
	MaxSellQuantity int    `json:"maxSellQty"`
	MarginRatio     string `json:"marginRatio"`
	PurchasePower   string `json:"purchasePower"`
}

func MaxBuySellResponseFromMap(data map[string]interface{}, symbol string) *MaxBuySellResponse {
	return &MaxBuySellResponse{
		AccountNo:       util.ToStr(data["accountNo"]),
		Symbol:          strings.ToUpper(symbol),
		MaxBuyQuantity:  util.ToInt(data["maxBuyQty"]),
		MaxSellQuantity: util.ToInt(data["maxSellQty"]),
		MarginRatio:     util.ToStr(data["marginRatio"]),
		PurchasePower:   util.ToStr(data["purchasePower"]),
	}
}

// ---------------------------------------------------------------------------
// FCO Models & Mappers
// ---------------------------------------------------------------------------

const (
	DefaultDeviceID  = "A1:B2:C3:D4:E5:F6"
	DefaultUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type FCOParams struct {
	StopPrice      float64     `json:"stopPrice,omitempty"`
	Side           OrderSide   `json:"side,omitempty"`
	ActivePrice    float64     `json:"activePrice,omitempty"`
	TrailingAmount float64     `json:"trailingAmount,omitempty"`
	TPActivePrice  float64     `json:"tpActivePrice,omitempty"`
	SLActivePrice  float64     `json:"slActivePrice,omitempty"`
	TPPrice        string      `json:"tpPrice,omitempty"`
	SLPrice        string      `json:"slPrice,omitempty"`
	TPSlip         float64     `json:"tpSlip,omitempty"`
	SLSlip         float64     `json:"slSlip,omitempty"`
	Operator       FCOOperator `json:"operator,omitempty"`
}

func FCOParamsFromMap(data map[string]interface{}) *FCOParams {
	if data == nil {
		return nil
	}
	return &FCOParams{
		StopPrice:      util.ToFloat64(data["stopPrice"]),
		Side:           OrderSide(util.ToStr(data["side"])),
		ActivePrice:    util.ToFloat64(data["activePrice"]),
		TrailingAmount: util.ToFloat64(data["trailingAmount"]),
		TPActivePrice:  util.ToFloat64(data["tpActivePrice"]),
		SLActivePrice:  util.ToFloat64(data["slActivePrice"]),
		TPPrice:        util.ToStr(data["tpPrice"]),
		SLPrice:        util.ToStr(data["slPrice"]),
		TPSlip:         util.ToFloat64(data["tpSlip"]),
		SLSlip:         util.ToFloat64(data["slSlip"]),
		Operator:       FCOOperator(util.ToStr(data["operator"])),
	}
}

type FCOInfo struct {
	FCOID           string     `json:"fcoId"`
	ClientID        string     `json:"clientId"`
	AccountNo       string     `json:"accountNo"`
	Quantity        int        `json:"quantity"`
	Price           string     `json:"price"`
	PriceSlip       float64    `json:"priceSlip"`
	Symbol          string     `json:"symbol"`
	Type            FCOType    `json:"type"`
	FromDate        string     `json:"from"`
	ToDate          string     `json:"to"`
	MatchedQuantity int        `json:"matchedQuantity"`
	IsPlaceOrder    bool       `json:"isPlaceOrder"`
	Status          FCOStatus  `json:"status"`
	Detail          string     `json:"detail"`
	Params          *FCOParams `json:"params"`
}

func FCOInfoFromMap(data map[string]interface{}) *FCOInfo {
	paramsRaw, _ := data["params"].(map[string]interface{})
	if paramsRaw == nil {
		paramsRaw, _ = data["fcoParams"].(map[string]interface{})
	}
	if paramsRaw == nil {
		paramsRaw, _ = data["fco_params"].(map[string]interface{})
	}

	return &FCOInfo{
		FCOID:           util.ToStr(data["fcoId"]),
		ClientID:        util.ToStr(data["username"]),
		AccountNo:       util.ToStr(data["accountNo"]),
		Quantity:        util.ToInt(data["quantity"]),
		Price:           util.ToStr(data["price"]),
		PriceSlip:       util.ToFloat64(data["priceSlip"]),
		Symbol:          util.ToStr(data["symbol"]),
		Type:            FCOType(util.ToStr(data["type"])),
		FromDate:        util.ToStr(data["from"]),
		ToDate:          util.ToStr(data["to"]),
		MatchedQuantity: util.ToInt(data["matchedQuantity"]),
		IsPlaceOrder:    util.ToBool(data["isPlaceOrder"]),
		Status:          FCOStatus(util.ToStr(data["status"])),
		Detail:          util.ToStr(data["detail"]),
		Params:          FCOParamsFromMap(paramsRaw),
	}
}

func FCOInfoSliceFromList(raw interface{}) []*FCOInfo {
	list, ok := raw.([]interface{})
	if !ok {
		return []*FCOInfo{}
	}
	result := make([]*FCOInfo, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, FCOInfoFromMap(m))
		}
	}
	return result
}

type FCOListRequest struct {
	AccountNo     string    `json:"accountNo"`
	FCOID         string    `json:"fcoId,omitempty"`
	Type          FCOType   `json:"type,omitempty"`
	ProcessStatus FCOStatus `json:"processStatus,omitempty"`
	Symbol        string    `json:"symbol,omitempty"`
	Side          OrderSide `json:"side,omitempty"`
	FromDate      string    `json:"from,omitempty"`
	ToDate        string    `json:"to,omitempty"`
	PageIndex     int       `json:"pageIndex,omitempty"`
	PageSize      int       `json:"pageSize,omitempty"`
}

func (r *FCOListRequest) ToMap() map[string]string {
	params := map[string]string{
		"accountNo": r.AccountNo,
	}
	if r.FCOID != "" {
		params["fcoId"] = r.FCOID
	}
	if r.Type != "" {
		params["type"] = string(r.Type)
	}
	if r.ProcessStatus != "" {
		params["processStatus"] = string(r.ProcessStatus)
	}
	if r.Symbol != "" {
		params["symbol"] = r.Symbol
	}
	if r.Side != "" {
		params["side"] = string(r.Side)
	}
	if r.FromDate != "" {
		params["from"] = r.FromDate
	}
	if r.ToDate != "" {
		params["to"] = r.ToDate
	}
	if r.PageIndex > 0 {
		params["pageIndex"] = strconv.Itoa(r.PageIndex)
	}
	if r.PageSize > 0 {
		params["pageSize"] = strconv.Itoa(r.PageSize)
	}
	return params
}

type FCOListResponse struct {
	PageIndex  int        `json:"pageIndex"`
	PageSize   int        `json:"pageSize"`
	ItemsCount int        `json:"itemsCount"`
	PagesCount int        `json:"pagesCount"`
	FCOList    []*FCOInfo `json:"fcoList"`
}

func FCOListResponseFromMap(data map[string]interface{}) *FCOListResponse {
	if data == nil {
		return &FCOListResponse{PageIndex: 1, PageSize: 10, FCOList: []*FCOInfo{}}
	}
	listRaw := data["data"]
	if listRaw == nil {
		listRaw = data["fcoList"]
	}
	if listRaw == nil {
		listRaw = data["fco_list"]
	}

	return &FCOListResponse{
		PageIndex:  util.ToInt(data["pageIndex"]),
		PageSize:   util.ToInt(data["pageSize"]),
		ItemsCount: util.ToInt(data["itemsCount"]),
		PagesCount: util.ToInt(data["pagesCount"]),
		FCOList:    FCOInfoSliceFromList(listRaw),
	}
}

type FCOOrderBookRequest struct {
	FCOID     string `json:"fcoId"`
	PageIndex int    `json:"pageIndex"`
	PageSize  int    `json:"pageSize"`
}

func (r *FCOOrderBookRequest) ToMap() map[string]string {
	params := map[string]string{
		"fcoId": r.FCOID,
	}
	if r.PageIndex > 0 {
		params["pageIndex"] = strconv.Itoa(r.PageIndex)
	}
	if r.PageSize > 0 {
		params["pageSize"] = strconv.Itoa(r.PageSize)
	}
	return params
}

type FCOOrder struct {
	FCOID           string      `json:"fcoId"`
	AccountNo       string      `json:"accountNo"`
	Quantity        float64     `json:"quantity"`
	Price           string      `json:"price"`
	Symbol          string      `json:"symbol"`
	Side            OrderSide   `json:"side"`
	OrderType       OrderType   `json:"orderType"`
	IsMainOrder     bool        `json:"isMainOrder"`
	IsAttachedOrder bool        `json:"isAttachedOrder"`
	CreatedTime     string      `json:"createdTime"`
	UpdatedTime     string      `json:"updatedTime"`
	UniqueID        string      `json:"uniqueId"`
	OrderID         string      `json:"orderId"`
	MatchedQuantity float64     `json:"matchedQuantity"`
	OSQuantity      float64     `json:"osQuantity"`
	AvgPrice        float64     `json:"avgPrice"`
	Status          OrderStatus `json:"status"`
	Detail          string      `json:"detail"`
}

func FCOOrderFromMap(data map[string]interface{}) *FCOOrder {
	return &FCOOrder{
		FCOID:           util.ToStr(data["fcoId"]),
		AccountNo:       util.ToStr(data["accountNo"]),
		Quantity:        util.ToFloat64(data["quantity"]),
		Price:           util.ToStr(data["price"]),
		Symbol:          util.ToStr(data["symbol"]),
		Side:            OrderSide(util.ToStr(data["side"])),
		OrderType:       OrderType(util.ToStr(data["orderType"])),
		IsMainOrder:     util.ToBool(data["isMainOrder"]),
		IsAttachedOrder: util.ToBool(data["isAttachedOrder"]),
		CreatedTime:     util.ToStr(data["createdTime"]),
		UpdatedTime:     util.ToStr(data["updatedTime"]),
		UniqueID:        util.ToStr(data["uniqueId"]),
		OrderID:         util.ToStr(data["orderId"]),
		MatchedQuantity: util.ToFloat64(data["matchedQuantity"]),
		OSQuantity:      util.ToFloat64(data["osQuantity"]),
		AvgPrice:        util.ToFloat64(data["avgPrice"]),
		Status:          OrderStatus(util.ToStr(data["status"])),
		Detail:          util.ToStr(data["detail"]),
	}
}

func FCOOrderSliceFromList(raw interface{}) []*FCOOrder {
	list, ok := raw.([]interface{})
	if !ok {
		return []*FCOOrder{}
	}
	result := make([]*FCOOrder, 0, len(list))
	for _, item := range list {
		if m, ok := item.(map[string]interface{}); ok {
			result = append(result, FCOOrderFromMap(m))
		}
	}
	return result
}

type FCOOrderBookResponse struct {
	PageIndex  int         `json:"pageIndex"`
	PageSize   int         `json:"pageSize"`
	ItemsCount int         `json:"itemsCount"`
	PagesCount int         `json:"pagesCount"`
	OrderBook  []*FCOOrder `json:"orderBook"`
}

func FCOOrderBookResponseFromMap(data map[string]interface{}) *FCOOrderBookResponse {
	if data == nil {
		return &FCOOrderBookResponse{PageIndex: 1, PageSize: 10, OrderBook: []*FCOOrder{}}
	}
	listRaw := data["data"]
	if listRaw == nil {
		listRaw = data["orderBook"]
	}
	if listRaw == nil {
		listRaw = data["order_book"]
	}

	return &FCOOrderBookResponse{
		PageIndex:  util.ToInt(data["pageIndex"]),
		PageSize:   util.ToInt(data["pageSize"]),
		ItemsCount: util.ToInt(data["itemsCount"]),
		PagesCount: util.ToInt(data["pagesCount"]),
		OrderBook:  FCOOrderSliceFromList(listRaw),
	}
}

type GTDParams struct {
	AccountNo string      `json:"accountNo"`
	Symbol    string      `json:"symbol"`
	Side      OrderSide   `json:"side"`
	Price     interface{} `json:"price"`
	PriceSlip float64     `json:"priceSlip"`
	Quantity  int         `json:"quantity"`
	FromDate  string      `json:"from"`
	ToDate    string      `json:"to"`
	DeviceID  string      `json:"deviceId"`
	UserAgent string      `json:"userAgent"`
}

func (p *GTDParams) ToMap() map[string]interface{} {
	var price string
	var slip = p.PriceSlip

	switch v := p.Price.(type) {
	case int, int64, float64:
		price = fmt.Sprintf("%v", v)
	case string:
		price = v
	case OrderType:
		price = string(v)
		slip = 0
	default:
		price = fmt.Sprintf("%v", p.Price)
	}

	deviceID := p.DeviceID
	if deviceID == "" {
		deviceID = DefaultDeviceID
	}
	userAgent := p.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	return map[string]interface{}{
		"accountNo": p.AccountNo,
		"type":      string(FCOTypeGTD),
		"symbol":    p.Symbol,
		"side":      string(p.Side),
		"price":     price,
		"priceSlip": slip,
		"quantity":  p.Quantity,
		"from":      p.FromDate,
		"to":        p.ToDate,
		"deviceId":  deviceID,
		"userAgent": userAgent,
	}
}

func (p *GTDParams) ToJSON() string {
	b, _ := json.Marshal(p.ToMap())
	return string(b)
}

type StopParams struct {
	AccountNo string      `json:"accountNo"`
	Symbol    string      `json:"symbol"`
	Side      OrderSide   `json:"side"`
	StopPrice float64     `json:"stopPrice"`
	Operator  FCOOperator `json:"operator"`
	Quantity  int         `json:"quantity"`
	FromDate  string      `json:"from"`
	ToDate    string      `json:"to"`
	Price     interface{} `json:"price,omitempty"`
	PriceSlip float64     `json:"priceSlip,omitempty"`
	FCOType   FCOType     `json:"type,omitempty"`
	DeviceID  string      `json:"deviceId"`
	UserAgent string      `json:"userAgent"`
}

func (p *StopParams) ToMap() map[string]interface{} {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeStop
	}

	var price string
	var slip float64
	if fcoType == FCOTypeStop {
		price = string(OrderTypeMTL)
		slip = 0
	} else {
		price = fmt.Sprintf("%v", p.Price)
		slip = p.PriceSlip
	}

	deviceID := p.DeviceID
	if deviceID == "" {
		deviceID = DefaultDeviceID
	}
	userAgent := p.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	return map[string]interface{}{
		"accountNo": p.AccountNo,
		"type":      string(fcoType),
		"symbol":    p.Symbol,
		"side":      string(p.Side),
		"price":     price,
		"priceSlip": slip,
		"quantity":  p.Quantity,
		"from":      p.FromDate,
		"to":        p.ToDate,
		"stopPrice": p.StopPrice,
		"operator":  string(p.Operator),
		"deviceId":  deviceID,
		"userAgent": userAgent,
	}
}

func (p *StopParams) ToJSON() string {
	b, _ := json.Marshal(p.ToMap())
	return string(b)
}

type TrailingStopParams struct {
	AccountNo      string    `json:"accountNo"`
	Symbol         string    `json:"symbol"`
	Side           OrderSide `json:"side"`
	Quantity       int       `json:"quantity"`
	ActivePrice    float64   `json:"activePrice"`
	TrailingAmount float64   `json:"trailingAmount"`
	PriceSlip      float64   `json:"priceSlip,omitempty"`
	FromDate       string    `json:"from"`
	ToDate         string    `json:"to"`
	FCOType        FCOType   `json:"type,omitempty"`
	DeviceID       string    `json:"deviceId"`
	UserAgent      string    `json:"userAgent"`
}

func (p *TrailingStopParams) ToMap() map[string]interface{} {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeTrailingStop
	}

	deviceID := p.DeviceID
	if deviceID == "" {
		deviceID = DefaultDeviceID
	}
	userAgent := p.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	res := map[string]interface{}{
		"accountNo":      p.AccountNo,
		"type":           string(fcoType),
		"symbol":         p.Symbol,
		"side":           string(p.Side),
		"quantity":       p.Quantity,
		"from":           p.FromDate,
		"to":             p.ToDate,
		"activePrice":    p.ActivePrice,
		"trailingAmount": p.TrailingAmount,
		"deviceId":       deviceID,
		"userAgent":      userAgent,
	}

	if fcoType == FCOTypeTrailingStop {
		res["price"] = string(OrderTypeMTL)
		res["priceSlip"] = 0
	} else {
		res["priceSlip"] = p.PriceSlip
	}

	return res
}

func (p *TrailingStopParams) ToJSON() string {
	b, _ := json.Marshal(p.ToMap())
	return string(b)
}

type OCOParams struct {
	AccountNo     string      `json:"accountNo"`
	Symbol        string      `json:"symbol"`
	Quantity      int         `json:"quantity"`
	Side          OrderSide   `json:"side"`
	TPActivePrice float64     `json:"tpActivePrice"`
	SLActivePrice float64     `json:"slActivePrice"`
	TPPrice       interface{} `json:"tpPrice"`
	SLPrice       interface{} `json:"slPrice"`
	TPSlip        float64     `json:"tpSlip"`
	SLSlip        float64     `json:"slSlip"`
	FromDate      string      `json:"from"`
	ToDate        string      `json:"to"`
	FCOType       FCOType     `json:"type,omitempty"`
	DeviceID      string      `json:"deviceId"`
	UserAgent     string      `json:"userAgent"`
}

func (p *OCOParams) ToMap() map[string]interface{} {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeOCO
	}

	tpPriceStr := fmt.Sprintf("%v", p.TPPrice)
	slPriceStr := fmt.Sprintf("%v", p.SLPrice)

	deviceID := p.DeviceID
	if deviceID == "" {
		deviceID = DefaultDeviceID
	}
	userAgent := p.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	return map[string]interface{}{
		"accountNo":      p.AccountNo,
		"type":           string(fcoType),
		"symbol":         p.Symbol,
		"side":           string(p.Side),
		"quantity":       p.Quantity,
		"from":           p.FromDate,
		"to":             p.ToDate,
		"tpActivePrice":  p.TPActivePrice,
		"slActivePrice":  p.SLActivePrice,
		"tpPrice":        tpPriceStr,
		"slPrice":        slPriceStr,
		"tpSlip":         p.TPSlip,
		"slSlip":         p.SLSlip,
		"deviceId":       deviceID,
		"userAgent":      userAgent,
		"price":          "MP",
		"priceSlip":      0,
		"stopPrice":      0,
		"activePrice":    0,
		"trailingAmount": 0,
		"operator":       "",
		"code":           "",
	}
}

func (p *OCOParams) ToJSON() string {
	b, _ := json.Marshal(p.ToMap())
	return string(b)
}

type BullBearParams struct {
	AccountNo     string      `json:"accountNo"`
	Symbol        string      `json:"symbol"`
	Quantity      int         `json:"quantity"`
	Side          OrderSide   `json:"side"`
	Price         interface{} `json:"price"`
	PriceSlip     float64     `json:"priceSlip"`
	TPActivePrice float64     `json:"tpActivePrice"`
	SLActivePrice float64     `json:"slActivePrice"`
	TPPrice       interface{} `json:"tpPrice"`
	SLPrice       interface{} `json:"slPrice"`
	TPSlip        float64     `json:"tpSlip"`
	SLSlip        float64     `json:"slSlip"`
	FromDate      string      `json:"from"`
	ToDate        string      `json:"to"`
	FCOType       FCOType     `json:"type,omitempty"`
	DeviceID      string      `json:"deviceId"`
	UserAgent     string      `json:"userAgent"`
}

func (p *BullBearParams) ToMap() map[string]interface{} {
	fcoType := p.FCOType
	if fcoType == "" {
		fcoType = FCOTypeBullBear
	}

	priceStr := fmt.Sprintf("%v", p.Price)
	tpPriceStr := fmt.Sprintf("%v", p.TPPrice)
	slPriceStr := fmt.Sprintf("%v", p.SLPrice)

	deviceID := p.DeviceID
	if deviceID == "" {
		deviceID = DefaultDeviceID
	}
	userAgent := p.UserAgent
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}

	return map[string]interface{}{
		"accountNo":     p.AccountNo,
		"type":          string(fcoType),
		"symbol":        p.Symbol,
		"side":          string(p.Side),
		"quantity":      p.Quantity,
		"price":         priceStr,
		"priceSlip":     p.PriceSlip,
		"from":          p.FromDate,
		"to":            p.ToDate,
		"tpActivePrice": p.TPActivePrice,
		"slActivePrice": p.SLActivePrice,
		"tpPrice":       tpPriceStr,
		"slPrice":       slPriceStr,
		"tpSlip":        p.TPSlip,
		"slSlip":        p.SLSlip,
		"deviceId":      deviceID,
		"userAgent":     userAgent,
	}
}

func (p *BullBearParams) ToJSON() string {
	b, _ := json.Marshal(p.ToMap())
	return string(b)
}

type FCOPlaceResponse struct {
	FCOID string `json:"fcoId"`
}

func FCOPlaceResponseFromMap(data map[string]interface{}) *FCOPlaceResponse {
	if data == nil {
		return &FCOPlaceResponse{}
	}
	respData, _ := data["data"].(map[string]interface{})
	if respData == nil {
		respData = data
	}
	return &FCOPlaceResponse{
		FCOID: util.ToStr(respData["fcoId"]),
	}
}

type FCOCancelResponse struct {
	FCOID string `json:"fcoId"`
}

func FCOCancelResponseFromMap(data map[string]interface{}) *FCOCancelResponse {
	if data == nil {
		return &FCOCancelResponse{}
	}
	respData, _ := data["data"].(map[string]interface{})
	if respData == nil {
		respData = data
	}
	return &FCOCancelResponse{
		FCOID: util.ToStr(respData["fcoId"]),
	}
}
