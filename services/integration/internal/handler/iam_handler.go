package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/davejduke/obvious/services/integration/internal/connector"
	"github.com/davejduke/obvious/services/integration/internal/iam"
)

// IAMHandler serves the IAM integration gateway endpoints.
type IAMHandler struct {
	registry  *iam.IAMRegistry
	scheduler *iam.Scheduler
}

// NewIAMHandler creates a new IAM handler.
func NewIAMHandler(reg *iam.IAMRegistry, sched *iam.Scheduler) *IAMHandler {
	return &IAMHandler{registry: reg, scheduler: sched}
}

// RegisterIAMRoutes mounts all IAM routes under /api/v1/iam.
func (h *IAMHandler) RegisterIAMRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		iamGroup := v1.Group("/iam")
		{
			iamGroup.GET("", h.ListIAMConnectors)
			iamGroup.GET("/:connector/health", h.IAMConnectorHealth)
			iamGroup.GET("/:connector/evidence", h.GetEvidence)
			iamGroup.POST("/:connector/sync", h.TriggerSync)
		}
	}
}

// ListIAMConnectors handles GET /api/v1/iam
func (h *IAMHandler) ListIAMConnectors(c *gin.Context) {
	names := h.registry.List()
	results := h.scheduler.AllResults()

	type status struct {
		Name        string     `json:"name"`
		LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
		EvidenceCount int      `json:"evidence_count"`
		SyncError   string     `json:"sync_error,omitempty"`
	}

	out := make([]status, 0, len(names))
	for _, n := range names {
		s := status{Name: n}
		if r, ok := results[n]; ok {
			s.LastSyncAt = &r.SyncedAt
			s.EvidenceCount = len(r.Evidence)
			if r.Error != nil {
				s.SyncError = r.Error.Error()
			}
		}
		out = append(out, s)
	}

	c.JSON(http.StatusOK, gin.H{"connectors": out, "total": len(out)})
}

// IAMConnectorHealth handles GET /api/v1/iam/:connector/health
func (h *IAMHandler) IAMConnectorHealth(c *gin.Context) {
	conn, ok := h.resolveConnector(c)
	if !ok {
		return
	}
	status := conn.Health(c.Request.Context())
	httpCode := http.StatusOK
	if !status.Healthy {
		httpCode = http.StatusServiceUnavailable
	}
	c.JSON(httpCode, status)
}

// GetEvidence handles GET /api/v1/iam/:connector/evidence
// Returns evidence items from the most recent sync result.
func (h *IAMHandler) GetEvidence(c *gin.Context) {
	name := c.Param("connector")
	if _, ok := h.registry.Get(name); !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":      "IAM_CONNECTOR_NOT_FOUND",
			"message":   "IAM connector not registered: " + name,
			"available": h.registry.List(),
		})
		return
	}

	result, ok := h.scheduler.LastResult(name)
	if !ok {
		c.JSON(http.StatusAccepted, gin.H{
			"code":    "SYNC_PENDING",
			"message": "no sync result yet; first sync is in progress",
		})
		return
	}
	if result.Error != nil {
		switch result.Error {
		case connector.ErrCircuitOpen:
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"code":    "CIRCUIT_OPEN",
				"message": "IAM connector circuit breaker is open; upstream is unavailable",
			})
		default:
			c.JSON(http.StatusBadGateway, gin.H{
				"code":    "SYNC_ERROR",
				"message": result.Error.Error(),
			})
		}
		return
	}

	evidence := result.Evidence
	if evidence == nil {
		evidence = []iam.EvidenceItem{}
	}

	// Optional filter by evidence_type query param.
	if filter := c.Query("type"); filter != "" {
		filtered := evidence[:0]
		for _, e := range evidence {
			if string(e.EvidenceType) == filter {
				filtered = append(filtered, e)
			}
		}
		evidence = filtered
	}

	c.JSON(http.StatusOK, gin.H{
		"connector":    name,
		"synced_at":    result.SyncedAt,
		"evidence":     evidence,
		"total":        len(evidence),
	})
}

// TriggerSync handles POST /api/v1/iam/:connector/sync
// Forces an immediate sync outside the normal scheduler interval.
func (h *IAMHandler) TriggerSync(c *gin.Context) {
	conn, ok := h.resolveConnector(c)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	snap, err := conn.Sync(ctx)
	if err != nil {
		switch err {
		case connector.ErrCircuitOpen:
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"code":    "CIRCUIT_OPEN",
				"message": "IAM connector circuit breaker is open; upstream is unavailable",
			})
		default:
			c.JSON(http.StatusBadGateway, gin.H{
				"code":    "SYNC_ERROR",
				"message": err.Error(),
			})
		}
		return
	}

	evidence := iam.MapSnapshotToEvidence(snap)
	c.JSON(http.StatusOK, gin.H{
		"connector":     conn.Name(),
		"synced_at":     snap.CollectedAt,
		"users":         len(snap.Users),
		"evidence":      len(evidence),
		"snapshot":      snap,
	})
}

// resolveConnector looks up an IAM connector by the :connector path param.
func (h *IAMHandler) resolveConnector(c *gin.Context) (iam.IAMConnector, bool) {
	name := c.Param("connector")
	conn, ok := h.registry.Get(name)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"code":      "IAM_CONNECTOR_NOT_FOUND",
			"message":   "IAM connector not registered: " + name,
			"available": h.registry.List(),
		})
		return nil, false
	}
	return conn, true
}
