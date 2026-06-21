// Package worker implements the background delivery worker.
// It polls for due deliveries every second, calls the target URL, signs the
// payload with HMAC-SHA256, and records success/failure with exponential backoff.
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	svclogging "github.com/davejduke/obvious/shared/logging"
	"github.com/davejduke/obvious/services/webhook/internal/domain"
	"github.com/davejduke/obvious/services/webhook/internal/signer"
	"github.com/davejduke/obvious/services/webhook/internal/store"
)

const (
	pollInterval = 1 * time.Second
	batchSize    = 50
	deliveryTimeout = 10 * time.Second
)

// Queue abstracts the delivery-queue operations the worker needs.
type Queue interface {
	DuePendingDeliveries(ctx context.Context, limit int) ([]*domain.Delivery, error)
	GetEndpoint(ctx context.Context, id, orgID string) (*domain.Endpoint, error)
	MarkDelivered(ctx context.Context, id string, code int, body string) error
	MarkFailed(ctx context.Context, id string, code int, body string, attempts int) error
}

// Worker polls the delivery queue and attempts HTTP delivery.
type Worker struct {
	queue  Queue
	client *http.Client
	logger *svclogging.Logger
}

// New returns a Worker.
func New(q Queue) *Worker {
	return &Worker{
		queue:  q,
		client: &http.Client{Timeout: deliveryTimeout},
		logger: svclogging.New("webhook-worker"),
	}
}

// Run starts the polling loop; it blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processBatch(ctx)
		}
	}
}

func (w *Worker) processBatch(ctx context.Context) {
	deliveries, err := w.queue.DuePendingDeliveries(ctx, batchSize)
	if err != nil {
		w.logger.Error(ctx, "worker.query_failed", map[string]any{"error": err.Error()})
		return
	}
	for _, d := range deliveries {
		w.deliver(ctx, d)
	}
}

func (w *Worker) deliver(ctx context.Context, d *domain.Delivery) {
	// Look up the endpoint to get URL and secret.
	// orgID is embedded in the endpoint record; we use an empty orgID to allow
	// the worker to fetch endpoints across orgs (internal worker bypass).
	ep, err := w.fetchEndpoint(ctx, d.EndpointID)
	if err != nil || ep == nil {
		w.logger.Error(ctx, "worker.endpoint_missing",
			map[string]any{"delivery_id": d.ID, "endpoint_id": d.EndpointID})
		_ = w.queue.MarkFailed(ctx, d.ID, 0, "endpoint not found", d.Attempts)
		return
	}

	now := time.Now().UTC()
	sig := signer.Sign(ep.Secret, d.Payload, now)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(d.Payload))
	if err != nil {
		w.logger.Error(ctx, "worker.build_request_failed",
			map[string]any{"delivery_id": d.ID, "error": err.Error()})
		_ = w.queue.MarkFailed(ctx, d.ID, 0, err.Error(), d.Attempts)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(signer.SignatureHeader, sig)
	req.Header.Set(signer.TimestampHeader, fmt.Sprintf("%d", now.Unix()))
	req.Header.Set("X-Webhook-Event", d.EventType)
	req.Header.Set("X-Delivery-ID", d.ID)

	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Warn(ctx, "worker.http_error",
			map[string]any{"delivery_id": d.ID, "error": err.Error(), "attempts": d.Attempts + 1})
		_ = w.queue.MarkFailed(ctx, d.ID, 0, err.Error(), d.Attempts)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.logger.Info(ctx, "worker.delivered",
			map[string]any{"delivery_id": d.ID, "status": resp.StatusCode})
		_ = w.queue.MarkDelivered(ctx, d.ID, resp.StatusCode, string(body))
	} else {
		w.logger.Warn(ctx, "worker.delivery_failed",
			map[string]any{"delivery_id": d.ID, "status": resp.StatusCode, "attempts": d.Attempts + 1})
		_ = w.queue.MarkFailed(ctx, d.ID, resp.StatusCode, string(body), d.Attempts)
	}
}

// fetchEndpoint fetches an endpoint by ID from the store using a background context.
// The worker needs cross-org access so it uses empty orgID with a raw query.
func (w *Worker) fetchEndpoint(ctx context.Context, endpointID string) (*domain.Endpoint, error) {
	if s, ok := w.queue.(*store.Store); ok {
		// Use the store's GetEndpoint with an empty orgID.
		// The store's GetEndpoint enforces org isolation, but since this is an
		// internal worker we fetch by ID only via a separate query.
		_ = s // suppress unused var
	}
	// For the worker, we fetch by ID using the queue interface;
	// the orgID filter is intentionally omitted for internal worker access.
	// In production, endpoint secrets would be org-scoped but the worker runs
	// in a trusted context.
	_ = endpointID // prevent unused var lint

	// Use the queue directly if it exposes GetEndpointByID.
	if q, ok := w.queue.(interface {
		GetEndpointByID(ctx context.Context, id string) (*domain.Endpoint, error)
	}); ok {
		return q.GetEndpointByID(ctx, endpointID)
	}
	// Fallback: use GetEndpoint with empty orgID (allowed for workers).
	return w.queue.GetEndpoint(ctx, endpointID, "")
}

// Envelope wraps a webhook event payload for delivery.
type Envelope struct {
	ID        string          `json:"id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	Timestamp time.Time       `json:"timestamp"`
}

