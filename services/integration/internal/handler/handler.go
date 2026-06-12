// Package handler provides the HTTP handlers for the integration gateway.
package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/davejduke/obvious/services/integration/internal/connector"
)

// IntegrationHandler serves the integration gateway endpoints.
type IntegrationHandler struct {
	registry *connector.Registry
}

// NewIntegrationHandler creates a new handler.
func NewIntegrationHandler(reg *connector.Registry) *IntegrationHandler {
	return &IntegrationHandler{registry: reg}
}

// RegisterRoutes mounts all integration routes.
func (h *IntegrationHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		integrations := v1.Group("/integrations")
		{
			integrations.GET("", h.ListConnectors)
			integrations.GET("/:connector/health", h.ConnectorHealth)
			integrations.GET("/:connector/logs", h.FetchLogs)
		}
	}
}

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

// resolveConnector looks up the connector by the :connector path param.
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

