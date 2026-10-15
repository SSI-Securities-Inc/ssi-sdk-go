package stream

import (
	"fmt"
	"sync"
	"time"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var streamLog = logger.New("ssi_sdk.stream")

const defaultListTimeout = 5 * time.Second

// Service provides real-time streaming over one WebSocket connection.
//
// SetOnData / SetOnTrading receive every message of their channel (typed, or the raw map for a
// frame without a model). The typed callbacks (SetOnOrder, SetOnMarketFlag, ...) receive only
// their own message type and subscribe nothing by themselves.
type Service struct {
	ws *transport.WebSocketClient

	mu             sync.Mutex
	onData         func(interface{})
	onTrading      func(interface{})
	onHeartbeat    func(*HeartbeatMessage)
	onMarketFlag   func(*MarketFlagMessage)
	onIndexSummary func(*IndexSummaryMessage)
	onOrder        func(*OrderStatusMessage)
	onOrderMatch   func(*OrderMatchMessage)
	onPortfolio    func(*PortfolioMessage)
	pingStop       chan struct{}
	pingDone       chan struct{}
}

func NewService(ws *transport.WebSocketClient) *Service {
	return &Service{ws: ws}
}

// callSafely runs fn and keeps a panicking callback from starving the others.
func callSafely(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			streamLog.Error("Error in a streaming callback: %v", r)
		}
	}()
	fn()
}

func (s *Service) handleData(msg map[string]interface{}) {
	message := ParseDataMessage(msg)
	s.mu.Lock()
	onData, onFlag, onSummary := s.onData, s.onMarketFlag, s.onIndexSummary
	s.mu.Unlock()
	if onData != nil {
		callSafely(func() { onData(message) })
	}
	switch m := message.(type) {
	case *MarketFlagMessage:
		if onFlag != nil {
			callSafely(func() { onFlag(m) })
		}
	case *IndexSummaryMessage:
		if onSummary != nil {
			callSafely(func() { onSummary(m) })
		}
	}
}

func (s *Service) handleTrading(msg map[string]interface{}) {
	message := ParseTradingMessage(msg)
	s.mu.Lock()
	onTrading, onOrder, onMatch, onPortfolio := s.onTrading, s.onOrder, s.onOrderMatch, s.onPortfolio
	s.mu.Unlock()
	if onTrading != nil {
		callSafely(func() { onTrading(message) })
	}
	switch m := message.(type) {
	case *OrderStatusMessage:
		if onOrder != nil {
			callSafely(func() { onOrder(m) })
		}
	case *OrderMatchMessage:
		if onMatch != nil {
			callSafely(func() { onMatch(m) })
		}
	case *PortfolioMessage:
		if onPortfolio != nil {
			callSafely(func() { onPortfolio(m) })
		}
	}
}

// refreshHandlers installs (or clears) the one handler per channel from the callbacks set now.
func (s *Service) refreshHandlers() {
	s.mu.Lock()
	wantsData := s.onData != nil || s.onMarketFlag != nil || s.onIndexSummary != nil
	wantsTrading := s.onTrading != nil || s.onOrder != nil || s.onOrderMatch != nil || s.onPortfolio != nil
	s.mu.Unlock()

	if wantsData {
		s.ws.On(string(StreamingChannelData), s.handleData)
	} else {
		s.ws.On(string(StreamingChannelData), nil)
	}
	if wantsTrading {
		s.ws.On(string(StreamingChannelTrading), s.handleTrading)
	} else {
		s.ws.On(string(StreamingChannelTrading), nil)
	}
}

// SetOnData sets the callback for every DATA-channel message; nil clears it.
func (s *Service) SetOnData(callback func(interface{})) {
	s.mu.Lock()
	s.onData = callback
	s.mu.Unlock()
	s.refreshHandlers()
}

// SetOnTrading sets the callback for every TRADING-channel message; nil clears it.
func (s *Service) SetOnTrading(callback func(interface{})) {
	s.mu.Lock()
	s.onTrading = callback
	s.mu.Unlock()
	s.refreshHandlers()
}

// SetOnHeartbeat sets the callback for HEARTBEAT-channel messages; nil clears it.
func (s *Service) SetOnHeartbeat(callback func(*HeartbeatMessage)) {
	s.mu.Lock()
	s.onHeartbeat = callback
	s.mu.Unlock()
	if callback == nil {
		s.ws.On(string(StreamingChannelHeartbeat), nil)
		return
	}
	s.ws.On(string(StreamingChannelHeartbeat), func(msg map[string]interface{}) {
		callback(HeartbeatMessageFromMap(msg))
	})
}

// SetOnMarketFlag receives only MarketFlagMessage (subscribe with SubscribeMarketFlag).
func (s *Service) SetOnMarketFlag(callback func(*MarketFlagMessage)) {
	s.mu.Lock()
	s.onMarketFlag = callback
	s.mu.Unlock()
	s.refreshHandlers()
}

// SetOnIndexSummary receives only IndexSummaryMessage (subscribe with SubscribeIndexSummary).
func (s *Service) SetOnIndexSummary(callback func(*IndexSummaryMessage)) {
	s.mu.Lock()
	s.onIndexSummary = callback
	s.mu.Unlock()
	s.refreshHandlers()
}

// SetOnOrder receives only OrderStatusMessage (subscribe with SubscribeOrderStatus).
func (s *Service) SetOnOrder(callback func(*OrderStatusMessage)) {
	s.mu.Lock()
	s.onOrder = callback
	s.mu.Unlock()
	s.refreshHandlers()
}

// SetOnOrderMatch receives only OrderMatchMessage (fills; same subscription as order status).
func (s *Service) SetOnOrderMatch(callback func(*OrderMatchMessage)) {
	s.mu.Lock()
	s.onOrderMatch = callback
	s.mu.Unlock()
	s.refreshHandlers()
}

// SetOnPortfolio receives only PortfolioMessage (subscribe with SubscribePortfolio).
func (s *Service) SetOnPortfolio(callback func(*PortfolioMessage)) {
	s.mu.Lock()
	s.onPortfolio = callback
	s.mu.Unlock()
	s.refreshHandlers()
}

// SetOnConnection sets the callback for connection state changes (connected, reconnecting,
// disconnected, ...); nil clears it.
func (s *Service) SetOnConnection(callback func(transport.ConnectionEvent)) {
	s.ws.SetOnState(callback)
}

// IsConnected reports whether the socket is connected right now.
func (s *Service) IsConnected() bool { return s.ws.IsConnected() }

// LastError is the last connection error, or nil.
func (s *Service) LastError() error { return s.ws.LastError() }

// ReconnectCount is how many times the connection was re-established.
func (s *Service) ReconnectCount() int { return s.ws.ReconnectCount() }

func (s *Service) send(req *RequestMessage, onResponse transport.MessageHandler) error {
	if onResponse != nil {
		// Acks have their own route: registering on the request's channel would replace the
		// caller's data/trading handler.
		s.ws.On(ssi.WSRouteAck, onResponse)
	}
	return s.ws.Send(req.ToMap())
}

func (s *Service) dataRequest(method StreamingMethod, topicType string, filters []string, interval market.Timeframe) (*RequestMessage, error) {
	topics, err := BuildTopics(topicType, filters, interval)
	if err != nil {
		return nil, err
	}
	return &RequestMessage{Method: method, Channel: StreamingChannelData, Topics: topics}, nil
}

func (s *Service) subscribeData(topicType string, filters []string, interval market.Timeframe, onResponse transport.MessageHandler) error {
	req, err := s.dataRequest(StreamingMethodSubscribe, topicType, filters, interval)
	if err != nil {
		return err
	}
	return s.send(req, onResponse)
}

func (s *Service) unsubscribeData(topicType string, filters []string, interval market.Timeframe) error {
	req, err := s.dataRequest(StreamingMethodUnsubscribe, topicType, filters, interval)
	if err != nil {
		return err
	}
	return s.ws.Send(req.ToMap())
}

func (s *Service) tradingRequest(method StreamingMethod, topicType, accountNo string) (*RequestMessage, error) {
	if accountNo == "" {
		accountNo = "*"
	}
	topic, err := BuildTopic(topicType, accountNo, "")
	if err != nil {
		return nil, err
	}
	return &RequestMessage{Method: method, Channel: StreamingChannelTrading, Topics: []string{topic}}, nil
}

// StopPingLoop stops the background ping loop, if one is running.
func (s *Service) StopPingLoop() {
	s.mu.Lock()
	stop, done := s.pingStop, s.pingDone
	s.pingStop, s.pingDone = nil, nil
	s.mu.Unlock()
	if stop != nil {
		close(stop)
		<-done
	}
}

// Ping sends one ping, or with a non-nil intervalMs starts a background loop that pings every
// intervalMs milliseconds (the first ping goes out after one interval). Calling Ping again
// replaces the loop. The loop ends when a send fails, i.e. the connection is gone.
func (s *Service) Ping(onResponse transport.MessageHandler, intervalMs *int) error {
	sendOnce := func() error {
		streamLog.Debug("Sending ping to WebSocket server")
		return s.send(&RequestMessage{
			Method:  StreamingMethodPingPong,
			Channel: StreamingChannelHeartbeat,
		}, onResponse)
	}
	if intervalMs == nil {
		return sendOnce()
	}
	if *intervalMs <= 0 {
		return ssi.NewValidationError("ping interval must be positive")
	}
	s.StopPingLoop()

	stop := make(chan struct{})
	done := make(chan struct{})
	s.mu.Lock()
	s.pingStop, s.pingDone = stop, done
	s.mu.Unlock()

	every := time.Duration(*intervalMs) * time.Millisecond
	go func() {
		defer close(done)
		ticker := time.NewTicker(every)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := sendOnce(); err != nil {
					streamLog.Debug("Ping loop stopped: the connection is gone")
					return
				}
			case <-stop:
				return
			}
		}
	}()
	streamLog.Debug("Ping loop started with interval=%d ms", *intervalMs)
	return nil
}

// --- Data channel: subscribe ---

// SubscribeSymbolOhlcv subscribes to candles of the given symbols. interval is tick, 1m or 5m.
func (s *Service) SubscribeSymbolOhlcv(symbols []string, interval market.Timeframe, onResponse transport.MessageHandler) error {
	if interval == "" {
		return ssi.NewValidationError("interval is required")
	}
	return s.subscribeData("trade", symbols, interval, onResponse)
}

func (s *Service) SubscribeSymbolTrade(symbols []string, onResponse transport.MessageHandler) error {
	return s.subscribeData("trade", symbols, "", onResponse)
}

func (s *Service) SubscribeSymbolQuote(symbols []string, onResponse transport.MessageHandler) error {
	return s.subscribeData("quote", symbols, "", onResponse)
}

func (s *Service) SubscribeSymbolRoom(symbols []string, onResponse transport.MessageHandler) error {
	return s.subscribeData("room", symbols, "", onResponse)
}

func (s *Service) SubscribeSymbolPutThrough(symbols []string, onResponse transport.MessageHandler) error {
	return s.subscribeData("put", symbols, "", onResponse)
}

func (s *Service) SubscribeSymbolOddLot(symbols []string, onResponse transport.MessageHandler) error {
	return s.subscribeData("oddlot", symbols, "", onResponse)
}

// SubscribeSymbol subscribes to trade, quote and foreign room of the symbols.
func (s *Service) SubscribeSymbol(symbols []string, onResponse transport.MessageHandler) error {
	if err := s.SubscribeSymbolTrade(symbols, onResponse); err != nil {
		return err
	}
	if err := s.SubscribeSymbolQuote(symbols, onResponse); err != nil {
		return err
	}
	return s.SubscribeSymbolRoom(symbols, onResponse)
}

func boardNames(boards []market.Board) []string {
	names := make([]string, len(boards))
	for i, b := range boards {
		names[i] = string(b)
	}
	return names
}

// groupTopicTypes are the topic types SubscribeBoard / SubscribeIndex cover; each goes out as
// <type>.<code>@*.
var groupTopicTypes = []string{"trade", "quote", "room"}

// groupRequests builds the per-type requests for an exchange / basket-index fan-out
// (trade.VN30@*). All are built (and so validated) before any is sent.
func (s *Service) groupRequests(method StreamingMethod, codes []string) ([]*RequestMessage, error) {
	reqs := make([]*RequestMessage, 0, len(groupTopicTypes))
	for _, topicType := range groupTopicTypes {
		req, err := s.dataRequest(method, topicType, codes, StreamIntervalAll)
		if err != nil {
			return nil, err
		}
		reqs = append(reqs, req)
	}
	return reqs, nil
}

func (s *Service) subscribeGroup(codes []string, onResponse transport.MessageHandler) error {
	reqs, err := s.groupRequests(StreamingMethodSubscribe, codes)
	if err != nil {
		return err
	}
	for _, req := range reqs {
		if err := s.send(req, onResponse); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) unsubscribeGroup(codes []string) error {
	reqs, err := s.groupRequests(StreamingMethodUnsubscribe, codes)
	if err != nil {
		return err
	}
	for _, req := range reqs {
		if err := s.ws.Send(req.ToMap()); err != nil {
			return err
		}
	}
	return nil
}

// SubscribeBoard subscribes to trade, quote and foreign room of the boards. Sent as
// trade.HOSE@* (etc.): the server only fans an exchange out to its symbols with the @* interval.
func (s *Service) SubscribeBoard(boards []market.Board, onResponse transport.MessageHandler) error {
	return s.subscribeGroup(boardNames(boards), onResponse)
}

// SubscribeIndex subscribes to trade, quote and foreign room under trade.<index>@* topics, i.e.
// the trades of the index's constituent symbols (the server only fans a basket out with the @*
// interval). For the ticks of the index itself use SubscribeIndexTrade.
func (s *Service) SubscribeIndex(indices []string, onResponse transport.MessageHandler) error {
	return s.subscribeGroup(indices, onResponse)
}

// SubscribeMarketFlag subscribes to the session flags (ATO, LO, ATC, ...) of all boards.
func (s *Service) SubscribeMarketFlag(onResponse transport.MessageHandler) error {
	return s.send(&RequestMessage{
		Method: StreamingMethodSubscribe, Channel: StreamingChannelData, Topics: []string{TopicMarketFlag},
	}, onResponse)
}

// SubscribeMarket subscribes to master data (ceiling, floor, reference) of symbols or boards.
func (s *Service) SubscribeMarket(symbols []string, onResponse transport.MessageHandler) error {
	return s.subscribeData("market", symbols, "", onResponse)
}

// SubscribeIndexTrade subscribes to the ticks of the index itself (trade.index.<code>); with
// interval 1m/5m they arrive as candles. UPCOMINDEX is sent as HNXUPCOMINDEX. An interval needs
// one concrete index (no wildcard).
func (s *Service) SubscribeIndexTrade(indices []string, interval market.Timeframe, onResponse transport.MessageHandler) error {
	return s.subscribeData("trade.index", indices, interval, onResponse)
}

// SubscribeIndexSummary subscribes to index summaries (indexsummary.<code>).
func (s *Service) SubscribeIndexSummary(indices []string, onResponse transport.MessageHandler) error {
	return s.subscribeData("indexsummary", indices, "", onResponse)
}

// --- Data channel: unsubscribe ---

func (s *Service) UnsubscribeSymbolTrade(symbols []string) error {
	return s.unsubscribeData("trade", symbols, "")
}

func (s *Service) UnsubscribeSymbolQuote(symbols []string) error {
	return s.unsubscribeData("quote", symbols, "")
}

func (s *Service) UnsubscribeSymbolRoom(symbols []string) error {
	return s.unsubscribeData("room", symbols, "")
}

func (s *Service) UnsubscribeSymbolPutThrough(symbols []string) error {
	return s.unsubscribeData("put", symbols, "")
}

func (s *Service) UnsubscribeSymbolOddLot(symbols []string) error {
	return s.unsubscribeData("oddlot", symbols, "")
}

// UnsubscribeSymbolOhlcv removes candle topics; pass the interval you subscribed with.
func (s *Service) UnsubscribeSymbolOhlcv(symbols []string, interval market.Timeframe) error {
	if interval == "" {
		return ssi.NewValidationError("interval is required")
	}
	return s.unsubscribeData("trade", symbols, interval)
}

func (s *Service) UnsubscribeSymbol(symbols []string) error {
	if err := s.UnsubscribeSymbolTrade(symbols); err != nil {
		return err
	}
	if err := s.UnsubscribeSymbolQuote(symbols); err != nil {
		return err
	}
	return s.UnsubscribeSymbolRoom(symbols)
}

// UnsubscribeBoard removes the trade/quote/room @* topics SubscribeBoard sent.
func (s *Service) UnsubscribeBoard(boards []market.Board) error {
	return s.unsubscribeGroup(boardNames(boards))
}

// UnsubscribeIndex removes the trade/quote/room @* topics SubscribeIndex sent.
func (s *Service) UnsubscribeIndex(indices []string) error {
	return s.unsubscribeGroup(indices)
}

func (s *Service) UnsubscribeMarketFlag() error {
	return s.ws.Send((&RequestMessage{
		Method: StreamingMethodUnsubscribe, Channel: StreamingChannelData, Topics: []string{TopicMarketFlag},
	}).ToMap())
}

func (s *Service) UnsubscribeMarket(symbols []string) error {
	return s.unsubscribeData("market", symbols, "")
}

// UnsubscribeIndexTrade removes trade.index.<code> topics; pass the interval you subscribed with.
func (s *Service) UnsubscribeIndexTrade(indices []string, interval market.Timeframe) error {
	return s.unsubscribeData("trade.index", indices, interval)
}

func (s *Service) UnsubscribeIndexSummary(indices []string) error {
	return s.unsubscribeData("indexsummary", indices, "")
}

// --- Trading channel ---

// SubscribeOrderStatus subscribes to order.<account>: order status and fills. An empty
// accountNo means all accounts ("*").
func (s *Service) SubscribeOrderStatus(accountNo string, onResponse transport.MessageHandler) error {
	req, err := s.tradingRequest(StreamingMethodSubscribe, "order", accountNo)
	if err != nil {
		return err
	}
	return s.send(req, onResponse)
}

// SubscribePortfolio subscribes to portfolio.<account> (derivative positions). An empty
// accountNo means all accounts ("*").
func (s *Service) SubscribePortfolio(accountNo string, onResponse transport.MessageHandler) error {
	req, err := s.tradingRequest(StreamingMethodSubscribe, "portfolio", accountNo)
	if err != nil {
		return err
	}
	return s.send(req, onResponse)
}

// UnsubscribeOrderStatus removes order.<account>; use the account given to subscribe.
func (s *Service) UnsubscribeOrderStatus(accountNo string) error {
	req, err := s.tradingRequest(StreamingMethodUnsubscribe, "order", accountNo)
	if err != nil {
		return err
	}
	return s.ws.Send(req.ToMap())
}

func (s *Service) UnsubscribePortfolio(accountNo string) error {
	req, err := s.tradingRequest(StreamingMethodUnsubscribe, "portfolio", accountNo)
	if err != nil {
		return err
	}
	return s.ws.Send(req.ToMap())
}

// ListSubscription asks the server which topics this connection is subscribed to. The result is
// {"trading": [...], "data": [...]}; a channel with nothing subscribed is an empty list. A
// timeoutMs <= 0 means 5000 (milliseconds).
func (s *Service) ListSubscription(timeoutMs int) (map[string][]string, error) {
	timeout := time.Duration(timeoutMs) * time.Millisecond
	if timeout <= 0 {
		timeout = defaultListTimeout
	}
	arrived := make(chan map[string][]string, 1)
	s.ws.On(ssi.WSRouteListSubscription, func(message map[string]interface{}) {
		select {
		case arrived <- transport.SplitSubscriptions(message):
		default:
		}
	})
	defer s.ws.Off(ssi.WSRouteListSubscription, nil)

	if err := s.ws.Send(map[string]interface{}{"method": string(StreamingMethodListSubscription)}); err != nil {
		return nil, err
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-arrived:
		return result, nil
	case <-timer.C:
		return nil, ssi.NewWebSocketError("Timed out waiting for the LIST_SUBSCRIPTION reply")
	}
}

// UnsubscribeAll asks the server what this connection is subscribed to and removes exactly that,
// so it also clears topics left by an earlier run of the same session. It returns the topics
// that were removed.
func (s *Service) UnsubscribeAll() (map[string][]string, error) {
	current, err := s.ListSubscription(0)
	if err != nil {
		return nil, err
	}
	channels := []struct {
		key     string
		channel StreamingChannel
	}{{"data", StreamingChannelData}, {"trading", StreamingChannelTrading}}
	for _, c := range channels {
		if topics := current[c.key]; len(topics) > 0 {
			req := &RequestMessage{Method: StreamingMethodUnsubscribe, Channel: c.channel, Topics: topics}
			if err := s.ws.Send(req.ToMap()); err != nil {
				return current, fmt.Errorf("unsubscribe %s: %w", c.key, err)
			}
		}
	}
	return current, nil
}

// Connect opens the connection. After it returns the client keeps the connection alive:
// reconnect with backoff, a fresh token per attempt, and re-subscribe.
func (s *Service) Connect() error { return s.ws.Connect() }

// Disconnect stops the ping loop and closes the connection.
func (s *Service) Disconnect() {
	s.StopPingLoop()
	s.ws.Disconnect()
}

// Wait blocks until the connection is closed or timeout elapses; nil waits indefinitely.
func (s *Service) Wait(timeoutMs *int) { s.ws.Wait(timeoutMs) }

// --- Clearer names: the same methods, named for what arrives ---

// SubscribeOrders is SubscribeOrderStatus (order status AND fills).
func (s *Service) SubscribeOrders(accountNo string, onResponse transport.MessageHandler) error {
	return s.SubscribeOrderStatus(accountNo, onResponse)
}

// SubscribeIndexTicks is SubscribeIndexTrade (ticks of the index itself).
func (s *Service) SubscribeIndexTicks(indices []string, interval market.Timeframe, onResponse transport.MessageHandler) error {
	return s.SubscribeIndexTrade(indices, interval, onResponse)
}

// SubscribeMasterData is SubscribeMarket (ceiling / floor / reference price).
func (s *Service) SubscribeMasterData(symbols []string, onResponse transport.MessageHandler) error {
	return s.SubscribeMarket(symbols, onResponse)
}

// SubscribeSessionFlag is SubscribeMarketFlag (ATO / LO / ATC ... per board).
func (s *Service) SubscribeSessionFlag(onResponse transport.MessageHandler) error {
	return s.SubscribeMarketFlag(onResponse)
}

func (s *Service) UnsubscribeOrders(accountNo string) error {
	return s.UnsubscribeOrderStatus(accountNo)
}

func (s *Service) UnsubscribeIndexTicks(indices []string, interval market.Timeframe) error {
	return s.UnsubscribeIndexTrade(indices, interval)
}

func (s *Service) UnsubscribeMasterData(symbols []string) error { return s.UnsubscribeMarket(symbols) }

// UnsubscribeSessionFlag is UnsubscribeMarketFlag.
func (s *Service) UnsubscribeSessionFlag() error { return s.UnsubscribeMarketFlag() }

// --- The classic topics too ---

// SubscribeTrades is SubscribeSymbolTrade (trade.<symbol>: matched trades).
func (s *Service) SubscribeTrades(symbols []string, onResponse transport.MessageHandler) error {
	return s.SubscribeSymbolTrade(symbols, onResponse)
}

// SubscribeQuotes is SubscribeSymbolQuote (quote.<symbol>: bid/ask depth).
func (s *Service) SubscribeQuotes(symbols []string, onResponse transport.MessageHandler) error {
	return s.SubscribeSymbolQuote(symbols, onResponse)
}

// SubscribeForeignRoom is SubscribeSymbolRoom (room.<symbol>: foreign room).
func (s *Service) SubscribeForeignRoom(symbols []string, onResponse transport.MessageHandler) error {
	return s.SubscribeSymbolRoom(symbols, onResponse)
}

// SubscribePutThrough is SubscribeSymbolPutThrough (put.<symbol>: negotiated trades).
func (s *Service) SubscribePutThrough(symbols []string, onResponse transport.MessageHandler) error {
	return s.SubscribeSymbolPutThrough(symbols, onResponse)
}

// SubscribeOddLot is SubscribeSymbolOddLot (oddlot.<symbol>: odd-lot trades).
func (s *Service) SubscribeOddLot(symbols []string, onResponse transport.MessageHandler) error {
	return s.SubscribeSymbolOddLot(symbols, onResponse)
}

// SubscribeCandles is SubscribeSymbolOhlcv (trade.<symbol>@tick|1m|5m).
func (s *Service) SubscribeCandles(symbols []string, interval market.Timeframe, onResponse transport.MessageHandler) error {
	return s.SubscribeSymbolOhlcv(symbols, interval, onResponse)
}

// SubscribeExchangeTrades is SubscribeBoard (trade/quote/room of a whole exchange).
func (s *Service) SubscribeExchangeTrades(boards []market.Board, onResponse transport.MessageHandler) error {
	return s.SubscribeBoard(boards, onResponse)
}

// SubscribeIndexConstituents is SubscribeIndex (trade/quote/room of an index's members).
func (s *Service) SubscribeIndexConstituents(indices []string, onResponse transport.MessageHandler) error {
	return s.SubscribeIndex(indices, onResponse)
}

// SubscribePositions is SubscribePortfolio (portfolio.<account>: derivative positions).
func (s *Service) SubscribePositions(accountNo string, onResponse transport.MessageHandler) error {
	return s.SubscribePortfolio(accountNo, onResponse)
}

func (s *Service) UnsubscribeTrades(symbols []string) error { return s.UnsubscribeSymbolTrade(symbols) }
func (s *Service) UnsubscribeQuotes(symbols []string) error { return s.UnsubscribeSymbolQuote(symbols) }
func (s *Service) UnsubscribeForeignRoom(symbols []string) error {
	return s.UnsubscribeSymbolRoom(symbols)
}
func (s *Service) UnsubscribePutThrough(symbols []string) error {
	return s.UnsubscribeSymbolPutThrough(symbols)
}
func (s *Service) UnsubscribeOddLot(symbols []string) error {
	return s.UnsubscribeSymbolOddLot(symbols)
}
func (s *Service) UnsubscribeCandles(symbols []string, interval market.Timeframe) error {
	return s.UnsubscribeSymbolOhlcv(symbols, interval)
}
func (s *Service) UnsubscribeExchangeTrades(boards []market.Board) error {
	return s.UnsubscribeBoard(boards)
}
func (s *Service) UnsubscribeIndexConstituents(indices []string) error {
	return s.UnsubscribeIndex(indices)
}
func (s *Service) UnsubscribePositions(accountNo string) error {
	return s.UnsubscribePortfolio(accountNo)
}
