// controls.go provides HTTP handlers for control CRUD, mappings, evidence, and assessments.
package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/davejduke/obvious/services/control-framework/internal/domain"
	"github.com/davejduke/obvious/services/control-framework/internal/service"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ControlHandler handles /controls routes.
type ControlHandler struct {
	svc *service.Service
}

// NewControlHandler creates a ControlHandler.
func NewControlHandler(svc *service.Service) *ControlHandler {
	return &ControlHandler{svc: svc}
}

// ListControls handles GET /controls
func (h *ControlHandler) ListControls(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing org context")
		return
	}

	filter := domain.ListControlsFilter{
		Limit:  50,
		Offset: 0,
	}

	q := r.URL.Query()
	if v := q.Get("framework_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid framework_id")
			return
		}
		filter.FrameworkID = &id
	}
	if v := q.Get("domain"); v != "" {
		filter.Domain = &v
	}
	if v := q.Get("article_ref"); v != "" {
		filter.ArticleRef = &v
	}
	if v := q.Get("search"); v != "" {
		filter.Search = &v
	}
	if v := q.Get("limit"); v != "" {
		n, _ := strconv.Atoi(v)
		if n > 0 {
			filter.Limit = n
		}
	}
	if v := q.Get("offset"); v != "" {
		n, _ := strconv.Atoi(v)
		if n >= 0 {
			filter.Offset = n
		}
	}
	if v := q.Get("is_active"); v != "" {
		b := v == "true"
		filter.IsActive = &b
	}

	result, err := h.svc.ListControls(r.Context(), orgID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: result})
}

// GetControl handles GET /controls/:id
func (h *ControlHandler) GetControl(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid control ID")
		return
	}

	c, err := h.svc.GetControl(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if c == nil {
		writeError(w, http.StatusNotFound, "control not found")
		return
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: c})
}

// CreateControl handles POST /controls
func (h *ControlHandler) CreateControl(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing org context")
		return
	}

	var req domain.CreateControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.svc.CreateControl(r.Context(), orgID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, domain.APIResponse{Data: c})
}

// UpdateControl handles PUT /controls/:id
func (h *ControlHandler) UpdateControl(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid control ID")
		return
	}

	var req domain.UpdateControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	c, err := h.svc.UpdateControl(r.Context(), id, req)
	if err != nil {
		if err.Error() == "control not found" {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: c})
}

// GetControlMappings handles GET /controls/:id/mappings
func (h *ControlHandler) GetControlMappings(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid control ID")
		return
	}

	mappings, err := h.svc.GetControlMappings(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: mappings})
}

// GetEvidenceRequirements handles GET /controls/:id/evidence-requirements
func (h *ControlHandler) GetEvidenceRequirements(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid control ID")
		return
	}

	reqs, err := h.svc.GetEvidenceRequirements(r.Context(), id)
	if err != nil {
		if err.Error() == "control not found" {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: reqs})
}

// AssessControl handles POST /controls/:id/assess
func (h *ControlHandler) AssessControl(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgIDFromContext(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing org context")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid control ID")
		return
	}

	var req domain.AssessControlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	a, err := h.svc.AssessControl(r.Context(), id, orgID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, domain.APIResponse{Data: a})
}

// --- Shared helpers ---

type contextKey string

const orgIDKey contextKey = "org_id"

func orgIDFromContext(r *http.Request) (uuid.UUID, bool) {
	v := r.Context().Value(orgIDKey)
	if v == nil {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, domain.APIResponse{Error: msg})
}

