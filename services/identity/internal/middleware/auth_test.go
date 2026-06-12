package middleware_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/identity/internal/middleware"
	"github.com/davejduke/obvious/services/identity/internal/models"
	jwtpkg "github.com/davejduke/obvious/services/identity/pkg/jwt"
	"github.com/davejduke/obvious/services/identity/pkg/rbac"
	"github.com/google/uuid"
)

func newManager(t *testing.T) *jwtpkg.Manager {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return jwtpkg.NewManagerFromKeys(priv, &priv.PublicKey, "test", 15*time.Minute, 7*24*time.Hour)
}

func TestAuthenticate_ValidToken(t *testing.T) {
	m := newManager(t)
	authn := middleware.NewAuthenticator(m)
	userID := uuid.New().String()
	orgID := uuid.New().String()
	tok, _ := m.IssueAccessToken(userID, orgID, "user@example.com", "User", "internal_auditor")

	var gotClaims *jwtpkg.AccessClaims
	handler := authn.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotClaims = middleware.ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
	if gotClaims == nil {
		t.Fatal("expected claims in context, got nil")
	}
	if gotClaims.Subject != userID {
		t.Errorf("sub: got %s, want %s", gotClaims.Subject, userID)
	}
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	m := newManager(t)
	authn := middleware.NewAuthenticator(m)

	handler := authn.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestAuthenticate_InvalidToken(t *testing.T) {
	m := newManager(t)
	authn := middleware.NewAuthenticator(m)

	handler := authn.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid.token.here")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("status: got %d, want 401", rr.Code)
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	m := newManager(t)
	checker := rbac.NewChecker()
	tok, _ := m.IssueAccessToken(uuid.New().String(), uuid.New().String(), "cae@test.com", "CAE", string(models.PersonaCAE))

	mw := middleware.RequirePermission(checker, rbac.ResourceEngagement, rbac.ActionManage)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims, _ := m.VerifyAccessToken(tok)
	ctx := context.WithValue(context.Background(), middleware.ClaimsKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status: got %d, want 200", rr.Code)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	m := newManager(t)
	checker := rbac.NewChecker()
	tok, _ := m.IssueAccessToken(uuid.New().String(), uuid.New().String(), "beta@test.com", "Beta", string(models.PersonaBetaTester))

	mw := middleware.RequirePermission(checker, rbac.ResourceEngagement, rbac.ActionWrite)
	handler := mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	claims, _ := m.VerifyAccessToken(tok)
	ctx := context.WithValue(context.Background(), middleware.ClaimsKey, claims)
	req := httptest.NewRequest(http.MethodPost, "/", nil).WithContext(ctx)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("status: got %d, want 403", rr.Code)
	}
}
