package trading

type OrderSide string

const (
	OrderSideBuy  OrderSide = "B"
	OrderSideSell OrderSide = "S"
)

type OrderType string

const (
	OrderTypeATO OrderType = "ATO"
	OrderTypeATC OrderType = "ATC"
	OrderTypeLO  OrderType = "LO"
	OrderTypeMTL OrderType = "MTL"
	OrderTypeMP  OrderType = "MP"
	OrderTypeMOK OrderType = "MOK"
	OrderTypeMAK OrderType = "MAK"
	OrderTypePLO OrderType = "PLO"
)

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
)
