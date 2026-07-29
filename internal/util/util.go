package util

import (
	"crypto/rand"
	"fmt"
	"math"
	"strings"
	"time"
)


func ToStr(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == math.Trunc(val) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", val)
	}
}

func ToFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		var f float64
		_, _ = fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

func ToInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case float32:
		return int(val)
	case int:
		return val
	case int64:
		return int(val)
	case string:
		var i int
		_, _ = fmt.Sscanf(val, "%d", &i)
		return i
	default:
		return 0
	}
}

func ToInt64(v interface{}) int64 {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	case int:
		return int64(val)
	case int64:
		return val
	case string:
		var i int64
		_, _ = fmt.Sscanf(val, "%d", &i)
		return i
	default:
		return 0
	}
}

func ToBool(v interface{}) bool {
	if v == nil {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return strings.EqualFold(val, "true") || val == "1"
	case float64:
		return val != 0
	case int:
		return val != 0
	default:
		return false
	}
}


func TodayDateStr() string {
	return time.Now().Format("2006/01/02")
}

func BeginningOfDay() string {
	return time.Now().Format("2006/01/02") + " 00:00:00"
}

func EndOfDay() string {
	return time.Now().Format("2006/01/02") + " 23:59:59"
}

func FromBeginningOfDay() string {
	return BeginningOfDay()
}

func FromEndOfDay() string {
	return EndOfDay()
}

func ConvertToDatetimeStr(t time.Time) string {
	return t.Format("2006/01/02 15:04:05")
}


const idAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const idSize = 20

func GenerateRequestID() string {
	alphabetLen := len(idAlphabet)
	mask := 1
	if alphabetLen > 1 {
		mask = (2 << int(math.Log(float64(alphabetLen-1))/math.Log(2))) - 1
	}
	step := int(math.Ceil(1.6 * float64(mask) * float64(idSize) / float64(alphabetLen)))

	result := make([]byte, 0, idSize)
	for {
		randomBytes := make([]byte, step)
		_, _ = rand.Read(randomBytes)
		for _, b := range randomBytes {
			randomByte := int(b) & mask
			if randomByte < alphabetLen {
				result = append(result, idAlphabet[randomByte])
				if len(result) == idSize {
					return string(result)
				}
			}
		}
	}
}
