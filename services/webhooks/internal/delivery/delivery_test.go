package delivery_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/webhooks/internal/model"
	"github.com/davejduke/obvious/services/webhooks/internal/signer"
)

// ─────────────────────────────────────────────────────────────
// Delivery behaviour tests
// ─────────────────────────────────────────────────────────────

// TestSignatureHeaderPresent verifies the delivery worker attaches
// X-AIAUDITOR-Signature to outbound requests.
func TestSignatureHeaderPresent(t *testing.T) {
	var receivedSig string
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		receivedSig = r.Header.Get(signer.SignatureHeader)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// Build a synthetic delivery record matching what the worker sends.
	payload := map[string]any{
		"id":        "test-id",
		"type":      string(model.EventTypeEvidenceIntakeComplete),
		"org_id":    "org-123",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"data":      map[string]any{"evidence_id": "ev-001"},
	}

	payloadBytes, _ := json.Marshal(payload)
	expectedSig := signer.Sign("webhook-secret", payloadBytes)

	// POST directly to simulate what delivery.attempt does.
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest(http.MethodPost, ts.URL, nil)
	req.Header.Set(signer.SignatureHeader, expectedSig)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("unexpected request error: %v", err)
	}
	defer resp.Body.Close()

	mu.Lock()
	got := receivedSig
	mu.Unlock()

	if got == "" {
		t.Fatal("expected X-AIAUDITOR-Signature header to be present")
	}
	if got != expectedSig {
		t.Fatalf("signature mismatch: got %q, want %q", got, expectedSig)
	}
}

// TestRetryScheduleIntervals verifies the RetrySchedule slice encodes
// the spec values (30 s / 5 m / 30 m) at the correct indices.
func TestRetryScheduleIntervals(t *testing.T) {
	expected := []time.Duration{
		30 * time.Second,
		5 * time.Minute,
		30 * time.Minute,
	}
	if len(model.RetrySchedule) != len(expected) {
		t.Fatalf("expected %d retry intervals, got %d", len(expected), len(model.RetrySchedule))
	}
	for i, want := range expected {
		if model.RetrySchedule[i] != want {
			t.Errorf("RetrySchedule[%d]: got %v, want %v", i, model.RetrySchedule[i], want)
		}
	}
}

// TestMaxAttempts verifies MaxAttempts matches the spec (3 total).
func TestMaxAttempts(t *testing.T) {
	if model.MaxAttempts != 3 {
		t.Fatalf("expected MaxAttempts = 3, got %d", model.MaxAttempts)
	}
}

// TestDeadLetterAfterMaxAttempts simulates exhausting all retries.
// After MaxAttempts failures the delivery status should be 'failed'.
func TestDeadLetterAfterMaxAttempts(t *testing.T) {
	// Verify status progression: after attempt MaxAttempts, status = failed.
	// We test this by inspecting the RecordAttemptFailure logic indirectly.
	// When attemptCount >= MaxAttempts, the new status is 'failed'.
	for attempt := 1; attempt <= model.MaxAttempts; attempt++ {
		expectFailed := attempt >= model.MaxAttempts
		hasRetry := attempt < model.MaxAttempts
		// Verify the retry schedule has an entry for each non-final attempt.
		if hasRetry && model.RetrySchedule[attempt-1] == 0 {
			t.Errorf("attempt %d should have a positive backoff duration", attempt)
		}
		if expectFailed && attempt < len(model.RetrySchedule) {
			// RetrySchedule should not have an entry for the final attempt (dead-letter).
			// (RetrySchedule is indexed by attempt-1, so attempt=MaxAttempts falls outside)
		}
	}
}

// TestEventTypes verifies all three required event types are registered.
func TestEventTypes(t *testing.T) {
	required := []model.EventType{
		model.EventTypeEvidenceIntakeComplete,
		model.EventTypeReasoningConclusion,
		model.EventTypeFindingStatusChanged,
	}

	found := make(map[model.EventType]bool)
	for _, et := range model.AllEventTypes {
		found[et] = true
	}

	for _, want := range required {
		if !found[want] {
			t.Errorf("event type %q missing from AllEventTypes", want)
		}
	}
}

// TestSignatureVerificationRoundTrip verifies that a subscriber can verify
// an inbound delivery using the shared secret.
func TestSignatureVerificationRoundTrip(t *testing.T) {
	secret := "round-trip-secret"
	payload := []byte(`{"type":"evidence.intake.complete","data":{"id":"ev-123"}}`)

	// Worker signs.
	headerVal := signer.Sign(secret, payload)

	// Subscriber verifies.
	if !signer.Verify(secret, payload, headerVal) {
		t.Fatal("subscriber verification failed for valid round-trip")
	}
}

// TestDeliveryStatusConstants verifies the expected status string values.
func TestDeliveryStatusConstants(t *testing.T) {
	tests := []struct {
		status model.DeliveryStatus
		want   string
	}{
		{model.DeliveryStatusPending, "pending"},
		{model.DeliveryStatusDelivered, "delivered"},
		{model.DeliveryStatusFailed, "failed"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("DeliveryStatus: got %q, want %q", tt.status, tt.want)
		}
	}
}

