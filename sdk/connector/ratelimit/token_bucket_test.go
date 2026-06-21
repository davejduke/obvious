package ratelimit_test

import (
	"context"
	"testing"
	"time"

	"github.com/davejduke/obvious/sdk/connector/ratelimit"
)

func TestNew_InvalidConfig(t *testing.T) {
	if _, err := ratelimit.New(ratelimit.Config{Rate: 0, Burst: 10}); err == nil {
		t.Fatal("expected error for Rate=0")
	}
	if _, err := ratelimit.New(ratelimit.Config{Rate: 10, Burst: 0}); err == nil {
		t.Fatal("expected error for Burst=0")
	}
}

func TestTokenBucket_FullBucketOnStart(t *testing.T) {
	tb, err := ratelimit.New(ratelimit.Config{Rate: 1, Burst: 5})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if tokens := tb.Tokens(); tokens < 4.9 {
		t.Errorf("expected ~5 tokens on start, got %.2f", tokens)
	}
}

func TestTokenBucket_TryAcquire(t *testing.T) {
	tb, _ := ratelimit.New(ratelimit.Config{Rate: 1, Burst: 3})
	for i := 0; i < 3; i++ {
		if !tb.TryAcquire() {
			t.Fatalf("expected TryAcquire to succeed on attempt %d", i+1)
		}
	}
	if tb.TryAcquire() {
		t.Fatal("expected TryAcquire to fail when bucket is empty")
	}
}

func TestTokenBucket_Acquire_ContextCancel(t *testing.T) {
	// Rate=0.001 → takes ~1000s to refill; context cancels first.
	tb, _ := ratelimit.New(ratelimit.Config{Rate: 0.001, Burst: 1})
	tb.TryAcquire() // drain the single token

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := tb.Acquire(ctx)
	if err == nil {
		t.Fatal("expected context timeout error")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	// Rate=100 → 100 tokens/s → refills quickly.
	tb, _ := ratelimit.New(ratelimit.Config{Rate: 100, Burst: 10})
	// Drain all tokens.
	for tb.TryAcquire() {
	}
	// Wait 100ms → should get ~10 tokens back.
	time.Sleep(110 * time.Millisecond)
	if tokens := tb.Tokens(); tokens < 5 {
		t.Errorf("expected tokens to refill, got %.2f", tokens)
	}
}

func TestTokenBucket_WaitTime(t *testing.T) {
	tb, _ := ratelimit.New(ratelimit.Config{Rate: 10, Burst: 10})
	// Drain all.
	for tb.TryAcquire() {
	}
	w := tb.WaitTime(1)
	if w <= 0 {
		t.Errorf("expected positive wait time, got %v", w)
	}
	if w > 200*time.Millisecond {
		t.Errorf("wait time seems too large: %v (rate=10/s)", w)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := ratelimit.DefaultConfig()
	if cfg.Rate <= 0 || cfg.Burst <= 0 {
		t.Errorf("DefaultConfig has invalid values: %+v", cfg)
	}
	_, err := ratelimit.New(cfg)
	if err != nil {
		t.Fatalf("DefaultConfig should produce a valid TokenBucket: %v", err)
	}
}
