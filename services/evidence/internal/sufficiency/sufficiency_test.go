package sufficiency_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/models"
	"github.com/davejduke/obvious/services/evidence/internal/sufficiency"
)

func makeEvidenceItem(controlID uuid.UUID) *models.Evidence {
	return &models.Evidence{
		ID:          uuid.New(),
		OrgID:       uuid.New(),
		ControlID:   controlID,
		Title:       "Evidence Item",
		Description: "A test evidence item",
		Content:     "content",
		SourceType:  models.SourceLogExport,
		CollectedAt: time.Now().UTC(),
	}
}

func makeQualityScore(evidenceID uuid.UUID, aggregate float64) *models.QualityScore {
	return &models.QualityScore{
		ID:                 uuid.New(),
		EvidenceID:         evidenceID,
		OrgID:              uuid.New(),
		CompletenessScore:  aggregate,
		CurrencyScore:      aggregate,
		SourceReliability:  aggregate,
		CorroborationScore: aggregate,
		RelevanceScore:     aggregate,
		AggregateScore:     aggregate,
		ScoredAt:           time.Now().UTC(),
	}
}

func TestSufficiency_Sufficient_Critical(t *testing.T) {
	calc := sufficiency.New()
	controlID := uuid.New()

	evidence := make([]*models.Evidence, 5)
	scores := make([]*models.QualityScore, 5)
	for i := range evidence {
		ev := makeEvidenceItem(controlID)
		evidence[i] = ev
		scores[i] = makeQualityScore(ev.ID, 0.85)
	}

	result := calc.Evaluate(models.RiskCritical, evidence, scores, models.RiskCritical)

	if !result.IsSufficient {
		t.Errorf("expected sufficient for critical with 5 items at 0.85 score, gaps: %v", result.Gaps)
	}
	if result.MinRequired != 5 {
		t.Errorf("expected min_required=5 for critical, got %d", result.MinRequired)
	}
	if result.MinScoreRequired != 0.80 {
		t.Errorf("expected min_score=0.80 for critical, got %.2f", result.MinScoreRequired)
	}
}

func TestSufficiency_Insufficient_NotEnoughItems(t *testing.T) {
	calc := sufficiency.New()
	controlID := uuid.New()

	// Only 2 items for high-risk (needs 4)
	evidence := make([]*models.Evidence, 2)
	scores := make([]*models.QualityScore, 2)
	for i := range evidence {
		ev := makeEvidenceItem(controlID)
		evidence[i] = ev
		scores[i] = makeQualityScore(ev.ID, 0.80)
	}

	result := calc.Evaluate(models.RiskHigh, evidence, scores, models.RiskHigh)

	if result.IsSufficient {
		t.Error("expected insufficient when only 2 items for high-risk (needs 4)")
	}
	if len(result.Gaps) == 0 {
		t.Error("expected at least one gap message")
	}
}

func TestSufficiency_Insufficient_LowScore(t *testing.T) {
	calc := sufficiency.New()
	controlID := uuid.New()

	// 4 items for high-risk but score too low (needs 0.70)
	evidence := make([]*models.Evidence, 4)
	scores := make([]*models.QualityScore, 4)
	for i := range evidence {
		ev := makeEvidenceItem(controlID)
		evidence[i] = ev
		scores[i] = makeQualityScore(ev.ID, 0.45) // below 0.70 threshold
	}

	result := calc.Evaluate(models.RiskHigh, evidence, scores, models.RiskHigh)

	if result.IsSufficient {
		t.Errorf("expected insufficient with avg score 0.45 for high-risk (needs 0.70), gaps: %v", result.Gaps)
	}
}

func TestSufficiency_EmptyEvidence_AlwaysInsufficient(t *testing.T) {
	calc := sufficiency.New()

	result := calc.Evaluate(models.RiskMedium, []*models.Evidence{}, []*models.QualityScore{}, models.RiskMedium)

	if result.IsSufficient {
		t.Error("expected insufficient for empty evidence")
	}
	if result.EvidenceCount != 0 {
		t.Errorf("expected 0 evidence count, got %d", result.EvidenceCount)
	}
}

func TestSufficiency_Thresholds_AllRiskLevels(t *testing.T) {
	calc := sufficiency.New()

	tests := []struct {
		risk     models.RiskRating
		minCount int
		minScore float64
	}{
		{models.RiskCritical, 5, 0.80},
		{models.RiskHigh, 4, 0.70},
		{models.RiskMedium, 3, 0.60},
		{models.RiskLow, 2, 0.50},
		{models.RiskInformational, 1, 0.40},
	}

	for _, tt := range tests {
		controlID := uuid.New()
		evidence := make([]*models.Evidence, tt.minCount)
		scores := make([]*models.QualityScore, tt.minCount)
		for i := range evidence {
			ev := makeEvidenceItem(controlID)
			evidence[i] = ev
			scores[i] = makeQualityScore(ev.ID, tt.minScore+0.05)
		}

		result := calc.Evaluate(tt.risk, evidence, scores, tt.risk)

		if !result.IsSufficient {
			t.Errorf("%s: expected sufficient with %d items at %.2f score, gaps: %v",
				tt.risk, tt.minCount, tt.minScore+0.05, result.Gaps)
		}
		if result.MinRequired != tt.minCount {
			t.Errorf("%s: expected MinRequired=%d, got %d", tt.risk, tt.minCount, result.MinRequired)
		}
		if result.MinScoreRequired != tt.minScore {
			t.Errorf("%s: expected MinScoreRequired=%.2f, got %.2f", tt.risk, tt.minScore, result.MinScoreRequired)
		}
	}
}

func TestSufficiency_Gaps_IdentifyCountAndScore(t *testing.T) {
	calc := sufficiency.New()
	controlID := uuid.New()

	// 1 item for medium (needs 3), poor quality
	ev := makeEvidenceItem(controlID)
	evidence := []*models.Evidence{ev}
	scores := []*models.QualityScore{makeQualityScore(ev.ID, 0.30)}

	result := calc.Evaluate(models.RiskMedium, evidence, scores, models.RiskMedium)

	if result.IsSufficient {
		t.Error("expected insufficient")
	}
	// Should report both count gap and score gap
	if len(result.Gaps) < 2 {
		t.Errorf("expected at least 2 gaps (count + score), got %d: %v", len(result.Gaps), result.Gaps)
	}
}

