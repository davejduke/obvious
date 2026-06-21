// Package handler provides the HTTP handlers for the integration gateway.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/davejduke/obvious/services/integration/internal/connector"
	"github.com/davejduke/obvious/services/integration/internal/grc"
	"github.com/davejduke/obvious/services/integration/internal/vulnconnector"
)

// IntegrationHandler serves the integration gateway endpoints.
type IntegrationHandler struct {
	registry     *connector.Registry
	grcRegistry  *grc.Registry
	vulnRegistry *vulnconnector.VulnRegistry
}

// NewIntegrationHandler creates a SIEM-only handler (no GRC or vuln registries).
func NewIntegrationHandler(reg *connector.Registry) *IntegrationHandler {
	return &IntegrationHandler{
		registry:     reg,
		grcRegistry:  grc.NewRegistry(),
		vulnRegistry: vulnconnector.NewVulnRegistry(),
	}
}

// NewIntegrationHandlerWithGRC creates a handler with SIEM and GRC registries.
func NewIntegrationHandlerWithGRC(reg *connector.Registry, grcReg *grc.Registry) *IntegrationHandler {
	return &IntegrationHandler{
		registry:     reg,
		grcRegistry:  grcReg,
		vulnRegistry: vulnconnector.NewVulnRegistry(),
	}
}

// NewIntegrationHandlerFull creates a handler with all three registries
// (SIEM, GRC outbound, vulnerability/endpoint).
func NewIntegrationHandlerFull(reg *connector.Registry, grcReg *grc.Registry, vulnReg *vulnconnector.VulnRegistry) *IntegrationHandler {
	return &IntegrationHandler{registry: reg, grcRegistry: grcReg, vulnRegistry: vulnReg}
}

// RegisterRoutes mounts all integration routes.
func (h *IntegrationHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		// SIEM integration routes
		integrations := v1.Group("/integrations")
		{
			integrations.GET("", h.ListConnectors)
			integrations.GET("/:connector/health", h.ConnectorHealth)
			integrations.GET("/:connector/logs", h.FetchLogs)
		}

		// GRC outbound routes
		grcGroup := v1.Group("/grc")
		{
			grcGroup.GET("", h.ListGRCConnectors)
			grcGroup.GET("/:connector/health", h.GRCConnectorHealth)
		}

		// Engagement export routes
		engagements := v1.Group("/engagements")
		{
			engagements.POST("/:id/export/servicenow", h.ExportToServiceNow)
		}

		// Vulnerability/endpoint connectors (Qualys, Tenable, CrowdStrike)
		vulnGroup := v1.Group("/vuln")
		{
			vulnGroup.GET("", h.ListVulnConnectors)
			vulnGroup.GET("/:connector/health", h.VulnConnectorHealth)
			vulnGroup.GET("/:connector/vulnerabilities", h.FetchVulnerabilities)
			vulnGroup.GET("/:connector/endpoints", h.FetchEndpoints)
			vulnGroup.GET("/:connector/evidence", h.FetchEvidence)
		}
	}
}

// ─── SIEM handlers ──────────────────────────────────────────────────────────

// ListConnectors handles GET /api/v1/integrations
func (h *IntegrationHandler) ListConnectors(c *gin.Context) {
	names := h.registry.List()
	c.JSON(http.StatusOK, gin.H{"connectors": names, "total": len(names)})
}

// ConnectorHealth handles GET /api/v1/integrations/:connector/health
func (h *IntegrationHandler) ConnectorHealth(c *gin.Context) {
	conn, ok := h.resolveConnector(c)
	if !ok {
		return
	}
	status := conn.Health(c.Request.Context())
	httpStatus := http.StatusOK
	if !status.Healthy {
		httpStatus = http.StatusServiceUnavailable
	}
	c.JSON(httpStatus, status)
}

// FetchLogs handles GET /api/v1/integrations/:connector/logs
func (h *IntegrationHandler) FetchLogs(c *gin.Context) {
	conn, ok := h.resolveConnector(c)
	if !ok {
		return
	}

	opts := connector.QueryOptions{}
	if q := c.Query("query"); q != "" {
		opts.Query = q
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30)
	defer cancel()

	logs, err := conn.FetchLogs(ctx, opts)
	if err != nil {
		switch err {
		case connector.ErrCircuitOpen:
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"code":    "CIRCUIT_OPEN",
				"message": "connector circuit breaker is open; upstream is unavailable",
			})
		default:
			c.JSON(http.StatusBadGateway, gin.H{"code": "CONNECTOR_ERROR", "message": err.Error()})
		}
		return
	}

	if logs == nil {
		logs = []connector.LogEntry{}
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs, "total": len(logs), "connector": conn.Name()})
}

// ─── GRC handlers ───────────────────────────────────────────────────────────

// ListGRCConnectors handles GET /api/v1/grc
func (h *IntegrationHandler) ListGRCConnectors(c *gin.Context) {
	names := h.grcRegistry.List()
	c.JSON(http.StatusOK, gin.H{"connectors": names, "total": len(names)})
}

// GRCConnectorHealth handles GET /api/v1/grc/:connector/health
func (h *IntegrationHandler) GRCConnectorHealth(c *gin.Context) {
	conn, ok := h.resolveGRCConnector(c)
	if !ok {
		return
	}
	status := conn.Health(c.Request.Context())
	httpStatus := http.StatusOK
	if !status.Healthy {
		httpStatus = http.StatusServiceUnavailable
	}
	c.JSON(httpStatus, status)
}

// ExportToServiceNow handles POST /api/v1/engagements/:id/export/servicenow
func (h *IntegrationHandler) ExportToServiceNow(c *gin.Context) {
	engagementID := c.Param("id")
	if engagementID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "engagement id is required"})
		return
	}

	conn, ok := h.resolveGRCConnector(c)
	if !ok {
		snConn, found := h.grcRegistry.Get("servicenow")
		if !found {
			c.JSON(http.StatusNotFound, gin.H{
				"code":      "CONNECTOR_NOT_FOUND",
				"message":   "servicenow grc connector not registered",
				"available": h.grcRegistry.List(),
			})
			return
		}
		conn = snConn
	}

	var req struct {
		Findings []grc.Finding `json:"findings"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	result, err := conn.ExportFindings(c.Request.Context(), req.Findings)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": "EXPORT_ERROR", "message": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"engagement_id": engagementID,
		"export":        result,
	})
}

// ─── Vulnerability / endpoint connector handlers ─────────────────────────────

// ListVulnConnectors handles GET /api/v1/vuln
func (h *IntegrationHandler) ListVulnConnectors(c *gin.Context) {
	names := h.vulnRegistry.List()
	c.JSON(http.StatusOK, gin.H{"connectors": names, "total": len(names)})
}

// VulnConnectorHealth handles GET /api/v1/vuln/:connector/health
func (h *IntegrationHandler) VulnConnectorHealth(c *gin.Context) {
	conn, ok := h.resolveVulnConnector(c)
	if !ok {
		return
	}
	status := conn.Health(c.Request.Context())
	httpStatus := http.StatusOK
	if !status.Healthy {
		httpStatus = http.StatusServiceUnavailable
	}
	c.JSON(httpStatus, status)
}

// FetchVulnerabilities handles GET /api/v1/vuln/:connector/vulnerabilities
func (h *IntegrationHandler) FetchVulnerabilities(c *gin.Context) {
	conn, ok := h.resolveVulnConnector(c)
	if !ok {
		return
	}
	opts := vulnconnector.QueryOptions{}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	findings, err := conn.FetchVulnerabilities(ctx, opts)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": "CONNECTOR_ERROR", "message": err.Error()})
		return
	}
	if findings == nil {
		findings = []vulnconnector.VulnFinding{}
	}
	c.JSON(http.StatusOK, gin.H{"vulnerabilities": findings, "total": len(findings), "connector": conn.Name()})
}

// FetchEndpoints handles GET /api/v1/vuln/:connector/endpoints
func (h *IntegrationHandler) FetchEndpoints(c *gin.Context) {
	conn, ok := h.resolveVulnConnector(c)
	if !ok {
		return
	}
	opts := vulnconnector.QueryOptions{}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	devices, err := conn.FetchEndpoints(ctx, opts)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": "CONNECTOR_ERROR", "message": err.Error()})
		return
	}
	if devices == nil {
		devices = []vulnconnector.EndpointDevice{}
	}
	c.JSON(http.StatusOK, gin.H{"endpoints": devices, "total": len(devices), "connector": conn.Name()})
}

// FetchEvidence handles GET /api/v1/vuln/:connector/evidence
// Returns vulnerability findings and endpoint devices mapped to AIAUDITOR EvidenceItems.
func (h *IntegrationHandler) FetchEvidence(c *gin.Context) {
	conn, ok := h.resolveVulnConnector(c)
	if !ok {
		return
	}
	opts := vulnconnector.QueryOptions{}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	findings, err := conn.FetchVulnerabilities(ctx, opts)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": "CONNECTOR_ERROR", "message": err.Error()})
		return
	}
	devices, err := conn.FetchEndpoints(ctx, opts)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"code": "CONNECTOR_ERROR", "message": err.Error()})
		return
	}

	vulnEvidence := vulnconnector.VulnFindingsToEvidence(findings)
	endpointEvidence := vulnconnector.EndpointDevicesToEvidence(devices)

	allEvidence := make([]vulnconnector.EvidenceItem, 0, len(vulnEvidence)+len(endpointEvidence))
	allEvidence = append(allEvidence, vulnEvidence...)
	allEvidence = append(allEvidence, endpointEvidence...)

	c.JSON(http.StatusOK, gin.H{
		"evidence":  allEvidence,
		"total":     len(allEvidence),
		"connector": conn.Name(),
	})
}

// ─── helpers ────────────────────────────────────────────────────────────────

// resolveConnector looks up the SIEM connector by the :connector path param.
func (h *IntegrationHandler) resolveConnector(c *gin.Context) (connector.Connector, bool) {
	name := c.Param("connector")
	conn, ok := h.registry.Get(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":      "CONNECTOR_NOT_FOUND",
			"message":   "connector not registered: " + name,
			"available": h.registry.List(),
		})
		return nil, false
	}
	return conn, true
}

// resolveGRCConnector looks up the GRC connector by the :connector path param.
func (h *IntegrationHandler) resolveGRCConnector(c *gin.Context) (grc.GRCConnector, bool) {
	name := c.Param("connector")
	if name == "" {
		return nil, false
	}
	conn, ok := h.grcRegistry.Get(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":      "CONNECTOR_NOT_FOUND",
			"message":   "grc connector not registered: " + name,
			"available": h.grcRegistry.List(),
		})
		return nil, false
	}
	return conn, true
}

// resolveVulnConnector looks up the vuln connector by the :connector path param.
func (h *IntegrationHandler) resolveVulnConnector(c *gin.Context) (vulnconnector.VulnConnector, bool) {
	name := c.Param("connector")
	conn, ok := h.vulnRegistry.Get(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":      "CONNECTOR_NOT_FOUND",
			"message":   "vuln connector not registered: " + name,
			"available": h.vulnRegistry.List(),
		})
		return nil, false
	}
	return conn, true
}
