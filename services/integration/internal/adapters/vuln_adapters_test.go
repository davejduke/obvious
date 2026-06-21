package adapters_test

import (
	"context"
	"testing"

	"github.com/davejduke/obvious/services/integration/internal/adapters"
	"github.com/davejduke/obvious/services/integration/internal/vulnconnector"
)

// ---------------------------------------------------------------------------
// Qualys adapter tests
// ---------------------------------------------------------------------------

func TestQualysAdapter_Name(t *testing.T) {
	q := adapters.NewQualysAdapter(adapters.QualysConfig{})
	if q.Name() != "qualys" {
		t.Errorf("expected name=qualys, got %s", q.Name())
	}
}

func TestQualysAdapter_MockHealth(t *testing.T) {
	q := adapters.NewQualysAdapter(adapters.QualysConfig{MockMode: true})
	status := q.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "qualys" {
		t.Errorf("expected connector=qualys, got %s", status.Connector)
	}
}

func TestQualysAdapter_MockFetchVulnerabilities(t *testing.T) {
	q := adapters.NewQualysAdapter(adapters.QualysConfig{MockMode: true})
	findings, err := q.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 5 {
		t.Errorf("expected 5 findings, got %d", len(findings))
	}
	for _, f := range findings {
		if f.Source != "qualys" {
			t.Errorf("expected source=qualys, got %s", f.Source)
		}
		if f.QID == "" {
			t.Error("expected non-empty QID")
		}
		if f.Severity == "" {
			t.Error("expected non-empty Severity")
		}
		if f.HostIP == "" {
			t.Error("expected non-empty HostIP")
		}
	}
}

func TestQualysAdapter_MockFetchVulnerabilities_DefaultLimit(t *testing.T) {
	q := adapters.NewQualysAdapter(adapters.QualysConfig{MockMode: true})
	findings, err := q.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Error("expected default mock findings")
	}
}

func TestQualysAdapter_MockFetchEndpoints(t *testing.T) {
	q := adapters.NewQualysAdapter(adapters.QualysConfig{MockMode: true})
	devices, err := q.FetchEndpoints(context.Background(), vulnconnector.QueryOptions{Limit: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 3 {
		t.Errorf("expected 3 devices, got %d", len(devices))
	}
	for _, d := range devices {
		if d.Source != "qualys" {
			t.Errorf("expected source=qualys, got %s", d.Source)
		}
		if d.Hostname == "" {
			t.Error("expected non-empty Hostname")
		}
		if len(d.IPAddresses) == 0 {
			t.Error("expected at least one IP address")
		}
	}
}

func TestQualysAdapter_CVSSNormalization(t *testing.T) {
	q := adapters.NewQualysAdapter(adapters.QualysConfig{MockMode: true})
	findings, _ := q.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{Limit: 10})
	for _, f := range findings {
		// Any finding with CVSS3 >= 9.0 must be Critical
		if f.CVSS3Score >= 9.0 && f.Severity != vulnconnector.SeverityCritical {
			t.Errorf("CVSS3=%.1f should be Critical, got %s", f.CVSS3Score, f.Severity)
		}
		// Any finding with CVSS3 >= 7.0 and < 9.0 must be High
		if f.CVSS3Score >= 7.0 && f.CVSS3Score < 9.0 && f.Severity != vulnconnector.SeverityHigh {
			t.Errorf("CVSS3=%.1f should be High, got %s", f.CVSS3Score, f.Severity)
		}
	}
}

// ---------------------------------------------------------------------------
// Tenable adapter tests
// ---------------------------------------------------------------------------

func TestTenableAdapter_Name(t *testing.T) {
	adapter := adapters.NewTenableAdapter(adapters.TenableConfig{})
	if adapter.Name() != "tenable" {
		t.Errorf("expected name=tenable, got %s", adapter.Name())
	}
}

func TestTenableAdapter_MockHealth(t *testing.T) {
	adapter := adapters.NewTenableAdapter(adapters.TenableConfig{MockMode: true})
	status := adapter.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "tenable" {
		t.Errorf("expected connector=tenable, got %s", status.Connector)
	}
}

func TestTenableAdapter_MockFetchVulnerabilities(t *testing.T) {
	adapter := adapters.NewTenableAdapter(adapters.TenableConfig{MockMode: true})
	findings, err := adapter.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{Limit: 6})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 6 {
		t.Errorf("expected 6 findings, got %d", len(findings))
	}
	for _, f := range findings {
		if f.Source != "tenable" {
			t.Errorf("expected source=tenable, got %s", f.Source)
		}
		if f.PluginID == "" {
			t.Error("expected non-empty PluginID")
		}
		if f.Severity == "" {
			t.Error("expected non-empty Severity")
		}
		if f.Title == "" {
			t.Error("expected non-empty Title")
		}
	}
}

func TestTenableAdapter_MockFetchVulnerabilities_DefaultLimit(t *testing.T) {
	adapter := adapters.NewTenableAdapter(adapters.TenableConfig{MockMode: true})
	findings, err := adapter.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) == 0 {
		t.Error("expected default mock findings")
	}
}

func TestTenableAdapter_MockFetchEndpoints(t *testing.T) {
	adapter := adapters.NewTenableAdapter(adapters.TenableConfig{MockMode: true})
	devices, err := adapter.FetchEndpoints(context.Background(), vulnconnector.QueryOptions{Limit: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 4 {
		t.Errorf("expected 4 devices, got %d", len(devices))
	}
	for _, d := range devices {
		if d.Source != "tenable" {
			t.Errorf("expected source=tenable, got %s", d.Source)
		}
		if d.Hostname == "" {
			t.Error("expected non-empty Hostname")
		}
	}
}

func TestTenableAdapter_PluginDataFields(t *testing.T) {
	adapter := adapters.NewTenableAdapter(adapters.TenableConfig{MockMode: true})
	findings, _ := adapter.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{Limit: 1})
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	f := findings[0]
	if f.RawData["plugin_family"] == "" {
		t.Error("expected plugin_family in RawData")
	}
	if f.HostID == "" || f.HostIP == "" {
		t.Error("expected host details populated")
	}
}

// ---------------------------------------------------------------------------
// CrowdStrike adapter tests
// ---------------------------------------------------------------------------

func TestCrowdStrikeAdapter_Name(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{})
	if cs.Name() != "crowdstrike" {
		t.Errorf("expected name=crowdstrike, got %s", cs.Name())
	}
}

func TestCrowdStrikeAdapter_MockHealth(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{MockMode: true})
	status := cs.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "crowdstrike" {
		t.Errorf("expected connector=crowdstrike, got %s", status.Connector)
	}
}

func TestCrowdStrikeAdapter_MockFetchVulnerabilities(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{MockMode: true})
	findings, err := cs.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{Limit: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 4 {
		t.Errorf("expected 4 findings, got %d", len(findings))
	}
	for _, f := range findings {
		if f.Source != "crowdstrike" {
			t.Errorf("expected source=crowdstrike, got %s", f.Source)
		}
		if f.Severity == "" {
			t.Error("expected non-empty Severity")
		}
		if f.Title == "" {
			t.Error("expected non-empty Title")
		}
	}
}

func TestCrowdStrikeAdapter_MockFetchEndpoints(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{MockMode: true})
	devices, err := cs.FetchEndpoints(context.Background(), vulnconnector.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(devices) != 5 {
		t.Errorf("expected 5 devices, got %d", len(devices))
	}
	for _, d := range devices {
		if d.Source != "crowdstrike" {
			t.Errorf("expected source=crowdstrike, got %s", d.Source)
		}
		if d.AgentVersion == "" {
			t.Error("expected non-empty AgentVersion")
		}
		if len(d.IPAddresses) == 0 {
			t.Error("expected at least one IP address")
		}
	}
}

func TestCrowdStrikeAdapter_MockFetchDetections(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{MockMode: true})
	events, err := cs.FetchDetections(context.Background(), vulnconnector.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(events) != 5 {
		t.Errorf("expected 5 detection events, got %d", len(events))
	}
	for _, e := range events {
		if e.Source != "crowdstrike" {
			t.Errorf("expected source=crowdstrike, got %s", e.Source)
		}
		if len(e.Tactics) == 0 {
			t.Error("expected at least one tactic")
		}
		if len(e.Techniques) == 0 {
			t.Error("expected at least one technique")
		}
	}
}

func TestCrowdStrikeAdapter_DetectionsMapToFindings(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{MockMode: true})
	findings, err := cs.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{Limit: 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range findings {
		// Detections mapped to findings must have a hostID and remediation status
		if f.HostID == "" {
			t.Error("expected non-empty HostID from detection mapping")
		}
		if f.RemediationStatus == "" {
			t.Error("expected non-empty RemediationStatus from detection mapping")
		}
	}
}

// ---------------------------------------------------------------------------
// CVSS normalization tests
// ---------------------------------------------------------------------------

func TestNormalizeCVSS3(t *testing.T) {
	cases := []struct {
		score float64
		want  vulnconnector.Severity
	}{
		{10.0, vulnconnector.SeverityCritical},
		{9.0, vulnconnector.SeverityCritical},
		{8.9, vulnconnector.SeverityHigh},
		{7.0, vulnconnector.SeverityHigh},
		{6.9, vulnconnector.SeverityMedium},
		{4.0, vulnconnector.SeverityMedium},
		{3.9, vulnconnector.SeverityLow},
		{0.1, vulnconnector.SeverityLow},
		{0.0, vulnconnector.SeverityInformational},
	}
	for _, tc := range cases {
		got := vulnconnector.NormalizeCVSS3(tc.score)
		if got != tc.want {
			t.Errorf("NormalizeCVSS3(%.1f) = %s, want %s", tc.score, got, tc.want)
		}
	}
}

func TestNormalizeCVSS2(t *testing.T) {
	cases := []struct {
		score float64
		want  vulnconnector.Severity
	}{
		{10.0, vulnconnector.SeverityHigh},
		{7.0, vulnconnector.SeverityHigh},
		{6.9, vulnconnector.SeverityMedium},
		{4.0, vulnconnector.SeverityMedium},
		{3.9, vulnconnector.SeverityLow},
		{0.1, vulnconnector.SeverityLow},
		{0.0, vulnconnector.SeverityInformational},
	}
	for _, tc := range cases {
		got := vulnconnector.NormalizeCVSS2(tc.score)
		if got != tc.want {
			t.Errorf("NormalizeCVSS2(%.1f) = %s, want %s", tc.score, got, tc.want)
		}
	}
}

func TestNormalizeSeverityLabel(t *testing.T) {
	cases := []struct {
		label string
		want  vulnconnector.Severity
	}{
		{"5", vulnconnector.SeverityCritical},
		{"critical", vulnconnector.SeverityCritical},
		{"Critical", vulnconnector.SeverityCritical},
		{"4", vulnconnector.SeverityHigh},
		{"HIGH", vulnconnector.SeverityHigh},
		{"3", vulnconnector.SeverityMedium},
		{"medium", vulnconnector.SeverityMedium},
		{"2", vulnconnector.SeverityLow},
		{"low", vulnconnector.SeverityLow},
		{"1", vulnconnector.SeverityInformational},
		{"info", vulnconnector.SeverityInformational},
		{"INFORMATIONAL", vulnconnector.SeverityInformational},
		{"unknown", vulnconnector.SeverityNone},
		{"", vulnconnector.SeverityNone},
	}
	for _, tc := range cases {
		got := vulnconnector.NormalizeSeverityLabel(tc.label)
		if got != tc.want {
			t.Errorf("NormalizeSeverityLabel(%q) = %s, want %s", tc.label, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Evidence mapping tests
// ---------------------------------------------------------------------------

func TestVulnFindingToEvidence(t *testing.T) {
	q := adapters.NewQualysAdapter(adapters.QualysConfig{MockMode: true})
	findings, _ := q.FetchVulnerabilities(context.Background(), vulnconnector.QueryOptions{Limit: 3})
	items := vulnconnector.VulnFindingsToEvidence(findings)
	if len(items) != 3 {
		t.Errorf("expected 3 evidence items, got %d", len(items))
	}
	for _, item := range items {
		if item.Category != vulnconnector.EvidenceCategoryVulnerability {
			t.Errorf("expected category=vulnerability, got %s", item.Category)
		}
		if item.Source != "qualys" {
			t.Errorf("expected source=qualys, got %s", item.Source)
		}
		if item.ID == "" {
			t.Error("expected non-empty evidence ID")
		}
		if item.Severity == "" {
			t.Error("expected non-empty Severity")
		}
	}
}

func TestEndpointDeviceToEvidence(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{MockMode: true})
	devices, _ := cs.FetchEndpoints(context.Background(), vulnconnector.QueryOptions{Limit: 2})
	items := vulnconnector.EndpointDevicesToEvidence(devices)
	if len(items) != 2 {
		t.Errorf("expected 2 evidence items, got %d", len(items))
	}
	for _, item := range items {
		if item.Category != vulnconnector.EvidenceCategoryEndpoint {
			t.Errorf("expected category=endpoint, got %s", item.Category)
		}
		if item.Source != "crowdstrike" {
			t.Errorf("expected source=crowdstrike, got %s", item.Source)
		}
	}
}

func TestDetectionEventToEvidence(t *testing.T) {
	cs := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{MockMode: true})
	detections, _ := cs.FetchDetections(context.Background(), vulnconnector.QueryOptions{Limit: 2})
	items := vulnconnector.DetectionEventsToEvidence(detections)
	if len(items) != 2 {
		t.Errorf("expected 2 evidence items, got %d", len(items))
	}
	for _, item := range items {
		if item.Category != vulnconnector.EvidenceCategoryDetection {
			t.Errorf("expected category=detection, got %s", item.Category)
		}
		if item.Metadata["status"] == nil {
			t.Error("expected status in evidence metadata")
		}
	}
}

