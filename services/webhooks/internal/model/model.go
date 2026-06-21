// Package model defines domain types for the webhooks service.
package model

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents a publishable AIAUDITOR platform event.
type EventType string

const (
	EventTypeEvidenceIntakeComplete EventType = "evidence.intake.complete"
	EventTypeReasoningConclusion    EventType = "reasoning.conclusion"
	EventTypeFindingStatusChanged   EventType = "finding.status.changed"
)

// AllEventTypes lists every valid event type for subscription validation.
var AllEventTypes = []EventType{
	EventTypeEvidenceIntakeComplete,
	EventTypeReasoningConclusion,
	EventTypeFindingStatusChanged,
}

// DeliveryStatus tracks the lifecycle of a single webhook delivery attempt.
type DeliveryStatus string

const (
	DeliveryStatusPending   DeliveryStatus = "pending"
	DeliveryStatusDelivered DeliveryStatus = "delivered"
	DeliveryStatusFailed    DeliveryStatus = "failed"
)

// RetrySchedule defines the exponential backoff intervals (§4.5):
// attempt 1 → 30 s, attempt 2 → 5 min, attempt 3 → 30 min.
var RetrySchedule = []time.Duration{
	30 * time.Second,
	5 * time.Minute,
	30 * time.Minute,
}

// MaxAttempts is the total number of delivery attempts (initial + 3 retries).
const MaxAttempts = 3

// WebhookSubscription represents a per-organisation webhook endpoint.
type WebhookSubscription struct {
	ID          uuid.UUID `json:"id"`
	OrgID       uuid.UUID `json:"org_id"`
	URL         string    `json:"url"`
	EventTypes  []string  `json:"event_types"`
	Description string    `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedBy   uuid.UUID `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// SecretValue is omitted from JSON responses — never expose the signing secret.
}

// WebhookDelivery is the delivery record written per dispatch attempt.
type WebhookDelivery struct {
	ID             uuid.UUID      `json:"id"`
	SubscriptionID uuid.UUID      `json:"subscription_id"`
	OrgID          uuid.UUID      `json:"org_id"`
	EventType      string         `json:"event_type"`
	Payload        map[string]any `json:"payload"`
	Status         DeliveryStatus `json:"status"`
	AttemptCount   int            `json:"attempt_count"`
	LastAttemptAt  *time.Time     `json:"last_attempt_at,omitempty"`
	NextRetryAt    *time.Time     `json:"next_retry_at,omitempty"`
	ResponseStatus *int           `json:"response_status,omitempty"`
	ErrorMessage   string         `json:"error_message,omitempty"`
	DeliveredAt    *time.Time     `json:"delivered_at,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// WebhookEvent is the canonical envelope dispatched to subscribers.
type WebhookEvent struct {
	ID        string         `json:"id"`          // UUID for idempotency
	Type      string         `json:"type"`        // EventType string
	OrgID     string         `json:"org_id"`
	Timestamp time.Time      `json:"timestamp"`
	Data      map[string]any `json:"data"`
}

// CreateSubscriptionRequest is the body for POST /webhooks.
type CreateSubscriptionRequest struct {
	URL         string   `json:"url"`
	Secret      string   `json:"secret"`       // caller supplies; stored for signing
	EventTypes  []string `json:"event_types"`
	Description string   `json:"description,omitempty"`
}

// UpdateSubscriptionRequest is the body for PUT /webhooks/{id}.
type UpdateSubscriptionRequest struct {
	URL         *string  `json:"url,omitempty"`
	EventTypes  []string `json:"event_types,omitempty"`
	Description *string  `json:"description,omitempty"`
	IsActive    *bool    `json:"is_active,omitempty"`
}

// DispatchRequest is the internal body for POST /webhooks/dispatch.
type DispatchRequest struct {
	OrgID     uuid.UUID      `json:"org_id"`
	EventType string         `json:"event_type"`
	Data      map[string]any `json:"data"`
}

