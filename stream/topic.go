package stream

import (
	"fmt"
	"regexp"
	"strings"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
	"github.com/SSI-Securities-Inc/ssi-sdk-go/v3/market"
)

// topicTypes are the topic types the server implements. asset.* / margin.* are not live yet, so
// subscribing to them would only ever be silently ignored.
var topicTypes = map[string]bool{
	"trade": true, "quote": true, "room": true, "put": true, "oddlot": true, "market": true,
	"order": true, "portfolio": true, "trade.index": true, "indexsummary": true,
}

var streamIntervals = map[string]bool{"tick": true, "1m": true, "5m": true}

var topicAliases = map[string]string{"UPCOMINDEX": "HNXUPCOMINDEX"}

// StreamIntervalAll is the "@*" interval: the server only fans a basket index (VN30) or an
// exchange (HOSE) out to its symbols with it, so a bare "trade.VN30" never matches a tick.
const StreamIntervalAll = market.Timeframe("*")

var allIntervalTopicTypes = map[string]bool{
	"trade": true, "quote": true, "room": true, "put": true, "oddlot": true,
}

var (
	// One filter per topic: "*", a code, or a range "A-B". The server does not split on commas.
	topicFilterRE   = regexp.MustCompile(`^(?:\*|[A-Za-z0-9]+(?:-[A-Za-z0-9]+)?)$`)
	concreteIndexRE = regexp.MustCompile(`^[A-Za-z0-9]+$`)
)

func intervalText(interval market.Timeframe) string {
	text := string(interval)
	if strings.EqualFold(text, "tick") {
		return "tick"
	}
	return text
}

// BuildTopic builds one topic string, e.g. "trade.VNM@1m" or "trade.index.VN30".
//
// filter is "*", a code ("VNM", "HOSE") or a range ("A-B"); one filter per topic (a comma list is
// rejected, pass a slice to BuildTopics instead). Codes are upper-cased except for order /
// portfolio accounts, and UPCOMINDEX is rewritten to HNXUPCOMINDEX. interval ("tick", "1m", "5m"
// or empty for none) is only valid for trade and trade.index, and for trade.index only with one
// concrete code. StreamIntervalAll ("*") is the basket/exchange fan-out ("trade.VN30@*",
// "quote.HOSE@*"): valid for trade, quote, room, put and oddlot with one concrete index or
// exchange code.
func BuildTopic(topicType, filter string, interval market.Timeframe) (string, error) {
	if !topicTypes[topicType] {
		return "", ssi.NewValidationError(fmt.Sprintf("unknown topic type '%s'", topicType))
	}
	if !topicFilterRE.MatchString(filter) {
		return "", ssi.NewValidationError(fmt.Sprintf("invalid topic filter '%s'", filter))
	}
	if topicType != "order" && topicType != "portfolio" {
		filter = strings.ToUpper(filter)
	}
	if alias, ok := topicAliases[filter]; ok {
		filter = alias
	}

	topic := topicType + "." + filter
	if interval == "" {
		return topic, nil
	}
	text := intervalText(interval)
	if text == string(StreamIntervalAll) {
		if !allIntervalTopicTypes[topicType] {
			return "", ssi.NewValidationError(fmt.Sprintf("interval '*' is not valid for %s", topicType))
		}
		if !concreteIndexRE.MatchString(filter) {
			return "", ssi.NewValidationError("interval '*' needs one concrete index or exchange code")
		}
		return topic + "@" + text, nil
	}
	if topicType != "trade" && topicType != "trade.index" {
		return "", ssi.NewValidationError(
			fmt.Sprintf("interval is only valid for trade and trade.index, not %s", topicType))
	}
	if !streamIntervals[text] {
		return "", ssi.NewValidationError(
			fmt.Sprintf("interval must be one of 1m, 5m, tick, got '%s'", text))
	}
	if topicType == "trade.index" && !concreteIndexRE.MatchString(filter) {
		return "", ssi.NewValidationError("trade.index intervals need one concrete index, not a wildcard")
	}
	return topic + "@" + text, nil
}

// BuildTopics builds one topic per filter; an empty list is an error.
func BuildTopics(topicType string, filters []string, interval market.Timeframe) ([]string, error) {
	if len(filters) == 0 {
		return nil, ssi.NewValidationError("at least one symbol is required")
	}
	topics := make([]string, 0, len(filters))
	for _, f := range filters {
		t, err := BuildTopic(topicType, f, interval)
		if err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, nil
}
