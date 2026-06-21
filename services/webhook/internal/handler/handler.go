// Package handler implements the HTTP API for the webhook service.
package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	svclogging "github.com/davejduke/obvious/shared/logging"
	"github.com/davejduke/obvious/services/webhook/internal/domain"
	"github.com/davejduke/obvious/services/webhook/internal/store"
)

// Handler serves the webhook REST API.
type Handler struct {
	store  *store.Store
	logger *svclogging.Logger
}

// New returns a configured Handler.
func New(s *store.Store) *Handler {
	return &Handler{store: s, logger: svclogging.New("webhook")}
}

func (h *Handler) log() *svclogging.Logger {
	if h.logger == nil {
		return svclogging.New("webhook")
	}
	return h.logger
}

// Routes mounts all webhook routes onto r.
func (h *Handler) Routes(r chi.Router) {
	// Endpoint management (org-scoped)
	r.Post("/webhook-endpoints", h.CreateEndpoint)
	r.Get("/webhook-endpoints", h.ListEndpoints)
	r.Get("/webhook-endpoints/{id}", h.GetEndpoint)
	r.Delete("/webhook-endpoints/{id}", h.DeleteEndpoint)

	// Delivery history for an endpoint
	r.Get("/webhook-endpoints/{id}/deliveries", h.ListDeliveries)

	// Internal event dispatch (called by other services)
	r.Post("/internal/events", h.DispatchEvent)
}

// -----------------------------------------------------------------
// Endpoint handlers
// -----------------------------------------------------------------

// CreateEndpoint POST /webhook-endpoints
func (h *Handler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		httpErr(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header is required")
		return
	}
	var req struct {
		URL         string   `json:"url"`
		EventTypes  []string `json:"event_types"`
		Description string   `json:"description"`
		Enabled     *bool    `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if req.URL == "" {
		httpErr(w, http.StatusBadRequest, "MISSING_FIELD", "url is required")
		return
	}
	if !strings.HasPrefix(req.URL, "http://") && !strings.HasPrefix(req.URL, "https://") {
		httpErr(w, http.StatusBadRequest, "INVALID_URL", "url must start with http:// or https://")
		return
	}
	if len(req.EventTypes) == 0 {
		req.EventTypes = []string{"*"}
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	secret, err := generateSecret()
	if err != nil {
		h.log().Error(r.Context(), "endpoint.generate_secret_failed", map[string]any{"error": err.Error()})
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to generate secret")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*secTmout)
	defer cancel()

	ep, err := h.store.CreateEndpoint(ctx, &domain.Endpoint{
		OrgID:       orgID,
		URL:         req.URL,
		Secret:      secret,
		EventTypes:  req.EventTypes,
		Description: req.Description,
		Enabled:     enabled,
	})
	if err != nil {
		h.log().Error(r.Context(), "endpoint.create_failed", map[string]any{"error": err.Error()})
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to create endpoint")
		return
	}

	writeJSON(w, http.StatusCreated, ep)
}

// ListEndpoints GET /webhook-endpoints
func (h *Handler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		httpErr(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header is required")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*secTmout)
	defer cancel()

	eps, err := h.store.ListEndpoints(ctx, orgID)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to list endpoints")
		return
	}
	if eps == nil {
		eps = []*domain.Endpoint{}
	}
	// Mask secrets in list response.
	for _, ep := range eps {
		ep.Secret = maskSecret(ep.Secret)
	}
	writeJSON(w, http.StatusOK, map[string]any{"endpoints": eps, "count": len(eps)})
}

// GetEndpoint GET /webhook-endpoints/{id}
func (h *Handler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		httpErr(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header is required")
		return
	}
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 10*secTmout)
	defer cancel()

	ep, err := h.store.GetEndpoint(ctx, id, orgID)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to get endpoint")
		return
	}
	if ep == nil {
		httpErr(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
		return
	}
	ep.Secret = maskSecret(ep.Secret)
	writeJSON(w, http.StatusOK, ep)
}

// DeleteEndpoint DELETE /webhook-endpoints/{id}
func (h *Handler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		httpErr(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header is required")
		return
	}
	id := chi.URLParam(r, "id")
	ctx, cancel := context.WithTimeout(r.Context(), 10*secTmout)
	defer cancel()

	if err := h.store.DeleteEndpoint(ctx, id, orgID); err != nil {
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to delete endpoint")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListDeliveries GET /webhook-endpoints/{id}/deliveries
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	orgID := r.Header.Get("X-Org-ID")
	if orgID == "" {
		httpErr(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header is required")
		return
	}
	id := chi.URLParam(r, "id")

	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 200 {
			limit = n
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	// Validate the endpoint belongs to this org.
	ctx, cancel := context.WithTimeout(r.Context(), 15*secTmout)
	defer cancel()

	ep, err := h.store.GetEndpoint(ctx, id, orgID)
	if err != nil || ep == nil {
		httpErr(w, http.StatusNotFound, "NOT_FOUND", "endpoint not found")
		return
	}

	deliveries, err := h.store.ListDeliveries(ctx, id, limit, offset)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to list deliveries")
		return
	}
	if deliveries == nil {
		deliveries = []*domain.Delivery{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": deliveries, "count": len(deliveries)})
}

// DispatchEvent POST /internal/events
// Called by other services to fan-out an event to all matching endpoints.
func (h *Handler) DispatchEvent(w http.ResponseWriter, r *http.Request) {
	var req domain.DispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpErr(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if req.OrgID == "" || req.EventType == "" || req.Payload == nil {
		httpErr(w, http.StatusBadRequest, "MISSING_FIELD", "org_id, event_type, and payload are required")
		return
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to marshal payload")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 15*secTmout)
	defer cancel()

	endpoints, err := h.store.EnabledEndpointsForEvent(ctx, req.OrgID, req.EventType)
	if err != nil {
		h.log().Error(r.Context(), "dispatch.query_endpoints_failed", map[string]any{"error": err.Error()})
		httpErr(w, http.StatusInternalServerError, "SERVER_ERROR", "failed to query endpoints")
		return
	}

	var enqueued int
	for _, ep := range endpoints {
		if _, err := h.store.EnqueueDelivery(ctx, ep.ID, req.EventType, payloadBytes); err != nil {
			h.log().Warn(r.Context(), "dispatch.enqueue_failed",
				map[string]any{"endpoint_id": ep.ID, "error": err.Error()})
			continue
		}
		enqueued++
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"event_type": req.EventType,
		"endpoints":  len(endpoints),
		"enqueued":   enqueued,
	})
}

// -----------------------------------------------------------------
// Helpers
// -----------------------------------------------------------------

const secTmout = 1000000000 // 1 second in nanoseconds (time.Second)

func httpErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// generateSecret creates a 32-byte random hex secret.
func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// maskSecret shows only the last 4 chars of a secret.
func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return "****" + s[len(s)-4:]
}

