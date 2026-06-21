// Package proxy implements the reverse-proxy routing layer for the API gateway.
package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	gatewaymw "github.com/davejduke/obvious/services/gateway/internal/middleware"
	"github.com/davejduke/obvious/services/gateway/internal/pagination"
)

// ServiceConfig holds upstream URLs for all 8 backend services.
type ServiceConfig struct {
	IdentityURL    string
	ControlsURL    string
	EvidenceURL    string
	EngagementURL  string
	IntegrationURL string
	AuditTrailURL  string
	ReportingURL   string
	WebhooksURL    string
}

// Build constructs the gateway Chi router with all routes and middleware.
func Build(
	cfg ServiceConfig,
	auth *gatewaymw.Authenticator,
	rl *gatewaymw.RateLimiter,
	frontendOrigin string,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(corsMiddleware(frontendOrigin))
	r.Use(rl.RateLimit)

	// Health check (public — no auth required)
	r.Get("/health", healthHandler)

	// API v1 — auth-required routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(auth.Authenticate)

		// ── Identity Service (/api/v1/identity/*) ───────────────────────────────
		r.Mount("/identity", proxyTo(cfg.IdentityURL, "/api/v1/identity"))

		// ── Control Framework (/api/v1/controls/*) ──────────────────────────────
		r.Mount("/controls", withPermission("control", "read",
			proxyTo(cfg.ControlsURL, "/api/v1/controls"),
		))

		// ── Evidence Pipeline (/api/v1/evidence/*) ──────────────────────────────
		r.Mount("/evidence", withPermission("evidence", "read",
			proxyTo(cfg.EvidenceURL, "/api/v1/evidence"),
		))

		// ── Engagement Service (/api/v1/engagements/*) ──────────────────────────
		r.Mount("/engagements", withPermission("engagement", "read",
			proxyTo(cfg.EngagementURL, "/api/v1/engagements"),
		))

		// ── Integration Service (/api/v1/integrations/*) ────────────────────────
		r.Mount("/integrations", withPermission("engagement", "read",
			proxyTo(cfg.IntegrationURL, "/api/v1/integrations"),
		))

		// ── Audit Trail (/api/v1/audit-trail/*) ─────────────────────────────────
		r.Mount("/audit-trail", withPermission("audit_trail", "read",
			proxyTo(cfg.AuditTrailURL, "/api/v1/audit-trail"),
		))

		// ── Reporting (/api/v1/reports/*) ───────────────────────────────────────
		r.Mount("/reports", withPermission("report", "read",
			proxyTo(cfg.ReportingURL, "/api/v1/reports"),
		))

		// ── Webhooks (/api/v1/webhooks/*) ───────────────────────────────────────
		// Subscription CRUD is org-scoped; /dispatch is internal-only.
		r.Mount("/webhooks", proxyTo(cfg.WebhooksURL, "/api/v1/webhooks"))

		// ── Pagination demo endpoint ─────────────────────────────────────────────
		r.Get("/paginate", paginateHandler)

	})

	return r
}

// proxyTo builds a reverse-proxy http.Handler that strips stripPrefix from the
// request path before forwarding to upstream.
func proxyTo(upstream, stripPrefix string) http.Handler {
	target, err := url.Parse(upstream)
	if err != nil {
		panic(fmt.Sprintf("gateway/proxy: invalid upstream URL %q: %v", upstream, err))
	}
	rp := httputil.NewSingleHostReverseProxy(target)
	orig := rp.Director
	rp.Director = func(req *http.Request) {
		orig(req)
		// Strip the gateway prefix so upstream only sees its own paths.
		req.URL.Path = strings.TrimPrefix(req.URL.Path, stripPrefix)
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		req.URL.RawPath = ""
		// Forward original host
		req.Header.Set("X-Forwarded-Host", req.Host)
		// Propagate the request ID for distributed tracing.
		if rid := req.Header.Get("X-Request-ID"); rid != "" {
			req.Header.Set("X-Request-ID", rid)
		}
	}
	rp.ModifyResponse = func(resp *http.Response) error {
		// Remove internal hop-by-hop headers.
		resp.Header.Del("X-Powered-By")
		return nil
	}
	return rp
}

// withPermission wraps a handler in an RBAC check.
func withPermission(resource, action string, h http.Handler) http.Handler {
	return gatewaymw.RequirePermission(resource, action)(h)
}

// healthHandler returns a simple JSON health response.
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"gateway","status":"healthy","version":"0.1.0"}`)
}

// paginateHandler demonstrates cursor-based pagination parsing/generation.
func paginateHandler(w http.ResponseWriter, r *http.Request) {
	cur := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")

	limit, err := pagination.ParseLimit(limitStr, 20)
	if err != nil {
		http.Error(w, `{"error":"invalid limit"}`, http.StatusBadRequest)
		return
	}

	var decoded string
	if cur != "" {
		decoded, err = pagination.DecodeCursor(cur)
		if err != nil {
			http.Error(w, `{"error":"invalid cursor"}`, http.StatusBadRequest)
			return
		}
	}

	// Echo back for API contract demonstration.
	nextCursor := pagination.EncodeCursor("next-page-token")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"cursor_decoded":%q,"limit":%d,"next_cursor":%q}`, decoded, limit, nextCursor)
}

// corsMiddleware allows the configured frontend origin.
func corsMiddleware(origin string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "X-RateLimit-Limit,X-RateLimit-Remaining,X-RateLimit-Reset,X-Request-ID")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
