// Package adapters contains SIEM connector implementations.
// aws_security_hub.go: AWS Security Hub adapter with ASFF normalization.
package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/connector"
)

// AWSSecurityHubConfig holds AWS Security Hub connection settings.
type AWSSecurityHubConfig struct {
	// Region is the AWS region to query, e.g. "us-east-1".
	Region string
	// AccountID is the 12-digit AWS account ID.
	AccountID string
	// AccessKeyID is the AWS access key (populated from env/vault).
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key (populated from env/vault).
	SecretAccessKey string
	// SessionToken is optional STS session token.
	SessionToken string
	// BaseURL overrides the default Security Hub endpoint (for sandbox/mock).
	BaseURL string
	// MockMode returns synthetic ASFF data without making real AWS API calls.
	MockMode bool
}

// AWSSecurityHubAdapter fetches security findings from AWS Security Hub
// and normalizes them to the AIAUDITOR evidence model via ASFF.
//
// It implements connector.Connector for compatibility with the existing
// integration registry and circuit breaker infrastructure.
type AWSSecurityHubAdapter struct {
	config     AWSSecurityHubConfig
	client     *http.Client
	normalizer *ASFFNormalizer
}

// NewAWSSecurityHubAdapter creates a ready-to-use Security Hub connector.
func NewAWSSecurityHubAdapter(cfg AWSSecurityHubConfig) *AWSSecurityHubAdapter {
	return &AWSSecurityHubAdapter{
		config:     cfg,
		client:     &http.Client{Timeout: 30 * time.Second},
		normalizer: NewASFFNormalizer(),
	}
}

// Name implements connector.Connector.
func (a *AWSSecurityHubAdapter) Name() string { return "aws-security-hub" }

// FetchLogs implements connector.Connector. It maps ASFF findings to the
// generic LogEntry format used by the integration registry so the existing
// HTTP handler and circuit breaker transparently work.
//
// For structured ASFF findings use FetchFindings; for normalized evidence
// items use FetchEvidenceItems.
func (a *AWSSecurityHubAdapter) FetchLogs(ctx context.Context, opts connector.QueryOptions) ([]connector.LogEntry, error) {
	findings, err := a.FetchFindings(ctx, opts)
	if err != nil {
		return nil, err
	}
	return asffToLogEntries(findings), nil
}

// Health implements connector.Connector.
func (a *AWSSecurityHubAdapter) Health(ctx context.Context) connector.HealthStatus {
	status := connector.HealthStatus{
		Connector:   a.Name(),
		LastChecked: time.Now().UTC(),
	}
	if a.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}

	url := fmt.Sprintf("%s/findings?MaxResults=1", a.apiBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
	a.signRequest(req)
	resp, err := a.client.Do(req)
	if err != nil {
		status.Message = "connectivity error: " + err.Error()
		return status
	}
	resp.Body.Close()
	status.Healthy = resp.StatusCode < 500
	status.Message = fmt.Sprintf("HTTP %d", resp.StatusCode)
	return status
}

// FetchFindings returns raw ASFF findings from Security Hub.
// In mock mode it returns deterministic synthetic findings.
func (a *AWSSecurityHubAdapter) FetchFindings(ctx context.Context, opts connector.QueryOptions) ([]ASFFinding, error) {
	if a.config.MockMode {
		return a.mockFindings(opts), nil
	}
	return a.fetchFindingsFromAPI(ctx, opts)
}

// FetchEvidenceItems returns ASFF findings already normalized to the
// AIAUDITOR AuditEvidenceItem model.
func (a *AWSSecurityHubAdapter) FetchEvidenceItems(ctx context.Context, opts connector.QueryOptions) ([]AuditEvidenceItem, error) {
	findings, err := a.FetchFindings(ctx, opts)
	if err != nil {
		return nil, err
	}
	return a.normalizer.NormalizeFindings(findings), nil
}

// FetchComplianceResults returns compliance standard summaries derived from
// active ASFF findings.
func (a *AWSSecurityHubAdapter) FetchComplianceResults(ctx context.Context) ([]ComplianceStandardSummary, error) {
	findings, err := a.FetchFindings(ctx, connector.QueryOptions{Limit: 100})
	if err != nil {
		return nil, err
	}
	return buildComplianceSummaries(findings), nil
}

// FetchSecurityScore returns the aggregated security posture score.
func (a *AWSSecurityHubAdapter) FetchSecurityScore(ctx context.Context) (SecurityScore, error) {
	findings, err := a.FetchFindings(ctx, connector.QueryOptions{Limit: 100})
	if err != nil {
		return SecurityScore{}, err
	}
	return calculateSecurityScore(a.config.AccountID, a.config.Region, findings), nil
}

// ────────────────────────────────────────────────────────────────────────────
// Real API (non-mock) implementation
// ────────────────────────────────────────────────────────────────────────────

func (a *AWSSecurityHubAdapter) fetchFindingsFromAPI(ctx context.Context, opts connector.QueryOptions) ([]ASFFinding, error) {
	limit := hubEffectiveLimit(opts)
	url := fmt.Sprintf("%s/findings?MaxResults=%d", a.apiBase(), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("aws-security-hub: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	a.signRequest(req)

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("aws-security-hub: request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("aws-security-hub: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("aws-security-hub: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Findings []ASFFinding `json:"Findings"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("aws-security-hub: parse response: %w", err)
	}
	return result.Findings, nil
}

// signRequest attaches AWS credentials to the request.
// In a production implementation this would use SigV4 signing.
// For the mock/stub adapter the method is a no-op.
func (a *AWSSecurityHubAdapter) signRequest(req *http.Request) {
	if a.config.AccessKeyID != "" {
		// Placeholder: real implementation would compute an AWS SigV4 Authorization header.
		req.Header.Set("X-Amz-Security-Token", a.config.SessionToken)
	}
}

func (a *AWSSecurityHubAdapter) apiBase() string {
	if a.config.BaseURL != "" {
		return a.config.BaseURL
	}
	return fmt.Sprintf("https://securityhub.%s.amazonaws.com", a.config.Region)
}

// ────────────────────────────────────────────────────────────────────────────
// Mock data — realistic ASFF findings covering all required resource types
// and compliance standards (CIS Benchmarks, PCI DSS, SOC 2).
// ────────────────────────────────────────────────────────────────────────────

var mockFindingTemplates = []struct {
	title       string
	description string
	severity    ASFFSeverityLabel
	normalized  int
	resType     string
	resID       string
	generatorID string
	findings    []string // related finding types
	compliance  *ASFFCompliance
}{
	{
		title:       "IAM root account MFA not enabled",
		description: "The root account does not have MFA enabled. Enabling MFA for the root account adds an extra layer of protection for access.",
		severity:    ASFFSeverityCritical,
		normalized:  95,
		resType:     "AwsIamUser",
		resID:       "arn:aws:iam::123456789012:root",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/1.1",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/IAM"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS AWS Foundations 1.1", "PCI DSS v3.2.1/8.3.1"},
			SecurityControlID:   "IAM.1",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1"},
			},
		},
	},
	{
		title:       "S3 bucket does not have server-side encryption enabled",
		description: "This control checks that your S3 bucket either has Amazon S3-managed encryption enabled or uses customer-managed keys.",
		severity:    ASFFSeverityHigh,
		normalized:  70,
		resType:     "AwsS3Bucket",
		resID:       "arn:aws:s3:::my-sensitive-data-bucket",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/2.7",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/S3"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS AWS Foundations 2.7", "PCI DSS v3.2.1/3.4", "SOC 2 CC6.1"},
			SecurityControlID:   "S3.4",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	},
	{
		title:       "EC2 security group allows unrestricted SSH access",
		description: "This control checks whether the EC2 security group disallows unrestricted incoming SSH traffic. Unrestricted access (0.0.0.0/0) increases the opportunity for malicious activity.",
		severity:    ASFFSeverityHigh,
		normalized:  75,
		resType:     "AwsEc2SecurityGroup",
		resID:       "arn:aws:ec2:us-east-1:123456789012:security-group/sg-0a1b2c3d4e5f67890",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/4.1",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/EC2"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS AWS Foundations 4.1", "PCI DSS v3.2.1/1.2.1"},
			SecurityControlID:   "EC2.13",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1"},
			},
		},
	},
	{
		title:       "RDS instance is not encrypted at rest",
		description: "This control checks whether storage encryption is enabled for your RDS DB instance. RDS DB instances should have encryption at rest enabled to protect your data.",
		severity:    ASFFSeverityMedium,
		normalized:  50,
		resType:     "AwsRdsDbInstance",
		resID:       "arn:aws:rds:us-east-1:123456789012:db:my-prod-db",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/2.3",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/RDS"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"PCI DSS v3.2.1/3.4", "SOC 2 CC6.1"},
			SecurityControlID:   "RDS.3",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	},
	{
		title:       "Lambda function environment variables contain secrets",
		description: "This control checks that Lambda function environment variables do not contain AWS credentials. Storing plaintext credentials in Lambda environment variables risks credential exposure.",
		severity:    ASFFSeverityHigh,
		normalized:  70,
		resType:     "AwsLambdaFunction",
		resID:       "arn:aws:lambda:us-east-1:123456789012:function:my-data-processor",
		generatorID: "arn:aws:securityhub:us-east-1:123456789012:security-control/Lambda.3",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/Lambda"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"SOC 2 CC6.7"},
			SecurityControlID:   "Lambda.3",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	},
	{
		title:       "IAM password policy does not prevent password reuse",
		description: "The IAM account password policy should prevent the reuse of the last 24 passwords to reduce the risk of compromised credentials being reused.",
		severity:    ASFFSeverityMedium,
		normalized:  40,
		resType:     "AwsIamPasswordPolicy",
		resID:       "arn:aws:iam::123456789012:account-policy",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/1.10",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/IAM"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS AWS Foundations 1.10", "PCI DSS v3.2.1/8.2.5"},
			SecurityControlID:   "IAM.10",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1"},
			},
		},
	},
	{
		title:       "S3 bucket public read access is enabled",
		description: "This control checks whether S3 bucket policies and ACLs allow public read access. Public read access could expose sensitive data.",
		severity:    ASFFSeverityCritical,
		normalized:  90,
		resType:     "AwsS3Bucket",
		resID:       "arn:aws:s3:::my-public-facing-assets",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/2.3",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/S3"},
		compliance: &ASFFCompliance{
			Status:              ASFFCompliancePassed,
			RelatedRequirements: []string{"CIS AWS Foundations 2.3", "SOC 2 CC6.1"},
			SecurityControlID:   "S3.2",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	},
	{
		title:       "CloudTrail is not enabled in all regions",
		description: "AWS CloudTrail should be enabled and configured with at least one multi-Region trail to ensure that API activity is monitored across all regions.",
		severity:    ASFFSeverityHigh,
		normalized:  75,
		resType:     "AwsCloudTrailTrail",
		resID:       "arn:aws:cloudtrail:us-east-1:123456789012:trail/management-events-trail",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/2.1",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/CloudTrail"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS AWS Foundations 2.1", "PCI DSS v3.2.1/10.1", "SOC 2 CC7.2"},
			SecurityControlID:   "CloudTrail.1",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/pci-dss/v/3.2.1"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	},
	{
		title:       "KMS key rotation is not enabled",
		description: "AWS KMS customer managed keys should be rotated automatically. Key rotation reduces the risk that a compromised key is used to decrypt data.",
		severity:    ASFFSeverityMedium,
		normalized:  40,
		resType:     "AwsKmsKey",
		resID:       "arn:aws:kms:us-east-1:123456789012:key/0a1b2c3d-4e5f-6789-abcd-ef0123456789",
		generatorID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0/rule/2.8",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/KMS"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceFailed,
			RelatedRequirements: []string{"CIS AWS Foundations 2.8", "SOC 2 CC6.7"},
			SecurityControlID:   "KMS.4",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:::ruleset/cis-aws-foundations-benchmark/v/1.2.0"},
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	},
	{
		title:       "EC2 instance is using a deprecated AMI",
		description: "This control checks whether an EC2 instance is running a deprecated AMI. Deprecated AMIs may contain unpatched vulnerabilities.",
		severity:    ASFFSeverityLow,
		normalized:  25,
		resType:     "AwsEc2Instance",
		resID:       "arn:aws:ec2:us-east-1:123456789012:instance/i-0a1b2c3d4e5f6789a",
		generatorID: "arn:aws:securityhub:us-east-1:123456789012:security-control/EC2.166",
		findings:    []string{"Software and Configuration Checks/AWS Security Best Practices/EC2"},
		compliance: &ASFFCompliance{
			Status:              ASFFComplianceWarning,
			RelatedRequirements: []string{"SOC 2 CC7.1"},
			SecurityControlID:   "EC2.166",
			AssociatedStandards: []ASFFAssociatedStandard{
				{StandardsID: "arn:aws:securityhub:us-east-1::standards/aws-foundational-security-best-practices/v/1.0.0"},
			},
		},
	},
}

// mockFindings returns deterministic synthetic ASFF findings.
func (a *AWSSecurityHubAdapter) mockFindings(opts connector.QueryOptions) []ASFFinding {
	limit := hubEffectiveLimit(opts)
	now := time.Now().UTC()
	account := a.config.AccountID
	if account == "" {
		account = "123456789012"
	}
	region := a.config.Region
	if region == "" {
		region = "us-east-1"
	}

	findings := make([]ASFFinding, 0, limit)
	templates := mockFindingTemplates
	for i := 0; i < limit; i++ {
		t := templates[i%len(templates)]
		findingID := fmt.Sprintf(
			"arn:aws:securityhub:%s:%s:subscription/aws-foundational-security-best-practices/v/1.0.0/%s/finding/MOCK-%04d",
			region, account, t.generatorID[len(t.generatorID)-6:], i+1,
		)
		findings = append(findings, ASFFinding{
			SchemaVersion:  "2018-10-08",
			ID:             findingID,
			ProductArn:     fmt.Sprintf("arn:aws:securityhub:%s::product/aws/securityhub", region),
			ProductName:    "Security Hub",
			CompanyName:    "AWS",
			GeneratorID:    t.generatorID,
			AwsAccountID:   account,
			Region:         region,
			Types:          t.findings,
			FirstObservedAt: now.Add(-time.Duration(i+7) * 24 * time.Hour),
			LastObservedAt:  now.Add(-time.Duration(i) * time.Hour),
			CreatedAt:       now.Add(-time.Duration(i+7) * 24 * time.Hour),
			UpdatedAt:       now.Add(-time.Duration(i) * time.Hour),
			Severity: ASFFSeverity{
				Label:      t.severity,
				Normalized: t.normalized,
			},
			Title:       t.title,
			Description: t.description,
			Remediation: ASFFRemediation{
				Recommendation: ASFFRemediationRecommendation{
					Text: fmt.Sprintf("Follow AWS Security Hub guidance for control %s", t.generatorID[len(t.generatorID)-6:]),
					URL:  "https://docs.aws.amazon.com/securityhub/latest/userguide/securityhub-controls-reference.html",
				},
			},
			Resources: []ASFFResource{
				{
					Type:      t.resType,
					ID:        t.resID,
					Region:    region,
					Partition: "aws",
				},
			},
			Compliance:     t.compliance,
			WorkflowStatus: ASFFWorkflowNew,
			RecordState:    ASFFRecordActive,
		})
	}
	return findings
}

// ────────────────────────────────────────────────────────────────────────────
// Aggregation helpers
// ────────────────────────────────────────────────────────────────────────────

// buildComplianceSummaries aggregates ASFF findings into per-standard summaries.
func buildComplianceSummaries(findings []ASFFinding) []ComplianceStandardSummary {
	// Group control results by standard.
	type controlKey struct {
		standard ComplianceStandard
		controlID string
	}
	type controlEntry struct {
		title    string
		statuses []ASFFComplianceStatus
		risk     ASFFSeverityLabel
		ids      []string
		ts       time.Time
	}
	controlMap := make(map[controlKey]*controlEntry)

	for _, f := range findings {
		if f.Compliance == nil {
			continue
		}
		for _, std := range f.Compliance.AssociatedStandards {
			stdName := ComplianceStandard(friendlyStandardName(std.StandardsID))
			key := controlKey{
				standard:  stdName,
				controlID: f.Compliance.SecurityControlID,
			}
			if key.controlID == "" {
				key.controlID = f.GeneratorID
			}
			e, ok := controlMap[key]
			if !ok {
				e = &controlEntry{
					title: f.Title,
					risk:  f.Severity.Label,
					ts:    f.LastObservedAt,
				}
				controlMap[key] = e
			}
			e.statuses = append(e.statuses, f.Compliance.Status)
			e.ids = append(e.ids, f.ID)
			if f.LastObservedAt.After(e.ts) {
				e.ts = f.LastObservedAt
			}
		}
	}

	// Aggregate per standard.
	type stdEntry struct {
		passed, failed, warning int
		controls                []ComplianceControlResult
	}
	stdMap := make(map[ComplianceStandard]*stdEntry)

	for key, entry := range controlMap {
		se, ok := stdMap[key.standard]
		if !ok {
			se = &stdEntry{}
			stdMap[key.standard] = se
		}
		// Aggregate statuses for this control across findings.
		status := aggregateControlStatus(entry.statuses)
		switch status {
		case ASFFCompliancePassed:
			se.passed++
		case ASFFComplianceFailed:
			se.failed++
		default:
			se.warning++
		}
		se.controls = append(se.controls, ComplianceControlResult{
			ControlID:    key.controlID,
			ControlTitle: entry.title,
			Standard:     key.standard,
			Status:       status,
			FindingCount: len(entry.ids),
			Findings:     entry.ids,
			RiskLevel:    MapASFFSeverity(entry.risk),
			LastChecked:  entry.ts,
		})
	}

	now := time.Now().UTC()
	summaries := make([]ComplianceStandardSummary, 0, len(stdMap))
	for std, se := range stdMap {
		total := se.passed + se.failed + se.warning
		var score float64
		if total > 0 {
			score = float64(se.passed) / float64(total)
		}
		summaries = append(summaries, ComplianceStandardSummary{
			Standard:        std,
			TotalControls:   total,
			PassedControls:  se.passed,
			FailedControls:  se.failed,
			WarningControls: se.warning,
			Score:           scoreRound(score),
			Controls:        se.controls,
			GeneratedAt:     now,
		})
	}
	return summaries
}

// aggregateControlStatus reduces multiple statuses for the same control to one:
// FAILED > WARNING > PASSED > NOT_AVAILABLE.
func aggregateControlStatus(statuses []ASFFComplianceStatus) ASFFComplianceStatus {
	result := ASFFComplianceNotAvailable
	for _, s := range statuses {
		switch s {
		case ASFFComplianceFailed:
			return ASFFComplianceFailed // short-circuit
		case ASFFComplianceWarning:
			result = ASFFComplianceWarning
		case ASFFCompliancePassed:
			if result == ASFFComplianceNotAvailable {
				result = ASFFCompliancePassed
			}
		}
	}
	return result
}

// calculateSecurityScore computes the aggregated security posture score.
// Score = 100 * (1 - weightedFailures/totalWeight) where CRITICAL=10, HIGH=5, MEDIUM=2, LOW=1.
func calculateSecurityScore(accountID, region string, findings []ASFFinding) SecurityScore {
	score := SecurityScore{
		AWSAccountID: accountID,
		Region:       region,
		CalculatedAt: time.Now().UTC(),
	}
	standardsSet := make(map[string]struct{})

	var totalWeight, failedWeight float64
	for _, f := range findings {
		if f.RecordState == ASFFRecordArchived {
			continue
		}
		score.TotalFindings++
		var w float64
		switch f.Severity.Label {
		case ASFFSeverityCritical:
			score.CriticalFindings++
			w = 10
		case ASFFSeverityHigh:
			score.HighFindings++
			w = 5
		case ASFFSeverityMedium:
			score.MediumFindings++
			w = 2
		case ASFFSeverityLow:
			score.LowFindings++
			w = 1
		default:
			score.InfoFindings++
			w = 0
		}
		totalWeight += w
		// Only count FAILED findings against the score.
		if f.Compliance != nil && f.Compliance.Status == ASFFComplianceFailed {
			failedWeight += w
		}
		if f.Compliance != nil {
			for _, s := range f.Compliance.AssociatedStandards {
				standardsSet[friendlyStandardName(s.StandardsID)] = struct{}{}
			}
		}
	}

	if totalWeight > 0 {
		score.OverallScore = scoreRound(100 * (1 - failedWeight/totalWeight))
	} else {
		score.OverallScore = 100
	}
	for std := range standardsSet {
		score.ActiveStandards = append(score.ActiveStandards, std)
	}
	return score
}

// asffToLogEntries maps ASFF findings to the generic LogEntry so the existing
// HTTP handler and circuit breaker continue to work unchanged.
func asffToLogEntries(findings []ASFFinding) []connector.LogEntry {
	entries := make([]connector.LogEntry, 0, len(findings))
	for _, f := range findings {
		rawData := map[string]string{
			"account_id":    f.AwsAccountID,
			"region":        f.Region,
			"generator_id":  f.GeneratorID,
			"record_state":  string(f.RecordState),
			"workflow":      string(f.WorkflowStatus),
			"product":       f.ProductName,
			"schema_version": f.SchemaVersion,
		}
		if len(f.Resources) > 0 {
			rawData["resource_type"] = NormalizeResourceType(f.Resources[0].Type)
			rawData["resource_id"] = f.Resources[0].ID
		}
		if f.Compliance != nil {
			rawData["compliance_status"] = string(f.Compliance.Status)
		}

		entries = append(entries, connector.LogEntry{
			Timestamp:   f.LastObservedAt,
			Source:      "aws-security-hub",
			Severity:    strings.ToLower(string(f.Severity.Label)),
			EventID:     f.ID,
			Title:       f.Title,
			Description: f.Description,
			RawData:     rawData,
		})
	}
	return entries
}

func hubEffectiveLimit(opts connector.QueryOptions) int {
	if opts.Limit > 0 && opts.Limit <= 100 {
		return opts.Limit
	}
	return 10
}

