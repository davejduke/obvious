// Package handler provides HTTP handlers for the reporting service.
package handler

import (
	"encoding/base64"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/reporting/internal/generate"
	"github.com/davejduke/obvious/services/reporting/internal/template"
)

// ReportingHandler serves the reporting service endpoints.
type ReportingHandler struct {
	pdfGen      *generate.PDFGenerator
	excelGen    *generate.ExcelGenerator
	evidenceGen *generate.EvidenceZIPGenerator
}

// NewReportingHandler creates a ReportingHandler with PDF, Excel, and evidence package generators.
func NewReportingHandler() *ReportingHandler {
	return &ReportingHandler{
		pdfGen:      generate.NewPDFGenerator(),
		excelGen:    generate.NewExcelGenerator(),
		evidenceGen: generate.NewEvidenceZIPGenerator(),
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

		// Evidence package export — POST /api/v1/engagements/:id/export
		engagements := v1.Group("/engagements")
		{
			engagements.POST("/:id/export", h.ExportEvidencePackage)
		}
	}
}

// ─── PDF / Excel handlers ────────────────────────────────────────────────────

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

// ─── Evidence package export ─────────────────────────────────────────────────

// ExportEvidencePackage handles POST /api/v1/engagements/:id/export
// Generates an encrypted evidence ZIP package for the given engagement.
// The package contains findings, evidence items, working papers, and an
// audit trail excerpt. When rsa_public_key_pem is set the ZIP is
// encrypted with AES-256-GCM and the session key is RSA-OAEP wrapped.
//
// Response (unencrypted):  application/zip binary download
// Response (encrypted):    JSON with base64-encoded encrypted_package_b64 and encrypted_key_b64
func (h *ReportingHandler) ExportEvidencePackage(c *gin.Context) {
	engagementID := c.Param("id")
	if engagementID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "engagement id is required"})
		return
	}

	var req struct {
		OrgName         string                     `json:"org_name"`
		Findings        []template.Finding         `json:"findings"`
		Evidence        []template.EvidenceItem    `json:"evidence"`
		WorkingPapers   []generate.WorkingPaper    `json:"working_papers"`
		AuditTrail      []generate.AuditTrailEntry `json:"audit_trail"`
		RSAPublicKeyPEM string                     `json:"rsa_public_key_pem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	pkgReq := generate.EvidencePackageRequest{
		EngagementID:    engagementID,
		OrgName:         req.OrgName,
		Findings:        req.Findings,
		Evidence:        req.Evidence,
		WorkingPapers:   req.WorkingPapers,
		AuditTrail:      req.AuditTrail,
		RSAPublicKeyPEM: req.RSAPublicKeyPEM,
	}
	if pkgReq.Findings == nil {
		pkgReq.Findings = []template.Finding{}
	}
	if pkgReq.Evidence == nil {
		pkgReq.Evidence = []template.EvidenceItem{}
	}

	result, err := h.evidenceGen.Generate(pkgReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "GENERATION_ERROR", "message": err.Error()})
		return
	}

	if !result.Encrypted {
		filename := "evidence-package-" + engagementID + ".zip"
		c.Header("Content-Disposition", "attachment; filename="+filename)
		c.Data(http.StatusOK, "application/zip", result.PackageZIP)
		return
	}

	// Encrypted: return JSON envelope with base64 payloads so the response
	// is easily consumed over JSON APIs without binary framing issues.
	c.JSON(http.StatusOK, gin.H{
		"engagement_id":         engagementID,
		"encrypted":             true,
		"manifest":              result.Manifest,
		"encrypted_package_b64": base64.StdEncoding.EncodeToString(result.EncryptedPackage),
		"encrypted_key_b64":     base64.StdEncoding.EncodeToString(result.EncryptedSessionKey),
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// parseReportData binds and validates a report request body.
func (h *ReportingHandler) parseReportData(c *gin.Context) (template.ReportData, bool) {
	var req struct {
		EngagementID   string                  `json:"engagement_id"`
		OrgName        string                  `json:"org_name"`
		Framework      string                  `json:"framework"`
		ReportTitle    string                  `json:"report_title"`
		AuditorName    string                  `json:"auditor_name"`
		AuditorEmail   string                  `json:"auditor_email"`
		PeriodStart    time.Time               `json:"period_start"`
		PeriodEnd      time.Time               `json:"period_end"`
		Classification string                  `json:"classification"`
		ExecSummary    string                  `json:"exec_summary"`
		Findings       []template.Finding      `json:"findings"`
		Evidence       []template.EvidenceItem `json:"evidence"`
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
