// Package store provides database access for the webhooks service.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/davejduke/obvious/services/webhooks/internal/model"
)

// Store wraps a PostgreSQL connection pool and exposes all webhook operations.
type Store struct {
	db *pgxpool.Pool
}

// New returns a Store backed by the supplied pool.
func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// ─────────────────────────────────────────────────────────────
// SUBSCRIPTION CRUD
// ─────────────────────────────────────────────────────────────

// CreateSubscription inserts a new webhook subscription and returns the
// persisted record.
func (s *Store) CreateSubscription(
	ctx context.Context,
	orgID uuid.UUID,
	url, secretValue, secretHash string,
	eventTypes []string,
	description string,
	createdBy *uuid.UUID,
) (*model.WebhookSubscription, error) {
	var sub model.WebhookSubscription

	row := s.db.QueryRow(ctx, `
		INSERT INTO webhook_subscriptions
			(org_id, url, secret_value, secret_hash, event_types, description, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, org_id, url, event_types, description, is_active,
		          COALESCE(created_by, '00000000-0000-0000-0000-000000000000'::uuid),
		          created_at, updated_at`,
		orgID, url, secretValue, secretHash, eventTypes, description, createdBy,
	)
	if err := row.Scan(
		&sub.ID, &sub.OrgID, &sub.URL, &sub.EventTypes,
		&sub.Description, &sub.IsActive, &sub.CreatedBy,
		&sub.CreatedAt, &sub.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("store: create subscription: %w", err)
	}
	return &sub, nil
}

// GetSubscription retrieves a single subscription by ID, scoped to orgID.
func (s *Store) GetSubscription(ctx context.Context, orgID, id uuid.UUID) (*model.WebhookSubscription, error) {
	var sub model.WebhookSubscription

	row := s.db.QueryRow(ctx, `
		SELECT id, org_id, url, event_types, description, is_active,
		       COALESCE(created_by, '00000000-0000-0000-0000-000000000000'::uuid),
		       created_at, updated_at
		FROM webhook_subscriptions
		WHERE id = $1 AND org_id = $2`,
		id, orgID,
	)
	if err := row.Scan(
		&sub.ID, &sub.OrgID, &sub.URL, &sub.EventTypes,
		&sub.Description, &sub.IsActive, &sub.CreatedBy,
		&sub.CreatedAt, &sub.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // caller checks for nil
		}
		return nil, fmt.Errorf("store: get subscription: %w", err)
	}
	return &sub, nil
}

// GetSubscriptionWithSecret retrieves a subscription including its secret_value.
// Used internally by the delivery worker for HMAC signing.
func (s *Store) GetSubscriptionWithSecret(ctx context.Context, id uuid.UUID) (sub *model.WebhookSubscription, secret string, err error) {
	sub = &model.WebhookSubscription{}
	row := s.db.QueryRow(ctx, `
		SELECT id, org_id, url, event_types, description, is_active,
		       COALESCE(created_by, '00000000-0000-0000-0000-000000000000'::uuid),
		       created_at, updated_at, secret_value
		FROM webhook_subscriptions
		WHERE id = $1 AND is_active = true`,
		id,
	)
	if scanErr := row.Scan(
		&sub.ID, &sub.OrgID, &sub.URL, &sub.EventTypes,
		&sub.Description, &sub.IsActive, &sub.CreatedBy,
		&sub.CreatedAt, &sub.UpdatedAt, &secret,
	); scanErr != nil {
		if scanErr == pgx.ErrNoRows {
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("store: get subscription with secret: %w", scanErr)
	}
	return sub, secret, nil
}

// ListSubscriptions returns all subscriptions for an org, newest-first.
func (s *Store) ListSubscriptions(ctx context.Context, orgID uuid.UUID) ([]*model.WebhookSubscription, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, org_id, url, event_types, description, is_active,
		       COALESCE(created_by, '00000000-0000-0000-0000-000000000000'::uuid),
		       created_at, updated_at
		FROM webhook_subscriptions
		WHERE org_id = $1
		ORDER BY created_at DESC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list subscriptions: %w", err)
	}
	defer rows.Close()

	var subs []*model.WebhookSubscription
	for rows.Next() {
		var sub model.WebhookSubscription
		if err := rows.Scan(
			&sub.ID, &sub.OrgID, &sub.URL, &sub.EventTypes,
			&sub.Description, &sub.IsActive, &sub.CreatedBy,
			&sub.CreatedAt, &sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("store: scan subscription: %w", err)
		}
		subs = append(subs, &sub)
	}
	return subs, rows.Err()
}

// UpdateSubscription applies partial updates to a subscription.
func (s *Store) UpdateSubscription(
	ctx context.Context,
	orgID, id uuid.UUID,
	req *model.UpdateSubscriptionRequest,
) (*model.WebhookSubscription, error) {
	// Build dynamic SET clause — only fields provided in the request.
	args := []any{orgID, id}
	i := 3
	set := ""

	if req.URL != nil {
		set += fmt.Sprintf(", url = $%d", i)
		args = append(args, *req.URL)
		i++
	}
	if req.Description != nil {
		set += fmt.Sprintf(", description = $%d", i)
		args = append(args, *req.Description)
		i++
	}
	if req.IsActive != nil {
		set += fmt.Sprintf(", is_active = $%d", i)
		args = append(args, *req.IsActive)
		i++
	}
	if len(req.EventTypes) > 0 {
		set += fmt.Sprintf(", event_types = $%d", i)
		args = append(args, req.EventTypes)
		i++
	}

	if set == "" {
		// Nothing to update — return current row.
		return s.GetSubscription(ctx, orgID, id)
	}
	// Strip leading comma.
	set = set[2:]

	var sub model.WebhookSubscription
	row := s.db.QueryRow(ctx, fmt.Sprintf(`
		UPDATE webhook_subscriptions
		SET %s
		WHERE org_id = $1 AND id = $2
		RETURNING id, org_id, url, event_types, description, is_active,
		          COALESCE(created_by, '00000000-0000-0000-0000-000000000000'::uuid),
		          created_at, updated_at`, set),
		args...,
	)
	if err := row.Scan(
		&sub.ID, &sub.OrgID, &sub.URL, &sub.EventTypes,
		&sub.Description, &sub.IsActive, &sub.CreatedBy,
		&sub.CreatedAt, &sub.UpdatedAt,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: update subscription: %w", err)
	}
	return &sub, nil
}

// DeleteSubscription soft-deletes by setting is_active = false, scoped to orgID.
func (s *Store) DeleteSubscription(ctx context.Context, orgID, id uuid.UUID) error {
	_, err := s.db.Exec(ctx, `
		UPDATE webhook_subscriptions
		SET is_active = false
		WHERE org_id = $1 AND id = $2`,
		orgID, id,
	)
	if err != nil {
		return fmt.Errorf("store: delete subscription: %w", err)
	}
	return nil
}

// ActiveSub is a lightweight subscription record used for delivery fan-out.
type ActiveSub struct {
	ID     uuid.UUID
	URL    string
	Secret string
}

// ActiveSubscriptionsForEvent returns all active subscriptions for an org that
// have opted in to the given event type.
func (s *Store) ActiveSubscriptionsForEvent(
	ctx context.Context,
	orgID uuid.UUID,
	eventType string,
) ([]*ActiveSub, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, url, secret_value
		FROM webhook_subscriptions
		WHERE org_id = $1
		  AND is_active = true
		  AND ($2 = ANY(event_types) OR 'wildcard' = ANY(event_types))`,
		orgID, eventType,
	)
	if err != nil {
		return nil, fmt.Errorf("store: active subscriptions: %w", err)
	}
	defer rows.Close()

	var result []*ActiveSub
	for rows.Next() {
		var r ActiveSub
		if err := rows.Scan(&r.ID, &r.URL, &r.Secret); err != nil {
			return nil, fmt.Errorf("store: scan active subscription: %w", err)
		}
		result = append(result, &r)
	}
	return result, rows.Err()
}

// ─────────────────────────────────────────────────────────────
// DELIVERY TRACKING
// ─────────────────────────────────────────────────────────────

// CreateDelivery inserts a new delivery record and returns it.
func (s *Store) CreateDelivery(
	ctx context.Context,
	subID, orgID uuid.UUID,
	eventType string,
	payload map[string]any,
) (*model.WebhookDelivery, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("store: marshal delivery payload: %w", err)
	}

	var d model.WebhookDelivery
	row := s.db.QueryRow(ctx, `
		INSERT INTO webhook_deliveries
			(subscription_id, org_id, event_type, payload, status)
		VALUES ($1, $2, $3, $4, 'pending')
		RETURNING id, subscription_id, org_id, event_type, payload,
		          status, attempt_count, last_attempt_at, next_retry_at,
		          response_status, error_message, delivered_at, created_at`,
		subID, orgID, eventType, payloadBytes,
	)
	if err := scanDelivery(row, &d); err != nil {
		return nil, fmt.Errorf("store: create delivery: %w", err)
	}
	return &d, nil
}

// MarkDelivered marks a delivery as successfully delivered.
func (s *Store) MarkDelivered(ctx context.Context, id uuid.UUID, responseStatus int) error {
	now := time.Now().UTC()
	_, err := s.db.Exec(ctx, `
		UPDATE webhook_deliveries
		SET status = 'delivered',
		    response_status = $2,
		    delivered_at = $3,
		    last_attempt_at = $3
		WHERE id = $1`,
		id, responseStatus, now,
	)
	if err != nil {
		return fmt.Errorf("store: mark delivered: %w", err)
	}
	return nil
}

// RecordAttemptFailure increments the attempt counter and, if retries remain,
// schedules the next retry at the appropriate backoff interval. Once
// MaxAttempts is reached the delivery is moved to 'failed' (dead-letter).
func (s *Store) RecordAttemptFailure(
	ctx context.Context,
	id uuid.UUID,
	attemptCount int,
	responseStatus *int,
	errMsg string,
) error {
	now := time.Now().UTC()

	var nextRetry *time.Time
	newStatus := string(model.DeliveryStatusFailed)

	if attemptCount < model.MaxAttempts {
		newStatus = string(model.DeliveryStatusPending)
		next := now.Add(model.RetrySchedule[attemptCount])
		nextRetry = &next
	}

	_, err := s.db.Exec(ctx, `
		UPDATE webhook_deliveries
		SET attempt_count = $2,
		    last_attempt_at = $3,
		    next_retry_at = $4,
		    response_status = $5,
		    error_message = $6,
		    status = $7
		WHERE id = $1`,
		id, attemptCount, now, nextRetry, responseStatus, errMsg, newStatus,
	)
	if err != nil {
		return fmt.Errorf("store: record failure: %w", err)
	}
	return nil
}

// PendingDeliveries returns deliveries ready for (re)processing.
func (s *Store) PendingDeliveries(ctx context.Context, limit int) ([]*model.WebhookDelivery, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, subscription_id, org_id, event_type, payload,
		       status, attempt_count, last_attempt_at, next_retry_at,
		       response_status, error_message, delivered_at, created_at
		FROM webhook_deliveries
		WHERE status = 'pending'
		  AND (next_retry_at IS NULL OR next_retry_at <= NOW())
		ORDER BY created_at ASC
		LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("store: pending deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		if err := scanDelivery(rows, &d); err != nil {
			return nil, fmt.Errorf("store: scan delivery: %w", err)
		}
		deliveries = append(deliveries, &d)
	}
	return deliveries, rows.Err()
}

// ListDeliveriesForSubscription returns delivery history for a subscription.
func (s *Store) ListDeliveriesForSubscription(
	ctx context.Context,
	subID uuid.UUID,
	limit, offset int,
) ([]*model.WebhookDelivery, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, subscription_id, org_id, event_type, payload,
		       status, attempt_count, last_attempt_at, next_retry_at,
		       response_status, error_message, delivered_at, created_at
		FROM webhook_deliveries
		WHERE subscription_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`,
		subID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("store: list deliveries: %w", err)
	}
	defer rows.Close()

	var deliveries []*model.WebhookDelivery
	for rows.Next() {
		var d model.WebhookDelivery
		if err := scanDelivery(rows, &d); err != nil {
			return nil, fmt.Errorf("store: scan delivery: %w", err)
		}
		deliveries = append(deliveries, &d)
	}
	return deliveries, rows.Err()
}

// HealthStats returns aggregate delivery stats for all subscriptions of an org.
func (s *Store) HealthStats(ctx context.Context, orgID uuid.UUID) (map[string]any, error) {
	row := s.db.QueryRow(ctx, `
		SELECT
		    COUNT(*) FILTER (WHERE status = 'pending')   AS pending,
		    COUNT(*) FILTER (WHERE status = 'delivered') AS delivered,
		    COUNT(*) FILTER (WHERE status = 'failed')    AS failed,
		    COUNT(DISTINCT ws.id)                         AS active_subscriptions
		FROM webhook_subscriptions ws
		LEFT JOIN webhook_deliveries wd ON wd.subscription_id = ws.id
		WHERE ws.org_id = $1 AND ws.is_active = true`,
		orgID,
	)

	var pending, delivered, failed, activeSubs int64
	if err := row.Scan(&pending, &delivered, &failed, &activeSubs); err != nil {
		return nil, fmt.Errorf("store: health stats: %w", err)
	}
	return map[string]any{
		"pending":              pending,
		"delivered":            delivered,
		"failed":               failed,
		"active_subscriptions": activeSubs,
	}, nil
}

// ─────────────────────────────────────────────────────────────
// Scanner helpers
// ─────────────────────────────────────────────────────────────

type scanner interface {
	Scan(dest ...any) error
}

func scanDelivery(row scanner, d *model.WebhookDelivery) error {
	var payloadBytes []byte
	if err := row.Scan(
		&d.ID, &d.SubscriptionID, &d.OrgID, &d.EventType, &payloadBytes,
		&d.Status, &d.AttemptCount, &d.LastAttemptAt, &d.NextRetryAt,
		&d.ResponseStatus, &d.ErrorMessage, &d.DeliveredAt, &d.CreatedAt,
	); err != nil {
		return err
	}
	if len(payloadBytes) > 0 {
		if err := json.Unmarshal(payloadBytes, &d.Payload); err != nil {
			return fmt.Errorf("unmarshal payload: %w", err)
		}
	}
	return nil
}

