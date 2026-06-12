package ingester_test

import (
	"testing"
	"time"

	"github.com/davejduke/obvious/services/evidence/internal/ingester"
	"github.com/davejduke/obvious/services/evidence/internal/models"
)

const (
	testOrgID        = "550e8400-e29b-41d4-a716-446655440000"
	testEngagementID = "550e8400-e29b-41d4-a716-446655440001"
	testControlID    = "550e8400-e29b-41d4-a716-446655440002"
)

func validRequest() models.IngestRequest {
	return models.IngestRequest{
		OrgID:         testOrgID,
		EngagementID:  testEngagementID,
		ControlID:     testControlID,
		Title:         "Firewall Configuration Export",
		Description:   "Monthly firewall rule export for compliance audit",
		SourceType:    models.SourceConfigurationExport,
		ContentFormat: models.FormatJSON,
		Content:       `{"rules": [{"port": 443, "action": "allow"}]}`,
		Tags:          []string{"firewall", "network", "compliance"},
	}
}

func TestIngest_JSON_Valid(t *testing.T) {
	ing := ingester.New()
	req := validRequest()

	ev, err := ing.Ingest(&req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ev.Title != req.Title {
		t.Errorf("title mismatch: got %q, want %q", ev.Title, req.Title)
	}
	if ev.ContentFormat != models.FormatJSON {
		t.Errorf("content format mismatch: got %q", ev.ContentFormat)
	}
	if ev.ID.String() == "" {
		t.Error("expected non-empty ID")
	}
	if ev.CollectedAt.IsZero() {
		t.Error("expected non-zero collected_at")
	}
	if len(ev.ProvenanceChain) == 0 {
		t.Error("expected provenance chain to be populated on ingest")
	}
}

func TestIngest_CSV_Valid(t *testing.T) {
	ing := ingester.New()
	req := validRequest()
	req.ContentFormat = models.FormatCSV
	req.Content = "user,action,timestamp\nalice,login,2024-01-01T00:00:00Z\nbob,logout,2024-01-01T01:00:00Z"

	ev, err := ing.Ingest(&req)
	if err != nil {
		t.Fatalf("unexpected error for valid CSV: %v", err)
	}
	if ev.ContentFormat != models.FormatCSV {
		t.Errorf("content format: got %q", ev.ContentFormat)
	}
}

func TestIngest_Log_Valid(t *testing.T) {
	ing := ingester.New()
	req := validRequest()
	req.SourceType = models.SourceLogExport
	req.ContentFormat = models.FormatLog
	req.Content = "2024-01-01T10:00:00Z INFO User alice logged in from 192.168.1.1\n2024-01-01T10:01:00Z WARN Failed login attempt"

	ev, err := ing.Ingest(&req)
	if err != nil {
		t.Fatalf("unexpected error for valid log: %v", err)
	}
	if ev.ContentFormat != models.FormatLog {
		t.Errorf("content format: got %q", ev.ContentFormat)
	}
}

func TestIngest_JSON_Invalid(t *testing.T) {
	ing := ingester.New()
	req := validRequest()
	req.Content = `{invalid json`

	_, err := ing.Ingest(&req)
	if err == nil {
		t.Error("expected error for invalid JSON content")
	}
}

func TestIngest_CSV_Invalid(t *testing.T) {
	ing := ingester.New()
	req := validRequest()
	req.ContentFormat = models.FormatCSV
	req.Content = "col1,col2\n\"unclosed quote"

	_, err := ing.Ingest(&req)
	if err == nil {
		t.Error("expected error for invalid CSV content")
	}
}

func TestIngest_MissingRequiredFields(t *testing.T) {
	ing := ingester.New()

	tests := []struct {
		name string
		mutate func(*models.IngestRequest)
	}{
		{"missing org_id", func(r *models.IngestRequest) { r.OrgID = "" }},
		{"missing engagement_id", func(r *models.IngestRequest) { r.EngagementID = "" }},
		{"missing control_id", func(r *models.IngestRequest) { r.ControlID = "" }},
		{"missing title", func(r *models.IngestRequest) { r.Title = "" }},
		{"invalid source_type", func(r *models.IngestRequest) { r.SourceType = "unknown_source" }},
		{"invalid content_format", func(r *models.IngestRequest) { r.ContentFormat = "xml" }},
	}

	for _, tt := range tests {
		req := validRequest()
		tt.mutate(&req)
		_, err := ing.Ingest(&req)
		if err == nil {
			t.Errorf("%s: expected validation error, got none", tt.name)
		}
	}
}

func TestIngest_CollectedAt_UseProvided(t *testing.T) {
	ing := ingester.New()
	req := validRequest()
	specificTime := time.Date(2024, 3, 15, 10, 30, 0, 0, time.UTC)
	req.CollectedAt = &specificTime

	ev, err := ing.Ingest(&req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ev.CollectedAt.Equal(specificTime) {
		t.Errorf("expected collected_at %v, got %v", specificTime, ev.CollectedAt)
	}
}

func TestIngest_Tags_Normalised(t *testing.T) {
	ing := ingester.New()
	req := validRequest()
	req.Tags = []string{"Firewall", "FIREWALL", "network ", "Network", ""}

	ev, err := ing.Ingest(&req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expect deduplication and lowercasing: firewall, network
	if len(ev.Tags) != 2 {
		t.Errorf("expected 2 deduplicated tags, got %d: %v", len(ev.Tags), ev.Tags)
	}
}

func TestIngest_ProvenanceChain_Populated(t *testing.T) {
	ing := ingester.New()
	req := validRequest()

	ev, err := ing.Ingest(&req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(ev.ProvenanceChain) == 0 {
		t.Fatal("provenance chain must not be empty after ingest")
	}
	entry := ev.ProvenanceChain[0]
	if entry.Action != "ingested" {
		t.Errorf("expected action 'ingested', got %q", entry.Action)
	}
	if entry.Timestamp.IsZero() {
		t.Error("provenance entry timestamp should not be zero")
	}
}

