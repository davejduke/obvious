package retry_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davejduke/obvious/sdk/connector/retry"
)

var errTransient = errors.New("transient error")

func TestDo_SuccessOnFirstAttempt(t *testing.T) {
	calls := 0
	err := retry.Do(context.Background(), retry.DefaultConfig(), func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestDo_RetriesAndSucceeds(t *testing.T) {
	calls := 0
	cfg := retry.Config{MaxAttempts: 3, BaseDelay: time.Millisecond, Multiplier: 2}
	err := retry.Do(context.Background(), cfg, func() error {
		calls++
		if calls < 3 {
			return errTransient
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success after 3 attempts, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDo_ExhaustRetries(t *testing.T) {
	calls := 0
	cfg := retry.Config{MaxAttempts: 3, BaseDelay: time.Millisecond, Multiplier: 2}
	err := retry.Do(context.Background(), cfg, func() error {
		calls++
		return errTransient
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDo_PermanentError_NoRetry(t *testing.T) {
	calls := 0
	cfg := retry.Config{MaxAttempts: 5, BaseDelay: time.Millisecond, Multiplier: 2}
	err := retry.Do(context.Background(), cfg, func() error {
		calls++
		return retry.Permanent(errors.New("auth failed"))
	})
	if err == nil {
		t.Fatal("expected permanent error")
	}
	if !retry.IsPermanent(err) {
		t.Errorf("expected IsPermanent=true, got false for %v", err)
	}
	if calls != 1 {
		t.Errorf("expected only 1 call for permanent error, got %d", calls)
	}
}

func TestDo_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	cfg := retry.Config{MaxAttempts: 5, BaseDelay: time.Millisecond, Multiplier: 2}
	err := retry.Do(ctx, cfg, func() error {
		return errTransient
	})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestDoWithAttempts_ReturnsAttemptLog(t *testing.T) {
	calls := 0
	cfg := retry.Config{MaxAttempts: 3, BaseDelay: time.Millisecond, Multiplier: 2}
	attempts, err := retry.DoWithAttempts(context.Background(), cfg, func() error {
		calls++
		if calls < 2 {
			return errTransient
		}
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(attempts) != 2 {
		t.Errorf("expected 2 attempts recorded, got %d", len(attempts))
	}
	if attempts[0].Err == nil {
		t.Error("first attempt should have an error")
	}
	if attempts[1].Err != nil {
		t.Error("second attempt should succeed")
	}
}

func TestDefaultConfig_IsValid(t *testing.T) {
	cfg := retry.DefaultConfig()
	if cfg.MaxAttempts <= 0 {
		t.Errorf("MaxAttempts must be > 0, got %d", cfg.MaxAttempts)
	}
	if cfg.BaseDelay <= 0 {
		t.Errorf("BaseDelay must be > 0, got %v", cfg.BaseDelay)
	}
}
