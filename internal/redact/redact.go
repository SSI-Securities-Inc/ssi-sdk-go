// Package redact masks secrets before anything reaches a log or an error.
package redact

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// SensitiveKeys are lower-cased names of fields whose values must never be logged.
var SensitiveKeys = map[string]bool{
	"apisecret": true, "api_secret": true, "apikey": true, "api_key": true,
	"otp": true, "pin": true, "token": true,
	"accesstoken": true, "access_token": true, "refreshtoken": true, "refresh_token": true,
	"authorization": true, "signature": true, "x-signature": true,
	"password": true, "privatekey": true, "private_key": true,
	"transactionid": true, "transaction_id": true,
	"accountno": true, "account_no": true, "account": true,
}

const visibleTail = 4

var bearerRe = regexp.MustCompile(`(?i)\b(bearer)\s+([A-Za-z0-9._~+/=-]+)`)

// Mask masks a secret, keeping at most the last 4 characters (none when it is short).
// Values of 8 characters or fewer reveal nothing.
func Mask(value string) string {
	r := []rune(value)
	if len(r) <= 2*visibleTail {
		return "***"
	}
	return "***" + string(r[len(r)-visibleTail:])
}

// MaskBearer masks the token in any "Bearer <token>" occurrence inside text.
func MaskBearer(text string) string {
	return bearerRe.ReplaceAllStringFunc(text, func(m string) string {
		sub := bearerRe.FindStringSubmatch(m)
		return sub[1] + " " + Mask(sub[2])
	})
}

// Redact returns a deep copy of data (maps, slices, strings) with sensitive values masked.
// Map keys are matched case-insensitively against SensitiveKeys. The input is not mutated.
func Redact(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(v))
		for k, val := range v {
			if SensitiveKeys[strings.ToLower(k)] && val != nil {
				out[k] = Mask(stringify(val))
			} else {
				out[k] = Redact(val)
			}
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(v))
		for k, val := range v {
			if SensitiveKeys[strings.ToLower(k)] {
				out[k] = Mask(val)
			} else {
				out[k] = MaskBearer(val)
			}
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = Redact(item)
		}
		return out
	case []map[string]interface{}:
		out := make([]interface{}, len(v))
		for i, item := range v {
			out[i] = Redact(item)
		}
		return out
	case string:
		return MaskBearer(v)
	default:
		return data
	}
}

// RedactMap is Redact for the common map[string]interface{} case.
func RedactMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	return Redact(data).(map[string]interface{})
}

func stringify(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

// SafeURL returns rawURL without its query string, fragment or userinfo (safe to log).
func SafeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	host := u.Hostname()
	if p := u.Port(); p != "" {
		host += ":" + p
	}
	return (&url.URL{Scheme: u.Scheme, Host: host, Path: u.Path}).String()
}
