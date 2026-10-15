package market

import "strings"

// Board is an exchange board. An unknown board from the server is kept as its raw string.
type Board string

const (
	BoardHOSE  Board = "HOSE"
	BoardHNX   Board = "HNX"
	BoardUPCOM Board = "UPCOM"
	// BoardDerivatives exists only on the stream and master data, not on the REST data endpoints.
	BoardDerivatives Board = "DERIVATIVES"
)

var knownBoards = []Board{BoardHOSE, BoardHNX, BoardUPCOM, BoardDerivatives}

// restBoards are the boards the REST data endpoints serve.
var restBoards = []Board{BoardHOSE, BoardHNX, BoardUPCOM}

// BoardFromValue maps a value (case-insensitive) to a known Board. ok is false for an empty or
// unknown value.
func BoardFromValue(raw string) (Board, bool) {
	upper := strings.ToUpper(strings.TrimSpace(raw))
	for _, b := range knownBoards {
		if string(b) == upper {
			return b, true
		}
	}
	return "", false
}

type Timeframe string

const (
	TimeframeMinute1  Timeframe = "1m"
	TimeframeMinute3  Timeframe = "3m"
	TimeframeMinute5  Timeframe = "5m"
	TimeframeMinute15 Timeframe = "15m"
	TimeframeHour1    Timeframe = "1h"
	TimeframeDay1     Timeframe = "1d"
	// TimeframeWeek1 is not supported by the server (error 400210).
	TimeframeWeek1 Timeframe = "1w"
	// TimeframeMonth1 is not supported by the server (error 400210).
	TimeframeMonth1 Timeframe = "1M"
)

// AllowedTimeframes are the timeframes the OHLC endpoint accepts. 1w/1M exist only for
// backward compatibility; the server rejects them.
var AllowedTimeframes = map[Timeframe]bool{
	TimeframeDay1:     true,
	TimeframeMinute1:  true,
	TimeframeMinute3:  true,
	TimeframeMinute5:  true,
	TimeframeMinute15: true,
	TimeframeHour1:    true,
}
