package tradingws

import (
	"context"
	"strings"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/stream"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

// Service is order entry and queries over the trading WebSocket.
//
// It builds the same validated payloads as the REST trading.Service (device id, idempotency key,
// plain decimal prices) and sends them as signed commands. Order methods return the immediate ack
// (orderStatus PD); the real outcome arrives on the order.<account> event stream.
//
// Scopes the token needs: tradingws:order:post / :put / :delete (place / amend / cancel, batch),
// tradingws:fco:post / :delete / :get (conditional orders) and tradingws:position:get /
// :maxBuySell:get / :accountBalance:get / :ppmmrAccount:get / :orderBook:get (queries). A missing
// scope is refused with 403. Rate limits: 50 commands per window for orders, 100 for queries; the
// response carries the remaining quota (Client().RateLimit()) and the client waits for the reset
// when it hits zero.
//
// Prices are float64 and serialized as plain decimal text by the payload builders; queries send
// them as a JSON number.
type Service struct {
	ws    *Client
	cfg   *ssi.Config
	build *trading.PayloadBuilder
}

// NewService wraps a connected client; cfg supplies the device id inputs and client id.
func NewService(client *Client, cfg *ssi.Config) *Service {
	return &Service{ws: client, cfg: cfg, build: trading.NewPayloadBuilder(cfg)}
}

// Client is the underlying socket client: set SetOnEvent to receive events and acks that carry no
// command id, and read RateLimit.
func (s *Service) Client() *Client { return s.ws }

// Close closes the socket and wakes any waiting request (call it when you are done).
func (s *Service) Close() { s.ws.Close() }

func asMap(v interface{}) map[string]interface{} {
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// Command sends any command and returns its first response frame (no retry). params is
// serialized once; a state-changing method signs exactly that text.
func (s *Service) Command(ctx context.Context, method string, params interface{}) (*Response, error) {
	return s.ws.Request(ctx, method, params)
}

// sendText sends params that were already serialized (and validated) by the payload builder.
func (s *Service) sendText(ctx context.Context, method, text string) (*Response, error) {
	return s.ws.request(ctx, method, text)
}

// ── orders ───────────────────────────────────────────────────────────────────

// PlaceOrder places an order (order.place). clientRequestID (<= 20 chars) is the idempotency key;
// empty means one is generated. A reused key is rejected by the server (409). Never retried.
func (s *Service) PlaceOrder(ctx context.Context, accountNo, symbol string, side trading.OrderSide, quantity int, price float64, orderType trading.OrderType, clientRequestID string) (*trading.PlaceOrderResponse, error) {
	text, err := s.build.PlaceOrder(accountNo, symbol, side, quantity, price, orderType, clientRequestID)
	if err != nil {
		return nil, err
	}
	resp, err := s.sendText(ctx, "order.place", text)
	if err != nil {
		return nil, err
	}
	return trading.PlaceOrderResponseFromMap(asMap(resp.Result)), nil
}

// AmendOrder changes the price or the quantity (not both) of one order (order.amend), identified
// by exactly one of orderID / clientRequestID.
func (s *Service) AmendOrder(ctx context.Context, accountNo string, price *float64, quantity *int, orderID, clientRequestID string) (*trading.ModifyOrderResponse, error) {
	text, err := s.build.ModifyOrder(accountNo, orderID, clientRequestID, price, quantity)
	if err != nil {
		return nil, err
	}
	resp, err := s.sendText(ctx, "order.amend", text)
	if err != nil {
		return nil, err
	}
	return trading.ModifyOrderResponseFromMap(asMap(resp.Result)), nil
}

// CancelOrder cancels one order identified by exactly one of orderID / clientRequestID.
func (s *Service) CancelOrder(ctx context.Context, accountNo, orderID, clientRequestID string) (*trading.CancelOrderResponse, error) {
	text, err := s.build.CancelOrder(accountNo, orderID, clientRequestID)
	if err != nil {
		return nil, err
	}
	resp, err := s.sendText(ctx, "order.cancel", text)
	if err != nil {
		return nil, err
	}
	return trading.CancelOrderResponseFromMap(asMap(resp.Result)), nil
}

// BatchNewOrders places up to Config.MaxBatchOrders orders atomically (order.batchNew), validated
// as a whole. The ack's result.results lists clientRequestId / orderStatus / success.
func (s *Service) BatchNewOrders(ctx context.Context, orders []trading.BatchPlaceOrderItem) (*Response, error) {
	text, err := s.build.BatchPlace(orders)
	if err != nil {
		return nil, err
	}
	return s.sendText(ctx, "order.batchNew", text)
}

// BatchCancelOrders cancels up to Config.MaxBatchOrders orders atomically (order.batchCancel).
func (s *Service) BatchCancelOrders(ctx context.Context, orders []trading.BatchCancelOrderItem) (*Response, error) {
	text, err := s.build.BatchCancel(orders)
	if err != nil {
		return nil, err
	}
	return s.sendText(ctx, "order.batchCancel", text)
}

// ── FCO ──────────────────────────────────────────────────────────────────────

// PlaceFCO places a conditional order (fco.place) from the same params types as REST
// (GTDParams, StopParams, TrailingStopParams, OCOParams, BullBearParams).
func (s *Service) PlaceFCO(ctx context.Context, params trading.FCOParamsRequest) (*trading.FCOPlaceResponse, error) {
	text, err := s.build.FCOPlace(params)
	if err != nil {
		return nil, err
	}
	resp, err := s.sendText(ctx, "fco.place", text)
	if err != nil {
		return nil, err
	}
	return trading.FCOPlaceResponseFromMap(asMap(resp.Result)), nil
}

// CancelFCO cancels a conditional order (fco.cancel, scope tradingws:fco:delete). code is the
// one-time code (OTP) when the account requires one; omitted when empty.
func (s *Service) CancelFCO(ctx context.Context, fcoID, code string) (*trading.FCOCancelResponse, error) {
	text, err := s.build.FCOCancel(fcoID, code)
	if err != nil {
		return nil, err
	}
	resp, err := s.sendText(ctx, "fco.cancel", text)
	if err != nil {
		return nil, err
	}
	return trading.FCOCancelResponseFromMap(asMap(resp.Result)), nil
}

// ListFCO lists conditional orders (fco.list). Result holds the page (data, pagesCount...).
// req.ProcessStatus takes at most one status.
func (s *Service) ListFCO(ctx context.Context, req *trading.FCOListRequest) (*Response, error) {
	params, err := s.build.FCOListParams(req)
	if err != nil {
		return nil, err
	}
	return s.ws.Request(ctx, "fco.list", params)
}

// FCOOrderBook is the execution log of one conditional order (fco.orderbook). Page defaults 1 / 10.
func (s *Service) FCOOrderBook(ctx context.Context, fcoID string, pageIndex, pageSize int) (*Response, error) {
	if err := ssi.RequireNonEmpty(fcoID, "fcoId"); err != nil {
		return nil, err
	}
	if pageIndex <= 0 {
		pageIndex = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.ws.Request(ctx, "fco.orderbook", map[string]interface{}{"fcoId": fcoID, "pageIndex": pageIndex, "pageSize": pageSize})
}

// FCOStatusHistory lists the state transitions of one conditional order (fco.statusHistory).
func (s *Service) FCOStatusHistory(ctx context.Context, fcoID string) (*Response, error) {
	if err := ssi.RequireNonEmpty(fcoID, "fcoId"); err != nil {
		return nil, err
	}
	return s.ws.Request(ctx, "fco.statusHistory", map[string]interface{}{"fcoId": fcoID})
}

// ── events ───────────────────────────────────────────────────────────────────

// accountTopic is order.<account> / portfolio.<account>: the server refuses wildcards and accounts
// the token does not own, so an empty, "*" or list account is rejected before anything is sent.
func accountTopic(kind, accountNo string) (string, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return "", err
	}
	if strings.ContainsAny(accountNo, "*,") {
		return "", ssi.NewValidationError("subscribe to one account at a time (no wildcard or list)")
	}
	return stream.BuildTopic(kind, accountNo, "")
}

func (s *Service) subscribeAccount(kind, accountNo string, subscribe bool) error {
	topic, err := accountTopic(kind, accountNo)
	if err != nil {
		return err
	}
	if subscribe {
		return s.ws.Subscribe("TRADING", []string{topic})
	}
	return s.ws.Unsubscribe("TRADING", []string{topic})
}

// SubscribeOrderEvents subscribes to order.<account> events (order status, fills). They arrive on
// the client's OnEvent handler as {"channel","topic","data"} frames (data.eventType: orderEvent /
// orderMatchEvent).
func (s *Service) SubscribeOrderEvents(accountNo string) error {
	return s.subscribeAccount("order", accountNo, true)
}

// UnsubscribeOrderEvents removes a SubscribeOrderEvents subscription.
func (s *Service) UnsubscribeOrderEvents(accountNo string) error {
	return s.subscribeAccount("order", accountNo, false)
}

// SubscribePortfolioEvents subscribes to portfolio.<account> events (derivative position updates,
// data.eventType clientPortfolioEvent).
func (s *Service) SubscribePortfolioEvents(accountNo string) error {
	return s.subscribeAccount("portfolio", accountNo, true)
}

// UnsubscribePortfolioEvents removes a SubscribePortfolioEvents subscription.
func (s *Service) UnsubscribePortfolioEvents(accountNo string) error {
	return s.subscribeAccount("portfolio", accountNo, false)
}

// UnsubscribeAllEvents drops every trading subscription of this socket (UNSUBSCRIBE with no
// topics).
func (s *Service) UnsubscribeAllEvents() error { return s.ws.Unsubscribe("TRADING", nil) }

// ── queries ──────────────────────────────────────────────────────────────────

// withClientID adds clientId only when configured (an empty one is omitted, not sent).
func (s *Service) withClientID(params map[string]interface{}) map[string]interface{} {
	if s.cfg.ClientID != "" {
		params["clientId"] = s.cfg.ClientID
	}
	return params
}

// QueryPosition returns the positions of an account (query.position, scope
// tradingws:position:get); querySummary adds the account summary.
func (s *Service) QueryPosition(ctx context.Context, accountNo string, querySummary bool) (*Response, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	return s.ws.Request(ctx, "query.position", s.withClientID(map[string]interface{}{"accountNo": accountNo, "querySummary": querySummary}))
}

// QueryMaxBuySell returns the maximum buy/sell quantity (query.maxBuySell, scope
// tradingws:maxBuySell:get). A nil price lets the server use its reference price.
func (s *Service) QueryMaxBuySell(ctx context.Context, accountNo, symbol string, price *float64) (*Response, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	if err := ssi.RequireNonEmpty(symbol, "symbol"); err != nil {
		return nil, err
	}
	params := map[string]interface{}{"accountNo": accountNo, "symbol": strings.ToUpper(symbol)}
	if price != nil {
		if err := ssi.RequireNonNegative(*price, "price"); err != nil {
			return nil, err
		}
		text, err := util.FormatPrice(*price)
		if err != nil {
			return nil, ssi.NewValidationError(err.Error())
		}
		params["price"] = jsonNumber(text)
	}
	return s.ws.Request(ctx, "query.maxBuySell", params)
}

// QueryAccountBalance returns the cash balance (query.accountBalance, scope
// tradingws:accountBalance:get).
func (s *Service) QueryAccountBalance(ctx context.Context, accountNo string) (*Response, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	return s.ws.Request(ctx, "query.accountBalance", s.withClientID(map[string]interface{}{"accountNo": accountNo}))
}

// QueryPPMMRAccount returns the purchasing power (query.ppmmrAccount, scope
// tradingws:ppmmrAccount:get).
func (s *Service) QueryPPMMRAccount(ctx context.Context, accountNo string) (*Response, error) {
	if err := ssi.RequireNonEmpty(accountNo, "accountNo"); err != nil {
		return nil, err
	}
	return s.ws.Request(ctx, "query.ppmmrAccount", map[string]interface{}{"accountNo": accountNo})
}

// OrderBookQuery filters query.orderBook. Empty / zero fields are omitted.
type OrderBookQuery struct {
	AccountNo   string
	FromDate    string // "YYYY/MM/DD"
	ToDate      string // "YYYY/MM/DD", not before FromDate
	Symbol      string
	OrderStatus trading.OrderStatus
	PageIndex   int // 1-based
	PageSize    int
}

// QueryOrderBook returns the order book of an account (query.orderBook, scope
// tradingws:orderBook:get).
func (s *Service) QueryOrderBook(ctx context.Context, q OrderBookQuery) (*Response, error) {
	if err := ssi.RequireNonEmpty(q.AccountNo, "accountNo"); err != nil {
		return nil, err
	}
	params := map[string]interface{}{"accountNo": q.AccountNo}
	if q.FromDate != "" && q.ToDate != "" {
		if err := ssi.RequireDateRange(q.FromDate, q.ToDate, false, false, 0); err != nil {
			return nil, err
		}
	}
	for _, kv := range [...]struct{ key, value string }{{"from", q.FromDate}, {"to", q.ToDate}} {
		if kv.value == "" {
			continue
		}
		if _, err := ssi.ParseDateArg(kv.value, kv.key, false, false); err != nil {
			return nil, err
		}
		params[kv.key] = kv.value
	}
	if q.Symbol != "" {
		params["symbol"] = q.Symbol
	}
	if q.OrderStatus != "" {
		params["orderStatus"] = string(q.OrderStatus)
	}
	if q.PageIndex > 0 {
		params["pageIndex"] = q.PageIndex
	}
	if q.PageSize > 0 {
		params["pageSize"] = q.PageSize
	}
	return s.ws.Request(ctx, "query.orderBook", params)
}
