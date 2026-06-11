// Package scorer implements the 5-dimension evidence quality scoring algorithm.
// Dimensions: completeness, currency, source reliability, corroboration, relevance.
// Each dimension scores 0.0–1.0; aggregate is a weighted average.
package scorer

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/models"
)

// weights for the aggregate score computation.
const (
	weightCompleteness  = 0.25
	weightCurrency      = 0.20
	weightReliability   = 0.25
	weightCorroboration = 0.15
	weightRelevance     = 0.15
)

// sourceReliabilityBase is a baseline trust score per source type.
var sourceReliabilityBase = map[models.SourceType]float64{
	models.SourceAPIIntegration:      0.90, // automated, direct from system
	models.SourceAutomatedScan:       0.85, // tool-generated, repeatable
	models.SourceConfigurationExport: 0.85, // direct config dump
	models.SourceLogExport:           0.80, // raw logs
	models.SourceScreenshot:          0.60, // human-captured, tamperable
	models.SourceManualUpload:        0.50, // most manual, varies widely
}

// Scorer computes multi-dimension quality scores for evidence.
type Scorer struct{}

// New returns a new Scorer.
func New() *Scorer {
	return &Scorer{}
}

// Score evaluates evidence quality across 5 dimensions and computes an aggregate.
func (s *Scorer) Score(ev *models.Evidence, allEvidenceForControl []*models.Evidence) *models.QualityScore {
	completeness, compDefs := s.scoreCompleteness(ev)
	currency, currDefs := s.scoreCurrency(ev)
	reliability, relDefs := s.scoreReliability(ev)
	corroboration, corrDefs := s.scoreCorroboration(ev, allEvidenceForControl)
	relevance, relevDefs := s.scoreRelevance(ev)

	aggregate := weightCompleteness*completeness +
		weightCurrency*currency +
		weightReliability*reliability +
		weightCorroboration*corroboration +
		weightRelevance*relevance

	var deficiencies []string
	deficiencies = append(deficiencies, compDefs...)
	deficiencies = append(deficiencies, currDefs...)
	deficiencies = append(deficiencies, relDefs...)
	deficiencies = append(deficiencies, corrDefs...)
	deficiencies = append(deficiencies, relevDefs...)

	return &models.QualityScore{
		ID:                 uuid.New(),
		EvidenceID:         ev.ID,
		OrgID:              ev.OrgID,
		CompletenessScore:  roundScore(completeness),
		CurrencyScore:      roundScore(currency),
		SourceReliability:  roundScore(reliability),
		CorroborationScore: roundScore(corroboration),
		RelevanceScore:     roundScore(relevance),
		AggregateScore:     roundScore(aggregate),
		Deficiencies:       deficiencies,
		ScoredAt:           time.Now().UTC(),
	}
}

// scoreCompleteness checks whether required fields are populated.
// Full score = title, description, content, collected_at all present.
func (s *Scorer) scoreCompleteness(ev *models.Evidence) (float64, []string) {
	var score float64 = 1.0
	var defs []string

	if strings.TrimSpace(ev.Title) == "" {
		score -= 0.25
		defs = append(defs, "missing title")
	}
	if strings.TrimSpace(ev.Description) == "" {
		score -= 0.15
		defs = append(defs, "missing description")
	}
	if strings.TrimSpace(ev.Content) == "" && strings.TrimSpace(ev.FilePath) == "" {
		score -= 0.40
		defs = append(defs, "no content or file attached")
	}
	if ev.CollectedAt.IsZero() {
		score -= 0.15
		defs = append(defs, "missing collection timestamp")
	}
	if len(ev.Tags) == 0 {
		score -= 0.05
		defs = append(defs, "no tags provided")
	}
	return clamp(score), defs
}

// scoreCurrency evaluates how recent the evidence is.
// Evidence older than 365 days is penalised; fresh (<30d) is ideal.
func (s *Scorer) scoreCurrency(ev *models.Evidence) (float64, []string) {
	if ev.CollectedAt.IsZero() {
		return 0.20, []string{"collection date unknown — currency cannot be verified"}
	}

	age := time.Since(ev.CollectedAt)
	days := age.Hours() / 24

	switch {
	case days <= 30:
		return 1.0, nil
	case days <= 90:
		return 0.90, nil
	case days <= 180:
		return 0.75, nil
	case days <= 365:
		return 0.55, []string{"evidence older than 6 months; consider refreshing"}
	case days <= 730:
		return 0.30, []string{"evidence older than 1 year; currency risk"}
	default:
		return 0.10, []string{"evidence older than 2 years; likely stale"}
	}
}

// scoreReliability scores the source trustworthiness.
func (s *Scorer) scoreReliability(ev *models.Evidence) (float64, []string) {
	base, ok := sourceReliabilityBase[ev.SourceType]
	if !ok {
		return 0.50, []string{"unknown source type; defaulting to 0.50 reliability"}
	}

	// Provenance chain bonus: each hop adds 0.02 up to 0.08
	provenanceBonus := float64(len(ev.ProvenanceChain)) * 0.02
	if provenanceBonus > 0.08 {
		provenanceBonus = 0.08
	}

	// Hash presence bonus: indicates tamper-evidence
	var hashBonus float64
	if strings.TrimSpace(ev.FileHash) != "" {
		hashBonus = 0.05
	}

	return clamp(base + provenanceBonus + hashBonus), nil
}

// scoreCorroboration checks how well this evidence is supported by other evidence for the same control.
// More pieces of evidence = higher corroboration, up to a max of 4 pieces.
func (s *Scorer) scoreCorroboration(ev *models.Evidence, peers []*models.Evidence) (float64, []string) {
	// exclude self from peer count
	peerCount := 0
	for _, p := range peers {
		if p.ID != ev.ID {
			peerCount++
		}
	}

	switch {
	case peerCount == 0:
		return 0.20, []string{"sole evidence for this control; no corroboration"}
	case peerCount == 1:
		return 0.50, nil
	case peerCount == 2:
		return 0.70, nil
	case peerCount == 3:
		return 0.85, nil
	default:
		return 1.0, nil
	}
}

// scoreRelevance measures how directly the evidence addresses its linked control.
// Uses tags and content keyword matching against control-related terms.
func (s *Scorer) scoreRelevance(ev *models.Evidence) (float64, []string) {
	// If tags are present they indicate deliberate linking; bonus.
	tagScore := 0.0
	if len(ev.Tags) > 0 {
		tagScore = 0.20
	}

	// Content analysis: look for control-related keywords.
	content := strings.ToLower(ev.Title + " " + ev.Description + " " + ev.Content)
	relevantTerms := []string{
		"control", "audit", "compliance", "security", "risk", "policy",
		"access", "authentication", "encryption", "monitoring", "vulnerability",
		"incident", "patch", "firewall", "backup", "continuity",
	}
	hits := 0
	for _, term := range relevantTerms {
		if strings.Contains(content, term) {
			hits++
		}
	}

	contentScore := 0.40 // baseline
	if hits >= 5 {
		contentScore = 0.80
	} else if hits >= 3 {
		contentScore = 0.65
	} else if hits >= 1 {
		contentScore = 0.50
	}

	score := clamp(contentScore + tagScore)
	var defs []string
	if score < 0.50 {
		defs = append(defs, "low relevance score; review if evidence maps to the correct control")
	}
	return score, defs
}

// clamp keeps v in [0.0, 1.0].
func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// roundScore rounds to 4 decimal places for consistent storage.
func roundScore(v float64) float64 {
	return float64(int(v*10000)) / 10000
}

