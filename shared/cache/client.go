// Package cache provides a Redis-backed caching layer for the AIAUDITOR platform.
// It implements three namespaces (session, ratelimit, scope) with connection pooling.
package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Config holds Redis connection and pooling configuration.
type Config struct {
	// URL is a Redis URL, e.g. "redis://localhost:6379" or "rediss://..."
	URL string
	// MaxRetries for failed commands (default 3).
	MaxRetries int
	// PoolSize is the maximum number of socket connections (default 10).
	PoolSize int
	// MinIdleConns is the minimum number of idle connections (default 2).
	MinIdleConns int
	// DialTimeout for establishing new connections (default 5s).
	DialTimeout time.Duration
	// ReadTimeout for socket reads (default 3s).
	ReadTimeout time.Duration
	// WriteTimeout for socket writes (default 3s).
	WriteTimeout time.Duration
}

// DefaultConfig returns a Config with sensible pooling defaults.
func DefaultConfig(url string) Config {
	return Config{
		URL:          url,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 2,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}
}

// Client wraps a go-redis client and exposes namespace-scoped helpers.
type Client struct {
	rdb *redis.Client
	// Session handles the session:{user_id} namespace.
	Session *SessionCache
	// RateLimit handles the ratelimit:{org_id}:{endpoint} namespace.
	RateLimit *RateLimitCache
	// Scope handles the scope:{engagement_id}:{version} namespace.
	Scope *ScopeCache
	// Invalidation provides event-driven cache invalidation helpers.
	Invalidation *InvalidationHandler
}

// New creates a Client from cfg, dials Redis, and wires up all namespace helpers.
// The caller must call Close() when done.
func New(cfg Config) (*Client, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("cache: parse url: %w", err)
	}

	opts.MaxRetries = cfg.MaxRetries
	opts.PoolSize = cfg.PoolSize
	opts.MinIdleConns = cfg.MinIdleConns
	opts.DialTimeout = cfg.DialTimeout
	opts.ReadTimeout = cfg.ReadTimeout
	opts.WriteTimeout = cfg.WriteTimeout

	rdb := redis.NewClient(opts)

	c := &Client{rdb: rdb}
	c.Session = &SessionCache{rdb: rdb}
	c.RateLimit = &RateLimitCache{rdb: rdb}
	c.Scope = &ScopeCache{rdb: rdb}
	c.Invalidation = &InvalidationHandler{session: c.Session, scope: c.Scope}

	return c, nil
}

// NewFromRaw creates a Client from an existing *redis.Client.
// Useful in services that manage their own connection (e.g. identity service).
func NewFromRaw(rdb *redis.Client) *Client {
	c := &Client{rdb: rdb}
	c.Session = &SessionCache{rdb: rdb}
	c.RateLimit = &RateLimitCache{rdb: rdb}
	c.Scope = &ScopeCache{rdb: rdb}
	c.Invalidation = &InvalidationHandler{session: c.Session, scope: c.Scope}
	return c
}

// Ping verifies the Redis connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

// Close releases the underlying Redis connection pool.
func (c *Client) Close() error {
	return c.rdb.Close()
}

// Raw returns the underlying *redis.Client for advanced / escape-hatch usage.
func (c *Client) Raw() *redis.Client {
	return c.rdb
}

