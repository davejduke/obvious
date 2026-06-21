package rest_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	connector "github.com/davejduke/obvious/sdk/connector"
	"github.com/davejduke/obvious/sdk/connector/auth"
	"github.com/davejduke/obvious/sdk/connector/examples/rest"
	"github.com/davejduke/obvious/sdk/connector/harness"
)

func newTestConnector(t *testing.T, h *harness.Harness) *rest.RESTConnector {
	t.Helper()
	c, err := rest.New(rest.Config{
		Name:       "test-rest",
		BaseURL:    h.URL(),
		DataPath:   "/data",
		HealthPath: "/health",
		Auth:       auth.NewAPIKeyAuth(auth.APIKeyConfig{Key: "test-key"}),
		RecordType: "finding",
		HTTPClient: h.Client(),
	})
	if err != nil {
		t.Fatalf("rest.New: %v", err)
	}
	return c
}

func TestRESTConnector_Connect(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/health", http.StatusOK, `{"status":"ok"}`)

	c := newTestConnector(t, h)
	if err := c.Connect(context.Background()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if h.RequestCount("/health") != 1 {
		t.Errorf("expected 1 health request, got %d", h.RequestCount("/health"))
	}
}

func TestRESTConnector_ConnectFails(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/health", http.StatusServiceUnavailable, `{"error":"down"}`)

	c := newTestConnector(t, h)
	if err := c.Connect(context.Background()); err == nil {
		t.Fatal("expected Connect to fail for 503")
	}
}

func TestRESTConnector_Sync(t *testing.T) {
	h := harness.New(t)
	h.AddJSONResponse("/data", http.StatusOK, map[string]any{
		"items": []map[string]any{
			{"id": "rec-1", "title": "First Finding", "severity": "HIGH", "updated_at": time.Now().UTC().Format(time.RFC3339)},
			{"id": "rec-2", "title": "Second Finding", "severity": "LOW", "updated_at": time.Now().UTC().Format(time.RFC3339)},
		},
		"next_cursor": "",
	})

	c := newTestConnector(t, h)
	ch, err := c.Sync(context.Background(), connector.SyncOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Sync: %v", err)
	}

	var records []connector.SyncRecord
	for rec := range ch {
		records = append(records, rec)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].ID != "rec-1" {
		t.Errorf("expected id 'rec-1', got %q", records[0].ID)
	}
	if records[0].Source != "test-rest" {
		t.Errorf("expected source 'test-rest', got %q", records[0].Source)
	}
}

func TestRESTConnector_Transform(t *testing.T) {
	h := harness.New(t)
	c := newTestConnector(t, h)

	result, err := c.Transform(context.Background(), connector.TransformRequest{
		RecordType: "finding",
		Raw: map[string]any{
			"id":       "vuln-99",
			"title":    "SQL Injection",
			"severity": "CRITICAL",
		},
	})
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if result.RecordType != "finding" {
		t.Errorf("expected RecordType 'finding', got %q", result.RecordType)
	}
	if result.Normalised["severity"] != "critical" {
		t.Errorf("expected normalised severity 'critical', got %v", result.Normalised["severity"])
	}
	if result.Normalised["external_id"] != "vuln-99" {
		t.Errorf("expected external_id 'vuln-99', got %v", result.Normalised["external_id"])
	}
}

func TestRESTConnector_Healthcheck(t *testing.T) {
	h := harness.New(t)
	h.AddResponse("/health", http.StatusOK, `{"status":"ok"}`)

	c := newTestConnector(t, h)
	status := c.Healthcheck(context.Background())
	if !status.Healthy {
		t.Errorf("expected Healthy=true, got message: %s", status.Message)
	}
	if status.Connector != "test-rest" {
		t.Errorf("expected connector name 'test-rest', got %q", status.Connector)
	}
}

func TestRESTConnector_New_MissingRequired(t *testing.T) {
	if _, err := rest.New(rest.Config{}); err == nil {
		t.Fatal("expected error for empty config")
	}
	if _, err := rest.New(rest.Config{Name: "x"}); err == nil {
		t.Fatal("expected error for missing BaseURL")
	}
	if _, err := rest.New(rest.Config{Name: "x", BaseURL: "http://x.com"}); err == nil {
		t.Fatal("expected error for missing Auth")
	}
}
