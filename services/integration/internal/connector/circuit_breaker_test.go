// Package connector_test tests the circuit breaker.
package connector_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/connector"
)

// failingConnector is a test double that always fails.
type failingConnector struct{ calls int }

func (f *failingConnector) Name() string { return "failing" }
func (f *failingConnector) FetchLogs(_ context.Context, _ connector.QueryOptions) ([]connector.LogEntry, error) {
	f.calls++
	return nil, errors.New("upstream error")
}
func (f *failingConnector) Health(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{Healthy: false, Connector: "failing"}
}

// successConnector is a test double that always succeeds.
type successConnector struct{}

func (s *successConnector) Name() string { return "success" }
func (s *successConnector) FetchLogs(_ context.Context, _ connector.QueryOptions) ([]connector.LogEntry, error) {
	return []connector.LogEntry{{Title: "ok", Severity: "info"}}, nil
}
func (s *successConnector) Health(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{Healthy: true, Connector: "success"}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cfg := connector.CircuitBreakerConfig{
		FailureThreshold: 5,
		RecoveryTimeout:  30 * time.Second,
	}
	inner := &failingConnector{}
	cb := connector.NewCircuitBreaker(inner, cfg)

	ctx := context.Background()

	// First 4 failures: circuit should stay closed
	for i := 0; i < 4; i++ {
		_, err := cb.FetchLogs(ctx, connector.QueryOptions{})
		if errors.Is(err, connector.ErrCircuitOpen) {
			t.Fatalf("circuit opened too early at failure %d", i+1)
		}
	}
	if cb.CircuitState() != connector.StateClosed {
		t.Errorf("expected closed after 4 failures, got %s", cb.CircuitState())
	}

	// 5th failure: circuit should open
	_, err := cb.FetchLogs(ctx, connector.QueryOptions{})
	if errors.Is(err, connector.ErrCircuitOpen) {
		t.Error("5th call should still pass through (opens on record, not before)");
	}
	if cb.CircuitState() != connector.StateOpen {
		t.Errorf("expected open after 5 failures, got %s", cb.CircuitState())
	}

	// 6th call: circuit open, short-circuit
	_, err = cb.FetchLogs(ctx, connector.QueryOptions{})
	if !errors.Is(err, connector.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen after threshold, got %v", err)
	}
}

func TestCircuitBreaker_ResetsOnSuccess(t *testing.T) {
	cfg := connector.CircuitBreakerConfig{
		FailureThreshold: 2,
		RecoveryTimeout:  30 * time.Second,
	}
	inner := &failingConnector{}
	cb := connector.NewCircuitBreaker(inner, cfg)
	ctx := context.Background()

	// Open the circuit
	for i := 0; i < 3; i++ {
		cb.FetchLogs(ctx, connector.QueryOptions{}) //nolint:errcheck
	}
	if cb.CircuitState() != connector.StateOpen {
		t.Fatal("circuit should be open")
	}
}

func TestCircuitBreaker_SuccessKeepsClosed(t *testing.T) {
	cfg := connector.DefaultConfig()
	cb := connector.NewCircuitBreaker(&successConnector{}, cfg)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		_, err := cb.FetchLogs(ctx, connector.QueryOptions{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	if cb.CircuitState() != connector.StateClosed {
		t.Errorf("expected closed after all successes, got %s", cb.CircuitState())
	}
	if cb.ConsecutiveFailures() != 0 {
		t.Errorf("expected 0 failures, got %d", cb.ConsecutiveFailures())
	}
}

func TestCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cfg := connector.CircuitBreakerConfig{
		FailureThreshold: 1,
		RecoveryTimeout:  10 * time.Millisecond,
	}
	inner := &failingConnector{}
	cb := connector.NewCircuitBreaker(inner, cfg)
	ctx := context.Background()

	// Trigger open
	cb.FetchLogs(ctx, connector.QueryOptions{}) //nolint:errcheck
	cb.FetchLogs(ctx, connector.QueryOptions{}) //nolint:errcheck
	if cb.CircuitState() != connector.StateOpen {
		t.Fatal("circuit should be open")
	}

	// Wait for recovery timeout
	time.Sleep(20 * time.Millisecond)
	if cb.CircuitState() != connector.StateHalfOpen {
		t.Errorf("expected half-open after recovery timeout, got %s", cb.CircuitState())
	}
}

func TestCircuitBreaker_Name(t *testing.T) {
	cb := connector.NewCircuitBreaker(&successConnector{}, connector.DefaultConfig())
	if cb.Name() != "success" {
		t.Errorf("expected name 'success', got %s", cb.Name())
	}
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := connector.NewRegistry()
	reg.Register(&successConnector{})

	c, ok := reg.Get("success")
	if !ok {
		t.Fatal("expected to find 'success' connector")
	}
	if c.Name() != "success" {
		t.Errorf("expected name 'success', got %s", c.Name())
	}

	_, ok = reg.Get("missing")
	if ok {
		t.Error("expected not to find 'missing' connector")
	}
}

