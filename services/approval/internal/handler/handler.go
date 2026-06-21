// Package handler provides the HTTP handlers for the approval workflow service.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/approval/internal/domain"
	"github.com/davejduke/obvious/services/approval/internal/store"
)

// Handler bundles the approval workflow HTTP handlers.
type Handler struct {
	store *store.Store
}

// New creates a new Handler.
func New(s *store.Store) *Handler {
	return &Handler{store: s}
}

// Routes mounts all approval workflow routes on the provided chi Router.
// All routes live under /api/v1/approvals (mounted by the caller).
func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Post("/{id}/submit", h.Submit)
	r.Post("/{id}/approve", h.Approve)
	r.Post("/{id}/reject", h.Reject)
	r.Post("/{id}/lock", h.Lock)
	r.Post("/{id}/return-to-draft", h.ReturnToDraft)
	r.Get("/{id}/history", h.History)
}

// ------------------------------------------------------------
// CRUD
// ------------------------------------------------------------

// Create handles POST /api/v1/approvals
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateWorkflowRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if req.OrgID == uuid.Nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "org_id is required")
		return
	}
	if req.WorkflowType == "" {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "workflow_type is required")
		return
	}
	if req.ResourceType == "" || req.ResourceID == uuid.Nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "resource_type and resource_id are required")
		return
	}

	wf := domain.NewWorkflow(req)
	if err := h.store.Create(wf); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, wf)
}

// List handles GET /api/v1/approvals?org_id=&resource_type=&resource_id=
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	resourceType := r.URL.Query().Get("resource_type")
	var resID *uuid.UUID
	if s := r.URL.Query().Get("resource_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid resource_id")
			return
		}
		resID = &id
	}
	workflows := h.store.List(orgID, resourceType, resID)
	if workflows == nil {
		workflows = []*domain.Workflow{}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"workflows": workflows, "total": len(workflows)})
}

// Get handles GET /api/v1/approvals/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	orgID, id, ok := requireOrgAndID(w, r)
	if !ok {
		return
	}
	wf, err := h.store.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrWorkflowNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, wf)
}

// ------------------------------------------------------------
// State-machine transitions
// ------------------------------------------------------------

// Submit handles POST /api/v1/approvals/{id}/submit
// Body: SubmitRequest.  Transitions draft → pending_approval.
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	wf, ok := h.loadWorkflow(w, r)
	if !ok {
		return
	}
	var req domain.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	entry, err := wf.Submit(req)
	if err != nil {
		h.handleTransitionError(w, err)
		return
	}
	h.saveAndRespond(w, r, wf, entry)
}

// Approve handles POST /api/v1/approvals/{id}/approve
// Transitions pending_approval → approved.
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	wf, ok := h.loadWorkflow(w, r)
	if !ok {
		return
	}
	var req domain.ApproveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	entry, err := wf.Approve(req)
	if err != nil {
		h.handleTransitionError(w, err)
		return
	}
	h.saveAndRespond(w, r, wf, entry)
}

// Reject handles POST /api/v1/approvals/{id}/reject
// Transitions pending_approval → rejected.
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	wf, ok := h.loadWorkflow(w, r)
	if !ok {
		return
	}
	var req domain.RejectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	entry, err := wf.Reject(req)
	if err != nil {
		h.handleTransitionError(w, err)
		return
	}
	h.saveAndRespond(w, r, wf, entry)
}

// Lock handles POST /api/v1/approvals/{id}/lock
// Transitions approved → locked (report_release requires password_confirm).
func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	wf, ok := h.loadWorkflow(w, r)
	if !ok {
		return
	}
	var req domain.LockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	entry, err := wf.Lock(req)
	if err != nil {
		h.handleTransitionError(w, err)
		return
	}
	h.saveAndRespond(w, r, wf, entry)
}

// ReturnToDraft handles POST /api/v1/approvals/{id}/return-to-draft
// Transitions rejected → draft so the submitter can revise.
func (h *Handler) ReturnToDraft(w http.ResponseWriter, r *http.Request) {
	wf, ok := h.loadWorkflow(w, r)
	if !ok {
		return
	}
	var req domain.SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	entry, err := wf.ReturnToDraft(req)
	if err != nil {
		h.handleTransitionError(w, err)
		return
	}
	h.saveAndRespond(w, r, wf, entry)
}

// History handles GET /api/v1/approvals/{id}/history
func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	wf, ok := h.loadWorkflow(w, r)
	if !ok {
		return
	}
	hist := wf.History
	if hist == nil {
		hist = []domain.HistoryEntry{}
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"history": hist, "total": len(hist)})
}

// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func (h *Handler) loadWorkflow(w http.ResponseWriter, r *http.Request) (*domain.Workflow, bool) {
	orgID, id, ok := requireOrgAndID(w, r)
	if !ok {
		return nil, false
	}
	wf, err := h.store.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrWorkflowNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "workflow not found")
		} else {
			jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
		return nil, false
	}
	return wf, true
}

func (h *Handler) saveAndRespond(w http.ResponseWriter, r *http.Request, wf *domain.Workflow, entry *domain.HistoryEntry) {
	if err := h.store.Update(wf); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, map[string]interface{}{"workflow": wf, "history_entry": entry})
}

func (h *Handler) handleTransitionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidTransition):
		jsonError(w, http.StatusUnprocessableEntity, "INVALID_TRANSITION", err.Error())
	case errors.Is(err, domain.ErrUnauthorizedRole):
		jsonError(w, http.StatusForbidden, "FORBIDDEN", err.Error())
	case errors.Is(err, domain.ErrWorkflowLocked):
		jsonError(w, http.StatusConflict, "WORKFLOW_LOCKED", err.Error())
	case errors.Is(err, domain.ErrPasswordRequired):
		jsonError(w, http.StatusUnprocessableEntity, "PASSWORD_REQUIRED", err.Error())
	default:
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	}
}

func requireOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	s := r.URL.Query().Get("org_id")
	if s == "" {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "org_id query param required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid org_id")
		return uuid.Nil, false
	}
	return id, true
}

func requireOrgAndID(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	rawID := chi.URLParam(r, "id")
	id, err := uuid.Parse(rawID)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid workflow id")
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, id, true
}

func jsonResponse(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func jsonError(w http.ResponseWriter, status int, code, message string) {
	jsonResponse(w, status, map[string]string{"code": code, "message": message})
}
