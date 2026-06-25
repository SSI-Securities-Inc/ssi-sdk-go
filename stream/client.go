package stream

import (
	"fmt"
	"sync"
	"time"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

var streamLog = logger.New("ssi_sdk.stream")

// Service provides real-time streaming via a single WebSocket connection.
type Service struct {
	ws                  *transport.WebSocketClient
	onDataCallback      func(interface{})
	onTradingCallback   func(interface{})
	onHeartbeatCallback func(*HeartbeatMessage)
	pingMu              sync.Mutex
	pingStop            chan struct{}
}

func NewService(ws *transport.WebSocketClient) *Service {
	return &Service{ws: ws}
}

// SetOnData sets a callback for market data messages.
func (s *Service) SetOnData(callback func(interface{})) {
	s.onDataCallback = callback
	if callback == nil {
		s.ws.On(string(StreamingChannelData), nil)
		return
	}
	s.ws.On(string(StreamingChannelData), func(msg map[string]interface{}) {
		topic := util.ToStr(msg["topic"])
		data, _ := msg["data"].(map[string]interface{})
		if data == nil {
			callback(msg)
			return
		}
		parsed := ParseStreamingDataMessage(topic, data)
		callback(parsed)
	})
}

// SetOnTrading sets a callback for trading messages.
func (s *Service) SetOnTrading(callback func(interface{})) {
	s.onTradingCallback = callback
	if callback == nil {
		s.ws.On(string(StreamingChannelTrading), nil)
		return
	}
	s.ws.On(string(StreamingChannelTrading), func(msg map[string]interface{}) {
		topic := util.ToStr(msg["topic"])
		data, _ := msg["data"].(map[string]interface{})
		if data == nil {
			callback(msg)
			return
		}
		parsed := ParseStreamingTradingMessage(topic, data)
		callback(parsed)
	})
}

// SetOnHeartbeat sets a callback for heartbeat messages.
func (s *Service) SetOnHeartbeat(callback func(*HeartbeatMessage)) {
	s.onHeartbeatCallback = callback
	if callback == nil {
		s.ws.On(string(StreamingChannelHeartbeat), nil)
		return
	}
	s.ws.On(string(StreamingChannelHeartbeat), func(msg map[string]interface{}) {
		callback(HeartbeatMessageFromMap(msg))
	})
}

func (s *Service) subscribe(req *RequestMessage, onResponse transport.MessageHandler) error {
	if onResponse != nil {
		s.ws.On(string(req.Channel), onResponse)
	}
	return s.ws.Send(req.ToMap())
}

// StopPingLoop stops any active background ping thread.
func (s *Service) StopPingLoop() {
	s.pingMu.Lock()
	defer s.pingMu.Unlock()
	if s.pingStop != nil {
		close(s.pingStop)
		s.pingStop = nil
	}
}

// Ping sends a single ping, or sets up a periodic ping loop if interval is non-nil.
// Any previous ping loop is stopped before starting a new one.
func (s *Service) Ping(onResponse transport.MessageHandler, interval *time.Duration) error {
	s.StopPingLoop()

	sendOnce := func() error {
		streamLog.Debug("Sending ping to WebSocket server")
		return s.subscribe(&RequestMessage{
			Method:  StreamingMethodPingPong,
			Channel: StreamingChannelHeartbeat,
		}, onResponse)
	}

	if interval == nil {
		return sendOnce()
	}

	s.pingMu.Lock()
	s.pingStop = make(chan struct{})
	stopChan := s.pingStop
	s.pingMu.Unlock()

	go func() {
		ticker := time.NewTicker(*interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := sendOnce(); err != nil {
					streamLog.Error("Failed to send periodic ping: %v", err)
				}
			case <-stopChan:
				return
			}
		}
	}()

	streamLog.Debug("Ping loop started")
	return nil
}

func (s *Service) SubscribeSymbolTrade(symbols []string, onResponse transport.MessageHandler) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("trade.%s", sym)
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelData,
		Topics:  topics,
	}, onResponse)
}

func (s *Service) SubscribeSymbolQuote(symbols []string, onResponse transport.MessageHandler) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("quote.%s", sym)
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelData,
		Topics:  topics,
	}, onResponse)
}

func (s *Service) SubscribeSymbolRoom(symbols []string, onResponse transport.MessageHandler) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("room.%s", sym)
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelData,
		Topics:  topics,
	}, onResponse)
}

func (s *Service) SubscribeSymbolPutThrough(symbols []string, onResponse transport.MessageHandler) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("put.%s", sym)
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelData,
		Topics:  topics,
	}, onResponse)
}

func (s *Service) SubscribeSymbolOddLot(symbols []string, onResponse transport.MessageHandler) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("oddlot.%s", sym)
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelData,
		Topics:  topics,
	}, onResponse)
}

func (s *Service) SubscribeSymbol(symbols []string, onResponse transport.MessageHandler) error {
	if err := s.SubscribeSymbolTrade(symbols, onResponse); err != nil {
		return err
	}
	if err := s.SubscribeSymbolQuote(symbols, onResponse); err != nil {
		return err
	}
	return s.SubscribeSymbolRoom(symbols, onResponse)
}

func (s *Service) SubscribeBoard(boards []market.Board, onResponse transport.MessageHandler) error {
	boardValues := make([]string, len(boards))
	for i, b := range boards {
		boardValues[i] = string(b)
	}
	if err := s.SubscribeSymbolTrade(boardValues, onResponse); err != nil {
		return err
	}
	if err := s.SubscribeSymbolQuote(boardValues, onResponse); err != nil {
		return err
	}
	return s.SubscribeSymbolRoom(boardValues, onResponse)
}

func (s *Service) SubscribeIndex(indices []string, onResponse transport.MessageHandler) error {
	if err := s.SubscribeSymbolTrade(indices, onResponse); err != nil {
		return err
	}
	if err := s.SubscribeSymbolQuote(indices, onResponse); err != nil {
		return err
	}
	return s.SubscribeSymbolRoom(indices, onResponse)
}

func (s *Service) unsubscribe(channel StreamingChannel, topics []string) error {
	return s.ws.Send((&RequestMessage{
		Method:  StreamingMethodUnsubscribe,
		Channel: channel,
		Topics:  topics,
	}).ToMap())
}

func (s *Service) UnsubscribeSymbolTrade(symbols []string) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("trade.%s", sym)
	}
	return s.unsubscribe(StreamingChannelData, topics)
}

func (s *Service) UnsubscribeSymbolQuote(symbols []string) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("quote.%s", sym)
	}
	return s.unsubscribe(StreamingChannelData, topics)
}

func (s *Service) UnsubscribeSymbolRoom(symbols []string) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("room.%s", sym)
	}
	return s.unsubscribe(StreamingChannelData, topics)
}

func (s *Service) UnsubscribeSymbolPutThrough(symbols []string) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("put.%s", sym)
	}
	return s.unsubscribe(StreamingChannelData, topics)
}

func (s *Service) UnsubscribeSymbolOddLot(symbols []string) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("oddlot.%s", sym)
	}
	return s.unsubscribe(StreamingChannelData, topics)
}

func (s *Service) SubscribeSymbolOhlcv(symbols []string, interval market.Timeframe, onResponse transport.MessageHandler) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("trade.%s@%s", sym, string(interval))
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelData,
		Topics:  topics,
	}, onResponse)
}

func (s *Service) UnsubscribeSymbolOhlcv(symbols []string, interval market.Timeframe) error {
	topics := make([]string, len(symbols))
	for i, sym := range symbols {
		topics[i] = fmt.Sprintf("trade.%s@%s", sym, string(interval))
	}
	return s.unsubscribe(StreamingChannelData, topics)
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

func (s *Service) UnsubscribeBoard(boards []market.Board) error {
	boardValues := make([]string, len(boards))
	for i, b := range boards {
		boardValues[i] = string(b)
	}
	if err := s.UnsubscribeSymbolTrade(boardValues); err != nil {
		return err
	}
	if err := s.UnsubscribeSymbolQuote(boardValues); err != nil {
		return err
	}
	return s.UnsubscribeSymbolRoom(boardValues)
}

func (s *Service) UnsubscribeIndex(indices []string) error {
	if err := s.UnsubscribeSymbolTrade(indices); err != nil {
		return err
	}
	if err := s.UnsubscribeSymbolQuote(indices); err != nil {
		return err
	}
	return s.UnsubscribeSymbolRoom(indices)
}

func (s *Service) SubscribeOrderStatus(accountNo string, onResponse transport.MessageHandler) error {
	if accountNo == "" {
		accountNo = "*"
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelTrading,
		Topics:  []string{fmt.Sprintf("order.%s", accountNo)},
	}, onResponse)
}

func (s *Service) SubscribePortfolio(accountNo string, onResponse transport.MessageHandler) error {
	if accountNo == "" {
		accountNo = "*"
	}
	return s.subscribe(&RequestMessage{
		Method:  StreamingMethodSubscribe,
		Channel: StreamingChannelTrading,
		Topics:  []string{fmt.Sprintf("portfolio.%s", accountNo)},
	}, onResponse)
}

func (s *Service) Wait(timeout *time.Duration) {
	s.ws.Wait(timeout)
}
