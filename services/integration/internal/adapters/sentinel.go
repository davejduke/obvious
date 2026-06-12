// Package adapters contains SIEM connector implementations.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/connector"
)

// SentinelConfig holds Azure Monitor / Sentinel connection settings.
type SentinelConfig struct {
	// WorkspaceID is the Log Analytics workspace GUID.
	WorkspaceID string
	// TenantID is the Azure AD tenant GUID.
	TenantID string
	// ClientID is the service principal application ID.
	ClientID string
	// ClientSecret is the service principal secret (populated from env/vault).
	ClientSecret string
	// BaseURL overrides the default Azure Monitor endpoint (for sandbox/mock).
	BaseURL string
	// MockMode returns synthetic data without making real API calls.
	MockMode bool
}

// SentinelAdapter fetches security alerts and logs from Microsoft Sentinel
// via the Azure Monitor Log Analytics REST API.
type SentinelAdapter struct {
	config SentinelConfig
	client *http.Client
}

// NewSentinelAdapter creates a ready-to-use Sentinel connector.
func NewSentinelAdapter(cfg SentinelConfig) *SentinelAdapter {
	return &SentinelAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements connector.Connector.
func (s *SentinelAdapter) Name() string { return "sentinel" }

// FetchLogs retrieves security alerts from Azure Monitor Log Analytics.
// In sandbox / mock mode it returns deterministic synthetic records.
func (s *SentinelAdapter) FetchLogs(ctx context.Context, opts connector.QueryOptions) ([]connector.LogEntry, error) {
	if s.config.MockMode {
		return s.mockLogs(opts), nil
	}
	return s.fetchFromAPI(ctx, opts)
}

// Health reports whether the Sentinel endpoint is reachable.
func (s *SentinelAdapter) Health(ctx context.Context) connector.HealthStatus {
	status := connector.HealthStatus{
		Connector:   s.Name(),
		LastChecked: time.Now().UTC(),
	}
	if s.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}

	url := fmt.Sprintf("%s/subscriptions/healthcheck", s.apiBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
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

// fetchFromAPI calls the Azure Monitor query API.
func (s *SentinelAdapter) fetchFromAPI(ctx context.Context, opts connector.QueryOptions) ([]connector.LogEntry, error) {
	query := opts.Query
	if query == "" {
		query = "SecurityAlert | order by TimeGenerated desc | limit 50"
	}

	url := fmt.Sprintf(
		"%s/v1/workspaces/%s/query?query=%s",
		s.apiBase(),
		s.config.WorkspaceID,
		query,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("sentinel: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sentinel: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("sentinel: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sentinel: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Tables []struct {
			Rows [][]interface{} `json:"rows"`
		} `json:"tables"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("sentinel: parse response: %w", err)
	}

	var logs []connector.LogEntry
	for _, table := range result.Tables {
		for _, row := range table.Rows {
			entry := connector.LogEntry{
				Timestamp: time.Now().UTC(),
				Source:    "sentinel",
				Severity:  "unknown",
			}
			if len(row) > 0 {
				if ts, ok := row[0].(string); ok {
					parsed, _ := time.Parse(time.RFC3339, ts)
					entry.Timestamp = parsed
				}
			}
			if len(row) > 1 {
				if title, ok := row[1].(string); ok {
					entry.Title = title
				}
			}
			logs = append(logs, entry)
		}
	}
	return logs, nil
}

// mockLogs returns deterministic synthetic Sentinel alerts.
func (s *SentinelAdapter) mockLogs(opts connector.QueryOptions) []connector.LogEntry {
	limit := opts.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	now := time.Now().UTC()
	logs := make([]connector.LogEntry, 0, limit)
	severities := []string{"High", "Medium", "Low", "Informational"}
	for i := 0; i < limit; i++ {
		logs = append(logs, connector.LogEntry{
			Timestamp:   now.Add(-time.Duration(i) * time.Minute),
			Source:      "sentinel",
			Severity:    severities[i%len(severities)],
			EventID:     fmt.Sprintf("SENTINEL-MOCK-%04d", i+1),
			Title:       fmt.Sprintf("Mock Sentinel Alert #%d", i+1),
			Description: fmt.Sprintf("Synthetic NIS2 Article 21 finding #%d from Azure Sentinel mock", i+1),
			RawData: map[string]string{
				"WorkspaceID": s.config.WorkspaceID,
				"Source":      "AzureActivity",
			},
		})
	}
	return logs
}

func (s *SentinelAdapter) apiBase() string {
	if s.config.BaseURL != "" {
		return s.config.BaseURL
	}
	return "https://api.loganalytics.io"
}

