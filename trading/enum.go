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

type FCOOperator string

const (
	FCOOperatorGreater        FCOOperator = "greater"
	FCOOperatorGreaterOrEqual FCOOperator = "greater_or_equal"
	FCOOperatorLesser         FCOOperator = "lesser"
	FCOOperatorLesserOrEqual  FCOOperator = "lesser_or_equal"
	FCOOperatorEqual          FCOOperator = "equal"
)


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

