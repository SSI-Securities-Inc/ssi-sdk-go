# SSI FastConnect Go SDK

Go SDK cho nền tảng giao dịch chứng khoán SSI. Hỗ trợ REST API và WebSocket streaming.

**Yêu cầu:** Go 1.22+

## Mục lục

- [Cài đặt](#cài-đặt)
- [Cấu hình](#cấu-hình)
- [Clients Overview](#clients-overview)
- [Auth — Xác thực](#1-auth--xác-thực)
- [Data — Dữ liệu thị trường](#2-data--dữ-liệu-thị-trường)
- [Trading — Giao dịch](#3-trading--giao-dịch)
- [Stream — Streaming realtime](#4-stream--streaming-realtime)
- [Xử lý lỗi](#5-xử-lý-lỗi)
- [Cấu hình nâng cao](#6-cấu-hình-nâng-cao)
- [Package & Enums](#package--enums)

---

## Cài đặt

```bash
go get github.com/SSI-Securities-Inc/ssi-sdk-go/v3
```

---

## Cấu hình

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
| `Timeout` | `int` | `60` | Timeout request (giây) |
| `MaxRetries` | `int` | `5` | Số lần retry tối đa |
| `RetryDelay` | `float64` | `2.0` | Delay cơ sở giữa các lần retry (exponential backoff, giây) |
| `RateLimitPerSecond` | `int` | `10` | Giới hạn request/giây (0 = không giới hạn) |
| `LogLevel` | `string` | `"INFO"` | Mức log: DEBUG, INFO, WARNING, ERROR |

---

## Clients Overview

SDK cung cấp 4 clients chuyên biệt, tương tự Python SDK:

| Client | Mục đích | OTP |
|--------|----------|-----|
| `Auth` | Xác thực + quản lý token. Entry point cho tất cả clients khác. | Không bắt buộc |
| `Data` | Dữ liệu thị trường (OHLC, index, securities). | Không cần OTP |
| `Trading` | Giao dịch + danh mục + tài khoản. | **Bắt buộc** |
| `Stream` | Real-time WebSocket streaming. | **Bắt buộc** |

```
Auth  ──┬──► Data     (market data)
        ├──► Trading  (orders, portfolio, account)
        └──► Stream   (real-time WebSocket)
```

---

## 1. Auth — Xác thực

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
```

### 1.2. Xác thực không cần OTP (chỉ dùng Data)

```go
token, err := auth.Authenticate("") // truyền chuỗi rỗng
```

### 1.3. Yêu cầu gửi OTP

```go
result, err := auth.RequestOTP()
if err != nil {
    log.Fatal(err)
}
log.Println(result)
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

## 2. Data — Dữ liệu thị trường

`Data` truy cập qua `data.MarketData`. Không cần OTP.

```go
auth := ssi.NewAuth(config)
defer auth.Close()

if _, err := auth.Authenticate(""); err != nil {
    log.Fatal(err)
}

data := ssi.NewData(auth)
```

### 2.1. Dữ liệu OHLC (nến)

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
| `GetOHLC1WeekHistorical(symbol, fromDate, toDate, page, size)` | 1 tuần |
| `GetOHLC1MonthHistorical(symbol, fromDate, toDate, page, size)` | 1 tháng |

```go
ohlc, err := data.MarketData.GetOHLC1DayHistorical(
    "SSI", "2026/03/27", "2026/04/22", 1, 100,
)
```

**Tham số historical:** `symbol`, `fromDate`/`toDate` (yyyy/MM/dd), `page` (bắt đầu từ 1), `size` (tối đa 1000).

### 2.2. Danh sách chỉ số thị trường

```go
indexes, err := data.MarketData.GetIndexes()

import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
indexes, err := data.MarketData.GetIndexesByBoard(market.BoardHOSE)
```

### 2.3. Tổng hợp chỉ số (Index Summary)

```go
summary, err := data.MarketData.GetIndexSummary("VNINDEX")
summary, err := data.MarketData.GetIndexSummaryHistorical("VNINDEX", "2026/01/15")
summary, err := data.MarketData.GetBoardSummary(market.BoardHOSE)
summary, err := data.MarketData.GetBoardSummaryHistorical(market.BoardHOSE, "2026/01/15")
```

### 2.4. Thông tin chứng khoán

```go
info, err := data.MarketData.GetSecuritiesInfo("SSI")
securities, err := data.MarketData.GetSecuritiesInfoByIndex("VN30")
securities, err := data.MarketData.GetSecuritiesInfoByBoard(market.BoardHOSE)
```

### 2.5. Tổng hợp chứng khoán (Securities Summary)

```go
summary, err := data.MarketData.GetSecuritiesSummary("SSI")
summary, err := data.MarketData.GetSecuritiesSummaryHistorical("SSI", "2026/03/01", "2026/03/31")
summary, err := data.MarketData.GetSecuritiesSummaryByIndex("VN30")
summary, err := data.MarketData.GetSecuritiesSummaryByIndexHistorical("VN30", "2026/03/01", "2026/03/31")
```

---

## 3. Trading — Giao dịch

`Trading` truy cập qua `t.Trading`, `t.Account`, `t.Portfolio`. Yêu cầu OTP.

```go
auth := ssi.NewAuth(config)
defer auth.Close()

if _, err := auth.Authenticate("222222"); err != nil {
    log.Fatal(err)
}

t := ssi.NewTrading(auth)
```

### 3.1. Tài khoản

```go
accounts, err := t.Account.GetAccountInfo()
for _, acc := range accounts {
    fmt.Printf("%s (%s)\n", acc.AccountNo, acc.AccountType)
}
```

### 3.2. Danh mục đầu tư

```go
// Số dư
balance, err := t.Portfolio.GetEquityBalance("1234561")
derBalance, err := t.Portfolio.GetDerivativeBalance("1234568")

// Vị thế
positions, err := t.Portfolio.GetEquityPositions("1234561")
derPositions, err := t.Portfolio.GetDerivativePositions("1234568")
openPos, err := t.Portfolio.GetOpenDerivativePositions("1234568")
closedPos, err := t.Portfolio.GetClosedDerivativePositions("1234568")

// Sổ lệnh
orders, err := t.Portfolio.GetTodayOrders("1234561")
orders, err := t.Portfolio.GetHistoricalOrders("1234561", "2026/01/01", "2026/01/31")

// PPMMR
ppmmr, err := t.Portfolio.GetEquityPPMMR("1234561")
derPPMMR, err := t.Portfolio.GetDerivativePPMMR("1234568")
```

### 3.3. Đặt lệnh

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"

result, err := t.Trading.PlaceLimitOrder("1234561", "SSI", trading.OrderSideBuy, 100, 66000)
result, err := t.Trading.PlaceMarketOrder("1234561", "SSI", trading.OrderSideBuy, 100)
result, err := t.Trading.PlaceATOOrder("1234561", "SSI", trading.OrderSideBuy, 100)
result, err := t.Trading.PlaceATCOrder("1234561", "SSI", trading.OrderSideSell, 100)
result, err := t.Trading.PlaceOrder("1234561", "SSI", trading.OrderSideBuy, 100, 66000, trading.OrderTypeLO)
```

**Loại lệnh:** `OrderTypeLO`, `OrderTypeMTL`, `OrderTypeMP`, `OrderTypeATO`, `OrderTypeATC`, `OrderTypeMOK`, `OrderTypeMAK`, `OrderTypePLO`.

### 3.4. Sửa lệnh

```go
result, err := t.Trading.ModifyOrderPrice("1234561", "REQ123", 68000)
result, err := t.Trading.ModifyOrderPriceByOrderID("1234561", "ORD123", 68000)
result, err := t.Trading.ModifyOrderQuantity("1234561", "REQ123", 200)
result, err := t.Trading.ModifyOrderQuantityByOrderID("1234561", "ORD123", 200)
```

### 3.5. Huỷ lệnh

```go
result, err := t.Trading.CancelOrder("1234561", "REQ123")
result, err := t.Trading.CancelOrderByOrderID("1234561", "ORD123")
```

### 3.6. Sức mua/bán tối đa

```go
price := 66000.0
maxBS, err := t.Trading.GetMaxBuySell("1234561", "SSI", &price)
maxBS, err := t.Trading.GetMaxBuySellAtMarketPrice("1234561", "SSI")
```

---

## 4. Stream — Streaming realtime

`Stream` truy cập qua `s.Streaming`. Yêu cầu OTP và gọi `Connect()` trước.

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

### 4.1. Thiết lập callback

```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/stream"

s.Streaming.SetOnData(func(msg interface{}) {
    switch m := msg.(type) {
    case *stream.TradeMessage:
        fmt.Printf("[TRADE] %s | Price: %.0f | Qty: %d\n", m.Symbol, m.Price, m.Quantity)
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
    case *stream.PortfolioMessage:
        fmt.Printf("[PORTFOLIO] %s | Total: %.0f\n", m.AccountNo, m.TotalAsset)
    }
})

s.Streaming.SetOnHeartbeat(func(msg *stream.HeartbeatMessage) {
    fmt.Printf("[HEARTBEAT] %v\n", msg)
})
```

**Message types qua `SetOnData`:** `*stream.TradeMessage`, `*stream.QuoteMessage`, `*stream.ForeignRoomMessage`, `*stream.MarketStatusMessage`, `*stream.PutMessage`, `*stream.OddLotMessage`.

**Message types qua `SetOnTrading`:** `*stream.OrderStatusMessage`, `*stream.PortfolioMessage`.

### 4.2. Subscribe

```go
s.Streaming.SubscribeSymbol([]string{"SSI", "HPG", "VIC"}, nil)
s.Streaming.SubscribeSymbolTrade([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolQuote([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolRoom([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolPutThrough([]string{"SSI"}, nil)
s.Streaming.SubscribeSymbolOddLot([]string{"SSI"}, nil)

import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
s.Streaming.SubscribeBoard([]market.Board{market.BoardHOSE, market.BoardHNX}, nil)
s.Streaming.SubscribeIndex([]string{"VNINDEX", "VN30"}, nil)

s.Streaming.SubscribeOrderStatus("", nil)       // tất cả tài khoản
s.Streaming.SubscribeOrderStatus("1234561", nil) // tài khoản cụ thể
s.Streaming.SubscribePortfolio("", nil)
```

### 4.3. Unsubscribe

```go
s.Streaming.UnsubscribeSymbol([]string{"SSI"})
s.Streaming.UnsubscribeSymbolTrade([]string{"SSI"})
s.Streaming.UnsubscribeBoard([]market.Board{market.BoardHOSE})
s.Streaming.UnsubscribeIndex([]string{"VNINDEX"})
```

### 4.4. Heartbeat & Wait

```go
s.Streaming.Ping()

s.Wait(nil)                              // chờ vô thời hạn
timeout := 30 * time.Second
s.Wait(&timeout)                         // chờ 30 giây
```

### 4.5. Ví dụ hoàn chỉnh

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

    timeout := 30 * time.Second
    s.Wait(&timeout)
}
```

---

## 5. Xử lý lỗi

| Error Type | Mô tả |
|------------|-------|
| `*fc.SSIError` | Base error cho tất cả lỗi SDK |
| `*fc.AuthenticationError` | Xác thực thất bại (sai credentials, token hết hạn) |
| `*fc.APIError` | API trả về lỗi — có thêm `StatusCode`, `ResponseBody` |
| `*fc.WebSocketError` | Lỗi kết nối hoặc giao tiếp WebSocket |
| `*fc.ValidationError` | Lỗi validate input |
| `*fc.RateLimitError` | Vượt quá giới hạn request — có thêm `RetryAfter` |

```go
import fc "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"

if err != nil {
    switch e := err.(type) {
    case *fc.AuthenticationError:
        log.Println("Cần xác thực lại")
    case *fc.RateLimitError:
        log.Printf("Rate limited, retry sau %.0fs", *e.RetryAfter)
    case *fc.APIError:
        log.Printf("API error %d: %s", e.StatusCode, e.Message)
    case *fc.ValidationError:
        log.Printf("Validation: %s", e.Message)
    default:
        log.Printf("Unknown error: %v", err)
    }
}
```

---

## 6. Cấu hình nâng cao

```go
config.Proxy              = "http://proxy.example.com:8080"
config.MaxRetries         = 3
config.RetryDelay         = 1.0
config.RateLimitPerSecond = 5
```

---

## Package & Enums

### Cấu trúc package

```
ssi-sdk-go/
├── config.go, errors.go, version.go   # Root package: shared types & errors
├── ssi/                                # ← Import chính: Auth, Data, Trading, Stream
├── auth/                               # Token types
├── account/                            # Account service
├── market/                             # Market data service + enums
├── trading/                            # Trading service + enums
├── portfolio/                          # Portfolio service
├── stream/                             # Streaming service + message types
├── transport/                          # HTTP & WebSocket clients
└── internal/                           # Nội bộ (signature, ratelimit, util)
```

| Package | Import | Mô tả |
|---------|--------|-------|
| `ssi` | `.../ssi-sdk-go/ssi` | **Import chính** — Auth, Data, Trading, Stream |
| `market` | `.../ssi-sdk-go/market` | Enum `Board`, `Timeframe` |
| `trading` | `.../ssi-sdk-go/trading` | Enum `OrderSide`, `OrderType`, `OrderStatus` |
| `stream` | `.../ssi-sdk-go/stream` | Message types cho streaming |
| root | `.../ssi-sdk-go` | Error types (`SSIError`, `APIError`, ...) |

### Enums

```go
import (
    "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
    "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
)

// Order Side
trading.OrderSideBuy   // "B"
trading.OrderSideSell  // "S"

// Order Type
trading.OrderTypeLO    // Limit Order
trading.OrderTypeMTL   // Market To Limit
trading.OrderTypeMP    // Market Price
trading.OrderTypeATO   // At The Open
trading.OrderTypeATC   // At The Close
trading.OrderTypeMOK   // Match Or Kill
trading.OrderTypeMAK   // Match And Kill
trading.OrderTypePLO   // Post Lunch Order

// Order Status
trading.OrderStatusPending | trading.OrderStatusFilled | trading.OrderStatusCancelled | ...

// Board
market.BoardHOSE   // "HOSE"
market.BoardHNX    // "HNX"
market.BoardUPCOM  // "UPCOM"

// Timeframe
market.TimeframeMinute1   // "1m"
market.TimeframeMinute3   // "3m"
market.TimeframeMinute5   // "5m"
market.TimeframeMinute15  // "15m"
market.TimeframeHour1     // "1h"
market.TimeframeDay1      // "1d"
market.TimeframeWeek1     // "1w"
market.TimeframeMonth1    // "1M"
```
