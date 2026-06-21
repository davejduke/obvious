// Package store provides PostgreSQL-backed persistence for webhook endpoints and deliveries.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/davejduke/obvious/services/webhook/internal/domain"
)

// Store wraps a connection pool and provides all persistence operations.
type Store struct {
	pool *pgxpool.Pool
}

// New returns a Store backed by the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// -----------------------------------------------------------------
// Endpoint CRUD
// -----------------------------------------------------------------

// CreateEndpoint inserts a new webhook endpoint and returns it.
func (s *Store) CreateEndpoint(ctx context.Context, ep *domain.Endpoint) (*domain.Endpoint, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO webhook_endpoints (org_id, url, secret, event_types, description, enabled)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, org_id, url, secret, event_types, description, enabled, created_at, updated_at`,
		ep.OrgID, ep.URL, ep.Secret, ep.EventTypes, ep.Description, ep.Enabled,
	)
	out := &domain.Endpoint{}
	err := row.Scan(
		&out.ID, &out.OrgID, &out.URL, &out.Secret, &out.EventTypes,
		&out.Description, &out.Enabled, &out.CreatedAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create endpoint: %w", err)
	}
	return out, nil
}

// ListEndpoints returns all enabled endpoints for an organisation.
func (s *Store) ListEndpoints(ctx context.Context, orgID string) ([]*domain.Endpoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, org_id, url, secret, event_types, description, enabled, created_at, updated_at
		FROM webhook_endpoints
		WHERE org_id = $1
		ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list endpoints: %w", err)
	}
	defer rows.Close()
	var out []*domain.Endpoint
	for rows.Next() {
		ep := &domain.Endpoint{}
		if err := rows.Scan(&ep.ID, &ep.OrgID, &ep.URL, &ep.Secret, &ep.EventTypes,
			&ep.Description, &ep.Enabled, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan endpoint: %w", err)
		}
		out = append(out, ep)
	}
	return out, rows.Err()
}

// GetEndpoint returns a single endpoint by ID.
func (s *Store) GetEndpoint(ctx context.Context, id, orgID string) (*domain.Endpoint, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, org_id, url, secret, event_types, description, enabled, created_at, updated_at
		FROM webhook_endpoints WHERE id = $1 AND org_id = $2`, id, orgID)
	ep := &domain.Endpoint{}
	err := row.Scan(&ep.ID, &ep.OrgID, &ep.URL, &ep.Secret, &ep.EventTypes,
		&ep.Description, &ep.Enabled, &ep.CreatedAt, &ep.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get endpoint: %w", err)
	}
	return ep, nil
}

// DeleteEndpoint removes a webhook endpoint.
func (s *Store) DeleteEndpoint(ctx context.Context, id, orgID string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM webhook_endpoints WHERE id = $1 AND org_id = $2`, id, orgID)
	if err != nil {
		return fmt.Errorf("delete endpoint: %w", err)
	}
	return nil
}

// EnabledEndpointsForEvent returns all enabled endpoints matching an event type
// (endpoints with event_types = ['*'] always match).
func (s *Store) EnabledEndpointsForEvent(ctx context.Context, orgID, eventType string) ([]*domain.Endpoint, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, org_id, url, secret, event_types, description, enabled, created_at, updated_at
		FROM webhook_endpoints
		WHERE org_id = $1 AND enabled = true
		  AND (event_types @> ARRAY['*'] OR event_types @> ARRAY[$2::text])`,
		orgID, eventType)
	if err != nil {
		return nil, fmt.Errorf("query endpoints for event: %w", err)
	}
	defer rows.Close()
	var out []*domain.Endpoint
	for rows.Next() {
		ep := &domain.Endpoint{}
		if err := rows.Scan(&ep.ID, &ep.OrgID, &ep.URL, &ep.Secret, &ep.EventTypes,
			&ep.Description, &ep.Enabled, &ep.CreatedAt, &ep.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan endpoint: %w", err)
		}
		out = append(out, ep)
	}
	return out, rows.Err()
}

// -----------------------------------------------------------------
// Delivery queue
// -----------------------------------------------------------------

// EnqueueDelivery creates a pending delivery record.
func (s *Store) EnqueueDelivery(ctx context.Context, endpointID, eventType string, payload []byte) (*domain.Delivery, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO webhook_deliveries (endpoint_id, event_type, payload, status, next_retry_at)
		VALUES ($1, $2, $3, 'pending', NOW())
		RETURNING id, endpoint_id, event_type, payload, status, attempts, max_attempts,
		          next_retry_at, delivered_at, dead_lettered_at,
		          last_response_code, last_response_body, created_at, updated_at`,
		endpointID, eventType, payload)
	return scanDelivery(row)
}

// DuePendingDeliveries returns at most limit deliveries whose next_retry_at <= now.
func (s *Store) DuePendingDeliveries(ctx context.Context, limit int) ([]*domain.Delivery, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, endpoint_id, event_type, payload, status, attempts, max_attempts,
		       next_retry_at, delivered_at, dead_lettered_at,
		       last_response_code, last_response_body, created_at, updated_at
		FROM webhook_deliveries
		WHERE status IN ('pending', 'failed') AND next_retry_at <= NOW()
		ORDER BY next_retry_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return nil, fmt.Errorf("query due deliveries: %w", err)
	}
	defer rows.Close()
	var out []*domain.Delivery
	for rows.Next() {
		d, err := scanDeliveryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// MarkDelivered records a successful delivery.
func (s *Store) MarkDelivered(ctx context.Context, id string, code int, body string) error {
	now := time.Now().UTC()
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = 'delivered', delivered_at = $2,
		    last_response_code = $3, last_response_body = $4,
		    attempts = attempts + 1
		WHERE id = $1`, id, now, code, truncate(body, 2000))
	return err
}

// MarkFailed records a failed attempt and schedules the next retry.
// If max_attempts is reached, the delivery is dead-lettered.
func (s *Store) MarkFailed(ctx context.Context, id string, code int, body string, attempts int) error {
	if attempts >= domain.MaxAttempts {
		now := time.Now().UTC()
		_, err := s.pool.Exec(ctx, `
			UPDATE webhook_deliveries
			SET status = 'dead_lettered', dead_lettered_at = $2,
			    last_response_code = $3, last_response_body = $4,
			    attempts = attempts + 1
			WHERE id = $1`, id, now, code, truncate(body, 2000))
		return err
	}
	delay := domain.RetryDelays[min(attempts, len(domain.RetryDelays)-1)]
	nextRetry := time.Now().UTC().Add(delay)
	_, err := s.pool.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = 'failed', next_retry_at = $2,
		    last_response_code = $3, last_response_body = $4,
		    attempts = attempts + 1
		WHERE id = $1`, id, nextRetry, code, truncate(body, 2000))
	return err
}

// ListDeliveries returns delivery history for an endpoint.
func (s *Store) ListDeliveries(ctx context.Context, endpointID string, limit, offset int) ([]*domain.Delivery, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, endpoint_id, event_type, payload, status, attempts, max_attempts,
		       next_retry_at, delivered_at, dead_lettered_at,
		       last_response_code, last_response_body, created_at, updated_at
		FROM webhook_deliveries
		WHERE endpoint_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`, endpointID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list deliveries: %w", err)
	}
	defer rows.Close()
	var out []*domain.Delivery
	for rows.Next() {
		d, err := scanDeliveryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// -----------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------

func scanDelivery(row pgx.Row) (*domain.Delivery, error) {
	d := &domain.Delivery{}
	var rawPayload []byte
	err := row.Scan(
		&d.ID, &d.EndpointID, &d.EventType, &rawPayload, &d.Status,
		&d.Attempts, &d.MaxAttempts, &d.NextRetryAt,
		&d.DeliveredAt, &d.DeadLetteredAt,
		&d.LastResponseCode, &d.LastResponseBody,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan delivery: %w", err)
	}
	d.Payload = rawPayload
	return d, nil
}

func scanDeliveryRow(rows pgx.Rows) (*domain.Delivery, error) {
	d := &domain.Delivery{}
	var rawPayload []byte
	err := rows.Scan(
		&d.ID, &d.EndpointID, &d.EventType, &rawPayload, &d.Status,
		&d.Attempts, &d.MaxAttempts, &d.NextRetryAt,
		&d.DeliveredAt, &d.DeadLetteredAt,
		&d.LastResponseCode, &d.LastResponseBody,
		&d.CreatedAt, &d.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan delivery row: %w", err)
	}
	d.Payload = rawPayload
	return d, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MemoryStore is an in-memory store for testing without a real DB.
type MemoryStore struct {
	Endpoints []*domain.Endpoint
	Deliveries []*domain.Delivery
}

// MarshalPayload marshals an event payload to JSON.
func MarshalPayload(v any) ([]byte, error) {
	return json.Marshal(v)
}

