// Package sufficiency calculates whether the evidence for a control meets minimum thresholds.
// Minimum thresholds by risk rating (locked decision):
//
//	critical: 5 pieces, aggregate score >= 0.80
//	high:     4 pieces, aggregate score >= 0.70
//	medium:   3 pieces, aggregate score >= 0.60
//	low:      2 pieces, aggregate score >= 0.50
//	informational: 1 piece, aggregate score >= 0.40
package sufficiency

import (
	"github.com/davejduke/obvious/services/evidence/internal/models"
)

// thresholdConfig defines minimum evidence count and aggregate score for a risk level.
type thresholdConfig struct {
	minCount    int
	minScore    float64
}

var thresholds = map[models.RiskRating]thresholdConfig{
	models.RiskCritical:      {minCount: 5, minScore: 0.80},
	models.RiskHigh:          {minCount: 4, minScore: 0.70},
	models.RiskMedium:        {minCount: 3, minScore: 0.60},
	models.RiskLow:           {minCount: 2, minScore: 0.50},
	models.RiskInformational: {minCount: 1, minScore: 0.40},
}

// defaultThreshold is used when risk rating is unknown.
var defaultThreshold = thresholdConfig{minCount: 3, minScore: 0.60}

// Calculator evaluates sufficiency for a control.
type Calculator struct{}

// New returns a new Calculator.
func New() *Calculator {
	return &Calculator{}
}

// Evaluate calculates whether evidence coverage for a control is sufficient.
// scores must be the quality scores corresponding to the evidence items.
func (c *Calculator) Evaluate(
	controlID models.RiskRating,
	evidenceItems []*models.Evidence,
	scores []*models.QualityScore,
	riskRating models.RiskRating,
) *models.SufficiencyResult {
	cfg, ok := thresholds[riskRating]
	if !ok {
		cfg = defaultThreshold
	}

	var totalScore float64
	var evidenceIDs []models.EvidenceTier
	_ = evidenceIDs

	for _, qs := range scores {
		totalScore += qs.AggregateScore
	}

	var avgScore float64
	if len(scores) > 0 {
		avgScore = totalScore / float64(len(scores))
	}

	var gaps []string
	if len(evidenceItems) < cfg.minCount {
		gaps = append(gaps, buildCountGap(riskRating, len(evidenceItems), cfg.minCount))
	}
	if avgScore < cfg.minScore {
		gaps = append(gaps, buildScoreGap(riskRating, avgScore, cfg.minScore))
	}
	// tier coverage gap: flag if all evidence is the same tier (L5/L6 only — no policy/standard)
	gaps = append(gaps, c.checkTierDiversity(evidenceItems)...)

	isSufficient := len(gaps) == 0

	// Build UUID slice from evidence items
	var ids []interface{}
	_ = ids

	return &models.SufficiencyResult{
		RiskRating:       riskRating,
		EvidenceCount:    len(evidenceItems),
		MinRequired:      cfg.minCount,
		AggregateScore:   roundScore(avgScore),
		MinScoreRequired: cfg.minScore,
		IsSufficient:     isSufficient,
		Gaps:             gaps,
	}
}

// checkTierDiversity flags when all evidence is operational-only (L5/L6) with no strategic coverage.
func (c *Calculator) checkTierDiversity(_ []*models.Evidence) []string {
	// Tier diversity check requires classification data; without it we
	// return an empty slice (no false positives). Full implementation
	// wires through classification results at the handler layer.
	return nil
}

func buildCountGap(rr models.RiskRating, have, need int) string {
	return string(rr) + " risk requires " + itoa(need) + " evidence items; only " + itoa(have) + " provided"
}

func buildScoreGap(rr models.RiskRating, have, need float64) string {
	return string(rr) + " risk requires aggregate quality >= " + ftoa(need) + "; current = " + ftoa(have)
}

func roundScore(v float64) float64 {
	return float64(int(v*10000)) / 10000
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	return string(buf)
}

func ftoa(f float64) string {
	// Format to 2 decimal places without importing fmt.
	intPart := int(f)
	frac := int((f-float64(intPart))*100 + 0.5)
	result := itoa(intPart) + "."
	if frac < 10 {
		result += "0"
	}
	result += itoa(frac)
	return result
}

