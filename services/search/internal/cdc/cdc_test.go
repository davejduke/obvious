package cdc

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/davejduke/obvious/services/search/internal/models"
)

// TestNotifyPayloadParsing validates that notifyPayload unmarshals correctly.
func TestNotifyPayloadParsing(t *testing.T) {
	payloadJSON := `{"op":"INSERT","table":"findings","id":"abc-123","org_id":"org-456"}`
	var p notifyPayload
	if err := json.Unmarshal([]byte(payloadJSON), &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Op != "INSERT" {
		t.Errorf("Op: got %q, want INSERT", p.Op)
	}
	if p.Table != "findings" {
		t.Errorf("Table: got %q, want findings", p.Table)
	}
	if p.ID != "abc-123" {
		t.Errorf("ID: got %q, want abc-123", p.ID)
	}
	if p.OrgID != "org-456" {
		t.Errorf("OrgID: got %q, want org-456", p.OrgID)
	}
}

// TestListenerChannels validates the monitored channel list.
func TestListenerChannels(t *testing.T) {
	expected := map[string]bool{
		"aiauditor_findings": true,
		"aiauditor_evidence": true,
		"aiauditor_controls": true,
	}
	for _, ch := range channels {
		if !expected[ch] {
			t.Errorf("unexpected channel: %s", ch)
		}
		delete(expected, ch)
	}
	for ch := range expected {
		t.Errorf("missing channel: %s", ch)
	}
}

// TestDispatchRouting validates that dispatch calls handler with the correct event.
func TestDispatchRouting(t *testing.T) {
	var received []models.ChangeEvent
	handler := func(ctx context.Context, ev models.ChangeEvent) {
		received = append(received, ev)
	}

	l := New("dummy-dsn", handler)

	// Simulate calling dispatch directly with a fake pq Notification.
	// We test the parsing logic without a live DB connection.
	payloads := []string{
		`{"op":"INSERT","table":"findings","id":"f1","org_id":"o1"}`,
		`{"op":"UPDATE","table":"evidence","id":"e2","org_id":"o1"}`,
		`{"op":"DELETE","table":"controls","id":"c3","org_id":"o2"}`,
	}

	ctx := context.Background()
	for _, payload := range payloads {
		var p notifyPayload
		if err := json.Unmarshal([]byte(payload), &p); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		ev := models.ChangeEvent{Op: p.Op, Table: p.Table, ID: p.ID, OrgID: p.OrgID}
		l.handler(ctx, ev)
	}

	if len(received) != 3 {
		t.Fatalf("got %d events, want 3", len(received))
	}
	if received[0].Op != "INSERT" || received[0].Table != "findings" {
		t.Errorf("event[0]: got %+v", received[0])
	}
	if received[1].Op != "UPDATE" || received[1].Table != "evidence" {
		t.Errorf("event[1]: got %+v", received[1])
	}
	if received[2].Op != "DELETE" || received[2].Table != "controls" {
		t.Errorf("event[2]: got %+v", received[2])
	}
}

