package circuit_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/davejduke/obvious/sdk/connector/circuit"
)

var errFake = errors.New("fake error")

func TestBreaker_InitiallyClosed(t *testing.T) {
	b := circuit.New(circuit.DefaultConfig())
	if b.State() != circuit.StateClosed {
		t.Errorf("expected Closed state initially, got %s", b.State())
	}
}

func TestBreaker_OpensAfterThreshold(t *testing.T) {
	b := circuit.New(circuit.Config{FailureThreshold: 3, RecoveryTimeout: 1 * time.Hour})

	for i := 0; i < 3; i++ {
		if err := b.Allow(); err != nil {
			t.Fatalf("Allow should succeed in closed state, got %v", err)
		}
		b.Record(errFake)
	}

	if b.State() != circuit.StateOpen {
		t.Errorf("expected Open after %d failures, got %s", 3, b.State())
	}
	if err := b.Allow(); !errors.Is(err, circuit.ErrOpen) {
		t.Errorf("expected ErrOpen from open circuit, got %v", err)
	}
}

func TestBreaker_ClosesOnSuccess(t *testing.T) {
	b := circuit.New(circuit.Config{FailureThreshold: 1, RecoveryTimeout: 1 * time.Hour})
	_ = b.Allow()
	b.Record(errFake) // Open

	// Manually force HalfOpen by backdating openedAt via zero recovery timeout.
	b2 := circuit.New(circuit.Config{FailureThreshold: 1, RecoveryTimeout: time.Nanosecond})
	_ = b2.Allow()
	b2.Record(errFake) // Open → immediately transitions to HalfOpen on next Allow

	if err := b2.Allow(); err != nil {
		t.Fatalf("expected Allow in half-open state, got %v", err)
	}
	b2.Record(nil) // success → Closed
	if b2.State() != circuit.StateClosed {
		t.Errorf("expected Closed after success in half-open, got %s", b2.State())
	}
}

func TestBreaker_HalfOpenRejectsSecondRequest(t *testing.T) {
	b := circuit.New(circuit.Config{FailureThreshold: 1, RecoveryTimeout: time.Nanosecond})
	_ = b.Allow()
	b.Record(errFake) // Open

	// First allow (probe).
	if err := b.Allow(); err != nil {
		t.Fatalf("first allow in half-open should succeed: %v", err)
	}
	// Second allow while probe is in flight — should reject.
	if err := b.Allow(); !errors.Is(err, circuit.ErrOpen) {
		t.Errorf("second allow in half-open should return ErrOpen, got %v", err)
	}
}

func TestBreaker_DoConvenience(t *testing.T) {
	b := circuit.New(circuit.Config{FailureThreshold: 3, RecoveryTimeout: 1 * time.Hour})

	// Successful Do.
	err := b.Do(func() error { return nil })
	if err != nil {
		t.Errorf("Do with nil fn should succeed, got %v", err)
	}

	// Trip the circuit.
	for i := 0; i < 3; i++ {
		_ = b.Do(func() error { return errFake })
	}
	// Do should now return ErrOpen without calling fn.
	called := false
	err = b.Do(func() error {
		called = true
		return nil
	})
	if !errors.Is(err, circuit.ErrOpen) {
		t.Errorf("expected ErrOpen, got %v", err)
	}
	if called {
		t.Error("fn should not be called when circuit is open")
	}
}

func TestBreaker_ConsecutiveFailures(t *testing.T) {
	b := circuit.New(circuit.Config{FailureThreshold: 5, RecoveryTimeout: 1 * time.Hour})
	for i := 1; i <= 4; i++ {
		_ = b.Allow()
		b.Record(errFake)
		if got := b.ConsecutiveFailures(); got != i {
			t.Errorf("after %d failures, ConsecutiveFailures=%d, want %d", i, got, i)
		}
	}
}

func TestBreaker_StateString(t *testing.T) {
	cases := []struct {
		state circuit.State
		want  string
	}{
		{circuit.StateClosed, "closed"},
		{circuit.StateOpen, "open"},
		{circuit.StateHalfOpen, "half-open"},
	}
	for _, tc := range cases {
		if got := tc.state.String(); got != tc.want {
			t.Errorf("State(%d).String() = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestBreaker_OnStateChange(t *testing.T) {
	var mu sync.Mutex
	var transitions []string
	b := circuit.New(circuit.Config{
		FailureThreshold: 1,
		RecoveryTimeout:  time.Nanosecond,
		OnStateChange: func(from, to circuit.State) {
			mu.Lock()
			transitions = append(transitions, from.String()+"→"+to.String())
			mu.Unlock()
		},
	})

	// Trigger Closed→Open.
	_ = b.Allow()
	b.Record(errFake)
	time.Sleep(20 * time.Millisecond) // allow async callback to run

	// Trigger Open→HalfOpen (recovery timeout=1ns) then HalfOpen→Closed.
	_ = b.Allow()
	b.Record(nil)
	time.Sleep(20 * time.Millisecond)

	mu.Lock()
	found := func(s string) bool {
		for _, tr := range transitions {
			if tr == s {
				return true
			}
		}
		return false
	}
	hasClosedOpen := found("closed→open")
	mu.Unlock()

	if !hasClosedOpen {
		mu.Lock()
		all := make([]string, len(transitions))
		copy(all, transitions)
		mu.Unlock()
		t.Errorf("missing closed→open transition, got %v", all)
	}
}
