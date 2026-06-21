// Package domain defines core types for the webhook delivery system.
package domain

import (
	"time"
)

// Status values for webhook deliveries.
const (
	StatusPending     = "pending"
	StatusDelivering  = "delivering"
	StatusDelivered   = "delivered"
	StatusFailed      = "failed"
	StatusDeadLettered = "dead_lettered"
)

// MaxAttempts is the maximum number of delivery attempts before dead-lettering.
const MaxAttempts = 3

// RetryDelays defines exponential backoff delays for retry attempts (attempt index 0-based).
var RetryDelays = []time.Duration{
	1 * time.Second,   // attempt 1 → wait 1s
	5 * time.Second,   // attempt 2 → wait 5s
	25 * time.Second,  // attempt 3 → dead-letter
}

// Endpoint represents a registered webhook endpoint.
type Endpoint struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	URL         string    `json:"url"`
	Secret      string    `json:"secret,omitempty"`
	EventTypes  []string  `json:"event_types"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Delivery represents a single webhook delivery attempt record.
type Delivery struct {
	ID                string     `json:"id"`
	EndpointID        string     `json:"endpoint_id"`
	EventType         string     `json:"event_type"`
	Payload           []byte     `json:"payload"`
	Status            string     `json:"status"`
	Attempts          int        `json:"attempts"`
	MaxAttempts       int        `json:"max_attempts"`
	NextRetryAt       time.Time  `json:"next_retry_at"`
	DeliveredAt       *time.Time `json:"delivered_at,omitempty"`
	DeadLetteredAt    *time.Time `json:"dead_lettered_at,omitempty"`
	LastResponseCode  *int       `json:"last_response_code,omitempty"`
	LastResponseBody  *string    `json:"last_response_body,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

// DispatchRequest is the payload accepted by POST /internal/events.
type DispatchRequest struct {
	OrgID     string `json:"org_id"`
	EventType string `json:"event_type"`
	Payload   any    `json:"payload"`
}

