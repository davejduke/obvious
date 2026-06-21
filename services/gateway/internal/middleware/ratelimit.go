package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter implements a sliding-window rate limiter backed by Redis.
//
// Algorithm: for each (key, window), a sorted set stores request timestamps.
// On every request:
//  1. Remove timestamps older than now - window
//  2. Count current members
//  3. If count < limit: ZADD the new timestamp and set TTL
//  4. If count >= limit: reject with 429
//
// An atomic Lua script executes steps 1-3 to avoid TOCTOU races.
type RateLimiter struct {
	redis    redis.Cmdable
	window   time.Duration // sliding window size
	standard int           // requests/window for standard tier
	enterprise int
	integration int
}

// limiterScript is an atomic Lua script that implements the sliding window check.
// KEYS[1] = Redis key  ARGV[1] = now (unix ms)  ARGV[2] = windowMs  ARGV[3] = limit
// Returns: {allowed: 0|1, remaining: int, resetMs: int}
const limiterScript = `
local key       = KEYS[1]
local now       = tonumber(ARGV[1])
local windowMs  = tonumber(ARGV[2])
local limit     = tonumber(ARGV[3])
local clearBefore = now - windowMs

-- Remove entries outside the window
redis.call('ZREMRANGEBYSCORE', key, '-inf', clearBefore)

-- Count current members
local count = redis.call('ZCARD', key)

if count < limit then
  -- Add this request
  redis.call('ZADD', key, now, now .. '-' .. math.random(1, 1000000))
  redis.call('PEXPIRE', key, windowMs)
  return {1, limit - count - 1, now + windowMs}
end

-- Rejected — find the oldest entry to calculate reset time
local oldest = redis.call('ZRANGE', key, 0, 0, 'WITHSCORES')
local resetMs = now + windowMs
if #oldest >= 2 then
  resetMs = tonumber(oldest[2]) + windowMs
end
return {0, 0, resetMs}
`

// NewRateLimiter creates a RateLimiter.
func NewRateLimiter(r redis.Cmdable, standard, enterprise, integration int) *RateLimiter {
	return &RateLimiter{
		redis:       r,
		window:      time.Minute,
		standard:    standard,
		enterprise:  enterprise,
		integration: integration,
	}
}

// limitFor returns the requests-per-minute limit for the given org tier.
func (rl *RateLimiter) limitFor(tier OrgTier) int {
	switch tier {
	case TierEnterprise:
		return rl.enterprise
	case TierIntegration:
		return rl.integration
	default:
		return rl.standard
	}
}

// RateLimit returns a Chi middleware that enforces per-org sliding-window rate limits.
func (rl *RateLimiter) RateLimit(next http.Handler) http.Handler {
	script := redis.NewScript(limiterScript)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())

		// Derive the rate limit key.
		// Authenticated: keyed by org id.  Unauthenticated: keyed by remote IP.
		var key string
		var tier OrgTier = TierStandard
		if claims != nil {
			key = "ratelimit:org:" + claims.OrgID
			tier = claims.OrgTier
			if tier == "" {
				tier = TierStandard
			}
		} else {
			key = "ratelimit:ip:" + remoteIP(r)
		}

		limit := rl.limitFor(tier)
		nowMs := time.Now().UnixMilli()
		windowMs := rl.window.Milliseconds()

		result, err := script.Run(
			context.Background(),
			rl.redis,
			[]string{key},
			nowMs,
			windowMs,
			limit,
		).Int64Slice()

		if err != nil {
			// Redis unavailable — fail open to avoid blocking all traffic.
			next.ServeHTTP(w, r)
			return
		}

		allowed := result[0] == 1
		remaining := result[1]
		resetMs := result[2]
		resetSec := resetMs / 1000

		w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetSec))

		if !allowed {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"code":"RATE_LIMITED","message":"too many requests"}`)) //nolint:errcheck
			return
		}

		next.ServeHTTP(w, r)
	})
}

// remoteIP extracts the client IP from the request.
func remoteIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first (leftmost) entry.
		for i, c := range xff {
			if c == ',' {
				return xff[:i]
			}
		}
		return xff
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Strip port from RemoteAddr.
	addr := r.RemoteAddr
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
