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

// SplunkConfig holds Splunk REST API connection settings.
type SplunkConfig struct {
	// BaseURL is the Splunk instance URL, e.g. https://splunk.example.com:8089
	BaseURL string
	// Token is the Splunk auth token (HEC or bearer).
	Token string
	// SavedSearch is the Splunk saved search name to run.
	SavedSearch string
	// MockMode returns synthetic data without real API calls.
	MockMode bool
}

// SplunkAdapter fetches security events from Splunk via the REST API.
type SplunkAdapter struct {
	config SplunkConfig
	client *http.Client
}

// NewSplunkAdapter creates a ready-to-use Splunk connector.
func NewSplunkAdapter(cfg SplunkConfig) *SplunkAdapter {
	return &SplunkAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements connector.Connector.
func (s *SplunkAdapter) Name() string { return "splunk" }

// FetchLogs retrieves results from a Splunk saved search.
// In sandbox / mock mode it returns deterministic synthetic records.
func (s *SplunkAdapter) FetchLogs(ctx context.Context, opts connector.QueryOptions) ([]connector.LogEntry, error) {
	if s.config.MockMode {
		return s.mockLogs(opts), nil
	}
	return s.fetchFromAPI(ctx, opts)
}

// Health reports whether the Splunk REST endpoint is reachable.
func (s *SplunkAdapter) Health(ctx context.Context) connector.HealthStatus {
	status := connector.HealthStatus{
		Connector:   s.Name(),
		LastChecked: time.Now().UTC(),
	}
	if s.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}

	url := fmt.Sprintf("%s/services/server/info?output_mode=json", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
	req.Header.Set("Authorization", "Bearer "+s.config.Token)
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

// fetchFromAPI runs the Splunk saved search via the REST API.
func (s *SplunkAdapter) fetchFromAPI(ctx context.Context, opts connector.QueryOptions) ([]connector.LogEntry, error) {
	savedSearch := s.config.SavedSearch
	if savedSearch == "" {
		savedSearch = "NIS2 Security Events"
	}

	url := fmt.Sprintf(
		"%s/servicesNS/nobody/search/saved/searches/%s/history?output_mode=json&count=%d",
		s.config.BaseURL,
		savedSearch,
		effectiveLimit(opts),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("splunk: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("splunk: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("splunk: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("splunk: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Entry []struct {
			Content struct {
				EventCount int    `json:"eventCount"`
				Search     string `json:"search"`
			} `json:"content"`
			Name string `json:"name"`
		} `json:"entry"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("splunk: parse response: %w", err)
	}

	var logs []connector.LogEntry
	for _, entry := range result.Entry {
		logs = append(logs, connector.LogEntry{
			Timestamp:   time.Now().UTC(),
			Source:      "splunk",
			Severity:    "info",
			EventID:     entry.Name,
			Title:       fmt.Sprintf("Splunk saved search: %s", entry.Name),
			Description: fmt.Sprintf("eventCount=%d", entry.Content.EventCount),
		})
	}
	return logs, nil
}

// mockLogs returns deterministic synthetic Splunk events.
func (s *SplunkAdapter) mockLogs(opts connector.QueryOptions) []connector.LogEntry {
	limit := effectiveLimit(opts)
	now := time.Now().UTC()
	logs := make([]connector.LogEntry, 0, limit)
	severities := []string{"critical", "high", "medium", "low"}
	for i := 0; i < limit; i++ {
		logs = append(logs, connector.LogEntry{
			Timestamp:   now.Add(-time.Duration(i*5) * time.Minute),
			Source:      "splunk",
			Severity:    severities[i%len(severities)],
			EventID:     fmt.Sprintf("SPLUNK-MOCK-%04d", i+1),
			Title:       fmt.Sprintf("Mock Splunk Event #%d", i+1),
			Description: fmt.Sprintf("Synthetic security event #%d from Splunk saved search '%s'", i+1, s.config.SavedSearch),
			RawData: map[string]string{
				"host":   "splunk-indexer-01",
				"source": "WinEventLog:Security",
			},
		})
	}
	return logs
}

func effectiveLimit(opts connector.QueryOptions) int {
	if opts.Limit > 0 && opts.Limit <= 100 {
		return opts.Limit
	}
	return 10
}

