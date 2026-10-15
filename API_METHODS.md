# API_METHODS.md — every public method

> Taken from the exported methods in the code. Narrative guides: `README.md` (usage), `AGENT.md` (cheat sheet), `CHANGELOG.md`.
> Every method returns `error` as its last value (omitted below when it is the only return). Prices are `float64`
> and are sent as plain decimal strings (`util.FormatPrice`). Calls are blocking; the services take no `context`
> except the experimental Trading WebSocket.

Clients (package `ssi`, import `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi`): `NewAuth(cfg)`, `NewData(auth)`,
`NewTrading(auth)`, `NewStream(auth)`, or the one-stop `NewClient(cfg)`. Config comes from `ssisdk.NewConfig(clientID)`.

## Authentication

Reach it through `auth.TokenManager` (the most used ones are also methods on `*ssi.Auth`).

| Method | Signature | What it does |
|---|---|---|
| `Authenticate` | `(otp string, transactionID ...string) (*Token, error)` | One call to obtain an access token with consumer credentials and an OTP. |
| `EnsureAuthenticated` | `(otp string, transactionID ...string) (string, error)` | Return a valid token, refreshing or authenticating as needed (polls Smart OTP). |
| `EnsureAuthenticatedWith` | `(ctx context.Context, opts EnsureOptions) (string, error)` | Same as above with a context and polling options. |
| `ssi.Auth.EnsureAuthenticatedWith` | `(ctx context.Context, opts auth.EnsureOptions) (string, error)` | Wrapper of the method above on `*ssi.Auth`. |
| `Refresh` | `() (*Token, error)` | Get a new access token with the current refresh token. |
| `RequestOTP` | `() (map[string]interface{}, error)` | Request an OTP (normal) or a push (Smart OTP). |
| `RequestOTPTyped` | `() (*OTPResponse, error)` | Same, typed (`TransactionID` exists only for Smart OTP). |
| `PollSmartOTP` | `(transactionID string, intervalMs int, maxRetries int) (*Token, error)` | Poll until the Smart OTP is approved. |
| `SetToken` | `(token *Token)` | Set the token manually. |
| `Token` / `AccessToken` | `() *Token` / `() string` | The current token / its access string. |
| `IsTokenExpired` | `() bool` | Missing or (nearly, 30s skew) expired. |
| `HasRefreshToken` / `CanRefresh` / `IsRefreshTokenExpired` | `() bool` | Refresh-token state. |
| `ConnectRestOnly` | `()` | Mark the manager as REST-only (no streaming token lead). |

## Market data

`data.MarketData` (`*market.Service`).

| Method | Signature | What it does |
|---|---|---|
| `GetOHLC1Minute` / `3Minute` / `5Minute` / `15Minute` / `1Hour` | `(symbol string) ([]OHLCData, error)` | Intraday bars for the latest trading day. |
| `GetOHLC{1,3,5,15}MinuteHistorical`, `GetOHLC1HourHistorical`, `GetOHLC1DayHistorical` | `(symbol, fromDate, toDate string, page, size int) ([]OHLCData, error)` | Bars between two dates. |
| `GetOHLC1WeekHistorical` / `GetOHLC1MonthHistorical` | same | Return `ValidationError` (server rejects, 400210). |
| `GetOHLCAll` | `(symbol string, tf Timeframe, q OHLCQuery) ([]OHLCData, error)` | Every bar in a range across all pages. |
| `IterOHLC` | `(symbol string, tf Timeframe, q OHLCQuery, yield func(OHLCData) bool) error` | Page-by-page iterator; return `false` to stop. |
| `DownloadOHLC1Minute` / `DownloadOHLC1Day` | `(symbol string) ([]OHLCData, error)` | Return `ErrNotImplemented` (no server endpoint yet). |
| `GetIndexes` | `() ([]MarketIndexes, error)` | All indexes. |
| `GetIndexesByBoard` | `(board Board) ([]MarketIndexes, error)` | Indexes of one board. |
| `GetIndexSummary` / `GetIndexSummaryHistorical` | `(index string[, tradingDate string]) (*MarketIndexSummary, error)` | `nil, nil` when not found. |
| `GetBoardSummary` / `GetBoardSummaryHistorical` | `(board Board[, tradingDate string]) (*MarketIndexSummary, error)` | `nil, nil` when not found. |
| `GetSecuritiesInfo` | `(symbol string) (*SecuritiesInfo, error)` | One security; `nil, nil` when not found. |
| `GetSecuritiesInfoByIndex` / `ByBoard` | `(index string)` / `(board Board)` `([]SecuritiesInfo, error)` | Securities of an index / board. |
| `GetSecuritiesSummary` / `…Historical` | `(symbol string[, fromDate, toDate string]) ([]SecuritiesSummary, error)` | Summary of a symbol. |
| `GetSecuritiesSummaryByIndex` / `…Historical` | `(index string[, fromDate, toDate string]) ([]SecuritiesSummary, error)` | Summary of an index's members. |
| `GetMasterData` / `GetMasterDataHistorical` | `() / (fromDate, toDate string) ([]MasterData, error)` | Ceiling/floor/reference for all symbols. |

## Account

`trading.Account` (`*account.Service`).

| Method | Signature | What it does |
|---|---|---|
| `GetAccountInfo` | `() ([]Account, error)` | Accessible trading accounts. |

## Portfolio

`trading.Portfolio` (`*portfolio.Service`). All take `accountNo string`.

| Method | Signature | What it does |
|---|---|---|
| `GetEquityBalance` | `(accountNo) (*EquityAccountBalance, error)` | Equity balance. |
| `GetDerivativeBalance` | `(accountNo) (*DerivativeAccountBalance, error)` | Derivative balance. |
| `GetEquityPositions` | `(accountNo) ([]EquityPosition, error)` | Equity positions. |
| `GetDerivativePositions` | `(accountNo) (*AllDerivativePosition, error)` | All derivative positions (open and closed). |
| `GetOpenDerivativePositions` / `GetClosedDerivativePositions` | `(accountNo) ([]DerivativePosition, error)` | Open / closed derivative positions. |
| `GetEquityPPMMR` / `GetDerivativePPMMR` | `(accountNo) (*EquityPPMMR / *DerivativePPMMR, error)` | Purchasing power and margin ratio. |
| `GetTodayOrders` | `(accountNo) ([]Order, error)` | Today's orders. |
| `GetHistoricalOrders` | `(accountNo, fromDate, toDate string) ([]Order, error)` | Orders in a date range. |

## Trading (orders, FCO, batch)

`trading.Trading` (`*trading.Service`). Options: `WithClientRequestID(id)` (`OrderOption`, idempotency key; generated when
omitted) and `WithOTPCode(code)` (`FCOOption`). A private key is required (fail closed).

| Method | Signature | What it does |
|---|---|---|
| `PlaceOrder` | `(accountNo, symbol string, side OrderSide, quantity int, price float64, orderType OrderType, opts ...OrderOption) (*PlaceOrderResponse, error)` | Any order type. |
| `PlaceLimitOrder` | `(accountNo, symbol, side, quantity, price, opts...)` | LO order. |
| `PlaceMarketOrder` / `PlaceATOOrder` / `PlaceATCOrder` | `(accountNo, symbol, side, quantity, opts...)` | MTL / ATO / ATC order. |
| `ModifyOrderPrice` / `ModifyOrderPriceByOrderID` | `(accountNo, clientRequestID \| orderID string, price float64) (*ModifyOrderResponse, error)` | Change the price. |
| `ModifyOrderQuantity` / `ModifyOrderQuantityByOrderID` | `(accountNo, clientRequestID \| orderID string, quantity int) (*ModifyOrderResponse, error)` | Change the quantity. |
| `CancelOrder` / `CancelOrderByOrderID` | `(accountNo, clientRequestID \| orderID string) (*CancelOrderResponse, error)` | Cancel one order. |
| `PlaceBatchOrders` / `CancelBatchOrders` | `(orders []BatchPlaceOrderItem \| []BatchCancelOrderItem) (*BatchOrderResponse, error)` | Signed batch, at most `Config.MaxBatchOrders` (default 20). |
| `GetMaxBuySell` | `(accountNo, symbol string, price *float64) (*MaxBuySellResponse, error)` | Max buy/sell at a price. |
| `GetMaxBuySellAtMarketPrice` | `(accountNo, symbol string) (*MaxBuySellResponse, error)` | Max buy/sell at market. |
| `PlaceFcoGtd` | `(accountNo, symbol, side, quantity, price interface{}, priceSlip float64, fromDate, toDate string, opts ...FCOOption) (*FCOPlaceResponse, error)` | GTD order (`price` is a number or an `OrderType`). |
| `PlaceFcoStop` / `PlaceFcoStopLimit` | `(…, stopPrice, operator FCOOperator, fromDate, toDate, opts...)` | Stop / stop-limit. |
| `PlaceFcoTrailingStop` / `PlaceFcoTrailingStopLimit` | `(…, activePrice, trailingAmount[, priceSlip], fromDate, toDate, opts...)` | Trailing stop / stop-limit. |
| `PlaceFcoOco` | `(…, tpActivePrice, slActivePrice, tpPrice, slPrice interface{}, tpSlip, slSlip, fromDate, toDate, opts...)` | OCO. |
| `PlaceFcoBullBear` | `(…, price interface{}, priceSlip, tpActivePrice, slActivePrice, tpPrice, slPrice interface{}, tpSlip, slSlip, fromDate, toDate, opts...)` | Bull-bear. |
| `CancelFco` | `(fcoID string, opts ...FCOOption) (*FCOCancelResponse, error)` | Cancel a FCO (optional OTP code). |
| `GetFcoByAccountNo` | `(accountNo string, pageIndex, pageSize int) (*FCOListResponse, error)` | FCO list of an account. |
| `GetFcoBySymbol` | `(accountNo, symbol string, pageIndex, pageSize int)` | Filter by symbol. |
| `GetFcoByStatus` / `GetFcoByStatuses` | `(accountNo string, status FCOStatus \| statuses []FCOStatus, pageIndex, pageSize int)` | Filter by processing status. |
| `GetFcoByDate` | `(accountNo, fromDate, toDate string, pageIndex, pageSize int)` | Filter by date range. |
| `GetFcoById` | `(accountNo, fcoID string) (*FCOInfo, error)` | One FCO; `nil, nil` when not found. |
| `GetFcoOrderBook` | `(fcoID string, pageIndex, pageSize int) (*FCOOrderBookResponse, error)` | Execution log of a FCO. |
| `GetFcoStatusHistory` | `(fcoID string) ([]FCOStatusHistoryItem, error)` | Status transitions of a FCO. |

## Streaming

`stream.Streaming` (`*stream.Service`). `onResponse transport.MessageHandler` may be `nil`.

| Method | Signature | What it does |
|---|---|---|
| `Connect` / `Disconnect` | `() error` / `()` | Open the socket (returns after the first connect, then supervised) / close it. |
| `Wait` | `(timeoutMs *int)` | Block until closed or the timeout (milliseconds). |
| `Ping` / `StopPingLoop` | `(onResponse, intervalMs *int) error` / `()` | Ping once or start the ping loop. |
| `ListSubscription` | `(timeoutMs int) (map[string][]string, error)` | Ask the server what this connection subscribes to. |
| `SubscribeSymbolTrade` / `Quote` / `Room` / `PutThrough` / `OddLot` | `(symbols []string, onResponse) error` | Per-symbol channels. |
| `SubscribeSymbolOhlcv` | `(symbols []string, interval market.Timeframe, onResponse) error` | OHLCV (`tick`/`1m`/`5m`). |
| `SubscribeSymbol` | `(symbols []string, onResponse) error` | Trade + quote + room. |
| `SubscribeBoard` / `SubscribeIndex` | `(boards []market.Board \| indices []string, onResponse) error` | Whole board / index constituents: trade + quote + room as `<type>.<code>@*` (the server fans out only with `@*`). |
| `SubscribeMarketFlag` | `(onResponse) error` | Session flags. |
| `SubscribeMarket` | `(symbols []string, onResponse) error` | Master data. |
| `SubscribeIndexTrade` | `(indices []string, interval market.Timeframe, onResponse) error` | Ticks of the index itself. |
| `SubscribeIndexSummary` | `(indices []string, onResponse) error` | `indexsummary.<code>`. |
| `SubscribeOrderStatus` / `SubscribePortfolio` | `(accountNo string, onResponse) error` | `order.<account>` / `portfolio.<account>` (account required). |
| `SubscribeOrders` / `SubscribeIndexTicks` / `SubscribeMasterData` / `SubscribeSessionFlag` | as the methods above | Clearer aliases. |
| `SubscribeTrades` / `Quotes` / `ForeignRoom` / `PutThrough` / `OddLot` | `(symbols []string, onResponse) error` | Aliases of the `SubscribeSymbol…` methods. |
| `SubscribeCandles` | `(symbols []string, interval market.Timeframe, onResponse) error` | Alias of `SubscribeSymbolOhlcv`. |
| `SubscribeExchangeTrades` | `(boards []market.Board, onResponse) error` | Alias of `SubscribeBoard`. |
| `SubscribeIndexConstituents` | `(indices []string, onResponse) error` | Alias of `SubscribeIndex`. |
| `SubscribePositions` | `(accountNo string, onResponse) error` | Alias of `SubscribePortfolio`. |
| `Unsubscribe…` | mirrors every `Subscribe…` without `onResponse` | Remove a subscription. |
| `UnsubscribeAll` | `() (map[string][]string, error)` | Drop everything. |
| `IsConnected` / `ReconnectCount` / `LastError` | `() bool` / `() int` / `() error` | Connection state. |

Callbacks (assign with `SetOn…`):

| Method | Callback type | Receives |
|---|---|---|
| `SetOnData` / `SetOnTrading` | `func(interface{})` | Every typed data / trading message (raw map for unknown frames). |
| `SetOnHeartbeat` | `func(*HeartbeatMessage)` | Heartbeats. |
| `SetOnMarketFlag` | `func(*MarketFlagMessage)` | Session flags. |
| `SetOnIndexSummary` | `func(*IndexSummaryMessage)` | `indexsummary.<code>`. |
| `SetOnOrder` | `func(*OrderStatusMessage)` | `orderEvent`: an order changed status. |
| `SetOnOrderMatch` | `func(*OrderMatchMessage)` | `orderMatchEvent`: a fill. |
| `SetOnPortfolio` | `func(*PortfolioMessage)` | Derivative position updates. |
| `SetOnConnection` | `func(transport.ConnectionEvent)` | connected / reconnected / disconnected / failed. |

## Trading WebSocket (EXPERIMENTAL)

Package `experimental`. `NewClient(rawURL, tokenProvider, privateKey, opts ...Option) (*Client, error)` then `NewService(client, cfg)`.
Options: `WithSignatureEncoding`, `WithRequestTimeoutMs`, `WithUserAgent`, `WithHandshakeGraceMs`, `WithRespectRateLimit`,
`WithMaxRateLimitWaitMs`, `WithPingIntervalMs` (all in milliseconds; `Call.Next(ctx, timeoutMs)`). Every command takes a `context.Context` first.

| Method | Signature | What it does |
|---|---|---|
| `Command` | `(ctx, method string, params interface{}) (*Response, error)` | Any command; first response frame, no retry. |
| `PlaceOrder` | `(ctx, accountNo, symbol, side, quantity, price, orderType, clientRequestID) (*trading.PlaceOrderResponse, error)` | `order.place`. |
| `AmendOrder` | `(ctx, accountNo string, price *float64, quantity *int, orderID, clientRequestID string)` | `order.amend` (price or quantity, not both). |
| `CancelOrder` | `(ctx, accountNo, orderID, clientRequestID string)` | `order.cancel` (exactly one id). |
| `BatchNewOrders` / `BatchCancelOrders` | `(ctx, orders …) (*Response, error)` | `order.batchNew` / `order.batchCancel`. |
| `PlaceFCO` / `CancelFCO` | `(ctx, params trading.FCOParamsRequest)` / `(ctx, fcoID, code string)` | `fco.place` / `fco.cancel`. |
| `ListFCO` | `(ctx, req *trading.FCOListRequest) (*Response, error)` | `fco.list`. |
| `FCOOrderBook` / `FCOStatusHistory` | `(ctx, fcoID[, pageIndex, pageSize])` | `fco.orderbook` / `fco.statusHistory`. |
| `QueryPosition` | `(ctx, accountNo string, querySummary bool)` | `query.position`. |
| `QueryMaxBuySell` | `(ctx, accountNo, symbol string, price *float64)` | `query.maxBuySell`. |
| `QueryAccountBalance` / `QueryPPMMRAccount` | `(ctx, accountNo string)` | `query.accountBalance` / `query.ppmmrAccount`. |
| `QueryOrderBook` | `(ctx, q OrderBookQuery)` | `query.orderBook`. |
| `SubscribeOrderEvents` / `UnsubscribeOrderEvents` | `(accountNo string) error` | `order.<account>` events. |
| `SubscribePortfolioEvents` / `UnsubscribePortfolioEvents` | `(accountNo string) error` | `portfolio.<account>` events. |
| `UnsubscribeAllEvents` | `() error` | Drop every trading subscription. |
| `Client` / `Close` | `() *Client` / `()` | Underlying socket (`SetOnEvent`, `Rejection`, `RateLimit`) / close and wake waiters. |

## Server error codes

`ssisdk.ServerErrorCode` lists the codes the servers return (alias names share a number, e.g. the Smart OTP and refresh
codes). `code.IsKnown()` and `ssisdk.LookupServerErrorCode(n)` tell whether a number is listed; an unknown code is never an error.
