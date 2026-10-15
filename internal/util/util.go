package util

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
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

// parseNumber parses a finite number from a number or a numeric string (thousands separators
// accepted). ok is false when v is nil, a bool, blank, not numeric or not finite.
func parseNumber(v interface{}) (f float64, ok bool) {
	switch val := v.(type) {
	case nil, bool:
		return 0, false
	case float64:
		f = val
	case float32:
		f = float64(val)
	case int:
		f = float64(val)
	case int32:
		f = float64(val)
	case int64:
		f = float64(val)
	case uint:
		f = float64(val)
	case uint64:
		f = float64(val)
	case json.Number:
		return parseNumber(val.String())
	case string:
		text := strings.TrimSpace(strings.ReplaceAll(val, ",", ""))
		if text == "" {
			return 0, false
		}
		p, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return 0, false
		}
		f = p
	default:
		return 0, false
	}
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0, false
	}
	return f, true
}

// ToFloat64 converts a value to float64 (0 when missing or invalid). Thousands separators
// ("2,493,089.5") are accepted; inf/nan are rejected.
func ToFloat64(v interface{}) float64 {
	f, _ := parseNumber(v)
	return f
}

// ToOptFloat64 is ToFloat64 returning nil (not 0) when the value is missing or invalid.
func ToOptFloat64(v interface{}) *float64 {
	f, ok := parseNumber(v)
	if !ok {
		return nil
	}
	return &f
}

// ToInt converts a value to int (0 when missing or invalid); fractions truncate toward zero.
func ToInt(v interface{}) int {
	return int(ToInt64(v))
}

// ToInt64 converts a value to int64 (0 when missing or invalid); fractions truncate toward zero.
func ToInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int64:
		return val
	}
	f, ok := parseNumber(v)
	if !ok {
		return 0
	}
	return int64(f)
}

// ToOptInt is ToInt returning nil (not 0) when the value is missing or invalid.
func ToOptInt(v interface{}) *int {
	f, ok := parseNumber(v)
	if !ok {
		return nil
	}
	i := int(f)
	return &i
}

// ToOptInt64 is ToInt64 returning nil (not 0) when the value is missing or invalid.
func ToOptInt64(v interface{}) *int64 {
	f, ok := parseNumber(v)
	if !ok {
		return nil
	}
	i := int64(f)
	return &i
}

// Pick returns the first value among keys that is present and not nil.
func Pick(data map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := data[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

// IsNoContent reports whether a response body is the server's {"code": 204} empty-result marker.
func IsNoContent(body interface{}) bool {
	m, ok := body.(map[string]interface{})
	if !ok {
		return false
	}
	c, present := m["code"]
	return present && ToStr(c) == "204"
}

// FormatPrice serializes a price for a request body without exponent notation
// (1e2 -> "100", 0.1 -> "0.1"). NaN/Inf are errors.
func FormatPrice(v float64) (string, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "", errors.New("price must be a finite number")
	}
	return strconv.FormatFloat(v, 'f', -1, 64), nil
}

var dateLayouts = []string{
	"2006/01/02 15:04:05",
	"2006-01-02 15:04:05",
	"02/01/2006 15:04:05",
	"2006/01/02",
	"2006-01-02",
	"01/02/2006",
	"02/01/2006",
}

// ParseDate parses a server date in any of the known layouts. ok is false for empty or
// unknown layouts. 01/02/2025 reads as MM/DD/YYYY (tried before DD/MM/YYYY).
func ParseDate(v interface{}) (time.Time, bool) {
	switch val := v.(type) {
	case time.Time:
		return val, true
	case string:
		text := strings.TrimSpace(val)
		if text == "" {
			return time.Time{}, false
		}
		for _, l := range dateLayouts {
			if t, err := time.ParseInLocation(l, text, time.Local); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
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

var (
	idMask int
	idStep int
)

func init() {
	alphabetLen := len(idAlphabet)
	idMask = 1
	if alphabetLen > 1 {
		idMask = (2 << int(math.Log(float64(alphabetLen-1))/math.Log(2))) - 1
	}
	idStep = int(math.Ceil(1.6 * float64(idMask) * float64(idSize) / float64(alphabetLen)))
}

func GenerateRequestID() string {
	alphabetLen := len(idAlphabet)
	result := make([]byte, 0, idSize)
	for {
		randomBytes := make([]byte, idStep)
		_, _ = rand.Read(randomBytes)
		for _, b := range randomBytes {
			randomByte := int(b) & idMask
			if randomByte < alphabetLen {
				result = append(result, idAlphabet[randomByte])
				if len(result) == idSize {
					return string(result)
				}
			}
		}
	}
}
