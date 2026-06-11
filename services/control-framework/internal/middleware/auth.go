// Package middleware provides JWT authentication for the Control Framework Service.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const orgIDKey contextKey = "org_id"

// DevAuth is a development-only middleware that extracts org_id from the
// X-Org-Id header. In production this will be replaced by JWT RS256 verification.
//
// The Identity Service issues JWT tokens containing org_id claims; all services
// verify the token using the RS256 public key. Until the Identity Service is wired,
// X-Org-Id provides a testable auth boundary.
func DevAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		orgIDStr := r.Header.Get("X-Org-Id")

		// Also accept Bearer token header (for future JWT wiring)
		if orgIDStr == "" {
			bearer := r.Header.Get("Authorization")
			if strings.HasPrefix(bearer, "Bearer ") {
				// In production: parse JWT and extract org_id claim.
				// For now, treat the token as a bare UUID for testing.
				orgIDStr = strings.TrimPrefix(bearer, "Bearer ")
			}
		}

		if orgIDStr == "" {
			http.Error(w, `{"error":"missing X-Org-Id header"}`, http.StatusUnauthorized)
			return
		}

		orgID, err := uuid.Parse(orgIDStr)
		if err != nil {
			http.Error(w, `{"error":"invalid org_id format"}`, http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), orgIDKey, orgID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// OrgIDFromContext extracts the org UUID from a request context.
// Returns uuid.Nil, false if not present.
func OrgIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	v := ctx.Value(orgIDKey)
	if v == nil {
		return uuid.Nil, false
	}
	id, ok := v.(uuid.UUID)
	return id, ok
}

