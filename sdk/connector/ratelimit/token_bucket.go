// Package ratelimit provides a token bucket rate limiter for the Connector SDK.
//
// The token bucket algorithm allows short bursts while enforcing a long-term
// average rate. Tokens accumulate at a configurable refill rate up to a
// maximum burst capacity. Each Acquire call consumes one token; if no tokens
// are available the call blocks until one becomes available or the context
// is cancelled.
package ratelimit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Config holds parameters for the token bucket.
type Config struct {
	// Rate is the number of tokens added per second (refill rate).
	Rate float64

	// Burst is the maximum number of tokens the bucket can hold.
	// This controls the maximum burst size.
	Burst float64
}

// DefaultConfig returns safe defaults: 10 req/s with a burst of 20.
func DefaultConfig() Config {
	return Config{Rate: 10, Burst: 20}
}

// TokenBucket is a thread-safe token bucket rate limiter.
type TokenBucket struct {
	mu         sync.Mutex
	cfg        Config
	tokens     float64
	lastRefill time.Time
}

// New creates a TokenBucket with the given Config.
// The bucket starts full (tokens == Burst).
func New(cfg Config) (*TokenBucket, error) {
	if cfg.Rate <= 0 {
		return nil, fmt.Errorf("ratelimit: Rate must be > 0, got %v", cfg.Rate)
	}
	if cfg.Burst <= 0 {
		return nil, fmt.Errorf("ratelimit: Burst must be > 0, got %v", cfg.Burst)
	}
	return &TokenBucket{
		cfg:        cfg,
		tokens:     cfg.Burst,
		lastRefill: time.Now(),
	}, nil
}

// Acquire blocks until a token is available or ctx is cancelled.
// Returns ctx.Err() if the context expires.
func (tb *TokenBucket) Acquire(ctx context.Context) error {
	for {
		tb.mu.Lock()
		tb.refill()
		if tb.tokens >= 1 {
			tb.tokens--
			tb.mu.Unlock()
			return nil
		}
		// Calculate how long until one token is available.
		wait := time.Duration((1-tb.tokens)/tb.cfg.Rate*float64(time.Second)) + time.Millisecond
		tb.mu.Unlock()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
			// Retry after waiting.
		}
	}
}

// TryAcquire returns true and consumes a token if one is available immediately.
// Returns false (non-blocking) when the bucket is empty.
func (tb *TokenBucket) TryAcquire() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// Tokens returns the current number of available tokens (for observability).
func (tb *TokenBucket) Tokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	return tb.tokens
}

// WaitTime returns how long the caller would need to wait for n tokens.
// Returns 0 if tokens are immediately available.
func (tb *TokenBucket) WaitTime(n float64) time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.refill()
	if tb.tokens >= n {
		return 0
	}
	deficit := n - tb.tokens
	return time.Duration(deficit / tb.cfg.Rate * float64(time.Second))
}

// refill adds tokens based on elapsed time since last refill.
// MUST be called with tb.mu held.
func (tb *TokenBucket) refill() {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.cfg.Rate
	if tb.tokens > tb.cfg.Burst {
		tb.tokens = tb.cfg.Burst
	}
	tb.lastRefill = now
}
