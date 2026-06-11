// Package service holds business logic for the identity service.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/davejduke/obvious/services/identity/internal/models"
)

const refreshKeyPrefix = "refresh:"

// SessionStore manages refresh token storage in Redis.
type SessionStore struct {
	client *redis.Client
	ttl    time.Duration
}

// NewSessionStore creates a SessionStore backed by the given Redis client.
func NewSessionStore(client *redis.Client, refreshTTL time.Duration) *SessionStore {
	return &SessionStore{client: client, ttl: refreshTTL}
}

// StoreRefreshToken persists a RefreshToken under refresh:<jti>.
func (s *SessionStore) StoreRefreshToken(ctx context.Context, tok *models.RefreshToken) error {
	data, err := json.Marshal(tok)
	if err != nil {
		return fmt.Errorf("session: marshal refresh token: %w", err)
	}
	key := refreshKeyPrefix + tok.TokenID
	return s.client.Set(ctx, key, data, time.Until(tok.ExpiresAt)).Err()
}

// GetRefreshToken retrieves a RefreshToken by JTI. Returns nil, nil if not found.
func (s *SessionStore) GetRefreshToken(ctx context.Context, jti string) (*models.RefreshToken, error) {
	key := refreshKeyPrefix + jti
	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("session: get refresh token: %w", err)
	}
	var tok models.RefreshToken
	if err := json.Unmarshal(data, &tok); err != nil {
		return nil, fmt.Errorf("session: unmarshal refresh token: %w", err)
	}
	return &tok, nil
}

// RevokeRefreshToken deletes a refresh token from Redis (logout / rotation).
func (s *SessionStore) RevokeRefreshToken(ctx context.Context, jti string) error {
	key := refreshKeyPrefix + jti
	return s.client.Del(ctx, key).Err()
}
