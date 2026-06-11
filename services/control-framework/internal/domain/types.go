// Package domain defines the core domain types for the Control Framework Service.
//
// NIS 2 Article 21 ontology:
//   (a) Risk analysis & IS policies
//   (b) Incident handling
//   (c) Business continuity & crisis management
//   (d) Supply chain security
//   (e) Network & IS acquisition/development/maintenance
//   (f) Vulnerability handling & disclosure
//   (g) Effectiveness assessment
//   (h) Cybersecurity hygiene & training
//   (i) Cryptography & encryption
//   (j) Authentication & access (MFA)
package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AssessmentStatus represents the lifecycle state of a control assessment.
type AssessmentStatus string

const (
	AssessmentStatusNotStarted  AssessmentStatus = "not_started"
	AssessmentStatusInProgress  AssessmentStatus = "in_progress"
	AssessmentStatusAssessed    AssessmentStatus = "assessed"
	AssessmentStatusRemediation AssessmentStatus = "remediation"
)

// ValidAssessmentTransitions defines allowed state transitions.
var ValidAssessmentTransitions = map[AssessmentStatus][]AssessmentStatus{
	AssessmentStatusNotStarted:  {AssessmentStatusInProgress},
	AssessmentStatusInProgress:  {AssessmentStatusAssessed, AssessmentStatusRemediation},
	AssessmentStatusAssessed:    {AssessmentStatusRemediation, AssessmentStatusInProgress},
	AssessmentStatusRemediation: {AssessmentStatusInProgress, AssessmentStatusAssessed},
}

// CanTransitionTo returns true if the transition from current to next is valid.
func (s AssessmentStatus) CanTransitionTo(next AssessmentStatus) bool {
	allowed, ok := ValidAssessmentTransitions[s]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == next {
			return true
		}
	}
	return false
}

// MappingType represents the relationship type between two mapped controls.
type MappingType string

const (
	MappingTypeEquivalent MappingType = "equivalent"
	MappingTypeSubset     MappingType = "subset"
	MappingTypeSuperset   MappingType = "superset"
	MappingTypePartial    MappingType = "partial"
	MappingTypeNone       MappingType = "none"
)

// ArticleRefNames maps NIS 2 article references to human-readable domain names.
var ArticleRefNames = map[string]string{
	"Art. 21(2)(a)": "Risk analysis and information system security policies",
	"Art. 21(2)(b)": "Incident handling",
	"Art. 21(2)(c)": "Business continuity and crisis management",
	"Art. 21(2)(d)": "Supply chain security",
	"Art. 21(2)(e)": "Security in network and information systems acquisition, development and maintenance",
	"Art. 21(2)(f)": "Policies and procedures to assess the effectiveness of cybersecurity risk management measures",
	"Art. 21(2)(g)": "Basic cyber hygiene practices and cybersecurity training",
	"Art. 21(2)(h)": "Policies and procedures regarding the use of cryptography and, where appropriate, encryption",
	"Art. 21(2)(i)": "Human resources security, access control policies and asset management",
	"Art. 21(2)(j)": "Use of multi-factor authentication or continuous authentication solutions, secured communications",
}

// NIS2Domains is the ordered list of all 10 NIS 2 Article 21 domains.
var NIS2Domains = []string{
	"Art. 21(2)(a)",
	"Art. 21(2)(b)",
	"Art. 21(2)(c)",
	"Art. 21(2)(d)",
	"Art. 21(2)(e)",
	"Art. 21(2)(f)",
	"Art. 21(2)(g)",
	"Art. 21(2)(h)",
	"Art. 21(2)(i)",
	"Art. 21(2)(j)",
}

// Framework represents a control framework (NIS 2, NIST CSF, ISO 27001, CIS Controls).
type Framework struct {
	ID          uuid.UUID              `json:"id"`
	OrgID       uuid.UUID              `json:"org_id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Authority   string                 `json:"authority"`
	Description string                 `json:"description,omitempty"`
	IsPublished bool                   `json:"is_published"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// EvidenceRequirement describes what evidence is needed for a control.
type EvidenceRequirement struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// TestProcedure describes a procedure for testing a control.
type TestProcedure struct {
	ID               string `json:"id"`
	Description      string `json:"description"`
	ExpectedEvidence string `json:"expected_evidence"`
}

// Control represents a NIS 2 or cross-framework security control.
type Control struct {
	ID                   uuid.UUID             `json:"id"`
	FrameworkID          uuid.UUID             `json:"framework_id"`
	OrgID                uuid.UUID             `json:"org_id"`
	ParentID             *uuid.UUID            `json:"parent_id,omitempty"`
	ControlID            string                `json:"control_id"`
	Title                string                `json:"title"`
	Description          string                `json:"description,omitempty"`
	Objective            string                `json:"objective,omitempty"`
	Category             string                `json:"category,omitempty"`
	Domain               string                `json:"domain,omitempty"`
	ArticleRef           string                `json:"article_ref,omitempty"`
	ImplementationNotes  string                `json:"implementation_notes,omitempty"`
	TestProcedures       []TestProcedure       `json:"test_procedures"`
	EvidenceRequirements []EvidenceRequirement `json:"evidence_requirements"`
	RiskWeight           float64               `json:"risk_weight"`
	Tags                 []string              `json:"tags"`
	IsActive             bool                  `json:"is_active"`
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

// FrameworkMapping represents a cross-framework control mapping.
type FrameworkMapping struct {
	ID                  uuid.UUID   `json:"id"`
	SourceFrameworkID   uuid.UUID   `json:"source_framework_id"`
	TargetFrameworkID   uuid.UUID   `json:"target_framework_id"`
	SourceControlID     uuid.UUID   `json:"source_control_id"`
	TargetControlID     uuid.UUID   `json:"target_control_id"`
	MappingType         MappingType `json:"mapping_type"`
	Confidence          float64     `json:"confidence"`
	Notes               string      `json:"notes,omitempty"`
	CreatedAt           time.Time   `json:"created_at"`
	// Enriched fields populated by joins
	TargetControlRef    string `json:"target_control_ref,omitempty"`
	TargetControlTitle  string `json:"target_control_title,omitempty"`
	TargetFrameworkName string `json:"target_framework_name,omitempty"`
}

// ControlAssessment tracks the assessment status of a control within an engagement.
type ControlAssessment struct {
	ID             uuid.UUID        `json:"id"`
	EngagementID   uuid.UUID        `json:"engagement_id"`
	ControlID      uuid.UUID        `json:"control_id"`
	OrgID          uuid.UUID        `json:"org_id"`
	Status         AssessmentStatus `json:"status"`
	AssessedByID   *uuid.UUID       `json:"assessed_by_id,omitempty"`
	Score          *float64         `json:"score,omitempty"`
	Notes          string           `json:"notes,omitempty"`
	EvidenceIDs    []uuid.UUID      `json:"evidence_ids"`
	TransitionedAt time.Time        `json:"transitioned_at"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// AuditObjective represents an AI-generated audit objective for a control.
type AuditObjective struct {
	ControlID      uuid.UUID `json:"control_id"`
	ControlRef     string    `json:"control_ref"`
	ArticleRef     string    `json:"article_ref"`
	Title          string    `json:"title"`
	Objective      string    `json:"objective"`
	TestApproaches []string  `json:"test_approaches"`
	RiskFocus      string    `json:"risk_focus"`
}

// CreateFrameworkRequest is the payload for creating a framework.
type CreateFrameworkRequest struct {
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Authority   string                 `json:"authority"`
	Description string                 `json:"description,omitempty"`
	IsPublished bool                   `json:"is_published"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// CreateControlRequest is the payload for creating a control.
type CreateControlRequest struct {
	FrameworkID          uuid.UUID             `json:"framework_id"`
	ParentID             *uuid.UUID            `json:"parent_id,omitempty"`
	ControlID            string                `json:"control_id"`
	Title                string                `json:"title"`
	Description          string                `json:"description,omitempty"`
	Objective            string                `json:"objective,omitempty"`
	Category             string                `json:"category,omitempty"`
	Domain               string                `json:"domain,omitempty"`
	ArticleRef           string                `json:"article_ref,omitempty"`
	ImplementationNotes  string                `json:"implementation_notes,omitempty"`
	TestProcedures       []TestProcedure       `json:"test_procedures,omitempty"`
	EvidenceRequirements []EvidenceRequirement `json:"evidence_requirements,omitempty"`
	RiskWeight           float64               `json:"risk_weight"`
	Tags                 []string              `json:"tags,omitempty"`
}

// UpdateControlRequest is the payload for updating a control.
type UpdateControlRequest struct {
	Title                *string               `json:"title,omitempty"`
	Description          *string               `json:"description,omitempty"`
	Objective            *string               `json:"objective,omitempty"`
	Category             *string               `json:"category,omitempty"`
	ImplementationNotes  *string               `json:"implementation_notes,omitempty"`
	TestProcedures       []TestProcedure       `json:"test_procedures,omitempty"`
	EvidenceRequirements []EvidenceRequirement `json:"evidence_requirements,omitempty"`
	RiskWeight           *float64              `json:"risk_weight,omitempty"`
	Tags                 []string              `json:"tags,omitempty"`
	IsActive             *bool                 `json:"is_active,omitempty"`
}

// AssessControlRequest is the payload for assessing a control.
type AssessControlRequest struct {
	EngagementID uuid.UUID        `json:"engagement_id"`
	Status       AssessmentStatus `json:"status"`
	Score        *float64         `json:"score,omitempty"`
	Notes        string           `json:"notes,omitempty"`
	EvidenceIDs  []uuid.UUID      `json:"evidence_ids,omitempty"`
}

// ListControlsFilter contains query parameters for listing controls.
type ListControlsFilter struct {
	FrameworkID *uuid.UUID
	Domain      *string
	ArticleRef  *string
	ParentID    *uuid.UUID
	IsActive    *bool
	Search      *string
	Limit       int
	Offset      int
}

// PaginatedControls wraps a list of controls with pagination metadata.
type PaginatedControls struct {
	Controls []Control `json:"controls"`
	Total    int       `json:"total"`
	Limit    int       `json:"limit"`
	Offset   int       `json:"offset"`
}

// APIResponse is a generic API wrapper.
type APIResponse struct {
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// MustMarshal safely marshals an interface to JSON or falls back to null.
func MustMarshal(v interface{}) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return b
}

