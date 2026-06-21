package adapters_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/adapters"
	"github.com/davejduke/obvious/services/integration/internal/connector"
)

func TestSentinelAdapter_MockFetchLogs(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{
		WorkspaceID: "test-workspace-id",
		MockMode:    true,
	})

	logs, err := adapter.FetchLogs(context.Background(), connector.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Source != "sentinel" {
			t.Errorf("expected source=sentinel, got %s", l.Source)
		}
		if l.EventID == "" {
			t.Error("expected non-empty EventID")
		}
		if l.Severity == "" {
			t.Error("expected non-empty Severity")
		}
	}
}

func TestSentinelAdapter_MockHealth(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{
		MockMode: true,
	})
	status := adapter.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "sentinel" {
		t.Errorf("expected connector=sentinel, got %s", status.Connector)
	}
}

func TestSentinelAdapter_DefaultLimit(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{MockMode: true})
	logs, _ := adapter.FetchLogs(context.Background(), connector.QueryOptions{})
	if len(logs) == 0 {
		t.Error("expected default mock logs")
	}
}

func TestSplunkAdapter_MockFetchLogs(t *testing.T) {
	adapter := adapters.NewSplunkAdapter(adapters.SplunkConfig{
		BaseURL:     "http://localhost:8089",
		Token:       "test-token",
		SavedSearch: "NIS2 Security Events",
		MockMode:    true,
	})

	logs, err := adapter.FetchLogs(context.Background(), connector.QueryOptions{Limit: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 7 {
		t.Errorf("expected 7 logs, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Source != "splunk" {
			t.Errorf("expected source=splunk, got %s", l.Source)
		}
		if l.EventID == "" {
			t.Error("expected non-empty EventID")
		}
	}
}

func TestSplunkAdapter_MockHealth(t *testing.T) {
	adapter := adapters.NewSplunkAdapter(adapters.SplunkConfig{MockMode: true})
	status := adapter.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "splunk" {
		t.Errorf("expected connector=splunk, got %s", status.Connector)
	}
}

func TestSplunkAdapter_Name(t *testing.T) {
	adapter := adapters.NewSplunkAdapter(adapters.SplunkConfig{})
	if adapter.Name() != "splunk" {
		t.Errorf("expected name=splunk, got %s", adapter.Name())
	}
}

func TestSentinelAdapter_Name(t *testing.T) {
	adapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{})
	if adapter.Name() != "sentinel" {
		t.Errorf("expected name=sentinel, got %s", adapter.Name())
	}
}


// ────────────────────────────────────────────────────────────────────────────
// AWS Security Hub adapter tests
// ────────────────────────────────────────────────────────────────────────────

func TestAWSSecurityHubAdapter_Name(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{})
	if a.Name() != "aws-security-hub" {
		t.Errorf("expected name=aws-security-hub, got %s", a.Name())
	}
}

func TestAWSSecurityHubAdapter_MockHealth(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{
		MockMode: true,
	})
	status := a.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "aws-security-hub" {
		t.Errorf("expected connector=aws-security-hub, got %s", status.Connector)
	}
}

func TestAWSSecurityHubAdapter_MockFetchLogs(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{
		Region:    "us-east-1",
		AccountID: "123456789012",
		MockMode:  true,
	})

	logs, err := a.FetchLogs(context.Background(), connector.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 5 {
		t.Errorf("expected 5 logs, got %d", len(logs))
	}
	for _, l := range logs {
		if l.Source != "aws-security-hub" {
			t.Errorf("expected source=aws-security-hub, got %s", l.Source)
		}
		if l.EventID == "" {
			t.Error("expected non-empty EventID")
		}
		if l.Severity == "" {
			t.Error("expected non-empty Severity")
		}
		if l.Title == "" {
			t.Error("expected non-empty Title")
		}
		if l.RawData["account_id"] == "" {
			t.Error("expected non-empty account_id in RawData")
		}
		if l.RawData["region"] == "" {
			t.Error("expected non-empty region in RawData")
		}
	}
}

func TestAWSSecurityHubAdapter_MockFetchFindings(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{
		Region:    "us-west-2",
		AccountID: "999888777666",
		MockMode:  true,
	})

	findings, err := a.FetchFindings(context.Background(), connector.QueryOptions{Limit: 8})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(findings) != 8 {
		t.Errorf("expected 8 findings, got %d", len(findings))
	}
	for _, f := range findings {
		if f.SchemaVersion != "2018-10-08" {
			t.Errorf("expected SchemaVersion=2018-10-08, got %s", f.SchemaVersion)
		}
		if f.ID == "" {
			t.Error("expected non-empty finding ID")
		}
		if f.AwsAccountID != "999888777666" {
			t.Errorf("expected account=999888777666, got %s", f.AwsAccountID)
		}
		if f.Region != "us-west-2" {
			t.Errorf("expected region=us-west-2, got %s", f.Region)
		}
		if f.Severity.Label == "" {
			t.Error("expected non-empty severity label")
		}
		if len(f.Resources) == 0 {
			t.Error("expected at least one resource")
		}
	}
}

func TestAWSSecurityHubAdapter_DefaultLimit(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{MockMode: true})
	logs, err := a.FetchLogs(context.Background(), connector.QueryOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) == 0 {
		t.Error("expected default mock findings")
	}
}

func TestAWSSecurityHubAdapter_FetchEvidenceItems(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{
		Region:    "us-east-1",
		AccountID: "123456789012",
		MockMode:  true,
	})

	items, err := a.FetchEvidenceItems(context.Background(), connector.QueryOptions{Limit: 5})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 5 {
		t.Errorf("expected 5 evidence items, got %d", len(items))
	}
	for _, item := range items {
		if item.SourceConnector != "aws-security-hub" {
			t.Errorf("expected source_connector=aws-security-hub, got %s", item.SourceConnector)
		}
		if item.SourceType != "api_integration" {
			t.Errorf("expected source_type=api_integration, got %s", item.SourceType)
		}
		if item.ContentFormat != "json" {
			t.Errorf("expected content_format=json, got %s", item.ContentFormat)
		}
		if item.Tier != adapters.EvidenceTierL6Telemetry {
			t.Errorf("expected tier=L6Telemetry, got %d", item.Tier)
		}
		if item.RiskLevel == "" {
			t.Error("expected non-empty risk_level")
		}
		if item.QualityScore.AggregateScore <= 0 {
			t.Errorf("expected positive aggregate quality score, got %f", item.QualityScore.AggregateScore)
		}
		if item.AWSAccountID == "" {
			t.Error("expected non-empty aws_account_id")
		}
		if len(item.Tags) == 0 {
			t.Error("expected non-empty tags")
		}
	}
}

func TestAWSSecurityHubAdapter_FetchComplianceResults(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{
		Region:   "us-east-1",
		MockMode: true,
	})

	summaries, err := a.FetchComplianceResults(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(summaries) == 0 {
		t.Error("expected at least one compliance standard summary")
	}
	for _, s := range summaries {
		if s.Standard == "" {
			t.Error("expected non-empty standard name")
		}
		if s.TotalControls == 0 {
			t.Errorf("standard %s: expected at least one control", s.Standard)
		}
		if s.Score < 0 || s.Score > 1 {
			t.Errorf("standard %s: score %f out of [0,1] range", s.Standard, s.Score)
		}
	}
}

func TestAWSSecurityHubAdapter_FetchSecurityScore(t *testing.T) {
	a := adapters.NewAWSSecurityHubAdapter(adapters.AWSSecurityHubConfig{
		Region:    "us-east-1",
		AccountID: "123456789012",
		MockMode:  true,
	})

	score, err := a.FetchSecurityScore(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if score.AWSAccountID != "123456789012" {
		t.Errorf("expected account=123456789012, got %s", score.AWSAccountID)
	}
	if score.TotalFindings == 0 {
		t.Error("expected non-zero total findings")
	}
	if score.OverallScore < 0 || score.OverallScore > 100 {
		t.Errorf("overall score %f out of [0,100] range", score.OverallScore)
	}
	if len(score.ActiveStandards) == 0 {
		t.Error("expected at least one active standard")
	}
}

// ────────────────────────────────────────────────────────────────────────────
// ASFF normalizer tests
// ────────────────────────────────────────────────────────────────────────────

func TestMapASFFSeverity_AllLabels(t *testing.T) {
	cases := []struct {
		input    adapters.ASFFSeverityLabel
		expected adapters.AIAuditorRiskLevel
	}{
		{adapters.ASFFSeverityCritical, adapters.RiskLevelCritical},
		{adapters.ASFFSeverityHigh, adapters.RiskLevelHigh},
		{adapters.ASFFSeverityMedium, adapters.RiskLevelMedium},
		{adapters.ASFFSeverityLow, adapters.RiskLevelLow},
		{adapters.ASFFSeverityInformational, adapters.RiskLevelInformational},
		{"UNKNOWN", adapters.RiskLevelInformational}, // unknown falls back to informational
	}
	for _, c := range cases {
		got := adapters.MapASFFSeverity(c.input)
		if got != c.expected {
			t.Errorf("MapASFFSeverity(%s): expected %s, got %s", c.input, c.expected, got)
		}
	}
}

func TestMapNormalizedSeverity(t *testing.T) {
	cases := []struct {
		score    int
		expected adapters.AIAuditorRiskLevel
	}{
		{100, adapters.RiskLevelCritical},
		{90, adapters.RiskLevelCritical},
		{89, adapters.RiskLevelHigh},
		{70, adapters.RiskLevelHigh},
		{69, adapters.RiskLevelMedium},
		{40, adapters.RiskLevelMedium},
		{39, adapters.RiskLevelLow},
		{1, adapters.RiskLevelLow},
		{0, adapters.RiskLevelInformational},
	}
	for _, c := range cases {
		got := adapters.MapNormalizedSeverity(c.score)
		if got != c.expected {
			t.Errorf("MapNormalizedSeverity(%d): expected %s, got %s", c.score, c.expected, got)
		}
	}
}

func TestNormalizeResourceType(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"AwsEc2Instance", "EC2"},
		{"AwsEc2SecurityGroup", "EC2"},
		{"AwsS3Bucket", "S3"},
		{"AwsIamUser", "IAM"},
		{"AwsIamPasswordPolicy", "IAM"},
		{"AwsRdsDbInstance", "RDS"},
		{"AwsLambdaFunction", "Lambda"},
		{"AwsKmsKey", "KMS"},
		{"AwsCloudTrailTrail", "CloudTrail"},
		{"AwsVpcNetworkAcl", "VPC"},
		{"AwsSqsQueue", "SQS"},
		{"AwsSnsTopicSubscription", "SNS"},
		{"CustomResourceType", "CustomResourceType"},
		{"", "Unknown"},
	}
	for _, c := range cases {
		got := adapters.NormalizeResourceType(c.input)
		if got != c.expected {
			t.Errorf("NormalizeResourceType(%q): expected %q, got %q", c.input, c.expected, got)
		}
	}
}

func TestASFFNormalizer_NormalizeFinding_Full(t *testing.T) {
	now := time.Now().UTC()
	n := adapters.NewASFFNormalizer()

	f := adapters.ASFFinding{
		SchemaVersion: "2018-10-08",
		ID:            "arn:aws:securityhub:us-east-1:123456789012:subscription/aws-foundational-security-best-practices/v/1.0.0/IAM.1/finding/abc123",
		ProductArn:    "arn:aws:securityhub:us-east-1::product/aws/securityhub",
		ProductName:   "Security Hub",
		GeneratorID:   "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/1.1",
		AwsAccountID:  "123456789012",
		Region:        "us-east-1",
		Types:         []string{"Software and Configuration Checks/AWS Security Best Practices/IAM"},
		FirstObservedAt: now.Add(-48 * time.Hour),
		LastObservedAt:  now.Add(-1 * time.Hour),
		CreatedAt:       now.Add(-48 * time.Hour),
		UpdatedAt:       now.Add(-1 * time.Hour),
		Severity: adapters.ASFFSeverity{
			Label:      adapters.ASFFSeverityCritical,
			Normalized: 95,
		},
		Title:       "IAM root account MFA not enabled",
		Description: "The root account does not have MFA enabled. This is a critical security risk.",
		Remediation: adapters.ASFFRemediation{
			Recommendation: adapters.ASFFRemediationRecommendation{
				Text: "Enable MFA on the root account.",
				URL:  "https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_mfa.html",
			},
		},
		Resources: []adapters.ASFFResource{
			{
				Type:   "AwsIamUser",
				ID:     "arn:aws:iam::123456789012:root",
				Region: "us-east-1",
			},
		},
		Compliance: &adapters.ASFFCompliance{
			Status:              adapters.ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS AWS Foundations 1.1", "PCI DSS v3.2.1/8.3.1"},
			SecurityControlID:   "IAM.1",
			AssociatedStandards: []adapters.ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1"},
			},
		},
		WorkflowStatus: adapters.ASFFWorkflowNew,
		RecordState:    adapters.ASFFRecordActive,
	}

	item := n.NormalizeFinding(f)

	// Core identity
	if item.SourceConnector != "aws-security-hub" {
		t.Errorf("expected source_connector=aws-security-hub, got %s", item.SourceConnector)
	}
	if item.SourceFindingID != f.ID {
		t.Errorf("expected source_finding_id=%s, got %s", f.ID, item.SourceFindingID)
	}

	// Severity mapping
	if item.RiskLevel != adapters.RiskLevelCritical {
		t.Errorf("expected risk_level=critical, got %s", item.RiskLevel)
	}

	// Compliance fields
	if item.ComplianceStatus != "FAILED" {
		t.Errorf("expected compliance_status=FAILED, got %s", item.ComplianceStatus)
	}
	if len(item.RelatedControls) != 2 {
		t.Errorf("expected 2 related controls, got %d", len(item.RelatedControls))
	}
	if len(item.ComplianceStandards) != 2 {
		t.Errorf("expected 2 compliance standards, got %d", len(item.ComplianceStandards))
	}

	// Evidence tier
	if item.Tier != adapters.EvidenceTierL6Telemetry {
		t.Errorf("expected tier=6 (L6 Telemetry), got %d", item.Tier)
	}

	// Resource normalisation
	if len(item.AffectedResources) != 1 {
		t.Fatalf("expected 1 affected resource, got %d", len(item.AffectedResources))
	}
	if item.AffectedResources[0].Type != "IAM" {
		t.Errorf("expected resource type=IAM, got %s", item.AffectedResources[0].Type)
	}

	// Quality score sanity
	qs := item.QualityScore
	if qs.AggregateScore <= 0 || qs.AggregateScore > 1 {
		t.Errorf("aggregate score %f out of (0,1]", qs.AggregateScore)
	}
	if qs.SourceReliability < 0.90 {
		t.Errorf("expected source reliability >= 0.90 for API integration, got %f", qs.SourceReliability)
	}

	// Remediation
	if item.RemediationText == "" {
		t.Error("expected non-empty remediation text")
	}
	if item.RemediationURL == "" {
		t.Error("expected non-empty remediation URL")
	}

	// Tags
	if len(item.Tags) == 0 {
		t.Error("expected non-empty tags")
	}
	if !containsStr(item.Tags, "source:aws-security-hub") {
		t.Error("expected tag source:aws-security-hub")
	}
	if !containsStr(item.Tags, "severity:critical") {
		t.Error("expected tag severity:critical")
	}
	if !containsStr(item.Tags, "resource:iam") {
		t.Error("expected tag resource:iam")
	}
}

func TestASFFNormalizer_NormalizeFinding_MinimalFinding(t *testing.T) {
	n := adapters.NewASFFNormalizer()
	// Minimal finding with no compliance block.
	f := adapters.ASFFinding{
		SchemaVersion: "2018-10-08",
		ID:            "arn:aws:securityhub:us-east-1:000000000000:finding/minimal",
		GeneratorID:   "test-generator",
		AwsAccountID:  "000000000000",
		Severity: adapters.ASFFSeverity{
			Label:      adapters.ASFFSeverityLow,
			Normalized: 10,
		},
		Title:       "Minimal test finding",
		Description: "A minimal finding with no compliance context.",
		RecordState: adapters.ASFFRecordActive,
	}

	item := n.NormalizeFinding(f)

	if item.RiskLevel != adapters.RiskLevelLow {
		t.Errorf("expected risk_level=low, got %s", item.RiskLevel)
	}
	if item.ComplianceStatus != "" {
		t.Errorf("expected empty compliance_status for finding without compliance block, got %s", item.ComplianceStatus)
	}
	// Score should still be computed.
	if item.QualityScore.AggregateScore <= 0 {
		t.Error("expected positive aggregate score even for minimal finding")
	}
}

func TestASFFNormalizer_NormalizeFindings_Batch(t *testing.T) {
	n := adapters.NewASFFNormalizer()
	now := time.Now().UTC()
	findings := make([]adapters.ASFFinding, 5)
	for i := range findings {
		findings[i] = adapters.ASFFinding{
			SchemaVersion: "2018-10-08",
			ID:            fmt.Sprintf("arn:aws:securityhub:us-east-1:123456789012:finding/batch-%d", i),
			GeneratorID:   "batch-generator",
			AwsAccountID:  "123456789012",
			LastObservedAt: now,
			Severity: adapters.ASFFSeverity{
				Label: adapters.ASFFSeverityMedium,
			},
			Title:       fmt.Sprintf("Batch finding %d", i),
			Description: "Batch normalization test",
			RecordState: adapters.ASFFRecordActive,
		}
	}

	items := n.NormalizeFindings(findings)
	if len(items) != 5 {
		t.Errorf("expected 5 normalized items, got %d", len(items))
	}
	for i, item := range items {
		if item.RiskLevel != adapters.RiskLevelMedium {
			t.Errorf("item[%d]: expected risk_level=medium, got %s", i, item.RiskLevel)
		}
	}
}

func TestASFFNormalizer_QualityScore_HighQualityFinding(t *testing.T) {
	n := adapters.NewASFFNormalizer()
	now := time.Now().UTC()

	f := adapters.ASFFinding{
		SchemaVersion:   "2018-10-08",
		ID:              "arn:aws:securityhub:us-east-1:123456789012:finding/high-quality",
		GeneratorID:     "test-generator",
		AwsAccountID:    "123456789012",
		Region:          "us-east-1",
		LastObservedAt:  now.Add(-6 * time.Hour),
		Types:           []string{"Software and Configuration Checks/AWS Security Best Practices"},
		Severity: adapters.ASFFSeverity{
			Label:      adapters.ASFFSeverityHigh,
			Normalized: 70,
		},
		Title:       "Security group allows unrestricted access",
		Description: "This security group allows unrestricted inbound access on sensitive ports.",
		Remediation: adapters.ASFFRemediation{
			Recommendation: adapters.ASFFRemediationRecommendation{
				Text: "Restrict inbound access to specific CIDR ranges.",
				URL:  "https://docs.aws.amazon.com",
			},
		},
		Resources: []adapters.ASFFResource{
			{Type: "AwsEc2SecurityGroup", ID: "sg-12345", Region: "us-east-1"},
		},
		Compliance: &adapters.ASFFCompliance{
			Status:              adapters.ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS 4.1"},
			SecurityControlID:   "EC2.13",
			AssociatedStandards: []adapters.ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
			},
		},
		WorkflowStatus: adapters.ASFFWorkflowNew,
		RecordState:    adapters.ASFFRecordActive,
	}

	item := n.NormalizeFinding(f)
	qs := item.QualityScore

	// Full finding should have high completeness.
	if qs.CompletenessScore < 0.9 {
		t.Errorf("expected completeness >= 0.9, got %f", qs.CompletenessScore)
	}
	// Recent finding (6h ago) should score high on currency.
	if qs.CurrencyScore < 0.90 {
		t.Errorf("expected currency >= 0.90 for recent finding, got %f", qs.CurrencyScore)
	}
	// API integration should give high source reliability.
	if qs.SourceReliability < 0.90 {
		t.Errorf("expected source reliability >= 0.90, got %f", qs.SourceReliability)
	}
	// Compliance + remediation + types all present → full corroboration.
	if qs.CorroborationScore < 0.95 {
		t.Errorf("expected corroboration >= 0.95, got %f", qs.CorroborationScore)
	}
	// Aggregate should be well above 0.7.
	if qs.AggregateScore < 0.70 {
		t.Errorf("expected aggregate score >= 0.70, got %f", qs.AggregateScore)
	}
}

func TestASFFNormalizer_ResourceTypesCoverage(t *testing.T) {
	// Verify that all five required resource types (EC2, S3, IAM, RDS, Lambda)
	// are correctly handled by the normalizer.
	requiredTypes := map[string]string{
		"AwsEc2Instance":     "EC2",
		"AwsS3Bucket":        "S3",
		"AwsIamUser":         "IAM",
		"AwsRdsDbInstance":   "RDS",
		"AwsLambdaFunction": "Lambda",
	}
	for asffType, expected := range requiredTypes {
		got := adapters.NormalizeResourceType(asffType)
		if got != expected {
			t.Errorf("NormalizeResourceType(%q): expected %q, got %q", asffType, expected, got)
		}
	}
}

// containsStr is a test helper for tag assertions.
func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

