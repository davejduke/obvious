// Package handlers contains HTTP handlers for the identity service.
package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/identity/internal/middleware"
	"github.com/davejduke/obvious/services/identity/internal/models"
	"github.com/davejduke/obvious/services/identity/internal/repository"
	"github.com/davejduke/obvious/services/identity/internal/service"
	"github.com/davejduke/obvious/services/identity/pkg/crypto"
	jwtpkg "github.com/davejduke/obvious/services/identity/pkg/jwt"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	repo    *repository.Repository
	jwt     *jwtpkg.Manager
	session *service.SessionStore
	refreshTTL time.Duration
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(repo *repository.Repository, jwt *jwtpkg.Manager, session *service.SessionStore, refreshTTL time.Duration) *AuthHandler {
	return &AuthHandler{repo: repo, jwt: jwt, session: session, refreshTTL: refreshTTL}
}

// RegisterRequest is the body for POST /auth/register.
type RegisterRequest struct {
	OrgID       string `json:"org_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
	Persona     string `json:"persona"`
}

// LoginRequest is the body for POST /auth/login.
type LoginRequest struct {
	OrgID    string `json:"org_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenResponse is the standard auth token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// Register creates a new user in the org.
// POST /auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.OrgID == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "org_id, email, display_name, and password are required")
		return
	}
	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid org_id")
		return
	}
	persona := models.Persona(req.Persona)
	if req.Persona != "" && !models.ValidPersona(persona) {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid persona")
		return
	}
	hash, err := crypto.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to hash password")
		return
	}
	user, err := h.repo.CreateUser(r.Context(), orgID, req.Email, req.DisplayName, hash)
	if err != nil {
		writeError(w, http.StatusConflict, "CONFLICT", "user already exists or org not found")
		return
	}
	// Ensure system roles exist then assign persona
	if err := h.repo.EnsureSystemRoles(r.Context(), orgID); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to setup roles")
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
		"created_at":   user.CreatedAt,
	})
}

// Login authenticates a user and issues tokens.
// POST /auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" || req.OrgID == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "org_id, email, and password are required")
		return
	}
	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid org_id")
		return
	}
	user, err := h.repo.GetUserByEmail(r.Context(), orgID, req.Email)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}
	if !user.IsActive {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "account deactivated")
		return
	}
	if user.PasswordHash == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}
	if err := crypto.VerifyPassword(req.Password, *user.PasswordHash); err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid credentials")
		return
	}

	persona, _ := h.repo.GetFirstPersonaForUser(r.Context(), orgID, user.ID)

	accessToken, err := h.jwt.IssueAccessToken(user.ID.String(), orgID.String(), user.Email, user.DisplayName, persona)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue access token")
		return
	}
	refreshToken, jti, err := h.jwt.IssueRefreshToken(user.ID.String(), orgID.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue refresh token")
		return
	}
	rt := &models.RefreshToken{
		TokenID:   jti,
		UserID:    user.ID.String(),
		OrgID:     orgID.String(),
		ExpiresAt: time.Now().Add(h.refreshTTL),
	}
	if err := h.session.StoreRefreshToken(r.Context(), rt); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to store session")
		return
	}
	_ = h.repo.UpdateLastLogin(r.Context(), user.ID)

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    900, // 15 minutes default
	})
}

// RefreshRequest is the body for POST /auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// Refresh rotates a refresh token and issues a new access token.
// POST /auth/refresh
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "refresh_token is required")
		return
	}
	claims, err := h.jwt.VerifyRefreshToken(req.RefreshToken)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired refresh token")
		return
	}
	stored, err := h.session.GetRefreshToken(r.Context(), claims.ID)
	if err != nil || stored == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "refresh token revoked or not found")
		return
	}
	// Rotate: revoke old, issue new
	_ = h.session.RevokeRefreshToken(r.Context(), claims.ID)

	orgID, err := uuid.Parse(claims.OrgID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
		return
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token claims")
		return
	}
	user, err := h.repo.GetUserByID(r.Context(), orgID, userID)
	if err != nil || !user.IsActive {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "user not found or deactivated")
		return
	}
	persona, _ := h.repo.GetFirstPersonaForUser(r.Context(), orgID, userID)

	accessToken, err := h.jwt.IssueAccessToken(user.ID.String(), orgID.String(), user.Email, user.DisplayName, persona)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue access token")
		return
	}
	newRefresh, newJTI, err := h.jwt.IssueRefreshToken(user.ID.String(), orgID.String())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to issue refresh token")
		return
	}
	rt := &models.RefreshToken{
		TokenID:   newJTI,
		UserID:    user.ID.String(),
		OrgID:     orgID.String(),
		ExpiresAt: time.Now().Add(h.refreshTTL),
	}
	if err := h.session.StoreRefreshToken(r.Context(), rt); err != nil {
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to store session")
		return
	}

	writeJSON(w, http.StatusOK, TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefresh,
		TokenType:    "Bearer",
		ExpiresIn:    900,
	})
}

// Logout revokes the refresh token for the current session.
// POST /auth/logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	claims := middleware.ClaimsFromContext(r.Context())
	if claims == nil {
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
		return
	}
	var req RefreshRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.RefreshToken != "" {
		if rc, err := h.jwt.VerifyRefreshToken(req.RefreshToken); err == nil {
			_ = h.session.RevokeRefreshToken(r.Context(), rc.ID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}
