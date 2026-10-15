package transport

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"

	ssi "github.com/SSI-Securities-Inc/ssi-sdk-go/v3"
)

// ConnectionEvent is a change in the state of a streaming connection (see OnConnection).
//
// State is one of "connected" (first connect), "reconnected" (re-opened after a drop,
// subscriptions restored), "disconnected" (the socket dropped; a reconnect follows when
// AutoReconnect is on) and "failed" (unrecoverable, e.g. a new OTP is needed; Err says why and no
// reconnect will happen).
type ConnectionEvent struct {
	State          string
	ReconnectCount int
	Err            error
}

// RejectionKind is why the server refused a connection it had already accepted.
type RejectionKind int

const (
	// RejectionAuth is 401/403: the token is not (or no longer) valid.
	RejectionAuth RejectionKind = iota
	// RejectionRateLimited is 429: more than 10 sockets for this client.
	RejectionRateLimited
	// RejectionTransient is 5xx and anything else: try again later.
	RejectionTransient
)

// BackoffDelayMs is the exponential backoff with jitter, bounded by maxDelayMs so a loop can
// never run hot. attempt is 0 for the first retry; baseMs is the first retry's delay in
// milliseconds; jitter is a uniform random number in [0, 1) that scales the delay into
// [50%, 100%] of the exponential value so many clients do not retry in step. A maxDelayMs <= 0
// means ssi.WSReconnectMaxDelayMs. The result (milliseconds) is at least 50.
func BackoffDelayMs(attempt int, baseMs, maxDelayMs int, jitter float64) int {
	if maxDelayMs <= 0 {
		maxDelayMs = ssi.WSReconnectMaxDelayMs
	}
	if attempt < 0 {
		attempt = 0
	}
	if attempt > 30 {
		attempt = 30
	}
	exponential := math.Min(float64(maxDelayMs), math.Max(float64(baseMs), 0)*math.Pow(2, float64(attempt)))
	jitter = math.Min(math.Max(jitter, 0), 1)
	return int(math.Max(50, math.Round(exponential*(0.5+0.5*jitter))))
}

// ClassifyRejection classifies a {"code","msg"} frame sent when the server refuses a
// connection. retryAfterMs is only set when the frame says so; the frame's retryAfter field is
// in seconds and is converted here.
func ClassifyRejection(message map[string]interface{}) (kind RejectionKind, retryAfterMs *int) {
	code := 0
	if f, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(message["code"])), 64); err == nil {
		code = int(f)
	}
	for _, key := range []string{"retryAfter", "retry_after"} {
		raw, ok := message[key]
		if !ok || raw == nil {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(fmt.Sprint(raw)), 64)
		if err != nil || f < 0 {
			continue
		}
		ms := int(math.Round(math.Min(f, 1e6) * 1000))
		retryAfterMs = &ms
		break
	}
	switch code {
	case 401, 403:
		return RejectionAuth, retryAfterMs
	case 429:
		return RejectionRateLimited, retryAfterMs
	}
	return RejectionTransient, retryAfterMs
}

// RateLimitWaitMs is the wait after a 429 in milliseconds: the server's Retry-After if longer,
// never under the floor.
func RateLimitWaitMs(retryAfterMs *int) int {
	wait := ssi.WSRateLimitBackoffMs
	if retryAfterMs != nil && *retryAfterMs > wait {
		wait = *retryAfterMs
	}
	return wait
}

// MsUntilRefresh is the time in milliseconds until the socket should be re-opened with a fresh
// token. The server only checks the JWT when a socket connects and never closes one whose token
// expires, so the client has to cycle the connection itself. expiresAtMs is the epoch
// millisecond when the credentials stop working (the earlier of access and refresh token); <= 0
// means unknown. ok is false when no expiry is known (never refresh proactively). A leadMs <= 0
// means ssi.WSTokenRefreshLeadMs.
func MsUntilRefresh(expiresAtMs, nowMs int64, leadMs int) (ms int64, ok bool) {
	if expiresAtMs <= 0 {
		return 0, false
	}
	if leadMs <= 0 {
		leadMs = ssi.WSTokenRefreshLeadMs
	}
	if d := expiresAtMs - int64(leadMs) - nowMs; d > 0 {
		return d, true
	}
	return 0, true
}

// SubscriptionBook remembers what was subscribed so it can be re-sent after a reconnect.
type SubscriptionBook struct {
	mu       sync.Mutex
	channels []string
	topics   map[string][]string
}

// NewSubscriptionBook starts with nothing subscribed.
func NewSubscriptionBook() *SubscriptionBook {
	return &SubscriptionBook{topics: map[string][]string{}}
}

// Record applies an outgoing subscribe/unsubscribe request to the book (others are ignored).
func (b *SubscriptionBook) Record(request map[string]interface{}) {
	method := strings.ToLower(fmt.Sprint(request["method"]))
	channel, ok := request["channel"].(string)
	topics, okTopics := topicsOf(request["topics"])
	if !ok || !okTopics {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	switch method {
	case "subscribe":
		if _, exists := b.topics[channel]; !exists {
			b.channels = append(b.channels, channel)
		}
		for _, t := range topics {
			if !contains(b.topics[channel], t) {
				b.topics[channel] = append(b.topics[channel], t)
			}
		}
	case "unsubscribe":
		current := b.topics[channel]
		kept := current[:0:0]
		for _, t := range current {
			if !contains(topics, t) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(b.topics, channel)
			for i, c := range b.channels {
				if c == channel {
					b.channels = append(b.channels[:i:i], b.channels[i+1:]...)
					break
				}
			}
		} else {
			b.topics[channel] = kept
		}
	}
}

// Requests is one subscribe request per channel that has topics, ready to send again.
func (b *SubscriptionBook) Requests() []map[string]interface{} {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []map[string]interface{}
	for _, channel := range b.channels {
		topics := b.topics[channel]
		if len(topics) == 0 {
			continue
		}
		out = append(out, map[string]interface{}{
			"method":  "subscribe",
			"channel": channel,
			"topics":  append([]string{}, topics...),
		})
	}
	return out
}

// Clear forgets everything (the caller is done with the connection).
func (b *SubscriptionBook) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.channels = nil
	b.topics = map[string][]string{}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
