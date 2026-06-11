// Package classifier implements the 6-tier evidence hierarchy classification engine.
// Tiers (locked decision): L1 Policy > L2 Standard > L3 Guideline > L4 Procedure > L5 Record > L6 Telemetry
package classifier

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/models"
)

// tierRule defines keyword patterns that signal a particular evidence tier.
type tierRule struct {
	tier     models.EvidenceTier
	keywords []string
	sources  []models.SourceType
	formats  []models.ContentFormat
}

// rules are evaluated top-down; first match wins.
var rules = []tierRule{
	{
		tier: models.TierL1Policy,
		keywords: []string{
			"policy", "governance", "mandate", "regulation", "directive", "gdpr", "nis2", "nis 2",
			"iso 27001", "soc 2", "hipaa", "pci dss", "executive order", "board resolution",
			"information security policy", "acceptable use policy", "aup",
		},
		sources: []models.SourceType{models.SourceManualUpload, models.SourceConfigurationExport},
	},
	{
		tier: models.TierL2Standard,
		keywords: []string{
			"standard", "framework", "benchmark", "cis benchmark", "nist", "iso standard",
			"baseline", "specification", "profile", "control catalog", "technical standard",
			"security baseline", "hardening standard", "configuration standard",
		},
		sources: []models.SourceType{models.SourceManualUpload, models.SourceConfigurationExport},
	},
	{
		tier: models.TierL3Guideline,
		keywords: []string{
			"guideline", "best practice", "recommendation", "advisory", "guidance",
			"handbook", "reference guide", "design principle", "architecture guideline",
			"security guideline", "operational guidance",
		},
		sources: []models.SourceType{models.SourceManualUpload},
	},
	{
		tier: models.TierL4Procedure,
		keywords: []string{
			"procedure", "sop", "runbook", "playbook", "work instruction", "process document",
			"incident response procedure", "change management procedure", "access request procedure",
			"backup procedure", "recovery procedure", "step-by-step",
		},
		sources: []models.SourceType{models.SourceManualUpload},
	},
	{
		tier: models.TierL5Record,
		keywords: []string{
			"log", "screenshot", "audit log", "access log", "event log", "change log",
			"scan result", "vulnerability report", "penetration test", "pentest",
			"configuration export", "firewall rule", "user list", "access review",
			"training record", "certificate", "approval record", "incident report",
			"evidence screenshot", "system output",
		},
		sources: []models.SourceType{models.SourceScreenshot, models.SourceLogExport, models.SourceConfigurationExport, models.SourceAutomatedScan},
	},
	{
		tier: models.TierL6Telemetry,
		keywords: []string{
			"telemetry", "metric", "monitoring", "alert", "siem", "splunk", "datadog",
			"cloudwatch", "prometheus", "grafana", "webhook", "api feed", "real-time",
			"continuous monitoring", "automated feed", "sensor data", "uptime", "heartbeat",
			"anomaly detection", "intrusion detection",
		},
		sources: []models.SourceType{models.SourceAPIIntegration, models.SourceAutomatedScan},
		formats: []models.ContentFormat{models.FormatLog, models.FormatJSON},
	},
}

// Classifier classifies evidence items into the 6-tier hierarchy.
type Classifier struct{}

// New creates a new Classifier.
func New() *Classifier {
	return &Classifier{}
}

// Classify inspects evidence fields and assigns the best-matching tier.
func (c *Classifier) Classify(ev *models.Evidence) *models.Classification {
	text := strings.ToLower(ev.Title + " " + ev.Description)

	bestTier := models.TierL5Record // default
	bestConfidence := 0.40
	bestReasoning := "defaulted to L5-Record; no stronger keyword match found"

	for _, rule := range rules {
		score := c.scoreRule(rule, text, ev.SourceType, ev.ContentFormat)
		if score > bestConfidence {
			bestConfidence = score
			bestTier = rule.tier
			bestReasoning = c.buildReasoning(rule, text)
		}
	}

	return &models.Classification{
		ID:           uuid.New(),
		EvidenceID:   ev.ID,
		OrgID:        ev.OrgID,
		Tier:         bestTier,
		TierLabel:    bestTier.String(),
		Confidence:   clamp(bestConfidence),
		Reasoning:    bestReasoning,
		ClassifiedAt: time.Now().UTC(),
	}
}

// scoreRule scores a rule against the evidence text and source/format signals.
func (c *Classifier) scoreRule(
	rule tierRule,
	text string,
	source models.SourceType,
	format models.ContentFormat,
) float64 {
	var keywordHits int
	for _, kw := range rule.keywords {
		if strings.Contains(text, kw) {
			keywordHits++
		}
	}
	if keywordHits == 0 {
		return 0
	}

	// keyword score: first hit = 0.55, each additional +0.05, max 0.80
	score := 0.50 + float64(keywordHits)*0.05
	if score > 0.80 {
		score = 0.80
	}

	// Source signal bonus
	for _, s := range rule.sources {
		if s == source {
			score += 0.10
			break
		}
	}

	// Format signal bonus
	for _, f := range rule.formats {
		if f == format {
			score += 0.05
			break
		}
	}

	return score
}

// buildReasoning produces a human-readable explanation of the classification.
func (c *Classifier) buildReasoning(rule tierRule, text string) string {
	var matched []string
	for _, kw := range rule.keywords {
		if strings.Contains(text, kw) {
			matched = append(matched, kw)
			if len(matched) == 3 {
				break
			}
		}
	}
	return "matched " + rule.tier.String() + " keywords: " + strings.Join(matched, ", ")
}

// clamp ensures a score is in [0.0, 1.0].
func clamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

