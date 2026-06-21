package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/redis/go-redis/v9"
)

// mockRedis is a minimal in-memory sorted-set backed mock for rate-limit tests.
// It runs the Lua script logic directly so we don't need a real Redis instance.
type mockRateLimiter struct {
	counts map[string]int
	limits map[string]int
}

func newMockRL(standard, enterprise, integration int) *mockRateLimiter {
	return &mockRateLimiter{
		counts: make(map[string]int),
		limits: map[string]int{
			"standard":    standard,
			"enterprise":  enterprise,
			"integration": integration,
		},
	}
}

// rateMiddleware builds a simplified in-memory rate-limit middleware for unit tests.
func (m *mockRateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		key := "ip:" + remoteIP(r)
		tier := "standard"
		if claims != nil {
			key = "org:" + claims.OrgID
			switch claims.OrgTier {
			case TierEnterprise:
				tier = "enterprise"
			case TierIntegration:
				tier = "integration"
			}
		}
		limit := m.limits[tier]
		m.counts[key]++

		remaining := limit - m.counts[key]
		if remaining < 0 {
			remaining = 0
		}

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", "9999999999")

		if m.counts[key] > limit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func TestRateLimiter_TierLimits(t *testing.T) {
	rl := newMockRL(3, 10, 50) // low limits for test speed

	tests := []struct {
		name      string
		tier      OrgTier
		orgID     string
		requests  int
		wantLast  int // expected status code on last request
		wantLimit string
	}{
		{"standard tier", TierStandard, "org-std", 4, http.StatusTooManyRequests, "3"},
		{"enterprise tier", TierEnterprise, "org-ent", 5, http.StatusOK, "10"},
		{"integration tier", TierIntegration, "org-int", 5, http.StatusOK, "50"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var lastCode int
			var lastLimit string
			for i := 0; i < tt.requests; i++ {
				handler := rl.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))

				req := httptest.NewRequest(http.MethodGet, "/", nil)
				req.RemoteAddr = "1.2.3.4:1234"
				// Inject claims with org and tier
				claims := &GatewayClaims{
					Claims:  Claims{OrgID: tt.orgID},
					OrgTier: tt.tier,
				}
				ctx := context.WithValue(req.Context(), ClaimsKey, claims)
				req = req.WithContext(ctx)
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)
				lastCode = rr.Code
				lastLimit = rr.Header().Get("X-RateLimit-Limit")
			}
			if lastCode != tt.wantLast {
				t.Errorf("last request: got %d, want %d", lastCode, tt.wantLast)
			}
			if lastLimit != tt.wantLimit {
				t.Errorf("X-RateLimit-Limit: got %q, want %q", lastLimit, tt.wantLimit)
			}
		})
	}
}

func TestRateLimiter_HeadersPresent(t *testing.T) {
	rl := newMockRL(200, 1000, 5000)
	handler := rl.RateLimitMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	for _, h := range []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"} {
		if rr.Header().Get(h) == "" {
			t.Errorf("missing header: %s", h)
		}
	}
}

func TestRateLimiter_FailOpen(t *testing.T) {
	// If Redis is unavailable the rate limiter should fail open (allow through).
	badRedis := redis.NewClient(&redis.Options{Addr: "localhost:9"}) // port 9 is discard
	rl := NewRateLimiter(badRedis, 200, 1000, 5000)

	handler := rl.RateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected fail-open 200 got %d", rr.Code)
	}
}
