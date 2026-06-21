// Package domain — unit tests for the audit planning domain.
package domain

import (
	"testing"

	"github.com/google/uuid"
)

var (
	testOrgID = uuid.New()
	testSPID  = uuid.New()
)

// ---------------------------------------------------------------------------
// StrategicPlan tests
// ---------------------------------------------------------------------------

func TestNewStrategicPlan_HappyPath(t *testing.T) {
	req := CreateStrategicPlanRequest{
		OrgID:     testOrgID,
		Name:      "3-Year Cyber Assurance Plan",
		StartYear: 2025,
		EndYear:   2027,
		Entities: []AuditableEntity{
			{Name: "IAM", BusinessUnit: "IT", RiskRating: RiskCritical, PlannedYear: 2025},
		},
	}
	p, err := NewStrategicPlan(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status != PlanStatusDraft {
		t.Errorf("expected draft status, got %s", p.Status)
	}
	if p.Version != 1 {
		t.Errorf("expected version 1, got %d", p.Version)
	}
	if len(p.Entities) != 1 {
		t.Errorf("expected 1 entity, got %d", len(p.Entities))
	}
	if p.Entities[0].ID == uuid.Nil {
		t.Error("entity should have an assigned ID")
	}
}

func TestNewStrategicPlan_Validation(t *testing.T) {
	cases := []struct {
		name string
		req  CreateStrategicPlanRequest
	}{
		{"missing name", CreateStrategicPlanRequest{OrgID: testOrgID, StartYear: 2025, EndYear: 2027}},
		{"missing org_id", CreateStrategicPlanRequest{Name: "Plan", StartYear: 2025, EndYear: 2027}},
		{"end before start", CreateStrategicPlanRequest{OrgID: testOrgID, Name: "Plan", StartYear: 2027, EndYear: 2025}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewStrategicPlan(tc.req)
			if err == nil {
				t.Error("expected validation error, got nil")
			}
		})
	}
}

func TestStrategicPlan_ApproveAndActivate(t *testing.T) {
	p, _ := NewStrategicPlan(CreateStrategicPlanRequest{
		OrgID: testOrgID, Name: "Plan", StartYear: 2025, EndYear: 2027,
	})

	if err := p.Approve("cae@example.com", nil); err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if p.Status != PlanStatusApproved {
		t.Errorf("expected approved, got %s", p.Status)
	}
	if p.ApprovedBy != "cae@example.com" {
		t.Errorf("expected approvedBy cae@example.com, got %s", p.ApprovedBy)
	}

	if err := p.Activate(); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	if p.Status != PlanStatusActive {
		t.Errorf("expected active, got %s", p.Status)
	}
}

func TestStrategicPlan_InvalidTransitions(t *testing.T) {
	p, _ := NewStrategicPlan(CreateStrategicPlanRequest{
		OrgID: testOrgID, Name: "Plan", StartYear: 2025, EndYear: 2027,
	})

	// Cannot activate from draft (must be approved first).
	if err := p.Activate(); err == nil {
		t.Error("expected invalid transition error")
	}

	// Cannot approve twice.
	_ = p.Approve("cae@example.com", nil)
	if err := p.Approve("other@example.com", nil); err == nil {
		t.Error("expected invalid transition error on double approve")
	}
}

func TestStrategicPlan_ApplyUpdate(t *testing.T) {
	p, _ := NewStrategicPlan(CreateStrategicPlanRequest{
		OrgID: testOrgID, Name: "Plan", StartYear: 2025, EndYear: 2027,
	})
	newName := "Updated Plan"
	entities := []AuditableEntity{
		{Name: "Network", BusinessUnit: "IT", RiskRating: RiskHigh, PlannedYear: 2025},
		{Name: "HR", BusinessUnit: "People", RiskRating: RiskMedium, PlannedYear: 2026},
	}
	p.Apply(UpdateStrategicPlanRequest{Name: &newName, Entities: &entities})

	if p.Name != "Updated Plan" {
		t.Errorf("expected updated name, got %s", p.Name)
	}
	if p.Version != 2 {
		t.Errorf("expected version 2, got %d", p.Version)
	}
	if len(p.Entities) != 2 {
		t.Errorf("expected 2 entities, got %d", len(p.Entities))
	}
}

// ---------------------------------------------------------------------------
// AnnualPlan tests
// ---------------------------------------------------------------------------

func TestNewAnnualPlan_HappyPath(t *testing.T) {
	req := CreateAnnualPlanRequest{
		OrgID: testOrgID,
		Year:  2025,
		Name:  "Annual Audit Plan 2025",
		Engagements: []PlannedEngagement{
			{Name: "IAM Audit", AuditableEntity: "IAM", Quarter: 1, BudgetDays: 10, StartDate: "2025-01-15", EndDate: "2025-01-30"},
		},
	}
	ap, err := NewAnnualPlan(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ap.Status != PlanStatusDraft {
		t.Errorf("expected draft, got %s", ap.Status)
	}
	if len(ap.Engagements) != 1 {
		t.Errorf("expected 1 engagement, got %d", len(ap.Engagements))
	}
	if ap.Engagements[0].Status != EngPlanPlanned {
		t.Errorf("expected planned status, got %s", ap.Engagements[0].Status)
	}
}

func TestAnnualPlan_ApproveAndActivate(t *testing.T) {
	ap, _ := NewAnnualPlan(CreateAnnualPlanRequest{OrgID: testOrgID, Year: 2025, Name: "Plan 2025"})

	approvalID := uuid.New()
	if err := ap.Approve(&approvalID); err != nil {
		t.Fatalf("approve failed: %v", err)
	}
	if ap.Status != PlanStatusApproved {
		t.Errorf("expected approved, got %s", ap.Status)
	}
	if err := ap.Activate(); err != nil {
		t.Fatalf("activate failed: %v", err)
	}
	if ap.Status != PlanStatusActive {
		t.Errorf("expected active, got %s", ap.Status)
	}
}

// ---------------------------------------------------------------------------
// AssuranceMap tests
// ---------------------------------------------------------------------------

func TestNewAssuranceMap_AutoMatrix(t *testing.T) {
	req := CreateAssuranceMapRequest{
		OrgID:          testOrgID,
		Year:           2025,
		Name:           "Assurance Map 2025",
		BusinessUnits:  []string{"Finance", "IT", "HR"},
		ControlDomains: []string{"IAM", "Data Protection", "Network Security"},
	}
	am, err := NewAssuranceMap(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expectedCells := 3 * 3 // 3 BUs x 3 CDs
	if len(am.Matrix) != expectedCells {
		t.Errorf("expected %d cells, got %d", expectedCells, len(am.Matrix))
	}
	for _, cell := range am.Matrix {
		if cell.Coverage != CoverageNone {
			t.Errorf("expected all cells to start at coverage_none, got %s", cell.Coverage)
		}
	}
}

func TestAssuranceMap_UpdateMatrix(t *testing.T) {
	req := CreateAssuranceMapRequest{
		OrgID: testOrgID, Year: 2025, Name: "AM",
		BusinessUnits: []string{"IT"}, ControlDomains: []string{"IAM"},
	}
	am, _ := NewAssuranceMap(req)

	newMatrix := []AssuranceCell{
		{BusinessUnit: "IT", ControlDomain: "IAM", Coverage: CoverageFull},
	}
	am.Apply(UpdateAssuranceMapRequest{Matrix: &newMatrix})

	if len(am.Matrix) != 1 {
		t.Fatalf("expected 1 cell, got %d", len(am.Matrix))
	}
	if am.Matrix[0].Coverage != CoverageFull {
		t.Errorf("expected full coverage, got %s", am.Matrix[0].Coverage)
	}
}

// ---------------------------------------------------------------------------
// ResourceCalendar tests
// ---------------------------------------------------------------------------

func TestNewResourceCalendar_HappyPath(t *testing.T) {
	req := CreateResourceCalendarRequest{
		OrgID: testOrgID,
		Year:  2025,
		Name:  "Team Calendar 2025",
		Auditors: []AuditorAllocation{
			{
				AuditorName:  "Alice Smith",
				AuditorEmail: "alice@example.com",
				Role:         "senior_auditor",
				AvailableDays: 220,
				Assignments: []Assignment{
					{EngagementName: "IAM Audit", StartDate: "2025-01-15", EndDate: "2025-01-30", AllocatedDays: 10, Quarter: 1},
				},
			},
		},
	}
	rc, err := NewResourceCalendar(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rc.Auditors) != 1 {
		t.Fatalf("expected 1 auditor, got %d", len(rc.Auditors))
	}
	a := rc.Auditors[0]
	if a.AllocatedDays() != 10 {
		t.Errorf("expected 10 allocated days, got %d", a.AllocatedDays())
	}
	if a.UtilisationPct() != 4 { // 10/220 = 4%
		t.Errorf("expected 4%% utilisation, got %d%%", a.UtilisationPct())
	}
}

func TestAuditorAllocation_UtilisationCap(t *testing.T) {
	a := AuditorAllocation{
		AvailableDays: 100,
		Assignments: []Assignment{
			{AllocatedDays: 60},
			{AllocatedDays: 60}, // over 100%
		},
	}
	if pct := a.UtilisationPct(); pct != 100 {
		t.Errorf("expected utilisation capped at 100, got %d", pct)
	}
}
