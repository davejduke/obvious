// Package grc defines the GRC connector interface for outbound GRC integrations.
// GRC connectors export audit findings to governance, risk, and compliance platforms.
package grc

import (
	"context"
	"time"
)

// Severity represents the severity level of an audit finding.
type Severity string

const (
	SeverityCritical      Severity = "critical"
	SeverityHigh          Severity = "high"
	SeverityMedium        Severity = "medium"
	SeverityLow           Severity = "low"
	SeverityInformational Severity = "informational"
)

// Finding represents an audit finding to be exported to a GRC platform.
type Finding struct {
	ID             string    `json:"id"`
	Ref            string    `json:"ref"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Severity       Severity  `json:"severity"`
	Recommendation string    `json:"recommendation"`
	ControlRef     string    `json:"control_ref,omitempty"`
	AssetRef       string    `json:"asset_ref,omitempty"`
	DetectedAt     time.Time `json:"detected_at"`
}

// GRCItem represents a mapped compliance/policy item in the GRC system (sn_grc_item).
type GRCItem struct {
	SysID       string    `json:"sys_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	Category    string    `json:"category"`
	Priority    string    `json:"priority"`
	SourceRef   string    `json:"source_ref"`
	ControlRef  string    `json:"control_ref,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// RiskEntry represents a risk register entry in the GRC system (sn_grc_risk).
type RiskEntry struct {
	SysID       string    `json:"sys_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	RiskScore   int       `json:"risk_score"`
	Category    string    `json:"category"`
	Treatment   string    `json:"treatment"`
	Owner       string    `json:"owner"`
	// CMDBCI is the reference to the affected configuration item (cmdb_ci table).
	CMDBCI    string    `json:"cmdb_ci"`
	SourceRef string    `json:"source_ref"`
	CreatedAt time.Time `json:"created_at"`
}

// RemediationTask represents a remediation task linked to a GRC item.
type RemediationTask struct {
	SysID       string    `json:"sys_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	Priority    string    `json:"priority"`
	AssignedTo  string    `json:"assigned_to"`
	DueDate     time.Time `json:"due_date"`
	GRCItemRef  string    `json:"grc_item_ref"`
	SourceRef   string    `json:"source_ref"`
}

// ExportResult holds the outcome of a GRC export operation.
type ExportResult struct {
	Connector        string            `json:"connector"`
	GRCItems         []GRCItem         `json:"grc_items"`
	RiskEntries      []RiskEntry       `json:"risk_entries"`
	RemediationTasks []RemediationTask `json:"remediation_tasks"`
	TotalFindings    int               `json:"total_findings"`
	ExportedAt       time.Time         `json:"exported_at"`
}

// HealthStatus represents the health of a GRC connector.
type HealthStatus struct {
	Healthy     bool      `json:"healthy"`
	Connector   string    `json:"connector"`
	LastChecked time.Time `json:"last_checked"`
	Message     string    `json:"message,omitempty"`
}

// GRCConnector is the interface all GRC outbound adapters must implement.
// Adapters normalise audit findings to the target GRC platform's data model.
type GRCConnector interface {
	// Name returns the unique connector identifier (e.g. "servicenow").
	Name() string

	// ExportFindings maps audit findings to GRC platform records.
	// Returns a summary of all records created across GRC tables.
	ExportFindings(ctx context.Context, findings []Finding) (*ExportResult, error)

	// Health reports the connectivity status of this GRC connector.
	Health(ctx context.Context) HealthStatus
}

// Registry holds all registered GRC connectors.
type Registry struct {
	connectors map[string]GRCConnector
}

// NewRegistry creates an empty GRC connector registry.
func NewRegistry() *Registry {
	return &Registry{connectors: make(map[string]GRCConnector)}
}

// Register adds a GRC connector to the registry.
func (r *Registry) Register(c GRCConnector) {
	r.connectors[c.Name()] = c
}

// Get returns a GRC connector by name.
func (r *Registry) Get(name string) (GRCConnector, bool) {
	c, ok := r.connectors[name]
	return c, ok
}

// List returns all registered GRC connector names.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.connectors))
	for n := range r.connectors {
		names = append(names, n)
	}
	return names
}

