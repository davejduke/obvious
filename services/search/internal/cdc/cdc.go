// Package cdc implements the Change Data Capture pipeline for AIAUDITOR search.
// It uses PostgreSQL LISTEN/NOTIFY (via lib/pq) to receive change events from
// the database triggers defined in migration 004_search_notifications.sql.
//
// Each notification payload has the form:
//
//	{"op":"INSERT","table":"findings","id":"<uuid>","org_id":"<uuid>"}
//
// The pipeline fans these events to the indexer which re-fetches the row and
// updates the OpenSearch document. Lag is typically <1s (NOTIFY is synchronous
// with the committing transaction).
package cdc

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/davejduke/obvious/services/search/internal/models"
	"github.com/lib/pq"
)

// channels we LISTEN on — one per monitored table.
var channels = []string{
	"aiauditor_findings",
	"aiauditor_evidence",
	"aiauditor_controls",
}

// notifyPayload is the JSON struct sent by the PostgreSQL trigger.
type notifyPayload struct {
	Op    string `json:"op"`
	Table string `json:"table"`
	ID    string `json:"id"`
	OrgID string `json:"org_id"`
}

// EventHandler is called for each change event received from PostgreSQL.
type EventHandler func(ctx context.Context, ev models.ChangeEvent)

// Listener wraps a pq listener and routes NOTIFY payloads to an EventHandler.
type Listener struct {
	dsn     string
	handler EventHandler
	pql     *pq.Listener
}

// New creates a Listener that will call handler for each change event.
func New(dsn string, handler EventHandler) *Listener {
	return &Listener{
		dsn:     dsn,
		handler: handler,
	}
}

// Start opens a PostgreSQL listener connection and begins listening.
// It blocks until ctx is cancelled.
func (l *Listener) Start(ctx context.Context) error {
	reportProblem := func(ev pq.ListenerEventType, err error) {
		if err != nil {
			log.Printf("[cdc] listener error: %v", err)
		}
	}

	l.pql = pq.NewListener(l.dsn, 10*time.Second, 60*time.Second, reportProblem)

	for _, ch := range channels {
		if err := l.pql.Listen(ch); err != nil {
			return fmt.Errorf("cdc: LISTEN %s: %w", ch, err)
		}
		log.Printf("[cdc] LISTEN %s", ch)
	}

	log.Printf("[cdc] CDC pipeline started; listening on %v", channels)
	return l.loop(ctx)
}

// Close shuts down the listener connection.
func (l *Listener) Close() {
	if l.pql != nil {
		_ = l.pql.Close()
	}
}

func (l *Listener) loop(ctx context.Context) error {
	pingTicker := time.NewTicker(90 * time.Second)
	defer pingTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case n, ok := <-l.pql.Notify:
			if !ok {
				return fmt.Errorf("cdc: notification channel closed")
			}
			if n == nil {
				continue // reconnect ping from pq
			}
			l.dispatch(ctx, n)

		case <-pingTicker.C:
			go func() {
				if err := l.pql.Ping(); err != nil {
					log.Printf("[cdc] ping error: %v", err)
				}
			}()
		}
	}
}

func (l *Listener) dispatch(ctx context.Context, n *pq.Notification) {
	var p notifyPayload
	if err := json.Unmarshal([]byte(n.Extra), &p); err != nil {
		log.Printf("[cdc] malformed payload on %s: %v", n.Channel, err)
		return
	}
	ev := models.ChangeEvent{
		Op:    p.Op,
		Table: p.Table,
		ID:    p.ID,
		OrgID: p.OrgID,
	}
	l.handler(ctx, ev)
}

