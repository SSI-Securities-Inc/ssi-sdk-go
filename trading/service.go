package trading

import (
	"encoding/json"
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
const epTradingFcoOrder = "/api/v3/trading/fco/order"
const epTradingFcoList = "/api/v3/trading/fco/list"
const epTradingFcoOrderBook = "/api/v3/trading/fco/orderbook"

// Service provides trading operations: place/cancel/modify orders and FCO orders.
type Service struct {
	rest       *transport.RestClient
	privateKey string
}


func NewService(rest *transport.RestClient, privateKey string) *Service {
	return &Service{rest: rest, privateKey: privateKey}
}

func (s *Service) placeOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType) (*PlaceOrderResponse, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(symbol, "symbol"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(string(side), "side"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(string(orderType), "orderType"); err != nil {
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

	headers := map[string]string{transport.HeaderSignature: sig}
	data, err := s.rest.Post(epTradingOrder, nil, []byte(orderJSON), headers)
	if err != nil {
		tradingLog.Error("Error placing order: %v", err)
		return nil, err
	}

	return PlaceOrderResponseFromMap(data), nil
}

func (s *Service) PlaceOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType) (*PlaceOrderResponse, error) {
	if err := ssi.RequireNonNegative(price, "price"); err != nil {
		return nil, err
	}
	return s.placeOrder(accountNo, symbol, side, quantity, price, orderType)
}

func (s *Service) PlaceLimitOrder(accountNo, symbol string, side OrderSide, quantity int, price float64) (*PlaceOrderResponse, error) {
	if err := ssi.RequirePositive(price, "price"); err != nil {
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
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
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

	headers := map[string]string{transport.HeaderSignature: sig}
	data, err := s.rest.Put(epTradingOrder, nil, []byte(modifyJSON), headers)
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
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
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

	headers := map[string]string{transport.HeaderSignature: sig}
	data, err := s.rest.Delete(epTradingOrder, nil, []byte(cancelJSON), headers)
	if err != nil {
		tradingLog.Error("Error canceling order: %v", err)
		return nil, err
	}

	return CancelOrderResponseFromMap(data), nil
}

func (s *Service) CancelOrder(accountNo, clientRequestID string) (*CancelOrderResponse, error) {
	if err := ssi.RequireNonEmpty(clientRequestID, "clientRequestId"); err != nil {
		return nil, err
	}
	return s.cancelOrder(accountNo, "", clientRequestID)
}

func (s *Service) CancelOrderByOrderID(accountNo, orderID string) (*CancelOrderResponse, error) {
	if err := ssi.RequireNonEmpty(orderID, "orderId"); err != nil {
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
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(symbol, "symbol"); err != nil {
		return nil, err
	}

	req := &MaxBuySellRequest{
		AccountNo: accountNo,
		Symbol:    symbol,
	}
	if price != nil {
		req.Price = *price
	}

	data, err := s.rest.Get(epTradingMaxBuySell, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting max buy/sell: %v", err)
		return nil, err
	}

	return MaxBuySellResponseFromMap(data, symbol), nil
}

// ---------------------------------------------------------------------------
// Flexible Conditional Orders (FCO)
// ---------------------------------------------------------------------------

func (s *Service) placeFcoOrder(payloadJSON string) (*FCOPlaceResponse, error) {
	sig, err := signature.Sign(payloadJSON, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign FCO request: %w", err)
	}

	headers := map[string]string{transport.HeaderSignature: sig}
	data, err := s.rest.Post(epTradingFcoOrder, nil, []byte(payloadJSON), headers)
	if err != nil {
		tradingLog.Error("Error placing FCO order: %v", err)
		return nil, err
	}

	return FCOPlaceResponseFromMap(data), nil
}

func (s *Service) PlaceFcoGtd(accountNo, symbol string, side OrderSide, quantity int, price interface{}, priceSlip float64, fromDate, toDate string) (*FCOPlaceResponse, error) {
	params := &GTDParams{
		AccountNo: accountNo,
		Symbol:    symbol,
		Side:      side,
		Quantity:  quantity,
		Price:     price,
		PriceSlip: priceSlip,
		FromDate:  fromDate,
		ToDate:    toDate,
	}
	return s.placeFcoOrder(params.ToJSON())
}

func (s *Service) PlaceFcoStop(accountNo, symbol string, side OrderSide, quantity int, stopPrice float64, operator FCOOperator, fromDate, toDate string) (*FCOPlaceResponse, error) {
	params := &StopParams{
		AccountNo: accountNo,
		Symbol:    symbol,
		Side:      side,
		Quantity:  quantity,
		StopPrice: stopPrice,
		Operator:  operator,
		FromDate:  fromDate,
		ToDate:    toDate,
		FCOType:   FCOTypeStop,
	}
	return s.placeFcoOrder(params.ToJSON())
}

func (s *Service) PlaceFcoStopLimit(accountNo, symbol string, side OrderSide, quantity int, price interface{}, priceSlip, stopPrice float64, operator FCOOperator, fromDate, toDate string) (*FCOPlaceResponse, error) {
	params := &StopParams{
		AccountNo: accountNo,
		Symbol:    symbol,
		Side:      side,
		Quantity:  quantity,
		Price:     price,
		PriceSlip: priceSlip,
		StopPrice: stopPrice,
		Operator:  operator,
		FromDate:  fromDate,
		ToDate:    toDate,
		FCOType:   FCOTypeStopLimit,
	}
	return s.placeFcoOrder(params.ToJSON())
}

func (s *Service) PlaceFcoTrailingStop(accountNo, symbol string, side OrderSide, quantity int, activePrice, trailingAmount float64, fromDate, toDate string) (*FCOPlaceResponse, error) {
	params := &TrailingStopParams{
		AccountNo:      accountNo,
		Symbol:         symbol,
		Side:           side,
		Quantity:       quantity,
		ActivePrice:    activePrice,
		TrailingAmount: trailingAmount,
		FromDate:       fromDate,
		ToDate:         toDate,
		FCOType:        FCOTypeTrailingStop,
	}
	return s.placeFcoOrder(params.ToJSON())
}

func (s *Service) PlaceFcoTrailingStopLimit(accountNo, symbol string, side OrderSide, quantity int, activePrice, trailingAmount, priceSlip float64, fromDate, toDate string) (*FCOPlaceResponse, error) {
	params := &TrailingStopParams{
		AccountNo:      accountNo,
		Symbol:         symbol,
		Side:           side,
		Quantity:       quantity,
		ActivePrice:    activePrice,
		TrailingAmount: trailingAmount,
		PriceSlip:      priceSlip,
		FromDate:       fromDate,
		ToDate:         toDate,
		FCOType:        FCOTypeTrailingStopLimit,
	}
	return s.placeFcoOrder(params.ToJSON())
}

func (s *Service) PlaceFcoOco(accountNo, symbol string, side OrderSide, quantity int, tpActivePrice, slActivePrice float64, tpPrice, slPrice interface{}, tpSlip, slSlip float64, fromDate, toDate string) (*FCOPlaceResponse, error) {
	params := &OCOParams{
		AccountNo:     accountNo,
		Symbol:        symbol,
		Side:          side,
		Quantity:      quantity,
		TPActivePrice: tpActivePrice,
		SLActivePrice: slActivePrice,
		TPPrice:       tpPrice,
		SLPrice:       slPrice,
		TPSlip:        tpSlip,
		SLSlip:        slSlip,
		FromDate:      fromDate,
		ToDate:        toDate,
		FCOType:       FCOTypeOCO,
	}
	return s.placeFcoOrder(params.ToJSON())
}

func (s *Service) PlaceFcoBullBear(accountNo, symbol string, side OrderSide, quantity int, price interface{}, priceSlip float64, tpActivePrice, slActivePrice float64, tpPrice, slPrice interface{}, tpSlip, slSlip float64, fromDate, toDate string) (*FCOPlaceResponse, error) {
	params := &BullBearParams{
		AccountNo:     accountNo,
		Symbol:        symbol,
		Side:          side,
		Quantity:      quantity,
		Price:         price,
		PriceSlip:     priceSlip,
		TPActivePrice: tpActivePrice,
		SLActivePrice: slActivePrice,
		TPPrice:       tpPrice,
		SLPrice:       slPrice,
		TPSlip:        tpSlip,
		SLSlip:        slSlip,
		FromDate:      fromDate,
		ToDate:        toDate,
		FCOType:       FCOTypeBullBear,
	}
	return s.placeFcoOrder(params.ToJSON())
}

func (s *Service) CancelFco(fcoID string) (*FCOCancelResponse, error) {
	if err := ssi.RequireNonEmpty(fcoID, "fcoId"); err != nil {
		return nil, err
	}

	payloadMap := map[string]interface{}{
		"fcoId":     fcoID,
		"deviceId":  DefaultDeviceID,
		"userAgent": DefaultUserAgent,
	}
	b, _ := json.Marshal(payloadMap)
	payloadJSON := string(b)

	sig, err := signature.Sign(payloadJSON, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign cancel FCO request: %w", err)
	}

	headers := map[string]string{transport.HeaderSignature: sig}
	data, err := s.rest.Delete(epTradingFcoOrder, nil, []byte(payloadJSON), headers)
	if err != nil {
		tradingLog.Error("Error canceling FCO order: %v", err)
		return nil, err
	}

	return FCOCancelResponseFromMap(data), nil
}

func (s *Service) GetFcoByAccountNo(accountNo string, pageIndex, pageSize int) (*FCOListResponse, error) {
	req := &FCOListRequest{AccountNo: accountNo, PageIndex: pageIndex, PageSize: pageSize}
	data, err := s.rest.Get(epTradingFcoList, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting FCO by account: %v", err)
		return nil, err
	}
	return FCOListResponseFromMap(data), nil
}

func (s *Service) GetFcoBySymbol(accountNo, symbol string, pageIndex, pageSize int) (*FCOListResponse, error) {
	req := &FCOListRequest{AccountNo: accountNo, Symbol: symbol, PageIndex: pageIndex, PageSize: pageSize}
	data, err := s.rest.Get(epTradingFcoList, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting FCO by symbol: %v", err)
		return nil, err
	}
	return FCOListResponseFromMap(data), nil
}

func (s *Service) GetFcoByStatus(accountNo string, status FCOStatus, pageIndex, pageSize int) (*FCOListResponse, error) {
	req := &FCOListRequest{AccountNo: accountNo, ProcessStatus: status, PageIndex: pageIndex, PageSize: pageSize}
	data, err := s.rest.Get(epTradingFcoList, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting FCO by status: %v", err)
		return nil, err
	}
	return FCOListResponseFromMap(data), nil
}

func (s *Service) GetFcoByDate(accountNo, fromDate, toDate string, pageIndex, pageSize int) (*FCOListResponse, error) {
	req := &FCOListRequest{AccountNo: accountNo, FromDate: fromDate, ToDate: toDate, PageIndex: pageIndex, PageSize: pageSize}
	data, err := s.rest.Get(epTradingFcoList, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting FCO by date: %v", err)
		return nil, err
	}
	return FCOListResponseFromMap(data), nil
}

func (s *Service) GetFcoById(accountNo, fcoID string) (*FCOInfo, error) {
	req := &FCOListRequest{AccountNo: accountNo, FCOID: fcoID}
	data, err := s.rest.Get(epTradingFcoList, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting FCO by ID: %v", err)
		return nil, err
	}
	res := FCOListResponseFromMap(data)
	if len(res.FCOList) > 0 {
		return res.FCOList[0], nil
	}
	return nil, nil
}

func (s *Service) GetFcoOrderBook(fcoID string, pageIndex, pageSize int) (*FCOOrderBookResponse, error) {
	req := &FCOOrderBookRequest{FCOID: fcoID, PageIndex: pageIndex, PageSize: pageSize}
	data, err := s.rest.Get(epTradingFcoOrderBook, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting FCO order book: %v", err)
		return nil, err
	}
	return FCOOrderBookResponseFromMap(data), nil
}

