# SSI Go SDK

Go SDK cho nền tảng giao dịch chứng khoán SSI. Hỗ trợ REST API và WebSocket streaming.

Phiên bản 3.3.0. Thay đổi và breaking changes: [CHANGELOG.md](CHANGELOG.md). Danh sách method: [API_METHODS.md](API_METHODS.md).

[![Go Reference](https://pkg.go.dev/badge/github.com/SSI-Securities-Inc/ssi-sdk-go.svg)](https://pkg.go.dev/github.com/SSI-Securities-Inc/ssi-sdk-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/SSI-Securities-Inc/ssi-sdk-go?logo=go)](https://github.com/SSI-Securities-Inc/ssi-sdk-go)
[![Release](https://img.shields.io/github/v/release/SSI-Securities-Inc/ssi-sdk-go)](https://github.com/SSI-Securities-Inc/ssi-sdk-go/releases)
[![License](https://img.shields.io/github/license/SSI-Securities-Inc/ssi-sdk-go)](https://github.com/SSI-Securities-Inc/ssi-sdk-go/blob/main/LICENSE)

## Mục lục

- [Cài đặt](#cài-đặt)
- [Cấu hình](#cấu-hình)
- [Kiến trúc Client](#kiến-trúc-client)
- [Xác thực](#1-xác-thực)
- [Tài khoản](#2-tài-khoản)
- [Dữ liệu thị trường](#3-dữ-liệu-thị-trường)
- [Danh mục đầu tư](#4-danh-mục-đầu-tư)
- [Giao dịch](#5-giao-dịch)
- [Streaming realtime](#6-streaming-realtime)
- [Xử lý lỗi](#7-xử-lý-lỗi)
- [Cấu hình nâng cao](#8-cấu-hình-nâng-cao)
- [API Reference](#api-reference)

---

## Cài đặt

```bash
go get github.com/SSI-Securities-Inc/ssi-sdk-go/v3
```

---

## Cấu hình

Tạo đối tượng `Config` với thông tin xác thực:

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"

config := ssi.NewConfig("YOUR_CLIENT_ID")
config.APIKey      = "YOUR_API_KEY"
config.APISecret   = "YOUR_API_SECRET"
config.PrivateKey  = "YOUR_PRIVATE_KEY"  // chỉ cần cho Trading
```

**Tất cả tuỳ chọn cấu hình:**

| Tham số | Kiểu | Mặc định | Mô tả |
|---------|------|----------|-------|
| `ClientID` | `string` | *(bắt buộc)* | Client ID xác thực |
| `APIKey` | `string` | `""` | API key từ SSI |
| `APISecret` | `string` | `""` | API secret từ SSI |
| `PrivateKey` | `string` | `""` | Private key để ký lệnh giao dịch |
| `APIURL` | `string` | `"https://api.ssi.com.vn"` | URL REST API |
| `StreamingURL` | `string` | `"wss://stream.ssi.com.vn/ws/v3"` | URL WebSocket streaming |
| `TimeoutMs` | `int` | `60000` | Timeout request (mili-giây). Chỉ `GET` được retry khi timeout; `POST`/`PUT`/`DELETE` (auth, OTP, lệnh) gửi đúng một lần |
| `MaxRetries` | `int` | `5` | Số lần retry tối đa |
| `RetryDelayMs` | `int` | `2000` | Delay cơ sở giữa các lần retry (exponential backoff, mili-giây) |
| `RateLimitPerSecond` | `int` | `10` | Giới hạn request/giây (0 = không giới hạn) |
| `LogLevel` | `string` | `"INFO"` | Mức log: DEBUG, INFO, WARNING, ERROR |
| `Proxy` | `string` | `""` | URL proxy HTTP/HTTPS cho REST. URL sai -> lỗi (không tự chuyển sang kết nối trực tiếp) |
| `TradingAPIDomain` | `string` | `"https://api.ssi.com.vn"` | Host của REST trading (`/api/v3/trading/*`, `/api/v3/account/*`). Mặc định theo `APIURL` |
| `TradingWSDomain` | `string` | `"wss://api.ssi.com.vn/ws/v3/trading"` | WebSocket đặt lệnh (gói `experimental`), chỉ `wss://` |
| `MaxBatchOrders` | `int` | `20` | Số lệnh tối đa mỗi batch (lớn hơn bị từ chối ngay tại SDK) |
| `UserAgent` | `string` | `""` | User-Agent cho HTTP và payload lệnh |
| `AutoReconnect` | `bool` | `true` | Tự kết nối lại WebSocket streaming khi bị rớt |
| `OTPPollIntervalMs` | `int` | `5000` | Mili-giây giữa các lần poll Smart OTP |
| `OTPPollMaxWaitMs` | `int` | `60000` | Tổng thời gian poll Smart OTP tối đa (mili-giây) |

Giá trị trống hoặc không hợp lệ được thay bằng mặc định; `Config` của bạn không bị sửa. `deviceId` của mọi lệnh là mã máy đọc từ hệ điều hành (không còn `Config.DeviceID`).

> `Config`, `Token`, `OTPResponse`... không in secret khi dùng `%v`/`%+v`/`%#v`. Nếu trước đây token từng bị ghi vào log, hãy thu hồi/rotate token đó.

---

## Kiến trúc Client

SDK sử dụng kiến trúc **modular** gồm 4 client chuyên biệt:

| Client | Mô tả | Yêu cầu |
|--------|-------|---------|
| `Auth` | Xác thực & quản lý token. Entry point cho tất cả clients khác. | `Config` |
| `Data` | Dữ liệu thị trường (OHLC, chỉ số, chứng khoán) | `Auth` (không cần OTP) |
| `Trading` | Giao dịch + danh mục + tài khoản | `Auth` (cần OTP) |
| `Stream` | Streaming realtime qua WebSocket | `Auth` (cần OTP) |

**Luồng khởi tạo:**

```
Config → Auth → Authenticate(otp=...) → Data / Trading / Stream
```

`Auth` là client gốc — quản lý REST client và token. Các client `Data`, `Trading`, `Stream` đều nhận `Auth` làm tham số và chia sẻ chung HTTP connection pool.

**Services được cung cấp bởi mỗi client:**

| Client | Service / Field | Truy cập | Mô tả |
|--------|---------|---------|-------|
| `Auth` | `TokenManager` | `auth.TokenManager` | Xác thực, OTP, refresh token |
| `Data` | `MarketData` | `data.MarketData` | OHLC, chỉ số, chứng khoán, securities summary |
| `Trading` | `Trading` | `trading.Trading` | Đặt/sửa/huỷ lệnh, sức mua/bán |
| `Trading` | `Account` | `trading.Account` | Thông tin tài khoản |
| `Trading` | `Portfolio` | `trading.Portfolio` | Số dư, vị thế, sổ lệnh, PPMMR |
| `Stream` | `Streaming` | `stream.Streaming` | Subscribe/unsubscribe realtime data |

### Chỉ dùng dữ liệu thị trường (không cần OTP)

```go
auth := ssi.NewAuth(config)
defer auth.Close()

// Xác thực với chuỗi rỗng khi không dùng OTP
_, err := auth.Authenticate("")
if err != nil {
    log.Fatal(err)
}

data := ssi.NewData(auth)
```

---

## 1. Xác thực

`Auth` là entry point duy nhất. Tất cả clients khác nhận `*Auth` làm tham số.

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"

auth := ssi.NewAuth(config)
defer auth.Close()
```

### 1.1. Xác thực với OTP (cho Trading & Stream)

```go
token, err := auth.Authenticate("222222")
if err != nil {
    log.Fatal(err)
}
log.Printf("Access token: %s", token.AccessToken)

// Smart OTP Push Notification: truyền transactionID thu được từ RequestOTP
token, err := auth.Authenticate("", "TRANSACTION_ID")
```

`Authenticate` chỉ gọi server **một lần**. Nếu Smart OTP chưa được approve, nó trả `*fc.SmartOTPPendingError`; muốn SDK tự poll, dùng `EnsureAuthenticated` (mục 1.3).

### 1.2. Yêu cầu gửi OTP

```go
result, err := auth.RequestOTP()
if err != nil {
    log.Fatal(err)
}
log.Println(result)
```

**Tài khoản đã kích hoạt Smart OTP có 2 cách để xác thực:**

1. **Approve trên app** — gọi `RequestOTP()` để bắn yêu cầu lên thiết bị,
   lấy `transactionId`, rồi dùng `EnsureAuthenticated("", transactionId)` để
   SDK tự poll cho đến khi user bấm approve.
2. **Lấy mã trực tiếp trên app** — mở app Smart OTP, đọc mã hiển thị sẵn, rồi
   điền thẳng mã đó vào `EnsureAuthenticated("123456")` (hoặc `Authenticate("123456")`)
   — **không cần** gọi `RequestOTP()` trước.

### 1.3. Tự động đảm bảo xác thực (EnsureAuthenticated)

```go
// 1) Refresh nếu có refresh token còn dùng được — không cần otp/transactionID
accessToken, err := auth.EnsureAuthenticated("")

// 2) OTP thường (SMS/email) hoặc mã Smart OTP lấy trực tiếp trên app
accessToken, err := auth.EnsureAuthenticated("222222")

// 3) Smart OTP dạng push-approval (truyền transactionID lấy từ RequestOTP)
otpResult, _ := auth.RequestOTP()
// Lấy transactionID từ otpResult["data"]
accessToken, err := auth.EnsureAuthenticated("", transactionID)
```

Hành vi: token hết hạn trước 30 giây được coi là đã hết hạn, token không rõ hạn dùng coi như hết hạn; nhiều goroutine cùng refresh chỉ tạo **một** request tới server (single-flight). Khi không thể refresh, SDK xoá token và trả `*fc.ReauthenticationRequiredError` — cần OTP mới. Smart OTP bị từ chối/hết hạn trả `*fc.SmartOTPRejectedError`. Dùng `EnsureAuthenticatedWith(ctx, opts)` (có trên cả `auth.TokenManager` và `*ssi.Auth`) khi cần `context` và tuỳ chỉnh poll.

```go
otp, err := auth.RequestOTPTyped() // otp.TransactionID chỉ có với Smart OTP
```

### 1.4. Làm mới token

```go
token, err := auth.Refresh()
if err != nil {
    log.Fatal(err)
}
```

### 1.5. Kiểm tra trạng thái token

```go
if auth.TokenManager.IsTokenExpired() {
    auth.Refresh()
}

accessToken := auth.AccessToken()
```

---

## 2. Tài khoản

`Trading` client cung cấp thông tin tài khoản qua `trading.Account`. Yêu cầu OTP.

```go
auth := ssi.NewAuth(config)
defer auth.Close()

if _, err := auth.Authenticate("222222"); err != nil {
    log.Fatal(err)
}

t := ssi.NewTrading(auth)
```

### 2.1. Lấy danh sách tài khoản

```go
accounts, err := t.Account.GetAccountInfo()
if err != nil {
    log.Fatal(err)
}
for _, acc := range accounts {
    fmt.Printf("%s (%s)\n", acc.AccountNo, acc.AccountType)
}
```

---

## 3. Dữ liệu thị trường

`Data` client cung cấp dữ liệu qua `data.MarketData`. Không cần OTP.

```go
auth := ssi.NewAuth(config)
defer auth.Close()

if _, err := auth.Authenticate(""); err != nil {
    log.Fatal(err)
}

data := ssi.NewData(auth)
```

### 3.1. Dữ liệu OHLC (nến)

**Lấy dữ liệu trong ngày:**

| Method | Timeframe |
|--------|-----------|
| `GetOHLC1Minute(symbol)` | 1 phút |
| `GetOHLC3Minute(symbol)` | 3 phút |
| `GetOHLC5Minute(symbol)` | 5 phút |
| `GetOHLC15Minute(symbol)` | 15 phút |
| `GetOHLC1Hour(symbol)` | 1 giờ |

```go
ohlc, err := data.MarketData.GetOHLC1Minute("SSI")
if err != nil {
    log.Fatal(err)
}
for _, candle := range ohlc {
    fmt.Printf("%s: O=%.0f H=%.0f L=%.0f C=%.0f V=%d\n",
        candle.TradingDate, candle.OpenPrice, candle.HighPrice,
        candle.LowPrice, candle.ClosePrice, candle.Volume)
}
```

**Lấy dữ liệu lịch sử:**

| Method | Timeframe |
|--------|-----------|
| `GetOHLC1MinuteHistorical(symbol, fromDate, toDate, page, size)` | 1 phút |
| `GetOHLC3MinuteHistorical(symbol, fromDate, toDate, page, size)` | 3 phút |
| `GetOHLC5MinuteHistorical(symbol, fromDate, toDate, page, size)` | 5 phút |
| `GetOHLC15MinuteHistorical(symbol, fromDate, toDate, page, size)` | 15 phút |
| `GetOHLC1HourHistorical(symbol, fromDate, toDate, page, size)` | 1 giờ |
| `GetOHLC1DayHistorical(symbol, fromDate, toDate, page, size)` | 1 ngày |
| `GetOHLC1WeekHistorical(symbol, fromDate, toDate, page, size)` | 1 tuần *(trả `*fc.ValidationError`: server từ chối 1w, mã 400210)* |
| `GetOHLC1MonthHistorical(symbol, fromDate, toDate, page, size)` | 1 tháng *(trả `*fc.ValidationError`: server từ chối 1M, mã 400210)* |

Ngày chỉ có `YYYY/MM/DD` được SDK bổ sung thành `00:00:00` / `23:59:59` (server trả 400213 nếu thiếu giờ). Với nến phút/giờ, `to` ở tương lai được cắt về cuối ngày hôm nay. `page`/`size` < 1 bị từ chối ngay tại SDK.

**Lấy toàn bộ nến trong một khoảng (tự duyệt các trang):**

```go
q := market.OHLCQuery{FromDate: "2026/03/01", ToDate: "2026/03/31", Size: 1000}
all, err := data.MarketData.GetOHLCAll("SSI", market.TimeframeDay1, q)

// hoặc duyệt từng nến, return false để dừng sớm
err = data.MarketData.IterOHLC("SSI", market.TimeframeDay1, q, func(c market.OHLCData) bool {
    fmt.Println(c.TradingDate, c.ClosePrice)
    return true
})
```

`DownloadOHLC1Minute` / `DownloadOHLC1Day` trả `market.ErrNotImplemented` (server chưa có endpoint tải hàng loạt). `GetIndexSummary*`, `GetBoardSummary*`, `GetSecuritiesInfo` trả `nil, nil` khi không có dữ liệu.

```go
ohlc, err := data.MarketData.GetOHLC1DayHistorical(
    "SSI", "2026/03/27", "2026/04/22", 1, 100,
)
```

**Tham số historical:** `symbol`, `fromDate`/`toDate` (yyyy/MM/dd), `page` (bắt đầu từ 1), `size` (tối đa 1000).

### 3.2. Danh sách chỉ số thị trường

```go
indexes, err := data.MarketData.GetIndexes()

import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
indexes, err := data.MarketData.GetIndexesByBoard(market.BoardHOSE)
```

### 3.3. Tổng hợp chỉ số (Index Summary)

```go
summary, err := data.MarketData.GetIndexSummary("VNINDEX")
summary, err := data.MarketData.GetIndexSummaryHistorical("VNINDEX", "2026/01/15")
summary, err := data.MarketData.GetBoardSummary(market.BoardHOSE)
summary, err := data.MarketData.GetBoardSummaryHistorical(market.BoardHOSE, "2026/01/15")
```

### 3.4. Thông tin chứng khoán

```go
info, err := data.MarketData.GetSecuritiesInfo("SSI")
securities, err := data.MarketData.GetSecuritiesInfoByIndex("VN30")
securities, err := data.MarketData.GetSecuritiesInfoByBoard(market.BoardHOSE)
```

### 3.5. Tổng hợp chứng khoán (Securities Summary)

```go
summary, err := data.MarketData.GetSecuritiesSummary("SSI")
summary, err := data.MarketData.GetSecuritiesSummaryHistorical("SSI", "2026/03/01", "2026/03/31")
summary, err := data.MarketData.GetSecuritiesSummaryByIndex("VN30")
summary, err := data.MarketData.GetSecuritiesSummaryByIndexHistorical("VN30", "2026/03/01", "2026/03/31")
```

### 3.6. Master Data (giá trần/sàn/tham chiếu)

SDK tự động phân trang để lấy đầy đủ tất cả các mã:

```go
// Hôm nay
masterData, err := data.MarketData.GetMasterData()

// Khoảng ngày tự chọn
masterHist, err := data.MarketData.GetMasterDataHistorical("2026/08/05", "2026/08/06")

for _, item := range masterData {
    fmt.Printf("%s %s: trần=%.2f sàn=%.2f TC=%.2f\n", item.Board, item.Symbol, item.Ceiling, item.Floor, item.RefPrice)
}
```

---

## 4. Danh mục đầu tư

`Trading` client cung cấp thông tin danh mục đầu tư qua `trading.Portfolio`. Yêu cầu OTP.

```go
auth := ssi.NewAuth(config)
defer auth.Close()

if _, err := auth.Authenticate("222222"); err != nil {
    log.Fatal(err)
}

t := ssi.NewTrading(auth)
```

### 4.1. Số dư tài khoản

```go
balance, err := t.Portfolio.GetEquityBalance("1234561")
derBalance, err := t.Portfolio.GetDerivativeBalance("1234568")
```

### 4.2. Vị thế (Positions)

```go
positions, err := t.Portfolio.GetEquityPositions("1234561")
derPositions, err := t.Portfolio.GetDerivativePositions("1234568")
openPos, err := t.Portfolio.GetOpenDerivativePositions("1234568")
closedPos, err := t.Portfolio.GetClosedDerivativePositions("1234568")
```

### 4.3. Sổ lệnh (Order Book)

```go
orders, err := t.Portfolio.GetTodayOrders("1234561")
orders, err := t.Portfolio.GetHistoricalOrders("1234561", "2026/01/01", "2026/01/31")
```

### 4.4. PPMMR (Purchasing Power / Margin Maintenance Ratio)

```go
ppmmr, err := t.Portfolio.GetEquityPPMMR("1234561")
derPPMMR, err := t.Portfolio.GetDerivativePPMMR("1234568")
```

---

## 5. Giao dịch

`Trading` client cung cấp khả năng đặt, sửa và huỷ lệnh qua `trading.Trading`. Yêu cầu OTP.

```go
auth := ssi.NewAuth(config)
defer auth.Close()

if _, err := auth.Authenticate("222222"); err != nil {
    log.Fatal(err)
}

t := ssi.NewTrading(auth)
```

### 5.1. Đặt lệnh

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"

result, err := t.Trading.PlaceLimitOrder("1234561", "SSI", trading.OrderSideBuy, 100, 66000)
result, err := t.Trading.PlaceMarketOrder("1234561", "SSI", trading.OrderSideBuy, 100)
result, err := t.Trading.PlaceATOOrder("1234561", "SSI", trading.OrderSideBuy, 100)
result, err := t.Trading.PlaceATCOrder("1234561", "SSI", trading.OrderSideSell, 100)
result, err := t.Trading.PlaceOrder("1234561", "SSI", trading.OrderSideBuy, 100, 66000, trading.OrderTypeLO)

// Idempotency key do bạn kiểm soát (tối đa 20 ký tự); bỏ trống thì SDK tự sinh một key riêng cho mỗi lệnh
result, err := t.Trading.PlaceLimitOrder("1234561", "SSI", trading.OrderSideBuy, 100, 66000,
    trading.WithClientRequestID("my-order-001"))
```

Giá là `float64` và được gửi dạng chuỗi thập phân thường. `LO`/`PLO` với giá `<= 0`, số lượng không dương, hoặc thiếu `PrivateKey` đều bị từ chối ngay tại SDK (`*fc.ValidationError`), không gửi lên server.

### 5.2. Sửa lệnh

```go
result, err := t.Trading.ModifyOrderPrice("1234561", "REQ123", 68000)
result, err := t.Trading.ModifyOrderPriceByOrderID("1234561", "ORD123", 68000)
result, err := t.Trading.ModifyOrderQuantity("1234561", "REQ123", 200)
result, err := t.Trading.ModifyOrderQuantityByOrderID("1234561", "ORD123", 200)
```

### 5.3. Huỷ lệnh

```go
result, err := t.Trading.CancelOrder("1234561", "REQ123")
result, err := t.Trading.CancelOrderByOrderID("1234561", "ORD123")
```

### 5.4. Sức mua/bán tối đa

```go
price := 66000.0
maxBS, err := t.Trading.GetMaxBuySell("1234561", "SSI", &price)
maxBS, err := t.Trading.GetMaxBuySellAtMarketPrice("1234561", "SSI")
```

### 5.5. Lệnh điều kiện (Flexible Conditional Orders - FCO)

SDK hỗ trợ 7 loại lệnh điều kiện linh hoạt FCO, tra cứu danh sách lệnh FCO, nhật ký sổ lệnh FCO và hủy lệnh FCO:

```go
import (
    "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
    "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

accountNo := "1234561"
fromDate := util.BeginningOfDay() // "YYYY/MM/DD 00:00:00"
toDate := util.EndOfDay()         // "YYYY/MM/DD 23:59:59"

// 1. Đặt lệnh GTD (Good Till Date)
gtd, err := t.Trading.PlaceFcoGtd(
    accountNo, "SSI", trading.OrderSideBuy, 100, 25000, 500, fromDate, toDate,
)

// 2. Đặt lệnh Stop Market
stop, err := t.Trading.PlaceFcoStop(
    accountNo, "SSI", trading.OrderSideSell, 100, 24000, trading.FCOOperatorLesserOrEqual, fromDate, toDate,
)

// 3. Đặt lệnh Stop Limit
stopLimit, err := t.Trading.PlaceFcoStopLimit(
    accountNo, "SSI", trading.OrderSideBuy, 100, 25500, 500, 25000, trading.FCOOperatorGreaterOrEqual, fromDate, toDate,
)

// 4. Đặt lệnh Trailing Stop
trailing, err := t.Trading.PlaceFcoTrailingStop(
    accountNo, "SSI", trading.OrderSideBuy, 100, 26000, 1000, fromDate, toDate,
)

// 5. Đặt lệnh Trailing Stop Limit
trailingLimit, err := t.Trading.PlaceFcoTrailingStopLimit(
    accountNo, "SSI", trading.OrderSideBuy, 100, 26000, 1000, 500, fromDate, toDate,
)

// 6. Đặt lệnh OCO (One-Cancels-the-Other)
oco, err := t.Trading.PlaceFcoOco(
    accountNo, "SSI", trading.OrderSideSell, 100, 30000, 24000, trading.OrderTypeMTL, trading.OrderTypeMTL, 500, 500, fromDate, toDate,
)

// 7. Đặt lệnh Bull Bear
bullBear, err := t.Trading.PlaceFcoBullBear(
    accountNo, "SSI", trading.OrderSideBuy, 1000, 100.5, 10, 30, 20, 30, 10, 0.1, 0, fromDate, toDate,
)

// 8. Tra cứu danh sách lệnh FCO
fcoList, err := t.Trading.GetFcoByAccountNo(accountNo, 1, 10)

// 9. Hủy lệnh FCO
cancelRes, err := t.Trading.CancelFco(gtd.FCOID)

// 10. Lệnh điều kiện có thể yêu cầu mã OTP: truyền qua option
gtd, err = t.Trading.PlaceFcoGtd(
    accountNo, "SSI", trading.OrderSideBuy, 100, 25000, 500, fromDate, toDate, trading.WithOTPCode("123456"),
)
cancelRes, err = t.Trading.CancelFco(gtd.FCOID, trading.WithOTPCode("123456"))

// 11. Lịch sử chuyển trạng thái của một lệnh FCO
history, err := t.Trading.GetFcoStatusHistory(gtd.FCOID)
```

Khi đặt FCO, ngày phải ở dạng `YYYY/MM/DD HH:MM:SS`, `from <= to` và khoảng tối đa 31 ngày (kiểm tra tại SDK).

| Method | Mô tả |
|--------|-------|
| `PlaceFcoGtd(...)` | Đặt lệnh GTD (Good Till Date) |
| `PlaceFcoStop(...)` | Đặt lệnh Stop Market |
| `PlaceFcoStopLimit(...)` | Đặt lệnh Stop Limit |
| `PlaceFcoTrailingStop(...)` | Đặt lệnh Trailing Stop Market |
| `PlaceFcoTrailingStopLimit(...)` | Đặt lệnh Trailing Stop Limit |
| `PlaceFcoOco(...)` | Đặt lệnh OCO (One Cancels the Other) |
| `PlaceFcoBullBear(...)` | Đặt lệnh Bull Bear |
| `CancelFco(fcoID)` | Hủy lệnh FCO theo `fcoID` |
| `GetFcoByAccountNo(...)` | Tra cứu danh sách FCO theo tài khoản |
| `GetFcoBySymbol(...)` | Tra cứu FCO lọc theo mã chứng khoán |
| `GetFcoByStatus(...)` | Tra cứu FCO lọc theo trạng thái |
| `GetFcoByDate(...)` | Tra cứu FCO lọc theo khoảng ngày |
| `GetFcoById(...)` | Lấy thông tin 1 lệnh FCO theo ID |
| `GetFcoOrderBook(...)` | Lấy lịch sử thực thi (Order Book) của FCO |


---

## 6. Streaming realtime

`Stream` client kết nối WebSocket qua `s.Streaming`. Yêu cầu OTP và gọi `Connect()` trước khi subscribe.

```go
auth := ssi.NewAuth(config)
defer auth.Close()

if _, err := auth.Authenticate("222222"); err != nil {
    log.Fatal(err)
}

s := ssi.NewStream(auth)
defer s.Disconnect()

if err := s.Connect(); err != nil {
    log.Fatal(err)
}
```

### 6.1. Thiết lập callback

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/stream"

s.Streaming.SetOnData(func(msg interface{}) {
    switch m := msg.(type) {
    case *stream.TradeMessage:
        fmt.Printf("[TRADE] %s | Price: %.0f | Qty: %d\n", m.Symbol, m.Price, m.Quantity)
    case *stream.IntervalMessage:
        fmt.Printf("[INTERVAL] %s | Open: %.0f | Close: %.0f | Vol: %d\n", m.Symbol, m.Open, m.Close, m.Volume)
    case *stream.QuoteMessage:
        fmt.Printf("[QUOTE] %s | Bids: %v | Asks: %v\n", m.Symbol, m.BidPrices, m.AskPrices)
    case *stream.ForeignRoomMessage:
        fmt.Printf("[ROOM]  %s | BuyQty: %d | SellQty: %d\n", m.Symbol, m.BuyQuantity, m.SellQuantity)
    }
})

s.Streaming.SetOnTrading(func(msg interface{}) {
    switch m := msg.(type) {
    case *stream.OrderStatusMessage:
        fmt.Printf("[ORDER] %s %s | Status: %s\n", m.Symbol, m.Side, m.Status)
    case *stream.FCOOrderUpdateMessage:
        fmt.Printf("[FCO ORDER] %s | ProcessStatus: %s\n", m.Symbol, m.ProcessStatus)
    case *stream.PortfolioMessage:
        fmt.Printf("[PORTFOLIO] %s | Net: %d\n", m.Symbol, m.Net)
    }
})

s.Streaming.SetOnHeartbeat(func(msg *stream.HeartbeatMessage) {
    fmt.Printf("[HEARTBEAT] %v\n", msg)
})
```

**Message types qua `SetOnData`:** `*stream.TradeMessage`, `*stream.IntervalMessage`, `*stream.QuoteMessage`, `*stream.ForeignRoomMessage`, `*stream.MarketStatusMessage`, `*stream.MarketFlagMessage`, `*stream.MarketDataMessage`, `*stream.IndexTickMessage`, `*stream.IndexSummaryMessage`, `*stream.PutMessage`, `*stream.OddLotMessage`. Frame không nhận ra được trả nguyên dạng `map[string]interface{}`.

**Message types qua `SetOnTrading`:** `*stream.OrderStatusMessage`, `*stream.OrderMatchMessage`, `*stream.PortfolioMessage`, `*stream.FCOOrderUpdateMessage`.

Số liệu tuỳ chọn là con trỏ (`nil` khi server không gửi), ví dụ `TradeMessage.Open`, `PortfolioMessage.MarketPrice`. Giá chữ (ATO/ATC/MP) nằm ở `OrderStatusMessage.PriceText`, `Price` là 0.

**Callback có kiểu riêng** (nhận cùng message với `SetOnData`/`SetOnTrading`, không cần type switch):

| Setter | Message |
|--------|---------|
| `SetOnMarketFlag` | `*stream.MarketFlagMessage` (phiên ATO/LO/ATC...) |
| `SetOnIndexSummary` | `*stream.IndexSummaryMessage` |
| `SetOnOrder` | `*stream.OrderStatusMessage` (lệnh đổi trạng thái) |
| `SetOnOrderMatch` | `*stream.OrderMatchMessage` (lệnh khớp) |
| `SetOnPortfolio` | `*stream.PortfolioMessage` |
| `SetOnConnection` | `transport.ConnectionEvent` (`State`: connected / reconnected / disconnected / failed) |

```go
s.Streaming.SetOnOrder(func(m *stream.OrderStatusMessage) { fmt.Println(m.Symbol, m.Status) })
s.Streaming.SetOnConnection(func(e transport.ConnectionEvent) { log.Println("ws:", e.State, e.ReconnectCount) })

s.Streaming.IsConnected()     // bool
s.Streaming.ReconnectCount()  // số lần mở lại socket
s.Streaming.LastError()       // lỗi làm kết nối dừng hẳn (nil khi bình thường)
```

**Tự kết nối lại:** `Connect()` trả về sau lần kết nối đầu tiên, sau đó SDK giám sát socket: backoff luỹ thừa có giới hạn + jitter, token mới mỗi lần kết nối, kết nối lại chủ động trước khi token hết hạn, tôn trọng frame từ chối `{"code","msg"}` (401/403 refresh một lần; 429 chờ tối thiểu 30 giây), tự subscribe lại, và luôn đóng socket cũ trước khi mở socket mới. Tắt bằng `config.AutoReconnect = false`. Khi `State == "failed"` (ví dụ cần OTP mới) sẽ không kết nối lại.

**Topic được kiểm tra trước khi gửi** (sai -> `*fc.ValidationError`); mã chỉ số/mã chứng khoán được viết hoa; khung interval chỉ nhận `tick`/`1m`/`5m`; yêu cầu subscribe dài được tách thành nhiều frame dưới 4096 byte.

### 6.2. Subscribe dữ liệu thị trường

```go
s.Streaming.SubscribeSymbol([]string{"SSI", "HPG", "VIC"}, nil)
s.Streaming.SubscribeSymbolTrade([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolQuote([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolRoom([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolPutThrough([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolOddLot([]string{"SSI"}, nil)

import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
s.Streaming.SubscribeBoard([]market.Board{market.BoardHOSE, market.BoardHNX}, nil) // trade/quote/room.<sàn>@*
s.Streaming.SubscribeIndex([]string{"VNINDEX", "VN30"}, nil)                       // trade/quote/room.<chỉ số>@*

s.Streaming.SubscribeMarketFlag(nil)                                       // cờ phiên: market.flag
s.Streaming.SubscribeMarket([]string{"SSI"}, nil)                          // master data (trần/sàn/tham chiếu)
s.Streaming.SubscribeIndexTrade([]string{"VN30"}, "", nil)                 // tick của chính chỉ số ("1m"/"5m" -> nến)
s.Streaming.SubscribeIndexSummary([]string{"VN30"}, nil)                   // indexsummary.<code>
s.Streaming.SubscribeSymbolOhlcv([]string{"SSI"}, market.TimeframeMinute1, nil)
```

`SubscribeBoard`/`SubscribeIndex` gửi `trade.<mã>@*` (+ `quote`, `room`): server chỉ rải sàn/rổ chỉ số ra các mã thành phần khi có `@*`.
Mỗi topic chỉ một mã (hoặc khoảng `A-B`); danh sách `VNM,FPT` trong một topic bị `ValidationError` vì server không tách dấu phẩy.

Có alias dễ đọc hơn: `SubscribeIndexTicks`, `SubscribeMasterData`, `SubscribeSessionFlag`, `SubscribeOrders`, `SubscribeTrades`, `SubscribeQuotes`, `SubscribeForeignRoom`, `SubscribePutThrough`, `SubscribeOddLot`, `SubscribeCandles`, `SubscribeExchangeTrades`, `SubscribeIndexConstituents`, `SubscribePositions` (đều có `Unsubscribe...` tương ứng).

### 6.3. Unsubscribe

```go
s.Streaming.UnsubscribeSymbol([]string{"SSI"})
s.Streaming.UnsubscribeSymbolTrade([]string{"SSI"})
s.Streaming.UnsubscribeBoard([]market.Board{market.BoardHOSE})
s.Streaming.UnsubscribeIndex([]string{"VNINDEX"})
s.Streaming.UnsubscribeOrderStatus("1234561")
s.Streaming.UnsubscribePortfolio("1234561")

subs, err := s.Streaming.ListSubscription(5000) // hỏi server đang subscribe gì (mili-giây)
left, err := s.Streaming.UnsubscribeAll()                  // bỏ tất cả
```

### 6.4. Subscribe giao dịch (order status & portfolio)

```go
s.Streaming.SubscribeOrderStatus("", nil)           // tất cả tài khoản ("" = "*")
s.Streaming.SubscribeOrderStatus("1234561", nil)     // tài khoản cụ thể (được kiểm tra định dạng)
s.Streaming.SubscribePortfolio("", nil)
```

Thay đổi của lệnh điều kiện (FCO) đến qua cùng topic `order.<account>` dưới dạng `*stream.FCOOrderUpdateMessage`; không còn `SubscribeFcoOrderStatus`.

### 6.5. Heartbeat & Wait

```go
s.Streaming.Ping(nil, nil)               // ping một lần
every := 20000 // mili-giây
s.Streaming.Ping(nil, &every)            // ping định kỳ (dừng khi Disconnect)

s.Wait(nil)                              // chờ vô thời hạn
timeout := 30000 // mili-giây
s.Wait(&timeout)                         // chờ 30 giây
```

### 6.6. Ví dụ streaming hoàn chỉnh

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"
    "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/stream"
)

func main() {
    config := ssi.NewConfig("YOUR_CLIENT_ID")
    config.APIKey     = "YOUR_API_KEY"
    config.APISecret  = "YOUR_API_SECRET"
    config.PrivateKey = "YOUR_PRIVATE_KEY"

    auth := ssi.NewAuth(config)
    defer auth.Close()

    if _, err := auth.Authenticate("222222"); err != nil {
        log.Fatal(err)
    }

    s := ssi.NewStream(auth)
    defer s.Disconnect()

    s.Streaming.SetOnData(func(msg interface{}) {
        switch m := msg.(type) {
        case *stream.TradeMessage:
            fmt.Printf("[TRADE] %s | %.0f | %d\n", m.Symbol, m.Price, m.Quantity)
        case *stream.QuoteMessage:
            fmt.Printf("[QUOTE] %s\n", m.Symbol)
        }
    })

    s.Streaming.SetOnTrading(func(msg interface{}) {
        fmt.Printf("[TRADING] %v\n", msg)
    })

    if err := s.Connect(); err != nil {
        log.Fatal(err)
    }

    s.Streaming.SubscribeSymbol([]string{"SSI", "HPG"}, nil)
    s.Streaming.SubscribeOrderStatus("", nil)

    timeout := 30000 // mili-giây
    s.Wait(&timeout)
}
```

---

## 7. Xử lý lỗi

| Error Type | Mô tả |
|------------|-------|
| `*fc.SSIError` | Base error cho tất cả lỗi SDK |
| `*fc.AuthenticationError` | Xác thực thất bại (sai credentials, token hết hạn) |
| `*fc.APIError` | API trả về lỗi — có thêm `StatusCode`, `ResponseBody` |
| `*fc.WebSocketError` | Lỗi kết nối hoặc giao tiếp WebSocket |
| `*fc.ValidationError` | Lỗi validate input |
| `*fc.RateLimitError` | Vượt quá giới hạn request — có thêm `RetryAfterMs` (mili-giây), `Limit`, `Remaining`, `Reset` (giữ nguyên số của server) |
| `*fc.SmartOTPPendingError` | Smart OTP chưa được approve (HTTP 202 + 401114); poll lại cùng `transactionId` |
| `*fc.SmartOTPRejectedError` | Smart OTP bị từ chối/hết hạn (401113/401115/401116) |
| `*fc.ReauthenticationRequiredError` | Không refresh được; token đã bị xoá, cần OTP mới |
| `*fc.DuplicateRequestError` | HTTP 409: `clientRequestId`/`batchRequestId` đã dùng. SDK không tự gửi lại với id khác vì lệnh gốc có thể vẫn còn hiệu lực |
| `*fc.TradingWSError` | Lỗi từ Trading WebSocket thử nghiệm (`Status`, `ErrorCode`, `ServerMessage`) |

Các lỗi con unwrap về lỗi cha, nên `errors.As` hoạt động cho cả hai (ví dụ `SmartOTPPendingError` cũng là `AuthenticationError`).

`APIError.Code` là mã lỗi của server (HTTP status nằm ở `StatusCode`). Thông điệp lỗi không chứa body gốc, URL hay token.

```go
import (
    "errors"

    fc "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

if err != nil {
    var (
        rl  *fc.RateLimitError
        re  *fc.ReauthenticationRequiredError
        dup *fc.DuplicateRequestError
        api *fc.APIError
        val *fc.ValidationError
    )
    switch {
    case errors.As(err, &re):
        log.Println("Cần OTP mới")
    case errors.As(err, &rl):
        if rl.RetryAfterMs != nil {
            log.Printf("Rate limited, retry sau %d ms", *rl.RetryAfterMs)
        } else {
            log.Println("Rate limited")
        }
    case errors.As(err, &dup):
        log.Println("Request id đã dùng — kiểm tra lệnh trước khi gửi lại")
    case errors.As(err, &api):
        log.Printf("API error %d (code %s): %s", api.StatusCode, api.Code, api.Message)
    case errors.As(err, &val):
        log.Printf("Validation: %s", val.Message)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

Với lệnh: mọi `Place*Order` có idempotency key (`trading.WithClientRequestID("...")`, tự sinh nếu bỏ trống). `POST`/`PUT`/`DELETE` không bao giờ được replay khi gặp `429` hoặc timeout. Giá được gửi dạng chuỗi thập phân thường (không có số mũ); lệnh `LO`/`PLO` với giá `<= 0` bị từ chối ngay tại SDK.

---

## 8. Cấu hình nâng cao

```go
config.Proxy              = "http://proxy.example.com:8080"
config.MaxRetries         = 3
config.RetryDelayMs       = 1000
config.RateLimitPerSecond = 5
config.MaxBatchOrders     = 20
config.TradingAPIDomain   = "https://api.ssi.com.vn"
```

SDK tự theo header rate-limit của server: `X-RateLimit-Limit` hạ tốc độ gửi (không bao giờ vượt `RateLimitPerSecond`), `X-RateLimit-Remaining: 0` nghỉ 1 giây, `429` trên `GET` được chờ (`Retry-After`, tối đa 30 giây, 3 lần) trước khi trả `RateLimitError`.

### Lệnh theo lô (batch)

```go
res, err := trading.Trading.PlaceBatchOrders([]tr.BatchPlaceOrderItem{
    {AccountNo: "1234561", Symbol: "VNM", Side: tr.OrderSideBuy, Quantity: 100, Price: 25.5, OrderType: tr.OrderTypeLO},
})
res, err = trading.Trading.CancelBatchOrders([]tr.BatchCancelOrderItem{{AccountNo: "1234561", ClientRequestID: "abc"}})
```

Tối đa `MaxBatchOrders` lệnh; chỉ cần một lệnh sai là cả lô bị từ chối tại SDK; cả body được ký một lần.

### Trading WebSocket (EXPERIMENTAL)

Gói `experimental` gửi lệnh đã ký qua WebSocket (`Config.TradingWSURL()`). Không được `Trading`/`Stream` dùng và API có thể đổi.

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/experimental"

ws, err := experimental.NewClient(config.TradingWSURL(),
    func() (string, error) { return auth.AccessToken(), nil }, config.PrivateKey)
if err != nil { log.Fatal(err) }
defer ws.Close()

svc := experimental.NewService(ws, config)
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
if err := ws.Connect(ctx); err != nil { log.Fatal(err) }

resp, err := svc.PlaceOrder(ctx, "1234561", "VNM", tr.OrderSideBuy, 100, 25.5, tr.OrderTypeLO, "")
```

- Lệnh làm thay đổi trạng thái cần `PrivateKey`; thiếu key thì lỗi ngay (không gửi).
- Chữ ký: `WithSignatureEncoding` (`auto` mặc định: base64 trước; bị từ chối 401006/401008/401013 thì gửi lại **một lần** dạng hex và nhớ lại).
- Lệnh bị timeout **không** được gửi lại. Frame từ chối `{"code","msg"}` sau khi upgrade làm `Connect` lỗi. Khi `remaining == 0` SDK chờ `resetMs` (tối đa 2 giây).
- Chỉ chấp nhận `wss://`. Token không bao giờ xuất hiện trong thông báo lỗi.
- Sự kiện (order/portfolio): `svc.SubscribeOrderEvents("1234561")`, nhận qua `ws.SetOnEvent(...)`.

---

## API Reference

### Enums

Tất cả enum của Go SDK được chia theo package tương ứng:

#### `trading.OrderSide`
Package: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading`

| Giá trị | Value | Mô tả |
|---------|-------|-------|
| `OrderSideBuy` | `"B"` | Mua |
| `OrderSideSell` | `"S"` | Bán |

#### `trading.OrderType`
Package: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading`

| Giá trị | Value | Mô tả |
|---------|-------|-------|
| `OrderTypeLO` | `"LO"` | Lệnh giới hạn (Limit Order) |
| `OrderTypeMTL` | `"MTL"` | Market To Limit |
| `OrderTypeMP` | `"MP"` | Market Price |
| `OrderTypeATO` | `"ATO"` | At The Open |
| `OrderTypeATC` | `"ATC"` | At The Close |
| `OrderTypeMOK` | `"MOK"` | Match Or Kill |
| `OrderTypeMAK` | `"MAK"` | Match And Kill |
| `OrderTypePLO` | `"PLO"` | Post Lunch Order |

#### `trading.OrderStatus`
Package: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading`

| Giá trị | Value | Mô tả |
|---------|-------|-------|
| `OrderStatusPending` | `"PD"` | Đang chờ |
| `OrderStatusPendingApproval` | `"WA"` | Chờ duyệt |
| `OrderStatusReady` | `"RS"` | Sẵn sàng |
| `OrderStatusSent` | `"SD"` | Đã gửi |
| `OrderStatusQueued` | `"QU"` | Đã xếp hàng |
| `OrderStatusFilled` | `"FF"` | Khớp toàn bộ |
| `OrderStatusPartialFilled` | `"PF"` | Khớp một phần |
| `OrderStatusPartialCancelled` | `"FFPC"` | Khớp một phần + huỷ phần còn lại |
| `OrderStatusPendingModify` | `"WM"` | Chờ sửa |
| `OrderStatusPendingCancel` | `"WC"` | Chờ huỷ |
| `OrderStatusCancelled` | `"CL"` | Đã huỷ |
| `OrderStatusRejected` | `"RJ"` | Bị từ chối |
| `OrderStatusExpired` | `"EX"` | Hết hạn |
| `OrderStatusPreSession` | `"IAV"` | Phiên trước |

#### `market.Board`
Package: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market`

| Sàn | Value | Mô tả |
|-----|-------|-------|
| `BoardHOSE` | `"HOSE"` | Sàn HOSE |
| `BoardHNX` | `"HNX"` | Sàn HNX |
| `BoardUPCOM` | `"UPCOM"` | Sàn UPCOM |

#### `account.AccountType`
Package: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/account`

| Giá trị | Value | Mô tả |
|---------|-------|-------|
| `AccountTypeEquity` | `"Cash"` | Tài khoản cơ sở |
| `AccountTypeEquityMargin` | `"Margin"` | Tài khoản ký quỹ |
| `AccountTypeDerivative` | `"Derivative"` | Tài khoản phái sinh |

#### `market.Timeframe`
Package: `github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market`

| Giá trị | Value | Mô tả |
|---------|-------|-------|
| `TimeframeMinute1` | `"1m"` | 1 phút |
| `TimeframeMinute3` | `"3m"` | 3 phút |
| `TimeframeMinute5` | `"5m"` | 5 phút |
| `TimeframeMinute15` | `"15m"` | 15 phút |
| `TimeframeHour1` | `"1h"` | 1 giờ |
| `TimeframeDay1` | `"1d"` | 1 ngày |
| `TimeframeWeek1` | `"1w"` | 1 tuần *(chưa được API v3 hỗ trợ)* |
| `TimeframeMonth1` | `"1M"` | 1 tháng *(chưa được API v3 hỗ trợ)* |

---

### Models

#### Authentication

**`auth.Token`** — Kết quả xác thực

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccessToken` | `string` | Token truy cập |
| `TokenType` | `string` | Loại token (mặc định `"Bearer"`) |
| `ExpiresAt` | `int64` | Thời điểm hết hạn (timestamp) |
| `RefreshToken` | `string` | Token làm mới |
| `RefreshTokenExpiresAt` | `int64` | Thời điểm refresh token hết hạn |

#### Account

**`account.Account`** — Thông tin tài khoản

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` | `string` | Số tài khoản |
| `AccountType` | `AccountType` | Loại tài khoản |

#### Market Data

**`market.OHLCData`** — Dữ liệu nến OHLC

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Symbol` | `string` | Mã chứng khoán |
| `TradingDate` | `string` | Ngày giao dịch |
| `OpenPrice` | `float64` | Giá mở cửa |
| `HighPrice` | `float64` | Giá cao nhất |
| `LowPrice` | `float64` | Giá thấp nhất |
| `ClosePrice` | `float64` | Giá đóng cửa |
| `Volume` | `int` | Khối lượng |
| `Value` | `float64` | Giá trị giao dịch |

**`market.MarketIndexes`** — Thông tin chỉ số

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Index` | `string` | Mã chỉ số |
| `IndexName` | `string` | Tên chỉ số |
| `Board` | `*Board` | Sàn giao dịch |

**`market.MarketIndexSummary`** — Tổng hợp chỉ số

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `TradingDate` | `string` | Ngày giao dịch |
| `IndexValue` | `float64` | Giá trị chỉ số |
| `IndexChange` | `float64` | Thay đổi |
| `IndexChangePercent` | `float64` | Thay đổi (%) |
| `TotalTrade` | `int` | Tổng KL giao dịch |
| `TotalTradeValue` | `float64` | Tổng giá trị giao dịch |
| `TotalMatch` | `int` | Tổng KL khớp lệnh |
| `TotalMatchValue` | `float64` | Tổng giá trị khớp lệnh |
| `TotalDeal` | `int` | Tổng KL thoả thuận |
| `TotalDealValue` | `float64` | Tổng giá trị thoả thuận |
| `TotalAdvanceStock` | `int` | Số mã tăng |
| `TotalDeclineStock` | `int` | Số mã giảm |
| `TotalSteadyStock` | `int` | Số mã đứng giá |
| `TotalCeilingStock` | `int` | Số mã trần |
| `TotalFloorStock` | `int` | Số mã sàn |
| `TotalPropBuy` | `int` | KL mua tự doanh |
| `TotalPropBuyValue` | `float64` | Giá trị mua tự doanh |
| `TotalPropSell` | `int` | KL bán tự doanh |
| `TotalPropSellValue` | `float64` | Giá trị bán tự doanh |

**`market.SecuritiesInfo`** — Thông tin chứng khoán

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Symbol` | `string` | Mã chứng khoán |
| `Board` | `*Board` | Sàn giao dịch |
| `Index` | `string` | Chỉ số |
| `SymbolNameVi` | `string` | Tên tiếng Việt |
| `SymbolNameEn` | `string` | Tên tiếng Anh |
| `LotSize` | `int` | Lô giao dịch |
| `MaturityDate` | `string` | Ngày đáo hạn |
| `FirstTradingDate` | `string` | Ngày giao dịch đầu tiên |
| `LastTradingDate` | `string` | Ngày giao dịch cuối cùng |
| `CWUnderlyingSymbol` | `string` | Mã CK cơ sở (CW) |
| `CWExercisePrice` | `float64` | Giá thực hiện (CW) |
| `CWExecutionRatio` | `float64` | Tỷ lệ chuyển đổi (CW) |
| `ListedShares` | `int` | Số CP niêm yết |
| `ICBCode` | `string` | Mã ngành ICB |
| `ICBName` | `string` | Tên ngành ICB |
| `IIndex` | `float64` | Chỉ số I |
| `INAV` | `float64` | NAV (ETF) |
| `OpenInterest` | `float64` | Khối lượng mở (phái sinh) |
| `SettlementPrice` | `float64` | Giá thanh toán (phái sinh) |

**`market.SecuritiesSummary`** — Tổng hợp chứng khoán

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Symbol` | `string` | Mã chứng khoán |
| `TradingDate` | `string` | Ngày giao dịch |
| `PriceChange` | `float64` | Thay đổi giá |
| `PriceChangePercent` | `float64` | Thay đổi giá (%) |
| `OpenPrice` | `float64` | Giá mở cửa |
| `HighPrice` | `float64` | Giá cao nhất |
| `LowPrice` | `float64` | Giá thấp nhất |
| `ClosePrice` | `float64` | Giá đóng cửa |
| `AveragePrice` | `float64` | Giá trung bình |
| `TotalMatch` | `int` | Tổng KL khớp |
| `TotalMatchValue` | `float64` | Tổng giá trị khớp |
| `TotalBuy` | `int` | Tổng KL mua |
| `TotalTradeBuy` | `float64` | Giá trị mua |
| `TotalSell` | `int` | Tổng KL bán |
| `TotalTradeSell` | `float64` | Giá trị bán |
| `TotalForeignBuy` | `int` | KL mua khối ngoại |
| `TotalForeignBuyValue` | `float64` | GT mua khối ngoại |
| `TotalForeignSell` | `int` | KL bán khối ngoại |
| `TotalForeignSellValue` | `float64` | GT bán khối ngoại |
| `RemainForeignRoom` | `int` | Room ngoại còn lại |
| `TotalForeignRoom` | `int` | Tổng room ngoại |
| `TotalDeal` | `int` | KL giao dịch thỏa thuận |
| `TotalDealValue` | `float64` | GT giao dịch thỏa thuận |
| `OpenInterest` | `float64` | Khối lượng mở (phái sinh) |
| `SettlementPrice` | `float64` | Giá thanh toán (phái sinh) |

#### Portfolio

**`portfolio.EquityAccountBalance`** — Số dư tài khoản cơ sở

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` | `string` | Số tài khoản |
| `AccountBalance` | `float64` | Số dư tài khoản |
| `TotalDebt` | `float64` | Tổng nợ |
| `InterestLoan` | `float64` | Lãi vay |
| `OverdueFeeLoan` | `float64` | Phí vay quá hạn |
| `Withdrawable` | `float64` | Tiền rút được |
| `OnHoldCash` | `float64` | Tiền tạm giữ |
| `SellUnmatched` | `float64` | Bán chưa khớp |
| `SellT0` / `SellT1` / `SellT2` | `float64` | Bán T+0/1/2 |
| `BuyUnmatched` | `float64` | Mua chưa khớp |
| `BuyT0` / `BuyT1` / `BuyT2` | `float64` | Mua T+0/1/2 |
| `AdvanceCashT0` / `AdvanceCashT1` | `float64` | Ứng trước T+0/1 |
| `HoldSubscription` | `float64` | Giữ đăng ký |
| `Dividend` | `float64` | Cổ tức chờ về/đã ghi nhận |

**`portfolio.DerivativeAccountBalance`** — Số dư tài khoản phái sinh

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` | `string` | Số tài khoản |
| `AccountBalance` | `float64` | Số dư tài khoản |
| `Fee` / `Commission` / `Interest` | `float64` | Phí / Hoa hồng / Lãi |
| `ExtInterest` | `float64` | Lãi ngoài |
| `Loan` | `float64` | Khoản vay |
| `DeliveryAmount` | `float64` | Giá trị chuyển giao |
| `FloatingPL` / `TradingPL` / `TotalPL` | `float64` | Lãi/lỗ |
| `Withdrawable` | `float64` | Rút được |
| `CashSSI` / `CashVSDC` | `float64` | Tiền tại SSI / VSDC |
| `ValidNonCashSSI` / `ValidNonCashVSDC` | `float64` | Tài sản phi tiền mặt |
| `WithdrawableSSI` / `WithdrawableVSDC` | `float64` | Tiền rút được tại SSI / VSDC (hàm `CashWithdrawableSSI()`/`CashWithdrawableVSDC()` cũ vẫn còn, deprecated) |

**`portfolio.EquityPosition`** — Vị thế cơ sở

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` | `string` | Số tài khoản |
| `Symbol` | `string` | Mã chứng khoán |
| `Quantity` | `int` | Tổng số lượng |
| `BlockQuantity` | `int` | SL phong toả |
| `DividendQuantity` | `int` | SL cổ tức |
| `BuyingQuantity` / `BoughtQuantity` | `int` | SL đang mua / đã mua |
| `SellingQuantity` / `SoldQuantity` | `int` | SL đang bán / đã bán |
| `T1SellQuantity` / `T2SellQuantity` | `int` | SL bán T+1/2 |
| `T1BuyQuantity` / `T2BuyQuantity` | `int` | SL mua T+1/2 |
| `CostPrice` | `float64` | Giá vốn |
| `MortgageQuantity` | `int` | SL cầm cố |
| `SellableQuantity` | `int` | Số lượng bán được |
| `RestrictedQuantity` | `int` | SL hạn chế |

**`portfolio.DerivativePosition`** — Vị thế phái sinh

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` | `string` | Số tài khoản |
| `Symbol` | `string` | Mã hợp đồng |
| `Long` / `Short` / `Net` | `int` | Vị thế mua / bán / ròng |
| `BidAvgPrice` / `AskAvgPrice` | `float64` | Giá TB mua / bán |
| `TradePrice` | `float64` | Giá giao dịch |
| `FloatingPL` / `TradingPL` | `float64` | Lãi/lỗ |

**`portfolio.Order`** — Thông tin lệnh

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` | `string` | Số tài khoản |
| `ClientRequestID` | `string` | ID yêu cầu client |
| `OrderID` | `string` | ID lệnh |
| `Symbol` | `string` | Mã chứng khoán |
| `Side` | `OrderSide` | Mua/Bán |
| `OrderType` | `OrderType` | Loại lệnh |
| `Price` / `AvgPrice` | `float64` | Giá đặt / Giá TB khớp |
| `PriceText` | `string` | Giá dạng chữ (ATO/ATC/MP...) khi lệnh không có giá số; lúc đó `Price` = 0 |
| `Quantity` | `int` | SL đặt |
| `OSQuantity` | `*int` | SL chờ khớp (`nil`: server không gửi) |
| `FilledQuantity` | `int` | SL đã khớp |
| `CancelQuantity` | `int` | SL đã huỷ |
| `Status` | `OrderStatus` | Trạng thái lệnh |
| `InputTime` / `ModifyTime` | `string` | Thời gian đặt / sửa |
| `Message` | `string` | Thông báo |

#### Trading

**`trading.PlaceOrderResponse`** — Kết quả đặt lệnh

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `OrderID` | `string` | ID lệnh |
| `ClientRequestID` | `string` | ID yêu cầu client |
| `Status` | `OrderStatus` | Trạng thái |

**`trading.ModifyOrderResponse`** — Kết quả sửa lệnh

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `ClientModifyID` | `string` | ID yêu cầu sửa |
| `OrderID` | `string` | ID lệnh |
| `ClientRequestID` | `string` | ID yêu cầu gốc |
| `Status` | `OrderStatus` | Trạng thái |

**`trading.CancelOrderResponse`** — Kết quả huỷ lệnh

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `ClientCancelID` | `string` | ID yêu cầu huỷ |
| `OrderID` | `string` | ID lệnh |
| `ClientRequestID` | `string` | ID yêu cầu gốc |
| `Status` | `OrderStatus` | Trạng thái |

**`trading.MaxBuySellResponse`** — Sức mua/bán tối đa

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` | `string` | Số tài khoản |
| `Symbol` | `string` | Mã chứng khoán |
| `MaxBuyQuantity` | `int` | SL mua tối đa |
| `MaxSellQuantity` | `int` | SL bán tối đa |
| `MarginRatio` | `*string` | Tỷ lệ ký quỹ (`nil` với tài khoản phái sinh) |
| `PurchasingPower` | `*int` | Sức mua (`nil` với tài khoản phái sinh). `PurchasePower()` cũ còn lại, deprecated |

#### Streaming Messages

**`stream.TradeMessage`** — Dữ liệu khớp lệnh

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `DataType` | `DataTypeTrade` |
| `TradingTime` | `string` | Thời gian |
| `Symbol` | `string` | Mã CK |
| `Price` | `float64` | Giá khớp |
| `Quantity` | `int` | KL khớp |
| `Side` | `string` | Bên mua/bán (`"B"`, `"S"`, `"U"`) |
| `TotalVolume` | `int` | Tổng KL |
| `Open` / `High` / `Low` / `Avg` | `*float64` | Giá mở/cao/thấp/TB (`nil` khi server không gửi) |

**`stream.OrderStatusMessage`** — Trạng thái lệnh (`order.<account>`)

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `AccountNo` / `ClientRequestID` / `OrderID` / `Symbol` | `string` | Định danh lệnh |
| `Side` / `OrderType` / `Status` | enum | Giá trị lạ từ server được giữ nguyên dạng chuỗi |
| `Price` / `PriceText` | `float64` / `string` | Giá số / giá chữ (ATO, ATC, MP) |
| `AvgPrice` / `AvgPriceText` | `float64` / `string` | Giá TB khớp số / chữ |
| `Quantity` / `OSQuantity` / `FilledQuantity` / `CancelQuantity` | `int` | Số lượng |
| `InputTime` / `ModifiedTime` | `string` | Thời gian đặt / sửa |
| `RejectReason` / `ErrorCode` / `ErrorMessage` / `Message` | `string` | Thông tin từ chối/lỗi |

**`stream.OrderMatchMessage`** — Lệnh khớp (`orderMatchEvent`): `MatchPrice`, `MatchQty` và định danh lệnh.

**`stream.PortfolioMessage`** — Vị thế phái sinh (`portfolio.<account>`): `Symbol`, `LongQty`, `ShortQty`, `Net`, `PositionType`; `BidAvgPrice`, `AskAvgPrice`, `TradePrice`, `MarketPrice`, `FloatingPL`, `TradingPL` là `*float64`. `TotalAsset`/`CashBalance`/`StockValue` (con trỏ) server không gửi — deprecated.

**`stream.IntervalMessage`** — Dữ liệu OHLCV theo interval

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `DataType` | `DataTypeTrade` |
| `IntervalTime` | `string` | Thời gian interval |
| `TradingTime` | `string` | Thời gian giao dịch |
| `Symbol` | `string` | Mã CK |
| `Open` / `High` / `Low` / `Close` | `float64` | Giá OHLC |
| `Volume` | `int` | Khối lượng |

**`stream.QuoteMessage`** — Dữ liệu giá bid/ask

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `DataType` | `DataTypeQuote` |
| `TradingTime` | `string` | Thời gian |
| `Symbol` | `string` | Mã CK |
| `BidPrices` / `BidVolumes` | `[]float64` / `[]int` | Giá/KL bid |
| `AskPrices` / `AskVolumes` | `[]float64` / `[]int` | Giá/KL ask |

**`stream.ForeignRoomMessage`** — Room nước ngoài

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `DataType` | `DataTypeRoom` |
| `TradingTime` | `string` | Thời gian |
| `Symbol` | `string` | Mã CK |
| `TotalRoom` / `CurrentRoom` | `int` | Tổng room / Room còn lại |
| `BuyQuantity` / `BuyValue` | `int` | KL/Giá trị mua NN |
| `SellQuantity` / `SellValue` | `int` | KL/Giá trị bán NN |

**`stream.PutMessage`** — Thoả thuận

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `DataType` | `DataTypePut` |
| `TradingTime` | `string` | Thời gian |
| `Symbol` | `string` | Mã CK |
| `Price` | `float64` | Giá |
| `Quantity` | `int` | Khối lượng |
| `TotalQuantity` / `TotalValue` | `int` | Tổng KL / Tổng giá trị |

**`stream.OddLotMessage`** — Lô lẻ

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `DataType` | `DataTypeOddLot` |
| `TradingTime` | `string` | Thời gian |
| `Symbol` | `string` | Mã CK |
| `Price` | `float64` | Giá |
| `Quantity` | `int` | Khối lượng |
| `BidPrices` / `BidVolumes` | `[]float64` / `[]int` | Giá/KL bid |
| `AskPrices` / `AskVolumes` | `[]float64` / `[]int` | Giá/KL ask |

**`stream.MarketStatusMessage`** — Trạng thái thị trường

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Market` | `string` | Thị trường |
| `Status` | `string` | Trạng thái |
| `TradingDate` | `string` | Ngày giao dịch |

**`stream.OrderStatusMessage`** — Trạng thái lệnh (streaming)

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `StreamingType` | `StreamingTypeOrder` |
| `AccountNo` | `string` | Số tài khoản |
| `ClientRequestID` / `OrderID` | `string` | ID yêu cầu / ID lệnh |
| `Symbol` | `string` | Mã CK |
| `Side` | `OrderSide` | Mua/Bán |
| `OrderType` | `OrderType` | Loại lệnh |
| `Price` | `float64` | Giá |
| `Quantity` / `OSQuantity` / `FilledQuantity` / `CancelQuantity` | `int` | SL đặt / chờ / khớp / huỷ |
| `Status` | `OrderStatus` | Trạng thái |
| `InputTime` / `ModifyTime` | `string` | Thời gian đặt / sửa |
| `Message` | `string` | Thông báo (Lý do từ chối) |

**`stream.PortfolioMessage`** — Thay đổi danh mục (streaming)

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Type` | `StreamingType` | `StreamingTypePortfolio` |
| `AccountNo` | `string` | Số tài khoản |
| `TotalAsset` | `float64` | Tổng tài sản |
| `CashBalance` | `float64` | Số dư tiền |
| `StockValue` | `float64` | Giá trị chứng khoán |

**`stream.HeartbeatMessage`** — Heartbeat

| Trường | Kiểu | Mô tả |
|--------|------|-------|
| `Method` | `StreamingMethod` | Method |
| `Channel` | `StreamingChannel` | Channel |
| `Status` | `string` | Trạng thái |
| `Message` | `string` | Thông báo |

---

### Clients & Services

| Client | Service / Field | Truy cập | Mô tả |
|--------|---------|---------|-------|
| `Auth` | `TokenManager` | `auth.TokenManager` | Xác thực, OTP, refresh token |
| `Data` | `MarketData` | `data.MarketData` | OHLC, chỉ số, chứng khoán, securities summary |
| `Trading` | `Trading` | `trading.Trading` | Đặt/sửa/huỷ lệnh, sức mua/bán |
| `Trading` | `Account` | `trading.Account` | Thông tin tài khoản |
| `Trading` | `Portfolio` | `trading.Portfolio` | Số dư, vị thế, sổ lệnh, PPMMR |
| `Stream` | `Streaming` | `stream.Streaming` | Streaming realtime qua WebSocket |
| `experimental` | `Service` | `experimental.NewService(client, cfg)` | EXPERIMENTAL: lệnh có chữ ký qua Trading WebSocket, API có thể đổi |

Xem thêm: [CHANGELOG.md](CHANGELOG.md) (thay đổi và breaking changes của 3.3.0), [API_METHODS.md](API_METHODS.md) (danh sách method), [AGENT.md](AGENT.md) (hướng dẫn cho AI agent).
