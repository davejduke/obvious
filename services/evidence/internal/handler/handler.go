// Package handler implements the Evidence Service HTTP handlers using Chi router.
// Endpoints:
//   POST   /evidence                      — ingest single or batch evidence
//   GET    /evidence/:id                  — get evidence by ID
//   GET    /evidence?control_id=X         — list evidence for a control
//   POST   /evidence/:id/classify         — (re-)classify evidence
//   GET    /evidence/:id/quality-score    — get quality score
//   GET    /controls/:controlId/sufficiency — get sufficiency for a control
package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/classifier"
	"github.com/davejduke/obvious/services/evidence/internal/ingester"
	"github.com/davejduke/obvious/services/evidence/internal/models"
	"github.com/davejduke/obvious/services/evidence/internal/repository"
	"github.com/davejduke/obvious/services/evidence/internal/scorer"
	"github.com/davejduke/obvious/services/evidence/internal/sufficiency"
)

// Handler wires together the evidence service's HTTP routes.
type Handler struct {
	repo        repository.Repository
	ingester    *ingester.Ingester
	classifier  *classifier.Classifier
	scorer      *scorer.Scorer
	sufficiency *sufficiency.Calculator
}

// New creates a Handler using the provided repository (may be memory or postgres).
func New(repo repository.Repository) *Handler {
	return &Handler{
		repo:        repo,
		ingester:    ingester.New(),
		classifier:  classifier.New(),
		scorer:      scorer.New(),
		sufficiency: sufficiency.New(),
	}
}

// Router builds and returns the Chi router for the evidence service.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.health)
	r.Get("/ready", h.ready)

	r.Post("/evidence", h.ingestEvidence)
	r.Get("/evidence/{id}", h.getEvidence)
	r.Get("/evidence", h.listEvidenceByControl)
	r.Post("/evidence/{id}/classify", h.classifyEvidence)
	r.Get("/evidence/{id}/quality-score", h.getQualityScore)
	r.Get("/controls/{controlId}/sufficiency", h.getControlSufficiency)

	return r
}

// health returns 200 for liveness checks.
func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"service": "evidence", "status": "healthy", "version": "0.1.0"})
}

// ready returns 200 for readiness checks.
func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ready": true})
}

// ingestEvidence handles POST /evidence.
// Accepts a single IngestRequest or a BatchIngestRequest.
func (h *Handler) ingestEvidence(w http.ResponseWriter, r *http.Request) {
	var raw json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Detect batch vs single by checking for "items" key
	var probe struct {
		Items json.RawMessage `json:"items"`
	}
	isBatch := json.Unmarshal(raw, &probe) == nil && len(probe.Items) > 0 && strings.HasPrefix(strings.TrimSpace(string(probe.Items)), "[")

	if isBatch {
		var batch models.BatchIngestRequest
		if err := json.Unmarshal(raw, &batch); err != nil {
			writeError(w, http.StatusBadRequest, "invalid batch payload: "+err.Error())
			return
		}
		h.handleBatchIngest(w, batch.Items)
		return
	}

	var req models.IngestRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid evidence payload: "+err.Error())
		return
	}
	h.handleSingleIngest(w, req)
}

func (h *Handler) handleSingleIngest(w http.ResponseWriter, req models.IngestRequest) {
	ev, err := h.ingester.Ingest(&req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.repo.Save(ev); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to store evidence")
		return
	}

	// Auto-classify and score on ingest
	classification := h.classifier.Classify(ev)
	_ = h.repo.SaveClassification(classification)

	peers, _ := h.repo.FindByControlID(ev.ControlID)
	qs := h.scorer.Score(ev, peers)
	_ = h.repo.SaveQualityScore(qs)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"evidence":       ev,
		"classification": classification,
		"quality_score":  qs,
	})
}

func (h *Handler) handleBatchIngest(w http.ResponseWriter, items []models.IngestRequest) {
	type result struct {
		Evidence       *models.Evidence       `json:"evidence"`
		Classification *models.Classification `json:"classification"`
		QualityScore   *models.QualityScore   `json:"quality_score"`
		Error          string                 `json:"error,omitempty"`
	}
	results := make([]result, 0, len(items))
	for _, req := range items {
		ev, err := h.ingester.Ingest(&req)
		if err != nil {
			results = append(results, result{Error: err.Error()})
			continue
		}
		_ = h.repo.Save(ev)
		classification := h.classifier.Classify(ev)
		_ = h.repo.SaveClassification(classification)
		peers, _ := h.repo.FindByControlID(ev.ControlID)
		qs := h.scorer.Score(ev, peers)
		_ = h.repo.SaveQualityScore(qs)
		results = append(results, result{Evidence: ev, Classification: classification, QualityScore: qs})
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"results": results})
}

// getEvidence handles GET /evidence/{id}.
func (h *Handler) getEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid evidence id")
		return
	}
	ev, err := h.repo.FindByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	writeJSON(w, http.StatusOK, ev)
}

// listEvidenceByControl handles GET /evidence?control_id=X.
func (h *Handler) listEvidenceByControl(w http.ResponseWriter, r *http.Request) {
	controlIDStr := r.URL.Query().Get("control_id")
	if controlIDStr == "" {
		writeError(w, http.StatusBadRequest, "control_id query parameter is required")
		return
	}
	controlID, err := parseUUID(controlIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid control_id")
		return
	}
	evidence, err := h.repo.FindByControlID(controlID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query evidence")
		return
	}
	if evidence == nil {
		evidence = []*models.Evidence{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"evidence": evidence, "count": len(evidence)})
}

// classifyEvidence handles POST /evidence/{id}/classify (re-classify on demand).
func (h *Handler) classifyEvidence(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid evidence id")
		return
	}
	ev, err := h.repo.FindByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "evidence not found")
		return
	}
	classification := h.classifier.Classify(ev)
	_ = h.repo.SaveClassification(classification)
	writeJSON(w, http.StatusOK, classification)
}

// getQualityScore handles GET /evidence/{id}/quality-score.
func (h *Handler) getQualityScore(w http.ResponseWriter, r *http.Request) {
	id, err := parseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid evidence id")
		return
	}
	qs, err := h.repo.FindQualityScoreByEvidenceID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "quality score not found")
		return
	}
	writeJSON(w, http.StatusOK, qs)
}

// getControlSufficiency handles GET /controls/{controlId}/sufficiency.
func (h *Handler) getControlSufficiency(w http.ResponseWriter, r *http.Request) {
	controlID, err := parseUUID(chi.URLParam(r, "controlId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid control_id")
		return
	}

	riskRatingStr := r.URL.Query().Get("risk_rating")
	if riskRatingStr == "" {
		riskRatingStr = string(models.RiskMedium)
	}
	riskRating := models.RiskRating(riskRatingStr)

	evidence, err := h.repo.FindByControlID(controlID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query evidence")
		return
	}
	scores, err := h.repo.FindQualityScoresByControlID(controlID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to query quality scores")
		return
	}

	result := h.sufficiency.Evaluate(riskRating, evidence, scores, riskRating)
	result.ControlID = controlID
	writeJSON(w, http.StatusOK, result)
}

// --- helpers ---

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

