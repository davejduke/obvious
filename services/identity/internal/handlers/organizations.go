package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/identity/internal/middleware"
	"github.com/davejduke/obvious/services/identity/internal/repository"
	"github.com/davejduke/obvious/services/identity/pkg/crypto"
)

// OrgHandler handles /organizations endpoints.
type OrgHandler struct {
	repo *repository.Repository
}

// NewOrgHandler creates an OrgHandler.
func NewOrgHandler(repo *repository.Repository) *OrgHandler {
	return &OrgHandler{repo: repo}
}

// CreateOrgRequest is the body for POST /organizations.
type CreateOrgRequest struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Tier string `json:"tier"`
}

// Create creates a new organization.
// POST /organizations
func (h *OrgHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" || req.Slug == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "name and slug are required")
		return
	}
	tier := req.Tier
	if tier == "" {
		tier = "standard"
	}
	org, err := h.repo.CreateOrganization(r.Context(), req.Name, req.Slug, tier)
	if err != nil {
		writeError(w, http.StatusConflict, "CONFLICT", "organization slug already exists")
		return
	}
	// Seed system roles for the new org
	_ = h.repo.EnsureSystemRoles(r.Context(), org.ID)
	writeJSON(w, http.StatusCreated, org)
}

// List returns all active organizations (admin endpoint).
// GET /organizations
func (h *OrgHandler) List(w http.ResponseWriter, r *http.Request) {
	orgs, err := h.repo.ListOrganizations(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list organizations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"organizations": orgs})
}

// Get returns a single organization by ID (org-isolated).
// GET /organizations/:id
func (h *OrgHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	paramID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid organization id")
		return
	}
	// Org isolation: callers can only see their own org
	if paramID != orgID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied to this organization")
		return
	}
	org, err := h.repo.GetOrganization(r.Context(), orgID)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "organization not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get organization")
		return
	}
	writeJSON(w, http.StatusOK, org)
}

// UpdateOrgRequest is the body for PUT /organizations/:id.
type UpdateOrgRequest struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

// Update modifies an organization.
// PUT /organizations/:id
func (h *OrgHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	paramID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid organization id")
		return
	}
	if paramID != orgID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied to this organization")
		return
	}
	var req UpdateOrgRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "name is required")
		return
	}
	if req.Tier == "" {
		req.Tier = "standard"
	}
	org, err := h.repo.UpdateOrganization(r.Context(), orgID, req.Name, req.Tier)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update organization")
		return
	}
	writeJSON(w, http.StatusOK, org)
}

// AddMemberRequest is the body for POST /organizations/:id/members.
type AddMemberRequest struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Persona     string `json:"persona"`
}

// AddMember invites a user into the org.
// POST /organizations/:id/members
func (h *OrgHandler) AddMember(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	paramID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid organization id")
		return
	}
	if paramID != orgID {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "access denied to this organization")
		return
	}
	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "email, display_name, and password are required")
		return
	}
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}
	user, err := h.repo.CreateUser(r.Context(), orgID, req.Email, req.DisplayName, hash)
	if err != nil {
		writeError(w, http.StatusConflict, "CONFLICT", "user already exists")
		return
	}
	personaSlug := "beta_tester"
	if req.Persona != "" {
		personaSlug = req.Persona
	}
	role, err := h.repo.GetRoleBySlug(r.Context(), orgID, personaSlug)
	if err == nil {
		_ = h.repo.AssignRole(r.Context(), user.ID, role.ID, nil, nil)
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":           user.ID,
		"email":        user.Email,
		"display_name": user.DisplayName,
		"org_id":       user.OrgID,
	})
}
