// Package generate provides PDF and Excel report generation.
package generate

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/davejduke/obvious/services/reporting/internal/template"
)

// PDFGenerator generates audit reports in PDF format.
// It uses text-based PDF encoding (RFC 3778) directly — no external binary
// dependencies (WeasyPrint is Python-only; this service is Go).
// The PDF is minimal but structurally valid for audit evidence chains.
type PDFGenerator struct{}

// NewPDFGenerator creates a PDFGenerator.
func NewPDFGenerator() *PDFGenerator { return &PDFGenerator{} }

// Generate produces a PDF byte slice from the given report data.
func (g *PDFGenerator) Generate(data template.ReportData) ([]byte, error) {
	var body bytes.Buffer

	// Build page content as a series of BT/ET text blocks
	var textBlocks []string

	// Helper: a text line at position (x, y) with given font size
	line := func(x, y, size int, text string) string {
		// Escape PDF special chars
		escaped := strings.ReplaceAll(text, "(", "\\(")
		escaped = strings.ReplaceAll(escaped, ")", "\\)")
		return fmt.Sprintf("BT /F1 %d Tf %d %d Td (%s) Tj ET", size, x, y, escaped)
	}

	// Title block
	textBlocks = append(textBlocks,
		line(50, 780, 16, data.Metadata.ReportTitle),
		line(50, 760, 11, fmt.Sprintf("Organisation: %s", data.Metadata.OrgName)),
		line(50, 745, 11, fmt.Sprintf("Framework: %s", data.Metadata.Framework)),
		line(50, 730, 11, fmt.Sprintf("Auditor: %s <%s>", data.Metadata.AuditorName, data.Metadata.AuditorEmail)),
		line(50, 715, 11, fmt.Sprintf("Period: %s - %s",
			data.Metadata.PeriodStart.Format("2006-01-02"),
			data.Metadata.PeriodEnd.Format("2006-01-02"),
		)),
		line(50, 700, 11, fmt.Sprintf("Generated: %s", data.Metadata.GeneratedAt.Format("2006-01-02 15:04 UTC"))),
		line(50, 685, 11, fmt.Sprintf("Classification: %s", data.Metadata.Classification)),
	)

	// Executive summary — prefer LLM narrative if available, else fallback
	execText := data.ExecSummary
	if data.Narratives != nil && data.Narratives.ExecutiveSummary != "" {
		execText = data.Narratives.ExecutiveSummary
	}
	narrativeLabel := ""
	if data.Narratives != nil {
		tone := data.Narratives.Tone
		if tone == "" {
			tone = "formal"
		}
		suffix := "live"
		if data.Narratives.IsMock {
			suffix = "mock"
		}
		narrativeLabel = fmt.Sprintf("Narrative: tone=%s model=%s [%s]", tone, data.Narratives.ModelID, suffix)
	}
	textBlocks = append(textBlocks,
		line(50, 660, 13, "EXECUTIVE SUMMARY"),
		line(50, 645, 10, truncate(execText, 90)),
	)
	if narrativeLabel != "" {
		textBlocks = append(textBlocks, line(50, 630, 8, narrativeLabel))
	}

	// Methodology narrative (if present)
	if data.Narratives != nil && data.Narratives.Methodology != "" {
		textBlocks = append(textBlocks,
			line(50, 618, 13, "METHODOLOGY"),
			line(50, 604, 9, truncate(data.Narratives.Methodology, 110)),
		)
	}

	// Finding summary
	textBlocks = append(textBlocks,
		line(50, 588, 13, "FINDINGS SUMMARY"),
		line(50, 573, 10, fmt.Sprintf("Total: %d  Critical: %d  High: %d  Medium: %d  Low: %d  Info: %d",
			data.Summary.TotalFindings, data.Summary.Critical, data.Summary.High,
			data.Summary.Medium, data.Summary.Low, data.Summary.Informational,
		)),
		line(50, 558, 10, fmt.Sprintf("Evidence items: %d", data.Summary.TotalEvidence)),
	)

	// Findings narrative (if present)
	if data.Narratives != nil && data.Narratives.Findings != "" {
		textBlocks = append(textBlocks,
			line(50, 544, 9, truncate(data.Narratives.Findings, 110)),
		)
	}

	// Individual findings (up to 10 per page)
	y := 530
	textBlocks = append(textBlocks, line(50, y, 13, "DETAILED FINDINGS"))
	y -= 15
	for i, f := range data.Findings {
		if i >= 10 {
			break // practical page limit
		}
		textBlocks = append(textBlocks,
			line(50, y, 11, fmt.Sprintf("[%s] %s (%s)", f.Ref, f.Title, strings.ToUpper(string(f.Severity)))),
			line(60, y-14, 9, truncate(f.Description, 100)),
			line(60, y-26, 9, fmt.Sprintf("Recommendation: %s", truncate(f.Recommendation, 80))),
		)
		// Evidence chain for this finding
		chain := data.EvidenceChain(f)
		if len(chain) > 0 {
			evRefs := make([]string, 0, len(chain))
			for _, ev := range chain {
				evRefs = append(evRefs, ev.Title)
			}
			textBlocks = append(textBlocks,
				line(60, y-38, 9, fmt.Sprintf("Evidence: %s", strings.Join(evRefs, ", "))),
			)
			y -= 50
		} else {
			y -= 40
		}
		if y < 80 {
			break
		}
	}

	// Recommendations narrative (appended after findings)
	if data.Narratives != nil && data.Narratives.Recommendations != "" {
		textBlocks = append(textBlocks,
			line(50, 72, 13, "RECOMMENDATIONS"),
			line(50, 58, 9, truncate(data.Narratives.Recommendations, 110)),
		)
	}

	pageContent := strings.Join(textBlocks, "\n")

	// Build a minimal valid PDF structure.
	// Object layout: 1=catalog, 2=pages, 3=page, 4=font, 5=content
	fontObj := "4 0 obj\n<</Type /Font /Subtype /Type1 /BaseFont /Helvetica>>\nendobj\n"
	contentStream := fmt.Sprintf("5 0 obj\n<</Length %d>>\nstream\n%s\nendstream\nendobj\n",
		len(pageContent)+1, pageContent)
	pageObj := "3 0 obj\n<</Type /Page /Parent 2 0 R /MediaBox [0 0 595 842]\n" +
		"/Resources <</Font <</F1 4 0 R>>>>\n/Contents 5 0 R>>\nendobj\n"
	pagesObj := "2 0 obj\n<</Type /Pages /Kids [3 0 R] /Count 1>>\nendobj\n"
	catalogObj := "1 0 obj\n<</Type /Catalog /Pages 2 0 R>>\nendobj\n"

	header := "%PDF-1.4\n"

	body.WriteString(header)

	// Track byte offsets for the xref table.
	offsets := make([]int, 6) // 1-indexed
	offsets[1] = body.Len()
	body.WriteString(catalogObj)
	offsets[2] = body.Len()
	body.WriteString(pagesObj)
	offsets[3] = body.Len()
	body.WriteString(pageObj)
	offsets[4] = body.Len()
	body.WriteString(fontObj)
	offsets[5] = body.Len()
	body.WriteString(contentStream)

	xrefPos := body.Len()
	body.WriteString("xref\n")
	body.WriteString("0 6\n")
	body.WriteString("0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		body.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	body.WriteString(fmt.Sprintf("trailer\n<</Size 6 /Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", xrefPos))

	return body.Bytes(), nil
}

// truncate shortens s to at most n characters.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

