// Package retry provides exponential backoff retry for the Connector SDK.
// The retry pattern mirrors the webhook delivery service (§4.5) — each
// attempt waits BaseDelay * Multiplier^(attempt-1), capped at MaxDelay,
// with optional jitter to prevent thundering-herd.
package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// Config controls the retry behaviour.
type Config struct {
	// MaxAttempts is the total number of attempts (first try + retries).
	// Defaults to 3 when zero.
	MaxAttempts int

	// BaseDelay is the wait time before the second attempt.
	// Defaults to 500ms when zero.
	BaseDelay time.Duration

	// MaxDelay caps the inter-attempt delay. Defaults to 30s when zero.
	MaxDelay time.Duration

	// Multiplier is the exponential backoff factor. Defaults to 2.0 when zero.
	Multiplier float64

	// Jitter adds a random fraction of the current delay to avoid thundering
	// herd. Set to 0 to disable. Useful values: 0.1–0.3.
	Jitter float64
}

// DefaultConfig returns safe defaults: 3 attempts, 500ms base, 30s cap, ×2.
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		BaseDelay:   500 * time.Millisecond,
		MaxDelay:    30 * time.Second,
		Multiplier:  2.0,
		Jitter:      0.1,
	}
}

// PermanentError wraps an error and signals that the operation should not be
// retried. Wrap errors with this type in your fn when a retry would be futile
// (e.g. HTTP 400 Bad Request, auth failure).
type PermanentError struct {
	Err error
}

func (e *PermanentError) Error() string { return "permanent: " + e.Err.Error() }
func (e *PermanentError) Unwrap() error { return e.Err }

// IsPermanent returns true if err is (or wraps) a PermanentError.
func IsPermanent(err error) bool {
	var pe *PermanentError
	return errors.As(err, &pe)
}

// Permanent wraps err as a PermanentError, signalling no further retries.
func Permanent(err error) error { return &PermanentError{Err: err} }

// Attempt records the outcome of a single retry attempt.
type Attempt struct {
	Number int
	Err    error
	Delay  time.Duration // delay applied AFTER this failed attempt (0 on last)
}

// Do calls fn up to cfg.MaxAttempts times with exponential backoff.
// It returns nil on the first success. Retries stop immediately when:
//   - fn returns nil (success)
//   - fn returns a PermanentError
//   - ctx is cancelled
//   - MaxAttempts is exhausted
//
// The returned error on exhaustion is wrapped with attempt metadata.
func Do(ctx context.Context, cfg Config, fn func() error) error {
	cfg = applyDefaults(cfg)

	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("retry: context cancelled on attempt %d: %w", attempt, err)
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
		if IsPermanent(lastErr) {
			return lastErr
		}

		if attempt == cfg.MaxAttempts {
			break
		}

		delay := delayForAttempt(cfg, attempt)
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry: context cancelled waiting for attempt %d: %w", attempt+1, ctx.Err())
		case <-time.After(delay):
		}
	}

	return fmt.Errorf("retry: exhausted %d attempts, last error: %w", cfg.MaxAttempts, lastErr)
}

// DoWithAttempts is like Do but also returns a slice of Attempt records for
// observability. Useful in tests and monitoring.
func DoWithAttempts(ctx context.Context, cfg Config, fn func() error) ([]Attempt, error) {
	cfg = applyDefaults(cfg)

	var attempts []Attempt
	var lastErr error

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return attempts, fmt.Errorf("retry: context cancelled on attempt %d: %w", attempt, err)
		}

		lastErr = fn()
		var delay time.Duration

		if lastErr == nil {
			attempts = append(attempts, Attempt{Number: attempt, Err: nil})
			return attempts, nil
		}
		if IsPermanent(lastErr) {
			attempts = append(attempts, Attempt{Number: attempt, Err: lastErr})
			return attempts, lastErr
		}

		if attempt < cfg.MaxAttempts {
			delay = delayForAttempt(cfg, attempt)
		}
		attempts = append(attempts, Attempt{Number: attempt, Err: lastErr, Delay: delay})

		if attempt == cfg.MaxAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return attempts, fmt.Errorf("retry: context cancelled waiting for attempt %d: %w", attempt+1, ctx.Err())
		case <-time.After(delay):
		}
	}

	return attempts, fmt.Errorf("retry: exhausted %d attempts, last error: %w", cfg.MaxAttempts, lastErr)
}

// applyDefaults fills zero values with sensible defaults.
func applyDefaults(cfg Config) Config {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.BaseDelay <= 0 {
		cfg.BaseDelay = 500 * time.Millisecond
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	return cfg
}

// delayForAttempt computes the backoff duration for a given attempt number
// (1-based). Attempt 1 failed → delay for attempt 2, etc.
func delayForAttempt(cfg Config, attempt int) time.Duration {
	delay := float64(cfg.BaseDelay) * math.Pow(cfg.Multiplier, float64(attempt-1))
	if cfg.Jitter > 0 {
		delay += delay * cfg.Jitter * rand.Float64() //nolint:gosec // non-crypto jitter
	}
	if d := time.Duration(delay); d > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return time.Duration(delay)
}
