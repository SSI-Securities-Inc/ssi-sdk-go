package stream

import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"

type StreamingMethod string

const (
	StreamingMethodSubscribe        StreamingMethod = "subscribe"
	StreamingMethodUnsubscribe      StreamingMethod = "unsubscribe"
	StreamingMethodPingPong         StreamingMethod = "ping_pong"
	StreamingMethodListSubscription StreamingMethod = "list_subscription"
)

type StreamingChannel string

const (
	StreamingChannelData      StreamingChannel = "DATA"
	StreamingChannelHeartbeat StreamingChannel = "HEARTBEAT"
	StreamingChannelTrading   StreamingChannel = "TRADING"
)

type StreamingType string

const (
	StreamingTypeOrder      StreamingType = "orderEvent"
	StreamingTypeOrderMatch StreamingType = "orderMatchEvent"
	StreamingTypePortfolio  StreamingType = "clientPortfolioEvent"
)

type DataTopic string

const (
	DataTopicQuote        DataTopic = "quote."
	DataTopicTrade        DataTopic = "trade."
	DataTopicOddLot       DataTopic = "oddlot."
	DataTopicMarket       DataTopic = "market."
	DataTopicRoom         DataTopic = "room."
	DataTopicPut          DataTopic = "put."
	DataTopicIndexSummary DataTopic = "indexsummary."
)

// TopicMarketFlag is the all-boards session-flag topic.
const TopicMarketFlag = "market.flag"

type DataType string

const (
	DataTypeQuote        DataType = "quote"
	DataTypeTrade        DataType = "trade"
	DataTypeOddLot       DataType = "oddlot"
	DataTypeMarket       DataType = "market"
	DataTypeRoom         DataType = "room"
	DataTypePut          DataType = "put"
	DataTypeIndexSummary DataType = "indexsummary"
)

// Intervals the stream serves, and only on trade / trade.index. Any other value is a
// ValidationError.
const (
	IntervalTick    market.Timeframe = "tick"
	IntervalMinute1 market.Timeframe = "1m"
	IntervalMinute5 market.Timeframe = "5m"
)
