package stream

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
	DataTopicQuote  DataTopic = "quote."
	DataTopicTrade  DataTopic = "trade."
	DataTopicOddLot DataTopic = "oddlot."
	DataTopicMarket DataTopic = "market."
	DataTopicRoom   DataTopic = "room."
	DataTopicPut    DataTopic = "put."
)

type DataType string

const (
	DataTypeQuote  DataType = "quote"
	DataTypeTrade  DataType = "trade"
	DataTypeOddLot DataType = "oddlot"
	DataTypeMarket DataType = "market"
	DataTypeRoom   DataType = "room"
	DataTypePut    DataType = "put"
)
