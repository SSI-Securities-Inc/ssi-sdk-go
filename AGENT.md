# AGENT.md — AI Agent Integration Guide for ssi-sdk-go

This guide provides AI coding assistants (Claude, Gemini, Cursor, Copilot, etc.) with instructions, code patterns, architectural conventions, and an API cheatsheet for integrating and interacting with the `ssi-sdk-go` package (v3).

---

## 1. Overview & Architecture

`ssi-sdk-go` is a high-performance Go SDK for SSI's **FastConnect v3 API**. It provides strongly-typed Go primitives for:
- Authentication & Token Management (OTP, Refresh Token, Auto-refresh)
- Market Data (OHLC 1min/1day, Indexes, Securities Information & Summary)
- Trading & Portfolio (Order placement/modification/cancellation, FCO conditional orders, Account balances, Positions, PPMMR)
- Realtime Streaming (WebSocket market data & trading events)

### Layering & Structure
```
Facade Client (Auth / Data / Trading / Stream)
   └── Services (market.Service, account.Service, portfolio.Service, trading.Service, stream.Service)
        └── Transport (transport.RestClient / transport.WebSocketClient)
             └── Leaf Modules (trading, portfolio, market, stream, internal/util)
```

### Key Architectural Constraints for AI Agents
1. **Modular Facade Pattern**: Sub-clients are created via `ssi.NewData(auth)`, `ssi.NewTrading(auth)`, `ssi.NewStream(auth)`.
2. **Type Safety with Enums**: All protocol constants (`OrderSide`, `OrderType`, `OrderStatus`, `Board`, `FCOType`, `FCOOperator`, `FCOStatus`, `Timeframe`) are defined as Go types in package `trading` and `market`.
3. **Header User-Agent Parity**: Requests to SSI API include a browser `User-Agent` header by default (`DefaultUserAgent`) to bypass Cloudflare firewall checks.

---

## 2. Authentication & Setup Pattern

### Configuration Initialization
```go
import "github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"

config := ssi.NewConfig("YOUR_CLIENT_ID")
config.APIKey = "YOUR_API_KEY"
config.APISecret = "YOUR_API_SECRET"
config.PrivateKey = "YOUR_PRIVATE_KEY" // Base64 RSA Private Key
```

### Authentication Flow
```go
auth := ssi.NewAuth(config)
defer auth.Close()

// Authenticate with OTP (Required for Trading & Streaming)
token, err := auth.Authenticate("123456")
if err != nil {
    log.Fatalf("Authentication failed: %v", err)
}

// Create sub-clients sharing the auth context
dataClient := ssi.NewData(auth)
tradingClient := ssi.NewTrading(auth)
streamClient := ssi.NewStream(auth)
```

---

## 3. Public API Cheatsheet for AI Agents

### 3.1 Market Data (`dataClient.Market`)

| Method | Parameters | Return Type | Description |
|--------|------------|-------------|-------------|
| `GetOHLC1Minute` | `symbol` | `([]OHLCData, error)` | Query 1-minute candle OHLCV data |
| `GetOHLC1Day` | `symbol` | `([]OHLCData, error)` | Query 1-day candle OHLCV data |
| `DownloadOHLC1Minute` | `symbol` | `([]OHLCData, error)` | Download full 1-minute OHLC history |
| `DownloadOHLC1Day` | `symbol` | `([]OHLCData, error)` | Download full 1-day OHLC history |
| `GetMarketIndexes` | `indexID` | `([]MarketIndex, error)` | Get list of market indexes |
| `GetMarketIndexSummary` | `indexID, fromDate, toDate, page, size` | `(*MarketIndexSummary, error)` | Summary metrics for an index |
| `GetSecuritiesInfo` | `symbol, market, page, size` | `(*SecuritiesInfo, error)` | Security details |
| `GetSecuritiesSummary` | `symbol, market, page, size` | `(*SecuritiesSummary, error)` | Summary of stock transactions |

### 3.2 Account & Portfolio (`tradingClient.Account` & `tradingClient.Portfolio`)

| Service | Method | Return Type | Description |
|---------|--------|-------------|-------------|
| `Account` | `GetAccountInfo()` | `([]AccountInfo, error)` | Query list of sub-accounts |
| `Portfolio` | `GetEquityBalance(accountNo)` | `(*EquityAccountBalance, error)` | Cash balance & debt for cash/margin account |
| `Portfolio` | `GetDerivativeBalance(accountNo)` | `(*DerivativeAccountBalance, error)` | Balance & margin for derivative account |
| `Portfolio` | `GetEquityPositions(accountNo)` | `([]EquityPosition, error)` | Equity stock holdings |
| `Portfolio` | `GetDerivativePositions(accountNo)` | `(*AllDerivativePosition, error)` | Derivative contract positions |
| `Portfolio` | `GetTodayOrders(accountNo)` | `([]Order, error)` | Intraday orders |
| `Portfolio` | `GetHistoricalOrders(accountNo, fromDate, toDate)` | `([]Order, error)` | Historical order book entries |
| `Portfolio` | `GetEquityPpmmr(accountNo)` | `(*EquityPPMMR, error)` | Purchasing power & margin ratio (Equity) |
| `Portfolio` | `GetDerivativePpmmr(accountNo)` | `(*DerivativePPMMR, error)` | Purchasing power & margin ratio (Derivative) |

### 3.3 Standard Trading (`tradingClient.Trading`)

| Method | Parameters | Return Type | Description |
|--------|------------|-------------|-------------|
| `PlaceLimitOrder` | `accountNo, symbol, side, quantity, price` | `(*PlaceOrderResponse, error)` | Place LO order |
| `PlaceMarketOrder` | `accountNo, symbol, side, quantity` | `(*PlaceOrderResponse, error)` | Place MTL market order |
| `PlaceATOOrder` | `accountNo, symbol, side, quantity` | `(*PlaceOrderResponse, error)` | Place ATO opening order |
| `PlaceATCOrder` | `accountNo, symbol, side, quantity` | `(*PlaceOrderResponse, error)` | Place ATC closing order |
| `PlaceOrder` | `accountNo, symbol, side, quantity, price, orderType` | `(*PlaceOrderResponse, error)` | Place order with custom OrderType |
| `ModifyOrderPriceByOrderID` | `accountNo, orderID, price` | `(*ModifyOrderResponse, error)` | Modify order price by server order ID |
| `ModifyOrderQuantityByOrderID` | `accountNo, orderID, quantity` | `(*ModifyOrderResponse, error)` | Modify order quantity by server order ID |
| `CancelOrderByOrderID` | `accountNo, orderID` | `(*CancelOrderResponse, error)` | Cancel order by server order ID |
| `GetMaxBuySell` | `accountNo, symbol, price` | `(*MaxBuySellResponse, error)` | Max buy/sell qty at given price |

### 3.4 Flexible Conditional Orders - FCO (`tradingClient.Trading`)

| Method | Key Parameters | Return Type | Description |
|--------|----------------|-------------|-------------|
| `PlaceFcoGtd` | `accountNo, symbol, side, quantity, price, priceSlip, fromDate, toDate` | `(*FCOPlaceResponse, error)` | Good Till Date order |
| `PlaceFcoStop` | `accountNo, symbol, side, quantity, stopPrice, operator, fromDate, toDate` | `(*FCOPlaceResponse, error)` | Stop Market order |
| `PlaceFcoStopLimit` | `accountNo, symbol, side, quantity, price, priceSlip, stopPrice, operator, fromDate, toDate` | `(*FCOPlaceResponse, error)` | Stop Limit order |
| `PlaceFcoTrailingStop` | `accountNo, symbol, side, quantity, activePrice, trailingAmount, fromDate, toDate` | `(*FCOPlaceResponse, error)` | Trailing Stop Market |
| `PlaceFcoTrailingStopLimit` | `accountNo, symbol, side, quantity, activePrice, trailingAmount, priceSlip, fromDate, toDate` | `(*FCOPlaceResponse, error)` | Trailing Stop Limit |
| `PlaceFcoOco` | `accountNo, symbol, side, quantity, tpActivePrice, slActivePrice, tpPrice, slPrice, tpSlip, slSlip, fromDate, toDate` | `(*FCOPlaceResponse, error)` | One-Cancels-the-Other |
| `PlaceFcoBullBear` | `accountNo, symbol, side, quantity, price, priceSlip, tpActivePrice, slActivePrice, tpPrice, slPrice, tpSlip, slSlip, fromDate, toDate` | `(*FCOPlaceResponse, error)` | Bull Bear order |
| `CancelFco` | `fcoID` | `(*FCOCancelResponse, error)` | Cancel FCO by ID |
| `GetFcoByAccountNo` | `accountNo, pageIndex, pageSize` | `(*FCOListResponse, error)` | List account's FCO orders |
| `GetFcoBySymbol` | `accountNo, symbol, pageIndex, pageSize` | `(*FCOListResponse, error)` | Filter FCO orders by symbol |
| `GetFcoByStatus` | `accountNo, status, pageIndex, pageSize` | `(*FCOListResponse, error)` | Filter FCO by status (`TRIT`, `WAIT`, etc.) |
| `GetFcoByDate` | `accountNo, fromDate, toDate, pageIndex, pageSize` | `(*FCOListResponse, error)` | Filter FCO by date range |
| `GetFcoById` | `accountNo, fcoID` | `(*FCOInfo, error)` | Single FCO order details |
| `GetFcoOrderBook` | `fcoID, pageIndex, pageSize` | `(*FCOOrderBookResponse, error)` | Execution logs of FCO |

---

## 4. Code Examples for Common AI Tasks

### Example 1: Placing & Instantly Cancelling a GTD FCO Order
```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/util"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/trading"
)

func main() {
	config := ssi.NewConfig("YOUR_CLIENT_ID")
	config.APIKey = "YOUR_API_KEY"
	config.APISecret = "YOUR_API_SECRET"
	config.PrivateKey = "YOUR_PRIVATE_KEY"

	auth := ssi.NewAuth(config)
	defer auth.Close()

	if _, err := auth.Authenticate("123456"); err != nil {
		log.Fatalf("Auth error: %v", err)
	}

	tradingClient := ssi.NewTrading(auth)
	accountNo := "1234561"
	fromDate := util.BeginningOfDay()
	toDate := time.Now().AddDate(0, 0, 7).Format("2006/01/02") + " 23:00:00"

	// 1. Place GTD order
	gtdRes, err := tradingClient.Trading.PlaceFcoGtd(
		accountNo, "SSI", trading.OrderSideBuy, 100, trading.OrderTypeMTL, 0.5, fromDate, toDate,
	)
	if err != nil {
		log.Fatalf("Place FCO failed: %v", err)
	}
	fmt.Printf("Placed FCO ID: %s\n", gtdRes.FCOID)

	// 2. Cancel FCO order
	cancelRes, err := tradingClient.Trading.CancelFco(gtdRes.FCOID)
	if err != nil {
		log.Fatalf("Cancel FCO failed: %v", err)
	}
	fmt.Printf("Cancelled FCO ID: %s\n", cancelRes.FCOID)
}
```

### Example 2: Realtime Streaming Callbacks
```go
package main

import (
	"fmt"

	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/ssi"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/stream"
)

func main() {
	auth := ssi.NewAuth(config)
	defer auth.Close()
	auth.Authenticate("123456")

	streamClient := ssi.NewStream(auth)
	streamClient.Streaming.Connect()
	defer streamClient.Streaming.Disconnect()

	handler := func(msg interface{}) {
		switch m := msg.(type) {
		case *stream.TradeMessage:
			fmt.Printf("[Trade] %s @ %.2f (qty: %d)\n", m.Symbol, m.Price, m.Quantity)
		case *stream.OrderStatusMessage:
			fmt.Printf("[Order] ID: %s, Status: %s\n", m.OrderID, m.Status)
		case *stream.FCOOrderUpdateMessage:
			fmt.Printf("[FCO Realtime] ID: %s, ProcessStatus: %s\n", m.FCOID, m.ProcessStatus)
		}
	}

	streamClient.Streaming.SubscribeSymbolTrade([]string{"SSI"}, handler)
	streamClient.Streaming.SubscribeOrderStatus("1234561", handler)
	streamClient.Streaming.Wait(nil)
}
```

---

## 5. Rules & Guidelines for AI Code Generators

1. **Date Format for FCO**: FCO date arguments (`fromDate`, `toDate`) **must** be formatted as `"YYYY/MM/DD HH:MM:SS"` (use `util.BeginningOfDay()` / `util.EndOfDay()`).
2. **Never hardcode secrets**: Access tokens, private keys, API secrets, and OTPs should never be committed to git.
3. **Use Enum constants**: Pass Enum constants (`trading.OrderSideBuy`, `trading.OrderTypeLO`, `trading.FCOOperatorGreaterOrEqual`) instead of raw strings.
4. **Clean Builds**: Maintain Go strict typing and run `go build ./...` to verify changes.
