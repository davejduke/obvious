package classifier_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/davejduke/obvious/services/evidence/internal/classifier"
	"github.com/davejduke/obvious/services/evidence/internal/models"
)

func makeEvidence(title, desc string, source models.SourceType, format models.ContentFormat) *models.Evidence {
	return &models.Evidence{
		ID:            uuid.New(),
		OrgID:         uuid.New(),
		Title:         title,
		Description:   desc,
		SourceType:    source,
		ContentFormat: format,
		CollectedAt:   time.Now().UTC(),
	}
}

func TestClassify_L1Policy(t *testing.T) {
	c := classifier.New()
	ev := makeEvidence("Information Security Policy 2024", "Governance policy covering all employees",
		models.SourceManualUpload, models.FormatText)

	result := c.Classify(ev)

	if result.Tier != models.TierL1Policy {
		t.Errorf("expected L1-Policy, got %s (confidence=%.2f, reasoning=%q)",
			result.TierLabel, result.Confidence, result.Reasoning)
	}
	if result.Confidence <= 0.50 {
		t.Errorf("expected confidence > 0.50, got %.4f", result.Confidence)
	}
}

func TestClassify_L2Standard(t *testing.T) {
	c := classifier.New()
	ev := makeEvidence("CIS Benchmark v8 Hardening Standard", "Security baseline for server configuration",
		models.SourceManualUpload, models.FormatText)

	result := c.Classify(ev)

	if result.Tier != models.TierL2Standard {
		t.Errorf("expected L2-Standard, got %s (confidence=%.2f, reasoning=%q)",
			result.TierLabel, result.Confidence, result.Reasoning)
	}
}

func TestClassify_L3Guideline(t *testing.T) {
	c := classifier.New()
	ev := makeEvidence("Cloud Security Guideline", "Best practice guidance for AWS configurations",
		models.SourceManualUpload, models.FormatText)

	result := c.Classify(ev)

	if result.Tier != models.TierL3Guideline {
		t.Errorf("expected L3-Guideline, got %s (confidence=%.2f)",
			result.TierLabel, result.Confidence)
	}
}

func TestClassify_L4Procedure(t *testing.T) {
	c := classifier.New()
	ev := makeEvidence("Incident Response Procedure SOP-IR-001",
		"Step-by-step runbook for handling security incidents",
		models.SourceManualUpload, models.FormatText)

	result := c.Classify(ev)

	if result.Tier != models.TierL4Procedure {
		t.Errorf("expected L4-Procedure, got %s (confidence=%.2f)",
			result.TierLabel, result.Confidence)
	}
}

func TestClassify_L5Record(t *testing.T) {
	c := classifier.New()
	ev := makeEvidence("Firewall Audit Log Export Q1 2024",
		"Access log and event log from perimeter firewall",
		models.SourceLogExport, models.FormatLog)

	result := c.Classify(ev)

	if result.Tier != models.TierL5Record {
		t.Errorf("expected L5-Record, got %s (confidence=%.2f)",
			result.TierLabel, result.Confidence)
	}
}

func TestClassify_L6Telemetry(t *testing.T) {
	c := classifier.New()
	ev := makeEvidence("Splunk SIEM Alert Feed",
		"Real-time telemetry and anomaly detection metrics from SIEM monitoring system",
		models.SourceAPIIntegration, models.FormatJSON)

	result := c.Classify(ev)

	if result.Tier != models.TierL6Telemetry {
		t.Errorf("expected L6-Telemetry, got %s (confidence=%.2f)",
			result.TierLabel, result.Confidence)
	}
}

func TestClassify_ConfidenceInRange(t *testing.T) {
	c := classifier.New()
	tests := []struct {
		title  string
		desc   string
		source models.SourceType
		fmt    models.ContentFormat
	}{
		{"Policy Document", "policy governance", models.SourceManualUpload, models.FormatText},
		{"Scan Result", "vulnerability report", models.SourceAutomatedScan, models.FormatJSON},
		{"Unknown item", "some data", models.SourceManualUpload, models.FormatText},
	}
	for _, tt := range tests {
		ev := makeEvidence(tt.title, tt.desc, tt.source, tt.fmt)
		res := c.Classify(ev)
		if res.Confidence < 0.0 || res.Confidence > 1.0 {
			t.Errorf("%q: confidence %f out of [0,1]", tt.title, res.Confidence)
		}
		if res.TierLabel == "" {
			t.Errorf("%q: empty tier label", tt.title)
		}
		if res.Reasoning == "" {
			t.Errorf("%q: empty reasoning", tt.title)
		}
	}
}

func TestClassify_AllTiersExistAsConstants(t *testing.T) {
	tiers := []models.EvidenceTier{
		models.TierL1Policy,
		models.TierL2Standard,
		models.TierL3Guideline,
		models.TierL4Procedure,
		models.TierL5Record,
		models.TierL6Telemetry,
	}
	expectedLabels := []string{
		"L1-Policy", "L2-Standard", "L3-Guideline", "L4-Procedure", "L5-Record", "L6-Telemetry",
	}
	for i, tier := range tiers {
		if got := tier.String(); got != expectedLabels[i] {
			t.Errorf("Tier %d: expected label %q, got %q", tier, expectedLabels[i], got)
		}
	}
}

