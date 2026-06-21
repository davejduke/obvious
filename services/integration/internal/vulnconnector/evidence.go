package vulnconnector

import (
	"fmt"
	"time"
)

// EvidenceCategory classifies the kind of evidence item.
type EvidenceCategory string

const (
	EvidenceCategoryVulnerability EvidenceCategory = "vulnerability"
	EvidenceCategoryEndpoint      EvidenceCategory = "endpoint"
	EvidenceCategoryDetection     EvidenceCategory = "detection"
)

// EvidenceItem is the AIAUDITOR-internal evidence model produced by all
// vulnerability/endpoint connectors. It carries the normalized fields
// consumed by the reasoning engine and the audit trail.
type EvidenceItem struct {
	// ID is a stable evidence identifier composed as "<source>/<original-id>".
	ID string `json:"id"`
	// Source is the connector that produced this item.
	Source string `json:"source"`
	// Category classifies the item (vulnerability, endpoint, detection).
	Category EvidenceCategory `json:"category"`
	// Severity is the normalized severity.
	Severity Severity `json:"severity"`
	// Title is the human-readable summary.
	Title string `json:"title"`
	// Description is the full narrative.
	Description string `json:"description"`
	// CollectedAt is when the evidence was gathered.
	CollectedAt time.Time `json:"collected_at"`
	// Metadata carries category-specific key/value pairs for downstream processing.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// VulnFindingToEvidence converts a VulnFinding into an EvidenceItem.
func VulnFindingToEvidence(f VulnFinding) EvidenceItem {
	meta := map[string]interface{}{
		"host_id":            f.HostID,
		"host_ip":            f.HostIP,
		"host_name":          f.HostName,
		"cvss3_score":        f.CVSS3Score,
		"cvss2_score":        f.CVSS2Score,
		"remediation_status": f.RemediationStatus,
		"first_detected":     f.FirstDetected.Format(time.RFC3339),
		"last_detected":      f.LastDetected.Format(time.RFC3339),
	}
	if len(f.CVEIDs) > 0 {
		meta["cve_ids"] = f.CVEIDs
	}
	if f.QID != "" {
		meta["qid"] = f.QID
	}
	if f.PluginID != "" {
		meta["plugin_id"] = f.PluginID
	}
	if f.Solution != "" {
		meta["solution"] = f.Solution
	}
	return EvidenceItem{
		ID:          fmt.Sprintf("%s/%s", f.Source, f.ID),
		Source:      f.Source,
		Category:    EvidenceCategoryVulnerability,
		Severity:    f.Severity,
		Title:       f.Title,
		Description: f.Description,
		CollectedAt: time.Now().UTC(),
		Metadata:    meta,
	}
}

// EndpointDeviceToEvidence converts an EndpointDevice into an EvidenceItem.
func EndpointDeviceToEvidence(d EndpointDevice) EvidenceItem {
	meta := map[string]interface{}{
		"hostname":         d.Hostname,
		"operating_system": d.OperatingSystem,
		"agent_version":    d.AgentVersion,
		"status":           d.Status,
		"last_seen":        d.LastSeen.Format(time.RFC3339),
	}
	if len(d.IPAddresses) > 0 {
		meta["ip_addresses"] = d.IPAddresses
	}
	if len(d.Tags) > 0 {
		meta["tags"] = d.Tags
	}
	return EvidenceItem{
		ID:          fmt.Sprintf("%s/%s", d.Source, d.ID),
		Source:      d.Source,
		Category:    EvidenceCategoryEndpoint,
		Severity:    SeverityInformational,
		Title:       fmt.Sprintf("Endpoint: %s", d.Hostname),
		Description: fmt.Sprintf("Managed endpoint %s (%s) running %s, status: %s", d.Hostname, d.Source, d.OperatingSystem, d.Status),
		CollectedAt: time.Now().UTC(),
		Metadata:    meta,
	}
}

// DetectionEventToEvidence converts a DetectionEvent into an EvidenceItem.
func DetectionEventToEvidence(e DetectionEvent) EvidenceItem {
	meta := map[string]interface{}{
		"status":     e.Status,
		"start_time": e.StartTime.Format(time.RFC3339),
		"end_time":   e.EndTime.Format(time.RFC3339),
	}
	if len(e.Tactics) > 0 {
		meta["tactics"] = e.Tactics
	}
	if len(e.Techniques) > 0 {
		meta["techniques"] = e.Techniques
	}
	if len(e.HostIDs) > 0 {
		meta["host_ids"] = e.HostIDs
	}
	return EvidenceItem{
		ID:          fmt.Sprintf("%s/%s", e.Source, e.ID),
		Source:      e.Source,
		Category:    EvidenceCategoryDetection,
		Severity:    e.Severity,
		Title:       e.Title,
		Description: e.Description,
		CollectedAt: time.Now().UTC(),
		Metadata:    meta,
	}
}

// VulnFindingsToEvidence bulk-converts a slice of VulnFinding to EvidenceItems.
func VulnFindingsToEvidence(findings []VulnFinding) []EvidenceItem {
	result := make([]EvidenceItem, len(findings))
	for i, f := range findings {
		result[i] = VulnFindingToEvidence(f)
	}
	return result
}

// EndpointDevicesToEvidence bulk-converts a slice of EndpointDevice to EvidenceItems.
func EndpointDevicesToEvidence(devices []EndpointDevice) []EvidenceItem {
	result := make([]EvidenceItem, len(devices))
	for i, d := range devices {
		result[i] = EndpointDeviceToEvidence(d)
	}
	return result
}

// DetectionEventsToEvidence bulk-converts a slice of DetectionEvent to EvidenceItems.
func DetectionEventsToEvidence(events []DetectionEvent) []EvidenceItem {
	result := make([]EvidenceItem, len(events))
	for i, e := range events {
		result[i] = DetectionEventToEvidence(e)
	}
	return result
}

