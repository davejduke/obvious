package iam

import (
	"fmt"
	"time"
)

// ---------------------------------------------------------------------------
// Evidence types
// ---------------------------------------------------------------------------

// EvidenceType classifies the kind of IAM evidence item.
type EvidenceType string

const (
	EvidenceTypeMFAEnrollment     EvidenceType = "mfa_enrollment"
	EvidenceTypeUserProvisioning  EvidenceType = "user_provisioning"
	EvidenceTypeAccessReview      EvidenceType = "access_review"
	EvidenceTypeGroupMembership   EvidenceType = "group_membership"
	EvidenceTypeConditionalAccess EvidenceType = "conditional_access"
	EvidenceTypeSystemLog         EvidenceType = "system_log"
)

// EvidenceItem is a single AIAUDITOR evidence record derived from IAM data.
// It is compatible with the evidence service IngestRequest shape:
// SourceType="api_integration", ContentFormat="json", Tier=L6-Telemetry.
type EvidenceItem struct {
	ID           string       `json:"id"`
	Provider     string       `json:"provider"`
	EvidenceType EvidenceType `json:"evidence_type"`
	SubjectID    string       `json:"subject_id"`
	SubjectName  string       `json:"subject_name,omitempty"`
	CollectedAt  time.Time    `json:"collected_at"`
	Summary      string       `json:"summary"`
	// Severity is the risk label derived from QualityScore:
	// score>=0.9=info, >=0.7=low, >=0.5=medium, >=0.3=high, else=critical.
	Severity     string   `json:"severity"`
	// QualityScore is a single aggregate IAM-specific quality score (0.0–1.0).
	// Higher means stronger security posture evidence.
	QualityScore float64  `json:"quality_score"`
	Tags         []string `json:"tags,omitempty"`
	RawRef       string   `json:"raw_ref,omitempty"` // source record ID
}

// ---------------------------------------------------------------------------
// Mapping
// ---------------------------------------------------------------------------

// MapSnapshotToEvidence converts an IAMSnapshot into AIAUDITOR EvidenceItems.
// Every user, access review, group, conditional access policy, and system log
// event produces one or more evidence items with quality scores.
func MapSnapshotToEvidence(snap *IAMSnapshot) []EvidenceItem {
	var items []EvidenceItem

	for _, u := range snap.Users {
		items = append(items, mfaEvidenceFromUser(u, snap.CollectedAt)...)
		items = append(items, provisioningEvidenceFromUser(u, snap.CollectedAt))
	}
	for _, ar := range snap.AccessReviews {
		items = append(items, accessReviewEvidence(ar, snap.Provider, snap.CollectedAt))
	}
	for _, gm := range snap.GroupMemberships {
		items = append(items, groupMembershipEvidence(gm, snap.Provider, snap.CollectedAt))
	}
	for _, cap := range snap.ConditionalPolicies {
		items = append(items, conditionalAccessEvidence(cap, snap.Provider, snap.CollectedAt))
	}
	for _, sl := range snap.SystemLogs {
		items = append(items, systemLogEvidence(sl, snap.Provider, snap.CollectedAt))
	}

	return items
}

// ---------------------------------------------------------------------------
// MFA evidence
// ---------------------------------------------------------------------------

func mfaEvidenceFromUser(u IAMUser, collectedAt time.Time) []EvidenceItem {
	if !u.MFAEnrolled || len(u.MFAFactors) == 0 {
		return []EvidenceItem{{
			ID:           fmt.Sprintf("%s-mfa-%s-none", u.Provider, u.ExternalID),
			Provider:     u.Provider,
			EvidenceType: EvidenceTypeMFAEnrollment,
			SubjectID:    u.ExternalID,
			SubjectName:  u.Email,
			CollectedAt:  collectedAt,
			Summary:      fmt.Sprintf("User %s has no MFA factors enrolled", u.Email),
			Severity:     severityFromScore(0.0),
			QualityScore: 0.0,
			Tags:         []string{"mfa", "unenrolled"},
			RawRef:       u.ID,
		}}
	}

	items := make([]EvidenceItem, 0, len(u.MFAFactors))
	for _, f := range u.MFAFactors {
		score := mfaFactorScore(f.Type)
		items = append(items, EvidenceItem{
			ID:           fmt.Sprintf("%s-mfa-%s-%s", u.Provider, u.ExternalID, f.ID),
			Provider:     u.Provider,
			EvidenceType: EvidenceTypeMFAEnrollment,
			SubjectID:    u.ExternalID,
			SubjectName:  u.Email,
			CollectedAt:  collectedAt,
			Summary:      fmt.Sprintf("User %s enrolled %s MFA factor (status: %s)", u.Email, f.Type, f.Status),
			Severity:     severityFromScore(score),
			QualityScore: score,
			Tags:         []string{"mfa", "enrolled", f.Type},
			RawRef:       f.ID,
		})
	}
	return items
}

// mfaFactorScore returns a quality score based on MFA factor strength.
// FIDO2/hardware keys score highest; SMS/voice score lowest.
func mfaFactorScore(factorType string) float64 {
	switch factorType {
	case "fido2", "webauthn", "hardware_otp":
		return 1.0
	case "totp", "hotp":
		return 0.9
	case "push":
		return 0.8
	case "otp":
		return 0.7
	case "email":
		return 0.6
	case "sms", "voice":
		return 0.5
	default:
		return 0.5
	}
}

// ---------------------------------------------------------------------------
// Provisioning evidence
// ---------------------------------------------------------------------------

func provisioningEvidenceFromUser(u IAMUser, collectedAt time.Time) EvidenceItem {
	score := userProvisioningScore(u, collectedAt)
	summary := buildProvisioningSummary(u, collectedAt)

	return EvidenceItem{
		ID:           fmt.Sprintf("%s-provisioning-%s", u.Provider, u.ExternalID),
		Provider:     u.Provider,
		EvidenceType: EvidenceTypeUserProvisioning,
		SubjectID:    u.ExternalID,
		SubjectName:  u.Email,
		CollectedAt:  collectedAt,
		Summary:      summary,
		Severity:     severityFromScore(score),
		QualityScore: score,
		Tags:         []string{"provisioning", string(u.Status)},
		RawRef:       u.ID,
	}
}

func buildProvisioningSummary(u IAMUser, now time.Time) string {
	switch u.Status {
	case UserStatusActive:
		if u.LastLoginAt == nil {
			return fmt.Sprintf("Active user %s has never logged in", u.Email)
		}
		if now.Sub(*u.LastLoginAt) > 90*24*time.Hour {
			return fmt.Sprintf("Active user %s last logged in >90 days ago", u.Email)
		}
		return fmt.Sprintf("Active user %s is provisioned and recently active", u.Email)
	case UserStatusInactive:
		return fmt.Sprintf("Inactive user %s remains provisioned", u.Email)
	case UserStatusSuspended:
		return fmt.Sprintf("Suspended user %s is provisioned", u.Email)
	default:
		return fmt.Sprintf("User %s has status: %s", u.Email, u.Status)
	}
}

// userProvisioningScore scores the user's provisioning state.
// Active + recent login = 1.0; inactive with app assignments = 0.2.
func userProvisioningScore(u IAMUser, now time.Time) float64 {
	if u.Status == UserStatusActive {
		if u.LastLoginAt == nil {
			return 0.4 // Never logged in — potential orphan account
		}
		age := now.Sub(*u.LastLoginAt)
		switch {
		case age <= 30*24*time.Hour:
			return 1.0
		case age <= 90*24*time.Hour:
			return 0.8
		default:
			return 0.5 // Stale active account
		}
	}
	// Inactive/suspended with remaining app assignments is a higher risk.
	if len(u.AppAssignments) > 0 {
		return 0.2
	}
	return 0.3
}

// ---------------------------------------------------------------------------
// Access review evidence
// ---------------------------------------------------------------------------

func accessReviewEvidence(ar AccessReview, provider string, collectedAt time.Time) EvidenceItem {
	score, summary := accessReviewScoreAndSummary(ar)
	return EvidenceItem{
		ID:           fmt.Sprintf("%s-review-%s", provider, ar.ID),
		Provider:     provider,
		EvidenceType: EvidenceTypeAccessReview,
		SubjectID:    ar.ResourceID,
		CollectedAt:  collectedAt,
		Summary:      summary,
		Severity:     severityFromScore(score),
		QualityScore: score,
		Tags:         []string{"access_review", ar.Decision},
		RawRef:       ar.ID,
	}
}

func accessReviewScoreAndSummary(ar AccessReview) (float64, string) {
	switch ar.Decision {
	case "deny":
		// Confirmed over-provisioning caught and remediated — highest quality evidence.
		return 1.0, fmt.Sprintf(
			"Access review DENIED for resource %s by reviewer %s — over-provisioning detected",
			ar.ResourceID, ar.ReviewerID)
	case "approve":
		return 0.85, fmt.Sprintf(
			"Access review approved for resource %s by reviewer %s",
			ar.ResourceID, ar.ReviewerID)
	default:
		return 0.4, fmt.Sprintf(
			"Access review decision '%s' for resource %s (reviewer: %s)",
			ar.Decision, ar.ResourceID, ar.ReviewerID)
	}
}

// ---------------------------------------------------------------------------
// Group membership evidence
// ---------------------------------------------------------------------------

func groupMembershipEvidence(gm GroupMembership, provider string, collectedAt time.Time) EvidenceItem {
	return EvidenceItem{
		ID:           fmt.Sprintf("%s-group-%s", provider, gm.GroupID),
		Provider:     provider,
		EvidenceType: EvidenceTypeGroupMembership,
		SubjectID:    gm.GroupID,
		SubjectName:  gm.GroupName,
		CollectedAt:  collectedAt,
		Summary:      fmt.Sprintf("Group '%s' has %d members", gm.GroupName, len(gm.MemberIDs)),
		Severity:     "info",
		QualityScore: 0.9,
		Tags:         []string{"group_membership"},
		RawRef:       gm.GroupID,
	}
}

// ---------------------------------------------------------------------------
// Conditional access evidence
// ---------------------------------------------------------------------------

func conditionalAccessEvidence(cap ConditionalAccessPolicy, provider string, collectedAt time.Time) EvidenceItem {
	score, severity := conditionalAccessScoreAndSeverity(cap.State)
	controls := cap.GrantControls

	tags := make([]string, 0, 3+len(controls))
	tags = append(tags, "conditional_access", cap.State)
	tags = append(tags, controls...)

	return EvidenceItem{
		ID:           fmt.Sprintf("%s-cap-%s", provider, cap.ID),
		Provider:     provider,
		EvidenceType: EvidenceTypeConditionalAccess,
		SubjectID:    cap.ID,
		SubjectName:  cap.DisplayName,
		CollectedAt:  collectedAt,
		Summary:      fmt.Sprintf("Conditional access policy '%s' is %s", cap.DisplayName, cap.State),
		Severity:     severity,
		QualityScore: score,
		Tags:         tags,
		RawRef:       cap.ID,
	}
}

func conditionalAccessScoreAndSeverity(state string) (float64, string) {
	switch state {
	case "enabled":
		return 0.9, "info"
	case "enabledForReportingButNotEnforced":
		return 0.5, "medium"
	case "disabled":
		return 0.1, "high"
	default:
		return 0.3, "medium"
	}
}

// ---------------------------------------------------------------------------
// System log evidence
// ---------------------------------------------------------------------------

func systemLogEvidence(sl SystemLogEvent, provider string, collectedAt time.Time) EvidenceItem {
	score := systemLogScore(sl)
	return EvidenceItem{
		ID:           fmt.Sprintf("%s-syslog-%s", provider, sl.EventID),
		Provider:     provider,
		EvidenceType: EvidenceTypeSystemLog,
		SubjectID:    sl.ActorID,
		SubjectName:  sl.ActorEmail,
		CollectedAt:  collectedAt,
		Summary: fmt.Sprintf(
			"[%s] %s — outcome=%s actor=%s",
			sl.Severity, sl.EventType, sl.Outcome, sl.ActorEmail),
		Severity:     oktaSeverityToAIAuditor(sl.Severity),
		QualityScore: score,
		Tags:         []string{"system_log", sl.EventType, sl.Outcome},
		RawRef:       sl.EventID,
	}
}

func systemLogScore(sl SystemLogEvent) float64 {
	if sl.Outcome == "SUCCESS" || sl.Outcome == "ALLOW" {
		return 0.9
	}
	if sl.Outcome == "FAILURE" || sl.Outcome == "DENY" {
		switch sl.Severity {
		case "ERROR":
			return 0.2
		case "WARN":
			return 0.4
		default:
			return 0.6
		}
	}
	return 0.7
}

// oktaSeverityToAIAuditor maps Okta log severity to AIAUDITOR severity labels.
func oktaSeverityToAIAuditor(s string) string {
	switch s {
	case "ERROR":
		return "high"
	case "WARN":
		return "medium"
	case "INFO":
		return "low"
	default:
		return "info"
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// severityFromScore converts a quality score to an AIAUDITOR severity label.
// Higher score = better posture = lower severity (info/low).
func severityFromScore(score float64) string {
	switch {
	case score >= 0.9:
		return "info"
	case score >= 0.7:
		return "low"
	case score >= 0.5:
		return "medium"
	case score >= 0.3:
		return "high"
	default:
		return "critical"
	}
}
