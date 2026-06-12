// Package middleware provides Chi-compatible HTTP middleware for the identity service.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/davejduke/obvious/services/identity/internal/models"
	jwtpkg "github.com/davejduke/obvious/services/identity/pkg/jwt"
	"github.com/davejduke/obvious/services/identity/pkg/rbac"
)

// ContextKey is the type for context keys in this package.
type ContextKey string

const (
	// ClaimsKey is the context key where JWT AccessClaims are stored.
	ClaimsKey ContextKey = "identity_claims"
)

// Authenticator validates JWTs and sets claims in request context.
type Authenticator struct {
	jwt *jwtpkg.Manager
}

// NewAuthenticator creates an Authenticator.
func NewAuthenticator(jwt *jwtpkg.Manager) *Authenticator {
	return &Authenticator{jwt: jwt}
}

// Authenticate is Chi middleware that validates the Bearer token.
func (a *Authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized(w, "missing authorization header")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			writeUnauthorized(w, "invalid authorization header format")
			return
		}
		claims, err := a.jwt.VerifyAccessToken(parts[1])
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission returns Chi middleware that enforces an RBAC permission.
func RequirePermission(checker *rbac.Checker, resource rbac.Resource, action rbac.Action) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "authentication required")
				return
			}
			persona := models.Persona(claims.Persona)
			if !checker.Can(persona, resource, action) {
				writeForbidden(w, "insufficient permissions for this operation")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// ClaimsFromContext extracts JWT claims from context.
func ClaimsFromContext(ctx context.Context) *jwtpkg.AccessClaims {
	v := ctx.Value(ClaimsKey)
	if v == nil {
		return nil
	}
	c, _ := v.(*jwtpkg.AccessClaims)
	return c
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"code":"UNAUTHORIZED","message":"` + msg + `"}`)) //nolint:errcheck
}

func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"code":"FORBIDDEN","message":"` + msg + `"}`)) //nolint:errcheck
}
