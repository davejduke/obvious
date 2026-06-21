package generate_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/reporting/internal/generate"
	"github.com/davejduke/obvious/services/reporting/internal/template"
)

func buildSampleReport() template.ReportData {
	ev1ID := uuid.New()
	ev2ID := uuid.New()

	data := template.ReportData{
		Metadata: template.ReportMetadata{
			ReportID:       uuid.New(),
			EngagementID:   uuid.New(),
			OrgName:        "Acme Corp",
			Framework:      "NIS 2 Article 21",
			ReportTitle:    "NIS2 Compliance Audit Report",
			AuditorName:    "Alice Auditor",
			AuditorEmail:   "alice@auditor.com",
			PeriodStart:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			PeriodEnd:      time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
			GeneratedAt:    time.Date(2024, 7, 1, 12, 0, 0, 0, time.UTC),
			Classification: "CONFIDENTIAL",
		},
		ExecSummary: "This audit assessed NIS2 Article 21 compliance for the period H1 2024.",
		Findings: []template.Finding{
			{
				ID:             uuid.New(),
				Ref:            "NIS2-F-001",
				Title:          "Missing MFA on privileged accounts",
				Description:    "Multi-factor authentication is not enforced for all privileged accounts.",
				Severity:       template.SeverityHigh,
				Recommendation: "Enforce MFA for all privileged accounts within 30 days.",
				ControlRef:     "NIS2-21b",
				EvidenceRefs:   []string{ev1ID.String()},
			},
			{
				ID:             uuid.New(),
				Ref:            "NIS2-F-002",
				Title:          "Incomplete incident response plan",
				Description:    "The incident response plan does not cover supply chain scenarios.",
				Severity:       template.SeverityMedium,
				Recommendation: "Update the incident response plan to include supply chain scenarios.",
				ControlRef:     "NIS2-21c",
				EvidenceRefs:   []string{ev2ID.String()},
			},
		},
		Evidence: []template.EvidenceItem{
			{
				ID:             ev1ID,
				Title:          "Azure AD MFA Report",
				Description:    "Exported from Azure AD showing accounts without MFA.",
				SourceType:     "api_integration",
				CollectedAt:    time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC),
				CollectedBy:    "sentinel",
				IntegrationSrc: "sentinel",
			},
			{
				ID:             ev2ID,
				Title:          "IR Plan Document v2.1",
				Description:    "Current incident response plan document.",
				SourceType:     "manual_upload",
				CollectedAt:    time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC),
				CollectedBy:    "alice@auditor.com",
			},
		},
	}
	data.BuildSummary()
	return data
}

func TestPDFGenerator_GeneratesValidPDF(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReport()

	pdf, err := gen.Generate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pdf) == 0 {
		t.Fatal("expected non-empty PDF")
	}
	// Must start with %PDF
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Errorf("PDF does not start with PDF header, got: %s", pdf[:10])
	}
	// Must contain %%EOF
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Error("PDF does not contain EOF trailer")
	}
}

func TestPDFGenerator_ContainsTitle(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReport()
	pdf, _ := gen.Generate(data)

	if !bytes.Contains(pdf, []byte("NIS2 Compliance Audit Report")) {
		t.Error("PDF does not contain report title")
	}
}

func TestPDFGenerator_ContainsFinding(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReport()
	pdf, _ := gen.Generate(data)

	if !bytes.Contains(pdf, []byte("NIS2-F-001")) {
		t.Error("PDF does not contain finding ref NIS2-F-001")
	}
}

func TestPDFGenerator_ContainsEvidenceChain(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReport()
	pdf, _ := gen.Generate(data)

	// Evidence chain should be referenced
	if !bytes.Contains(pdf, []byte("Azure AD MFA Report")) {
		t.Error("PDF does not contain evidence chain reference")
	}
}

func TestExcelGenerator_GeneratesSummary(t *testing.T) {
	gen := generate.NewExcelGenerator()
	data := buildSampleReport()

	out, err := gen.Generate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Summary) == 0 {
		t.Fatal("expected non-empty summary CSV")
	}
	summary := string(out.Summary)
	if !strings.Contains(summary, "NIS2 Compliance Audit Report") {
		t.Error("summary CSV missing report title")
	}
	if !strings.Contains(summary, "CONFIDENTIAL") {
		t.Error("summary CSV missing classification")
	}
}

func TestExcelGenerator_GeneratesFindings(t *testing.T) {
	gen := generate.NewExcelGenerator()
	data := buildSampleReport()

	out, _ := gen.Generate(data)
	findings := string(out.Findings)

	if !strings.Contains(findings, "NIS2-F-001") {
		t.Error("findings CSV missing NIS2-F-001")
	}
	if !strings.Contains(findings, "high") {
		t.Error("findings CSV missing severity=high")
	}
	if !strings.Contains(findings, "NIS2-21b") {
		t.Error("findings CSV missing control ref NIS2-21b")
	}
}

func TestExcelGenerator_GeneratesEvidence(t *testing.T) {
	gen := generate.NewExcelGenerator()
	data := buildSampleReport()

	out, _ := gen.Generate(data)
	evidence := string(out.Evidence)

	if !strings.Contains(evidence, "Azure AD MFA Report") {
		t.Error("evidence CSV missing Azure AD MFA Report")
	}
	if !strings.Contains(evidence, "sentinel") {
		t.Error("evidence CSV missing sentinel integration source")
	}
}

func TestReportData_BuildSummary(t *testing.T) {
	data := buildSampleReport()

	if data.Summary.TotalFindings != 2 {
		t.Errorf("expected 2 findings, got %d", data.Summary.TotalFindings)
	}
	if data.Summary.High != 1 {
		t.Errorf("expected 1 high finding, got %d", data.Summary.High)
	}
	if data.Summary.Medium != 1 {
		t.Errorf("expected 1 medium finding, got %d", data.Summary.Medium)
	}
	if data.Summary.TotalEvidence != 2 {
		t.Errorf("expected 2 evidence items, got %d", data.Summary.TotalEvidence)
	}
}

func TestReportData_EvidenceChain(t *testing.T) {
	data := buildSampleReport()

	chain := data.EvidenceChain(data.Findings[0])
	if len(chain) != 1 {
		t.Errorf("expected 1 evidence item in chain, got %d", len(chain))
	}
	if chain[0].Title != "Azure AD MFA Report" {
		t.Errorf("unexpected evidence title: %s", chain[0].Title)
	}
}

// ---------------------------------------------------------------------------
// Working paper narrative PDF tests
// ---------------------------------------------------------------------------

func buildSampleReportWithNarratives() template.ReportData {
	data := buildSampleReport()
	data.Narratives = &template.WorkingPaperNarratives{
		ModelID:          "anthropic.claude-sonnet-3-7-20250219-v1:0#mock",
		Tone:             "formal",
		IsMock:           true,
		ExecutiveSummary: "This formal executive summary wraps the deterministic audit conclusion: Effective at 87.5% confidence.",
		Methodology:      "Audit conducted per IIA Standard 4.1 using Cochran 1977 sampling and Bayesian risk scoring.",
		Findings:         "2 findings detected. Risk scores are deterministic; this text is LLM-generated narrative only.",
		Recommendations:  "Immediate action required on 0 critical items. Follow-up engagement recommended within 90 days.",
	}
	return data
}

func TestPDFGenerator_WithNarratives_GeneratesValidPDF(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReportWithNarratives()

	pdf, err := gen.Generate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF")) {
		t.Error("PDF with narratives does not start with PDF header")
	}
	if !bytes.Contains(pdf, []byte("%%EOF")) {
		t.Error("PDF with narratives missing EOF trailer")
	}
}

func TestPDFGenerator_WithNarratives_ContainsExecutiveSummaryText(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReportWithNarratives()
	pdf, _ := gen.Generate(data)

	// LLM narrative executive summary should appear in the PDF
	if !bytes.Contains(pdf, []byte("deterministic audit conclusion")) {
		t.Error("PDF missing LLM executive summary narrative text")
	}
}

func TestPDFGenerator_WithNarratives_ContainsMethodologySection(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReportWithNarratives()
	pdf, _ := gen.Generate(data)

	if !bytes.Contains(pdf, []byte("METHODOLOGY")) {
		t.Error("PDF missing METHODOLOGY section header")
	}
	if !bytes.Contains(pdf, []byte("IIA Standard 4.1")) {
		t.Error("PDF missing IIA Standard 4.1 reference in methodology")
	}
}

func TestPDFGenerator_WithNarratives_ContainsRecommendationsSection(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReportWithNarratives()
	pdf, _ := gen.Generate(data)

	if !bytes.Contains(pdf, []byte("RECOMMENDATIONS")) {
		t.Error("PDF missing RECOMMENDATIONS section")
	}
}

func TestPDFGenerator_WithNarratives_ContainsNarrativeLabel(t *testing.T) {
	gen := generate.NewPDFGenerator()
	data := buildSampleReportWithNarratives()
	pdf, _ := gen.Generate(data)

	// Narrative label should include tone and mock marker
	if !bytes.Contains(pdf, []byte("tone=formal")) {
		t.Error("PDF missing narrative tone label")
	}
	if !bytes.Contains(pdf, []byte("[mock]")) {
		t.Error("PDF missing mock marker in narrative label")
	}
}

func TestPDFGenerator_WithNarratives_DeterministicFindingsPreserved(t *testing.T) {
	// Deterministic findings must still appear even with narratives present
	gen := generate.NewPDFGenerator()
	data := buildSampleReportWithNarratives()
	pdf, _ := gen.Generate(data)

	if !bytes.Contains(pdf, []byte("NIS2-F-001")) {
		t.Error("deterministic finding ref NIS2-F-001 missing from narrative PDF")
	}
	if !bytes.Contains(pdf, []byte("NIS2-F-002")) {
		t.Error("deterministic finding ref NIS2-F-002 missing from narrative PDF")
	}
}

func TestPDFGenerator_NilNarratives_FallsBackToExecSummary(t *testing.T) {
	// When Narratives is nil, ExecSummary field should be used (backwards compat)
	gen := generate.NewPDFGenerator()
	data := buildSampleReport() // Narratives == nil
	pdf, err := gen.Generate(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(pdf, []byte("NIS2 Article 21 compliance")) {
		t.Error("fallback ExecSummary text missing when Narratives is nil")
	}
}

func TestWorkingPaperNarratives_JSONRoundTrip(t *testing.T) {
	raw := `{"model_id":"anthropic.claude-sonnet-3-7#mock","tone":"executive","is_mock":true,"executive_summary":"Exec summary text","methodology":"Method text","findings":"Findings text","recommendations":"Recs text"}`
	var n template.WorkingPaperNarratives
	if err := json.Unmarshal([]byte(raw), &n); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if n.ExecutiveSummary != "Exec summary text" {
		t.Errorf("unexpected ExecutiveSummary: %s", n.ExecutiveSummary)
	}
	if n.Tone != "executive" {
		t.Errorf("unexpected Tone: %s", n.Tone)
	}
	if !n.IsMock {
		t.Error("IsMock should be true")
	}
}

func TestWorkingPaperNarratives_FieldsPresent(t *testing.T) {
	n := template.WorkingPaperNarratives{
		ModelID:          "anthropic.claude-sonnet-3-7#mock",
		Tone:             "technical",
		IsMock:           true,
		ExecutiveSummary: "exec",
		Methodology:      "method",
		Findings:         "findings",
		Recommendations:  "recs",
	}
	if n.ExecutiveSummary != "exec" {
		t.Error("ExecutiveSummary field not set")
	}
	if !n.IsMock {
		t.Error("IsMock should be true for stub")
	}
	if n.Tone != "technical" {
		t.Error("Tone not set correctly")
	}
}

