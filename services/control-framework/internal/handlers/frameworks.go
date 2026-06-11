// Package handlers provides HTTP handlers for framework operations.
package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/davejduke/obvious/services/control-framework/internal/domain"
	"github.com/davejduke/obvious/services/control-framework/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// FrameworkHandler handles /frameworks routes.
type FrameworkHandler struct {
	svc *service.Service
}

// NewFrameworkHandler creates a FrameworkHandler.
func NewFrameworkHandler(svc *service.Service) *FrameworkHandler {
	return &FrameworkHandler{svc: svc}
}

// ListFrameworks handles GET /frameworks
func (h *FrameworkHandler) ListFrameworks(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing org context")
		return
	}

	frameworks, err := h.svc.ListFrameworks(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if frameworks == nil {
		frameworks = []domain.Framework{}
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: frameworks})
}

// GetFramework handles GET /frameworks/:id
func (h *FrameworkHandler) GetFramework(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid framework ID")
		return
	}

	f, err := h.svc.GetFramework(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if f == nil {
		writeError(w, http.StatusNotFound, "framework not found")
		return
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: f})
}

// CreateFramework handles POST /frameworks
func (h *FrameworkHandler) CreateFramework(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing org context")
		return
	}

	var req domain.CreateFrameworkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	f, err := h.svc.CreateFramework(r.Context(), orgID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, domain.APIResponse{Data: f})
}

// GetNIS2Domains handles GET /frameworks/nis2/domains
func (h *FrameworkHandler) GetNIS2Domains(w http.ResponseWriter, r *http.Request) {
	domains := h.svc.GetNIS2Domains()
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: domains})
}

