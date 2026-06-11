package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/identity/internal/middleware"
	"github.com/davejduke/obvious/services/identity/internal/repository"
)

// UserHandler handles /users endpoints.
type UserHandler struct {
	repo *repository.Repository
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(repo *repository.Repository) *UserHandler {
	return &UserHandler{repo: repo}
}

// List returns all users in the caller's org.
// GET /users
func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid org context")
		return
	}
	users, err := h.repo.ListUsers(r.Context(), orgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": users})
}

// Get returns a single user by ID (org-isolated).
// GET /users/:id
func (h *UserHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user id")
		return
	}
	user, err := h.repo.GetUserByID(r.Context(), orgID, userID)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// UpdateRequest is the PUT /users/:id body.
type UpdateRequest struct {
	DisplayName string `json:"display_name"`
}

// Update modifies a user's display name.
// PUT /users/:id
func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user id")
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "display_name is required")
		return
	}
	user, err := h.repo.UpdateUser(r.Context(), orgID, userID, req.DisplayName)
	if err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update user")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

// Delete deactivates a user.
// DELETE /users/:id
func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	orgID, _ := uuid.Parse(claims.OrgID)
	userID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user id")
		return
	}
	if err := h.repo.DeactivateUser(r.Context(), orgID, userID); err != nil {
		if err == repository.ErrNotFound {
			writeError(w, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to deactivate user")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
