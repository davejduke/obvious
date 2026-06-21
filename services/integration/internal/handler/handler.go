// Package handler provides the HTTP handlers for the integration gateway.
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/davejduke/obvious/services/integration/internal/connector"
	"github.com/davejduke/obvious/services/integration/internal/grc"
)

// IntegrationHandler serves the integration gateway endpoints.
type IntegrationHandler struct {
	registry    *connector.Registry
	grcRegistry *grc.Registry
}

// NewIntegrationHandler creates a new handler.
func NewIntegrationHandler(reg *connector.Registry) *IntegrationHandler {
	return &IntegrationHandler{registry: reg, grcRegistry: grc.NewRegistry()}
}

// NewIntegrationHandlerWithGRC creates a handler with both SIEM and GRC registries.
func NewIntegrationHandlerWithGRC(reg *connector.Registry, grcReg *grc.Registry) *IntegrationHandler {
	return &IntegrationHandler{registry: reg, grcRegistry: grcReg}
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
// Exports audit findings for the given engagement to ServiceNow GRC tables.
func (h *IntegrationHandler) ExportToServiceNow(c *gin.Context) {
	engagementID := c.Param("id")
	if engagementID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "engagement id is required"})
		return
	}

	conn, ok := h.resolveGRCConnector(c)
	if !ok {
		// Try to resolve "servicenow" directly from the registry
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
