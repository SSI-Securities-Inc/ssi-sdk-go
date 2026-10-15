package trading

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/device"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/signature"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var tradingLog = logger.New("ssi_sdk.services.trading")

const (
	epTradingOrder           = "/api/v3/trading/order"
	epTradingOrderBatch      = "/api/v3/trading/order/batch"
	epTradingMaxBuySell      = "/api/v3/trading/maxBuySell"
	epTradingFcoOrder        = "/api/v3/trading/fco/order"
	epTradingFcoList         = "/api/v3/trading/fco/list"
	epTradingFcoOrderBook    = "/api/v3/trading/fco/orderbook"
	epTradingFcoStatusHisory = "/api/v3/trading/fco/statusHistory"

	fcoMaxRangeDays = 31
)

// Service provides trading operations: place/cancel/modify orders, batches and FCO orders.
//
// The SDK never retries an order by itself. A caller-chosen client request id (WithClientRequestID)
// lets an order that timed out be looked up, or re-sent safely: the server answers 409
// (ssi.DuplicateRequestError) when the first one went through.
type Service struct {
	rest       *transport.RestClient
	privateKey string
}

func NewService(rest *transport.RestClient, privateKey string) *Service {
	return &Service{rest: rest, privateKey: privateKey}
}

// OrderOption tunes one order call.
type OrderOption func(*orderOptions)

type orderOptions struct{ clientRequestID string }

// WithClientRequestID sets your own idempotency key (<= 20 characters). Without it the SDK
// generates one. Re-sending a used key within the day is rejected with a DuplicateRequestError
// (HTTP 409).
func WithClientRequestID(id string) OrderOption {
	return func(o *orderOptions) { o.clientRequestID = id }
}

// FCOOption tunes one FCO call.
type FCOOption func(*fcoOptions)

type fcoOptions struct{ code string }

// WithOTPCode sends the one-time code (OTP) when the account requires it for the command.
func WithOTPCode(code string) FCOOption {
	return func(o *fcoOptions) { o.code = code }
}

func orderOpts(opts []OrderOption) orderOptions {
	var o orderOptions
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

func fcoOpts(opts []FCOOption) fcoOptions {
	var o fcoOptions
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// identity is the device id (this machine's id) and User-Agent every order payload carries.
func identity(cfg *ssi.Config) (deviceID, userAgent string) {
	userAgent = cfg.UserAgent
	if userAgent == "" {
		userAgent = defaultPayloadUserAgent()
	}
	return device.ID(), userAgent
}

// resolveClientRequestID uses the caller's idempotency key, or generates a random one.
func resolveClientRequestID(value string) (string, error) {
	if value == "" {
		return util.GenerateRequestID(), nil
	}
	if err := ssi.RequireNonEmpty(value, "clientRequestId"); err != nil {
		return "", err
	}
	if len(value) > ssi.MaxClientRequestIDLength {
		return "", ssi.NewValidationError(fmt.Sprintf("clientRequestId must be at most %d characters", ssi.MaxClientRequestIDLength))
	}
	return value, nil
}

func resolveOperator(op FCOOperator) (FCOOperator, error) {
	if resolved, ok := FCOOperatorFromValue(string(op)); ok {
		return resolved, nil
	}
	return "", ssi.NewValidationError(fmt.Sprintf("operator must be one of %s, got '%s'", strings.Join(fcoOps, ", "), op))
}

func buildPlaceOrder(cfg *ssi.Config, accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType, clientRequestID string) (*PlaceOrderRequest, error) {
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
	if err := ssi.RequirePositive(float64(quantity), "quantity"); err != nil {
		return nil, err
	}
	if pricedOrderType(orderType) && !(price > 0) {
		return nil, ssi.NewValidationError(fmt.Sprintf("price is required for %s orders", orderType))
	}
	id, err := resolveClientRequestID(clientRequestID)
	if err != nil {
		return nil, err
	}
	deviceID, ua := identity(cfg)
	return &PlaceOrderRequest{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity, Price: price,
		OrderType: orderType, ClientRequestID: id, DeviceID: deviceID, UserAgent: ua,
	}, nil
}

func buildModifyOrder(cfg *ssi.Config, accountNo, orderID, clientRequestID string, price *float64, quantity *int) (*ModifyOrderRequest, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	if err := ssi.RequireExactlyOne(orderID, clientRequestID, "orderId", "clientRequestId"); err != nil {
		return nil, err
	}
	if price == nil && quantity == nil {
		return nil, ssi.NewValidationError("either price or quantity is required")
	}
	if price != nil && quantity != nil {
		return nil, ssi.NewValidationError("price and quantity cannot be modified in the same request")
	}
	if price != nil {
		if err := ssi.RequirePositive(*price, "price"); err != nil {
			return nil, err
		}
	}
	if quantity != nil {
		if err := ssi.RequirePositive(float64(*quantity), "quantity"); err != nil {
			return nil, err
		}
	}
	deviceID, ua := identity(cfg)
	return &ModifyOrderRequest{
		AccountNo: accountNo, Quantity: quantity, Price: price, OrderID: orderID,
		ClientRequestID: clientRequestID, ClientModifyID: util.GenerateRequestID(),
		DeviceID: deviceID, UserAgent: ua,
	}, nil
}

func buildCancelOrder(cfg *ssi.Config, accountNo, orderID, clientRequestID string) (*CancelOrderRequest, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	if err := ssi.RequireExactlyOne(orderID, clientRequestID, "orderId", "clientRequestId"); err != nil {
		return nil, err
	}
	deviceID, ua := identity(cfg)
	return &CancelOrderRequest{
		AccountNo: accountNo, OrderID: orderID, ClientRequestID: clientRequestID,
		ClientCancelID: util.GenerateRequestID(), DeviceID: deviceID, UserAgent: ua,
	}, nil
}

// checkBatch: a batch holds 1..limit orders; the server rejects bigger ones, so fail early.
func checkBatch(n, limit int) error {
	if limit <= 0 {
		limit = ssi.MaxBatchOrders
	}
	if n == 0 {
		return ssi.NewValidationError("a batch needs at least one order")
	}
	if n > limit {
		return ssi.NewValidationError(fmt.Sprintf("a batch holds at most %d orders, got %d", limit, n))
	}
	return nil
}

// newBatch is the batch envelope: a random id (a reused one is a 409) and the current time in
// epoch milliseconds (outside the server's replay window it is a 400111).
func newBatch(orders []interface{}, now time.Time) *BatchOrderRequest {
	return &BatchOrderRequest{
		BatchRequestID:   util.GenerateRequestID(),
		BatchRequestTime: now.UnixMilli(),
		Orders:           orders,
	}
}

// BuildBatchPlace validates every order first (one bad order rejects the whole batch) and builds
// the body.
func buildBatchPlace(cfg *ssi.Config, items []BatchPlaceOrderItem, now time.Time) (*BatchOrderRequest, error) {
	if err := checkBatch(len(items), cfg.MaxBatchOrders); err != nil {
		return nil, err
	}
	orders := make([]interface{}, 0, len(items))
	for _, it := range items {
		req, err := buildPlaceOrder(cfg, it.AccountNo, it.Symbol, it.Side, it.Quantity, it.Price, it.OrderType, it.ClientRequestID)
		if err != nil {
			return nil, err
		}
		w, err := req.wire()
		if err != nil {
			return nil, err
		}
		orders = append(orders, w)
	}
	return newBatch(orders, now), nil
}

func buildBatchCancel(cfg *ssi.Config, items []BatchCancelOrderItem, now time.Time) (*BatchOrderRequest, error) {
	if err := checkBatch(len(items), cfg.MaxBatchOrders); err != nil {
		return nil, err
	}
	orders := make([]interface{}, 0, len(items))
	for _, it := range items {
		req, err := buildCancelOrder(cfg, it.AccountNo, it.OrderID, it.ClientRequestID)
		if err != nil {
			return nil, err
		}
		orders = append(orders, req.wire())
	}
	return newBatch(orders, now), nil
}

// sign signs the exact body text that is then sent.
func (s *Service) sign(body string) (map[string]string, error) {
	sig, err := signature.Sign(body, s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	return map[string]string{transport.HeaderSignature: sig}, nil
}

func (s *Service) send(method, path, body string) (map[string]interface{}, error) {
	headers, err := s.sign(body)
	if err != nil {
		return nil, err
	}
	return s.rest.Do(context.Background(), transport.Request{
		Method: method, Path: path, Content: []byte(body), Headers: headers,
	})
}

func (s *Service) placeOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType, opts []OrderOption) (*PlaceOrderResponse, error) {
	req, err := buildPlaceOrder(s.rest.Config(), accountNo, symbol, side, quantity, price, orderType, orderOpts(opts).clientRequestID)
	if err != nil {
		return nil, err
	}
	body, err := req.ToJSON()
	if err != nil {
		return nil, err
	}
	data, err := s.send("POST", epTradingOrder, body)
	if err != nil {
		tradingLog.Error("Error placing order: %v", err)
		return nil, err
	}
	return PlaceOrderResponseFromMap(data), nil
}

// PlaceOrder places an order of any type. price is in VND (0 for market-priced types; LO and PLO
// need a positive price).
func (s *Service) PlaceOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType, opts ...OrderOption) (*PlaceOrderResponse, error) {
	if err := ssi.RequireNonNegative(price, "price"); err != nil {
		return nil, err
	}
	return s.placeOrder(accountNo, symbol, side, quantity, price, orderType, opts)
}

func (s *Service) PlaceLimitOrder(accountNo, symbol string, side OrderSide, quantity int, price float64, opts ...OrderOption) (*PlaceOrderResponse, error) {
	if err := ssi.RequirePositive(price, "price"); err != nil {
		return nil, err
	}
	return s.PlaceOrder(accountNo, symbol, side, quantity, price, OrderTypeLO, opts...)
}

func (s *Service) PlaceMarketOrder(accountNo, symbol string, side OrderSide, quantity int, opts ...OrderOption) (*PlaceOrderResponse, error) {
	return s.PlaceOrder(accountNo, symbol, side, quantity, 0, OrderTypeMTL, opts...)
}

func (s *Service) PlaceATOOrder(accountNo, symbol string, side OrderSide, quantity int, opts ...OrderOption) (*PlaceOrderResponse, error) {
	return s.PlaceOrder(accountNo, symbol, side, quantity, 0, OrderTypeATO, opts...)
}

func (s *Service) PlaceATCOrder(accountNo, symbol string, side OrderSide, quantity int, opts ...OrderOption) (*PlaceOrderResponse, error) {
	return s.PlaceOrder(accountNo, symbol, side, quantity, 0, OrderTypeATC, opts...)
}

func (s *Service) modifyOrder(accountNo, orderID, clientRequestID string, price *float64, quantity *int) (*ModifyOrderResponse, error) {
	req, err := buildModifyOrder(s.rest.Config(), accountNo, orderID, clientRequestID, price, quantity)
	if err != nil {
		return nil, err
	}
	body, err := req.ToJSON()
	if err != nil {
		return nil, err
	}
	data, err := s.send("PUT", epTradingOrder, body)
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
	req, err := buildCancelOrder(s.rest.Config(), accountNo, orderID, clientRequestID)
	if err != nil {
		return nil, err
	}
	body, err := req.ToJSON()
	if err != nil {
		return nil, err
	}
	data, err := s.send("DELETE", epTradingOrder, body)
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

// PlaceBatchOrders places 1..Config.MaxBatchOrders orders in one request. The whole batch is
// validated first (one bad order rejects all); the response holds one result per order.
func (s *Service) PlaceBatchOrders(orders []BatchPlaceOrderItem) (*BatchOrderResponse, error) {
	req, err := buildBatchPlace(s.rest.Config(), orders, time.Now())
	if err != nil {
		return nil, err
	}
	return s.sendBatch("POST", req)
}

// CancelBatchOrders cancels 1..Config.MaxBatchOrders orders in one request.
func (s *Service) CancelBatchOrders(orders []BatchCancelOrderItem) (*BatchOrderResponse, error) {
	req, err := buildBatchCancel(s.rest.Config(), orders, time.Now())
	if err != nil {
		return nil, err
	}
	return s.sendBatch("DELETE", req)
}

func (s *Service) sendBatch(method string, req *BatchOrderRequest) (*BatchOrderResponse, error) {
	body, err := req.ToJSON()
	if err != nil {
		return nil, err
	}
	data, err := s.send(method, epTradingOrderBatch, body)
	if err != nil {
		tradingLog.Error("Error sending batch (%s): %v", method, err)
		return nil, err
	}
	return BatchOrderResponseFromMap(data), nil
}

// GetMaxBuySell returns the max buy/sell quantity at a price (positive when given; nil = market
// price).
func (s *Service) GetMaxBuySell(accountNo, symbol string, price *float64) (*MaxBuySellResponse, error) {
	if price != nil {
		if err := ssi.RequirePositive(*price, "price"); err != nil {
			return nil, err
		}
	}
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
	params, err := (&MaxBuySellRequest{AccountNo: accountNo, Symbol: symbol, Price: price}).ToMap()
	if err != nil {
		return nil, err
	}
	data, err := s.rest.Get(epTradingMaxBuySell, params, nil)
	if err != nil {
		tradingLog.Error("Error getting max buy/sell: %v", err)
		return nil, err
	}
	return MaxBuySellResponseFromMap(data, symbol), nil
}

// ---------------------------------------------------------------------------
// Flexible Conditional Orders (FCO)
// ---------------------------------------------------------------------------

// prepareFCO validates the validity window (YYYY/MM/DD HH:MM:SS, from <= to, at most 31 days:
// the API contract says so but the server does not check it, so the SDK does) and fills the
// caller identity. The device id is always this machine's; the server records the IP.
func prepareFCO(cfg *ssi.Config, p FCOParamsRequest) error {
	from, to := p.window()
	if err := ssi.RequireDateRange(from, to, false, true, fcoMaxRangeDays); err != nil {
		return err
	}
	deviceID, ua := identity(cfg)
	p.setIdentity(deviceID, ua)
	return nil
}

func (s *Service) placeFco(p FCOParamsRequest) (*FCOPlaceResponse, error) {
	if err := prepareFCO(s.rest.Config(), p); err != nil {
		return nil, err
	}
	body, err := p.ToJSON()
	if err != nil {
		return nil, err
	}
	data, err := s.send("POST", epTradingFcoOrder, body)
	if err != nil {
		tradingLog.Error("Error placing FCO order: %v", err)
		return nil, err
	}
	return FCOPlaceResponseFromMap(data), nil
}

// PlaceFcoGtd places a good-till-date order. price is a number or an OrderType word (slip is then
// ignored).
func (s *Service) PlaceFcoGtd(accountNo, symbol string, side OrderSide, quantity int, price interface{}, priceSlip float64, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error) {
	return s.placeFco(&GTDParams{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity, Price: price,
		PriceSlip: priceSlip, FromDate: fromDate, ToDate: toDate, Code: fcoOpts(opts).code,
	})
}

func (s *Service) PlaceFcoStop(accountNo, symbol string, side OrderSide, quantity int, stopPrice float64, operator FCOOperator, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error) {
	op, err := resolveOperator(operator)
	if err != nil {
		return nil, err
	}
	return s.placeFco(&StopParams{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity, StopPrice: stopPrice,
		Operator: op, FromDate: fromDate, ToDate: toDate, FCOType: FCOTypeStop, Code: fcoOpts(opts).code,
	})
}

func (s *Service) PlaceFcoStopLimit(accountNo, symbol string, side OrderSide, quantity int, price interface{}, priceSlip, stopPrice float64, operator FCOOperator, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error) {
	op, err := resolveOperator(operator)
	if err != nil {
		return nil, err
	}
	return s.placeFco(&StopParams{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity, Price: price,
		PriceSlip: priceSlip, StopPrice: stopPrice, Operator: op, FromDate: fromDate, ToDate: toDate,
		FCOType: FCOTypeStopLimit, Code: fcoOpts(opts).code,
	})
}

func (s *Service) PlaceFcoTrailingStop(accountNo, symbol string, side OrderSide, quantity int, activePrice, trailingAmount float64, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error) {
	return s.placeFco(&TrailingStopParams{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity, ActivePrice: activePrice,
		TrailingAmount: trailingAmount, FromDate: fromDate, ToDate: toDate,
		FCOType: FCOTypeTrailingStop, Code: fcoOpts(opts).code,
	})
}

func (s *Service) PlaceFcoTrailingStopLimit(accountNo, symbol string, side OrderSide, quantity int, activePrice, trailingAmount, priceSlip float64, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error) {
	return s.placeFco(&TrailingStopParams{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity, ActivePrice: activePrice,
		TrailingAmount: trailingAmount, PriceSlip: priceSlip, FromDate: fromDate, ToDate: toDate,
		FCOType: FCOTypeTrailingStopLimit, Code: fcoOpts(opts).code,
	})
}

func (s *Service) PlaceFcoOco(accountNo, symbol string, side OrderSide, quantity int, tpActivePrice, slActivePrice float64, tpPrice, slPrice interface{}, tpSlip, slSlip float64, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error) {
	return s.placeFco(&OCOParams{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity,
		TPActivePrice: tpActivePrice, SLActivePrice: slActivePrice, TPPrice: tpPrice, SLPrice: slPrice,
		TPSlip: tpSlip, SLSlip: slSlip, FromDate: fromDate, ToDate: toDate, FCOType: FCOTypeOCO,
		Code: fcoOpts(opts).code,
	})
}

func (s *Service) PlaceFcoBullBear(accountNo, symbol string, side OrderSide, quantity int, price interface{}, priceSlip float64, tpActivePrice, slActivePrice float64, tpPrice, slPrice interface{}, tpSlip, slSlip float64, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error) {
	return s.placeFco(&BullBearParams{
		AccountNo: accountNo, Symbol: symbol, Side: side, Quantity: quantity, Price: price,
		PriceSlip: priceSlip, TPActivePrice: tpActivePrice, SLActivePrice: slActivePrice,
		TPPrice: tpPrice, SLPrice: slPrice, TPSlip: tpSlip, SLSlip: slSlip, FromDate: fromDate,
		ToDate: toDate, FCOType: FCOTypeBullBear, Code: fcoOpts(opts).code,
	})
}

// CancelFco cancels an FCO. Pass WithOTPCode when the account requires a code for the command.
func (s *Service) CancelFco(fcoID string, opts ...FCOOption) (*FCOCancelResponse, error) {
	if err := ssi.RequireNonEmpty(fcoID, "fcoId"); err != nil {
		return nil, err
	}
	deviceID, ua := identity(s.rest.Config())
	req := &FCOCancelRequest{FCOID: fcoID, DeviceID: deviceID, UserAgent: ua, Code: fcoOpts(opts).code}
	body, err := req.ToJSON()
	if err != nil {
		return nil, err
	}
	data, err := s.send("DELETE", epTradingFcoOrder, body)
	if err != nil {
		tradingLog.Error("Error canceling FCO order: %v", err)
		return nil, err
	}
	return FCOCancelResponseFromMap(data), nil
}

func (s *Service) getFcoList(req *FCOListRequest) (*FCOListResponse, error) {
	if err := ssi.RequireNonEmpty(req.AccountNo, "accountNo"); err != nil {
		return nil, err
	}
	if req.FromDate != "" && req.ToDate != "" {
		if err := ssi.RequireDateRange(req.FromDate, req.ToDate, true, false, 0); err != nil {
			return nil, err
		}
	} else {
		for name, v := range map[string]string{"from": req.FromDate, "to": req.ToDate} {
			if v == "" {
				continue
			}
			if _, err := ssi.ParseDateArg(v, name, true, false); err != nil {
				return nil, err
			}
		}
	}
	data, err := s.query(epTradingFcoList, req.ToQuery())
	if err != nil {
		tradingLog.Error("Error getting FCO list: %v", err)
		return nil, err
	}
	return FCOListResponseFromMap(data), nil
}

func (s *Service) query(path string, q url.Values) (map[string]interface{}, error) {
	return s.rest.Do(context.Background(), transport.Request{Method: "GET", Path: path, Query: q})
}

func (s *Service) GetFcoByAccountNo(accountNo string, pageIndex, pageSize int) (*FCOListResponse, error) {
	return s.getFcoList(&FCOListRequest{AccountNo: accountNo, PageIndex: pageIndex, PageSize: pageSize})
}

func (s *Service) GetFcoBySymbol(accountNo, symbol string, pageIndex, pageSize int) (*FCOListResponse, error) {
	return s.getFcoList(&FCOListRequest{AccountNo: accountNo, Symbol: symbol, PageIndex: pageIndex, PageSize: pageSize})
}

func (s *Service) GetFcoByStatus(accountNo string, status FCOStatus, pageIndex, pageSize int) (*FCOListResponse, error) {
	return s.GetFcoByStatuses(accountNo, []FCOStatus{status}, pageIndex, pageSize)
}

// GetFcoByStatuses lists FCOs in any of the given statuses (sent as a repeated processStatus
// key).
func (s *Service) GetFcoByStatuses(accountNo string, statuses []FCOStatus, pageIndex, pageSize int) (*FCOListResponse, error) {
	return s.getFcoList(&FCOListRequest{AccountNo: accountNo, ProcessStatus: statuses, PageIndex: pageIndex, PageSize: pageSize})
}

// GetFcoByDate lists FCOs in a range ("YYYY/MM/DD HH:MM:SS", or a bare "YYYY/MM/DD").
func (s *Service) GetFcoByDate(accountNo, fromDate, toDate string, pageIndex, pageSize int) (*FCOListResponse, error) {
	return s.getFcoList(&FCOListRequest{AccountNo: accountNo, FromDate: fromDate, ToDate: toDate, PageIndex: pageIndex, PageSize: pageSize})
}

// GetFcoById returns one FCO, or nil when it does not exist.
func (s *Service) GetFcoById(accountNo, fcoID string) (*FCOInfo, error) {
	res, err := s.getFcoList(&FCOListRequest{AccountNo: accountNo, FCOID: fcoID})
	if err != nil {
		return nil, err
	}
	if len(res.FCOList) > 0 {
		return res.FCOList[0], nil
	}
	return nil, nil
}

// GetFcoOrderBook lists the orders an FCO has produced (page defaults 1 / 10).
func (s *Service) GetFcoOrderBook(fcoID string, pageIndex, pageSize int) (*FCOOrderBookResponse, error) {
	if err := ssi.RequireNonEmpty(fcoID, "fcoId"); err != nil {
		return nil, err
	}
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	req := &FCOOrderBookRequest{FCOID: fcoID, PageIndex: pageIndex, PageSize: pageSize}
	data, err := s.rest.Get(epTradingFcoOrderBook, req.ToMap(), nil)
	if err != nil {
		tradingLog.Error("Error getting FCO order book: %v", err)
		return nil, err
	}
	return FCOOrderBookResponseFromMap(data), nil
}

// GetFcoStatusHistory lists the status transitions of one FCO.
func (s *Service) GetFcoStatusHistory(fcoID string) ([]FCOStatusHistoryItem, error) {
	if err := ssi.RequireNonEmpty(fcoID, "fcoId"); err != nil {
		return nil, err
	}
	data, err := s.rest.Get(epTradingFcoStatusHisory, map[string]string{"fcoid": fcoID}, nil)
	if err != nil {
		tradingLog.Error("Error getting FCO status history: %v", err)
		return nil, err
	}
	return FCOStatusHistoryFromMap(data), nil
}
