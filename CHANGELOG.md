# Changelog

All notable changes to `ssi-sdk-go`. The version lives in `version.go` (`ssisdk.Version`).

Items marked **BREAKING** change behaviour or signatures callers may depend on; the rest are additive or fixes.
This release brings the Go SDK to feature parity with `ssi-sdk-python` 3.3.0. Go-specific notes: services return
`error` (no exceptions), optional server fields are pointers, and unknown options are caught at compile time
through typed fields instead of a runtime `TypeError`.

## 3.3.2 - 2026-10-20

### Added
- Corporate actions (`GET /api/v3/data/corporateActions`) as two `market.Service` methods:
  `GetCorporateActions(opts ...CorporateActionsOption)` for today and
  `GetCorporateActionsHistorical(fromDate, toDate string, opts ...CorporateActionsOption)` for a past range (the server
  allows at most 5 days per request). Options: `WithCorporateActionsSymbol(symbol)` (default: every security) and
  `WithCorporateActionsLanguage("vn"|"en")` (default `"vn"`). Both fetch every page (`pageSize` 100, by `pagesCount`) and
  return `[]CorporateAction` (`Symbol`, `EventTitle`, `ExrightDate`, `RecordDate`, `IssueDate`, `Exchange`, `Ratio`, all
  strings as the server sends them; a null or missing field is `""`). Checked on the client before any call: date format,
  `from <= to`, 5-day span, symbol, language. Written from the official API doc; not yet run against the live server.

### Changed
- **BREAKING:** Trading WebSocket is no longer experimental. It is now a normal public package, `tradingws`, with the
  same names, signatures, defaults and behaviour (`TradingWS`, `NewClient`, `NewService`, `Client`, `Service`,
  `Response`, `Option`/`With...`, config `TradingWSDomain` / `TradingWSURL()`). It is still an explicit opt-in:
  `Trading` and `Stream` do not connect to it. `TradingWSError` stays in the root package (`ssisdk`).
  - Old import: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/experimental`
  - New import: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/tradingws`
  - The old package was removed without an alias or a deprecated re-export, so old imports fail to compile.
  - Migration:
    ```go
    import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/tradingws"   // was .../v3/experimental
    svc, err := tradingws.TradingWS(ctx, auth)                       // was experimental.TradingWS(ctx, auth)
    defer svc.Close()                                                // everything else keeps its name
    ```

## 3.3.0 - 2026-10-08

### Security
- Secrets no longer reach logs, `%v`/`%+v`/`%#v` output or error text: `Config`, `auth.Token`, `TokenRequest`,
  `OTPRequest`, `RefreshTokenRequest` and `OTPResponse` implement `String()`/`GoString()` and hide `APIKey`,
  `APISecret`, `PrivateKey`, proxy credentials, tokens, OTP and `transaction_id` (`internal/redact`).
- The REST client logs only method and path; the WebSocket client no longer logs the `Authorization` header, raw
  frames or URL query strings; `APIError` no longer echoes the raw response body or request URL.
- `NewAuth(config)` works on a copy (`Config.Normalized()`); the caller's `Config` is never mutated.
- A malformed proxy URL fails closed (no silent direct connection). The token-refresh call is an ordinary REST call: it carries the stored
  `Authorization` header like every other request (same as the Python SDK; it is never logged).
- TLS verification is always on. The experimental Trading WebSocket accepts `wss://` only and never puts the token in an error.
- **Action for the owner:** older versions could write a Bearer token to the log through the WebSocket client;
  revoke/rotate any token that was logged.

### Added
- `auth`: `EnsureAuthenticated` / `EnsureAuthenticatedWith(ctx, EnsureOptions)` (Smart OTP polling, single-flight
  refresh: concurrent callers make one server call), `RequestOTPTyped` + `OTPResponse`, `PollSmartOTP`,
  `CanRefresh`, `HasRefreshToken`, `IsRefreshTokenExpired`, `ConnectRestOnly`.
- Errors: `SmartOTPPendingError` (HTTP 202 + 401114), `SmartOTPRejectedError` (401113/401115/401116),
  `ReauthenticationRequiredError`, `DuplicateRequestError` (HTTP 409), `TradingWSError`; `RateLimitError` carries
  `StatusCode`, `RetryAfterMs`, `Limit`, `Remaining`, `Reset`. All of them unwrap to their parent (`errors.As`).
  `constant.go` holds the Smart OTP and refresh code sets (`SmartOTPTerminalCodes`, `RefreshReauthCodes`); any
  refresh-token-unusable code (401101-401104, 401106-401108) forces re-authentication.
- `Config`: `TradingAPIDomain` (+ `TradingAPIURL()`; the REST client routes `/api/v3/trading/*` and `/api/v3/account/*`
  there and shares the token and rate limiter), `TradingWSDomain` (+ `TradingWSURL()`, `wss://<domain>/ws/v3/trading`),
  `UserAgent`, `AutoReconnect`, `OTPPollIntervalMs`, `OTPPollMaxWaitMs`, `MaxBatchOrders` (default 20), `Proxy`.
  Blank values fall back to the defaults in `constant.go`. `Proxy` applies to REST; the experimental Trading WebSocket
  uses the standard proxy environment variables.
- Market data: `IterOHLC` (callback iterator) / `GetOHLCAll`; `OHLCQuery` struct; `securitiesSummary` and `masterdata`
  follow `pagesCount`; foreign-trade fields on `MarketIndexSummary`; `StockType` on `SecuritiesInfo`; optional
  `Ceiling`/`Floor`/`RefPrice` on `SecuritiesSummary`; `AllowedTimeframes`. Minute/hour OHLC with a `to` after today is
  cut to end of today; bare dates are completed to `00:00:00` / `23:59:59`; `page`/`size` < 1 are refused locally.
- Trading: `PlaceBatchOrders` / `CancelBatchOrders` (`POST`/`DELETE /order/batch`, at most `Config.MaxBatchOrders`,
  signed over the whole body, `BatchOrderResponse`); `GetFcoStatusHistory`; `WithClientRequestID` option on every
  `Place*Order` (idempotency key; one is generated when omitted); `WithOTPCode` option on every `PlaceFco*` and `CancelFco`;
  `PayloadBuilder` builds and validates every request body; signing verifies its own result; one shared FCO field parser
  (`ParseFCOCommon`) for REST and stream models; `OrderStatus` lists every status the servers emit.
- Streaming: typed callbacks `SetOnMarketFlag`, `SetOnIndexSummary`, `SetOnOrder`, `SetOnOrderMatch`, `SetOnPortfolio`
  (alongside `SetOnData`/`SetOnTrading`, which receive everything, typed); `SetOnConnection` (`ConnectionEvent`:
  connected / reconnected / disconnected / failed); `IsConnected`, `ReconnectCount`, `LastError`; `ListSubscription`,
  `SubscribeMarket`, `SubscribeIndexTrade`, `SubscribeIndexSummary`, `UnsubscribeOrderStatus`, `UnsubscribePortfolio`,
  `UnsubscribeAll`; clearer aliases `SubscribeOrders`, `SubscribeIndexTicks`, `SubscribeMasterData`,
  `SubscribeSessionFlag` and their `Unsubscribe...`. New messages `OrderMatchMessage`, `MarketDataMessage`,
  `IndexSummaryMessage`, `IndexTickMessage`; position fields on `PortfolioMessage`; open/high/low/average on `TradeMessage`.
  Topic codes are upper-cased and validated (`stream/topic.go`).
- Streaming connections are supervised: reconnect with capped exponential backoff and jitter, a fresh token on every
  connect, a proactive reconnect before the token expires, `{"code","msg"}` refusal frames honoured (401/403: refresh once;
  429: wait at least 30s), automatic resubscribe, old socket always closed before a new one opens. Subscribe frames are
  split under the server's 4096-byte limit.
- REST client follows the server's rate-limit headers: `X-RateLimit-Limit` caps the rate (never above
  `Config.RateLimitPerSecond`), `X-RateLimit-Remaining: 0` rests one second, and a `429` on a `GET` is waited out
  (`Retry-After`, else 1s doubling; at most 3 retries, never over 30s). `POST`/`PUT`/`DELETE` are never replayed.
- New package `experimental` (EXPERIMENTAL Trading WebSocket): signed order/FCO commands over `Config.TradingWSURL()`.
  Not used by `Trading`/`Stream`. Frame `{"id","method","params","signature","timestamp"}`; the params text is signed and
  embedded verbatim; mutating commands without a private key fail closed. `WithSignatureEncoding` (`auto` default: base64
  first, a refusal with 401006/401008/401013 is re-sent once as hex and remembered; `base64`; `hex`), `WithHandshakeGrace`
  (post-upgrade `{"code","msg"}` refusal fails `Connect`), `WithRespectRateLimit` / `WithMaxRateLimitWait` (waits `resetMs`
  at `remaining == 0`, capped at 2s), `WithPingIntervalMs`, `WithRequestTimeoutMs`, `WithUserAgent`. A timed-out command is
  never retried. Services: place/amend/cancel order, batch new/cancel, place/cancel/list FCO, order book, status history,
  `QueryPosition`/`QueryMaxBuySell`/`QueryAccountBalance`/`QueryPPMMRAccount`/`QueryOrderBook`, order and portfolio
  event subscriptions.
- `ssi` facade: `NewAuth`, `NewData`, `NewTrading`, `NewStream` and `Client` helpers.
- Unit tests for every package (signing, token manager incl. concurrency, REST via `httptest`, reconnect/backoff,
  topics, message parsing, payload builders, experimental client over a fake connection). No network is needed.
- `API_METHODS.md` (list of every public method) and an updated `AGENT.md` / `README.md`.

- `ServerErrorCode` enum (`constant.go`, 29 distinct codes, alias names share a number) with `IsKnown()` and
  `LookupServerErrorCode(code)`; an unknown code is reported as not known and never fails.
- Streaming aliases (same names as Python): `SubscribeTrades`, `SubscribeQuotes`, `SubscribeForeignRoom`,
  `SubscribePutThrough`, `SubscribeOddLot`, `SubscribeCandles`, `SubscribeExchangeTrades`,
  `SubscribeIndexConstituents`, `SubscribePositions` and their `Unsubscribe...`, plus `UnsubscribeSessionFlag`.
- `ssi.Auth.EnsureAuthenticatedWith(ctx, auth.EnsureOptions)` exposes the Smart OTP poll options on the wrapper.

### Changed
- **BREAKING: every duration the SDK defines is an integer number of milliseconds with an `Ms` suffix.** The old
  names no longer exist (compile error, no aliases). Defaults changed with the unit (60 s -> 60000 ms, ...).
  `RateLimitError.Limit`/`Remaining`/`Reset` keep the numbers the server sent (not durations the SDK converts).
  The `Retry-After` header and the WebSocket `retryAfter` field are in seconds on the wire and are converted to ms.

  | Old | New |
  |---|---|
  | `Config.Timeout` (s) | `Config.TimeoutMs` (default 60000) |
  | `Config.RetryDelay` (s, float) | `Config.RetryDelayMs` (default 2000) |
  | `Config.OTPPollInterval` (s, float) | `Config.OTPPollIntervalMs` (default 5000) |
  | `Config.OTPPollMaxWait` (s, float) | `Config.OTPPollMaxWaitMs` (default 60000) |
  | `RateLimitError.RetryAfter` (`*float64`, s) | `RateLimitError.RetryAfterMs` (`*int`) |
  | `NewRateLimitError(msg, *float64)` | `NewRateLimitError(msg, *int)` (ms) |
  | `auth.EnsureOptions.PollInterval` (`*time.Duration`) | `auth.EnsureOptions.PollIntervalMs` (`*int`) |
  | `TokenManager.PollSmartOTP(id, time.Duration, n)` | `PollSmartOTP(id, intervalMs int, n)` |
  | `auth.SmartOTPPollInterval` | `auth.SmartOTPPollIntervalMs` (5000) |
  | `ssi.DefaultTimeout` / `DefaultRetryDelay` | `DefaultTimeoutMs` / `DefaultRetryDelayMs` |
  | `DefaultOTPPollInterval` / `DefaultOTPPollMaxWait` | `DefaultOTPPollIntervalMs` / `DefaultOTPPollMaxWaitMs` |
  | `TokenExpirySkewSeconds` | `TokenExpirySkewMs` (30000) |
  | `RateLimitDefaultWait` / `RateLimitMaxWait` | `RateLimitDefaultWaitMs` (1000) / `RateLimitMaxWaitMs` (30000) |
  | `WSReconnectMaxDelay` / `WSRateLimitBackoff` / `WSTokenRefreshLead` | `WSReconnectMaxDelayMs` / `WSRateLimitBackoffMs` / `WSTokenRefreshLeadMs` |
  | `transport.BackoffDelay` / `RateLimitWait` / `SecondsUntilRefresh` | `BackoffDelayMs` / `RateLimitWaitMs` / `MsUntilRefresh` |
  | `ClassifyRejection` second result `*float64` (s) | `*int` (ms) |
  | `Stream.Wait` / `Service.Wait` / `WebSocketClient.Wait(*time.Duration)` | `Wait(timeoutMs *int)` |
  | `Service.Ping(cb, *time.Duration)` | `Ping(cb, intervalMs *int)` |
  | `Service.ListSubscription(time.Duration)` | `ListSubscription(timeoutMs int)` |
  | `experimental.WithRequestTimeout` / `WithHandshakeGrace` / `WithMaxRateLimitWait` / `WithPingInterval` | `...Ms(int)` |
  | `experimental.Call.Next(ctx, time.Duration)` | `Next(ctx, timeoutMs int)` |
- The REST client's DEBUG log carries only the method and path. Request bodies and params (which can hold account
  numbers) are no longer logged, even redacted.
- Token refresh checks that a refresh token exists and is usable before it checks the API credentials, so a session
  without a refresh token reports "No refresh token available" like Python.
- `CancelFco` sends `userAgent` only when `Config.UserAgent` is set; the place payloads still send the default.
- FCO price words ATO/ATC/MTL/MP/MOK/MAK (a string or an `OrderType`, any case, trimmed) always send `priceSlip` 0;
  `LO`/`PLO` and numeric prices keep the slip.
- `portfolio.Order`: side, order type and status are matched case-insensitively and canonicalised; an unknown value is
  kept as sent. A price word in `PriceText` is upper-cased.
- Experimental `QueryOrderBook` validates `from` before `to`, always in that order.
- **BREAKING** `Config.DeviceID` is removed. The `deviceId` of every order/FCO is this machine's id read from the OS
  (`internal/device`: macOS `IOPlatformUUID`, Linux machine-id, Windows `MachineGuid`, else MAC / host name; raw and
  stable). The placeholder MAC is gone. `trading.DefaultDeviceID` / `DefaultUserAgent` are removed.
- **BREAKING** `Authenticate(otp, transactionID...)` is a single call. Smart OTP polling moved to `EnsureAuthenticated`.
- **BREAKING** `EnsureAuthenticated` treats a token with unknown expiry as expired, expires tokens 30s early and returns
  `ReauthenticationRequiredError` when the session cannot be refreshed (401101/401103 clear the stored token).
- **BREAKING** Only `GET` is retried on timeout. `POST`/`PUT`/`DELETE` (auth, OTP, orders, FCO) are sent exactly once.
- **BREAKING** `APIError.Code` is the server's error code when the body has one (HTTP status stays in `StatusCode`).
  HTTP 200 carrying a `code` other than 200/204 is an `APIError`; the empty-result marker `{"code": 204}` yields empty results.
- **BREAKING** `Account.AccountType` is a pointer (`nil` when the server omits it; unknown type kept as the raw string).
- **BREAKING** `Download*` return `ErrNotImplemented` (the server has no `data/file` endpoint yet).
- **BREAKING** `GetOHLC1WeekHistorical` / `GetOHLC1MonthHistorical` return `ValidationError` (the server rejects them, 400210).
- **BREAKING** `GetIndexSummary*`, `GetBoardSummary*` and `GetSecuritiesInfo` return `nil, nil` when nothing is found.
  Optional market fields are pointers.
- **BREAKING** OHLC reads and `IterOHLC`/`GetOHLCAll` use the `OHLCQuery` struct.
- **BREAKING** `portfolio.Order` has `Price` + `PriceText` (word prices ATO/ATC/MP) and `OSQuantity *int`; position/balance
  sections are `nil` when the server sends nothing; `DerivativePositionFromList` / `PositionFromMap` take the account number;
  `DerivativeAccountBalance.CashWithdrawable*` is now `Withdrawable*` (old accessors kept, deprecated). JSON tags are
  removed from the portfolio, trading and stream models (they are parsed explicitly from the server maps).
- **BREAKING** `trading`: request `ToJSON()` returns `(string, error)`; `MaxBuySellRequest.ToMap()` returns an error;
  `MaxBuySellResponse.PurchasePower` (string) is now `PurchasingPower *int` and `MarginRatio *string`; FCO optional fields
  are pointers; `FCOInfo.PriceSlip` is `*float64`; `CancelFco(fcoID, ...FCOOption)` accepts an OTP code; `LO`/`PLO` with
  `price <= 0` is rejected locally.
- **BREAKING** `transport.WebSocketClient` constructor takes options; `Connect` returns after the first connect and then
  supervises the socket; `SetOnData`/`SetOnTrading` deliver typed messages (raw map for unknown frames); `OnResponse`
  callbacks receive acks and no longer replace the data/trading/heartbeat handlers; interval topics accept only
  `tick`/`1m`/`5m`; topics are validated before sending (invalid -> `ValidationError`).
- **BREAKING** Stream parsers are `ParseDataMessage(msg)` / `ParseTradingMessage(msg)` and take the whole frame.
  `OrderStatusMessage` has `Price` + `PriceText` and `ModifiedTime` (was `ModifyTime`); `PortfolioMessage` is reshaped
  (pointer fields); optional numbers are pointers.
  `SubscribeOrderStatus` / `SubscribePortfolio` take an `accountNo` and validate it.
- **BREAKING** `Order.OSQuantity` is `nil` (the server never sends it); `FilledQuantity` and `CancelQuantity` read
  `filledQty` / `cancelQty`.
- FCO: dates must be `YYYY/MM/DD HH:MM:SS` with `from <= to` and a window of at most 31 days; `fco/cancel` sends `deviceId`;
  operators given as `0..4` in responses are parsed; response models tolerate unknown enum values.
- Prices are sent as plain decimal strings (no exponent notation) through `util.FormatPrice`; `quantity` must be a positive
  integer. Market-priced helpers (ATO/ATC/MTL) keep sending `"0"`. Prices stay `float64` in the Go API (public contract);
  they are never concatenated into SQL or compared for money totals by the SDK.
- `GetMaxBuySell` reads `purchasingPower`; derivative responses no longer invent `0` for fields the server leaves out.
  Numeric parsing accepts `"1234.0"` and `"2,493,089"` and rejects inf/nan.
- `RequirePositive` / `RequireNonNegative` reject NaN; new `RequireSymbol`, `RequireExactlyOne`, `RequireDateRange`.

### Deprecated
- `MaxBuySellResponse.PurchasePower` -> `PurchasingPower`; `DerivativeAccountBalance.CashWithdrawable*` -> `Withdrawable*`;
  `PortfolioMessage.TotalAsset`/`CashBalance`/`StockValue` (the server never sends them).

### Removed
- `SubscribeFcoOrderStatus` (the server has no separate FCO order-status topic; FCO changes arrive on `order.<account>`
  as `FCOOrderUpdateMessage`).

### Fixed
- `portfolio.EquityPosition` now carries `T1BuyQuantity` / `T2BuyQuantity` (JSON `t1BuyQuantity` / `t2BuyQuantity`, which
  the server's positions response already returned; a missing field is 0, a numeric string is parsed to `int`).
- `SubscribeBoard`/`SubscribeIndex` (and their `Unsubscribe...`) now send `trade.<code>@*`, `quote.<code>@*` and
  `room.<code>@*`. By the server's routing (`ConnectionManager.IsBasketIndexPattern`/`IsExchangePattern`) a basket
  index or exchange is only fanned out to its symbols with the `@*` interval; the bare `trade.VN30`/`trade.HOSE`
  was acked "ok" but never matched a tick. The topic builder accepts `@*` (`stream.StreamIntervalAll`) for `trade`/`quote`/
  `room`/`put`/`oddlot` with one concrete code. Based on reading the server code (its public docs still show the bare
  form); not yet confirmed against a live server.
- A comma list in one topic (`VNM,FPT`) is rejected with `ValidationError`: the server never splits on commas, so it
  was acked "ok" and matched nothing. Pass a slice of filters instead (one topic each).
- Server quirks the SDK now shields callers from: `/data/ohlc` 400213 on a bare date, minute/hour OHLC with a future `to`,
  `page`/`size` < 1 not validated by the server, a 600-symbol subscribe exceeding the 4096-byte frame limit.
- `market.FLAG` (tail upper-cased) is recognised as the session flag; topic types are matched case-insensitively.
- `trade.index.<code>` ticks are `IndexTickMessage`, not a symbol `TradeMessage`.
- `Disconnect` stops the ping loop (`StopPingLoop`); a non-JSON-object frame no longer breaks the receive loop; the `PING_PONG`
  ack no longer breaks heartbeat parsing.
- `Sign` returns `ValidationError` for a missing or malformed `PrivateKey` (fail closed, no key material in the message).

### Not implemented (by design, pending decisions; same as Python)
- Bulk OHLC download (`DownloadOHLC*`: the server has no `data/file` endpoint yet).
- A dedicated `fcoEvent` stream (same as Python): FCO changes arrive as `FCOOrderUpdateMessage` on `order.<account>`.
- Python-only items with no Go equivalent: sync/async duality (Go calls are blocking; cancel with the `context` passed to
  the experimental client), `__getattr__` delegation and IDE-typing aliases (`PriceLike`, `NumberLike`, ...),
  `TypedDict`s (Go uses structs).
