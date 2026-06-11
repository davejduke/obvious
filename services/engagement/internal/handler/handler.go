// Package handler provides the HTTP handlers for the engagement service.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/engagement/internal/domain"
	"github.com/davejduke/obvious/services/engagement/internal/store"
)

// EngagementHandler handles engagement CRUD and lifecycle transitions.
type EngagementHandler struct {
	store *store.EngagementStore
}

// NewEngagementHandler creates a new handler with the given store.
func NewEngagementHandler(s *store.EngagementStore) *EngagementHandler {
	return &EngagementHandler{store: s}
}

// RegisterRoutes mounts all engagement routes on r under /api/v1.
func (h *EngagementHandler) RegisterRoutes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		engagements := v1.Group("/engagements")
		{
			engagements.POST("", h.Create)
			engagements.GET("", h.List)
			engagements.GET("/:id", h.Get)
			engagements.PATCH("/:id", h.Update)
			engagements.POST("/:id/transition", h.Transition)
		}
	}
}

// Create handles POST /api/v1/engagements
func (h *EngagementHandler) Create(c *gin.Context) {
	var req domain.CreateEngagementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "name is required"})
		return
	}
	if req.OrgID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "org_id is required"})
		return
	}
	if req.FrameworkID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "framework_id is required"})
		return
	}

	eng := domain.NewEngagement(req)
	if err := h.store.Create(eng); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, eng)
}

// List handles GET /api/v1/engagements
func (h *EngagementHandler) List(c *gin.Context) {
	orgIDStr := c.Query("org_id")
	if orgIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "org_id query param required"})
		return
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid org_id"})
		return
	}
	engagements := h.store.List(orgID)
	if engagements == nil {
		engagements = []*domain.Engagement{}
	}
	c.JSON(http.StatusOK, gin.H{"engagements": engagements, "total": len(engagements)})
}

// Get handles GET /api/v1/engagements/:id
func (h *EngagementHandler) Get(c *gin.Context) {
	orgID, id, ok := h.parseOrgAndID(c)
	if !ok {
		return
	}
	eng, err := h.store.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrEngagementNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "engagement not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, eng)
}

// Update handles PATCH /api/v1/engagements/:id
func (h *EngagementHandler) Update(c *gin.Context) {
	orgID, id, ok := h.parseOrgAndID(c)
	if !ok {
		return
	}
	eng, err := h.store.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrEngagementNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "engagement not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	var req domain.UpdateEngagementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	if req.Name != nil {
		eng.Name = *req.Name
	}
	if req.Description != nil {
		eng.Description = *req.Description
	}
	if req.Scope != nil {
		eng.Scope = req.Scope
	}
	if req.LeadAuditorID != nil {
		eng.LeadAuditorID = req.LeadAuditorID
	}
	if req.TargetStartDate != nil {
		eng.TargetStartDate = req.TargetStartDate
	}
	if req.TargetEndDate != nil {
		eng.TargetEndDate = req.TargetEndDate
	}

	if err := h.store.Update(eng); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, eng)
}

// Transition handles POST /api/v1/engagements/:id/transition
func (h *EngagementHandler) Transition(c *gin.Context) {
	orgID, id, ok := h.parseOrgAndID(c)
	if !ok {
		return
	}
	eng, err := h.store.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrEngagementNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "engagement not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}

	var req domain.TransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": err.Error()})
		return
	}

	event, err := eng.Transition(req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidTransition):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"code":           "INVALID_TRANSITION",
				"message":        err.Error(),
				"current_phase":  eng.Phase,
				"allowed_phases": eng.AllowedNextPhases(),
			})
		case errors.Is(err, domain.ErrEngagementTerminal):
			c.JSON(http.StatusUnprocessableEntity, gin.H{
				"code":    "TERMINAL_STATE",
				"message": err.Error(),
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		}
		return
	}

	if err := h.store.Update(eng); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "INTERNAL_ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"engagement": eng, "event": event})
}

// parseOrgAndID parses org_id query param and :id path param.
func (h *EngagementHandler) parseOrgAndID(c *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	orgIDStr := c.Query("org_id")
	if orgIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "org_id query param required"})
		return uuid.Nil, uuid.Nil, false
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid org_id"})
		return uuid.Nil, uuid.Nil, false
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "INVALID_REQUEST", "message": "invalid id"})
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, id, true
}

