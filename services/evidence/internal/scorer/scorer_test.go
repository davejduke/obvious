package scorer_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/models"
	"github.com/davejduke/obvious/services/evidence/internal/scorer"
)

func makeEvidence(id uuid.UUID, controlID uuid.UUID, title, desc, content string,
	source models.SourceType, collected time.Time, tags []string, fileHash string) *models.Evidence {
	return &models.Evidence{
		ID:          id,
		OrgID:       uuid.New(),
		ControlID:   controlID,
		Title:       title,
		Description: desc,
		Content:     content,
		SourceType:  source,
		CollectedAt: collected,
		Tags:        tags,
		FileHash:    fileHash,
	}
}

func TestScore_AggregateInRange(t *testing.T) {
	s := scorer.New()
	ev := makeEvidence(
		uuid.New(), uuid.New(),
		"Firewall Access Control Policy",
		"Monthly access log and authentication review for compliance audit",
		`{"entries": 1200, "source": "firewall"}`,
		models.SourceLogExport,
		time.Now().UTC(),
		[]string{"firewall", "access-control"},
		"sha256:abc123",
	)

	qs := s.Score(ev, []*models.Evidence{ev})

	if qs.AggregateScore < 0.0 || qs.AggregateScore > 1.0 {
		t.Errorf("aggregate score %f out of [0,1]", qs.AggregateScore)
	}
}

func TestScore_FreshEvidenceHighCurrency(t *testing.T) {
	s := scorer.New()
	ev := makeEvidence(
		uuid.New(), uuid.New(),
		"Recent Vulnerability Scan",
		"Automated scan result from today",
		"<scan-data/>",
		models.SourceAutomatedScan,
		time.Now().UTC(),
		[]string{"vulnerability"},
		"",
	)

	qs := s.Score(ev, []*models.Evidence{ev})

	if qs.CurrencyScore < 0.90 {
		t.Errorf("expected high currency for fresh evidence, got %.4f", qs.CurrencyScore)
	}
}

func TestScore_OldEvidenceLowCurrency(t *testing.T) {
	s := scorer.New()
	old := time.Now().UTC().AddDate(-2, 0, 0) // 2 years ago
	ev := makeEvidence(
		uuid.New(), uuid.New(),
		"Old Policy Document",
		"A governance policy from 2 years ago",
		"content here",
		models.SourceManualUpload,
		old,
		[]string{},
		"",
	)

	qs := s.Score(ev, []*models.Evidence{ev})

	if qs.CurrencyScore > 0.20 {
		t.Errorf("expected low currency for old evidence, got %.4f", qs.CurrencyScore)
	}
}

func TestScore_APISourceHighReliability(t *testing.T) {
	s := scorer.New()
	ev := makeEvidence(
		uuid.New(), uuid.New(),
		"API Integration Feed", "Direct API monitoring data",
		`{}`, models.SourceAPIIntegration, time.Now().UTC(),
		[]string{"api"}, "",
	)

	qs := s.Score(ev, []*models.Evidence{})

	if qs.SourceReliability < 0.85 {
		t.Errorf("expected high reliability for API source, got %.4f", qs.SourceReliability)
	}
}

func TestScore_ScreenshotLowerReliability(t *testing.T) {
	s := scorer.New()
	ev := makeEvidence(
		uuid.New(), uuid.New(),
		"Screenshot", "Manual screenshot evidence",
		"", models.SourceScreenshot, time.Now().UTC(),
		[]string{}, "",
	)

	qs := s.Score(ev, []*models.Evidence{})

	// Screenshot reliability should be lower than API
	if qs.SourceReliability > 0.75 {
		t.Errorf("expected lower reliability for screenshot, got %.4f", qs.SourceReliability)
	}
}

func TestScore_MissingContentLowCompleteness(t *testing.T) {
	s := scorer.New()
	ev := makeEvidence(
		uuid.New(), uuid.New(),
		"Some Evidence", "", // no description
		"",                  // no content
		models.SourceManualUpload, time.Now().UTC(),
		[]string{}, "",
	)

	qs := s.Score(ev, []*models.Evidence{})

	if qs.CompletenessScore > 0.60 {
		t.Errorf("expected low completeness for missing content, got %.4f", qs.CompletenessScore)
	}
}

func TestScore_MultipleCorroboratingEvidence(t *testing.T) {
	s := scorer.New()
	controlID := uuid.New()
	ev1 := makeEvidence(uuid.New(), controlID, "E1", "desc", "content", models.SourceLogExport, time.Now(), []string{}, "")
	ev2 := makeEvidence(uuid.New(), controlID, "E2", "desc", "content", models.SourceAutomatedScan, time.Now(), []string{}, "")
	ev3 := makeEvidence(uuid.New(), controlID, "E3", "desc", "content", models.SourceManualUpload, time.Now(), []string{}, "")
	ev4 := makeEvidence(uuid.New(), controlID, "E4", "desc", "content", models.SourceAPIIntegration, time.Now(), []string{}, "")

	peers := []*models.Evidence{ev1, ev2, ev3, ev4}
	qs := s.Score(ev1, peers)

	// With 3 peers, corroboration should be high
	if qs.CorroborationScore < 0.80 {
		t.Errorf("expected high corroboration with 4 peers, got %.4f", qs.CorroborationScore)
	}
}

func TestScore_AllScoresHaveProperRange(t *testing.T) {
	s := scorer.New()
	ev := makeEvidence(
		uuid.New(), uuid.New(),
		"Security Audit Control Evidence",
		"Compliance audit authentication access control monitoring",
		`{"firewall": true, "backup": "daily"}`,
		models.SourceConfigurationExport,
		time.Now().UTC(),
		[]string{"security", "compliance"},
		"sha256:xyz",
	)
	peers := []*models.Evidence{ev}

	qs := s.Score(ev, peers)

	for name, score := range map[string]float64{
		"completeness":  qs.CompletenessScore,
		"currency":      qs.CurrencyScore,
		"reliability":   qs.SourceReliability,
		"corroboration": qs.CorroborationScore,
		"relevance":     qs.RelevanceScore,
		"aggregate":     qs.AggregateScore,
	} {
		if score < 0.0 || score > 1.0 {
			t.Errorf("%s score %.4f out of [0,1]", name, score)
		}
	}
}

