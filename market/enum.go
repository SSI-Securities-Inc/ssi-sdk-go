package market

type Board string

const (
	BoardHOSE  Board = "HOSE"
	BoardHNX   Board = "HNX"
	BoardUPCOM Board = "UPCOM"
)

type Timeframe string

const (
	TimeframeMinute1  Timeframe = "1m"
	TimeframeMinute3  Timeframe = "3m"
	TimeframeMinute5  Timeframe = "5m"
	TimeframeMinute15 Timeframe = "15m"
	TimeframeHour1    Timeframe = "1h"
	TimeframeDay1     Timeframe = "1d"
	TimeframeWeek1    Timeframe = "1w"
	TimeframeMonth1   Timeframe = "1M"
)
