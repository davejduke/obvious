package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/davejduke/obvious/services/search/internal/models"
	"github.com/davejduke/obvious/services/search/internal/opensearch"
)

// mockOSServer runs a fake OpenSearch that returns a canned search response.
func newMockOSServer(hits []opensearch.SearchHit) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead:
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == "/_cluster/health":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "green"})
		default:
			w.Header().Set("Content-Type", "application/json")
			resp := map[string]interface{}{
				"hits": map[string]interface{}{
					"total": map[string]interface{}{"value": len(hits)},
					"hits":  hits,
				},
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
}

// TestSearchHandlerNoQuery verifies that missing q param returns 400.
func TestSearchHandlerNoQuery(t *testing.T) {
	os := opensearch.New("http://localhost:9200")
	h := New(os)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status: got %d, want 400", rr.Code)
	}
}

// TestSearchHandlerReturnsGroupedResults validates result grouping by entity type.
func TestSearchHandlerReturnsGroupedResults(t *testing.T) {
	hits := []opensearch.SearchHit{
		{
			Index: "aiauditor-findings",
			ID:    "f1",
			Score: 1.5,
			Source: map[string]interface{}{
				"title":       "NIS2 access control finding",
				"description": "Insufficient MFA coverage",
				"org_id":      "org1",
			},
			Highlight: map[string][]string{
				"title": {"NIS2 access <mark>control</mark> finding"},
			},
		},
		{
			Index:  "aiauditor-evidence",
			ID:     "e1",
			Score:  1.2,
			Source: map[string]interface{}{"title": "MFA audit log", "org_id": "org1"},
		},
		{
			Index:  "aiauditor-controls",
			ID:     "c1",
			Score:  0.9,
			Source: map[string]interface{}{"title": "Access control policy", "article_ref": "NIS2-21a", "org_id": "org1"},
		},
	}

	srv := newMockOSServer(hits)
	defer srv.Close()

	osClient := opensearch.New(srv.URL)
	h := New(osClient)

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?q=access+control", nil)
	h.Router().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status: got %d, want 200; body: %s", rr.Code, rr.Body.String())
	}

	var resp models.SearchResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if resp.Query != "access control" {
		t.Errorf("query: got %q, want 'access control'", resp.Query)
	}
	if resp.Total != 3 {
		t.Errorf("total: got %d, want 3", resp.Total)
	}
	if len(resp.Findings) != 1 {
		t.Errorf("findings: got %d, want 1", len(resp.Findings))
	}
	if len(resp.Evidence) != 1 {
		t.Errorf("evidence: got %d, want 1", len(resp.Evidence))
	}
	if len(resp.Controls) != 1 {
		t.Errorf("controls: got %d, want 1", len(resp.Controls))
	}

	// Verify highlight was applied to title
	if resp.Findings[0].Title != "NIS2 access <mark>control</mark> finding" {
		t.Errorf("highlight: got %q", resp.Findings[0].Title)
	}
}

// TestTypeFilter validates that type=finding only queries the findings index.
func TestTypeFilter(t *testing.T) {
	indices := indicesToQuery("finding")
	if len(indices) != 1 || indices[0] != "aiauditor-findings" {
		t.Errorf("indicesToQuery(finding): got %v", indices)
	}

	indices = indicesToQuery("evidence")
	if len(indices) != 1 || indices[0] != "aiauditor-evidence" {
		t.Errorf("indicesToQuery(evidence): got %v", indices)
	}

	indices = indicesToQuery("control")
	if len(indices) != 1 || indices[0] != "aiauditor-controls" {
		t.Errorf("indicesToQuery(control): got %v", indices)
	}

	indices = indicesToQuery("")
	if len(indices) != 3 {
		t.Errorf("indicesToQuery(all): got %v", indices)
	}
}

