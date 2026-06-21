// Package adapters contains SIEM connector implementations.
// asff_normalizer.go: ASFF (AWS Security Finding Format) normalization to AIAUDITOR evidence model.
package adapters

import (
	"fmt"
	"strings"
	"time"
)

// ────────────────────────────────────────────────────────────────────────────
// ASFF finding types (AWS Security Finding Format – schema 2018-10-08)
// ────────────────────────────────────────────────────────────────────────────

// ASFFSeverityLabel mirrors the ASFF severity label enum.
type ASFFSeverityLabel string

const (
	ASFFSeverityCritical      ASFFSeverityLabel = "CRITICAL"
	ASFFSeverityHigh          ASFFSeverityLabel = "HIGH"
	ASFFSeverityMedium        ASFFSeverityLabel = "MEDIUM"
	ASFFSeverityLow           ASFFSeverityLabel = "LOW"
	ASFFSeverityInformational ASFFSeverityLabel = "INFORMATIONAL"
)

// ASFFComplianceStatus represents the ASFF compliance status field.
type ASFFComplianceStatus string

const (
	ASFFCompliancePassed       ASFFComplianceStatus = "PASSED"
	ASFFComplianceFailed       ASFFComplianceStatus = "FAILED"
	ASFFComplianceWarning      ASFFComplianceStatus = "WARNING"
	ASFFComplianceNotAvailable ASFFComplianceStatus = "NOT_AVAILABLE"
)

// ASFFWorkflowStatus represents finding workflow state.
type ASFFWorkflowStatus string

const (
	ASFFWorkflowNew        ASFFWorkflowStatus = "NEW"
	ASFFWorkflowAssigned   ASFFWorkflowStatus = "ASSIGNED"
	ASFFWorkflowInProgress ASFFWorkflowStatus = "IN_PROGRESS"
	ASFFWorkflowResolved   ASFFWorkflowStatus = "RESOLVED"
)

// ASFFRecordState describes whether the finding is active or archived.
type ASFFRecordState string

const (
	ASFFRecordActive   ASFFRecordState = "ACTIVE"
	ASFFRecordArchived ASFFRecordState = "ARCHIVED"
)

// ASFFSeverity is the ASFF severity object embedded in a finding.
type ASFFSeverity struct {
	Label      ASFFSeverityLabel `json:"Label"`
	Normalized int               `json:"Normalized"` // 0-100
	Original   string            `json:"Original,omitempty"`
}

// ASFFResource describes a cloud resource associated with a finding.
type ASFFResource struct {
	Type      string                 `json:"Type"`                 // e.g. "AwsEc2Instance"
	ID        string                 `json:"Id"`                   // ARN or other identifier
	Region    string                 `json:"Region"`               // e.g. "us-east-1"
	Partition string                 `json:"Partition,omitempty"`  // e.g. "aws"
	Details   map[string]interface{} `json:"Details,omitempty"`
}

// ASFFRemediation holds remediation guidance.
type ASFFRemediation struct {
	Recommendation ASFFRemediationRecommendation `json:"Recommendation,omitempty"`
}

// ASFFRemediationRecommendation is the recommendation nested struct.
type ASFFRemediationRecommendation struct {
	Text string `json:"Text,omitempty"`
	URL  string `json:"Url,omitempty"`
}

// ASFFAssociatedStandard links a finding to a compliance framework.
type ASFFAssociatedStandard struct {
	StandardsID string `json:"StandardsId"` // e.g. "arn:aws:securityhub:...cis-aws-foundations-benchmark"
}

// ASFFCompliance is the ASFF compliance block.
type ASFFCompliance struct {
	Status              ASFFComplianceStatus    `json:"Status"`
	RelatedRequirements []string                `json:"RelatedRequirements,omitempty"`
	SecurityControlID   string                  `json:"SecurityControlId,omitempty"`
	AssociatedStandards []ASFFAssociatedStandard `json:"AssociatedStandards,omitempty"`
}

// ASFFNote holds analyst notes on a finding.
type ASFFNote struct {
	Text      string `json:"Text"`
	UpdatedBy string `json:"UpdatedBy"`
	UpdatedAt string `json:"UpdatedAt"`
}

// ASFFGeneratorDetails holds information about the rule that generated the finding.
type ASFFGeneratorDetails struct {
	Name        string   `json:"Name,omitempty"`
	Description string   `json:"Description,omitempty"`
	Tags        []string `json:"Tags,omitempty"`
}

// ASFFinding is a single AWS Security Finding Format record.
// Follows schema version 2018-10-08.
type ASFFinding struct {
	SchemaVersion        string                 `json:"SchemaVersion"`              // "2018-10-08"
	ID                   string                 `json:"Id"`                         // unique finding ARN
	ProductArn           string                 `json:"ProductArn"`                 // product ARN
	ProductName          string                 `json:"ProductName,omitempty"`
	CompanyName          string                 `json:"CompanyName,omitempty"`
	GeneratorID          string                 `json:"GeneratorId"`                // rule/detector ID
	GeneratorDetails     *ASFFGeneratorDetails  `json:"GeneratorDetails,omitempty"`
	AwsAccountID         string                 `json:"AwsAccountId"`               // 12-digit account ID
	Region               string                 `json:"Region,omitempty"`
	Types                []string               `json:"Types"`                      // finding type taxonomy
	FirstObservedAt      time.Time              `json:"FirstObservedAt"`
	LastObservedAt       time.Time              `json:"LastObservedAt"`
	CreatedAt            time.Time              `json:"CreatedAt"`
	UpdatedAt            time.Time              `json:"UpdatedAt"`
	Severity             ASFFSeverity           `json:"Severity"`
	Title                string                 `json:"Title"`
	Description          string                 `json:"Description"`
	Remediation          ASFFRemediation        `json:"Remediation,omitempty"`
	SourceURL            string                 `json:"SourceUrl,omitempty"`
	Resources            []ASFFResource         `json:"Resources"`
	Compliance           *ASFFCompliance        `json:"Compliance,omitempty"`
	WorkflowStatus       ASFFWorkflowStatus     `json:"WorkflowStatus,omitempty"`
	RecordState          ASFFRecordState        `json:"RecordState"`
	Note                 *ASFFNote              `json:"Note,omitempty"`
	FindingProviderFields map[string]interface{} `json:"FindingProviderFields,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// AIAUDITOR risk level
// ────────────────────────────────────────────────────────────────────────────

// AIAuditorRiskLevel maps to the AIAUDITOR risk rating enum.
type AIAuditorRiskLevel string

const (
	RiskLevelCritical      AIAuditorRiskLevel = "critical"
	RiskLevelHigh          AIAuditorRiskLevel = "high"
	RiskLevelMedium        AIAuditorRiskLevel = "medium"
	RiskLevelLow           AIAuditorRiskLevel = "low"
	RiskLevelInformational AIAuditorRiskLevel = "informational"
)

// ────────────────────────────────────────────────────────────────────────────
// AIAUDITOR evidence model (normalised from ASFF)
// ────────────────────────────────────────────────────────────────────────────

// EvidenceTier mirrors services/evidence evidence tier constants.
type EvidenceTier int

const (
	EvidenceTierL5Record    EvidenceTier = 5 // scan results, logs, configs
	EvidenceTierL6Telemetry EvidenceTier = 6 // automated API feeds
)

// AuditEvidenceItem is the AIAUDITOR internal evidence model produced by
// normalising an ASFFinding.
type AuditEvidenceItem struct {
	// Core identity
	SourceFindingID   string    `json:"source_finding_id"`
	SourceConnector   string    `json:"source_connector"`   // "aws-security-hub"
	SourceProductName string    `json:"source_product_name"`
	NormalizedAt      time.Time `json:"normalized_at"`

	// Evidence payload
	Title         string       `json:"title"`
	Description   string       `json:"description"`
	ContentFormat string       `json:"content_format"` // "json"
	SourceType    string       `json:"source_type"`    // "api_integration"
	SourceRef     string       `json:"source_ref"`     // finding ARN
	CollectedAt   time.Time    `json:"collected_at"`
	Tier          EvidenceTier `json:"tier"`

	// Risk and compliance
	RiskLevel           AIAuditorRiskLevel `json:"risk_level"`
	ASFFSeverity        ASFFSeverity       `json:"asff_severity"`
	ComplianceStatus    string             `json:"compliance_status"`
	RelatedControls     []string           `json:"related_controls,omitempty"`
	ComplianceStandards []string           `json:"compliance_standards,omitempty"`

	// Resource context
	AWSAccountID      string        `json:"aws_account_id"`
	Region            string        `json:"region"`
	AffectedResources []ResourceRef `json:"affected_resources"`

	// Quality scores (0.0–1.0)
	QualityScore QualityScoreBreakdown `json:"quality_score"`

	// Remediation
	RemediationText string `json:"remediation_text,omitempty"`
	RemediationURL  string `json:"remediation_url,omitempty"`

	// Workflow
	WorkflowStatus string `json:"workflow_status"`
	RecordState    string `json:"record_state"`

	// Tags for downstream classification
	Tags []string `json:"tags"`
}

// ResourceRef identifies a cloud resource affected by a finding.
type ResourceRef struct {
	Type   string `json:"type"`   // normalised resource type (e.g. "EC2", "S3", "IAM")
	ID     string `json:"id"`     // ARN or resource identifier
	Region string `json:"region"`
}

// QualityScoreBreakdown mirrors the evidence service QualityScore model.
type QualityScoreBreakdown struct {
	CompletenessScore  float64  `json:"completeness_score"`
	CurrencyScore      float64  `json:"currency_score"`
	SourceReliability  float64  `json:"source_reliability"`
	CorroborationScore float64  `json:"corroboration_score"`
	RelevanceScore     float64  `json:"relevance_score"`
	AggregateScore     float64  `json:"aggregate_score"`
	Deficiencies       []string `json:"deficiencies,omitempty"`
}

// ────────────────────────────────────────────────────────────────────────────
// Compliance result types
// ────────────────────────────────────────────────────────────────────────────

// ComplianceStandard identifies a compliance framework.
type ComplianceStandard string

const (
	StandardCISBenchmark ComplianceStandard = "CIS AWS Foundations Benchmark"
	StandardPCIDSS       ComplianceStandard = "PCI DSS"
	StandardSOC2         ComplianceStandard = "SOC 2"
	StandardNIST80053    ComplianceStandard = "NIST SP 800-53"
)

// ComplianceControlResult holds the pass/fail result for a single control check.
type ComplianceControlResult struct {
	ControlID    string               `json:"control_id"`
	ControlTitle string               `json:"control_title"`
	Standard     ComplianceStandard   `json:"standard"`
	Status       ASFFComplianceStatus `json:"status"`
	FindingCount int                  `json:"finding_count"`
	Findings     []string             `json:"finding_ids"`
	RiskLevel    AIAuditorRiskLevel   `json:"risk_level"`
	LastChecked  time.Time            `json:"last_checked"`
}

// ComplianceStandardSummary summarises findings across a single standard.
type ComplianceStandardSummary struct {
	Standard        ComplianceStandard        `json:"standard"`
	TotalControls   int                       `json:"total_controls"`
	PassedControls  int                       `json:"passed_controls"`
	FailedControls  int                       `json:"failed_controls"`
	WarningControls int                       `json:"warning_controls"`
	Score           float64                   `json:"score"` // 0.0–1.0
	Controls        []ComplianceControlResult `json:"controls,omitempty"`
	GeneratedAt     time.Time                 `json:"generated_at"`
}

// SecurityScore is the aggregated security posture score for an account.
type SecurityScore struct {
	AWSAccountID     string    `json:"aws_account_id"`
	Region           string    `json:"region"`
	OverallScore     float64   `json:"overall_score"`       // 0.0–100.0
	CriticalFindings int       `json:"critical_findings"`
	HighFindings     int       `json:"high_findings"`
	MediumFindings   int       `json:"medium_findings"`
	LowFindings      int       `json:"low_findings"`
	InfoFindings     int       `json:"informational_findings"`
	TotalFindings    int       `json:"total_findings"`
	ActiveStandards  []string  `json:"active_standards"`
	CalculatedAt     time.Time `json:"calculated_at"`
}

// ────────────────────────────────────────────────────────────────────────────
// ASFFNormalizer: converts ASFF findings → AIAUDITOR evidence model
// ────────────────────────────────────────────────────────────────────────────

// ASFFNormalizer converts AWS Security Hub ASFF findings to the
// AIAUDITOR internal evidence model.
type ASFFNormalizer struct{}

// NewASFFNormalizer returns a ready-to-use normalizer.
func NewASFFNormalizer() *ASFFNormalizer {
	return &ASFFNormalizer{}
}

// NormalizeFinding converts a single ASFF finding to an AuditEvidenceItem.
func (n *ASFFNormalizer) NormalizeFinding(f ASFFinding) AuditEvidenceItem {
	now := time.Now().UTC()

	riskLevel := MapASFFSeverity(f.Severity.Label)
	resources := normalizeResources(f.Resources)
	tags := buildTags(f)
	complianceStatus := ""
	var relatedControls []string
	var complianceStandards []string
	if f.Compliance != nil {
		complianceStatus = string(f.Compliance.Status)
		relatedControls = f.Compliance.RelatedRequirements
		for _, s := range f.Compliance.AssociatedStandards {
			complianceStandards = append(complianceStandards, friendlyStandardName(s.StandardsID))
		}
	}

	qs := n.scoreEvidence(f)

	productName := f.ProductName
	if productName == "" {
		productName = "AWS Security Hub"
	}

	return AuditEvidenceItem{
		SourceFindingID:   f.ID,
		SourceConnector:   "aws-security-hub",
		SourceProductName: productName,
		NormalizedAt:      now,

		Title:         f.Title,
		Description:   f.Description,
		ContentFormat: "json",
		SourceType:    "api_integration",
		SourceRef:     f.ID,
		CollectedAt:   f.LastObservedAt,
		Tier:          EvidenceTierL6Telemetry,

		RiskLevel:           riskLevel,
		ASFFSeverity:        f.Severity,
		ComplianceStatus:    complianceStatus,
		RelatedControls:     relatedControls,
		ComplianceStandards: complianceStandards,

		AWSAccountID:      f.AwsAccountID,
		Region:            f.Region,
		AffectedResources: resources,

		QualityScore: qs,

		RemediationText: f.Remediation.Recommendation.Text,
		RemediationURL:  f.Remediation.Recommendation.URL,

		WorkflowStatus: string(f.WorkflowStatus),
		RecordState:    string(f.RecordState),
		Tags:           tags,
	}
}

// NormalizeFindings batch-normalizes a slice of ASFF findings.
func (n *ASFFNormalizer) NormalizeFindings(findings []ASFFinding) []AuditEvidenceItem {
	items := make([]AuditEvidenceItem, 0, len(findings))
	for _, f := range findings {
		items = append(items, n.NormalizeFinding(f))
	}
	return items
}

// MapASFFSeverity converts an ASFF severity label to an AIAUDITOR risk level.
func MapASFFSeverity(label ASFFSeverityLabel) AIAuditorRiskLevel {
	switch label {
	case ASFFSeverityCritical:
		return RiskLevelCritical
	case ASFFSeverityHigh:
		return RiskLevelHigh
	case ASFFSeverityMedium:
		return RiskLevelMedium
	case ASFFSeverityLow:
		return RiskLevelLow
	case ASFFSeverityInformational:
		return RiskLevelInformational
	default:
		return RiskLevelInformational
	}
}

// MapNormalizedSeverity converts the ASFF Normalized score (0-100) to a risk level.
// Used as fallback when Label is absent.
func MapNormalizedSeverity(score int) AIAuditorRiskLevel {
	switch {
	case score >= 90:
		return RiskLevelCritical
	case score >= 70:
		return RiskLevelHigh
	case score >= 40:
		return RiskLevelMedium
	case score >= 1:
		return RiskLevelLow
	default:
		return RiskLevelInformational
	}
}

// NormalizeResourceType maps an ASFF resource type string to a canonical short type.
// e.g. "AwsEc2Instance" → "EC2", "AwsS3Bucket" → "S3".
func NormalizeResourceType(asffType string) string {
	switch {
	case strings.Contains(asffType, "Ec2") || strings.Contains(asffType, "EC2"):
		return "EC2"
	case strings.Contains(asffType, "S3"):
		return "S3"
	case strings.Contains(asffType, "Iam") || strings.Contains(asffType, "IAM"):
		return "IAM"
	case strings.Contains(asffType, "Rds") || strings.Contains(asffType, "RDS"):
		return "RDS"
	case strings.Contains(asffType, "Lambda"):
		return "Lambda"
	case strings.Contains(asffType, "Kms") || strings.Contains(asffType, "KMS"):
		return "KMS"
	case strings.Contains(asffType, "CloudTrail"):
		return "CloudTrail"
	case strings.Contains(asffType, "Config"):
		return "Config"
	case strings.Contains(asffType, "GuardDuty"):
		return "GuardDuty"
	case strings.Contains(asffType, "SecurityGroup") || strings.Contains(asffType, "Vpc") ||
		strings.Contains(asffType, "VPC"):
		return "VPC"
	case strings.Contains(asffType, "Sns") || strings.Contains(asffType, "SNS"):
		return "SNS"
	case strings.Contains(asffType, "Sqs") || strings.Contains(asffType, "SQS"):
		return "SQS"
	default:
		if asffType != "" {
			return asffType
		}
		return "Unknown"
	}
}

// ────────────────────────────────────────────────────────────────────────────
// quality scoring (inline — mirrors evidence/scorer logic without import)
// ────────────────────────────────────────────────────────────────────────────

const (
	qWeightCompleteness  = 0.25
	qWeightCurrency      = 0.20
	qWeightReliability   = 0.25
	qWeightCorroboration = 0.15
	qWeightRelevance     = 0.15
)

func (n *ASFFNormalizer) scoreEvidence(f ASFFinding) QualityScoreBreakdown {
	completeness, compDefs := scoreASFFCompleteness(f)
	currency, currDefs := scoreASFFCurrency(f)
	// API integration baseline (matches evidence/scorer sourceReliabilityBase).
	reliability := 0.90
	// Compliance presence bonus: structured compliance data increases trust.
	if f.Compliance != nil && f.Compliance.Status != "" {
		reliability = scoreClamp(reliability + 0.05)
	}
	corroboration, corrDefs := scoreASFFCorroboration(f)
	relevance, relDefs := scoreASFFRelevance(f)

	aggregate := qWeightCompleteness*completeness +
		qWeightCurrency*currency +
		qWeightReliability*reliability +
		qWeightCorroboration*corroboration +
		qWeightRelevance*relevance

	var defs []string
	defs = append(defs, compDefs...)
	defs = append(defs, currDefs...)
	defs = append(defs, corrDefs...)
	defs = append(defs, relDefs...)

	return QualityScoreBreakdown{
		CompletenessScore:  scoreRound(completeness),
		CurrencyScore:      scoreRound(currency),
		SourceReliability:  scoreRound(reliability),
		CorroborationScore: scoreRound(corroboration),
		RelevanceScore:     scoreRound(relevance),
		AggregateScore:     scoreRound(aggregate),
		Deficiencies:       defs,
	}
}

func scoreASFFCompleteness(f ASFFinding) (float64, []string) {
	score := 1.0
	var defs []string
	if strings.TrimSpace(f.Title) == "" {
		score -= 0.25
		defs = append(defs, "missing title")
	}
	if strings.TrimSpace(f.Description) == "" {
		score -= 0.15
		defs = append(defs, "missing description")
	}
	if f.ID == "" {
		score -= 0.20
		defs = append(defs, "missing finding ID")
	}
	if f.AwsAccountID == "" {
		score -= 0.10
		defs = append(defs, "missing AWS account ID")
	}
	if len(f.Resources) == 0 {
		score -= 0.20
		defs = append(defs, "no affected resources listed")
	}
	if f.LastObservedAt.IsZero() {
		score -= 0.10
		defs = append(defs, "missing last-observed timestamp")
	}
	return scoreClamp(score), defs
}

func scoreASFFCurrency(f ASFFinding) (float64, []string) {
	ref := f.LastObservedAt
	if ref.IsZero() {
		ref = f.UpdatedAt
	}
	if ref.IsZero() {
		return 0.20, []string{"no observation timestamp — currency cannot be verified"}
	}
	age := time.Since(ref)
	days := age.Hours() / 24
	switch {
	case days <= 1:
		return 1.0, nil
	case days <= 7:
		return 0.95, nil
	case days <= 30:
		return 0.90, nil
	case days <= 90:
		return 0.75, nil
	case days <= 180:
		return 0.55, []string{"finding not observed in 90+ days"}
	default:
		return 0.25, []string{"finding not observed in 180+ days; may be stale"}
	}
}

func scoreASFFCorroboration(f ASFFinding) (float64, []string) {
	// Corroboration: compliance block + remediation + type taxonomy.
	score := 0.20
	var defs []string
	if f.Compliance != nil {
		score += 0.35
	}
	if f.Remediation.Recommendation.Text != "" {
		score += 0.25
	}
	if len(f.Types) > 0 {
		score += 0.20
	}
	if score < 0.50 {
		defs = append(defs, "limited corroboration — no compliance or remediation context")
	}
	return scoreClamp(score), defs
}

func scoreASFFRelevance(f ASFFinding) (float64, []string) {
	content := strings.ToLower(f.Title + " " + f.Description)
	relevantTerms := []string{
		"security", "compliance", "access", "authentication", "encryption",
		"monitoring", "vulnerability", "incident", "patch", "firewall",
		"policy", "risk", "audit", "control", "backup",
	}
	hits := 0
	for _, term := range relevantTerms {
		if strings.Contains(content, term) {
			hits++
		}
	}
	var defs []string
	switch {
	case hits >= 5:
		return 0.95, nil
	case hits >= 3:
		return 0.80, nil
	case hits >= 1:
		return 0.60, nil
	default:
		defs = append(defs, "no audit-relevant terms in finding title/description")
		return 0.40, defs
	}
}

// ────────────────────────────────────────────────────────────────────────────
// internal helpers
// ────────────────────────────────────────────────────────────────────────────

func normalizeResources(resources []ASFFResource) []ResourceRef {
	refs := make([]ResourceRef, 0, len(resources))
	for _, r := range resources {
		refs = append(refs, ResourceRef{
			Type:   NormalizeResourceType(r.Type),
			ID:     r.ID,
			Region: r.Region,
		})
	}
	return refs
}

func buildTags(f ASFFinding) []string {
	tags := []string{"source:aws-security-hub", "content:cloud-security"}
	if f.Severity.Label != "" {
		tags = append(tags, fmt.Sprintf("severity:%s", strings.ToLower(string(f.Severity.Label))))
	}
	for _, r := range f.Resources {
		norm := NormalizeResourceType(r.Type)
		tag := fmt.Sprintf("resource:%s", strings.ToLower(norm))
		if !containsString(tags, tag) {
			tags = append(tags, tag)
		}
	}
	if f.Compliance != nil {
		for _, std := range f.Compliance.AssociatedStandards {
			name := friendlyStandardName(std.StandardsID)
			tag := fmt.Sprintf("standard:%s", strings.ToLower(strings.ReplaceAll(name, " ", "-")))
			if !containsString(tags, tag) {
				tags = append(tags, tag)
			}
		}
	}
	if f.Region != "" {
		tags = append(tags, fmt.Sprintf("region:%s", f.Region))
	}
	return tags
}

// friendlyStandardName converts an ASFF standards ARN to a readable name.
func friendlyStandardName(arn string) string {
	switch {
	case strings.Contains(arn, "cis-aws-foundations-benchmark"):
		return string(StandardCISBenchmark)
	case strings.Contains(arn, "pci-dss"):
		return string(StandardPCIDSS)
	case strings.Contains(arn, "soc2") || strings.Contains(arn, "aws-foundational-security"):
		return string(StandardSOC2)
	case strings.Contains(arn, "nist-800-53"):
		return string(StandardNIST80053)
	default:
		parts := strings.Split(arn, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
		return arn
	}
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func scoreClamp(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func scoreRound(v float64) float64 {
	return float64(int(v*10000)) / 10000
}

