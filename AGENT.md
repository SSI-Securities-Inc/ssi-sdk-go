# AGENT.md — AI Agent Integration Guide for ssi-sdk-go

Instructions, patterns and an API cheatsheet for AI coding assistants (Claude, Gemini, Cursor, Copilot, ...) using `ssi-sdk-go` (v3.3.0).
Full method list: `API_METHODS.md`. Usage guide (Vietnamese): `README.md`. History and breaking changes: `CHANGELOG.md`.

---

## 1. Overview & Architecture

`ssi-sdk-go` is a Go SDK for SSI's **FastConnect v3 API**:
- Authentication & token management (OTP, Smart OTP, refresh, single-flight auto-refresh)
- Market data (OHLC, indexes, securities info & summary, master data)
- Trading & portfolio (orders, batch orders, FCO conditional orders, balances, positions, PPMMR)
- Realtime streaming (WebSocket market data & trading events, supervised with auto-reconnect)
- EXPERIMENTAL Trading WebSocket (`experimental` package)

```
Facade (ssi: Auth / Data / Trading / Stream)
   └── Services (market, account, portfolio, trading, stream, experimental)
        └── Transport (transport.RestClient / transport.WebSocketClient)
             └── internal/{signature, redact, device, ratelimit, util}
```

Module `github.com/SSI-Securities-Inc/ssi-sdk-go/v3` (root package `ssisdk`, imported as `fc` in the docs; go 1.22; only dependency: gorilla/websocket).

### Constraints for AI agents
1. **Facade**: create clients with `ssi.NewAuth(cfg)`, `ssi.NewData(auth)`, `ssi.NewTrading(auth)`, `ssi.NewStream(auth)`. Fields: `data.MarketData`, `trading.Trading`, `trading.Account`, `trading.Portfolio`, `stream.Streaming`, `auth.TokenManager`.
2. **Typed enums**: `OrderSide`, `OrderType`, `OrderStatus`, `FCOType`, `FCOOperator`, `FCOStatus` (package `trading`), `Board`, `Timeframe` (package `market`). Use the constants, not raw strings.
3. **Errors, not panics**: every call returns `error`. Use `errors.As` against the types in section 5. Never compare error strings.
4. **Optional server values are pointers** (`nil` = the server did not send it). Do not substitute `0`.
5. **Blocking calls**: services take no `context` (except `experimental`); cancel by closing the client.
6. **Do not mutate or share `Config` assumptions**: the SDK works on a copy; blank endpoints fall back to defaults.

---

## 2. Authentication & Setup

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"

config := ssi.NewConfig(os.Getenv("SSI_CLIENT_ID"))
config.APIKey     = os.Getenv("SSI_API_KEY")
config.APISecret  = os.Getenv("SSI_API_SECRET")
config.PrivateKey = os.Getenv("SSI_PRIVATE_KEY") // Base64 RSA key; required to place orders
// Fail fast when a variable is missing. Never hardcode a default secret.

auth := ssi.NewAuth(config)
defer auth.Close()

// Normal OTP (SMS/email) or a code read from the Smart OTP app: one call.
if _, err := auth.Authenticate(otp); err != nil { log.Fatal(err) }

// Smart OTP push approval: request, then let the SDK poll until approved.
otp, err := auth.RequestOTPTyped()                  // otp.TransactionID
token, err := auth.EnsureAuthenticated("", otp.TransactionID)

data, trading, stream := ssi.NewData(auth), ssi.NewTrading(auth), ssi.NewStream(auth)
```

- `Authenticate` calls the server **once** (a pending Smart OTP gives `*SmartOTPPendingError`). Polling lives in `EnsureAuthenticated` / `auth.TokenManager.EnsureAuthenticatedWith(ctx, opts)`.
- Tokens count as expired 30s early; unknown expiry = expired; concurrent refreshes make **one** server call. No refreshable session -> `*ReauthenticationRequiredError` (the token is cleared; ask for a new OTP).
- `Config`, `Token`, `OTPResponse` never print secrets. If a token was ever logged by an older SDK version, revoke it.
- Market data does not need an OTP: `auth.Authenticate("")`.

### Config fields worth knowing
`APIURL`, `StreamingURL`, `TradingAPIDomain` (REST trading host), `TradingWSDomain` (experimental), `MaxBatchOrders` (20), `UserAgent`, `AutoReconnect` (true), `OTPPollIntervalMs`, `OTPPollMaxWaitMs`, `Proxy` (REST), `RateLimitPerSecond`, `TimeoutMs`, `RetryDelayMs`. Every duration the SDK defines is an integer in **milliseconds** with an `Ms` suffix (`Wait(*int)`, `Ping(.., *int)`, `ListSubscription(int)`, `WithRequestTimeoutMs`, ...). There is **no** `DeviceID`: the order `deviceId` is this machine's id read from the OS.

---

## 3. Public API Cheatsheet

All methods return `error` last. Prices are `float64`, sent as plain decimal strings.

### 3.1 Market data (`data.MarketData`)

| Method | Parameters | Return | Notes |
|---|---|---|---|
| `GetOHLC1Minute` / `3Minute` / `5Minute` / `15Minute` / `1Hour` | `symbol` | `[]OHLCData` | Latest trading day |
| `GetOHLC{1,3,5,15}MinuteHistorical`, `GetOHLC1HourHistorical`, `GetOHLC1DayHistorical` | `symbol, fromDate, toDate, page, size` | `[]OHLCData` | `page`,`size` >= 1; bare dates completed to `00:00:00`/`23:59:59` |
| `GetOHLC1WeekHistorical` / `GetOHLC1MonthHistorical` | same | - | Always `ValidationError` (server rejects) |
| `GetOHLCAll` | `symbol, tf, OHLCQuery{FromDate, ToDate, Size, Ascending}` | `[]OHLCData` | All pages |
| `IterOHLC` | `symbol, tf, OHLCQuery, yield func(OHLCData) bool` | - | Return `false` to stop |
| `DownloadOHLC1Minute` / `1Day` | `symbol` | - | `ErrNotImplemented` |
| `GetIndexes`, `GetIndexesByBoard(board)` | | `[]MarketIndexes` | |
| `GetIndexSummary(index)`, `...Historical(index, date)` | | `*MarketIndexSummary` | `nil, nil` if none |
| `GetBoardSummary(board)`, `...Historical(board, date)` | | `*MarketIndexSummary` | `nil, nil` if none |
| `GetSecuritiesInfo(symbol)` | | `*SecuritiesInfo` | `nil, nil` if none |
| `GetSecuritiesInfoByIndex(index)`, `ByBoard(board)` | | `[]SecuritiesInfo` | |
| `GetSecuritiesSummary(symbol)`, `...Historical(symbol, from, to)` | | `[]SecuritiesSummary` | |
| `GetSecuritiesSummaryByIndex(index)`, `...Historical(index, from, to)` | | `[]SecuritiesSummary` | |
| `GetMasterData()`, `GetMasterDataHistorical(from, to)` | | `[]MasterData` | Ceiling/floor/reference |

### 3.2 Account & portfolio (`trading.Account`, `trading.Portfolio`)

| Service | Method | Return |
|---|---|---|
| Account | `GetAccountInfo()` | `[]Account` (`AccountType` is a pointer) |
| Portfolio | `GetEquityBalance(acc)` / `GetDerivativeBalance(acc)` | `*EquityAccountBalance` / `*DerivativeAccountBalance` |
| Portfolio | `GetEquityPositions(acc)` | `[]EquityPosition` |
| Portfolio | `GetDerivativePositions(acc)` | `*AllDerivativePosition` |
| Portfolio | `GetOpenDerivativePositions(acc)` / `GetClosedDerivativePositions(acc)` | `[]DerivativePosition` |
| Portfolio | `GetEquityPPMMR(acc)` / `GetDerivativePPMMR(acc)` | `*EquityPPMMR` / `*DerivativePPMMR` |
| Portfolio | `GetTodayOrders(acc)` / `GetHistoricalOrders(acc, from, to)` | `[]Order` (`Price`+`PriceText`, `OSQuantity *int`) |

### 3.3 Orders (`trading.Trading`)

Order options: `trading.WithClientRequestID(id)` (idempotency key, <= 20 chars; generated when omitted). A `PrivateKey` is required: no key -> `ValidationError`, nothing is sent.

| Method | Parameters | Return |
|---|---|---|
| `PlaceLimitOrder` | `acc, symbol, side, qty, price, ...OrderOption` | `*PlaceOrderResponse` |
| `PlaceMarketOrder` / `PlaceATOOrder` / `PlaceATCOrder` | `acc, symbol, side, qty, ...OrderOption` | `*PlaceOrderResponse` |
| `PlaceOrder` | `acc, symbol, side, qty, price, orderType, ...OrderOption` | `*PlaceOrderResponse` |
| `ModifyOrderPrice` / `ModifyOrderQuantity` | `acc, clientRequestID, price \| qty` | `*ModifyOrderResponse` |
| `ModifyOrderPriceByOrderID` / `ModifyOrderQuantityByOrderID` | `acc, orderID, price \| qty` | `*ModifyOrderResponse` |
| `CancelOrder` / `CancelOrderByOrderID` | `acc, clientRequestID \| orderID` | `*CancelOrderResponse` |
| `PlaceBatchOrders` / `CancelBatchOrders` | `[]BatchPlaceOrderItem` / `[]BatchCancelOrderItem` | `*BatchOrderResponse` (<= `MaxBatchOrders`, all-or-nothing validation) |
| `GetMaxBuySell` | `acc, symbol, price *float64` | `*MaxBuySellResponse` (`PurchasingPower *int`) |
| `GetMaxBuySellAtMarketPrice` | `acc, symbol` | `*MaxBuySellResponse` |

Rules enforced locally: `quantity` > 0 integer; `LO`/`PLO` need `price > 0`; modify takes exactly one of price/quantity; cancel takes exactly one id; ATO/ATC/MTL send price `"0"`.

### 3.4 FCO (`trading.Trading`)

FCO options: `trading.WithOTPCode(code)`. Dates: `YYYY/MM/DD HH:MM:SS`, `from <= to`, window <= 31 days.

| Method | Key parameters |
|---|---|
| `PlaceFcoGtd` | `acc, symbol, side, qty, price (number or OrderType), priceSlip, from, to, ...FCOOption` |
| `PlaceFcoStop` | `acc, symbol, side, qty, stopPrice, operator, from, to, ...` |
| `PlaceFcoStopLimit` | `acc, symbol, side, qty, price, priceSlip, stopPrice, operator, from, to, ...` |
| `PlaceFcoTrailingStop` | `acc, symbol, side, qty, activePrice, trailingAmount, from, to, ...` |
| `PlaceFcoTrailingStopLimit` | `acc, symbol, side, qty, activePrice, trailingAmount, priceSlip, from, to, ...` |
| `PlaceFcoOco` | `acc, symbol, side, qty, tpActivePrice, slActivePrice, tpPrice, slPrice, tpSlip, slSlip, from, to, ...` |
| `PlaceFcoBullBear` | `acc, symbol, side, qty, price, priceSlip, tpActivePrice, slActivePrice, tpPrice, slPrice, tpSlip, slSlip, from, to, ...` |
| `CancelFco` | `fcoID, ...FCOOption` |
| `GetFcoByAccountNo` / `BySymbol` / `ByStatus` / `ByStatuses` / `ByDate` | `acc, ..., pageIndex, pageSize` -> `*FCOListResponse` |
| `GetFcoById` | `acc, fcoID` -> `*FCOInfo` (`nil, nil` if none) |
| `GetFcoOrderBook` | `fcoID, pageIndex, pageSize` |
| `GetFcoStatusHistory` | `fcoID` -> `[]FCOStatusHistoryItem` |

### 3.5 Streaming (`stream.Streaming`)

- `Connect()` returns after the first connect; the socket is then supervised (backoff + jitter, fresh token per connect, proactive reconnect before token expiry, automatic resubscribe). Disable with `config.AutoReconnect = false`.
- Topics are validated (invalid -> `ValidationError`); intervals accept only `tick`/`1m`/`5m`; requests are split under 4096 bytes.
- `SubscribeOrderStatus(accountNo, h)` / `SubscribePortfolio(accountNo, h)`: `""` means all accounts; a given account is validated.
- FCO changes arrive on `order.<account>` as `*stream.FCOOrderUpdateMessage`. There is no `SubscribeFcoOrderStatus`.
- Callbacks: `SetOnData`, `SetOnTrading` (typed messages; raw `map[string]interface{}` for unknown frames), `SetOnHeartbeat`, `SetOnMarketFlag`, `SetOnIndexSummary`, `SetOnOrder`, `SetOnOrderMatch`, `SetOnPortfolio`, `SetOnConnection`.
- State: `IsConnected()`, `ReconnectCount()`, `LastError()`; `ListSubscription(timeout)`, `UnsubscribeAll()`.
- Parsers: `stream.ParseDataMessage(frame)` / `stream.ParseTradingMessage(frame)` take the whole frame.

### 3.6 EXPERIMENTAL Trading WebSocket (`experimental`)

Signed order/FCO commands over `config.TradingWSURL()` (`wss://` only). Not used by `Trading`/`Stream`; the API may change.
`experimental.NewClient(url, tokenProvider, privateKey, opts...)` + `experimental.NewService(client, cfg)`. Commands take `context.Context`; mutating commands without a key fail closed; a timed-out command is never retried; signature encoding is `auto` (base64, one hex retry on 401006/401008/401013).

---

## 4. Code Examples

### Place and cancel a GTD FCO order
```go
auth := ssi.NewAuth(config)
defer auth.Close()
if _, err := auth.Authenticate(otp); err != nil { log.Fatal(err) }

t := ssi.NewTrading(auth)
from := time.Now().Format("2006/01/02") + " 09:00:00"
to := time.Now().AddDate(0, 0, 7).Format("2006/01/02") + " 15:00:00"

res, err := t.Trading.PlaceFcoGtd("1234561", "SSI", trading.OrderSideBuy, 100, trading.OrderTypeMTL, 0.5, from, to)
if err != nil { log.Fatal(err) }
if _, err := t.Trading.CancelFco(res.FCOID); err != nil { log.Fatal(err) }
```

### Streaming with typed callbacks
```go
s := ssi.NewStream(auth)
defer s.Disconnect()

s.Streaming.SetOnData(func(msg interface{}) {
	if m, ok := msg.(*stream.TradeMessage); ok {
		fmt.Printf("[TRADE] %s %.2f x %d\n", m.Symbol, m.Price, m.Quantity)
	}
})
s.Streaming.SetOnOrder(func(m *stream.OrderStatusMessage) { fmt.Println(m.OrderID, m.Status) })
s.Streaming.SetOnConnection(func(e transport.ConnectionEvent) { log.Println("ws", e.State) })

if err := s.Connect(); err != nil { log.Fatal(err) }
_ = s.Streaming.SubscribeSymbolTrade([]string{"SSI"}, nil)
_ = s.Streaming.SubscribeOrderStatus("1234561", nil)
s.Wait(nil)
```

---

## 5. Errors

All embed `SSIError` (`Message`, `Code`, `StatusCode`, `ResponseBody`, `Headers`) and unwrap to their parent, so `errors.As` works at any level.

| Type | When |
|---|---|
| `AuthenticationError` | Bad credentials / token |
| `SmartOTPPendingError` | HTTP 202 + 401114: poll again with the same transaction id |
| `SmartOTPRejectedError` | 401113 / 401115 / 401116: approval can no longer succeed |
| `ReauthenticationRequiredError` | Session cannot be refreshed; token cleared |
| `APIError` | API error; `Code` is the server code, `StatusCode` the HTTP status |
| `DuplicateRequestError` | HTTP 409: request id already used (the original order may be live: never retry under a new id) |
| `RateLimitError` | 429; `RetryAfterMs` (ms), `Limit`, `Remaining`, `Reset` (server numbers, unconverted) |
| `ValidationError` | Local input validation failed (nothing was sent) |
| `WebSocketError` / `TradingWSError` | WebSocket failures |

Error text never contains the raw response body, URL, token, account number or signed params.

---

## 6. Rules & Guidelines for AI Code Generators

1. **FCO dates**: `"YYYY/MM/DD HH:MM:SS"`, `from <= to`, window <= 31 days. (`util.BeginningOfDay()`/`EndOfDay()` are in `internal/util`: external code must format dates itself.)
2. **Never hardcode secrets**: read API key/secret/private key/OTP from the environment or a vault, fail fast when missing, never log them, never put them in tests or examples. Customer account numbers in examples must be placeholders.
3. **Use enum constants** (`trading.OrderSideBuy`, `trading.OrderTypeLO`, `trading.FCOOperatorGreaterOrEqual`), not raw strings.
4. **Orders are money paths**: pass an idempotency key you can reconcile (`WithClientRequestID`); on a timeout or `DuplicateRequestError` query the order book before resubmitting; never replay a `POST` blindly.
5. **Check pointers** for optional fields before dereferencing (`OSQuantity`, `PurchasingPower`, `TradeMessage.Open`, `PortfolioMessage.MarketPrice`, ...).
6. **Do not disable TLS verification** or bypass the signing path, even in dev code.
7. **Human decision authority**: the SDK can place and cancel real orders. Trading/limit approval stays with the responsible human; do not auto-submit orders from generated code without that sign-off.
8. **Clean builds**: run `gofmt -l .`, `go build ./...`, `go vet ./...` and `go test ./...` after every change.
