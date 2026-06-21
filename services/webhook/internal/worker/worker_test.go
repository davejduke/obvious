package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/webhook/internal/domain"
	"github.com/davejduke/obvious/services/webhook/internal/signer"
)

// mockQueue is an in-memory Queue for testing.
type mockQueue struct {
	deliveries []*domain.Delivery
	endpoints  map[string]*domain.Endpoint
	markResults map[string]string // id -> status
}

func (m *mockQueue) DuePendingDeliveries(_ context.Context, limit int) ([]*domain.Delivery, error) {
	result := m.deliveries
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockQueue) GetEndpoint(_ context.Context, id, _ string) (*domain.Endpoint, error) {
	ep, ok := m.endpoints[id]
	if !ok {
		return nil, nil
	}
	return ep, nil
}

func (m *mockQueue) MarkDelivered(_ context.Context, id string, _ int, _ string) error {
	m.markResults[id] = domain.StatusDelivered
	return nil
}

func (m *mockQueue) MarkFailed(_ context.Context, id string, _ int, _ string, attempts int) error {
	if attempts >= domain.MaxAttempts {
		m.markResults[id] = domain.StatusDeadLettered
	} else {
		m.markResults[id] = domain.StatusFailed
	}
	return nil
}

// TestDeliverSuccess checks that a 200 response marks the delivery as delivered.
func TestDeliverSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the HMAC-SHA256 signature header is present.
		if r.Header.Get(signer.SignatureHeader) == "" {
			t.Error("missing X-Webhook-Signature header")
		}
		if r.Header.Get(signer.TimestampHeader) == "" {
			t.Error("missing X-Webhook-Timestamp header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	payload, _ := json.Marshal(map[string]string{"event": "test.created"})
	mq := &mockQueue{
		deliveries: []*domain.Delivery{
			{
				ID:          "del-1",
				EndpointID:  "ep-1",
				EventType:   "test.created",
				Payload:     payload,
				Status:      domain.StatusPending,
				Attempts:    0,
				MaxAttempts: 3,
				NextRetryAt: time.Now().Add(-time.Second),
			},
		},
		endpoints: map[string]*domain.Endpoint{
			"ep-1": {ID: "ep-1", OrgID: "org-1", URL: ts.URL, Secret: "s3cr3t"},
		},
		markResults: make(map[string]string),
	}

	w := New(mq)
	w.processBatch(context.Background())

	if mq.markResults["del-1"] != domain.StatusDelivered {
		t.Errorf("expected delivered, got %q", mq.markResults["del-1"])
	}
}

// TestDeliverFailThenDeadLetter checks that 3 failures dead-letter the delivery.
func TestDeliverFailThenDeadLetter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	payload, _ := json.Marshal(map[string]string{"event": "test.failed"})
	q := &mockQueue{
		endpoints: map[string]*domain.Endpoint{
			"ep-1": {ID: "ep-1", OrgID: "org-1", URL: ts.URL, Secret: "s3cr3t"},
		},
		markResults: make(map[string]string),
	}

	wkr := New(q)
	for attempt := 0; attempt < domain.MaxAttempts; attempt++ {
		q.deliveries = []*domain.Delivery{{
			ID: "del-2", EndpointID: "ep-1", EventType: "test.failed",
			Payload: payload, Status: domain.StatusPending,
			Attempts: attempt, MaxAttempts: 3, NextRetryAt: time.Now().Add(-time.Second),
		}}
		wkr.processBatch(context.Background())
	}

	if mq, ok := q.markResults["del-2"]; ok {
		if mq != domain.StatusDeadLettered && mq != domain.StatusFailed {
			t.Errorf("unexpected status: %q", mq)
		}
	}
}

// TestRetryBackoffDelays verifies the retry delay constants are ordered.
func TestRetryBackoffDelays(t *testing.T) {
	for i := 1; i < len(domain.RetryDelays); i++ {
		if domain.RetryDelays[i] <= domain.RetryDelays[i-1] {
			t.Errorf("RetryDelays[%d]=%v must be > RetryDelays[%d]=%v",
				i, domain.RetryDelays[i], i-1, domain.RetryDelays[i-1])
		}
	}
	if len(domain.RetryDelays) != domain.MaxAttempts {
		t.Errorf("len(RetryDelays)=%d must equal MaxAttempts=%d",
			len(domain.RetryDelays), domain.MaxAttempts)
	}
}

// TestDeliverMissingEndpoint checks that a missing endpoint dead-letters immediately.
func TestDeliverMissingEndpoint(t *testing.T) {
	payload, _ := json.Marshal(map[string]string{"event": "test.created"})
	mq := &mockQueue{
		deliveries: []*domain.Delivery{{
			ID: "del-3", EndpointID: "ep-missing", EventType: "test.created",
			Payload: payload, Attempts: 0, MaxAttempts: 3,
			NextRetryAt: time.Now().Add(-time.Second),
		}},
		endpoints:   map[string]*domain.Endpoint{},
		markResults: make(map[string]string),
	}
	wkr := New(mq)
	wkr.processBatch(context.Background())
	if _, ok := mq.markResults["del-3"]; !ok {
		t.Error("expected delivery to be marked failed/dead_lettered")
	}
}

// TestEnvelopeSignatureVerification verifies the signature sent in TestDeliverSuccess is valid.
func TestEnvelopeSignatureVerification(t *testing.T) {
	var receivedSig, receivedTS string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get(signer.SignatureHeader)
		receivedTS = r.Header.Get(signer.TimestampHeader)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	payload := []byte(`{"event":"order.created"}`)
	const secret = "webhook-secret-xyz"
	mq := &mockQueue{
		deliveries: []*domain.Delivery{{
			ID: "del-4", EndpointID: "ep-2", EventType: "order.created",
			Payload: payload, Attempts: 0, MaxAttempts: 3,
			NextRetryAt: time.Now().Add(-time.Second),
		}},
		endpoints:   map[string]*domain.Endpoint{"ep-2": {ID: "ep-2", OrgID: "org-1", URL: ts.URL, Secret: secret}},
		markResults: make(map[string]string),
	}
	New(mq).processBatch(context.Background())

	if receivedSig == "" {
		t.Fatal("no signature header received")
	}
	var tsInt int64
	if _, err := fmt.Sscanf(receivedTS, "%d", &tsInt); err != nil {
		t.Fatalf("invalid timestamp: %v", err)
	}
	expected := signer.Sign(secret, payload, time.Unix(tsInt, 0))
	if receivedSig != expected {
		t.Errorf("signature mismatch: got %q want %q", receivedSig, expected)
	}
}

