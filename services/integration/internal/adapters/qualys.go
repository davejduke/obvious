// Package adapters — Qualys vulnerability scanner connector.
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

// QualysConfig holds Qualys VMDR API connection settings.
type QualysConfig struct {
	// BaseURL is the Qualys API server URL, e.g. https://qualysapi.qualys.com
	BaseURL string
	// Username is the Qualys API account username.
	Username string
	// Password is the Qualys API account password (populated from env/vault).
	Password string
	// MockMode returns synthetic data without making real API calls.
	MockMode bool
}

// QualysAdapter fetches vulnerability findings and asset inventory from
// Qualys VMDR (Vulnerability Management, Detection and Response).
type QualysAdapter struct {
	config QualysConfig
	client *http.Client
}

// NewQualysAdapter creates a ready-to-use Qualys connector.
func NewQualysAdapter(cfg QualysConfig) *QualysAdapter {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://qualysapi.qualys.com"
	}
	return &QualysAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements vulnconnector.VulnConnector.
func (q *QualysAdapter) Name() string { return "qualys" }

// FetchVulnerabilities returns normalized VMDR vulnerability findings.
func (q *QualysAdapter) FetchVulnerabilities(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.VulnFinding, error) {
	if q.config.MockMode {
		return q.mockVulnerabilities(opts), nil
	}
	return q.fetchVulnsFromAPI(ctx, opts)
}

// FetchEndpoints returns the Qualys asset inventory (hosts).
func (q *QualysAdapter) FetchEndpoints(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.EndpointDevice, error) {
	if q.config.MockMode {
		return q.mockEndpoints(opts), nil
	}
	return q.fetchHostsFromAPI(ctx, opts)
}

// Health reports whether the Qualys API is reachable.
func (q *QualysAdapter) Health(ctx context.Context) vulnconnector.HealthStatus {
	status := vulnconnector.HealthStatus{
		Connector:   q.Name(),
		LastChecked: time.Now().UTC(),
	}
	if q.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}
	url := q.config.BaseURL + "/msp/about.php"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
	req.SetBasicAuth(q.config.Username, q.config.Password)
	resp, err := q.client.Do(req)
	if err != nil {
		status.Message = "connectivity error: " + err.Error()
		return status
	}
	resp.Body.Close()
	status.Healthy = resp.StatusCode < 500
	status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return status
}

// fetchVulnsFromAPI calls the Qualys VMDR vulnerability list API.
func (q *QualysAdapter) fetchVulnsFromAPI(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.VulnFinding, error) {
	limit := qualysEffectiveLimit(opts)
	url := fmt.Sprintf(
		"%s/api/2.0/fo/asset/host/vm/detection/?action=list&output_format=JSON&truncation_limit=%d",
		q.config.BaseURL, limit,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("qualys: build request: %w", err)
	}
	req.SetBasicAuth(q.config.Username, q.config.Password)
	req.Header.Set("X-Requested-With", "AIAUDITOR")

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qualys: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("qualys: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qualys: HTTP %d: %s", resp.StatusCode, string(body))
	}

	// Qualys returns XML by default; when JSON is requested the structure is:
	// { "HOST_LIST_VM_DETECTION_OUTPUT": { "RESPONSE": { "HOST_LIST": [ ... ] } } }
	var result struct {
		Output struct {
			Response struct {
				HostList []struct {
					IP       string `json:"IP"`
					ID       string `json:"ID"`
					DNS      string `json:"DNS"`
					DetList []struct {
						QID      string  `json:"QID"`
						Severity string  `json:"SEVERITY"`
						CVSS3    float64 `json:"CVSS3_FINAL,string"`
						CVSS2    float64 `json:"CVSS_FINAL,string"`
						CVEs     string  `json:"CVE_LIST"`
						Title    string  `json:"VULN_TITLE"`
						Status   string  `json:"STATUS"`
						FirstFound string `json:"FIRST_FOUND_DATETIME"`
						LastFound  string `json:"LAST_FOUND_DATETIME"`
					} `json:"DETECTION_LIST"`
				} `json:"HOST"`
			} `json:"RESPONSE"`
		} `json:"HOST_LIST_VM_DETECTION_OUTPUT"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("qualys: parse response: %w", err)
	}

	var findings []vulnconnector.VulnFinding
	for _, host := range result.Output.Response.HostList {
		for _, det := range host.DetList {
			sev := vulnconnector.NormalizeCVSS3(det.CVSS3)
			if sev == vulnconnector.SeverityInformational && det.CVSS2 > 0 {
				sev = vulnconnector.NormalizeCVSS2(det.CVSS2)
			}
			if sev == vulnconnector.SeverityInformational {
				sev = vulnconnector.NormalizeSeverityLabel(det.Severity)
			}
			first, _ := time.Parse(time.RFC3339, det.FirstFound)
			last, _ := time.Parse(time.RFC3339, det.LastFound)
			findings = append(findings, vulnconnector.VulnFinding{
				ID:                fmt.Sprintf("%s-%s", host.ID, det.QID),
				Source:            "qualys",
				HostID:            host.ID,
				HostIP:            host.IP,
				HostName:          host.DNS,
				QID:               det.QID,
				CVSS2Score:        det.CVSS2,
				CVSS3Score:        det.CVSS3,
				Severity:          sev,
				Title:             det.Title,
				RemediationStatus: det.Status,
				FirstDetected:     first,
				LastDetected:      last,
			})
		}
	}
	return findings, nil
}

// fetchHostsFromAPI retrieves the Qualys host asset list.
func (q *QualysAdapter) fetchHostsFromAPI(ctx context.Context, opts vulnconnector.QueryOptions) ([]vulnconnector.EndpointDevice, error) {
	limit := qualysEffectiveLimit(opts)
	url := fmt.Sprintf(
		"%s/api/2.0/fo/asset/host/?action=list&output_format=JSON&truncation_limit=%d",
		q.config.BaseURL, limit,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("qualys: build host request: %w", err)
	}
	req.SetBasicAuth(q.config.Username, q.config.Password)
	req.Header.Set("X-Requested-With", "AIAUDITOR")

	resp, err := q.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("qualys: host request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("qualys: read host body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qualys: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Output struct {
			Response struct {
				HostList []struct {
					ID  string `json:"ID"`
					IP  string `json:"IP"`
					DNS string `json:"DNS"`
					OS  string `json:"OS"`
				} `json:"HOST"`
			} `json:"RESPONSE"`
		} `json:"HOST_LIST_OUTPUT"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("qualys: parse host response: %w", err)
	}

	devices := make([]vulnconnector.EndpointDevice, 0, len(result.Output.Response.HostList))
	for _, h := range result.Output.Response.HostList {
		devices = append(devices, vulnconnector.EndpointDevice{
			ID:              h.ID,
			Source:          "qualys",
			Hostname:        h.DNS,
			IPAddresses:     []string{h.IP},
			OperatingSystem: h.OS,
			Status:          "active",
			LastSeen:        time.Now().UTC(),
		})
	}
	return devices, nil
}

// mockVulnerabilities returns deterministic synthetic Qualys findings.
func (q *QualysAdapter) mockVulnerabilities(opts vulnconnector.QueryOptions) []vulnconnector.VulnFinding {
	limit := qualysEffectiveLimit(opts)
	now := time.Now().UTC()

	type mockDef struct {
		qid    string
		title  string
		cvss3  float64
		cvss2  float64
		cves   []string
		fix    string
		status string
	}
	defs := []mockDef{
		{"12345", "OpenSSL Buffer Overflow (CVE-2024-0001)", 9.8, 9.3, []string{"CVE-2024-0001"}, "Upgrade OpenSSL to 3.2.1", "New"},
		{"12346", "Apache Log4j Remote Code Execution (CVE-2021-44228)", 10.0, 9.8, []string{"CVE-2021-44228"}, "Upgrade Log4j to 2.17.1", "Active"},
		{"23456", "SSH Weak Cipher Negotiation", 5.3, 4.3, []string{}, "Disable weak ciphers in sshd_config", "Active"},
		{"34567", "Windows SMB EternalBlue (MS17-010)", 8.1, 7.8, []string{"CVE-2017-0144"}, "Apply KB4012212 security update", "Active"},
		{"45678", "TLS 1.0 Protocol Enabled", 4.0, 3.7, []string{}, "Disable TLS 1.0 in server config", "Fixed"},
		{"56789", "PHP Remote File Inclusion", 7.5, 7.0, []string{"CVE-2023-1234"}, "Upgrade PHP to 8.2", "New"},
		{"67890", "MySQL Unauthenticated Remote Access", 9.1, 8.5, []string{"CVE-2023-5678"}, "Restrict MySQL bind address", "Active"},
		{"78901", "Sudo Privilege Escalation (CVE-2021-3156)", 7.8, 7.2, []string{"CVE-2021-3156"}, "Upgrade sudo to 1.9.5p2", "Fixed"},
		{"89012", "DNS Amplification Vulnerability", 5.0, 4.3, []string{}, "Enable response rate limiting on DNS", "New"},
		{"90123", "Insecure HTTP Cookie (No Secure Flag)", 3.7, 2.6, []string{}, "Set Secure flag on session cookies", "Active"},
	}

	findings := make([]vulnconnector.VulnFinding, 0, limit)
	hosts := []struct{ id, ip, dns string }{
		{"HOST-001", "10.0.1.10", "web-01.internal.example.com"},
		{"HOST-002", "10.0.1.11", "db-01.internal.example.com"},
		{"HOST-003", "10.0.2.20", "app-01.internal.example.com"},
	}

	for i := 0; i < limit && i < len(defs)*len(hosts); i++ {
		d := defs[i%len(defs)]
		h := hosts[i%len(hosts)]
		sev := vulnconnector.NormalizeCVSS3(d.cvss3)
		cves := make([]string, len(d.cves))
		copy(cves, d.cves)
		findings = append(findings, vulnconnector.VulnFinding{
			ID:                fmt.Sprintf("%s-%s-MOCK", h.id, d.qid),
			Source:            "qualys",
			HostID:            h.id,
			HostIP:            h.ip,
			HostName:          h.dns,
			QID:               d.qid,
			CVEIDs:            cves,
			CVSS2Score:        d.cvss2,
			CVSS3Score:        d.cvss3,
			Severity:          sev,
			Title:             d.title,
			Description:       fmt.Sprintf("Mock Qualys finding: %s on host %s", d.title, h.dns),
			Solution:          d.fix,
			RemediationStatus: d.status,
			FirstDetected:     now.Add(-time.Duration(i+7) * 24 * time.Hour),
			LastDetected:      now.Add(-time.Duration(i) * time.Hour),
			RawData: map[string]string{
				"scanner": "qualys-vmdr",
				"qid":     d.qid,
			},
		})
	}
	return findings
}

// mockEndpoints returns deterministic synthetic Qualys host assets.
func (q *QualysAdapter) mockEndpoints(opts vulnconnector.QueryOptions) []vulnconnector.EndpointDevice {
	limit := qualysEffectiveLimit(opts)
	now := time.Now().UTC()
	hosts := []vulnconnector.EndpointDevice{
		{
			ID: "HOST-001", Source: "qualys", Hostname: "web-01.internal.example.com",
			IPAddresses: []string{"10.0.1.10"}, OperatingSystem: "Ubuntu 22.04 LTS",
			AgentVersion: "3.1.6.0", Status: "active", LastSeen: now.Add(-5 * time.Minute),
			Tags: []string{"web", "production"},
		},
		{
			ID: "HOST-002", Source: "qualys", Hostname: "db-01.internal.example.com",
			IPAddresses: []string{"10.0.1.11"}, OperatingSystem: "Ubuntu 20.04 LTS",
			AgentVersion: "3.1.6.0", Status: "active", LastSeen: now.Add(-3 * time.Minute),
			Tags: []string{"database", "production"},
		},
		{
			ID: "HOST-003", Source: "qualys", Hostname: "app-01.internal.example.com",
			IPAddresses: []string{"10.0.2.20"}, OperatingSystem: "Windows Server 2022",
			AgentVersion: "3.0.9.0", Status: "active", LastSeen: now.Add(-10 * time.Minute),
			Tags: []string{"app", "staging"},
		},
		{
			ID: "HOST-004", Source: "qualys", Hostname: "vpn-gw.example.com",
			IPAddresses: []string{"203.0.113.5"}, OperatingSystem: "pfSense 2.7",
			AgentVersion: "3.1.2.0", Status: "active", LastSeen: now.Add(-2 * time.Minute),
			Tags: []string{"network", "gateway"},
		},
	}
	if limit < len(hosts) {
		return hosts[:limit]
	}
	return hosts
}

func qualysEffectiveLimit(opts vulnconnector.QueryOptions) int {
	if opts.Limit > 0 && opts.Limit <= 100 {
		return opts.Limit
	}
	return 10
}

