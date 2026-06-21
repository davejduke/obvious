package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newTestRouter() chi.Router {
	r := chi.NewRouter()
	// Handler without a real DB; routes will return 500s for DB ops, but
	// we test input validation which doesn't touch the store.
	h := &Handler{store: nil, logger: nil}
	h.Routes(r)
	return r
}

// TestCreateEndpoint_MissingOrgID verifies that requests without X-Org-ID are rejected.
func TestCreateEndpoint_MissingOrgID(t *testing.T) {
	r := newTestRouter()
	body := `{"url":"https://example.com/hook"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook-endpoints", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	var resp map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["code"] != "MISSING_ORG" {
		t.Errorf("expected MISSING_ORG, got %q", resp["code"])
	}
}

// TestCreateEndpoint_MissingURL verifies that missing URL is rejected.
func TestCreateEndpoint_MissingURL(t *testing.T) {
	r := newTestRouter()
	body := `{"description":"test"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook-endpoints", bytes.NewBufferString(body))
	req.Header.Set("X-Org-ID", "org-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["code"] != "MISSING_FIELD" {
		t.Errorf("expected MISSING_FIELD, got %q", resp["code"])
	}
}

// TestCreateEndpoint_InvalidURL verifies that non-HTTP URLs are rejected.
func TestCreateEndpoint_InvalidURL(t *testing.T) {
	r := newTestRouter()
	body := `{"url":"ftp://example.com/hook"}`
	req := httptest.NewRequest(http.MethodPost, "/webhook-endpoints", bytes.NewBufferString(body))
	req.Header.Set("X-Org-ID", "org-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["code"] != "INVALID_URL" {
		t.Errorf("expected INVALID_URL, got %q", resp["code"])
	}
}

// TestDispatchEvent_MissingFields verifies that dispatch without required fields is rejected.
func TestDispatchEvent_MissingFields(t *testing.T) {
	r := newTestRouter()
	body := `{"org_id":"org-1"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/events", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

// TestMaskSecret verifies secret masking behaviour.
func TestMaskSecret(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abc", "****"},
		{"abcd", "****abcd"},
		{"deadbeefcafe1234", "****1234"},
	}
	for _, tt := range tests {
		got := maskSecret(tt.input)
		if got != tt.want {
			t.Errorf("maskSecret(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestGenerateSecret verifies that generateSecret returns 64 hex chars.
func TestGenerateSecret(t *testing.T) {
	s, err := generateSecret()
	if err != nil {
		t.Fatalf("generateSecret: %v", err)
	}
	if len(s) != 64 {
		t.Errorf("expected 64 chars, got %d", len(s))
	}
	s2, _ := generateSecret()
	if s == s2 {
		t.Error("two secrets should not be identical")
	}
}

