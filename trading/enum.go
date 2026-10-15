package trading

import (
	"strconv"
	"strings"
)

// OrderSide is the side of an order. An unknown value from the server is kept as its raw string.
type OrderSide string

const (
	OrderSideBuy  OrderSide = "B"
	OrderSideSell OrderSide = "S"
)

// OrderType is the order type. price is required for LO/PLO and ignored by market-priced types.
type OrderType string

const (
	// OrderTypeATO is matched in the opening auction.
	OrderTypeATO OrderType = "ATO"
	// OrderTypeATC is matched in the closing auction.
	OrderTypeATC OrderType = "ATC"
	// OrderTypeLO is a limit order at the given price.
	OrderTypeLO OrderType = "LO"
	// OrderTypeMTL is market-to-limit (HOSE): the unfilled part becomes a limit order.
	OrderTypeMTL OrderType = "MTL"
	// OrderTypeMP is market price (HNX/UPCOM).
	OrderTypeMP OrderType = "MP"
	// OrderTypeMOK is match-or-kill (HNX).
	OrderTypeMOK OrderType = "MOK"
	// OrderTypeMAK is match-and-kill (HNX).
	OrderTypeMAK OrderType = "MAK"
	// OrderTypePLO is a post-close limit order.
	OrderTypePLO OrderType = "PLO"
)

// pricedOrderType reports whether the type carries a real price (every other type is priced by
// the market).
func pricedOrderType(t OrderType) bool { return t == OrderTypeLO || t == OrderTypePLO }

// OrderStatus is the order status as the servers report it. The server docs do not define every
// code, so unknown values are always kept as the raw string.
type OrderStatus string

const (
	OrderStatusPending          OrderStatus = "PD"
	OrderStatusPendingApproval  OrderStatus = "WA"
	OrderStatusReady            OrderStatus = "RS"
	OrderStatusSent             OrderStatus = "SD"
	OrderStatusQueued           OrderStatus = "QU"
	OrderStatusFilled           OrderStatus = "FF"
	OrderStatusPartialFilled    OrderStatus = "PF"
	OrderStatusPartialCancelled OrderStatus = "FFPC"
	OrderStatusPendingModify    OrderStatus = "WM"
	OrderStatusPendingCancel    OrderStatus = "WC"
	OrderStatusCancelled        OrderStatus = "CL"
	OrderStatusRejected         OrderStatus = "RJ"
	OrderStatusExpired          OrderStatus = "EX"
	OrderStatusPreSession       OrderStatus = "IAV"
	// OrderStatusSOI, OrderStatusPAS and OrderStatusCPL are reported by the order engines and
	// named after their codes.
	OrderStatusSOI OrderStatus = "SOI"
	OrderStatusPAS OrderStatus = "PAS"
	OrderStatusCPL OrderStatus = "CPL"
	// OrderStatusRequested is "request received" (FCO engine).
	OrderStatusRequested OrderStatus = "RQ"
	// OrderStatusError is an error reported by the engine.
	OrderStatusError OrderStatus = "ERR"
)

// FCOType is the flexible conditional order type.
type FCOType string

const (
	FCOTypeGTD               FCOType = "gtd"
	FCOTypeStop              FCOType = "stop"
	FCOTypeStopLimit         FCOType = "stop_limit"
	FCOTypeTrailingStop      FCOType = "trailing_stop"
	FCOTypeTrailingStopLimit FCOType = "trailing_stop_limit"
	FCOTypeOCO               FCOType = "oco"
	FCOTypeBullBear          FCOType = "bullbear"
)

// FCOOperator is the comparison that fires an FCO trigger. Requests send the name; responses
// may give the number 0..4 (see ParseFCOOperator).
type FCOOperator string

const (
	// FCOOperatorGreater is price > trigger (response number 2).
	FCOOperatorGreater FCOOperator = "greater"
	// FCOOperatorGreaterOrEqual is price >= trigger (0).
	FCOOperatorGreaterOrEqual FCOOperator = "greater_or_equal"
	// FCOOperatorLesser is price < trigger (3).
	FCOOperatorLesser FCOOperator = "lesser"
	// FCOOperatorLesserOrEqual is price <= trigger (4).
	FCOOperatorLesserOrEqual FCOOperator = "lesser_or_equal"
	// FCOOperatorEqual is price == trigger (1).
	FCOOperatorEqual FCOOperator = "equal"
)

// FCOStatus is the processing status of an FCO. Keep unknown values as raw strings.
type FCOStatus string

const (
	FCOStatusInit FCOStatus = "INIT"
	FCOStatusWait FCOStatus = "WAIT"
	FCOStatusTri  FCOStatus = "TRI"
	FCOStatusTrit FCOStatus = "TRIT"
	FCOStatusTer  FCOStatus = "TER"
	FCOStatusFis  FCOStatus = "FIS"
	FCOStatusWc   FCOStatus = "WC"
	FCOStatusExp  FCOStatus = "EXP"
	FCOStatusErr  FCOStatus = "ERR"
)

var (
	orderSides = []string{string(OrderSideBuy), string(OrderSideSell)}
	orderTypes = []string{"ATO", "ATC", "LO", "MTL", "MP", "MOK", "MAK", "PLO"}
	fcoTypes   = []string{"gtd", "stop", "stop_limit", "trailing_stop", "trailing_stop_limit", "oco", "bullbear"}
	fcoOps     = []string{"greater", "greater_or_equal", "lesser", "lesser_or_equal", "equal"}
	fcoStatus  = []string{"INIT", "WAIT", "TRI", "TRIT", "TER", "FIS", "WC", "EXP", "ERR"}
)

// canonical returns the member of known matching raw case-insensitively.
func canonical(raw string, known []string) (string, bool) {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	for _, k := range known {
		if strings.ToUpper(k) == upper {
			return k, true
		}
	}
	return "", false
}

// OrderSideFromValue resolves a side case-insensitively ("b", "B"); ok is false when unknown.
func OrderSideFromValue(raw string) (OrderSide, bool) {
	v, ok := canonical(raw, orderSides)
	return OrderSide(v), ok
}

// OrderTypeFromValue resolves an order type case-insensitively.
func OrderTypeFromValue(raw string) (OrderType, bool) {
	v, ok := canonical(raw, orderTypes)
	return OrderType(v), ok
}

// FCOTypeFromValue resolves an FCO type case-insensitively.
func FCOTypeFromValue(raw string) (FCOType, bool) {
	v, ok := canonical(raw, fcoTypes)
	return FCOType(v), ok
}

// FCOOperatorFromValue resolves an operator name case-insensitively.
func FCOOperatorFromValue(raw string) (FCOOperator, bool) {
	v, ok := canonical(raw, fcoOps)
	return FCOOperator(v), ok
}

// FCOStatusFromValue resolves an FCO status case-insensitively.
func FCOStatusFromValue(raw string) (FCOStatus, bool) {
	v, ok := canonical(raw, fcoStatus)
	return FCOStatus(v), ok
}

var operatorByNumber = map[int]FCOOperator{
	0: FCOOperatorGreaterOrEqual,
	1: FCOOperatorEqual,
	2: FCOOperatorGreater,
	3: FCOOperatorLesser,
	4: FCOOperatorLesserOrEqual,
}

// ParseFCOOperator reads an operator given as a name or as the server's number 0..4 (0
// greater-or-equal, 1 equal, 2 greater, 3 lesser, 4 lesser-or-equal). An unknown value is kept
// raw; an empty one gives "".
func ParseFCOOperator(raw string) FCOOperator {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if n, err := strconv.Atoi(raw); err == nil {
		if op, ok := operatorByNumber[n]; ok {
			return op
		}
	}
	if op, ok := FCOOperatorFromValue(raw); ok {
		return op
	}
	return FCOOperator(raw)
}

var orderStatuses = []string{
	"PD", "WA", "RS", "SD", "QU", "FF", "PF", "FFPC", "WM", "WC", "CL", "RJ", "EX", "IAV",
	"SOI", "PAS", "CPL", "RQ", "ERR",
}

// OrderStatusFromValue resolves a status case-insensitively; ok is false when unknown.
func OrderStatusFromValue(raw string) (OrderStatus, bool) {
	v, ok := canonical(raw, orderStatuses)
	return OrderStatus(v), ok
}

// normOrderStatus canonicalizes a known status and keeps an unknown one as its raw string.
func normOrderStatus(raw string) OrderStatus {
	if s, ok := OrderStatusFromValue(raw); ok {
		return s
	}
	return OrderStatus(raw)
}

func normOrderSide(raw string) OrderSide {
	if s, ok := OrderSideFromValue(raw); ok {
		return s
	}
	return OrderSide(raw)
}

func normOrderType(raw string) OrderType {
	if s, ok := OrderTypeFromValue(raw); ok {
		return s
	}
	return OrderType(raw)
}

func normFCOType(raw string) FCOType {
	if s, ok := FCOTypeFromValue(raw); ok {
		return s
	}
	return FCOType(raw)
}

func normFCOStatus(raw string) FCOStatus {
	if s, ok := FCOStatusFromValue(raw); ok {
		return s
	}
	return FCOStatus(raw)
}
