package harness_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/davejduke/obvious/sdk/connector/harness"
)

func TestHarness_AddResponseAndRequest(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/health", http.StatusOK, `{"status":"ok"}`)

	resp, err := http.Get(h.URL() + "/health")
	if err != nil {
		t.Fatalf("GET /health: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "ok") {
		t.Errorf("expected body to contain 'ok', got %q", string(body))
	}
}

func TestHarness_StickyLastResponse(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/data", http.StatusOK, `{"items":[]}`)

	for i := 0; i < 3; i++ {
		resp, err := http.Get(h.URL() + "/data")
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i, resp.StatusCode)
		}
	}
	if h.RequestCount("/data") != 3 {
		t.Errorf("expected 3 recorded requests, got %d", h.RequestCount("/data"))
	}
}

func TestHarness_QueuedResponses(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/seq", http.StatusOK, `{"n":1}`)
	h.AddResponse("/seq", http.StatusOK, `{"n":2}`)
	h.AddResponse("/seq", http.StatusOK, `{"n":3}`)

	for want := 1; want <= 3; want++ {
		resp, _ := http.Get(h.URL() + "/seq")
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), string(rune('0'+want))) {
			t.Errorf("request %d: expected n=%d in body, got %q", want, want, string(body))
		}
	}
}

func TestHarness_404WhenNoFixture(t *testing.T) {
	h := harness.New(t)
	resp, err := http.Get(h.URL() + "/unknown")
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404, got %d", resp.StatusCode)
	}
}

func TestHarness_RecordsRequestBody(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/ingest", http.StatusAccepted, `{}`)

	http.Post(h.URL()+"/ingest", "application/json", strings.NewReader(`{"key":"value"}`))

	reqs := h.Requests("/ingest")
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	if !strings.Contains(string(reqs[0].Body), "value") {
		t.Errorf("expected body to contain 'value', got %q", string(reqs[0].Body))
	}
}

func TestHarness_Reset(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/a", http.StatusOK, `{}`)
	http.Get(h.URL() + "/a")
	h.Reset()
	if h.RequestCount("/a") != 0 {
		t.Error("expected RequestCount=0 after Reset")
	}
}

func TestHarness_AddJSONResponse(t *testing.T) {
	h := harness.New(t)
	h.AddJSONResponse("/api", http.StatusOK, map[string]string{"hello": "world"})

	resp, _ := http.Get(h.URL() + "/api")
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if !strings.Contains(string(body), "world") {
		t.Errorf("expected JSON body, got %q", string(body))
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("expected application/json content-type, got %q", ct)
	}
}
