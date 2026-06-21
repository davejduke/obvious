// Package handler implements the HTTP handlers for the webhooks service.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	svclogging "github.com/davejduke/obvious/shared/logging"
	"github.com/davejduke/obvious/services/webhooks/internal/delivery"
	"github.com/davejduke/obvious/services/webhooks/internal/model"
	"github.com/davejduke/obvious/services/webhooks/internal/signer"
	"github.com/davejduke/obvious/services/webhooks/internal/store"
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	store    *store.Store
	worker   *delivery.Worker
	logger   *svclogging.Logger
}

// New returns a configured Handler.
func New(s *store.Store, w *delivery.Worker) *Handler {
	return &Handler{store: s, worker: w, logger: svclogging.New("webhooks")}
}

// Routes registers all webhook routes on the given chi.Router.
//
// Route layout:
//
//	  POST   /organizations/{orgId}/webhooks              — create subscription
//	  GET    /organizations/{orgId}/webhooks              — list subscriptions
//	  GET    /organizations/{orgId}/webhooks/{webhookId}  — get subscription
//	  PUT    /organizations/{orgId}/webhooks/{webhookId}  — update subscription
//	  DELETE /organizations/{orgId}/webhooks/{webhookId}  — delete subscription
//	  GET    /organizations/{orgId}/webhooks/{webhookId}/deliveries — delivery history
//	  POST   /dispatch                                    — internal: fan-out event
//	  GET    /admin/health                                — admin health dashboard
func (h *Handler) Routes(r chi.Router) {
	r.Route("/organizations/{orgId}/webhooks", func(r chi.Router) {
		r.Post("/", h.CreateSubscription)
		r.Get("/", h.ListSubscriptions)
		r.Get("/{webhookId}", h.GetSubscription)
		r.Put("/{webhookId}", h.UpdateSubscription)
		r.Delete("/{webhookId}", h.DeleteSubscription)
		r.Get("/{webhookId}/deliveries", h.ListDeliveries)
	})

	// Internal: called by other services to dispatch a platform event.
	r.Post("/dispatch", h.Dispatch)

	// Admin dashboard.
	r.Get("/admin/health", h.AdminHealth)
}

// ─────────────────────────────────────────────────────────────
// Subscription CRUD
// ─────────────────────────────────────────────────────────────

// CreateSubscription handles POST /organizations/{orgId}/webhooks
func (h *Handler) CreateSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}

	var req model.CreateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.URL == "" {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "url is required")
		return
	}
	if req.Secret == "" {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "secret is required")
		return
	}
	if len(req.EventTypes) == 0 {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "event_types must not be empty")
		return
	}
	if err := validateEventTypes(req.EventTypes); err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_EVENT_TYPE", err.Error())
		return
	}

	secretHash := signer.HashSecret(req.Secret)

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sub, err := h.store.CreateSubscription(
		ctx, orgID,
		req.URL, req.Secret, secretHash,
		req.EventTypes, req.Description,
		nil,
	)
	if err != nil {
		h.logger.Error(ctx, "webhook.create_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create subscription")
		return
	}

	writeJSON(w, http.StatusCreated, sub)
}

// ListSubscriptions handles GET /organizations/{orgId}/webhooks
func (h *Handler) ListSubscriptions(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	subs, err := h.store.ListSubscriptions(ctx, orgID)
	if err != nil {
		h.logger.Error(ctx, "webhook.list_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list subscriptions")
		return
	}

	if subs == nil {
		subs = []*model.WebhookSubscription{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscriptions": subs, "count": len(subs)})
}

// GetSubscription handles GET /organizations/{orgId}/webhooks/{webhookId}
func (h *Handler) GetSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}
	webhookID, ok := parseWebhookID(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sub, err := h.store.GetSubscription(ctx, orgID, webhookID)
	if err != nil {
		h.logger.Error(ctx, "webhook.get_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get subscription")
		return
	}
	if sub == nil {
		httpError(w, http.StatusNotFound, "NOT_FOUND", "webhook subscription not found")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// UpdateSubscription handles PUT /organizations/{orgId}/webhooks/{webhookId}
func (h *Handler) UpdateSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}
	webhookID, ok := parseWebhookID(w, r)
	if !ok {
		return
	}

	var req model.UpdateSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if len(req.EventTypes) > 0 {
		if err := validateEventTypes(req.EventTypes); err != nil {
			httpError(w, http.StatusBadRequest, "INVALID_EVENT_TYPE", err.Error())
			return
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	sub, err := h.store.UpdateSubscription(ctx, orgID, webhookID, &req)
	if err != nil {
		h.logger.Error(ctx, "webhook.update_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update subscription")
		return
	}
	if sub == nil {
		httpError(w, http.StatusNotFound, "NOT_FOUND", "webhook subscription not found")
		return
	}
	writeJSON(w, http.StatusOK, sub)
}

// DeleteSubscription handles DELETE /organizations/{orgId}/webhooks/{webhookId}
func (h *Handler) DeleteSubscription(w http.ResponseWriter, r *http.Request) {
	orgID, ok := parseOrgID(w, r)
	if !ok {
		return
	}
	webhookID, ok := parseWebhookID(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.store.DeleteSubscription(ctx, orgID, webhookID); err != nil {
		h.logger.Error(ctx, "webhook.delete_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete subscription")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ListDeliveries handles GET /organizations/{orgId}/webhooks/{webhookId}/deliveries
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	_, ok := parseOrgID(w, r) // validates org scope
	if !ok {
		return
	}
	webhookID, ok := parseWebhookID(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	limit, offset := paginationParams(q.Get("limit"), q.Get("offset"))

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	deliveries, err := h.store.ListDeliveriesForSubscription(ctx, webhookID, limit, offset)
	if err != nil {
		h.logger.Error(ctx, "webhook.list_deliveries_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list deliveries")
		return
	}
	if deliveries == nil {
		deliveries = []*model.WebhookDelivery{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"deliveries": deliveries, "count": len(deliveries)})
}

// ─────────────────────────────────────────────────────────────
// Event dispatch & admin
// ─────────────────────────────────────────────────────────────

// Dispatch handles POST /dispatch — internal endpoint for other services to
// trigger webhook delivery for a platform event.
func (h *Handler) Dispatch(w http.ResponseWriter, r *http.Request) {
	var req model.DispatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.OrgID == (uuid.UUID{}) {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "org_id is required")
		return
	}
	if req.EventType == "" {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "event_type is required")
		return
	}
	if err := validateEventTypes([]string{req.EventType}); err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_EVENT_TYPE", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := h.worker.Dispatch(ctx, req.OrgID, req.EventType, req.Data); err != nil {
		h.logger.Error(ctx, "webhook.dispatch_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "dispatch failed")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":     "dispatched",
		"org_id":     req.OrgID,
		"event_type": req.EventType,
	})
}

// AdminHealth handles GET /admin/health — returns per-org delivery health stats.
func (h *Handler) AdminHealth(w http.ResponseWriter, r *http.Request) {
	orgIDStr := r.URL.Query().Get("org_id")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_PARAM", "org_id query param must be a valid UUID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stats, err := h.store.HealthStats(ctx, orgID)
	if err != nil {
		h.logger.Error(ctx, "webhook.health_stats_failed", map[string]any{"error": err.Error()})
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve health stats")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"org_id":  orgID,
		"service": "webhooks",
		"stats":   stats,
	})
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

func parseOrgID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgIDStr := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_PARAM", "orgId must be a valid UUID")
		return uuid.UUID{}, false
	}
	return orgID, true
}

func parseWebhookID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	webhookIDStr := chi.URLParam(r, "webhookId")
	webhookID, err := uuid.Parse(webhookIDStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_PARAM", "webhookId must be a valid UUID")
		return uuid.UUID{}, false
	}
	return webhookID, true
}

func validateEventTypes(eventTypes []string) error {
	valid := make(map[string]bool)
	for _, et := range model.AllEventTypes {
		valid[string(et)] = true
	}
	for _, et := range eventTypes {
		if !valid[et] {
			return fmt.Errorf("unknown event type %q; valid: evidence.intake.complete, reasoning.conclusion, finding.status.changed", et)
		}
	}
	return nil
}

func paginationParams(limitStr, offsetStr string) (int, int) {
	const defaultLimit = 50
	const maxLimit = 500
	limit := defaultLimit
	offset := 0
	if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
		if v > maxLimit {
			v = maxLimit
		}
		limit = v
	}
	if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}

func httpError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

