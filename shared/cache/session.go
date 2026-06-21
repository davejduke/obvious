package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// SessionTTL is the time-to-live for session cache entries (8 hours).
	SessionTTL = 8 * time.Hour
	// sessionKeyPrefix is the Redis key prefix for the session namespace.
	sessionKeyPrefix = "session:"
)

// SessionEntry stores the data cached for an active user session.
// Key: session:{user_id}
type SessionEntry struct {
	UserID    string    `json:"user_id"`
	OrgID     string    `json:"org_id"`
	TokenID   string    `json:"token_id"` // current valid refresh token JTI
	Roles     []string  `json:"roles"`
	IssuedAt  time.Time `json:"issued_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// SessionCache manages the session:{user_id} namespace.
// Each entry stores the user's active session metadata with an 8-hour TTL.
type SessionCache struct {
	rdb *redis.Client
}

// key returns the Redis key for a given user ID.
func (s *SessionCache) key(userID string) string {
	return sessionKeyPrefix + userID
}

// Set stores a SessionEntry for userID with SessionTTL.
func (s *SessionCache) Set(ctx context.Context, userID string, entry *SessionEntry) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("session cache: marshal entry: %w", err)
	}
	if err := s.rdb.Set(ctx, s.key(userID), data, SessionTTL).Err(); err != nil {
		return fmt.Errorf("session cache: set %s: %w", userID, err)
	}
	return nil
}

// Get retrieves the SessionEntry for userID.
// Returns nil, nil when the key does not exist (expired or never set).
func (s *SessionCache) Get(ctx context.Context, userID string) (*SessionEntry, error) {
	data, err := s.rdb.Get(ctx, s.key(userID)).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("session cache: get %s: %w", userID, err)
	}
	var entry SessionEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("session cache: unmarshal entry: %w", err)
	}
	return &entry, nil
}

// Delete removes the session entry for userID (e.g. on logout).
func (s *SessionCache) Delete(ctx context.Context, userID string) error {
	if err := s.rdb.Del(ctx, s.key(userID)).Err(); err != nil {
		return fmt.Errorf("session cache: delete %s: %w", userID, err)
	}
	return nil
}

// Refresh resets the TTL on an existing session entry to SessionTTL.
// Returns false if the key does not exist.
func (s *SessionCache) Refresh(ctx context.Context, userID string) (bool, error) {
	ok, err := s.rdb.Expire(ctx, s.key(userID), SessionTTL).Result()
	if err != nil {
		return false, fmt.Errorf("session cache: refresh ttl %s: %w", userID, err)
	}
	return ok, nil
}

