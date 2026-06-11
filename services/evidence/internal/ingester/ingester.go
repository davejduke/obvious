// Package ingester handles parsing and normalising evidence in JSON, CSV, and log formats.
package ingester

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/models"
)

// Ingester normalises raw evidence payloads into the Evidence domain model.
type Ingester struct{}

// New returns a new Ingester.
func New() *Ingester {
	return &Ingester{}
}

// Ingest transforms an IngestRequest into a fully-populated Evidence value.
// It parses and validates the content according to the declared content format.
func (i *Ingester) Ingest(req *models.IngestRequest) (*models.Evidence, error) {
	if err := i.validate(req); err != nil {
		return nil, fmt.Errorf("validation: %w", err)
	}

	orgID, err := uuid.Parse(req.OrgID)
	if err != nil {
		return nil, fmt.Errorf("invalid org_id: %w", err)
	}
	engagementID, err := uuid.Parse(req.EngagementID)
	if err != nil {
		return nil, fmt.Errorf("invalid engagement_id: %w", err)
	}
	controlID, err := uuid.Parse(req.ControlID)
	if err != nil {
		return nil, fmt.Errorf("invalid control_id: %w", err)
	}

	// Parse and normalise content according to format
	normContent, parseErr := i.parseContent(req.Content, req.ContentFormat)
	if parseErr != nil {
		return nil, fmt.Errorf("content parse (%s): %w", req.ContentFormat, parseErr)
	}

	collectedAt := time.Now().UTC()
	if req.CollectedAt != nil {
		collectedAt = req.CollectedAt.UTC()
	}

	evID := uuid.New()

	ev := &models.Evidence{
		ID:            evID,
		OrgID:         orgID,
		EngagementID:  engagementID,
		ControlID:     controlID,
		Title:         strings.TrimSpace(req.Title),
		Description:   strings.TrimSpace(req.Description),
		SourceType:    req.SourceType,
		SourceRef:     req.SourceRef,
		ContentFormat: req.ContentFormat,
		Content:       normContent,
		CollectedAt:   collectedAt,
		Tags:          normaliseTags(req.Tags),
		Metadata:      req.Metadata,
		ProvenanceChain: []models.ProvenanceEntry{
			{
				Timestamp:   time.Now().UTC(),
				Action:      "ingested",
				Description: "evidence ingested via API; format=" + string(req.ContentFormat) + " source=" + string(req.SourceType),
			},
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	return ev, nil
}

// validate checks required fields on an IngestRequest.
func (i *Ingester) validate(req *models.IngestRequest) error {
	if strings.TrimSpace(req.OrgID) == "" {
		return fmt.Errorf("org_id is required")
	}
	if strings.TrimSpace(req.EngagementID) == "" {
		return fmt.Errorf("engagement_id is required")
	}
	if strings.TrimSpace(req.ControlID) == "" {
		return fmt.Errorf("control_id is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return fmt.Errorf("title is required")
	}
	switch req.SourceType {
	case models.SourceManualUpload,
		models.SourceAPIIntegration,
		models.SourceAutomatedScan,
		models.SourceScreenshot,
		models.SourceLogExport,
		models.SourceConfigurationExport:
	// valid
	default:
		return fmt.Errorf("invalid source_type: %q", req.SourceType)
	}
	switch req.ContentFormat {
	case models.FormatJSON, models.FormatCSV, models.FormatLog, models.FormatText:
	// valid
	default:
		return fmt.Errorf("invalid content_format: %q", req.ContentFormat)
	}
	return nil
}

// parseContent validates and normalises evidence content for a given format.
// JSON is re-serialised for canonical form; CSV is validated; logs are kept as-is.
func (i *Ingester) parseContent(content string, format models.ContentFormat) (string, error) {
	switch format {
	case models.FormatJSON:
		return parseJSON(content)
	case models.FormatCSV:
		return parseCSV(content)
	case models.FormatLog:
		return parseLog(content)
	case models.FormatText:
		return content, nil
	default:
		return content, nil
	}
}

// parseJSON validates JSON and returns canonical (compact) form.
func parseJSON(content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	var v interface{}
	if err := json.Unmarshal([]byte(content), &v); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("re-marshal: %w", err)
	}
	return string(b), nil
}

// parseCSV validates CSV and returns the content unchanged if valid.
func parseCSV(content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	r := csv.NewReader(strings.NewReader(content))
	records, err := r.ReadAll()
	if err != nil {
		return "", fmt.Errorf("invalid CSV: %w", err)
	}
	if len(records) == 0 {
		return "", fmt.Errorf("CSV has no records")
	}
	return content, nil
}

// parseLog performs basic sanity checks on log content and returns it unchanged.
func parseLog(content string) (string, error) {
	if strings.TrimSpace(content) == "" {
		return "", nil
	}
	lines := strings.Split(content, "\n")
	if len(lines) == 0 {
		return "", fmt.Errorf("log content has no lines")
	}
	return content, nil
}

// normaliseTags deduplicates and lowercases tags.
func normaliseTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		result = append(result, t)
	}
	if result == nil {
		return []string{}
	}
	return result
}

