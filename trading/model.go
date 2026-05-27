package trading

import (
	"encoding/json"
	"fmt"
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
