// Package hashchain implements the SHA-256 hash chain used by the audit trail.
//
// Chain rule:
//
//	event_hash = hex( SHA-256( previous_hash + canonical_payload ) )
//
// The canonical payload is the JSON-encoded AppendRequest with the org_id
// and the RFC3339Nano timestamp of the event, so the hash captures every
// meaningful field that identifies the event.
package hashchain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/davejduke/obvious/services/audit-trail/internal/model"
)

// GenesisHash is the fixed previous_hash for the very first event in a chain.
// Using an empty string keeps it simple while still being deterministic.
const GenesisHash = ""

// Compute returns the SHA-256 hash for a new event given its predecessor's
// hash and the fields that will be stored.
//
// The canonical input to the hash function is:
//
//	"<previousHash>" +
//	  JSON({ org_id, actor_email, action, event_type,
//	         resource_type, resource_id, engagement_id,
//	         changes, occurred_at })
//
// Note: ip_address / user_agent / context are deliberately excluded from the
// chain so that operational metadata cannot be used to manufacture collisions.
func Compute(previousHash string, req *model.AppendRequest, occurredAt time.Time) (string, error) {
	type canonicalFields struct {
		OrgID        string                 `json:"org_id"`
		ActorEmail   string                 `json:"actor_email"`
		Action       string                 `json:"action"`
		EventType    string                 `json:"event_type"`
		ResourceType string                 `json:"resource_type"`
		ResourceID   string                 `json:"resource_id"`
		EngagementID string                 `json:"engagement_id"`
		Changes      map[string]interface{} `json:"changes"`
		OccurredAt   string                 `json:"occurred_at"`
	}

	resourceID := ""
	if req.ResourceID != nil {
		resourceID = req.ResourceID.String()
	}
	engagementID := ""
	if req.EngagementID != nil {
		engagementID = req.EngagementID.String()
	}

	canonical := canonicalFields{
		OrgID:        req.OrgID.String(),
		ActorEmail:   req.ActorEmail,
		Action:       req.Action,
		EventType:    string(req.EventType),
		ResourceType: req.ResourceType,
		ResourceID:   resourceID,
		EngagementID: engagementID,
		Changes:      req.Changes,
		OccurredAt:   occurredAt.UTC().Format(time.RFC3339Nano),
	}

	payload, err := json.Marshal(canonical)
	if err != nil {
		return "", fmt.Errorf("hashchain: marshal canonical fields: %w", err)
	}

	input := previousHash + string(payload)
	sum := sha256.Sum256([]byte(input))
	return hex.EncodeToString(sum[:]), nil
}

// Verify recomputes the hash for a stored event and checks that it matches
// event.EventHash. It also checks that event.PreviousHash equals the supplied
// expectedPrevious.
//
// This is called row-by-row during chain verification.
func Verify(event *model.AuditEvent, expectedPrevious string) bool {
	if event.PreviousHash != expectedPrevious {
		return false
	}

	// Reconstruct the canonical fields from the stored event.
	type canonicalFields struct {
		OrgID        string                 `json:"org_id"`
		ActorEmail   string                 `json:"actor_email"`
		Action       string                 `json:"action"`
		EventType    string                 `json:"event_type"`
		ResourceType string                 `json:"resource_type"`
		ResourceID   string                 `json:"resource_id"`
		EngagementID string                 `json:"engagement_id"`
		Changes      map[string]interface{} `json:"changes"`
		OccurredAt   string                 `json:"occurred_at"`
	}

	resourceID := ""
	if event.ResourceID != nil {
		resourceID = event.ResourceID.String()
	}
	engagementID := ""
	if event.EngagementID != nil {
		engagementID = event.EngagementID.String()
	}

	canonical := canonicalFields{
		OrgID:        event.OrgID.String(),
		ActorEmail:   event.ActorEmail,
		Action:       event.Action,
		EventType:    string(event.EventType),
		ResourceType: event.ResourceType,
		ResourceID:   resourceID,
		EngagementID: engagementID,
		Changes:      event.Changes,
		OccurredAt:   event.OccurredAt.UTC().Format(time.RFC3339Nano),
	}

	payload, err := json.Marshal(canonical)
	if err != nil {
		return false
	}

	input := expectedPrevious + string(payload)
	sum := sha256.Sum256([]byte(input))
	expected := hex.EncodeToString(sum[:])
	return expected == event.EventHash
}

