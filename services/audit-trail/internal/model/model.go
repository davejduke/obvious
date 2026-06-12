// Package model defines the domain types for the audit-trail service.
package model

import (
	"time"

	"github.com/google/uuid"
)

// EventType classifies the kind of activity recorded in an audit event.
type EventType string

const (
	EventTypeUserAction      EventType = "user_action"
	EventTypeSystemAction    EventType = "system_action"
	EventTypeEvidenceChange  EventType = "evidence_change"
	EventTypeFindingChange   EventType = "finding_change"
	EventTypeEngagementChange EventType = "engagement_change"
	EventTypeAuthEvent       EventType = "auth_event"
)

// AuditEvent mirrors the audit_events table row. The record is immutable
// once written; the hash chain fields link each event to its predecessor.
type AuditEvent struct {
	// ID is the monotonically-increasing surrogate key (BIGSERIAL).
	ID int64 `json:"id"`

	OrgID      uuid.UUID  `json:"org_id"`
	EventID    uuid.UUID  `json:"event_id"`
	ActorID    *uuid.UUID `json:"actor_id,omitempty"`
	ActorEmail string     `json:"actor_email,omitempty"`

	// Action is a dot-namespaced verb, e.g. "evidence.uploaded".
	Action string `json:"action"`

	// EventType categorises the event for meta-audit queries.
	EventType EventType `json:"event_type"`

	ResourceType string     `json:"resource_type"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty"`

	// EngagementID allows fast meta-audit queries per engagement.
	EngagementID *uuid.UUID `json:"engagement_id,omitempty"`

	Changes   map[string]interface{} `json:"changes,omitempty"`
	Context   map[string]interface{} `json:"context"`
	IPAddress string                 `json:"ip_address,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`

	// PreviousHash is the event_hash of the immediately preceding row in
	// the chain (empty string for the genesis event).
	PreviousHash string `json:"previous_hash"`

	// EventHash is SHA-256(previous_hash || canonical_event_data).
	EventHash string `json:"event_hash"`

	OccurredAt time.Time `json:"occurred_at"`
}

// AppendRequest is the payload accepted by POST /events.
type AppendRequest struct {
	OrgID        uuid.UUID              `json:"org_id"`
	ActorID      *uuid.UUID             `json:"actor_id,omitempty"`
	ActorEmail   string                 `json:"actor_email,omitempty"`
	Action       string                 `json:"action"`
	EventType    EventType              `json:"event_type"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   *uuid.UUID             `json:"resource_id,omitempty"`
	EngagementID *uuid.UUID             `json:"engagement_id,omitempty"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	UserAgent    string                 `json:"user_agent,omitempty"`
}

// VerifyResult is returned by POST /events/verify-chain.
type VerifyResult struct {
	// Valid is true when the entire chain is intact.
	Valid bool `json:"valid"`
	// TotalEvents is the number of events scanned.
	TotalEvents int64 `json:"total_events"`
	// TamperedAt is the sequential ID of the first broken link, if any.
	TamperedAt *int64 `json:"tampered_at,omitempty"`
	Message    string `json:"message"`
}

// MetaAuditEntry is a single row in the engagement history response.
type MetaAuditEntry struct {
	EventID      uuid.UUID  `json:"event_id"`
	OccurredAt   time.Time  `json:"occurred_at"`
	ActorEmail   string     `json:"actor_email,omitempty"`
	Action       string     `json:"action"`
	EventType    EventType  `json:"event_type"`
	ResourceType string     `json:"resource_type"`
	ResourceID   *uuid.UUID `json:"resource_id,omitempty"`
	EventHash    string     `json:"event_hash"`
}

