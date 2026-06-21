// Package delivery implements async webhook dispatch with HMAC-SHA256 signing,
// 3-attempt exponential backoff, and dead-letter logging.
//
// Architecture:
//
//	  Dispatch(orgID, eventType, data)
//	  ├─ fan out to all active subscriptions for that event type
//	  ├─ create a WebhookDelivery row per subscription (status=pending)
//	  └─ attempt HTTP POST immediately (within goroutine)
//	     ├─ success (2xx) → mark delivered
//	     └─ failure       → RecordAttemptFailure (schedules retry or dead-letters)
//
// A background poller (StartRetryWorker) drains pending rows whose
// next_retry_at has elapsed, enabling the 30 s / 5 m / 30 m schedule.
package delivery

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	svclogging "github.com/davejduke/obvious/shared/logging"
	"github.com/davejduke/obvious/services/webhooks/internal/model"
	"github.com/davejduke/obvious/services/webhooks/internal/signer"
	"github.com/davejduke/obvious/services/webhooks/internal/store"
)

const (
	// deliveryTimeout is the per-request HTTP deadline.
	deliveryTimeout = 10 * time.Second
	// retryPollInterval is how often the background worker polls for due retries.
	retryPollInterval = 15 * time.Second
	// retryBatchSize caps how many pending deliveries the worker processes per tick.
	retryBatchSize = 50
)

// Worker handles outbound webhook delivery.
type Worker struct {
	store  *store.Store
	client *http.Client
	logger *svclogging.Logger
}

// New returns a Worker with a default HTTP client.
func New(s *store.Store) *Worker {
	return &Worker{
		store:  s,
		client: &http.Client{Timeout: deliveryTimeout},
		logger: svclogging.New("webhooks-delivery"),
	}
}

// Dispatch fans out a platform event to all active subscriptions for the org
// that match the event type. Each delivery is created synchronously and
// dispatched in its own goroutine so the caller is never blocked.
func (w *Worker) Dispatch(
	ctx context.Context,
	orgID uuid.UUID,
	eventType string,
	data map[string]any,
) error {
	subs, err := w.store.ActiveSubscriptionsForEvent(ctx, orgID, eventType)
	if err != nil {
		return fmt.Errorf("delivery: list subscriptions: %w", err)
	}

	event := model.WebhookEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		OrgID:     orgID.String(),
		Timestamp: time.Now().UTC(),
		Data:      data,
	}

	payload := map[string]any{
		"id":        event.ID,
		"type":      event.Type,
		"org_id":    event.OrgID,
		"timestamp": event.Timestamp.Format(time.RFC3339Nano),
		"data":      event.Data,
	}

	w.logger.Info(ctx, "delivery.dispatch", map[string]any{
		"org_id":     orgID,
		"event_type": eventType,
		"fan_out":    len(subs),
	})

	for _, sub := range subs {
		// Create delivery record in the current goroutine so the caller can
		// confirm the delivery was enqueued before the request returns.
		del, err := w.store.CreateDelivery(ctx, sub.ID, orgID, eventType, payload)
		if err != nil {
			w.logger.Error(ctx, "delivery.create_record_failed", map[string]any{
				"subscription_id": sub.ID,
				"error":           err.Error(),
			})
			continue
		}

		// Capture loop vars for the goroutine.
		subURL, subSecret := sub.URL, sub.Secret
		// Attempt delivery in a goroutine — failure is recorded, never panics.
		go w.attempt(context.Background(), del, subURL, subSecret)
	}
	return nil
}

// attempt makes a single HTTP POST to the subscriber URL.
// On failure it calls RecordAttemptFailure which either schedules a retry or
// moves the delivery to failed (dead-letter).
func (w *Worker) attempt(
	ctx context.Context,
	del *model.WebhookDelivery,
	endpointURL, secret string,
) {
	payloadBytes, err := json.Marshal(del.Payload)
	if err != nil {
		w.recordFailure(ctx, del, nil, fmt.Sprintf("marshal payload: %v", err))
		return
	}

	sig := signer.Sign(secret, payloadBytes)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpointURL, bytes.NewReader(payloadBytes))
	if err != nil {
		w.recordFailure(ctx, del, nil, fmt.Sprintf("build request: %v", err))
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(signer.SignatureHeader, sig)
	req.Header.Set("X-AIAUDITOR-Event", del.EventType)
	req.Header.Set("X-AIAUDITOR-Delivery", del.ID.String())

	resp, err := w.client.Do(req)
	if err != nil {
		w.recordFailure(ctx, del, nil, fmt.Sprintf("http error: %v", err))
		return
	}
	defer resp.Body.Close()
	// Drain body to allow connection reuse.
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		w.logger.Info(ctx, "delivery.success", map[string]any{
			"delivery_id":     del.ID,
			"subscription_id": del.SubscriptionID,
			"status":          resp.StatusCode,
		})
		if err := w.store.MarkDelivered(ctx, del.ID, resp.StatusCode); err != nil {
			w.logger.Error(ctx, "delivery.mark_delivered_failed", map[string]any{
				"delivery_id": del.ID,
				"error":        err.Error(),
			})
		}
		return
	}

	// Non-2xx — treat as failure and schedule retry.
	w.recordFailure(ctx, del, &resp.StatusCode,
		fmt.Sprintf("non-2xx response: %d", resp.StatusCode))
}

func (w *Worker) recordFailure(
	ctx context.Context,
	del *model.WebhookDelivery,
	respStatus *int,
	errMsg string,
) {
	newAttemptCount := del.AttemptCount + 1
	w.logger.Warn(ctx, "delivery.failed", map[string]any{
		"delivery_id":   del.ID,
		"attempt":       newAttemptCount,
		"max_attempts":  model.MaxAttempts,
		"error":         errMsg,
		"dead_lettered": newAttemptCount >= model.MaxAttempts,
	})
	if err := w.store.RecordAttemptFailure(ctx, del.ID, newAttemptCount, respStatus, errMsg); err != nil {
		w.logger.Error(ctx, "delivery.record_failure_error", map[string]any{
			"delivery_id": del.ID,
			"error":        err.Error(),
		})
	}
}

// StartRetryWorker launches a background goroutine that periodically drains
// pending deliveries whose next_retry_at has elapsed.
// Call the returned stop function to shut it down cleanly.
func (w *Worker) StartRetryWorker(ctx context.Context) (stop func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)
		ticker := time.NewTicker(retryPollInterval)
		defer ticker.Stop()

		w.logger.Info(ctx, "delivery.retry_worker.started", nil)
		for {
			select {
			case <-ctx.Done():
				w.logger.Info(ctx, "delivery.retry_worker.stopped", nil)
				return
			case <-ticker.C:
				w.runRetryBatch(ctx)
			}
		}
	}()

	return func() {
		cancel()
		<-done
	}
}

// runRetryBatch fetches up to retryBatchSize due deliveries and re-attempts them.
func (w *Worker) runRetryBatch(ctx context.Context) {
	deliveries, err := w.store.PendingDeliveries(ctx, retryBatchSize)
	if err != nil {
		w.logger.Error(ctx, "delivery.retry_batch.query_failed", map[string]any{"error": err.Error()})
		return
	}
	if len(deliveries) == 0 {
		return
	}

	w.logger.Info(ctx, "delivery.retry_batch", map[string]any{"count": len(deliveries)})

	for _, del := range deliveries {
		sub, secret, err := w.store.GetSubscriptionWithSecret(ctx, del.SubscriptionID)
		if err != nil {
			w.logger.Error(ctx, "delivery.retry.lookup_failed", map[string]any{
				"subscription_id": del.SubscriptionID,
				"delivery_id":     del.ID,
				"error":           err.Error(),
			})
			continue
		}
		if sub == nil || secret == "" {
			w.logger.Error(ctx, "delivery.retry.subscription_not_found", map[string]any{
				"subscription_id": del.SubscriptionID,
				"delivery_id":     del.ID,
			})
			continue
		}
		// Capture loop vars for the goroutine.
		subURL, subSecret := sub.URL, secret
		dCopy := del
		go w.attempt(ctx, dCopy, subURL, subSecret)
	}
}

