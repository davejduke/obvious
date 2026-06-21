// Package vulnconnector defines the pluggable connector interface for vulnerability
// scanners and endpoint security platforms.
package vulnconnector

import (
	"context"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/connector"
)

// Severity is a normalized severity level aligned to CVSSv3 bands.
type Severity string

const (
	SeverityCritical      Severity = "critical"     // CVSSv3 >= 9.0
	SeverityHigh          Severity = "high"          // CVSSv3 7.0-8.9
	SeverityMedium        Severity = "medium"        // CVSSv3 4.0-6.9
	SeverityLow           Severity = "low"           // CVSSv3 0.1-3.9
	SeverityInformational Severity = "informational" // CVSSv3 == 0 / no score
	SeverityNone          Severity = "none"          // Not applicable
)

// NormalizeCVSS3 maps a CVSSv3 base score to a normalized Severity.
// Thresholds follow the NVD / FIRST CVSSv3 severity rating scale.
func NormalizeCVSS3(score float64) Severity {
	switch {
	case score >= 9.0:
		return SeverityCritical
	case score >= 7.0:
		return SeverityHigh
	case score >= 4.0:
		return SeverityMedium
	case score > 0:
		return SeverityLow
	default:
		return SeverityInformational
	}
}

// NormalizeCVSS2 maps a CVSSv2 base score to a normalized Severity.
// CVSSv2 has no Critical tier; scores 7.0+ map to High per NIST guidelines.
func NormalizeCVSS2(score float64) Severity {
	switch {
	case score >= 7.0:
		return SeverityHigh
	case score >= 4.0:
		return SeverityMedium
	case score > 0:
		return SeverityLow
	default:
		return SeverityInformational
	}
}

// NormalizeSeverityLabel maps provider-specific severity strings to Severity.
// Handles Qualys (numeric 1-5), Tenable (word), and CrowdStrike (word) conventions.
func NormalizeSeverityLabel(label string) Severity {
	switch label {
	case "5", "CRITICAL", "Critical", "critical":
		return SeverityCritical
	case "4", "HIGH", "High", "high":
		return SeverityHigh
	case "3", "MEDIUM", "Medium", "medium":
		return SeverityMedium
	case "2", "LOW", "Low", "low":
		return SeverityLow
	case "1", "INFO", "Info", "info", "INFORMATIONAL", "Informational", "informational":
		return SeverityInformational
	default:
		return SeverityNone
	}
}

// QueryOptions mirrors connector.QueryOptions for consistency.
type QueryOptions = connector.QueryOptions

// VulnFinding is the normalized representation of a vulnerability finding
// produced by any vulnerability scanner (Qualys, Tenable, etc.).
type VulnFinding struct {
	// ID is the provider-assigned finding identifier.
	ID string `json:"id"`
	// Source identifies the originating connector (e.g. "qualys", "tenable").
	Source string `json:"source"`
	// HostID is the provider's internal host/asset identifier.
	HostID string `json:"host_id"`
	// HostIP is the IPv4/IPv6 address of the affected host.
	HostIP string `json:"host_ip"`
	// HostName is the FQDN or NetBIOS name of the affected host.
	HostName string `json:"host_name"`
	// QID is the Qualys QID (empty for other providers).
	QID string `json:"qid,omitempty"`
	// PluginID is the Tenable plugin ID (empty for other providers).
	PluginID string `json:"plugin_id,omitempty"`
	// CVEIDs contains all associated CVE identifiers.
	CVEIDs []string `json:"cve_ids,omitempty"`
	// CVSS2Score is the CVSSv2 base score (0-10). Zero means unscored.
	CVSS2Score float64 `json:"cvss2_score,omitempty"`
	// CVSS3Score is the CVSSv3 base score (0-10). Zero means unscored.
	CVSS3Score float64 `json:"cvss3_score,omitempty"`
	// Severity is the normalized severity derived from CVSS scores or provider labels.
	Severity Severity `json:"severity"`
	// Title is a brief description of the vulnerability.
	Title string `json:"title"`
	// Description is the full vulnerability description.
	Description string `json:"description"`
	// Solution is the recommended remediation step.
	Solution string `json:"solution,omitempty"`
	// RemediationStatus is the provider-reported fix/patch status.
	RemediationStatus string `json:"remediation_status,omitempty"`
	// FirstDetected is when the finding was first observed.
	FirstDetected time.Time `json:"first_detected"`
	// LastDetected is when the finding was most recently observed.
	LastDetected time.Time `json:"last_detected"`
	// RawData preserves provider-specific fields for downstream consumers.
	RawData map[string]string `json:"raw_data,omitempty"`
}

// EndpointDevice is the normalized representation of a managed endpoint or
// cloud workload returned by an endpoint security platform.
type EndpointDevice struct {
	// ID is the provider-assigned device identifier.
	ID string `json:"id"`
	// Source identifies the originating connector (e.g. "crowdstrike", "qualys").
	Source string `json:"source"`
	// Hostname is the device FQDN or short name.
	Hostname string `json:"hostname"`
	// IPAddresses contains all known IPv4/IPv6 addresses.
	IPAddresses []string `json:"ip_addresses,omitempty"`
	// OperatingSystem is the OS name + version string.
	OperatingSystem string `json:"operating_system,omitempty"`
	// AgentVersion is the installed sensor/agent version.
	AgentVersion string `json:"agent_version,omitempty"`
	// Status is the provider-reported device status (e.g. "online", "contained").
	Status string `json:"status,omitempty"`
	// LastSeen is the most recent check-in time.
	LastSeen time.Time `json:"last_seen"`
	// Tags contains platform labels or groups applied to the device.
	Tags []string `json:"tags,omitempty"`
	// RawData preserves provider-specific fields.
	RawData map[string]string `json:"raw_data,omitempty"`
}

// DetectionEvent is the normalized representation of a CrowdStrike endpoint
// detection or incident summary.
type DetectionEvent struct {
	// ID is the CrowdStrike detection or incident identifier.
	ID string `json:"id"`
	// Source is always "crowdstrike".
	Source string `json:"source"`
	// Severity is the normalized detection severity.
	Severity Severity `json:"severity"`
	// Status is the detection lifecycle status (e.g. "new", "in_progress", "closed").
	Status string `json:"status"`
	// Title is a brief summary of the detection.
	Title string `json:"title"`
	// Description contains tactic, technique, and indicator details.
	Description string `json:"description"`
	// Tactics lists MITRE ATT&CK tactic names.
	Tactics []string `json:"tactics,omitempty"`
	// Techniques lists MITRE ATT&CK technique IDs.
	Techniques []string `json:"techniques,omitempty"`
	// HostIDs lists the device IDs involved in this detection.
	HostIDs []string `json:"host_ids,omitempty"`
	// StartTime is when the detection was first observed.
	StartTime time.Time `json:"start_time"`
	// EndTime is when the detection was last updated.
	EndTime time.Time `json:"end_time"`
	// RawData preserves provider-specific fields.
	RawData map[string]string `json:"raw_data,omitempty"`
}

// HealthStatus re-exports connector.HealthStatus for package consumers.
type HealthStatus = connector.HealthStatus

// VulnConnector is the interface all vulnerability / endpoint adapters must implement.
// Adapters that focus on a subset of capabilities (e.g. CrowdStrike does not expose
// traditional CVE findings) may return empty slices for methods outside their scope.
type VulnConnector interface {
	// Name returns the unique connector identifier.
	Name() string

	// FetchVulnerabilities returns normalized vulnerability findings.
	// For CrowdStrike this returns detection events mapped to the finding schema.
	FetchVulnerabilities(ctx context.Context, opts QueryOptions) ([]VulnFinding, error)

	// FetchEndpoints returns normalized endpoint / device records.
	FetchEndpoints(ctx context.Context, opts QueryOptions) ([]EndpointDevice, error)

	// Health reports the connectivity status of this connector.
	Health(ctx context.Context) HealthStatus
}

// VulnRegistry holds all registered VulnConnectors.
type VulnRegistry struct {
	connectors map[string]VulnConnector
}

// NewVulnRegistry creates an empty registry.
func NewVulnRegistry() *VulnRegistry {
	return &VulnRegistry{connectors: make(map[string]VulnConnector)}
}

// Register adds a connector to the registry.
func (r *VulnRegistry) Register(c VulnConnector) {
	r.connectors[c.Name()] = c
}

// Get returns a connector by name.
func (r *VulnRegistry) Get(name string) (VulnConnector, bool) {
	c, ok := r.connectors[name]
	return c, ok
}

// List returns all registered connector names.
func (r *VulnRegistry) List() []string {
	names := make([]string, 0, len(r.connectors))
	for n := range r.connectors {
		names = append(names, n)
	}
	return names
}

