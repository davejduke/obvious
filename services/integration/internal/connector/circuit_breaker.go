// Package connector provides the circuit breaker pattern for external integrations.
package connector

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State represents the circuit breaker state.
type State int

const (
	StateClosed   State = iota // Normal operation — requests pass through.
	StateOpen                  // Circuit open — requests are short-circuited.
	StateHalfOpen              // Probe state — one request allowed to test recovery.
)

// String returns a human-readable circuit state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when a request is rejected by an open circuit.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig holds tuning parameters.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold int
	// RecoveryTimeout is how long to wait before trying half-open.
	RecoveryTimeout time.Duration
}

// DefaultConfig returns safe defaults: open after 5 failures, try recovery after 30s.
func DefaultConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
	}
}

// CircuitBreaker wraps a Connector with failure tracking and short-circuit logic.
type CircuitBreaker struct {
	mu               sync.Mutex
	inner            Connector
	config           CircuitBreakerConfig
	state            State
	consecutiveFails int
	openedAt         time.Time
}

// NewCircuitBreaker wraps the given connector with circuit breaker logic.
func NewCircuitBreaker(inner Connector, cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		inner:  inner,
		config: cfg,
		state:  StateClosed,
	}
}

// Name implements Connector.
func (cb *CircuitBreaker) Name() string {
	return cb.inner.Name()
}

// State returns the current circuit state (for observability).
func (cb *CircuitBreaker) CircuitState() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.maybeTransitionToHalfOpen()
	return cb.state
}

// ConsecutiveFailures returns current failure count (for tests/observability).
func (cb *CircuitBreaker) ConsecutiveFailures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.consecutiveFails
}

// FetchLogs implements Connector with circuit breaker protection.
func (cb *CircuitBreaker) FetchLogs(ctx context.Context, opts QueryOptions) ([]LogEntry, error) {
	cb.mu.Lock()
	cb.maybeTransitionToHalfOpen()
	if cb.state == StateOpen {
		cb.mu.Unlock()
		return nil, ErrCircuitOpen
	}
	// Allow the probe in half-open.
	cb.mu.Unlock()

	entries, err := cb.inner.FetchLogs(ctx, opts)
	cb.record(err)
	return entries, err
}

// Health implements Connector — circuit state does not block health checks.
func (cb *CircuitBreaker) Health(ctx context.Context) HealthStatus {
	status := cb.inner.Health(ctx)
	cb.mu.Lock()
	cb.maybeTransitionToHalfOpen()
	state := cb.state
	cb.mu.Unlock()
	if state != StateClosed {
		status.Message = "circuit " + state.String() + "; " + status.Message
	}
	return status
}

// record updates the circuit state based on whether the last call succeeded.
// MUST be called without holding mu.
func (cb *CircuitBreaker) record(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err == nil {
		cb.consecutiveFails = 0
		cb.state = StateClosed
		return
	}
	cb.consecutiveFails++
	if cb.consecutiveFails >= cb.config.FailureThreshold && cb.state != StateOpen {
		cb.state = StateOpen
		cb.openedAt = time.Now()
	}
}

// maybeTransitionToHalfOpen moves Open → HalfOpen when the recovery timeout expires.
// MUST be called with mu held.
func (cb *CircuitBreaker) maybeTransitionToHalfOpen() {
	if cb.state == StateOpen && time.Since(cb.openedAt) >= cb.config.RecoveryTimeout {
		cb.state = StateHalfOpen
	}
}

