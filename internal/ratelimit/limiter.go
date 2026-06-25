package ratelimit

import (
	"math"
	"sync"
	"time"
)

// Limiter is a token-bucket rate limiter (thread-safe).
type Limiter struct {
	max    int
	tokens float64
	last   time.Time
	mu     sync.Mutex
}

func New(maxPerSecond int) *Limiter {
	return &Limiter{
		max:    maxPerSecond,
		tokens: float64(maxPerSecond),
		last:   time.Now(),
	}
}

func (r *Limiter) Acquire() {
	if r.max <= 0 {
		return
	}
	for {
		r.mu.Lock()
		now := time.Now()
		elapsed := now.Sub(r.last).Seconds()
		r.last = now
		r.tokens = math.Min(float64(r.max), r.tokens+elapsed*float64(r.max))
		if r.tokens >= 1 {
			r.tokens--
			r.mu.Unlock()
			return
		}
		deficit := 1.0 - r.tokens
		waitSeconds := deficit / float64(r.max)
		r.mu.Unlock()
		time.Sleep(time.Duration(waitSeconds * float64(time.Second)))
	}
}
