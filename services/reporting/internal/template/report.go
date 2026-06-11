// Package template provides the report data model and template engine.
package template

import (
	"time"

	"github.com/google/uuid"
)

// Severity levels used in findings.
type Severity string

const (
	SeverityCritical      Severity = "critical"
	SeverityHigh          Severity = "high"
	SeverityMedium        Severity = "medium"
	SeverityLow           Severity = "low"
	SeverityInformational Severity = "informational"
)

// Finding represents a single audit finding included in a report.
type Finding struct {
	ID             uuid.UUID `json:"id"`
	Ref            string    `json:"ref"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Severity       Severity  `json:"severity"`
	Recommendation string    `json:"recommendation"`
	ControlRef     string    `json:"control_ref,omitempty"`
	EvidenceRefs   []string  `json:"evidence_refs,omitempty"`
}

// EvidenceItem represents a piece of evidence included in a report.
type EvidenceItem struct {
	ID             uuid.UUID `json:"id"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	SourceType     string    `json:"source_type"`
	CollectedAt    time.Time `json:"collected_at"`
	CollectedBy    string    `json:"collected_by"`
	Hash           string    `json:"hash,omitempty"`
	IntegrationSrc string    `json:"integration_src,omitempty"`
}

// ReportMetadata holds the report header information.
type ReportMetadata struct {
	ReportID      uuid.UUID `json:"report_id"`
	EngagementID  uuid.UUID `json:"engagement_id"`
	OrgName       string    `json:"org_name"`
	Framework     string    `json:"framework"`
	ReportTitle   string    `json:"report_title"`
	AuditorName   string    `json:"auditor_name"`
	AuditorEmail  string    `json:"auditor_email"`
	PeriodStart   time.Time `json:"period_start"`
	PeriodEnd     time.Time `json:"period_end"`
	GeneratedAt   time.Time `json:"generated_at"`
	Classification string   `json:"classification"`
}

// SummaryStats aggregates finding counts by severity.
type SummaryStats struct {
	TotalFindings    int `json:"total_findings"`
	Critical         int `json:"critical"`
	High             int `json:"high"`
	Medium           int `json:"medium"`
	Low              int `json:"low"`
	Informational    int `json:"informational"`
	TotalEvidence    int `json:"total_evidence"`
}

// ReportData is the complete data model for report generation.
type ReportData struct {
	Metadata   ReportMetadata `json:"metadata"`
	Summary    SummaryStats   `json:"summary"`
	Findings   []Finding      `json:"findings"`
	Evidence   []EvidenceItem `json:"evidence"`
	ExecSummary string        `json:"exec_summary"`
}

// BuildSummary computes SummaryStats from the Findings and Evidence slices.
func (r *ReportData) BuildSummary() {
	stats := SummaryStats{TotalEvidence: len(r.Evidence)}
	for _, f := range r.Findings {
		stats.TotalFindings++
		switch f.Severity {
		case SeverityCritical:
			stats.Critical++
		case SeverityHigh:
			stats.High++
		case SeverityMedium:
			stats.Medium++
		case SeverityLow:
			stats.Low++
		case SeverityInformational:
			stats.Informational++
		}
	}
	r.Summary = stats
}

// EvidenceChain returns evidence items linked to a finding (by ID string match).
func (r *ReportData) EvidenceChain(finding Finding) []EvidenceItem {
	var chain []EvidenceItem
	for _, ref := range finding.EvidenceRefs {
		for _, ev := range r.Evidence {
			if ev.ID.String() == ref {
				chain = append(chain, ev)
			}
		}
	}
	return chain
}

