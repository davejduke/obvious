// Package handler provides HTTP handlers for the reporting service.
package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/reporting/internal/generate"
	"github.com/davejduke/obvious/services/reporting/internal/template"
)

// ReportingHandler serves the reporting service endpoints.
type ReportingHandler struct {
	pdfGen   *generate.PDFGenerator
	excelGen *generate.ExcelGenerator
}

// NewReportingHandler creates a ReportingHandler with PDF and Excel generators.
func NewReportingHandler() *ReportingHandler {
	return &ReportingHandler{
		pdfGen:   generate.NewPDFGenerator(),
		excelGen: generate.NewExcelGenerator(),
	}
}

// RegisterRoutes mounts all reporting routes.
func (h *ReportingHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		reports := v1.Group("/reports")
		{
			reports.POST("/pdf", h.GeneratePDF)
			reports.POST("/excel", h.GenerateExcel)
		}
	}
}

// GeneratePDF handles POST /api/v1/reports/pdf
// Accepts a ReportData payload and returns application/pdf.
func (h *ReportingHandler) GeneratePDF(c *gin.Context) {
	data, ok := h.parseReportData(c)
	if !ok {
		return
	}

	pdf, err := h.pdfGen.Generate(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "GENERATION_ERROR", "message": err.Error()})
		return
	}

	filename := "audit-report-" + data.Metadata.EngagementID.String() + ".pdf"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/pdf", pdf)
}

// GenerateExcel handles POST /api/v1/reports/excel
// Returns a JSON payload with base64-encoded CSV sheets.
func (h *ReportingHandler) GenerateExcel(c *gin.Context) {
	data, ok := h.parseReportData(c)
	if !ok {
		return
	}

	out, err := h.excelGen.Generate(data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "GENERATION_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"engagement_id": data.Metadata.EngagementID,
		"sheets": gin.H{
			"summary":  string(out.Summary),
			"findings": string(out.Findings),
			"evidence": string(out.Evidence),
		},
		"generated_at": data.Metadata.GeneratedAt,
	})
}

// parseReportData binds and validates a report request body.
func (h *ReportingHandler) parseReportData(c *gin.Context) (template.ReportData, bool) {
	var req struct {
		EngagementID  string                   `json:"engagement_id"`
		OrgName       string                   `json:"org_name"`
		Framework     string                   `json:"framework"`
		ReportTitle   string                   `json:"report_title"`
		AuditorName   string                   `json:"auditor_name"`
		AuditorEmail  string                   `json:"auditor_email"`
		PeriodStart   time.Time                `json:"period_start"`
		PeriodEnd     time.Time                `json:"period_end"`
		Classification string                  `json:"classification"`
		ExecSummary   string                   `json:"exec_summary"`
		Findings      []template.Finding       `json:"findings"`
		Evidence      []template.EvidenceItem  `json:"evidence"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return template.ReportData{}, false
	}

	var engagementID uuid.UUID
	if req.EngagementID != "" {
		var err error
		engagementID, err = uuid.Parse(req.EngagementID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid engagement_id"})
			return template.ReportData{}, false
		}
	} else {
		engagementID = uuid.New()
	}

	data := template.ReportData{
		Metadata: template.ReportMetadata{
			ReportID:       uuid.New(),
			EngagementID:   engagementID,
			OrgName:        req.OrgName,
			Framework:      req.Framework,
			ReportTitle:    req.ReportTitle,
			AuditorName:    req.AuditorName,
			AuditorEmail:   req.AuditorEmail,
			PeriodStart:    req.PeriodStart,
			PeriodEnd:      req.PeriodEnd,
			GeneratedAt:    time.Now().UTC(),
			Classification: req.Classification,
		},
		ExecSummary: req.ExecSummary,
		Findings:    req.Findings,
		Evidence:    req.Evidence,
	}
	if data.Findings == nil {
		data.Findings = []template.Finding{}
	}
	if data.Evidence == nil {
		data.Evidence = []template.EvidenceItem{}
	}
	data.BuildSummary()
	return data, true
}

