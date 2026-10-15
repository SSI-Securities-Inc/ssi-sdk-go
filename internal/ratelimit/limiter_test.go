package ratelimit

import (
	"context"
	"testing"
	"time"
)

func TestCapLowersRateButNeverRaises(t *testing.T) {
	l := New(10)
	l.Cap(3)
	if l.Rate() != 3 {
		t.Fatalf("want 3, got %d", l.Rate())
	}
	l.Cap(50)
	if l.Rate() != 10 {
		t.Fatalf("cap above configured must fall back to configured, got %d", l.Rate())
	}
	l.Cap(0)
	l.Cap(-1)
	if l.Rate() != 10 {
		t.Fatalf("invalid caps are ignored, got %d", l.Rate())
	}
}

func TestUnlimitedStaysUnlimited(t *testing.T) {
	l := New(0)
	l.Cap(2)
	if l.Rate() != 0 {
		t.Fatalf("unlimited limiter must stay unlimited, got %d", l.Rate())
	}
}

func TestPauseBlocksAcquire(t *testing.T) {
	l := New(0)
	l.PauseMs(150)
	l.PauseMs(10) // never shortens
	start := time.Now()
	l.Acquire()
	if d := time.Since(start); d < 120*time.Millisecond {
		t.Fatalf("Acquire returned after %v, expected to wait for the pause", d)
	}
}

func TestAcquireCtxCancel(t *testing.T) {
	l := New(0)
	l.PauseMs(5000)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := l.AcquireCtx(ctx); err == nil {
		t.Fatal("expected context error")
	}
}

func TestBucketThrottles(t *testing.T) {
	l := New(20)
	start := time.Now()
	for i := 0; i < 24; i++ {
		l.Acquire()
	}
	if time.Since(start) < 150*time.Millisecond {
		t.Fatal("bucket did not throttle")
	}
}
