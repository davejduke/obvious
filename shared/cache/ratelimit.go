package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// rateLimitKeyPrefix is the Redis key prefix for the ratelimit namespace.
	rateLimitKeyPrefix = "ratelimit:"
)

// RateLimitResult holds the outcome of a sliding-window rate-limit check.
type RateLimitResult struct {
	// Count is the number of requests in the current window.
	Count int64
	// Allowed is true if the request is within the configured limit.
	Allowed bool
	// RetryAfter is how long until the oldest request falls outside the window.
	// Only meaningful when Allowed == false.
	RetryAfter time.Duration
}

// RateLimitCache manages the ratelimit:{org_id}:{endpoint} namespace.
// The sliding window is implemented with a Redis sorted set:
//   - Members are unique request IDs (nanosecond timestamps).
//   - Scores are Unix timestamps in milliseconds.
//   - Expired members (outside the window) are pruned on every Increment.
type RateLimitCache struct {
	rdb *redis.Client
}

// key returns the sorted-set key for an org+endpoint pair.
func (r *RateLimitCache) key(orgID, endpoint string) string {
	return fmt.Sprintf("%s%s:%s", rateLimitKeyPrefix, orgID, endpoint)
}

// Increment records a new request in the sliding window and returns whether it
// is allowed under limit.
//
//	- windowSize: length of the sliding window (e.g. 1*time.Minute)
//	- limit:      maximum requests allowed in the window
func (r *RateLimitCache) Increment(ctx context.Context, orgID, endpoint string, windowSize time.Duration, limit int64) (*RateLimitResult, error) {
	now := time.Now()
	nowMs := float64(now.UnixMilli())
	windowStartMs := float64(now.Add(-windowSize).UnixMilli())
	expireSeconds := int(windowSize.Seconds()) + 1

	key := r.key(orgID, endpoint)

	// Use a pipeline to atomically: prune expired entries, add new entry, count.
	pipe := r.rdb.Pipeline()
	// Prune members older than the window.
	pipe.ZRemRangeByScore(ctx, key, "-inf", fmt.Sprintf("%.0f", windowStartMs))
	// Add the current request. Use nanosecond timestamp as unique member.
	member := fmt.Sprintf("%d", now.UnixNano())
	pipe.ZAdd(ctx, key, redis.Z{Score: nowMs, Member: member})
	// Count members remaining in the window.
	countCmd := pipe.ZCard(ctx, key)
	// Set key expiry to auto-clean idle keys.
	pipe.Expire(ctx, key, time.Duration(expireSeconds)*time.Second)

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("ratelimit cache: pipeline for %s/%s: %w", orgID, endpoint, err)
	}

	count := countCmd.Val()
	allowed := count <= limit

	var retryAfter time.Duration
	if !allowed {
		// Find the oldest entry to calculate when the window clears.
		oldest, err := r.rdb.ZRangeByScoreWithScores(ctx, key, &redis.ZRangeBy{
			Min:   "-inf",
			Max:   "+inf",
			Count: 1,
		}).Result()
		if err == nil && len(oldest) > 0 {
			oldestMs := int64(oldest[0].Score)
			oldestTime := time.UnixMilli(oldestMs)
			resetsAt := oldestTime.Add(windowSize)
			if resetsAt.After(now) {
				retryAfter = resetsAt.Sub(now)
			}
		}
	}

	return &RateLimitResult{
		Count:      count,
		Allowed:    allowed,
		RetryAfter: retryAfter,
	}, nil
}

// Count returns the number of requests in the current sliding window without
// recording a new request.
func (r *RateLimitCache) Count(ctx context.Context, orgID, endpoint string, windowSize time.Duration) (int64, error) {
	windowStartMs := float64(time.Now().Add(-windowSize).UnixMilli())
	key := r.key(orgID, endpoint)

	count, err := r.rdb.ZCount(ctx, key,
		fmt.Sprintf("%.0f", windowStartMs), "+inf").Result()
	if err != nil {
		return 0, fmt.Errorf("ratelimit cache: count %s/%s: %w", orgID, endpoint, err)
	}
	return count, nil
}

// Reset removes all entries for an org+endpoint pair, effectively clearing the
// rate limit counter (useful for testing or manual override).
func (r *RateLimitCache) Reset(ctx context.Context, orgID, endpoint string) error {
	if err := r.rdb.Del(ctx, r.key(orgID, endpoint)).Err(); err != nil {
		return fmt.Errorf("ratelimit cache: reset %s/%s: %w", orgID, endpoint, err)
	}
	return nil
}

