package trading

import (
	"fmt"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/signature"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var tradingLog = logger.New("ssi_sdk.services.trading")

const epTradingOrder = "/api/v3/trading/order"
const epTradingMaxBuySell = "/api/v3/trading/maxBuySell"

// Service provides trading operations: place/cancel/modify orders.
type Service struct {
	rest       *transport.RestClient
	privateKey string
}

func NewService(rest *transport.RestClient, privateKey string) *Service {
	return &Service{rest: rest, privateKey: privateKey}
}

func (s *Service) placeOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType) (*PlaceOrderResponse, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(string(side), "Order side must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(string(orderType), "Order type must be provided"); err != nil {
		return nil, err
	}

	req := &PlaceOrderRequest{
		AccountNo:       accountNo,
		Symbol:          symbol,
		Side:            side,
		Quantity:        quantity,
		Price:           price,
		OrderType:       orderType,
		ClientRequestID: util.GenerateRequestID(),
		DeviceID:        "A1:B2:C3:D4:E5:F6",
		UserAgent:       "SSI Go SDK/" + ssi.Version,
	}

	orderJSON := req.ToJSON()
	sig, err := signature.Sign(orderJSON, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign order request: %w", err)
	}
	s.rest.SetSignatureHeader(sig)

	data, err := s.rest.Post(epTradingOrder, nil, []byte(orderJSON), s.rest.GetHeaders())
	if err != nil {
		tradingLog.Error("Error placing order: %v", err)
		return nil, err
	}

	return PlaceOrderResponseFromMap(data), nil
}

func (s *Service) PlaceOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType) (*PlaceOrderResponse, error) {
	if err := ssi.RequireNonNegative(price, "Price must be positive"); err != nil {
		return nil, err
	}
	return s.placeOrder(accountNo, symbol, side, quantity, price, orderType)
}

func (s *Service) PlaceLimitOrder(accountNo, symbol string, side OrderSide, quantity int, price float64) (*PlaceOrderResponse, error) {
	if err := ssi.RequirePositive(price, "Price must be positive"); err != nil {
		return nil, err
	}
	return s.PlaceOrder(accountNo, symbol, side, quantity, price, OrderTypeLO)
}

func (s *Service) PlaceMarketOrder(accountNo, symbol string, side OrderSide, quantity int) (*PlaceOrderResponse, error) {
	return s.PlaceOrder(accountNo, symbol, side, quantity, 0, OrderTypeMTL)
}

func (s *Service) PlaceATOOrder(accountNo, symbol string, side OrderSide, quantity int) (*PlaceOrderResponse, error) {
	return s.PlaceOrder(accountNo, symbol, side, quantity, 0, OrderTypeATO)
}

func (s *Service) PlaceATCOrder(accountNo, symbol string, side OrderSide, quantity int) (*PlaceOrderResponse, error) {
	return s.PlaceOrder(accountNo, symbol, side, quantity, 0, OrderTypeATC)
}

func (s *Service) modifyOrder(accountNo string, orderID, clientRequestID string, price *float64, quantity *int) (*ModifyOrderResponse, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}

	req := &ModifyOrderRequest{
		AccountNo:       accountNo,
		Quantity:        quantity,
		Price:           price,
		OrderID:         orderID,
		ClientRequestID: clientRequestID,
		ClientModifyID:  util.GenerateRequestID(),
		DeviceID:        "A1:B2:C3:D4:E5:F6",
		UserAgent:       "SSI Go SDK/" + ssi.Version,
	}

	modifyJSON := req.ToJSON()
	sig, err := signature.Sign(modifyJSON, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign modify request: %w", err)
	}
	s.rest.SetSignatureHeader(sig)

	data, err := s.rest.Put(epTradingOrder, nil, []byte(modifyJSON), s.rest.GetHeaders())
	if err != nil {
		tradingLog.Error("Error modifying order: %v", err)
		return nil, err
	}

	return ModifyOrderResponseFromMap(data), nil
}

func (s *Service) ModifyOrderPrice(accountNo, clientRequestID string, price float64) (*ModifyOrderResponse, error) {
	return s.modifyOrder(accountNo, "", clientRequestID, &price, nil)
}

func (s *Service) ModifyOrderPriceByOrderID(accountNo, orderID string, price float64) (*ModifyOrderResponse, error) {
	return s.modifyOrder(accountNo, orderID, "", &price, nil)
}

func (s *Service) ModifyOrderQuantity(accountNo, clientRequestID string, quantity int) (*ModifyOrderResponse, error) {
	return s.modifyOrder(accountNo, "", clientRequestID, nil, &quantity)
}

func (s *Service) ModifyOrderQuantityByOrderID(accountNo, orderID string, quantity int) (*ModifyOrderResponse, error) {
	return s.modifyOrder(accountNo, orderID, "", nil, &quantity)
}

func (s *Service) cancelOrder(accountNo, orderID, clientRequestID string) (*CancelOrderResponse, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}

	req := &CancelOrderRequest{
		AccountNo:       accountNo,
		OrderID:         orderID,
		ClientRequestID: clientRequestID,
		ClientCancelID:  util.GenerateRequestID(),
		DeviceID:        "A1:B2:C3:D4:E5:F6",
		UserAgent:       "SSI Go SDK/" + ssi.Version,
	}

	cancelJSON := req.ToJSON()
	sig, err := signature.Sign(cancelJSON, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign cancel request: %w", err)
	}
	s.rest.SetSignatureHeader(sig)

	data, err := s.rest.Delete(epTradingOrder, nil, []byte(cancelJSON), s.rest.GetHeaders())
	if err != nil {
		tradingLog.Error("Error canceling order: %v", err)
		return nil, err
	}

	return CancelOrderResponseFromMap(data), nil
}

func (s *Service) CancelOrder(accountNo, clientRequestID string) (*CancelOrderResponse, error) {
	if err := ssi.RequireNonEmpty(clientRequestID, "Client request ID must be provided"); err != nil {
		return nil, err
	}
	return s.cancelOrder(accountNo, "", clientRequestID)
}

func (s *Service) CancelOrderByOrderID(accountNo, orderID string) (*CancelOrderResponse, error) {
	if err := ssi.RequireNonEmpty(orderID, "Order ID must be provided"); err != nil {
		return nil, err
	}
	return s.cancelOrder(accountNo, orderID, "")
}

func (s *Service) GetMaxBuySell(accountNo, symbol string, price *float64) (*MaxBuySellResponse, error) {
	return s.getMaxBuySell(accountNo, symbol, price)
}

func (s *Service) GetMaxBuySellAtMarketPrice(accountNo, symbol string) (*MaxBuySellResponse, error) {
	return s.getMaxBuySell(accountNo, symbol, nil)
}

func (s *Service) getMaxBuySell(accountNo, symbol string, price *float64) (*MaxBuySellResponse, error) {
	if err := ssi.RequireNonEmpty(accountNo, "Account number must be provided"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(symbol, "Symbol must be provided"); err != nil {
		return nil, err
	}

	req := &MaxBuySellRequest{
		AccountNo: accountNo,
		Symbol:    symbol,
	}
	if price != nil {
		req.Price = *price
	}

	data, err := s.rest.Get(epTradingMaxBuySell, req.ToMap(), s.rest.GetHeaders())
	if err != nil {
		tradingLog.Error("Error getting max buy/sell: %v", err)
		return nil, err
	}

	return MaxBuySellResponseFromMap(data, symbol), nil
}
