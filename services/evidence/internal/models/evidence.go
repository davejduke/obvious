// Package models defines evidence domain models for the AIAUDITOR evidence service.
package models

import (
	"time"

	"github.com/google/uuid"
)

// EvidenceTier represents the 6-tier evidence hierarchy (locked decision).
// L1 Policy > L2 Standard > L3 Guideline > L4 Procedure > L5 Record > L6 Telemetry
type EvidenceTier int

const (
	TierL1Policy    EvidenceTier = 1 // Governance policies, regulatory mandates
	TierL2Standard  EvidenceTier = 2 // Standards, frameworks, benchmarks
	TierL3Guideline EvidenceTier = 3 // Guidelines, best practices, recommendations
	TierL4Procedure EvidenceTier = 4 // SOPs, work instructions, playbooks
	TierL5Record    EvidenceTier = 5 // Logs, screenshots, configs, scan results
	TierL6Telemetry EvidenceTier = 6 // Automated telemetry, real-time metrics, API feeds
)

// String returns the human-readable label for a tier.
func (t EvidenceTier) String() string {
	switch t {
	case TierL1Policy:
		return "L1-Policy"
	case TierL2Standard:
		return "L2-Standard"
	case TierL3Guideline:
		return "L3-Guideline"
	case TierL4Procedure:
		return "L4-Procedure"
	case TierL5Record:
		return "L5-Record"
	case TierL6Telemetry:
		return "L6-Telemetry"
	default:
		return "Unknown"
	}
}

// SourceType represents how evidence was collected.
type SourceType string

const (
	SourceManualUpload        SourceType = "manual_upload"
	SourceAPIIntegration      SourceType = "api_integration"
	SourceAutomatedScan       SourceType = "automated_scan"
	SourceScreenshot          SourceType = "screenshot"
	SourceLogExport           SourceType = "log_export"
	SourceConfigurationExport SourceType = "configuration_export"
)

// ContentFormat is the wire format of evidence content.
type ContentFormat string

const (
	FormatJSON ContentFormat = "json"
	FormatCSV  ContentFormat = "csv"
	FormatLog  ContentFormat = "log"
	FormatText ContentFormat = "text"
)

// RiskRating mirrors the shared platform risk rating.
type RiskRating string

const (
	RiskCritical      RiskRating = "critical"
	RiskHigh          RiskRating = "high"
	RiskMedium        RiskRating = "medium"
	RiskLow           RiskRating = "low"
	RiskInformational RiskRating = "informational"
)

// ProvenanceEntry records a step in the evidence chain of custody.
type ProvenanceEntry struct {
	Timestamp   time.Time  `json:"timestamp"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	Action      string     `json:"action"`
	Description string     `json:"description"`
	Hash        string     `json:"hash,omitempty"`
}

// Evidence is a single piece of audit evidence.
type Evidence struct {
	ID              uuid.UUID              `json:"id"`
	OrgID           uuid.UUID              `json:"org_id"`
	EngagementID    uuid.UUID              `json:"engagement_id"`
	ControlID       uuid.UUID              `json:"control_id"`
	UploadedByID    *uuid.UUID             `json:"uploaded_by_id,omitempty"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description,omitempty"`
	SourceType      SourceType             `json:"source_type"`
	SourceRef       string                 `json:"source_ref,omitempty"`
	ContentFormat   ContentFormat          `json:"content_format"`
	Content         string                 `json:"content,omitempty"`
	FilePath        string                 `json:"file_path,omitempty"`
	FileHash        string                 `json:"file_hash,omitempty"`
	CollectedAt     time.Time              `json:"collected_at"`
	Tags            []string               `json:"tags"`
	Metadata        map[string]interface{} `json:"metadata"`
	ProvenanceChain []ProvenanceEntry      `json:"provenance_chain"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// Classification is the result of classifying evidence into the 6-tier hierarchy.
type Classification struct {
	ID           uuid.UUID    `json:"id"`
	EvidenceID   uuid.UUID    `json:"evidence_id"`
	OrgID        uuid.UUID    `json:"org_id"`
	Tier         EvidenceTier `json:"tier"`
	TierLabel    string       `json:"tier_label"`
	Confidence   float64      `json:"confidence"`
	Reasoning    string       `json:"reasoning"`
	ClassifiedAt time.Time    `json:"classified_at"`
}

// QualityScore holds multi-dimension quality scores for evidence (0.0–1.0 each).
type QualityScore struct {
	ID                 uuid.UUID `json:"id"`
	EvidenceID         uuid.UUID `json:"evidence_id"`
	OrgID              uuid.UUID `json:"org_id"`
	CompletenessScore  float64   `json:"completeness_score"`  // Required fields present?
	CurrencyScore      float64   `json:"currency_score"`      // How recent?
	SourceReliability  float64   `json:"source_reliability"`  // Source trustworthiness
	CorroborationScore float64   `json:"corroboration_score"` // Supported by other evidence?
	RelevanceScore     float64   `json:"relevance_score"`     // Directly addresses the control?
	AggregateScore     float64   `json:"aggregate_score"`     // Weighted average (0.0–1.0)
	Deficiencies       []string  `json:"deficiencies"`
	ScoredAt           time.Time `json:"scored_at"`
}

// SufficiencyResult holds the sufficiency evaluation for a control.
type SufficiencyResult struct {
	ControlID        uuid.UUID   `json:"control_id"`
	RiskRating       RiskRating  `json:"risk_rating"`
	EvidenceCount    int         `json:"evidence_count"`
	MinRequired      int         `json:"min_required"`
	AggregateScore   float64     `json:"aggregate_score"`
	MinScoreRequired float64     `json:"min_score_required"`
	IsSufficient     bool        `json:"is_sufficient"`
	Gaps             []string    `json:"gaps"`
	EvidenceIDs      []uuid.UUID `json:"evidence_ids"`
}

// IngestRequest is the request body for ingesting a single evidence item.
type IngestRequest struct {
	OrgID         string                 `json:"org_id"`
	EngagementID  string                 `json:"engagement_id"`
	ControlID     string                 `json:"control_id"`
	Title         string                 `json:"title"`
	Description   string                 `json:"description,omitempty"`
	SourceType    SourceType             `json:"source_type"`
	SourceRef     string                 `json:"source_ref,omitempty"`
	ContentFormat ContentFormat          `json:"content_format"`
	Content       string                 `json:"content"`
	Tags          []string               `json:"tags,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	CollectedAt   *time.Time             `json:"collected_at,omitempty"`
}

// BatchIngestRequest holds multiple evidence items for batch ingestion.
type BatchIngestRequest struct {
	Items []IngestRequest `json:"items"`
}

