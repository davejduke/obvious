// Tenable.io connector — fetches vulnerability findings, severity, affected hosts, and plugin data.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/vulnconnector"
)

// TenableConfig holds Tenable.io REST API connection settings.
type TenableConfig struct {
	// BaseURL is the Tenable.io API base URL.
	BaseURL string
	// AccessKey is the Tenable.io API access key.
	AccessKey string
	// SecretKey is the Tenable.io API secret key (populated from env/vault).
	SecretKey string
	// MockMode returns synthetic data without making real API calls.
	MockMode bool
}

// TenableAdapter fetches vulnerability findings from Tenable.io via the
// Tenable.io REST API v3.
type TenableAdapter struct {
	config TenableConfig
	client *http.Client
}

// NewTenableAdapter creates a ready-to-use Tenable connector.
func NewTenableAdapter(cfg TenableConfig) *TenableAdapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://cloud.tenable.com"
	}
	return &TenableAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements vulnconnector.VulnConnector.
func (t *TenableAdapter) Name() string { return "tenable" }

// FetchVulnerabilities returns normalized Tenable vulnerability findings.
func (t *TenableAdapter) FetchVulnerabilities(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.VulnFinding, error) {
	if t.config.MockMode {
		return t.mockVulnerabilities(opts), nil
	}
	return t.fetchVulnsFromAPI(ctx, opts)
}

// FetchEndpoints returns the Tenable-managed asset list.
func (t *TenableAdapter) FetchEndpoints(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.EndpointDevice, error) {
	if t.config.MockMode {
		return t.mockEndpoints(opts), nil
	}
	return t.fetchAssetsFromAPI(ctx, opts)
}

// Health reports whether the Tenable.io API is reachable.
func (t *TenableAdapter) Health(ctx context.Context) vulnconnector.HealthStatus {
	status := vulnconnector.HealthStatus{
		Connector:   t.Name(),
		LastChecked: time.Now().UTC(),
	}
	if t.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}
	url := t.config.BaseURL + "/session"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
	t.setAuth(req)
	resp, err := t.client.Do(req)
	if err != nil {
		status.Message = "connectivity error: " + err.Error()
		return status
	}
	resp.Body.Close()
	status.Healthy = resp.StatusCode < 500
	status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return status
}

// fetchVulnsFromAPI calls the Tenable.io vulnerabilities/findings endpoint.
func (t *TenableAdapter) fetchVulnsFromAPI(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.VulnFinding, error) {
	limit := tenableEffectiveLimit(opts)
	url := fmt.Sprintf("%s/workbenches/vulnerabilities?num_findings=%d", t.config.BaseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("tenable: build request: %w", err)
	}
	t.setAuth(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tenable: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tenable: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenable: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Tenable.io workbenches response shape:
	// { "vulnerabilities": [ { "plugin_id": ..., "plugin_name": ..., "severity": ..., ... } ] }
	var result struct {
		Vulns []struct {
			PluginID        int     `json:"plugin_id"`
			PluginName      string  `json:"plugin_name"`
			PluginFamily    string  `json:"plugin_family"`
			Severity        int     `json:"severity"` // 0=info,1=low,2=medium,3=high,4=critical
			Count           int     `json:"count"`
			CVSS3BaseScore  float64 `json:"cvss3_base_score"`
			CVSS2BaseScore  float64 `json:"cvss_base_score"`
			VulnText        string  `json:"vulnerability_text"`
			Solution        string  `json:"solution"`
			AffectedHosts   []struct {
				Hostname string `json:"hostname"`
				IPv4     string `json:"ipv4"`
				ID       string `json:"id"`
			} `json:"affected_hosts"`
			CVEs []string `json:"cve"`
		} `json:"vulnerabilities"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("tenable: parse response: %w", err)
	}

	now := time.Now().UTC()
	findings := make([]vulnconnector.VulnFinding, 0, len(result.Vulns))
	for _, v := range result.Vulns {
		sev := tenableSeverityFromInt(v.Severity)
		if v.CVSS3BaseScore > 0 {
			sev = vulnconnector.NormalizeCVSS3(v.CVSS3BaseScore)
		}
		hostID, hostIP, hostname := "", "", ""
		if len(v.AffectedHosts) > 0 {
			hostID = v.AffectedHosts[0].ID
			hostIP = v.AffectedHosts[0].IPv4
			hostname = v.AffectedHosts[0].Hostname
		}
		findings = append(findings, vulnconnector.VulnFinding{
			ID:          fmt.Sprintf("%d", v.PluginID),
			Source:      "tenable",
			HostID:      hostID,
			HostIP:      hostIP,
			HostName:    hostname,
			PluginID:    fmt.Sprintf("%d", v.PluginID),
			CVEIDs:      v.CVEs,
			CVSS2Score:  v.CVSS2BaseScore,
			CVSS3Score:  v.CVSS3BaseScore,
			Severity:    sev,
			Title:       v.PluginName,
			Description: v.VulnText,
			Solution:    v.Solution,
			FirstDetected: now.Add(-72 * time.Hour),
			LastDetected:  now,
			RawData: map[string]string{
				"plugin_family": v.PluginFamily,
				"count":         fmt.Sprintf("%d", v.Count),
			},
		})
	}
	return findings, nil
}

// fetchAssetsFromAPI retrieves the Tenable-managed asset inventory.
func (t *TenableAdapter) fetchAssetsFromAPI(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.EndpointDevice, error) {
	limit := tenableEffectiveLimit(opts)
	url := fmt.Sprintf("%s/assets?limit=%d", t.config.BaseURL, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("tenable: build asset request: %w", err)
	}
	t.setAuth(req)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tenable: asset request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("tenable: read asset body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tenable: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Assets []struct {
			ID        string   `json:"id"`
			Hostnames []string `json:"hostnames"`
			IPv4s     []string `json:"ipv4s"`
			OS        []string `json:"operating_systems"`
			LastSeen  string   `json:"last_seen"`
			Tags      []struct {
				Key   string `json:"tag_key"`
				Value string `json:"tag_value"`
			} `json:"tags"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("tenable: parse asset response: %w", err)
	}

	devices := make([]vulnconnector.EndpointDevice, 0, len(result.Assets))
	for _, a := range result.Assets {
		hostname := ""
		if len(a.Hostnames) > 0 {
			hostname = a.Hostnames[0]
		}
		os := ""
		if len(a.OS) > 0 {
			os = a.OS[0]
		}
		lastSeen, _ := time.Parse(time.RFC3339, a.LastSeen)
		tags := make([]string, 0, len(a.Tags))
		for _, tag := range a.Tags {
			tags = append(tags, tag.Key+":"+tag.Value)
		}
		devices = append(devices, vulnconnector.EndpointDevice{
			ID:              a.ID,
			Source:          "tenable",
			Hostname:        hostname,
			IPAddresses:     a.IPv4s,
			OperatingSystem: os,
			LastSeen:        lastSeen,
			Tags:            tags,
			Status:          "active",
		})
	}
	return devices, nil
}

// mockVulnerabilities returns deterministic synthetic Tenable findings with
// plugin metadata representative of real Tenable.io responses.
func (t *TenableAdapter) mockVulnerabilities(opts vulnconnector.QueryOptions) []vulnconnector.VulnFinding {
	limit := tenableEffectiveLimit(opts)
	now := time.Now().UTC()

	type pluginDef struct {
		id     int
		name   string
		Family string
		cvss3  float64
		cvss2  float64
		cves   []string
		fix    string
	}
	plugins := []pluginDef{
		{112526, "Apache Log4j Unsupported Version", "Web Servers", 9.8, 9.3, []string{"CVE-2021-44228"}, "Upgrade Apache Log4j to a supported version."},
		{156032, "OpenSSL Multiple Vulnerabilities", "General", 9.1, 8.8, []string{"CVE-2023-0286", "CVE-2023-0215"}, "Upgrade OpenSSL to 3.0.8 or later."},
		{19506, "Nessus Scan Information", "Settings", 0.0, 0.0, []string{}, "N/A"},
		{11213, "HTTP TRACE Method Enabled", "Web Servers", 5.8, 4.3, []string{}, "Disable HTTP TRACE method in web server configuration."},
		{42873, "SSL Medium Strength Cipher Suites (SWEET32)", "General", 5.3, 4.3, []string{"CVE-2016-2183"}, "Reconfigure SSL/TLS to avoid medium strength cipher suites."},
		{65821, "SMB Signing Not Required", "Windows", 5.3, 5.0, []string{}, "Enable mandatory SMB signing on Windows hosts."},
		{57610, "Microsoft Windows Remote Desktop Protocol (RDP) Use-After-Free", "Windows", 8.8, 8.3, []string{"CVE-2012-0002"}, "Apply Microsoft security update MS12-020."},
		{71049, "SSH Server CBC Mode Ciphers Enabled", "General", 5.3, 4.3, []string{}, "Disable CBC mode ciphers in SSH server configuration."},
		{93360, "Apache Struts RCE (CVE-2017-5638)", "CGI abuses", 9.8, 10.0, []string{"CVE-2017-5638"}, "Upgrade Apache Struts to 2.3.32 or 2.5.10.1."},
		{31705, "Insecure Windows Service Permissions", "Windows", 6.7, 6.5, []string{}, "Review and restrict Windows service permissions."},
	}

	hosts := []struct{ id, ip, hostname string }{
		{"tn-host-001", "192.168.10.5", "server-prod-01.example.com"},
		{"tn-host-002", "192.168.10.6", "server-prod-02.example.com"},
		{"tn-host-003", "192.168.20.10", "workstation-finance-01.example.com"},
	}

	findings := make([]vulnconnector.VulnFinding, 0, limit)
	for i := 0; i < limit && i < len(plugins)*len(hosts); i++ {
		p := plugins[i%len(plugins)]
		h := hosts[i%len(hosts)]
		sev := tenableSeverityFromInt(0)
		if p.cvss3 > 0 {
			sev = vulnconnector.NormalizeCVSS3(p.cvss3)
		}
		cves := make([]string, len(p.cves))
		copy(cves, p.cves)
		findings = append(findings, vulnconnector.VulnFinding{
			ID:          fmt.Sprintf("%d-%s-MOCK", p.id, h.id),
			Source:      "tenable",
			HostID:      h.id,
			HostIP:      h.ip,
			HostName:    h.hostname,
			PluginID:    fmt.Sprintf("%d", p.id),
			CVEIDs:      cves,
			CVSS2Score:  p.cvss2,
			CVSS3Score:  p.cvss3,
			Severity:    sev,
			Title:       p.name,
			Description: fmt.Sprintf("Tenable plugin %d (%s): %s detected on %s", p.id, p.Family, p.name, h.hostname),
			Solution:    p.fix,
			FirstDetected: now.Add(-time.Duration(i+3) * 24 * time.Hour),
			LastDetected:  now.Add(-time.Duration(i) * time.Hour),
			RawData: map[string]string{
				"plugin_family": p.Family,
				"scanner":       "tenable-io",
			},
		})
	}
	return findings
}

// mockEndpoints returns deterministic synthetic Tenable assets.
func (t *TenableAdapter) mockEndpoints(opts vulnconnector.QueryOptions) []vulnconnector.EndpointDevice {
	limit := tenableEffectiveLimit(opts)
	now := time.Now().UTC()
	assets := []vulnconnector.EndpointDevice{
		{
			ID: "tn-host-001", Source: "tenable", Hostname: "server-prod-01.example.com",
			IPAddresses: []string{"192.168.10.5"}, OperatingSystem: "Linux Kernel 5.15",
			Status: "active", LastSeen: now.Add(-15 * time.Minute),
			Tags: []string{"env:production", "role:web"},
		},
		{
			ID: "tn-host-002", Source: "tenable", Hostname: "server-prod-02.example.com",
			IPAddresses: []string{"192.168.10.6"}, OperatingSystem: "Linux Kernel 5.15",
			Status: "active", LastSeen: now.Add(-8 * time.Minute),
			Tags: []string{"env:production", "role:api"},
		},
		{
			ID: "tn-host-003", Source: "tenable", Hostname: "workstation-finance-01.example.com",
			IPAddresses: []string{"192.168.20.10"}, OperatingSystem: "Windows 11 Enterprise",
			Status: "active", LastSeen: now.Add(-30 * time.Minute),
			Tags: []string{"env:corporate", "dept:finance"},
		},
		{
			ID: "tn-host-004", Source: "tenable", Hostname: "scan-target-dmz-01.example.com",
			IPAddresses: []string{"10.50.0.25"}, OperatingSystem: "CentOS 7",
			Status: "stale", LastSeen: now.Add(-72 * time.Hour),
			Tags: []string{"env:dmz", "legacy:true"},
		},
	}
	if limit < len(assets) {
		return assets[:limit]
	}
	return assets
}

// setAuth applies the Tenable API key authentication header.
func (t *TenableAdapter) setAuth(req *http.Request) {
	req.Header.Set("X-ApiKeys", fmt.Sprintf("accessKey=%s;secretKey=%s", t.config.AccessKey, t.config.SecretKey))
	req.Header.Set("Content-Type", "application/json")
}

// tenableSeverityFromInt maps Tenable severity integer (0-4) to Severity.
func tenableSeverityFromInt(s int) vulnconnector.Severity {
	switch s {
	case 4:
		return vulnconnector.SeverityCritical
	case 3:
		return vulnconnector.SeverityHigh
	case 2:
		return vulnconnector.SeverityMedium
	case 1:
		return vulnconnector.SeverityLow
	default:
		return vulnconnector.SeverityInformational
	}
}

func tenableEffectiveLimit(opts vulnconnector.QueryOptions) int {
	if opts.Limit > 0 && opts.Limit <= 100 {
		return opts.Limit
	}
	return 10
}


