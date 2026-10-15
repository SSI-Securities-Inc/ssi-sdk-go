package ssisdk

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

// RequireNonEmpty fails when value is blank (empty or whitespace only).
func RequireNonEmpty(value, fieldName string) error {
	if strings.TrimSpace(value) == "" {
		return NewValidationError(fmt.Sprintf("%s is required and cannot be empty", fieldName))
	}
	return nil
}

// RequireEmpty fails when value is not empty.
func RequireEmpty(value, fieldName string) error {
	if value != "" {
		return NewValidationError(fmt.Sprintf("%s must be empty, got '%s'", fieldName, value))
	}
	return nil
}

func RequirePositive(value float64, fieldName string) error {
	if !(value > 0) {
		return NewValidationError(fmt.Sprintf("%s must be positive, got %v", fieldName, value))
	}
	return nil
}

func RequireNonNegative(value float64, fieldName string) error {
	if !(value >= 0) {
		return NewValidationError(fmt.Sprintf("%s must be non-negative, got %v", fieldName, value))
	}
	return nil
}

// RequireIn fails when value is not one of allowed.
func RequireIn(value string, allowed []string, fieldName string) error {
	for _, a := range allowed {
		if a == value {
			return nil
		}
	}
	sorted := append([]string(nil), allowed...)
	sort.Strings(sorted)
	return NewValidationError(fmt.Sprintf("%s must be one of %v, got '%s'", fieldName, sorted, value))
}

var symbolRe = regexp.MustCompile(`^[A-Za-z0-9]+$`)

// RequireSymbol validates a single alphanumeric ticker symbol (no lists, no separators).
func RequireSymbol(value, fieldName string) error {
	if err := RequireNonEmpty(value, fieldName); err != nil {
		return err
	}
	if !symbolRe.MatchString(value) {
		return NewValidationError(fmt.Sprintf("%s must be a single alphanumeric symbol, got '%s'", fieldName, value))
	}
	return nil
}

const (
	DateLayout     = "2006/01/02"
	DateTimeLayout = "2006/01/02 15:04:05"
)

// ParseDateArg parses a YYYY/MM/DD (optionally YYYY/MM/DD HH:MM:SS) request date.
// allowTime also accepts the datetime layout; timeRequired accepts only it (FCO dates).
func ParseDateArg(value, fieldName string, allowTime, timeRequired bool) (time.Time, error) {
	if err := RequireNonEmpty(value, fieldName); err != nil {
		return time.Time{}, err
	}
	layouts := []string{DateLayout}
	expected := "YYYY/MM/DD"
	switch {
	case timeRequired:
		layouts = []string{DateTimeLayout}
		expected = "YYYY/MM/DD HH:MM:SS"
	case allowTime:
		layouts = []string{DateLayout, DateTimeLayout}
		expected = "YYYY/MM/DD HH:MM:SS or YYYY/MM/DD"
	}
	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, value, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, NewValidationError(fmt.Sprintf("%s must be formatted %s, got '%s'", fieldName, expected, value))
}

// RequireDateRange validates a from/to pair: both well formed, from <= to, optional span cap
// (maxDays <= 0 means no cap).
func RequireDateRange(from, to string, allowTime, timeRequired bool, maxDays int) error {
	start, err := ParseDateArg(from, "from", allowTime, timeRequired)
	if err != nil {
		return err
	}
	end, err := ParseDateArg(to, "to", allowTime, timeRequired)
	if err != nil {
		return err
	}
	if start.After(end) {
		return NewValidationError(fmt.Sprintf("from must not be after to (%s > %s)", from, to))
	}
	if maxDays > 0 && end.Sub(start) > time.Duration(maxDays)*24*time.Hour {
		return NewValidationError(fmt.Sprintf("date range must not exceed %d days", maxDays))
	}
	return nil
}

// RequireExactlyOne validates that exactly one of two optional arguments was given.
func RequireExactlyOne(first, second, firstName, secondName string) error {
	if (first != "") == (second != "") {
		return NewValidationError(fmt.Sprintf("exactly one of %s or %s is required", firstName, secondName))
	}
	return nil
}
