package generate

import (
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"github.com/davejduke/obvious/services/reporting/internal/template"
)

// ExcelGenerator generates audit report data as Excel-compatible CSV.
// A pure Go CSV approach removes the xlsx/CGO dependency constraint while
// remaining importable directly by Excel and Google Sheets.
type ExcelGenerator struct{}

// NewExcelGenerator creates an ExcelGenerator.
func NewExcelGenerator() *ExcelGenerator { return &ExcelGenerator{} }

// ExcelOutput holds the generated sheets as CSV bytes.
type ExcelOutput struct {
	// Summary contains the executive summary sheet.
	Summary []byte
	// Findings contains the detailed findings sheet.
	Findings []byte
	// Evidence contains the evidence chain sheet.
	Evidence []byte
}

// Generate creates a multi-sheet Excel-compatible CSV export.
func (g *ExcelGenerator) Generate(data template.ReportData) (ExcelOutput, error) {
	summaryBytes, err := g.generateSummarySheet(data)
	if err != nil {
		return ExcelOutput{}, fmt.Errorf("excel: summary sheet: %w", err)
	}
	findingsBytes, err := g.generateFindingsSheet(data)
	if err != nil {
		return ExcelOutput{}, fmt.Errorf("excel: findings sheet: %w", err)
	}
	evidenceBytes, err := g.generateEvidenceSheet(data)
	if err != nil {
		return ExcelOutput{}, fmt.Errorf("excel: evidence sheet: %w", err)
	}
	return ExcelOutput{
		Summary:  summaryBytes,
		Findings: findingsBytes,
		Evidence: evidenceBytes,
	}, nil
}

func (g *ExcelGenerator) generateSummarySheet(data template.ReportData) ([]byte, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)

	_ = w.Write([]string{"Field", "Value"})
	_ = w.Write([]string{"Report Title", data.Metadata.ReportTitle})
	_ = w.Write([]string{"Organisation", data.Metadata.OrgName})
	_ = w.Write([]string{"Framework", data.Metadata.Framework})
	_ = w.Write([]string{"Auditor", fmt.Sprintf("%s <%s>", data.Metadata.AuditorName, data.Metadata.AuditorEmail)})
	_ = w.Write([]string{"Period Start", data.Metadata.PeriodStart.Format(time.RFC3339)})
	_ = w.Write([]string{"Period End", data.Metadata.PeriodEnd.Format(time.RFC3339)})
	_ = w.Write([]string{"Generated At", data.Metadata.GeneratedAt.Format(time.RFC3339)})
	_ = w.Write([]string{"Classification", data.Metadata.Classification})
	_ = w.Write([]string{"", ""})
	_ = w.Write([]string{"Total Findings", fmt.Sprintf("%d", data.Summary.TotalFindings)})
	_ = w.Write([]string{"Critical", fmt.Sprintf("%d", data.Summary.Critical)})
	_ = w.Write([]string{"High", fmt.Sprintf("%d", data.Summary.High)})
	_ = w.Write([]string{"Medium", fmt.Sprintf("%d", data.Summary.Medium)})
	_ = w.Write([]string{"Low", fmt.Sprintf("%d", data.Summary.Low)})
	_ = w.Write([]string{"Informational", fmt.Sprintf("%d", data.Summary.Informational)})
	_ = w.Write([]string{"Total Evidence", fmt.Sprintf("%d", data.Summary.TotalEvidence)})
	_ = w.Write([]string{"", ""})
	_ = w.Write([]string{"Executive Summary", data.ExecSummary})

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}

func (g *ExcelGenerator) generateFindingsSheet(data template.ReportData) ([]byte, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)

	_ = w.Write([]string{"Ref", "Title", "Severity", "Description", "Recommendation", "Control Ref", "Evidence Count"})

	for _, f := range data.Findings {
		chain := data.EvidenceChain(f)
		_ = w.Write([]string{
			f.Ref,
			f.Title,
			string(f.Severity),
			f.Description,
			f.Recommendation,
			f.ControlRef,
			fmt.Sprintf("%d", len(chain)),
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}

func (g *ExcelGenerator) generateEvidenceSheet(data template.ReportData) ([]byte, error) {
	var sb strings.Builder
	w := csv.NewWriter(&sb)

	_ = w.Write([]string{"ID", "Title", "Source Type", "Collected At", "Collected By", "Hash", "Integration Source"})

	for _, ev := range data.Evidence {
		_ = w.Write([]string{
			ev.ID.String(),
			ev.Title,
			ev.SourceType,
			ev.CollectedAt.Format(time.RFC3339),
			ev.CollectedBy,
			ev.Hash,
			ev.IntegrationSrc,
		})
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return []byte(sb.String()), nil
}

