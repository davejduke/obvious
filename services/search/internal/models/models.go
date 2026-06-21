// Package models defines the domain types for the search service.
package models

import "time"

// EntityType is the kind of document indexed in OpenSearch.
type EntityType string

const (
	EntityFinding  EntityType = "finding"
	EntityEvidence EntityType = "evidence"
	EntityControl  EntityType = "control"
)

// FindingDoc is the OpenSearch document for a finding.
type FindingDoc struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"org_id"`
	EngagementID string    `json:"engagement_id"`
	FindingRef  string     `json:"finding_ref"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	RootCause   string     `json:"root_cause"`
	Severity    string     `json:"severity"`
	Status      string     `json:"status"`
	Tags        []string   `json:"tags"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// EvidenceDoc is the OpenSearch document for an evidence item.
type EvidenceDoc struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	ControlID    string    `json:"control_id"`
	Source       string    `json:"source"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	RelevanceTags []string `json:"relevance_tags"`
	EvidenceType string    `json:"evidence_type"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ControlDoc is the OpenSearch document for a control objective.
type ControlDoc struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	ArticleRef  string    `json:"article_ref"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Domain      string    `json:"domain"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ChangeEvent is emitted by the CDC pipeline when a row changes.
type ChangeEvent struct {
	Op     string // INSERT, UPDATE, DELETE
	Table  string // findings, evidence, controls
	ID     string // row UUID
	OrgID  string // tenant UUID
}

// SearchResult is a single hit returned from OpenSearch.
type SearchResult struct {
	ID         string            `json:"id"`
	Type       EntityType        `json:"type"`
	Title      string            `json:"title"`
	Description string           `json:"description"`
	OrgID      string            `json:"org_id"`
	Score      float64           `json:"score"`
	Highlights map[string][]string `json:"highlights"`
	Meta       map[string]interface{} `json:"meta"`
}

// SearchResponse groups results by entity type.
type SearchResponse struct {
	Query    string         `json:"query"`
	Total    int            `json:"total"`
	Findings []SearchResult `json:"findings"`
	Evidence []SearchResult `json:"evidence"`
	Controls []SearchResult `json:"controls"`
}

