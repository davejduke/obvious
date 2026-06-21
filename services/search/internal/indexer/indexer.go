// Package indexer manages the OpenSearch indices and document indexing for AIAUDITOR.
// Index names:
//
//	aiauditor-findings  — finding documents
//	aiauditor-evidence  — evidence documents
//	aiauditor-controls  — control objective documents
package indexer

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/davejduke/obvious/services/search/internal/models"
	"github.com/davejduke/obvious/services/search/internal/opensearch"
	_ "github.com/lib/pq" // PostgreSQL driver
)

const (
	IndexFindings = "aiauditor-findings"
	IndexEvidence = "aiauditor-evidence"
	IndexControls = "aiauditor-controls"
)

// Indexer fetches rows from PostgreSQL and indexes them into OpenSearch.
type Indexer struct {
	db *sql.DB
	os *opensearch.Client
}

// New creates an Indexer wired to the given DB and OpenSearch client.
func New(db *sql.DB, osClient *opensearch.Client) *Indexer {
	return &Indexer{db: db, os: osClient}
}

// EnsureIndices creates all three indices with their mappings if they do not exist.
func (ix *Indexer) EnsureIndices(ctx context.Context) error {
	for _, m := range []struct {
		name    string
		mapping opensearch.IndexMapping
	}{
		{IndexFindings, findingsMapping()},
		{IndexEvidence, evidenceMapping()},
		{IndexControls, controlsMapping()},
	} {
		if err := ix.os.EnsureIndex(ctx, m.name, m.mapping); err != nil {
			return fmt.Errorf("indexer: ensure index %s: %w", m.name, err)
		}
	}
	return nil
}

// IndexFinding fetches a finding row from PostgreSQL and indexes it.
func (ix *Indexer) IndexFinding(ctx context.Context, id string) error {
	row := ix.db.QueryRowContext(ctx, `
		SELECT id, org_id, engagement_id, finding_ref, title, description,
		       COALESCE(root_cause,''), severity, status, tags, updated_at
		FROM findings WHERE id = $1`, id)

	var doc models.FindingDoc
	var tags []byte
	if err := row.Scan(
		&doc.ID, &doc.OrgID, &doc.EngagementID, &doc.FindingRef,
		&doc.Title, &doc.Description, &doc.RootCause,
		&doc.Severity, &doc.Status, &tags, &doc.UpdatedAt,
	); err == sql.ErrNoRows {
		return nil // row deleted before we could index it; skip
	} else if err != nil {
		return fmt.Errorf("indexer: fetch finding %s: %w", id, err)
	}
	doc.Tags = parseTextArray(tags)
	return ix.os.IndexDoc(ctx, IndexFindings, id, doc)
}

// DeleteFinding removes a finding document from the index.
func (ix *Indexer) DeleteFinding(ctx context.Context, id string) error {
	return ix.os.DeleteDoc(ctx, IndexFindings, id)
}

// IndexEvidence fetches an evidence row and indexes it.
func (ix *Indexer) IndexEvidence(ctx context.Context, id string) error {
	row := ix.db.QueryRowContext(ctx, `
		SELECT id, org_id, control_id, source, title, COALESCE(description,''),
		       COALESCE(evidence_type,''), updated_at
		FROM evidence WHERE id = $1`, id)

	var doc models.EvidenceDoc
	if err := row.Scan(
		&doc.ID, &doc.OrgID, &doc.ControlID, &doc.Source,
		&doc.Title, &doc.Description, &doc.EvidenceType, &doc.UpdatedAt,
	); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return fmt.Errorf("indexer: fetch evidence %s: %w", id, err)
	}
	return ix.os.IndexDoc(ctx, IndexEvidence, id, doc)
}

// DeleteEvidence removes an evidence document from the index.
func (ix *Indexer) DeleteEvidence(ctx context.Context, id string) error {
	return ix.os.DeleteDoc(ctx, IndexEvidence, id)
}

// IndexControl fetches a control row and indexes it.
func (ix *Indexer) IndexControl(ctx context.Context, id string) error {
	row := ix.db.QueryRowContext(ctx, `
		SELECT id, org_id, COALESCE(article_ref,''), title, COALESCE(description,''),
		       COALESCE(domain,''), updated_at
		FROM controls WHERE id = $1`, id)

	var doc models.ControlDoc
	if err := row.Scan(
		&doc.ID, &doc.OrgID, &doc.ArticleRef,
		&doc.Title, &doc.Description, &doc.Domain, &doc.UpdatedAt,
	); err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		return fmt.Errorf("indexer: fetch control %s: %w", id, err)
	}
	return ix.os.IndexDoc(ctx, IndexControls, id, doc)
}

// DeleteControl removes a control document from the index.
func (ix *Indexer) DeleteControl(ctx context.Context, id string) error {
	return ix.os.DeleteDoc(ctx, IndexControls, id)
}

// BulkReindex performs a full initial index of all findings, evidence, and controls.
// Used at startup to catch up with any rows written before the CDC pipeline was active.
func (ix *Indexer) BulkReindex(ctx context.Context) error {
	if err := ix.bulkIndexFindings(ctx); err != nil {
		return err
	}
	if err := ix.bulkIndexEvidence(ctx); err != nil {
		return err
	}
	return ix.bulkIndexControls(ctx)
}

func (ix *Indexer) bulkIndexFindings(ctx context.Context) error {
	rows, err := ix.db.QueryContext(ctx, `
		SELECT id, org_id, engagement_id, finding_ref, title, description,
		       COALESCE(root_cause,''), severity, status, tags, updated_at
		FROM findings`)
	if err != nil {
		return fmt.Errorf("indexer: bulk findings query: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var doc models.FindingDoc
		var tags []byte
		if err := rows.Scan(
			&doc.ID, &doc.OrgID, &doc.EngagementID, &doc.FindingRef,
			&doc.Title, &doc.Description, &doc.RootCause,
			&doc.Severity, &doc.Status, &tags, &doc.UpdatedAt,
		); err != nil {
			continue
		}
		doc.Tags = parseTextArray(tags)
		if err := ix.os.IndexDoc(ctx, IndexFindings, doc.ID, doc); err != nil {
			log.Printf("[indexer] bulk: finding %s: %v", doc.ID, err)
		}
		count++
	}
	log.Printf("[indexer] bulk-indexed %d findings", count)
	return rows.Err()
}

func (ix *Indexer) bulkIndexEvidence(ctx context.Context) error {
	rows, err := ix.db.QueryContext(ctx, `
		SELECT id, org_id, control_id, source, title,
		       COALESCE(description,''), COALESCE(evidence_type,''), updated_at
		FROM evidence`)
	if err != nil {
		return fmt.Errorf("indexer: bulk evidence query: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var doc models.EvidenceDoc
		if err := rows.Scan(
			&doc.ID, &doc.OrgID, &doc.ControlID, &doc.Source,
			&doc.Title, &doc.Description, &doc.EvidenceType, &doc.UpdatedAt,
		); err != nil {
			continue
		}
		if err := ix.os.IndexDoc(ctx, IndexEvidence, doc.ID, doc); err != nil {
			log.Printf("[indexer] bulk: evidence %s: %v", doc.ID, err)
		}
		count++
	}
	log.Printf("[indexer] bulk-indexed %d evidence items", count)
	return rows.Err()
}

func (ix *Indexer) bulkIndexControls(ctx context.Context) error {
	rows, err := ix.db.QueryContext(ctx, `
		SELECT id, org_id, COALESCE(article_ref,''), title,
		       COALESCE(description,''), COALESCE(domain,''), updated_at
		FROM controls`)
	if err != nil {
		return fmt.Errorf("indexer: bulk controls query: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		var doc models.ControlDoc
		if err := rows.Scan(
			&doc.ID, &doc.OrgID, &doc.ArticleRef,
			&doc.Title, &doc.Description, &doc.Domain, &doc.UpdatedAt,
		); err != nil {
			continue
		}
		if err := ix.os.IndexDoc(ctx, IndexControls, doc.ID, doc); err != nil {
			log.Printf("[indexer] bulk: control %s: %v", doc.ID, err)
		}
		count++
	}
	log.Printf("[indexer] bulk-indexed %d controls", count)
	return rows.Err()
}

// HandleEvent routes a ChangeEvent to the appropriate indexer method.
func (ix *Indexer) HandleEvent(ctx context.Context, ev models.ChangeEvent) {
	var err error
	switch ev.Table {
	case "findings":
		if ev.Op == "DELETE" {
			err = ix.DeleteFinding(ctx, ev.ID)
		} else {
			err = ix.IndexFinding(ctx, ev.ID)
		}
	case "evidence":
		if ev.Op == "DELETE" {
			err = ix.DeleteEvidence(ctx, ev.ID)
		} else {
			err = ix.IndexEvidence(ctx, ev.ID)
		}
	case "controls":
		if ev.Op == "DELETE" {
			err = ix.DeleteControl(ctx, ev.ID)
		} else {
			err = ix.IndexControl(ctx, ev.ID)
		}
	}
	if err != nil {
		log.Printf("[indexer] handle event %s %s/%s: %v", ev.Op, ev.Table, ev.ID, err)
	}
}

// --- helpers ---

// parseTextArray parses a PostgreSQL text[] literal like {foo,bar,baz}.
func parseTextArray(b []byte) []string {
	s := string(b)
	if s == "{}" || s == "" {
		return []string{}
	}
	s = s[1 : len(s)-1] // strip { }
	if s == "" {
		return []string{}
	}
	parts := splitCSV(s)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = trimQuotes(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitCSV(s string) []string {
	var parts []string
	var cur []byte
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, string(cur))
			cur = cur[:0]
		} else {
			cur = append(cur, s[i])
		}
	}
	parts = append(parts, string(cur))
	return parts
}

func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// --- index mappings ---

func findingsMapping() opensearch.IndexMapping {
	return opensearch.IndexMapping{
		Settings: map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		Mappings: map[string]interface{}{
			"properties": map[string]interface{}{
				"id":            map[string]interface{}{"type": "keyword"},
				"org_id":        map[string]interface{}{"type": "keyword"},
				"engagement_id": map[string]interface{}{"type": "keyword"},
				"finding_ref":   map[string]interface{}{"type": "keyword"},
				"title":         map[string]interface{}{"type": "text", "analyzer": "standard"},
				"description":   map[string]interface{}{"type": "text", "analyzer": "standard"},
				"root_cause":    map[string]interface{}{"type": "text", "analyzer": "standard"},
				"severity":      map[string]interface{}{"type": "keyword"},
				"status":        map[string]interface{}{"type": "keyword"},
				"tags":          map[string]interface{}{"type": "keyword"},
				"updated_at":    map[string]interface{}{"type": "date"},
			},
		},
	}
}

func evidenceMapping() opensearch.IndexMapping {
	return opensearch.IndexMapping{
		Settings: map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		Mappings: map[string]interface{}{
			"properties": map[string]interface{}{
				"id":             map[string]interface{}{"type": "keyword"},
				"org_id":         map[string]interface{}{"type": "keyword"},
				"control_id":     map[string]interface{}{"type": "keyword"},
				"source":         map[string]interface{}{"type": "text", "analyzer": "standard"},
				"title":          map[string]interface{}{"type": "text", "analyzer": "standard"},
				"description":    map[string]interface{}{"type": "text", "analyzer": "standard"},
				"relevance_tags": map[string]interface{}{"type": "keyword"},
				"evidence_type":  map[string]interface{}{"type": "keyword"},
				"updated_at":     map[string]interface{}{"type": "date"},
			},
		},
	}
}

func controlsMapping() opensearch.IndexMapping {
	return opensearch.IndexMapping{
		Settings: map[string]interface{}{
			"number_of_shards":   1,
			"number_of_replicas": 0,
		},
		Mappings: map[string]interface{}{
			"properties": map[string]interface{}{
				"id":          map[string]interface{}{"type": "keyword"},
				"org_id":      map[string]interface{}{"type": "keyword"},
				"article_ref": map[string]interface{}{"type": "keyword"},
				"title":       map[string]interface{}{"type": "text", "analyzer": "standard"},
				"description": map[string]interface{}{"type": "text", "analyzer": "standard"},
				"domain":      map[string]interface{}{"type": "keyword"},
				"updated_at":  map[string]interface{}{"type": "date"},
			},
		},
	}
}

// Startup waits for OpenSearch to be ready, then runs EnsureIndices.
func (ix *Indexer) Startup(ctx context.Context) error {
	for i := 0; i < 30; i++ {
		if err := ix.os.Ping(ctx); err == nil {
			break
		}
		log.Printf("[indexer] waiting for OpenSearch... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	return ix.EnsureIndices(ctx)
}

