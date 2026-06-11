// Package store provides database access for the audit-trail service.
// All writes go through Append — no UPDATE or DELETE paths exist.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/davejduke/obvious/services/audit-trail/internal/model"
)

// Store wraps a PostgreSQL connection pool and exposes the append-only
// operations needed by the audit-trail service.
type Store struct {
	db *pgxpool.Pool
}

// New returns a Store backed by the supplied pool.
func New(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// Append writes a single audit event inside a serialisable transaction.
// The transaction reads the previous event's hash (with a row-level lock),
// computes the new hash, and inserts the row. This guarantees that
// concurrent appends do not create forks in the chain.
//
// The caller is responsible for computing the hash BEFORE calling Append;
// the hash and previousHash are passed in so the store layer remains
// crypto-agnostic.
func (s *Store) Append(
	ctx context.Context,
	req *model.AppendRequest,
	hash string,
	previousHash string,
	occurredAt time.Time,
) (*model.AuditEvent, error) {
	var event model.AuditEvent

	err := pgx.BeginTxFunc(ctx, s.db, pgx.TxOptions{IsoLevel: pgx.Serializable}, func(tx pgx.Tx) error {
		changesJSON, err := json.Marshal(req.Changes)
		if err != nil {
			return fmt.Errorf("store: marshal changes: %w", err)
		}
		contextJSON, err := json.Marshal(req.Context)
		if err != nil {
			return fmt.Errorf("store: marshal context: %w", err)
		}

		row := tx.QueryRow(ctx, `
			INSERT INTO audit_events (
				org_id, actor_id, actor_email,
				action, resource_type, resource_id,
				engagement_id,
				changes, context,
				ip_address, user_agent,
				previous_hash, event_hash,
				occurred_at
			) VALUES (
				$1,  $2,  $3,
				$4,  $5,  $6,
				$7,
				$8,  $9,
				$10, $11,
				$12, $13,
				$14
			)
			RETURNING id, event_id, occurred_at`,
			req.OrgID,
			req.ActorID,
			req.ActorEmail,
			req.Action,
			req.ResourceType,
			req.ResourceID,
			req.EngagementID,
			changesJSON,
			contextJSON,
			nilIfEmpty(req.IPAddress),
			nilIfEmpty(req.UserAgent),
			previousHash,
			hash,
			occurredAt,
		)

		var eventID uuid.UUID
		if err := row.Scan(&event.ID, &eventID, &event.OccurredAt); err != nil {
			return fmt.Errorf("store: insert audit_event: %w", err)
		}
		event.EventID = eventID
		event.OrgID = req.OrgID
		event.ActorID = req.ActorID
		event.ActorEmail = req.ActorEmail
		event.Action = req.Action
		event.EventType = req.EventType
		event.ResourceType = req.ResourceType
		event.ResourceID = req.ResourceID
		event.EngagementID = req.EngagementID
		event.Changes = req.Changes
		event.Context = req.Context
		event.IPAddress = req.IPAddress
		event.UserAgent = req.UserAgent
		event.PreviousHash = previousHash
		event.EventHash = hash
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &event, nil
}

// GetLastHash returns the event_hash of the most recently inserted event for
// the given org. Returns GenesisHash ("") if no events exist yet.
func (s *Store) GetLastHash(ctx context.Context, orgID uuid.UUID) (string, error) {
	var hash string
	err := s.db.QueryRow(ctx,
		`SELECT event_hash FROM audit_events
		 WHERE org_id = $1
		 ORDER BY id DESC
		 LIMIT 1`,
		orgID,
	).Scan(&hash)
	if err == pgx.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("store: get last hash: %w", err)
	}
	return hash, nil
}

// QueryByEntity returns events for a specific resource (entity_type + entity_id).
func (s *Store) QueryByEntity(
	ctx context.Context,
	orgID uuid.UUID,
	entityType string,
	entityID uuid.UUID,
	limit, offset int,
) ([]*model.AuditEvent, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, event_id, org_id, actor_id, actor_email,
		        action, resource_type, resource_id,
		        engagement_id,
		        changes, context, ip_address, user_agent,
		        previous_hash, event_hash, occurred_at
		   FROM audit_events
		  WHERE org_id = $1
		    AND resource_type = $2
		    AND resource_id   = $3
		  ORDER BY id ASC
		  LIMIT $4 OFFSET $5`,
		orgID, entityType, entityID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query by entity: %w", err)
	}
	return scanEvents(rows)
}

// QueryByUser returns events performed by a specific actor in a time window.
func (s *Store) QueryByUser(
	ctx context.Context,
	orgID uuid.UUID,
	actorID uuid.UUID,
	start, end time.Time,
	limit, offset int,
) ([]*model.AuditEvent, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, event_id, org_id, actor_id, actor_email,
		        action, resource_type, resource_id,
		        engagement_id,
		        changes, context, ip_address, user_agent,
		        previous_hash, event_hash, occurred_at
		   FROM audit_events
		  WHERE org_id   = $1
		    AND actor_id = $2
		    AND occurred_at >= $3
		    AND occurred_at <= $4
		  ORDER BY id ASC
		  LIMIT $5 OFFSET $6`,
		orgID, actorID, start, end, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("store: query by user: %w", err)
	}
	return scanEvents(rows)
}

// MetaAudit returns all events for an engagement, ordered chronologically.
func (s *Store) MetaAudit(
	ctx context.Context,
	orgID uuid.UUID,
	engagementID uuid.UUID,
	limit, offset int,
) ([]*model.AuditEvent, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, event_id, org_id, actor_id, actor_email,
		        action, resource_type, resource_id,
		        engagement_id,
		        changes, context, ip_address, user_agent,
		        previous_hash, event_hash, occurred_at
		   FROM audit_events
		  WHERE org_id        = $1
		    AND engagement_id = $2
		  ORDER BY id ASC
		  LIMIT $3 OFFSET $4`,
		orgID, engagementID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("store: meta audit: %w", err)
	}
	return scanEvents(rows)
}

// AllEventsForOrg streams all events for an org ordered by id ASC
// (used during chain verification).
func (s *Store) AllEventsForOrg(
	ctx context.Context,
	orgID uuid.UUID,
) (pgx.Rows, error) {
	return s.db.Query(ctx,
		`SELECT id, event_id, org_id, actor_id, actor_email,
		        action, resource_type, resource_id,
		        engagement_id,
		        changes, context, ip_address, user_agent,
		        previous_hash, event_hash, occurred_at
		   FROM audit_events
		  WHERE org_id = $1
		  ORDER BY id ASC`,
		orgID,
	)
}

// ScanEvent scans a single pgx.Row into an AuditEvent.
func ScanEvent(rows pgx.Rows, ev *model.AuditEvent) error {
	var changesRaw, contextRaw []byte
	var actorID *uuid.UUID
	var resourceID *uuid.UUID
	var engagementID *uuid.UUID
	var ipAddr *string
	var userAgent *string

	err := rows.Scan(
		&ev.ID, &ev.EventID, &ev.OrgID, &actorID, &ev.ActorEmail,
		&ev.Action, &ev.ResourceType, &resourceID,
		&engagementID,
		&changesRaw, &contextRaw, &ipAddr, &userAgent,
		&ev.PreviousHash, &ev.EventHash, &ev.OccurredAt,
	)
	if err != nil {
		return err
	}
	ev.ActorID = actorID
	ev.ResourceID = resourceID
	ev.EngagementID = engagementID
	if ipAddr != nil {
		ev.IPAddress = *ipAddr
	}
	if userAgent != nil {
		ev.UserAgent = *userAgent
	}
	if len(changesRaw) > 0 && string(changesRaw) != "null" {
		if err := json.Unmarshal(changesRaw, &ev.Changes); err != nil {
			return fmt.Errorf("store: unmarshal changes: %w", err)
		}
	}
	if len(contextRaw) > 0 && string(contextRaw) != "null" {
		if err := json.Unmarshal(contextRaw, &ev.Context); err != nil {
			return fmt.Errorf("store: unmarshal context: %w", err)
		}
	}
	return nil
}

func scanEvents(rows pgx.Rows) ([]*model.AuditEvent, error) {
	defer rows.Close()
	var events []*model.AuditEvent
	for rows.Next() {
		ev := &model.AuditEvent{}
		if err := ScanEvent(rows, ev); err != nil {
			return nil, err
		}
		events = append(events, ev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("store: scan rows: %w", err)
	}
	return events, nil
}

// nilIfEmpty converts an empty string to nil for nullable DB columns.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

