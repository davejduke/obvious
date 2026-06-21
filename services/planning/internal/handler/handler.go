// Package handler provides HTTP handlers for the audit planning service.
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/planning/internal/domain"
	"github.com/davejduke/obvious/services/planning/internal/store"
)

// Handler bundles all planning HTTP handlers.
type Handler struct {
	stores *store.Stores
}

// New creates a new Handler.
func New(s *store.Stores) *Handler {
	return &Handler{stores: s}
}

// Routes mounts all planning routes.
// Intended to be mounted by the caller at /api/v1/planning.
func (h *Handler) Routes(r chi.Router) {
	// Strategic plans
	r.Route("/strategic-plans", func(r chi.Router) {
		r.Post("/", h.CreateStrategicPlan)
		r.Get("/", h.ListStrategicPlans)
		r.Get("/{id}", h.GetStrategicPlan)
		r.Patch("/{id}", h.UpdateStrategicPlan)
		r.Post("/{id}/approve", h.ApproveStrategicPlan)
		r.Post("/{id}/activate", h.ActivateStrategicPlan)
		r.Post("/{id}/archive", h.ArchiveStrategicPlan)
	})

	// Annual plans
	r.Route("/annual-plans", func(r chi.Router) {
		r.Post("/", h.CreateAnnualPlan)
		r.Get("/", h.ListAnnualPlans)
		r.Get("/{id}", h.GetAnnualPlan)
		r.Patch("/{id}", h.UpdateAnnualPlan)
		r.Post("/{id}/approve", h.ApproveAnnualPlan)
		r.Post("/{id}/activate", h.ActivateAnnualPlan)
		r.Post("/{id}/archive", h.ArchiveAnnualPlan)
	})

	// Assurance maps
	r.Route("/assurance-maps", func(r chi.Router) {
		r.Post("/", h.CreateAssuranceMap)
		r.Get("/", h.ListAssuranceMaps)
		r.Get("/{id}", h.GetAssuranceMap)
		r.Patch("/{id}", h.UpdateAssuranceMap)
	})

	// Resource calendars
	r.Route("/resource-calendars", func(r chi.Router) {
		r.Post("/", h.CreateResourceCalendar)
		r.Get("/", h.ListResourceCalendars)
		r.Get("/{id}", h.GetResourceCalendar)
		r.Patch("/{id}", h.UpdateResourceCalendar)
	})
}

// ---------------------------------------------------------------------------
// Strategic Plans
// ---------------------------------------------------------------------------

func (h *Handler) CreateStrategicPlan(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateStrategicPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	p, err := domain.NewStrategicPlan(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if err := h.stores.StrategicPlans.Create(p); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, p)
}

func (h *Handler) ListStrategicPlans(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	plans := h.stores.StrategicPlans.List(orgID)
	if plans == nil {
		plans = []*domain.StrategicPlan{}
	}
	jsonResponse(w, http.StatusOK, plans)
}

func (h *Handler) GetStrategicPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.stores.StrategicPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "strategic plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, p)
}

func (h *Handler) UpdateStrategicPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.stores.StrategicPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "strategic plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	var req domain.UpdateStrategicPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	p.Apply(req)
	if err := h.stores.StrategicPlans.Update(p); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, p)
}

func (h *Handler) ApproveStrategicPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.stores.StrategicPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "strategic plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	var body struct {
		ApprovedBy string     `json:"approved_by"`
		ApprovalID *uuid.UUID `json:"approval_id,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if body.ApprovedBy == "" {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "approved_by is required")
		return
	}
	if err := p.Approve(body.ApprovedBy, body.ApprovalID); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			jsonError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := h.stores.StrategicPlans.Update(p); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, p)
}

func (h *Handler) ActivateStrategicPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.stores.StrategicPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "strategic plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := p.Activate(); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			jsonError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := h.stores.StrategicPlans.Update(p); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, p)
}

func (h *Handler) ArchiveStrategicPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	p, err := h.stores.StrategicPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "strategic plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := p.Archive(); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			jsonError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := h.stores.StrategicPlans.Update(p); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Annual Plans
// ---------------------------------------------------------------------------

func (h *Handler) CreateAnnualPlan(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAnnualPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	ap, err := domain.NewAnnualPlan(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if err := h.stores.AnnualPlans.Create(ap); err != nil {
		if errors.Is(err, domain.ErrDuplicateYear) {
			jsonError(w, http.StatusConflict, "DUPLICATE_YEAR", err.Error())
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, ap)
}

func (h *Handler) ListAnnualPlans(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	year := 0
	if y := r.URL.Query().Get("year"); y != "" {
		var err error
		year, err = strconv.Atoi(y)
		if err != nil {
			jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "year must be an integer")
			return
		}
	}
	plans := h.stores.AnnualPlans.List(orgID, year)
	if plans == nil {
		plans = []*domain.AnnualPlan{}
	}
	jsonResponse(w, http.StatusOK, plans)
}

func (h *Handler) GetAnnualPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	ap, err := h.stores.AnnualPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "annual plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, ap)
}

func (h *Handler) UpdateAnnualPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	ap, err := h.stores.AnnualPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "annual plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	var req domain.UpdateAnnualPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	ap.Apply(req)
	if err := h.stores.AnnualPlans.Update(ap); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, ap)
}

func (h *Handler) ApproveAnnualPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	ap, err := h.stores.AnnualPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "annual plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	var body struct {
		ApprovalID *uuid.UUID `json:"approval_id,omitempty"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if err := ap.Approve(body.ApprovalID); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			jsonError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := h.stores.AnnualPlans.Update(ap); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, ap)
}

func (h *Handler) ActivateAnnualPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	ap, err := h.stores.AnnualPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "annual plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := ap.Activate(); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			jsonError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := h.stores.AnnualPlans.Update(ap); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, ap)
}

func (h *Handler) ArchiveAnnualPlan(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	ap, err := h.stores.AnnualPlans.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "annual plan not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := ap.Archive(); err != nil {
		if errors.Is(err, domain.ErrInvalidTransition) {
			jsonError(w, http.StatusConflict, "INVALID_TRANSITION", err.Error())
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	if err := h.stores.AnnualPlans.Update(ap); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, ap)
}

// ---------------------------------------------------------------------------
// Assurance Maps
// ---------------------------------------------------------------------------

func (h *Handler) CreateAssuranceMap(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateAssuranceMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	am, err := domain.NewAssuranceMap(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if err := h.stores.AssuranceMaps.Create(am); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, am)
}

func (h *Handler) ListAssuranceMaps(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	maps := h.stores.AssuranceMaps.List(orgID)
	if maps == nil {
		maps = []*domain.AssuranceMap{}
	}
	jsonResponse(w, http.StatusOK, maps)
}

func (h *Handler) GetAssuranceMap(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	am, err := h.stores.AssuranceMaps.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "assurance map not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, am)
}

func (h *Handler) UpdateAssuranceMap(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	am, err := h.stores.AssuranceMaps.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "assurance map not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	var req domain.UpdateAssuranceMapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	am.Apply(req)
	if err := h.stores.AssuranceMaps.Update(am); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, am)
}

// ---------------------------------------------------------------------------
// Resource Calendars
// ---------------------------------------------------------------------------

func (h *Handler) CreateResourceCalendar(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateResourceCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	rc, err := domain.NewResourceCalendar(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	if err := h.stores.ResourceCalendars.Create(rc); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusCreated, rc)
}

func (h *Handler) ListResourceCalendars(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	cals := h.stores.ResourceCalendars.List(orgID)
	if cals == nil {
		cals = []*domain.ResourceCalendar{}
	}
	jsonResponse(w, http.StatusOK, cals)
}

func (h *Handler) GetResourceCalendar(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	rc, err := h.stores.ResourceCalendars.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "resource calendar not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, rc)
}

func (h *Handler) UpdateResourceCalendar(w http.ResponseWriter, r *http.Request) {
	orgID, ok := requireOrgID(w, r)
	if !ok {
		return
	}
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	rc, err := h.stores.ResourceCalendars.Get(orgID, id)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			jsonError(w, http.StatusNotFound, "NOT_FOUND", "resource calendar not found")
			return
		}
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	var req domain.UpdateResourceCalendarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	rc.Apply(req)
	if err := h.stores.ResourceCalendars.Update(rc); err != nil {
		jsonError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	jsonResponse(w, http.StatusOK, rc)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func requireOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	s := r.Header.Get("X-Org-ID")
	if s == "" {
		s = r.URL.Query().Get("org_id")
	}
	if s == "" {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "X-Org-ID header or org_id query param required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(s)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid org_id")
		return uuid.Nil, false
	}
	return id, true
}

func parseID(w http.ResponseWriter, r *http.Request, param string) (uuid.UUID, bool) {
	s := chi.URLParam(r, param)
	id, err := uuid.Parse(s)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid "+param)
		return uuid.Nil, false
	}
	return id, true
}

func jsonResponse(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, status int, code, message string) {
	jsonResponse(w, status, map[string]string{"code": code, "message": message})
}
