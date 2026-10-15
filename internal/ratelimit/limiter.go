package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

// Limiter is a token-bucket rate limiter (goroutine-safe) that follows what the server says.
//
// maxPerSecond is the caller's own ceiling (0 = unlimited). Cap lowers the effective rate to
// the server's advertised limit, and Pause holds every caller for a while (the server asked us
// to wait). A rate of 0 stays unlimited: the caller opted out.
type Limiter struct {
	configured int
	max        int
	tokens     float64
	last       time.Time
	pauseUntil time.Time
	mu         sync.Mutex
}

func New(maxPerSecond int) *Limiter {
	return &Limiter{
		configured: maxPerSecond,
		max:        maxPerSecond,
		tokens:     float64(maxPerSecond),
		last:       time.Now(),
	}
}

// Rate is the effective requests-per-second ceiling now (0 = unlimited).
func (r *Limiter) Rate() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.max
}

// Cap never goes faster than serverLimit per second (ignored when unlimited or invalid).
func (r *Limiter) Cap(serverLimit int) {
	if serverLimit <= 0 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.configured <= 0 {
		return
	}
	if serverLimit < r.configured {
		r.max = serverLimit
	} else {
		r.max = r.configured
	}
	r.tokens = math.Min(r.tokens, float64(r.max))
}

// PauseMs makes every Acquire wait until ms milliseconds from now (never shortens a pause).
func (r *Limiter) PauseMs(ms int) {
	if ms <= 0 {
		return
	}
	until := time.Now().Add(time.Duration(ms) * time.Millisecond)
	r.mu.Lock()
	defer r.mu.Unlock()
	if until.After(r.pauseUntil) {
		r.pauseUntil = until
	}
}

// reserve takes a token and returns 0, or returns how long to wait before trying again.
func (r *Limiter) reserve() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	if now.Before(r.pauseUntil) {
		return r.pauseUntil.Sub(now)
	}
	if r.max <= 0 {
		return 0
	}
	elapsed := now.Sub(r.last).Seconds()
	r.last = now
	r.tokens = math.Min(float64(r.max), r.tokens+elapsed*float64(r.max))
	if r.tokens >= 1 {
		r.tokens--
		return 0
	}
	return time.Duration((1.0 - r.tokens) / float64(r.max) * float64(time.Second))
}

// Acquire blocks until a request token is available.
func (r *Limiter) Acquire() {
	_ = r.AcquireCtx(context.Background())
}

// AcquireCtx blocks until a request token is available or ctx is done.
func (r *Limiter) AcquireCtx(ctx context.Context) error {
	for {
		wait := r.reserve()
		if wait <= 0 {
			return nil
		}
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}
