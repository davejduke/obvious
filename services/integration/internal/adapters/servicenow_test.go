package adapters_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/adapters"
	"github.com/davejduke/obvious/services/integration/internal/grc"
)

// ─── Mock mode tests ────────────────────────────────────────────────────────

func TestServiceNowAdapter_Name(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})
	if a.Name() != "servicenow" {
		t.Errorf("expected name=servicenow, got %s", a.Name())
	}
}

func TestServiceNowAdapter_MockHealth(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})
	status := a.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "servicenow" {
		t.Errorf("expected connector=servicenow, got %s", status.Connector)
	}
}

func TestServiceNowAdapter_MockExport_GRCItems(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})

	findings := []grc.Finding{
		{
			ID:             "f-001",
			Ref:            "NIS2-ART21-001",
			Title:          "MFA not enforced",
			Description:    "Multi-factor authentication is not enforced for privileged accounts.",
			Severity:       grc.SeverityCritical,
			Recommendation: "Enable MFA for all privileged accounts.",
			ControlRef:     "NIS2-4.1",
		},
		{
			ID:             "f-002",
			Ref:            "NIS2-ART21-002",
			Title:          "Patch management gap",
			Description:    "Critical patches are applied beyond the 30-day SLA.",
			Severity:       grc.SeverityMedium,
			Recommendation: "Implement automated patching within the SLA window.",
			ControlRef:     "NIS2-4.4",
		},
	}

	result, err := a.ExportFindings(context.Background(), findings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Every finding maps to a sn_grc_item
	if len(result.GRCItems) != 2 {
		t.Errorf("expected 2 GRC items, got %d", len(result.GRCItems))
	}
	if result.TotalFindings != 2 {
		t.Errorf("expected TotalFindings=2, got %d", result.TotalFindings)
	}
	if result.Connector != "servicenow" {
		t.Errorf("expected connector=servicenow, got %s", result.Connector)
	}
}

func TestServiceNowAdapter_MockExport_TableMapping(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})

	findings := []grc.Finding{
		{
			ID: "f-crit", Title: "Critical finding",
			Severity:       grc.SeverityCritical,
			Recommendation: "Fix immediately.",
		},
		{
			ID: "f-high", Title: "High finding",
			Severity:       grc.SeverityHigh,
			Recommendation: "Fix within 30 days.",
		},
		{
			ID: "f-med", Title: "Medium finding",
			Severity: grc.SeverityMedium,
			// No recommendation — no remediation task expected
		},
		{
			ID: "f-low", Title: "Low finding",
			Severity:       grc.SeverityLow,
			Recommendation: "Consider fixing.",
		},
	}

	result, err := a.ExportFindings(context.Background(), findings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 4 findings → sn_grc_item
	if len(result.GRCItems) != 4 {
		t.Errorf("expected 4 GRC items, got %d", len(result.GRCItems))
	}

	// Only Critical and High → sn_grc_risk
	if len(result.RiskEntries) != 2 {
		t.Errorf("expected 2 risk entries (critical+high), got %d", len(result.RiskEntries))
	}

	// Critical, High, Low have recommendations → 3 remediation tasks (Medium has none)
	if len(result.RemediationTasks) != 3 {
		t.Errorf("expected 3 remediation tasks, got %d", len(result.RemediationTasks))
	}
}

func TestServiceNowAdapter_MockExport_CMDBCIReference(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})

	findingsWithAsset := []grc.Finding{
		{
			ID:       "f-001",
			Title:    "Firewall misconfiguration",
			Severity: grc.SeverityHigh,
			AssetRef: "ci_firewall_prod_01",
		},
	}
	result, err := a.ExportFindings(context.Background(), findingsWithAsset)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.RiskEntries) != 1 {
		t.Fatalf("expected 1 risk entry, got %d", len(result.RiskEntries))
	}
	// cmdb_ci should use the provided asset reference
	if result.RiskEntries[0].CMDBCI != "ci_firewall_prod_01" {
		t.Errorf("expected cmdb_ci=ci_firewall_prod_01, got %s", result.RiskEntries[0].CMDBCI)
	}

	// Without asset ref, default CI is used
	findingsNoAsset := []grc.Finding{
		{ID: "f-002", Title: "No asset", Severity: grc.SeverityCritical},
	}
	result2, _ := a.ExportFindings(context.Background(), findingsNoAsset)
	if result2.RiskEntries[0].CMDBCI != "aiauditor_default_ci" {
		t.Errorf("expected default cmdb_ci, got %s", result2.RiskEntries[0].CMDBCI)
	}
}

func TestServiceNowAdapter_MockExport_SysIDsAreUnique(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})

	findings := make([]grc.Finding, 5)
	for i := range findings {
		findings[i] = grc.Finding{
			ID:             fmt.Sprintf("f-%03d", i),
			Title:          fmt.Sprintf("Finding %d", i),
			Severity:       grc.SeverityHigh,
			Recommendation: "Remediate.",
		}
	}

	result, _ := a.ExportFindings(context.Background(), findings)
	seen := map[string]bool{}
	for _, item := range result.GRCItems {
		if seen[item.SysID] {
			t.Errorf("duplicate GRC item SysID: %s", item.SysID)
		}
		seen[item.SysID] = true
	}
}

func TestServiceNowAdapter_MockExport_RiskScores(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})

	cases := []struct {
		severity      grc.Severity
		expectedScore int
	}{
		{grc.SeverityCritical, 90},
		{grc.SeverityHigh, 70},
	}

	for _, tc := range cases {
		result, _ := a.ExportFindings(context.Background(), []grc.Finding{
			{ID: "f", Title: "T", Severity: tc.severity},
		})
		if len(result.RiskEntries) == 0 {
			t.Fatalf("expected risk entry for severity %s", tc.severity)
		}
		if result.RiskEntries[0].RiskScore != tc.expectedScore {
			t.Errorf("severity %s: expected risk score %d, got %d",
				tc.severity, tc.expectedScore, result.RiskEntries[0].RiskScore)
		}
	}
}

func TestServiceNowAdapter_MockExport_RemediationDueDates(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})

	findings := []grc.Finding{
		{ID: "f-c", Title: "Critical", Severity: grc.SeverityCritical, Recommendation: "Fix"},
		{ID: "f-h", Title: "High", Severity: grc.SeverityHigh, Recommendation: "Fix"},
	}

	result, _ := a.ExportFindings(context.Background(), findings)
	if len(result.RemediationTasks) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(result.RemediationTasks))
	}

	now := time.Now().UTC()
	// Critical SLA: 7 days
	critDue := result.RemediationTasks[0].DueDate
	if critDue.Before(now.Add(6 * 24 * time.Hour)) {
		t.Error("critical due date should be at least 6 days out")
	}
	// High SLA: 30 days
	highDue := result.RemediationTasks[1].DueDate
	if highDue.Before(now.Add(29 * 24 * time.Hour)) {
		t.Error("high due date should be at least 29 days out")
	}
}

func TestServiceNowAdapter_MockExport_EmptyFindings(t *testing.T) {
	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{MockMode: true})

	result, err := a.ExportFindings(context.Background(), []grc.Finding{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TotalFindings != 0 {
		t.Errorf("expected TotalFindings=0, got %d", result.TotalFindings)
	}
	if len(result.GRCItems) != 0 {
		t.Errorf("expected 0 GRC items, got %d", len(result.GRCItems))
	}
}

// ─── Live mode (mock HTTP server) tests ─────────────────────────────────────

func TestServiceNowAdapter_LiveExport_GRCItem(t *testing.T) {
	// Spin up a fake ServiceNow Table API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"result": map[string]string{"sys_id": "live-sys-id-001"},
		})
	}))
	defer server.Close()

	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "password",
		MockMode: false,
	})

	findings := []grc.Finding{
		{
			ID:             "f-live-001",
			Title:          "Live test finding",
			Description:    "Test description",
			Severity:       grc.SeverityHigh,
			Recommendation: "Remediate.",
		},
	}

	result, err := a.ExportFindings(context.Background(), findings)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.GRCItems) != 1 {
		t.Errorf("expected 1 GRC item, got %d", len(result.GRCItems))
	}
	if result.GRCItems[0].SysID != "live-sys-id-001" {
		t.Errorf("expected SysID=live-sys-id-001, got %s", result.GRCItems[0].SysID)
	}
}

func TestServiceNowAdapter_LiveHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	a := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{
		BaseURL:  server.URL,
		Username: "admin",
		Password: "password",
		MockMode: false,
	})

	status := a.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy, got: %s", status.Message)
	}
}
