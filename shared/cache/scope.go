package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// ScopeTTL is the time-to-live for compiled scope DAG entries (30 minutes).
	ScopeTTL = 30 * time.Minute
	// scopeKeyPrefix is the Redis key prefix for the scope namespace.
	scopeKeyPrefix = "scope:"
)

// ScopeCacheEntry stores a compiled scope DAG for an engagement.
// Key: scope:{engagement_id}:{version}
type ScopeCacheEntry struct {
	EngagementID string          `json:"engagement_id"`
	Version      int             `json:"version"`
	// DAG is the serialised scope dependency graph produced by the engine.
	DAG          json.RawMessage `json:"dag"`
	ComputedAt   time.Time       `json:"computed_at"`
}

// ScopeCache manages the scope:{engagement_id}:{version} namespace.
// Entries are automatically invalidated on scope change events via Invalidate().
type ScopeCache struct {
	rdb *redis.Client
}

// key returns the Redis key for a specific engagement version.
func (s *ScopeCache) key(engagementID string, version int) string {
	return fmt.Sprintf("%s%s:%d", scopeKeyPrefix, engagementID, version)
}

// patternForEngagement returns a glob pattern matching all versions of an engagement.
func (s *ScopeCache) patternForEngagement(engagementID string) string {
	return fmt.Sprintf("%s%s:*", scopeKeyPrefix, engagementID)
}

// Set stores a compiled scope DAG with ScopeTTL.
func (s *ScopeCache) Set(ctx context.Context, engagementID string, version int, entry *ScopeCacheEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("scope cache: marshal entry: %w", err)
	}
	if err := s.rdb.Set(ctx, s.key(engagementID, version), data, ScopeTTL).Err(); err != nil {
		return fmt.Errorf("scope cache: set %s@%d: %w", engagementID, version, err)
	}
	return nil
}

// Get retrieves a compiled scope DAG by engagement ID and version.
// Returns nil, nil when the key does not exist (TTL expired or not cached).
func (s *ScopeCache) Get(ctx context.Context, engagementID string, version int) (*ScopeCacheEntry, error) {
	data, err := s.rdb.Get(ctx, s.key(engagementID, version)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scope cache: get %s@%d: %w", engagementID, version, err)
	}
	var entry ScopeCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("scope cache: unmarshal entry: %w", err)
	}
	return &entry, nil
}

// Invalidate deletes ALL cached scope DAGs for an engagement (all versions).
// This is the primary event-driven invalidation path: called when scope
// configuration changes and any cached DAG for the engagement is stale.
//
// Implementation uses SCAN + DEL to avoid blocking the Redis server with KEYS.
func (s *ScopeCache) Invalidate(ctx context.Context, engagementID string) error {
	pattern := s.patternForEngagement(engagementID)
	var cursor uint64
	var deleted int64

	for {
		keys, nextCursor, err := s.rdb.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("scope cache: scan for invalidation %s: %w", engagementID, err)
		}
		if len(keys) > 0 {
			n, err := s.rdb.Del(ctx, keys...).Result()
			if err != nil {
				return fmt.Errorf("scope cache: delete keys for %s: %w", engagementID, err)
			}
			deleted += n
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	_ = deleted // count available for metrics/logging if needed
	return nil
}

// InvalidateVersion deletes the cache entry for a specific engagement version.
func (s *ScopeCache) InvalidateVersion(ctx context.Context, engagementID string, version int) error {
	if err := s.rdb.Del(ctx, s.key(engagementID, version)).Err(); err != nil {
		return fmt.Errorf("scope cache: invalidate version %s@%d: %w", engagementID, version, err)
	}
	return nil
}

// TTL returns the remaining time-to-live for a specific scope entry.
// Returns -1 if the key does not have a TTL; -2 if the key does not exist.
func (s *ScopeCache) TTL(ctx context.Context, engagementID string, version int) (time.Duration, error) {
	ttl, err := s.rdb.TTL(ctx, s.key(engagementID, version)).Result()
	if err != nil {
		return 0, fmt.Errorf("scope cache: ttl %s@%d: %w", engagementID, version, err)
	}
	return ttl, nil
}

// ParseScopeKey extracts the engagement_id and version from a scope cache key.
// Inverse of key(). Returns an error if the key format is unexpected.
func ParseScopeKey(k string) (engagementID string, version string, err error) {
	withoutPrefix := strings.TrimPrefix(k, scopeKeyPrefix)
	parts := strings.SplitN(withoutPrefix, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("scope cache: unexpected key format: %q", k)
	}
	return parts[0], parts[1], nil
}

