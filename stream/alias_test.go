package stream

import (
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/transport"
)

// Compile-time check that every alias exists with the expected signature.
var (
	_ func(*Service, []string, transport.MessageHandler) error                   = (*Service).SubscribeTrades
	_ func(*Service, []string, transport.MessageHandler) error                   = (*Service).SubscribeQuotes
	_ func(*Service, []string, transport.MessageHandler) error                   = (*Service).SubscribeForeignRoom
	_ func(*Service, []string, transport.MessageHandler) error                   = (*Service).SubscribePutThrough
	_ func(*Service, []string, transport.MessageHandler) error                   = (*Service).SubscribeOddLot
	_ func(*Service, []string, market.Timeframe, transport.MessageHandler) error = (*Service).SubscribeCandles
	_ func(*Service, []market.Board, transport.MessageHandler) error             = (*Service).SubscribeExchangeTrades
	_ func(*Service, []string, transport.MessageHandler) error                   = (*Service).SubscribeIndexConstituents
	_ func(*Service, string, transport.MessageHandler) error                     = (*Service).SubscribePositions
	_ func(*Service, []string) error                                             = (*Service).UnsubscribeTrades
	_ func(*Service, []string) error                                             = (*Service).UnsubscribeQuotes
	_ func(*Service, []string) error                                             = (*Service).UnsubscribeForeignRoom
	_ func(*Service, []string) error                                             = (*Service).UnsubscribePutThrough
	_ func(*Service, []string) error                                             = (*Service).UnsubscribeOddLot
	_ func(*Service, []string, market.Timeframe) error                           = (*Service).UnsubscribeCandles
	_ func(*Service, []market.Board) error                                       = (*Service).UnsubscribeExchangeTrades
	_ func(*Service, []string) error                                             = (*Service).UnsubscribeIndexConstituents
	_ func(*Service, string) error                                               = (*Service).UnsubscribePositions
	_ func(*Service) error                                                       = (*Service).UnsubscribeSessionFlag
)
