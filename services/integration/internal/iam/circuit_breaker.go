package iam

import (
	"context"
	"sync"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/connector"
)

// IAMCircuitBreaker wraps an IAMConnector with the same failure-tracking and
// short-circuit logic used by the SIEM connector.CircuitBreaker.
// It reuses connector.State, connector.ErrCircuitOpen, and
// connector.CircuitBreakerConfig to avoid duplicating the state machine.
type IAMCircuitBreaker struct {
	mu               sync.Mutex
	inner            IAMConnector
	config           connector.CircuitBreakerConfig
	state            connector.State
	consecutiveFails int
	openedAt         time.Time
}

// NewIAMCircuitBreaker wraps the given IAMConnector with circuit breaker logic.
func NewIAMCircuitBreaker(inner IAMConnector, cfg connector.CircuitBreakerConfig) *IAMCircuitBreaker {
	return &IAMCircuitBreaker{
		inner:  inner,
		config: cfg,
		state:  connector.StateClosed,
	}
}

// Name implements IAMConnector.
func (cb *IAMCircuitBreaker) Name() string { return cb.inner.Name() }

// CircuitState returns the current circuit state (for observability / tests).
func (cb *IAMCircuitBreaker) CircuitState() connector.State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.maybeTransitionToHalfOpen()
	return cb.state
}

// ConsecutiveFailures returns the current failure counter (for tests).
func (cb *IAMCircuitBreaker) ConsecutiveFailures() int {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.consecutiveFails
}

// Sync implements IAMConnector with circuit breaker protection.
func (cb *IAMCircuitBreaker) Sync(ctx context.Context) (*IAMSnapshot, error) {
	cb.mu.Lock()
	cb.maybeTransitionToHalfOpen()
	if cb.state == connector.StateOpen {
		cb.mu.Unlock()
		return nil, connector.ErrCircuitOpen
	}
	cb.mu.Unlock()

	snap, err := cb.inner.Sync(ctx)
	cb.record(err)
	return snap, err
}

// Health implements IAMConnector — circuit state does not block health checks.
func (cb *IAMCircuitBreaker) Health(ctx context.Context) connector.HealthStatus {
	status := cb.inner.Health(ctx)
	cb.mu.Lock()
	cb.maybeTransitionToHalfOpen()
	state := cb.state
	cb.mu.Unlock()
	if state != connector.StateClosed {
		status.Message = "circuit " + state.String() + "; " + status.Message
	}
	return status
}

// record updates circuit state based on whether the last call succeeded.
// Must be called without holding mu.
func (cb *IAMCircuitBreaker) record(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if err == nil {
		cb.consecutiveFails = 0
		cb.state = connector.StateClosed
		return
	}
	cb.consecutiveFails++
	if cb.consecutiveFails >= cb.config.FailureThreshold && cb.state != connector.StateOpen {
		cb.state = connector.StateOpen
		cb.openedAt = time.Now()
	}
}

// maybeTransitionToHalfOpen moves Open → HalfOpen when recovery timeout expires.
// Must be called with mu held.
func (cb *IAMCircuitBreaker) maybeTransitionToHalfOpen() {
	if cb.state == connector.StateOpen && time.Since(cb.openedAt) >= cb.config.RecoveryTimeout {
		cb.state = connector.StateHalfOpen
	}
}
