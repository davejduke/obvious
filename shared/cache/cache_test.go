package cache_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/davejduke/obvious/shared/cache"
)

// newTestClient spins up a miniredis instance and returns a Client backed by it.
func newTestClient(t *testing.T) (*cache.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return cache.NewFromRaw(rdb), mr
}

// ─── Session Cache ────────────────────────────────────────────────────────────

func TestSessionCache_SetGet(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	entry := &cache.SessionEntry{
		UserID:    "user-1",
		OrgID:     "org-1",
		TokenID:   "jti-abc",
		Roles:     []string{"internal_auditor"},
		IssuedAt:  time.Now().UTC(),
		ExpiresAt: time.Now().Add(8 * time.Hour).UTC(),
	}

	if err := c.Session.Set(ctx, "user-1", entry); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := c.Session.Get(ctx, "user-1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get: expected entry, got nil")
	}
	if got.UserID != entry.UserID {
		t.Errorf("UserID: want %q, got %q", entry.UserID, got.UserID)
	}
	if got.TokenID != entry.TokenID {
		t.Errorf("TokenID: want %q, got %q", entry.TokenID, got.TokenID)
	}
}

func TestSessionCache_Miss(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	got, err := c.Session.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get: unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("Get: want nil, got %+v", got)
	}
}

func TestSessionCache_Delete(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	entry := &cache.SessionEntry{UserID: "user-2", OrgID: "org-1", TokenID: "jti-xyz"}
	if err := c.Session.Set(ctx, "user-2", entry); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Session.Delete(ctx, "user-2"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	got, err := c.Session.Get(ctx, "user-2")
	if err != nil {
		t.Fatalf("Get after Delete: %v", err)
	}
	if got != nil {
		t.Errorf("Get after Delete: want nil, got %+v", got)
	}
}

func TestSessionCache_TTL(t *testing.T) {
	c, mr := newTestClient(t)
	ctx := context.Background()

	entry := &cache.SessionEntry{UserID: "user-3", OrgID: "org-1", TokenID: "jti-ttl"}
	if err := c.Session.Set(ctx, "user-3", entry); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Fast-forward miniredis time past the TTL.
	mr.FastForward(cache.SessionTTL + time.Second)

	got, err := c.Session.Get(ctx, "user-3")
	if err != nil {
		t.Fatalf("Get after TTL: %v", err)
	}
	if got != nil {
		t.Errorf("Get after TTL: want nil (expired), got %+v", got)
	}
}

// ─── Rate Limit Cache ─────────────────────────────────────────────────────────

func TestRateLimitCache_AllowUnderLimit(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	window := time.Minute
	const limit = 5

	for i := 0; i < limit; i++ {
		res, err := c.RateLimit.Increment(ctx, "org-1", "/api/v1/engagements", window, limit)
		if err != nil {
			t.Fatalf("Increment #%d: %v", i, err)
		}
		if !res.Allowed {
			t.Errorf("Increment #%d: want allowed, got denied (count=%d)", i, res.Count)
		}
	}
}

func TestRateLimitCache_DenyOverLimit(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	window := time.Minute
	const limit = 3

	// Use up the limit.
	for i := 0; i < int(limit); i++ {
		if _, err := c.RateLimit.Increment(ctx, "org-2", "/api/v1/findings", window, limit); err != nil {
			t.Fatalf("Increment #%d: %v", i, err)
		}
	}
	// One more — must be denied.
	res, err := c.RateLimit.Increment(ctx, "org-2", "/api/v1/findings", window, limit)
	if err != nil {
		t.Fatalf("Increment over limit: %v", err)
	}
	if res.Allowed {
		t.Errorf("Increment over limit: want denied, got allowed (count=%d)", res.Count)
	}
	if res.RetryAfter == 0 {
		t.Error("RetryAfter should be > 0 when denied")
	}
}

func TestRateLimitCache_Reset(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	const limit = 1
	window := time.Minute

	// Exhaust.
	c.RateLimit.Increment(ctx, "org-3", "/health", window, limit) //nolint:errcheck
	c.RateLimit.Increment(ctx, "org-3", "/health", window, limit) //nolint:errcheck

	// Reset.
	if err := c.RateLimit.Reset(ctx, "org-3", "/health"); err != nil {
		t.Fatalf("Reset: %v", err)
	}
	// Now should be allowed again.
	res, err := c.RateLimit.Increment(ctx, "org-3", "/health", window, limit)
	if err != nil {
		t.Fatalf("Increment after Reset: %v", err)
	}
	if !res.Allowed {
		t.Errorf("Increment after Reset: want allowed, got denied")
	}
}

// ─── Scope Cache ──────────────────────────────────────────────────────────────

func TestScopeCache_SetGet(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	dag, _ := json.Marshal(map[string]interface{}{"nodes": []string{"ctrl-1", "ctrl-2"}})
	entry := &cache.ScopeCacheEntry{
		EngagementID: "eng-1",
		Version:      1,
		DAG:          dag,
		ComputedAt:   time.Now().UTC(),
	}

	if err := c.Scope.Set(ctx, "eng-1", 1, entry); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Scope.Get(ctx, "eng-1", 1)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("Get: expected entry, got nil")
	}
	if got.EngagementID != "eng-1" {
		t.Errorf("EngagementID: want %q, got %q", "eng-1", got.EngagementID)
	}
}

func TestScopeCache_TTL_Expires(t *testing.T) {
	c, mr := newTestClient(t)
	ctx := context.Background()

	dag, _ := json.Marshal(map[string]interface{}{"nodes": []string{"ctrl-1"}})
	entry := &cache.ScopeCacheEntry{
		EngagementID: "eng-2",
		Version:      1,
		DAG:          dag,
		ComputedAt:   time.Now().UTC(),
	}
	if err := c.Scope.Set(ctx, "eng-2", 1, entry); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// Advance past 30-minute TTL.
	mr.FastForward(cache.ScopeTTL + time.Second)

	got, err := c.Scope.Get(ctx, "eng-2", 1)
	if err != nil {
		t.Fatalf("Get after TTL: %v", err)
	}
	if got != nil {
		t.Errorf("Get after TTL: want nil (expired), got %+v", got)
	}
}

func TestScopeCache_Invalidate(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	dag, _ := json.Marshal(map[string]interface{}{"nodes": []string{"ctrl-1"}})
	// Cache two versions.
	for _, ver := range []int{1, 2, 3} {
		entry := &cache.ScopeCacheEntry{EngagementID: "eng-3", Version: ver, DAG: dag, ComputedAt: time.Now().UTC()}
		if err := c.Scope.Set(ctx, "eng-3", ver, entry); err != nil {
			t.Fatalf("Set v%d: %v", ver, err)
		}
	}

	// Invalidate all versions of eng-3.
	if err := c.Scope.Invalidate(ctx, "eng-3"); err != nil {
		t.Fatalf("Invalidate: %v", err)
	}

	for _, ver := range []int{1, 2, 3} {
		got, err := c.Scope.Get(ctx, "eng-3", ver)
		if err != nil {
			t.Fatalf("Get v%d after Invalidate: %v", ver, err)
		}
		if got != nil {
			t.Errorf("Get v%d after Invalidate: want nil, got %+v", ver, got)
		}
	}
}

// ─── InvalidationHandler ─────────────────────────────────────────────────────

func TestInvalidationHandler_OnScopeChanged(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	dag, _ := json.Marshal(map[string]interface{}{"nodes": []string{"ctrl-1"}})
	entry := &cache.ScopeCacheEntry{EngagementID: "eng-4", Version: 1, DAG: dag, ComputedAt: time.Now().UTC()}
	if err := c.Scope.Set(ctx, "eng-4", 1, entry); err != nil {
		t.Fatalf("Set: %v", err)
	}

	if err := c.Invalidation.OnScopeChanged(ctx, "eng-4"); err != nil {
		t.Fatalf("OnScopeChanged: %v", err)
	}

	got, err := c.Scope.Get(ctx, "eng-4", 1)
	if err != nil {
		t.Fatalf("Get after invalidation: %v", err)
	}
	if got != nil {
		t.Errorf("Get after invalidation: want nil, got %+v", got)
	}
}

func TestInvalidationHandler_OnUserLogout(t *testing.T) {
	c, _ := newTestClient(t)
	ctx := context.Background()

	entry := &cache.SessionEntry{UserID: "user-logout", OrgID: "org-1", TokenID: "jti"}
	if err := c.Session.Set(ctx, "user-logout", entry); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := c.Invalidation.OnUserLogout(ctx, "user-logout"); err != nil {
		t.Fatalf("OnUserLogout: %v", err)
	}
	got, err := c.Session.Get(ctx, "user-logout")
	if err != nil {
		t.Fatalf("Get after logout: %v", err)
	}
	if got != nil {
		t.Errorf("Get after logout: want nil, got %+v", got)
	}
}

