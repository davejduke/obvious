// Package iam defines the pluggable IAM connector interface and normalised evidence model
// for AIAUDITOR's integration service.
package iam

import (
	"context"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/connector"
)

// ---------------------------------------------------------------------------
// Enumerations
// ---------------------------------------------------------------------------

// UserStatus represents the lifecycle state of an IAM user across providers.
type UserStatus string

const (
	UserStatusActive        UserStatus = "active"
	UserStatusInactive      UserStatus = "inactive"
	UserStatusSuspended     UserStatus = "suspended"
	UserStatusDeprovisioned UserStatus = "deprovisioned"
)

// ---------------------------------------------------------------------------
// Normalised models
// ---------------------------------------------------------------------------

// MFAFactor represents a single enrolled MFA factor for a user.
type MFAFactor struct {
	ID       string `json:"id"`
	Type     string `json:"type"`     // fido2, webauthn, totp, push, sms, email, otp
	Status   string `json:"status"`   // active, pending_activation, expired
	Provider string `json:"provider"` // entraid | okta
}

// AppAssignment represents a user's assignment to an application.
type AppAssignment struct {
	AppID   string `json:"app_id"`
	AppName string `json:"app_name"`
	Status  string `json:"status"` // active | inactive
}

// IAMUser is the canonical user record normalised across IAM providers.
type IAMUser struct {
	ID             string          `json:"id"`
	ExternalID     string          `json:"external_id"`      // provider-native ID
	Email          string          `json:"email"`
	DisplayName    string          `json:"display_name"`
	Status         UserStatus      `json:"status"`
	CreatedAt      time.Time       `json:"created_at"`
	LastLoginAt    *time.Time      `json:"last_login_at,omitempty"`
	MFAEnrolled    bool            `json:"mfa_enrolled"`
	MFAFactors     []MFAFactor     `json:"mfa_factors,omitempty"`
	Groups         []string        `json:"groups,omitempty"`
	AppAssignments []AppAssignment `json:"app_assignments,omitempty"`
	Provider       string          `json:"provider"` // entraid | okta
}

// AccessReview represents an access review decision (primarily Entra ID).
type AccessReview struct {
	ID            string    `json:"id"`
	ReviewerID    string    `json:"reviewer_id"`
	ResourceID    string    `json:"resource_id"`
	Decision      string    `json:"decision"` // approve | deny | dontknow
	Justification string    `json:"justification,omitempty"`
	CompletedAt   time.Time `json:"completed_at"`
}

// GroupMembership represents a directory group with its current member list.
type GroupMembership struct {
	GroupID     string   `json:"group_id"`
	GroupName   string   `json:"group_name"`
	Description string   `json:"description,omitempty"`
	MemberIDs   []string `json:"member_ids,omitempty"`
}

// ConditionalAccessPolicy represents a conditional access policy (Entra ID).
type ConditionalAccessPolicy struct {
	ID            string            `json:"id"`
	DisplayName   string            `json:"display_name"`
	State         string            `json:"state"`                   // enabled | disabled | enabledForReportingButNotEnforced
	GrantControls []string          `json:"grant_controls,omitempty"` // mfa, compliantDevice, ...
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// SystemLogEvent represents an Okta system log event.
type SystemLogEvent struct {
	EventID     string    `json:"event_id"`
	PublishedAt time.Time `json:"published_at"`
	EventType   string    `json:"event_type"`
	Severity    string    `json:"severity"`            // DEBUG | INFO | WARN | ERROR
	ActorID     string    `json:"actor_id"`
	ActorEmail  string    `json:"actor_email,omitempty"`
	TargetID    string    `json:"target_id,omitempty"`
	TargetType  string    `json:"target_type,omitempty"`
	Outcome     string    `json:"outcome"` // SUCCESS | FAILURE | SKIPPED | ALLOW | DENY | CHALLENGE | UNKNOWN
}

// IAMSnapshot is the result of a full provider sync.
type IAMSnapshot struct {
	Provider            string                    `json:"provider"`
	CollectedAt         time.Time                 `json:"collected_at"`
	Users               []IAMUser                 `json:"users"`
	AccessReviews       []AccessReview            `json:"access_reviews,omitempty"`
	GroupMemberships    []GroupMembership         `json:"group_memberships,omitempty"`
	ConditionalPolicies []ConditionalAccessPolicy `json:"conditional_policies,omitempty"`
	SystemLogs          []SystemLogEvent          `json:"system_logs,omitempty"`
}

// ---------------------------------------------------------------------------
// Connector interface
// ---------------------------------------------------------------------------

// IAMConnector is the interface all IAM adapters must implement.
// It mirrors the Connector SDK shape used by SIEM adapters, adapted for IAM.
type IAMConnector interface {
	// Name returns the unique connector identifier (e.g. "entraid", "okta").
	Name() string

	// Sync performs a full data pull and returns a normalised IAM snapshot.
	Sync(ctx context.Context) (*IAMSnapshot, error)

	// Health reports the connectivity status of this connector.
	Health(ctx context.Context) connector.HealthStatus
}

// ---------------------------------------------------------------------------
// Registry
// ---------------------------------------------------------------------------

// IAMRegistry holds all registered IAM connectors.
type IAMRegistry struct {
	connectors map[string]IAMConnector
}

// NewIAMRegistry creates an empty IAM registry.
func NewIAMRegistry() *IAMRegistry {
	return &IAMRegistry{connectors: make(map[string]IAMConnector)}
}

// Register adds an IAM connector to the registry.
func (r *IAMRegistry) Register(c IAMConnector) {
	r.connectors[c.Name()] = c
}

// Get returns an IAM connector by name.
func (r *IAMRegistry) Get(name string) (IAMConnector, bool) {
	c, ok := r.connectors[name]
	return c, ok
}

// List returns all registered IAM connector names.
func (r *IAMRegistry) List() []string {
	names := make([]string, 0, len(r.connectors))
	for n := range r.connectors {
		names = append(names, n)
	}
	return names
}
