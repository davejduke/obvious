// Package handler implements the Search Service HTTP API.
// Endpoints:
//
//	GET /health                              — liveness
//	GET /ready                               — readiness
//	GET /api/v1/search?q=...&type=...&size=N — full-text search
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/davejduke/obvious/services/search/internal/models"
	"github.com/davejduke/obvious/services/search/internal/opensearch"
	"github.com/davejduke/obvious/services/search/internal/indexer"
)

// Handler handles HTTP requests for the search service.
type Handler struct {
	os *opensearch.Client
}

// New creates a Handler backed by the given OpenSearch client.
func New(osClient *opensearch.Client) *Handler {
	return &Handler{os: osClient}
}

// Router builds and returns the Chi router.
func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/health", h.health)
	r.Get("/ready", h.ready)
	r.Get("/api/v1/search", h.search)

	return r
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"service": "search", "status": "healthy", "version": "0.1.0"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ready": true})
}

// search handles GET /api/v1/search?q=<query>[&type=finding|evidence|control][&size=N]
func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeError(w, http.StatusBadRequest, "q parameter is required")
		return
	}

	typeFilter := r.URL.Query().Get("type") // finding|evidence|control|"" (all)
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if size <= 0 {
		size = 20
	}

	// Determine which indices to query
	indices := indicesToQuery(typeFilter)

	osResp, err := h.os.Search(r.Context(), indices, q, size)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "search unavailable: "+err.Error())
		return
	}

	resp := buildResponse(q, osResp)
	writeJSON(w, http.StatusOK, resp)
}

// indicesToQuery maps an optional type filter to the list of index names.
func indicesToQuery(typeFilter string) []string {
	switch typeFilter {
	case "finding":
		return []string{indexer.IndexFindings}
	case "evidence":
		return []string{indexer.IndexEvidence}
	case "control":
		return []string{indexer.IndexControls}
	default:
		return []string{indexer.IndexFindings, indexer.IndexEvidence, indexer.IndexControls}
	}
}

// buildResponse groups OpenSearch hits by index into a SearchResponse.
func buildResponse(query string, osResp *opensearch.SearchResponse) models.SearchResponse {
	resp := models.SearchResponse{
		Query:    query,
		Findings: []models.SearchResult{},
		Evidence: []models.SearchResult{},
		Controls: []models.SearchResult{},
	}

	for _, hit := range osResp.Hits.Hits {
		result := hitToResult(hit)
		switch hit.Index {
		case indexer.IndexFindings:
			result.Type = models.EntityFinding
			resp.Findings = append(resp.Findings, result)
		case indexer.IndexEvidence:
			result.Type = models.EntityEvidence
			resp.Evidence = append(resp.Evidence, result)
		case indexer.IndexControls:
			result.Type = models.EntityControl
			resp.Controls = append(resp.Controls, result)
		}
		resp.Total++
	}

	return resp
}

func hitToResult(hit opensearch.SearchHit) models.SearchResult {
	result := models.SearchResult{
		ID:         hit.ID,
		Score:      hit.Score,
		Highlights: hit.Highlight,
		Meta:       hit.Source,
	}
	// Lift title/description from _source for convenience
	if v, ok := hit.Source["title"].(string); ok {
		result.Title = v
	}
	if v, ok := hit.Source["description"].(string); ok {
		result.Description = v
	}
	if v, ok := hit.Source["org_id"].(string); ok {
		result.OrgID = v
	}
	// Prefer highlighted title if available
	if hl, ok := result.Highlights["title"]; ok && len(hl) > 0 {
		result.Title = hl[0]
	}
	return result
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

