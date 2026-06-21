// CrowdStrike Falcon connector — endpoint detection events, device inventory, incident summaries.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/vulnconnector"
)

// CrowdStrikeConfig holds CrowdStrike Falcon API connection settings.
type CrowdStrikeConfig struct {
	// BaseURL is the Falcon API base URL (tenant-specific cloud endpoint).
	BaseURL string
	// ClientID is the Falcon API OAuth2 client ID.
	ClientID string
	// ClientSecret is the Falcon API OAuth2 client secret (populated from env/vault).
	ClientSecret string
	// MockMode returns synthetic data without making real API calls.
	MockMode bool
}

// CrowdStrikeAdapter fetches endpoint detection events, device inventory, and
// incident summaries from the CrowdStrike Falcon platform via the Falcon API.
type CrowdStrikeAdapter struct {
	config CrowdStrikeConfig
	client *http.Client
}

// NewCrowdStrikeAdapter creates a ready-to-use CrowdStrike connector.
func NewCrowdStrikeAdapter(cfg CrowdStrikeConfig) *CrowdStrikeAdapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.crowdstrike.com"
	}
	return &CrowdStrikeAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements vulnconnector.VulnConnector.
func (cs *CrowdStrikeAdapter) Name() string { return "crowdstrike" }

// FetchVulnerabilities returns detection events and incident summaries mapped
// to the VulnFinding schema so downstream evidence consumers have a uniform type.
func (cs *CrowdStrikeAdapter) FetchVulnerabilities(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.VulnFinding, error) {
	if cs.config.MockMode {
		return cs.mockDetectionsAsFindings(opts), nil
	}
	return cs.fetchDetectionsAsFindings(ctx, opts)
}

// FetchEndpoints returns the CrowdStrike Falcon device inventory.
func (cs *CrowdStrikeAdapter) FetchEndpoints(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.EndpointDevice, error) {
	if cs.config.MockMode {
		return cs.mockEndpoints(opts), nil
	}
	return cs.fetchDevicesFromAPI(ctx, opts)
}

// FetchDetections returns CrowdStrike detections as DetectionEvent records.
// This is the CrowdStrike-native method; use FetchVulnerabilities for the
// normalized VulnFinding representation.
func (cs *CrowdStrikeAdapter) FetchDetections(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.DetectionEvent, error) {
	if cs.config.MockMode {
		return cs.mockDetections(opts), nil
	}
	return cs.fetchDetectionsFromAPI(ctx, opts)
}

// Health reports whether the Falcon API is reachable.
func (cs *CrowdStrikeAdapter) Health(ctx context.Context) vulnconnector.HealthStatus {
	status := vulnconnector.HealthStatus{
		Connector:   cs.Name(),
		LastChecked: time.Now().UTC(),
	}
	if cs.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}
	// CrowdStrike OAuth2 token endpoint doubles as a connectivity check.
	url := cs.config.BaseURL + "/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
	resp, err := cs.client.Do(req)
	if err != nil {
		status.Message = "connectivity error: " + err.Error()
		return status
	}
	resp.Body.Close()
	// 405 Method Not Allowed is expected for HEAD — API is reachable.
	status.Healthy = resp.StatusCode < 500
	status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return status
}

// fetchDetectionsFromAPI queries Falcon detections via the API.
func (cs *CrowdStrikeAdapter) fetchDetectionsFromAPI(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.DetectionEvent, error) {
	limit := csEffectiveLimit(opts)
	// Step 1: query detection IDs
	url := fmt.Sprintf("%s/detects/queries/detects/v1?limit=%d", cs.config.BaseURL, limit)
	token, err := cs.bearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: auth: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: build query request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := cs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: query request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: read query body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crowdstrike: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ids struct {
		Resources []string `json:"resources"`
	}
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("crowdstrike: parse ids: %w", err)
	}
	if len(ids.Resources) == 0 {
		return []vulnconnector.DetectionEvent{}, nil
	}

	// Step 2: summarize detections by ID
	sumURL := fmt.Sprintf("%s/detects/entities/summaries/GET/v1", cs.config.BaseURL)
	payload := map[string][]string{"ids": ids.Resources}
	payloadBytes, _ := json.Marshal(payload)
	sumReq, err := http.NewRequestWithContext(ctx, http.MethodPost, sumURL,
		io.NopCloser(strings.NewReader(string(payloadBytes))))
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: build summary request: %w", err)
	}
	sumReq.Header.Set("Authorization", "Bearer "+token)
	sumReq.Header.Set("Content-Type", "application/json")
	sumResp, err := cs.client.Do(sumReq)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: summary request: %w", err)
	}
	defer sumResp.Body.Close()
	sumBody, err := io.ReadAll(sumResp.Body)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: read summary body: %w", err)
	}
	if sumResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crowdstrike: HTTP %d: %s", sumResp.StatusCode, string(sumBody))
	}

	var summaries struct {
		Resources []csDetection `json:"resources"`
	}
	if err := json.Unmarshal(sumBody, &summaries); err != nil {
		return nil, fmt.Errorf("crowdstrike: parse summaries: %w", err)
	}

	return csDetectionsToEvents(summaries.Resources), nil
}

// fetchDetectionsAsFindings converts CrowdStrike detections to VulnFinding.
func (cs *CrowdStrikeAdapter) fetchDetectionsAsFindings(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.VulnFinding, error) {
	events, err := cs.fetchDetectionsFromAPI(ctx, opts)
	if err != nil {
		return nil, err
	}
	return csEventsToFindings(events), nil
}

// fetchDevicesFromAPI retrieves the Falcon device inventory.
func (cs *CrowdStrikeAdapter) fetchDevicesFromAPI(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.EndpointDevice, error) {
	limit := csEffectiveLimit(opts)
	token, err := cs.bearerToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: auth: %w", err)
	}

	// Step 1: list device IDs
	url := fmt.Sprintf("%s/devices/queries/devices/v1?limit=%d", cs.config.BaseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: build device query request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := cs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: device query request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: read device ids body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crowdstrike: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var ids struct {
		Resources []string `json:"resources"`
	}
	if err := json.Unmarshal(body, &ids); err != nil {
		return nil, fmt.Errorf("crowdstrike: parse device ids: %w", err)
	}
	if len(ids.Resources) == 0 {
		return []vulnconnector.EndpointDevice{}, nil
	}

	// Step 2: retrieve device details
	detailURL := fmt.Sprintf("%s/devices/entities/devices/v2?ids=%s", cs.config.BaseURL, strings.Join(ids.Resources, "&ids="))
	detailReq, err := http.NewRequestWithContext(ctx, http.MethodGet, detailURL, nil)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: build device detail request: %w", err)
	}
	detailReq.Header.Set("Authorization", "Bearer "+token)
	detailResp, err := cs.client.Do(detailReq)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: device detail request: %w", err)
	}
	defer detailResp.Body.Close()
	detailBody, err := io.ReadAll(detailResp.Body)
	if err != nil {
		return nil, fmt.Errorf("crowdstrike: read device detail body: %w", err)
	}
	if detailResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("crowdstrike: HTTP %d: %s", detailResp.StatusCode, string(detailBody))
	}

	var deviceResult struct {
		Resources []csDevice `json:"resources"`
	}
	if err := json.Unmarshal(detailBody, &deviceResult); err != nil {
		return nil, fmt.Errorf("crowdstrike: parse device details: %w", err)
	}
	return csDevicesToEndpoints(deviceResult.Resources), nil
}

// bearerToken obtains a short-lived OAuth2 token from the Falcon API.
func (cs *CrowdStrikeAdapter) bearerToken(ctx context.Context) (string, error) {
	url := cs.config.BaseURL + "/oauth2/token"
	body := fmt.Sprintf("client_id=%s&client_secret=%s", cs.config.ClientID, cs.config.ClientSecret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := cs.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("token request failed HTTP %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(respBody, &tok); err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

// -------------------------------------------------------------------
// Mock data
// -------------------------------------------------------------------

// mockDetections returns deterministic synthetic CrowdStrike detections.
func (cs *CrowdStrikeAdapter) mockDetections(opts vulnconnector.QueryOptions) []vulnconnector.DetectionEvent {
	limit := csEffectiveLimit(opts)
	now := time.Now().UTC()

	type detDef struct {
		id         string
		severity   vulnconnector.Severity
		status     string
		title      string
		desc       string
		tactics    []string
		techniques []string
		hosts      []string
	}
	defs := []detDef{
		{"ldt:aaa001", vulnconnector.SeverityCritical, "new",
			"Ransomware Execution Detected",
			"Falcon detected active ransomware execution on endpoint. File encryption behavior observed.",
			[]string{"Impact"}, []string{"T1486"}, []string{"cs-dev-001"}},
		{"ldt:aaa002", vulnconnector.SeverityHigh, "in_progress",
			"Credential Dumping via LSASS",
			"Process attempting to read LSASS memory for credential harvesting.",
			[]string{"Credential Access"}, []string{"T1003.001"}, []string{"cs-dev-002"}},
		{"ldt:aaa003", vulnconnector.SeverityHigh, "new",
			"Lateral Movement via Pass-the-Hash",
			"Hash-based authentication detected across multiple hosts.",
			[]string{"Lateral Movement"}, []string{"T1550.002"}, []string{"cs-dev-001", "cs-dev-003"}},
		{"ldt:aaa004", vulnconnector.SeverityMedium, "closed",
			"Suspicious PowerShell Execution",
			"PowerShell invoked with encoded command from unusual parent process.",
			[]string{"Execution"}, []string{"T1059.001"}, []string{"cs-dev-004"}},
		{"ldt:aaa005", vulnconnector.SeverityMedium, "in_progress",
			"C2 Beacon Traffic Detected",
			"Periodic outbound connections to known C2 infrastructure.",
			[]string{"Command and Control"}, []string{"T1071.001"}, []string{"cs-dev-002"}},
		{"ldt:aaa006", vulnconnector.SeverityLow, "closed",
			"Scheduled Task Created for Persistence",
			"A new Windows Scheduled Task was registered for persistence by a user-space process.",
			[]string{"Persistence"}, []string{"T1053.005"}, []string{"cs-dev-003"}},
		{"ldt:aaa007", vulnconnector.SeverityCritical, "new",
			"Zero-Day Exploit Attempt Blocked",
			"Exploit prevention blocked an attempt to exploit a zero-day vulnerability in a browser component.",
			[]string{"Initial Access"}, []string{"T1189"}, []string{"cs-dev-005"}},
		{"ldt:aaa008", vulnconnector.SeverityHigh, "new",
			"Data Exfiltration via Cloud Storage",
			"Large volume of files transferred to external cloud storage endpoint.",
			[]string{"Exfiltration"}, []string{"T1567.002"}, []string{"cs-dev-001"}},
	}

	events := make([]vulnconnector.DetectionEvent, 0, limit)
	for i := 0; i < limit && i < len(defs); i++ {
		d := defs[i]
		events = append(events, vulnconnector.DetectionEvent{
			ID:          d.id,
			Source:      "crowdstrike",
			Severity:    d.severity,
			Status:      d.status,
			Title:       d.title,
			Description: d.desc,
			Tactics:     d.tactics,
			Techniques:  d.techniques,
			HostIDs:     d.hosts,
			StartTime:   now.Add(-time.Duration(i+1) * 2 * time.Hour),
			EndTime:     now.Add(-time.Duration(i) * time.Hour),
			RawData: map[string]string{
				"platform": "crowdstrike-falcon",
				"product":  "Falcon Prevent",
			},
		})
	}
	return events
}

// mockDetectionsAsFindings returns mock detections mapped to VulnFinding.
func (cs *CrowdStrikeAdapter) mockDetectionsAsFindings(opts vulnconnector.QueryOptions) []vulnconnector.VulnFinding {
	return csEventsToFindings(cs.mockDetections(opts))
}

// mockEndpoints returns deterministic synthetic CrowdStrike device inventory.
func (cs *CrowdStrikeAdapter) mockEndpoints(opts vulnconnector.QueryOptions) []vulnconnector.EndpointDevice {
	limit := csEffectiveLimit(opts)
	now := time.Now().UTC()
	devices := []vulnconnector.EndpointDevice{
		{
			ID: "cs-dev-001", Source: "crowdstrike", Hostname: "laptop-exec-ceo.example.com",
			IPAddresses: []string{"10.100.0.50"}, OperatingSystem: "Windows 11 Enterprise 23H2",
			AgentVersion: "7.14.17202.0", Status: "online", LastSeen: now.Add(-1 * time.Minute),
			Tags: []string{"group:executive", "tier:1"},
		},
		{
			ID: "cs-dev-002", Source: "crowdstrike", Hostname: "server-dc-01.example.com",
			IPAddresses: []string{"10.0.0.5"}, OperatingSystem: "Windows Server 2022 Datacenter",
			AgentVersion: "7.14.17202.0", Status: "online", LastSeen: now.Add(-2 * time.Minute),
			Tags: []string{"group:servers", "role:domain-controller"},
		},
		{
			ID: "cs-dev-003", Source: "crowdstrike", Hostname: "workstation-dev-01.example.com",
			IPAddresses: []string{"10.100.1.20"}, OperatingSystem: "macOS 14.4 Sonoma",
			AgentVersion: "7.12.16605.0", Status: "online", LastSeen: now.Add(-5 * time.Minute),
			Tags: []string{"group:engineering", "os:macos"},
		},
		{
			ID: "cs-dev-004", Source: "crowdstrike", Hostname: "server-web-prod-02.example.com",
			IPAddresses: []string{"10.0.10.15"}, OperatingSystem: "Amazon Linux 2023",
			AgentVersion: "7.14.17202.0", Status: "reduced-functionality", LastSeen: now.Add(-20 * time.Minute),
			Tags: []string{"group:servers", "role:web", "env:production"},
		},
		{
			ID: "cs-dev-005", Source: "crowdstrike", Hostname: "laptop-finance-02.example.com",
			IPAddresses: []string{"10.100.2.33"}, OperatingSystem: "Windows 10 Enterprise 22H2",
			AgentVersion: "7.11.16303.0", Status: "contained", LastSeen: now.Add(-45 * time.Minute),
			Tags: []string{"group:finance", "isolated:true"},
		},
	}
	if limit < len(devices) {
		return devices[:limit]
	}
	return devices
}

// -------------------------------------------------------------------
// Internal types and conversion helpers
// -------------------------------------------------------------------

type csDetection struct {
	DetectionID string `json:"detection_id"`
	Hostname    string `json:"hostname"`
	DeviceID    string `json:"device_id"`
	Status      string `json:"status"`
	Severity    string `json:"severity_name"`
	Description string `json:"description"`
	Behaviors   []struct {
		TacticID   string `json:"tactic_id"`
		Tactic     string `json:"tactic"`
		TechniqueID string `json:"technique_id"`
		Filepath   string `json:"filepath"`
	} `json:"behaviors"`
	CreatedTimestamp string `json:"created_timestamp"`
	UpdatedTimestamp string `json:"updated_timestamp"`
}

type csDevice struct {
	DeviceID        string   `json:"device_id"`
	Hostname        string   `json:"hostname"`
	LocalIP         string   `json:"local_ip"`
	ExternalIP      string   `json:"external_ip"`
	OSVersion       string   `json:"os_version"`
	AgentVersion    string   `json:"agent_version"`
	Status          string   `json:"status"`
	LastSeen        string   `json:"last_seen"`
	Groups          []string `json:"groups"`
	Tags            []string `json:"tags"`
}

func csDetectionsToEvents(dets []csDetection) []vulnconnector.DetectionEvent {
	events := make([]vulnconnector.DetectionEvent, 0, len(dets))
	for _, d := range dets {
		sev := vulnconnector.NormalizeSeverityLabel(d.Severity)
		start, _ := time.Parse(time.RFC3339, d.CreatedTimestamp)
		end, _ := time.Parse(time.RFC3339, d.UpdatedTimestamp)
		tactics := make([]string, 0, len(d.Behaviors))
		techniques := make([]string, 0, len(d.Behaviors))
		for _, b := range d.Behaviors {
			if b.Tactic != "" {
				tactics = append(tactics, b.Tactic)
			}
			if b.TechniqueID != "" {
				techniques = append(techniques, b.TechniqueID)
			}
		}
		events = append(events, vulnconnector.DetectionEvent{
			ID:          d.DetectionID,
			Source:      "crowdstrike",
			Severity:    sev,
			Status:      d.Status,
			Description: d.Description,
			Tactics:     tactics,
			Techniques:  techniques,
			HostIDs:     []string{d.DeviceID},
			StartTime:   start,
			EndTime:     end,
		})
	}
	return events
}

func csDevicesToEndpoints(devs []csDevice) []vulnconnector.EndpointDevice {
	endpoints := make([]vulnconnector.EndpointDevice, 0, len(devs))
	for _, d := range devs {
		var ips []string
		if d.LocalIP != "" {
			ips = append(ips, d.LocalIP)
		}
		if d.ExternalIP != "" && d.ExternalIP != d.LocalIP {
			ips = append(ips, d.ExternalIP)
		}
		lastSeen, _ := time.Parse(time.RFC3339, d.LastSeen)
		tags := make([]string, 0, len(d.Tags)+len(d.Groups))
		tags = append(tags, d.Tags...)
		tags = append(tags, d.Groups...)
		endpoints = append(endpoints, vulnconnector.EndpointDevice{
			ID:              d.DeviceID,
			Source:          "crowdstrike",
			Hostname:        d.Hostname,
			IPAddresses:     ips,
			OperatingSystem: d.OSVersion,
			AgentVersion:    d.AgentVersion,
			Status:          d.Status,
			LastSeen:        lastSeen,
			Tags:            tags,
		})
	}
	return endpoints
}

// csEventsToFindings maps DetectionEvent to VulnFinding for interface compatibility.
func csEventsToFindings(events []vulnconnector.DetectionEvent) []vulnconnector.VulnFinding {
	findings := make([]vulnconnector.VulnFinding, 0, len(events))
	for _, e := range events {
		hostID := ""
		if len(e.HostIDs) > 0 {
			hostID = e.HostIDs[0]
		}
		findings = append(findings, vulnconnector.VulnFinding{
			ID:                e.ID,
			Source:            "crowdstrike",
			HostID:            hostID,
			Severity:          e.Severity,
			Title:             e.Title,
			Description:       e.Description,
			RemediationStatus: e.Status,
			FirstDetected:     e.StartTime,
			LastDetected:      e.EndTime,
			RawData: map[string]string{
				"source":     "crowdstrike",
				"type":       "detection",
			},
		})
	}
	return findings
}

func csEffectiveLimit(opts vulnconnector.QueryOptions) int {
	if opts.Limit > 0 && opts.Limit <= 100 {
		return opts.Limit
	}
	return 10
}

