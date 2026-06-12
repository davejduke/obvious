package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/identity/internal/middleware"
	"github.com/davejduke/obvious/services/identity/internal/repository"
)

// RoleHandler handles /roles and /users/:id/roles endpoints.
type RoleHandler struct {
	repo *repository.Repository
}

// NewRoleHandler creates a RoleHandler.
func NewRoleHandler(repo *repository.Repository) *RoleHandler {
	return &RoleHandler{repo: repo}
}

// List returns all roles in the org.
// GET /roles
func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	roles, err := h.repo.GetSystemRoles(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list roles")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"roles": roles})
}

// AssignRoleRequest is the body for POST /users/:id/roles.
type AssignRoleRequest struct {
	RoleSlug string `json:"role_slug"`
}

// AssignToUser assigns a role (by slug) to a user.
// POST /users/:id/roles
func (h *RoleHandler) AssignToUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user id")
		return
	}
	var req AssignRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.RoleSlug == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "role_slug is required")
		return
	}
	// Verify user exists in org
	if _, err := h.repo.GetUserByID(r.Context(), orgID, userID); err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
		return
	}
	role, err := h.repo.GetRoleBySlug(r.Context(), orgID, req.RoleSlug)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "role not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get role")
		return
	}
	callerID, _ := uuid.Parse(claims.Subject)
	if err := h.repo.AssignRole(r.Context(), userID, role.ID, &callerID, nil); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to assign role")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"user_id": userID,
		"role":    role,
	})
}
