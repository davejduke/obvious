// Package types provides shared domain types for the AIAUDITOR platform.
package types

import (
	"time"

	"github.com/google/uuid"
)

// Persona represents the 7 RBAC personas in AIAUDITOR.
type Persona string

const (
	PersonaInternalAuditor   Persona = "internal_auditor"
	PersonaCAE               Persona = "cae"
	PersonaAuditCommittee    Persona = "audit_committee"
	PersonaAuditeeCISO       Persona = "auditee_ciso"
	PersonaCosourcedProvider Persona = "cosourced_provider"
	PersonaPTWGMember        Persona = "ptwg_member"
	PersonaBetaTester        Persona = "beta_tester"
)

// EngagementStatus represents the lifecycle state of an audit engagement.
type EngagementStatus string

const (
	EngagementStatusPlanning   EngagementStatus = "planning"
	EngagementStatusFieldwork  EngagementStatus = "fieldwork"
	EngagementStatusReview     EngagementStatus = "review"
	EngagementStatusReporting  EngagementStatus = "reporting"
	EngagementStatusCompleted  EngagementStatus = "completed"
	EngagementStatusCancelled  EngagementStatus = "cancelled"
)

// FindingSeverity represents the severity of an audit finding.
type FindingSeverity string

const (
	FindingSeverityCritical      FindingSeverity = "critical"
	FindingSeverityHigh          FindingSeverity = "high"
	FindingSeverityMedium        FindingSeverity = "medium"
	FindingSeverityLow           FindingSeverity = "low"
	FindingSeverityInformational FindingSeverity = "informational"
)

// RiskRating represents overall risk rating.
type RiskRating string

const (
	RiskRatingCritical      RiskRating = "critical"
	RiskRatingHigh          RiskRating = "high"
	RiskRatingMedium        RiskRating = "medium"
	RiskRatingLow           RiskRating = "low"
	RiskRatingInformational RiskRating = "informational"
)

// EvidenceSourceType represents how evidence was collected.
type EvidenceSourceType string

const (
	EvidenceSourceManualUpload          EvidenceSourceType = "manual_upload"
	EvidenceSourceAPIIntegration        EvidenceSourceType = "api_integration"
	EvidenceSourceAutomatedScan         EvidenceSourceType = "automated_scan"
	EvidenceSourceScreenshot            EvidenceSourceType = "screenshot"
	EvidenceSourceLogExport             EvidenceSourceType = "log_export"
	EvidenceSourceConfigurationExport   EvidenceSourceType = "configuration_export"
)

// Organization represents a tenant organization.
type Organization struct {
	ID          uuid.UUID              `json:"id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Tier        string                 `json:"tier"`
	Industry    string                 `json:"industry,omitempty"`
	CountryCode string                 `json:"country_code,omitempty"`
	Settings    map[string]interface{} `json:"settings"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// User represents a platform user.
type User struct {
	ID          uuid.UUID              `json:"id"`
	OrgID       uuid.UUID              `json:"org_id"`
	Email       string                 `json:"email"`
	DisplayName string                 `json:"display_name"`
	IsActive    bool                   `json:"is_active"`
	LastLoginAt *time.Time             `json:"last_login_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// Control represents a NIS 2 or other framework control.
type Control struct {
	ID                   uuid.UUID              `json:"id"`
	FrameworkID          uuid.UUID              `json:"framework_id"`
	OrgID                uuid.UUID              `json:"org_id"`
	ParentID             *uuid.UUID             `json:"parent_id,omitempty"`
	ControlID            string                 `json:"control_id"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description,omitempty"`
	Objective            string                 `json:"objective,omitempty"`
	Category             string                 `json:"category,omitempty"`
	Domain               string                 `json:"domain,omitempty"`
	ArticleRef           string                 `json:"article_ref,omitempty"`
	RiskWeight           float64                `json:"risk_weight"`
	Tags                 []string               `json:"tags"`
	IsActive             bool                   `json:"is_active"`
	CreatedAt            time.Time              `json:"created_at"`
	UpdatedAt            time.Time              `json:"updated_at"`
}

// Engagement represents an audit engagement.
type Engagement struct {
	ID               uuid.UUID              `json:"id"`
	OrgID            uuid.UUID              `json:"org_id"`
	FrameworkID      uuid.UUID              `json:"framework_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description,omitempty"`
	Status           EngagementStatus       `json:"status"`
	ScopeJSON        map[string]interface{} `json:"scope_json"`
	LeadAuditorID    *uuid.UUID             `json:"lead_auditor_id,omitempty"`
	TargetStartDate  *time.Time             `json:"target_start_date,omitempty"`
	TargetEndDate    *time.Time             `json:"target_end_date,omitempty"`
	OverallScore     *float64               `json:"overall_score,omitempty"`
	RiskRating       RiskRating             `json:"risk_rating,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// Finding represents an audit finding.
type Finding struct {
	ID           uuid.UUID              `json:"id"`
	OrgID        uuid.UUID              `json:"org_id"`
	EngagementID uuid.UUID              `json:"engagement_id"`
	ControlID    uuid.UUID              `json:"control_id"`
	FindingRef   string                 `json:"finding_ref"`
	Title        string                 `json:"title"`
	Description  string                 `json:"description"`
	RootCause    string                 `json:"root_cause,omitempty"`
	Impact       string                 `json:"impact,omitempty"`
	Severity     FindingSeverity        `json:"severity"`
	Status       string                 `json:"status"`
	DueDate      *time.Time             `json:"due_date,omitempty"`
	EvidenceIDs  []uuid.UUID            `json:"evidence_ids"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// AuditEvent represents an immutable audit log entry.
type AuditEvent struct {
	ID           int64                  `json:"id"`
	OrgID        uuid.UUID              `json:"org_id"`
	EventID      uuid.UUID              `json:"event_id"`
	ActorID      *uuid.UUID             `json:"actor_id,omitempty"`
	ActorEmail   string                 `json:"actor_email,omitempty"`
	Action       string                 `json:"action"`
	ResourceType string                 `json:"resource_type"`
	ResourceID   *uuid.UUID             `json:"resource_id,omitempty"`
	Changes      map[string]interface{} `json:"changes,omitempty"`
	Context      map[string]interface{} `json:"context"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	PreviousHash string                 `json:"previous_hash"`
	EventHash    string                 `json:"event_hash"`
	OccurredAt   time.Time              `json:"occurred_at"`
}

