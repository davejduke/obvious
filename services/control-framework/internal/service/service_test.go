// Package service_test tests the control framework service using the in-memory repository.
package service_test

import (
	"context"
	"testing"

	"github.com/davejduke/obvious/services/control-framework/internal/domain"
	"github.com/davejduke/obvious/services/control-framework/internal/repository"
	"github.com/davejduke/obvious/services/control-framework/internal/service"
	"github.com/google/uuid"
)

var (
	testOrgID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
)

func newTestService() *service.Service {
	return service.New(repository.NewMemoryRepository())
}

// --- Framework tests ---

func TestCreateAndGetFramework(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	req := domain.CreateFrameworkRequest{
		Name:        "NIS 2 Directive",
		Version:     "2022/2555",
		Authority:   "European Parliament",
		Description: "NIS 2 security requirements",
		IsPublished: true,
	}

	f, err := svc.CreateFramework(ctx, testOrgID, req)
	if err != nil {
		t.Fatalf("CreateFramework: %v", err)
	}
	if f.Name != req.Name {
		t.Errorf("name = %q, want %q", f.Name, req.Name)
	}

	got, err := svc.GetFramework(ctx, f.ID)
	if err != nil {
		t.Fatalf("GetFramework: %v", err)
	}
	if got == nil {
		t.Fatal("expected framework, got nil")
	}
	if got.ID != f.ID {
		t.Errorf("id = %v, want %v", got.ID, f.ID)
	}
}

func TestCreateFramework_Validation(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		req  domain.CreateFrameworkRequest
	}{
		{"missing name", domain.CreateFrameworkRequest{Version: "1.0", Authority: "Test"}},
		{"missing version", domain.CreateFrameworkRequest{Name: "Test", Authority: "Test"}},
		{"missing authority", domain.CreateFrameworkRequest{Name: "Test", Version: "1.0"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.CreateFramework(ctx, testOrgID, tc.req)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

// --- Control tests ---

func TestCreateAndListControls(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	// Create a framework first
	fw, _ := svc.CreateFramework(ctx, testOrgID, domain.CreateFrameworkRequest{
		Name: "NIS 2", Version: "2022/2555", Authority: "EU",
	})

	// Create the 10 NIS 2 domains
	nis2Controls := []struct {
		controlID  string
		title      string
		articleRef string
	}{
		{"NIS2-21a", "Risk analysis and IS policies", "Art. 21(2)(a)"},
		{"NIS2-21b", "Incident handling", "Art. 21(2)(b)"},
		{"NIS2-21c", "Business continuity", "Art. 21(2)(c)"},
		{"NIS2-21d", "Supply chain security", "Art. 21(2)(d)"},
		{"NIS2-21e", "Network and IS acquisition", "Art. 21(2)(e)"},
		{"NIS2-21f", "Effectiveness assessment", "Art. 21(2)(f)"},
		{"NIS2-21g", "Cyber hygiene and training", "Art. 21(2)(g)"},
		{"NIS2-21h", "Cryptography and encryption", "Art. 21(2)(h)"},
		{"NIS2-21i", "Access control and HR security", "Art. 21(2)(i)"},
		{"NIS2-21j", "MFA and secure communications", "Art. 21(2)(j)"},
	}

	for _, ctrl := range nis2Controls {
		_, err := svc.CreateControl(ctx, testOrgID, domain.CreateControlRequest{
			FrameworkID: fw.ID,
			ControlID:   ctrl.controlID,
			Title:       ctrl.title,
			ArticleRef:  ctrl.articleRef,
			RiskWeight:  2.0,
		})
		if err != nil {
			t.Fatalf("CreateControl %s: %v", ctrl.controlID, err)
		}
	}

	// List all controls — should return all 10
	result, err := svc.ListControls(ctx, testOrgID, domain.ListControlsFilter{
		FrameworkID: &fw.ID,
	})
	if err != nil {
		t.Fatalf("ListControls: %v", err)
	}
	if result.Total != 10 {
		t.Errorf("expected 10 NIS 2 controls, got %d", result.Total)
	}

	// Verify all 10 domains are present
	articleRefs := make(map[string]bool)
	for _, c := range result.Controls {
		articleRefs[c.ArticleRef] = true
	}
	for _, expected := range domain.NIS2Domains {
		if !articleRefs[expected] {
			t.Errorf("missing NIS 2 domain: %s", expected)
		}
	}
}

func TestGetEvidenceRequirements(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	fw, _ := svc.CreateFramework(ctx, testOrgID, domain.CreateFrameworkRequest{
		Name: "NIS 2", Version: "2022/2555", Authority: "EU",
	})

	c, err := svc.CreateControl(ctx, testOrgID, domain.CreateControlRequest{
		FrameworkID: fw.ID,
		ControlID:   "NIS2-21a",
		Title:       "Risk analysis",
		ArticleRef:  "Art. 21(2)(a)",
		RiskWeight:  2.0,
		EvidenceRequirements: []domain.EvidenceRequirement{
			{Type: "policy_document", Description: "IS risk policy", Required: true},
			{Type: "risk_register", Description: "Current risk register", Required: true},
		},
	})
	if err != nil {
		t.Fatalf("CreateControl: %v", err)
	}

	evidence, err := svc.GetEvidenceRequirements(ctx, c.ID)
	if err != nil {
		t.Fatalf("GetEvidenceRequirements: %v", err)
	}
	if len(evidence) != 2 {
		t.Errorf("expected 2 evidence requirements, got %d", len(evidence))
	}
	if evidence[0].Type != "policy_document" {
		t.Errorf("first evidence type = %q, want %q", evidence[0].Type, "policy_document")
	}
}

// --- Assessment lifecycle tests ---

func TestAssessmentLifecycle(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	fw, _ := svc.CreateFramework(ctx, testOrgID, domain.CreateFrameworkRequest{
		Name: "NIS 2", Version: "2022/2555", Authority: "EU",
	})
	c, _ := svc.CreateControl(ctx, testOrgID, domain.CreateControlRequest{
		FrameworkID: fw.ID,
		ControlID:   "NIS2-21a",
		Title:       "Risk analysis",
		RiskWeight:  2.0,
	})

	engagementID := uuid.New()

	// Initial state: not_started -> in_progress
	a, err := svc.AssessControl(ctx, c.ID, testOrgID, domain.AssessControlRequest{
		EngagementID: engagementID,
		Status:       domain.AssessmentStatusInProgress,
		Notes:        "Starting fieldwork",
	})
	if err != nil {
		t.Fatalf("in_progress transition: %v", err)
	}
	if a.Status != domain.AssessmentStatusInProgress {
		t.Errorf("status = %q, want %q", a.Status, domain.AssessmentStatusInProgress)
	}

	// in_progress -> assessed
	score := 0.85
	a, err = svc.AssessControl(ctx, c.ID, testOrgID, domain.AssessControlRequest{
		EngagementID: engagementID,
		Status:       domain.AssessmentStatusAssessed,
		Score:        &score,
		Notes:        "Controls reviewed and validated",
	})
	if err != nil {
		t.Fatalf("assessed transition: %v", err)
	}
	if a.Status != domain.AssessmentStatusAssessed {
		t.Errorf("status = %q, want %q", a.Status, domain.AssessmentStatusAssessed)
	}

	// assessed -> remediation
	a, err = svc.AssessControl(ctx, c.ID, testOrgID, domain.AssessControlRequest{
		EngagementID: engagementID,
		Status:       domain.AssessmentStatusRemediation,
		Notes:        "Finding requires remediation",
	})
	if err != nil {
		t.Fatalf("remediation transition: %v", err)
	}
	if a.Status != domain.AssessmentStatusRemediation {
		t.Errorf("status = %q, want %q", a.Status, domain.AssessmentStatusRemediation)
	}
}

func TestAssessmentInvalidTransition(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	fw, _ := svc.CreateFramework(ctx, testOrgID, domain.CreateFrameworkRequest{
		Name: "NIS 2", Version: "2022/2555", Authority: "EU",
	})
	c, _ := svc.CreateControl(ctx, testOrgID, domain.CreateControlRequest{
		FrameworkID: fw.ID,
		ControlID:   "NIS2-21a",
		Title:       "Risk analysis",
		RiskWeight:  2.0,
	})

	engagementID := uuid.New()

	// Try to jump from not_started directly to assessed (invalid)
	_, err := svc.AssessControl(ctx, c.ID, testOrgID, domain.AssessControlRequest{
		EngagementID: engagementID,
		Status:       domain.AssessmentStatusAssessed,
	})
	if err == nil {
		t.Error("expected error for invalid not_started -> assessed transition, got nil")
	}
}

func TestAssessmentInvalidStatus(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	fw, _ := svc.CreateFramework(ctx, testOrgID, domain.CreateFrameworkRequest{
		Name: "NIS 2", Version: "2022/2555", Authority: "EU",
	})
	c, _ := svc.CreateControl(ctx, testOrgID, domain.CreateControlRequest{
		FrameworkID: fw.ID,
		ControlID:   "NIS2-21a",
		Title:       "Risk analysis",
		RiskWeight:  2.0,
	})

	_, err := svc.AssessControl(ctx, c.ID, testOrgID, domain.AssessControlRequest{
		EngagementID: uuid.New(),
		Status:       domain.AssessmentStatus("invalid_status"),
	})
	if err == nil {
		t.Error("expected error for invalid status, got nil")
	}
}

func TestGenerateAuditObjective(t *testing.T) {
	svc := newTestService()
	ctx := context.Background()

	fw, _ := svc.CreateFramework(ctx, testOrgID, domain.CreateFrameworkRequest{
		Name: "NIS 2", Version: "2022/2555", Authority: "EU",
	})
	c, _ := svc.CreateControl(ctx, testOrgID, domain.CreateControlRequest{
		FrameworkID: fw.ID,
		ControlID:   "NIS2-21b",
		Title:       "Incident handling",
		Objective:   "Ensure effective incident response",
		ArticleRef:  "Art. 21(2)(b)",
		RiskWeight:  2.5,
		TestProcedures: []domain.TestProcedure{
			{ID: "T1", Description: "Review incident response plan", ExpectedEvidence: "IRP document"},
		},
	})

	obj, err := svc.GenerateAuditObjective(ctx, c.ID)
	if err != nil {
		t.Fatalf("GenerateAuditObjective: %v", err)
	}
	if obj.ControlRef != "NIS2-21b" {
		t.Errorf("control_ref = %q, want %q", obj.ControlRef, "NIS2-21b")
	}
	if len(obj.TestApproaches) == 0 {
		t.Error("expected at least one test approach")
	}
	if obj.RiskFocus == "" {
		t.Error("expected non-empty risk focus")
	}
}

func TestGetNIS2Domains(t *testing.T) {
	svc := newTestService()
	domains := svc.GetNIS2Domains()

	if len(domains) != 10 {
		t.Errorf("expected 10 NIS 2 domains, got %d", len(domains))
	}

	// Verify all 10 article references have names
	for _, ref := range domain.NIS2Domains {
		name, ok := domains[ref]
		if !ok || name == "" {
			t.Errorf("missing or empty name for domain %s", ref)
		}
	}
}

