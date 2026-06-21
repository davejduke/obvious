// Package adapters contains integration connector implementations.
package adapters

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/grc"
)

// ServiceNowConfig holds ServiceNow instance connection settings.
type ServiceNowConfig struct {
	// BaseURL is the ServiceNow instance URL, e.g. https://dev12345.service-now.com
	BaseURL string
	// Username is the ServiceNow user account (populated from env/vault).
	Username string
	// Password is the ServiceNow user password (populated from env/vault).
	Password string
	// MockMode returns synthetic data without making real API calls.
	MockMode bool
}

// ServiceNowAdapter exports audit findings to ServiceNow GRC tables:
//   - sn_grc_item  — one record per finding (compliance/policy item)
//   - sn_grc_risk  — one record per High/Critical finding (risk register)
//   - cmdb_ci      — referenced by each risk entry (affected configuration item)
//
// All writes use the ServiceNow Table API (/api/now/table/{tableName}).
type ServiceNowAdapter struct {
	config ServiceNowConfig
	client *http.Client
}

// NewServiceNowAdapter creates a ready-to-use ServiceNow GRC connector.
func NewServiceNowAdapter(cfg ServiceNowConfig) *ServiceNowAdapter {
	return &ServiceNowAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements grc.GRCConnector.
func (s *ServiceNowAdapter) Name() string { return "servicenow" }

// ExportFindings maps findings to ServiceNow GRC records.
// In mock mode it returns deterministic synthetic records; in live mode it
// POSTs to the ServiceNow Table API.
func (s *ServiceNowAdapter) ExportFindings(ctx context.Context, findings []grc.Finding) (*grc.ExportResult, error) {
	if s.config.MockMode {
		return s.mockExport(findings), nil
	}
	return s.liveExport(ctx, findings)
}

// Health reports whether the ServiceNow instance is reachable.
func (s *ServiceNowAdapter) Health(ctx context.Context) grc.HealthStatus {
	status := grc.HealthStatus{
		Connector:   s.Name(),
		LastChecked: time.Now().UTC(),
	}
	if s.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}

	url := fmt.Sprintf("%s/api/now/table/sys_properties?sysparm_limit=1", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
	req.SetBasicAuth(s.config.Username, s.config.Password)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		status.Message = "connectivity error: " + err.Error()
		return status
	}
	resp.Body.Close()
	status.Healthy = resp.StatusCode < 500
	status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return status
}

// liveExport calls the ServiceNow Table API to create GRC records.
func (s *ServiceNowAdapter) liveExport(ctx context.Context, findings []grc.Finding) (*grc.ExportResult, error) {
	result := &grc.ExportResult{
		Connector:  s.Name(),
		ExportedAt: time.Now().UTC(),
	}

	for _, f := range findings {
		// Create sn_grc_item record
		item, err := s.createGRCItem(ctx, f)
		if err != nil {
			return nil, fmt.Errorf("servicenow: create grc item for %s: %w", f.ID, err)
		}
		result.GRCItems = append(result.GRCItems, *item)

		// Create sn_grc_risk for High and Critical findings
		if f.Severity == grc.SeverityCritical || f.Severity == grc.SeverityHigh {
			risk, err := s.createRiskEntry(ctx, f, item.SysID)
			if err != nil {
				return nil, fmt.Errorf("servicenow: create risk entry for %s: %w", f.ID, err)
			}
			result.RiskEntries = append(result.RiskEntries, *risk)
		}

		// Create remediation task for all findings with a recommendation
		if f.Recommendation != "" {
			task, err := s.createRemediationTask(ctx, f, item.SysID)
			if err != nil {
				return nil, fmt.Errorf("servicenow: create remediation task for %s: %w", f.ID, err)
			}
			result.RemediationTasks = append(result.RemediationTasks, *task)
		}
	}

	result.TotalFindings = len(findings)
	return result, nil
}

// createGRCItem POSTs a record to the sn_grc_item table.
func (s *ServiceNowAdapter) createGRCItem(ctx context.Context, f grc.Finding) (*grc.GRCItem, error) {
	payload := map[string]interface{}{
		"name":        fmt.Sprintf("[AIAUDITOR] %s", f.Title),
		"description": f.Description,
		"state":       "open",
		"category":    "compliance",
		"priority":    severityToSNPriority(f.Severity),
		"u_source_ref": f.ID,
		"u_control_ref": f.ControlRef,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/now/table/sn_grc_item", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.config.Username, s.config.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result struct {
			SysID string `json:"sys_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &grc.GRCItem{
		SysID:       result.Result.SysID,
		Name:        fmt.Sprintf("[AIAUDITOR] %s", f.Title),
		Description: f.Description,
		State:       "open",
		Category:    "compliance",
		Priority:    severityToSNPriority(f.Severity),
		SourceRef:   f.ID,
		ControlRef:  f.ControlRef,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// createRiskEntry POSTs a record to the sn_grc_risk table.
func (s *ServiceNowAdapter) createRiskEntry(ctx context.Context, f grc.Finding, grcItemSysID string) (*grc.RiskEntry, error) {
	cmdbCI := resolveCMDBCI(f.AssetRef)
	payload := map[string]interface{}{
		"name":         fmt.Sprintf("[AIAUDITOR Risk] %s", f.Title),
		"description":  f.Description,
		"risk_score":   severityToRiskScore(f.Severity),
		"category":     "information_security",
		"treatment":    "mitigate",
		"cmdb_ci":      cmdbCI,
		"u_source_ref": f.ID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/now/table/sn_grc_risk", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.config.Username, s.config.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result struct {
			SysID string `json:"sys_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &grc.RiskEntry{
		SysID:       result.Result.SysID,
		Name:        fmt.Sprintf("[AIAUDITOR Risk] %s", f.Title),
		Description: f.Description,
		RiskScore:   severityToRiskScore(f.Severity),
		Category:    "information_security",
		Treatment:   "mitigate",
		CMDBCI:      cmdbCI,
		SourceRef:   f.ID,
		CreatedAt:   time.Now().UTC(),
	}, nil
}

// createRemediationTask POSTs a remediation task linked to the GRC item.
func (s *ServiceNowAdapter) createRemediationTask(ctx context.Context, f grc.Finding, grcItemSysID string) (*grc.RemediationTask, error) {
	dueDate := time.Now().UTC().Add(dueDateForSeverity(f.Severity))
	payload := map[string]interface{}{
		"name":           fmt.Sprintf("[AIAUDITOR Remediation] %s", f.Title),
		"description":    f.Recommendation,
		"state":          "open",
		"priority":       severityToSNPriority(f.Severity),
		"assigned_to":    "aiauditor_integration",
		"due_date":       dueDate.Format(time.RFC3339),
		"u_grc_item_ref": grcItemSysID,
		"u_source_ref":   f.ID,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/api/now/table/sn_grc_task", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(s.config.Username, s.config.Password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Result struct {
			SysID string `json:"sys_id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, err
	}

	return &grc.RemediationTask{
		SysID:       result.Result.SysID,
		Name:        fmt.Sprintf("[AIAUDITOR Remediation] %s", f.Title),
		Description: f.Recommendation,
		State:       "open",
		Priority:    severityToSNPriority(f.Severity),
		AssignedTo:  "aiauditor_integration",
		DueDate:     dueDate,
		GRCItemRef:  grcItemSysID,
		SourceRef:   f.ID,
	}, nil
}

// mockExport returns deterministic synthetic GRC records without API calls.
func (s *ServiceNowAdapter) mockExport(findings []grc.Finding) *grc.ExportResult {
	result := &grc.ExportResult{
		Connector:  s.Name(),
		ExportedAt: time.Now().UTC(),
	}

	for i, f := range findings {
		sysIDBase := fmt.Sprintf("mock%08x", i+1)

		// Every finding gets a sn_grc_item record
		item := grc.GRCItem{
			SysID:       "grc" + sysIDBase,
			Name:        fmt.Sprintf("[AIAUDITOR] %s", f.Title),
			Description: f.Description,
			State:       "open",
			Category:    "compliance",
			Priority:    severityToSNPriority(f.Severity),
			SourceRef:   f.ID,
			ControlRef:  f.ControlRef,
			CreatedAt:   time.Now().UTC(),
		}
		result.GRCItems = append(result.GRCItems, item)

		// High/Critical findings get a sn_grc_risk entry with cmdb_ci reference
		if f.Severity == grc.SeverityCritical || f.Severity == grc.SeverityHigh {
			cmdbCI := resolveCMDBCI(f.AssetRef)
			risk := grc.RiskEntry{
				SysID:       "risk" + sysIDBase,
				Name:        fmt.Sprintf("[AIAUDITOR Risk] %s", f.Title),
				Description: f.Description,
				RiskScore:   severityToRiskScore(f.Severity),
				Category:    "information_security",
				Treatment:   "mitigate",
				Owner:       "aiauditor_integration",
				CMDBCI:      cmdbCI,
				SourceRef:   f.ID,
				CreatedAt:   time.Now().UTC(),
			}
			result.RiskEntries = append(result.RiskEntries, risk)
		}

		// All findings with a recommendation get a remediation task
		if f.Recommendation != "" {
			task := grc.RemediationTask{
				SysID:       "task" + sysIDBase,
				Name:        fmt.Sprintf("[AIAUDITOR Remediation] %s", f.Title),
				Description: f.Recommendation,
				State:       "open",
				Priority:    severityToSNPriority(f.Severity),
				AssignedTo:  "aiauditor_integration",
				DueDate:     time.Now().UTC().Add(dueDateForSeverity(f.Severity)),
				GRCItemRef:  item.SysID,
				SourceRef:   f.ID,
			}
			result.RemediationTasks = append(result.RemediationTasks, task)
		}
	}

	result.TotalFindings = len(findings)
	return result
}

// severityToSNPriority maps AIAUDITOR severity to ServiceNow priority values.
// ServiceNow uses 1=Critical, 2=High, 3=Moderate, 4=Low, 5=Planning.
func severityToSNPriority(s grc.Severity) string {
	switch s {
	case grc.SeverityCritical:
		return "1"
	case grc.SeverityHigh:
		return "2"
	case grc.SeverityMedium:
		return "3"
	case grc.SeverityLow:
		return "4"
	default:
		return "5"
	}
}

// severityToRiskScore maps severity to a 0–100 risk score.
func severityToRiskScore(s grc.Severity) int {
	switch s {
	case grc.SeverityCritical:
		return 90
	case grc.SeverityHigh:
		return 70
	case grc.SeverityMedium:
		return 50
	case grc.SeverityLow:
		return 25
	default:
		return 10
	}
}

// dueDateForSeverity returns the remediation SLA window based on severity.
func dueDateForSeverity(s grc.Severity) time.Duration {
	switch s {
	case grc.SeverityCritical:
		return 7 * 24 * time.Hour // 7 days
	case grc.SeverityHigh:
		return 30 * 24 * time.Hour // 30 days
	case grc.SeverityMedium:
		return 90 * 24 * time.Hour // 90 days
	default:
		return 180 * 24 * time.Hour // 180 days
	}
}

// resolveCMDBCI resolves a cmdb_ci reference from the finding's asset reference.
// When no asset ref is provided, a default CI is used for grouping.
func resolveCMDBCI(assetRef string) string {
	if assetRef != "" {
		return assetRef
	}
	return "aiauditor_default_ci"
}
