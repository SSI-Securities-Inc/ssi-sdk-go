package transport

import (
	"encoding/json"
	"fmt"
	"strings"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/internal/logger"
)

var dispatchLog = logger.New("ssi_sdk.transport.dispatch")

// DecodeMessage decodes a raw frame into a map. It returns nil when the frame is not a JSON
// object. The frame content is never logged (it can carry account numbers); only its size is.
func DecodeMessage(raw []byte) map[string]interface{} {
	var message map[string]interface{}
	if err := json.Unmarshal(raw, &message); err != nil || message == nil {
		dispatchLog.Debug("Received a non-object message (%d bytes)", len(raw))
		return nil
	}
	return message
}

// IsAck reports whether a frame is a reply to a request (has method and status, no topic).
func IsAck(message map[string]interface{}) bool {
	_, hasMethod := message["method"]
	_, hasStatus := message["status"]
	_, hasTopic := message["topic"]
	return hasMethod && hasStatus && !hasTopic
}

// RoutesOf returns every route whose handlers should see this frame, most specific first:
//
//   - a frame with a channel goes to that channel's handlers (DATA/TRADING/HEARTBEAT), and an
//     ack among them also goes to the ack route;
//   - {"code","msg"} without a channel is the server rejecting the connection;
//   - a frame with an id answers a request sent on the trading socket;
//   - {"trading": "...", "data": "..."} is the reply to LIST_SUBSCRIPTION.
//
// Anything else yields no route and is dropped.
func RoutesOf(message map[string]interface{}) []string {
	var routes []string
	if channel, ok := message["channel"].(string); ok && channel != "" {
		routes = append(routes, channel)
		if IsAck(message) {
			routes = append(routes, ssi.WSRouteAck)
		}
		return routes
	}
	_, hasID := message["id"]
	_, hasCode := message["code"]
	_, hasMsg := message["msg"]
	_, hasMessage := message["message"]
	_, hasTopic := message["topic"]
	_, hasTrading := message["trading"]
	_, hasData := message["data"]
	switch {
	case hasID:
		routes = append(routes, ssi.WSRouteID)
	case hasCode && (hasMsg || hasMessage):
		routes = append(routes, ssi.WSRouteError)
	case !hasTopic && (hasTrading || hasData):
		routes = append(routes, ssi.WSRouteListSubscription)
	case IsAck(message):
		routes = append(routes, ssi.WSRouteAck)
	}
	return routes
}

func hasRoute(routes []string, route string) bool {
	for _, r := range routes {
		if r == route {
			return true
		}
	}
	return false
}

// AckErrorCategory says why the server refused a request, from an {"status": "error"} ack:
// "Denied (account)", "Denied (scope)", "Invalid" or "error". It returns "" when the frame is not
// an error ack. Only the category is returned: the server's text names the topics (and so
// possibly account numbers), which must not reach a log.
func AckErrorCategory(message map[string]interface{}) string {
	if !IsAck(message) || strings.ToLower(fmt.Sprint(message["status"])) != "error" {
		return ""
	}
	text := ""
	if v, ok := message["message"]; ok && v != nil {
		text = strings.ToLower(fmt.Sprint(v))
	}
	for _, category := range []string{"Denied (account)", "Denied (scope)", "Invalid"} {
		if strings.Contains(text, strings.ToLower(category)) {
			return category
		}
	}
	return "error"
}

func topicsOf(value interface{}) ([]string, bool) {
	switch t := value.(type) {
	case []string:
		return t, true
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, v := range t {
			s, ok := v.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	}
	return nil, false
}

func withTopics(data map[string]interface{}, topics []string) map[string]interface{} {
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		out[k] = v
	}
	out["topics"] = topics
	return out
}

func encodedLen(data map[string]interface{}) int {
	b, err := json.Marshal(data)
	if err != nil {
		return 0
	}
	return len(b)
}

// SplitRequest splits a subscribe/unsubscribe request whose JSON would exceed maxBytes. The
// server drops frames over its MaxMessageSize (4096 bytes), so a request naming many topics is
// sent as several requests with the same method/channel and a share of the topics each.
// Anything else (no topics list, or small enough) is returned unchanged. A maxBytes <= 0 means
// ssi.WSMaxFrameBytes.
func SplitRequest(data map[string]interface{}, maxBytes int) []map[string]interface{} {
	if maxBytes <= 0 {
		maxBytes = ssi.WSMaxFrameBytes
	}
	topics, ok := topicsOf(data["topics"])
	if !ok || encodedLen(data) <= maxBytes {
		return []map[string]interface{}{data}
	}
	var parts []map[string]interface{}
	var current []string
	for _, topic := range topics {
		trial := withTopics(data, append(append([]string{}, current...), topic))
		if len(current) > 0 && encodedLen(trial) > maxBytes {
			parts = append(parts, withTopics(data, current))
			current = nil
		}
		current = append(current, topic)
	}
	if len(current) > 0 {
		parts = append(parts, withTopics(data, current))
	}
	return parts
}

// SplitSubscriptions parses a LIST_SUBSCRIPTION reply into {"trading": [...], "data": [...]}.
// The server joins topics with ";". A missing key or an empty string is an empty list.
func SplitSubscriptions(message map[string]interface{}) map[string][]string {
	result := map[string][]string{}
	for _, key := range []string{"trading", "data"} {
		text, _ := message[key].(string)
		list := []string{}
		for _, topic := range strings.Split(text, ";") {
			if topic != "" {
				list = append(list, topic)
			}
		}
		result[key] = list
	}
	return result
}
