// Package handler implements the HTTP handlers for the audit-trail service.
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
	"go.uber.org/zap"

	"github.com/davejduke/obvious/services/audit-trail/internal/hashchain"
	"github.com/davejduke/obvious/services/audit-trail/internal/model"
	"github.com/davejduke/obvious/services/audit-trail/internal/store"
)

const (
	defaultLimit = 100
	maxLimit     = 1000
)

// Handler holds dependencies for all HTTP handlers.
type Handler struct {
	store  *store.Store
	logger *zap.Logger
}

// New returns a configured Handler.
func New(s *store.Store, logger *zap.Logger) *Handler {
	return &Handler{store: s, logger: logger}
}

// Routes registers all audit-trail routes on the given chi.Router.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/events", h.AppendEvent)
	r.Get("/events", h.QueryEvents)
	r.Post("/events/verify-chain", h.VerifyChain)
	r.Get("/events/meta-audit/{engagementId}", h.MetaAudit)
}

// ─────────────────────────────────────────────────────────────
// POST /events
// ─────────────────────────────────────────────────────────────

// AppendEvent appends a new immutable event to the audit log.
func (h *Handler) AppendEvent(w http.ResponseWriter, r *http.Request) {
	var req model.AppendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	if req.OrgID == (uuid.UUID{}) {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "org_id is required")
		return
	}
	if req.Action == "" {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "action is required")
		return
	}
	if req.ResourceType == "" {
		httpError(w, http.StatusBadRequest, "MISSING_FIELD", "resource_type is required")
		return
	}
	if !validEventType(req.EventType) {
		httpError(w, http.StatusBadRequest, "INVALID_EVENT_TYPE",
			"event_type must be one of: user_action, system_action, evidence_change, finding_change, engagement_change, auth_event")
		return
	}

	occurredAt := time.Now().UTC()

	// Fetch the previous hash inside a per-request context with a sensible timeout.
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	prevHash, err := h.store.GetLastHash(ctx, req.OrgID)
	if err != nil {
		h.logger.Error("failed to get last hash", zap.Error(err))
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve chain head")
		return
	}

	eventHash, err := hashchain.Compute(prevHash, &req, occurredAt)
	if err != nil {
		h.logger.Error("failed to compute hash", zap.Error(err))
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to compute event hash")
		return
	}

	event, err := h.store.Append(ctx, &req, eventHash, prevHash, occurredAt)
	if err != nil {
		h.logger.Error("failed to append event", zap.Error(err))
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to persist event")
		return
	}

	writeJSON(w, http.StatusCreated, event)
}

// ─────────────────────────────────────────────────────────────
// GET /events
// ─────────────────────────────────────────────────────────────

// QueryEvents supports two query modes:
//
//	?entity_id=<uuid>&entity_type=<string>   — events for a resource
//	?user_id=<uuid>&start=<RFC3339>&end=<RFC3339> — events by actor
func (h *Handler) QueryEvents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Derive org_id from the context (set by auth middleware) or header.
	orgIDStr := r.Header.Get("X-Org-ID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header must be a valid UUID")
		return
	}

	limit, offset := pagination(q.Get("limit"), q.Get("offset"))

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Branch 1: entity query
	if entityID := q.Get("entity_id"); entityID != "" {
		eid, err := uuid.Parse(entityID)
		if err != nil {
			httpError(w, http.StatusBadRequest, "INVALID_PARAM", "entity_id must be a valid UUID")
			return
		}
		entityType := q.Get("entity_type")
		if entityType == "" {
			httpError(w, http.StatusBadRequest, "MISSING_PARAM", "entity_type is required when entity_id is set")
			return
		}
		events, err := h.store.QueryByEntity(ctx, orgID, entityType, eid, limit, offset)
		if err != nil {
			h.logger.Error("query by entity failed", zap.Error(err))
			httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "query failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"events": events, "count": len(events)})
		return
	}

	// Branch 2: user + time range query
	if userIDStr := q.Get("user_id"); userIDStr != "" {
		actorID, err := uuid.Parse(userIDStr)
		if err != nil {
			httpError(w, http.StatusBadRequest, "INVALID_PARAM", "user_id must be a valid UUID")
			return
		}
		start, end, err := parseTimeRange(q.Get("start"), q.Get("end"))
		if err != nil {
			httpError(w, http.StatusBadRequest, "INVALID_PARAM", err.Error())
			return
		}
		events, err := h.store.QueryByUser(ctx, orgID, actorID, start, end, limit, offset)
		if err != nil {
			h.logger.Error("query by user failed", zap.Error(err))
			httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "query failed")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"events": events, "count": len(events)})
		return
	}

	httpError(w, http.StatusBadRequest, "MISSING_PARAM", "provide entity_id+entity_type or user_id+start+end")
}

// ─────────────────────────────────────────────────────────────
// POST /events/verify-chain
// ─────────────────────────────────────────────────────────────

// VerifyChain walks the entire chain for an org and confirms that every link
// is intact. Returns the first tampered event's ID if a break is found.
func (h *Handler) VerifyChain(w http.ResponseWriter, r *http.Request) {
	orgIDStr := r.Header.Get("X-Org-ID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header must be a valid UUID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	rows, err := h.store.AllEventsForOrg(ctx, orgID)
	if err != nil {
		h.logger.Error("verify chain: stream failed", zap.Error(err))
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to stream events")
		return
	}
	defer rows.Close()

	prevHash := hashchain.GenesisHash
	var totalEvents int64
	var tamperedAt *int64

	for rows.Next() {
		ev := &model.AuditEvent{}
		if err := store.ScanEvent(rows, ev); err != nil {
			h.logger.Error("verify chain: scan error", zap.Error(err))
			httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "scan error during verification")
			return
		}
		totalEvents++

		if tamperedAt == nil && !hashchain.Verify(ev, prevHash) {
			id := ev.ID
			tamperedAt = &id
		}
		prevHash = ev.EventHash
	}
	if err := rows.Err(); err != nil {
		h.logger.Error("verify chain: rows error", zap.Error(err))
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "rows error during verification")
		return
	}

	result := model.VerifyResult{
		Valid:       tamperedAt == nil,
		TotalEvents: totalEvents,
		TamperedAt:  tamperedAt,
	}
	if result.Valid {
		result.Message = "chain integrity verified"
	} else {
		result.Message = "chain integrity violation detected"
	}

	status := http.StatusOK
	if !result.Valid {
		status = http.StatusConflict
	}
	writeJSON(w, status, result)
}

// ─────────────────────────────────────────────────────────────
// GET /events/meta-audit/{engagementId}
// ─────────────────────────────────────────────────────────────

// MetaAudit returns the full chronological history of an engagement,
// enabling point-in-time reconstruction of who did what and when.
func (h *Handler) MetaAudit(w http.ResponseWriter, r *http.Request) {
	engagementIDStr := chi.URLParam(r, "engagementId")
	engagementID, err := uuid.Parse(engagementIDStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "INVALID_PARAM", "engagementId must be a valid UUID")
		return
	}

	orgIDStr := r.Header.Get("X-Org-ID")
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		httpError(w, http.StatusBadRequest, "MISSING_ORG", "X-Org-ID header must be a valid UUID")
		return
	}

	q := r.URL.Query()
	limit, offset := pagination(q.Get("limit"), q.Get("offset"))

	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	events, err := h.store.MetaAudit(ctx, orgID, engagementID, limit, offset)
	if err != nil {
		h.logger.Error("meta audit query failed", zap.Error(err))
		httpError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "meta audit query failed")
		return
	}

	// Return lightweight MetaAuditEntry projections.
	entries := make([]model.MetaAuditEntry, 0, len(events))
	for _, ev := range events {
		entries = append(entries, model.MetaAuditEntry{
			EventID:      ev.EventID,
			OccurredAt:   ev.OccurredAt,
			ActorEmail:   ev.ActorEmail,
			Action:       ev.Action,
			EventType:    ev.EventType,
			ResourceType: ev.ResourceType,
			ResourceID:   ev.ResourceID,
			EventHash:    ev.EventHash,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"engagement_id": engagementID,
		"events":        entries,
		"count":         len(entries),
	})
}

// ─────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────

func httpError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": message})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func validEventType(et model.EventType) bool {
	switch et {
	case model.EventTypeUserAction,
		model.EventTypeSystemAction,
		model.EventTypeEvidenceChange,
		model.EventTypeFindingChange,
		model.EventTypeEngagementChange,
		model.EventTypeAuthEvent:
		return true
	}
	return false
}

func parseTimeRange(startStr, endStr string) (time.Time, time.Time, error) {
	var start, end time.Time
	var err error

	if startStr == "" {
		start = time.Time{} // zero = no lower bound
	} else {
		start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			return start, end, fmt.Errorf("start must be RFC3339: %w", err)
		}
	}

	if endStr == "" {
		end = time.Now().UTC()
	} else {
		end, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return start, end, fmt.Errorf("end must be RFC3339: %w", err)
		}
	}
	return start, end, nil
}

func pagination(limitStr, offsetStr string) (int, int) {
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

