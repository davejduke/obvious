// Package circuit implements the circuit breaker pattern for the Connector SDK.
//
// The circuit breaker has three states:
//
//	Closed    — normal operation; requests pass through; failures are counted.
//	Open      — circuit tripped; requests are immediately rejected with ErrOpen.
//	HalfOpen  — recovery probe; one request is allowed through; on success the
//	            circuit closes, on failure it opens again.
//
// State transitions:
//
//	Closed → Open:     consecutive failure count reaches FailureThreshold
//	Open → HalfOpen:  RecoveryTimeout elapsed since the circuit opened
//	HalfOpen → Closed: probe request succeeds
//	HalfOpen → Open:   probe request fails
package circuit

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// State represents the operational state of the circuit breaker.
type State int

const (
	// StateClosed is normal operation — requests pass through.
	StateClosed State = iota
	// StateOpen means the circuit has tripped — requests are rejected.
	StateOpen
	// StateHalfOpen allows one probe request through to test recovery.
	StateHalfOpen
)

// String returns the state name for logging.
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

// ErrOpen is returned when an operation is rejected because the circuit is open.
var ErrOpen = errors.New("circuit breaker is open")

// Config holds tuning parameters for the circuit breaker.
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening.
	// Defaults to 5.
	FailureThreshold int

	// RecoveryTimeout is how long to wait in Open state before trying HalfOpen.
	// Defaults to 30s.
	RecoveryTimeout time.Duration

	// OnStateChange is called whenever the circuit changes state (optional).
	// The callback must not block.
	OnStateChange func(from, to State)
}

// DefaultConfig returns safe production defaults.
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
	}
}

// Breaker is a thread-safe circuit breaker.
type Breaker struct {
	mu               sync.Mutex
	cfg              Config
	state            State
	consecutiveFails int
	openedAt         time.Time
	halfOpenInFlight bool // true while a half-open probe is in flight
}

// New returns a Breaker in the Closed state.
func New(cfg Config) *Breaker {
	if cfg.FailureThreshold <= 0 {
		cfg.FailureThreshold = 5
	}
	if cfg.RecoveryTimeout <= 0 {
		cfg.RecoveryTimeout = 30 * time.Second
	}
	return &Breaker{cfg: cfg, state: StateClosed}
}

// Allow returns nil if the request should be forwarded, or ErrOpen if the
// circuit is open. Call Record when the operation completes.
func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maybeTransitionToHalfOpen()

	switch b.state {
	case StateOpen:
		return fmt.Errorf("%w (opens again in ~%s)", ErrOpen,
			b.cfg.RecoveryTimeout-time.Since(b.openedAt))
	case StateHalfOpen:
		if b.halfOpenInFlight {
			// Only one probe at a time in half-open state.
			return ErrOpen
		}
		b.halfOpenInFlight = true
	}
	return nil
}

// Record updates the circuit based on the operation outcome.
// err == nil means success; err != nil means failure.
func (b *Breaker) Record(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if err == nil {
		// Success — close the circuit and reset counters.
		if b.state != StateClosed {
			b.transition(StateClosed)
		}
		b.consecutiveFails = 0
		b.halfOpenInFlight = false
		return
	}

	// Failure.
	b.halfOpenInFlight = false
	b.consecutiveFails++

	if b.state == StateHalfOpen || (b.state == StateClosed && b.consecutiveFails >= b.cfg.FailureThreshold) {
		b.transition(StateOpen)
		b.openedAt = time.Now()
	}
}

// Do is a convenience wrapper: calls Allow, runs fn, then calls Record.
func (b *Breaker) Do(fn func() error) error {
	if err := b.Allow(); err != nil {
		return err
	}
	err := fn()
	b.Record(err)
	return err
}

// State returns the current circuit state (for observability).
func (b *Breaker) State() State {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.maybeTransitionToHalfOpen()
	return b.state
}

// ConsecutiveFailures returns the current consecutive failure count.
func (b *Breaker) ConsecutiveFailures() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.consecutiveFails
}

// transition updates the state and fires OnStateChange.
// MUST be called with b.mu held.
func (b *Breaker) transition(next State) {
	prev := b.state
	b.state = next
	if b.cfg.OnStateChange != nil && prev != next {
		go b.cfg.OnStateChange(prev, next)
	}
}

// maybeTransitionToHalfOpen advances Open → HalfOpen when RecoveryTimeout
// has elapsed. MUST be called with b.mu held.
func (b *Breaker) maybeTransitionToHalfOpen() {
	if b.state == StateOpen && time.Since(b.openedAt) >= b.cfg.RecoveryTimeout {
		b.transition(StateHalfOpen)
	}
}
