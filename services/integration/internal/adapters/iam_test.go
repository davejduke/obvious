package adapters_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/adapters"
	"github.com/davejduke/obvious/services/integration/internal/connector"
	"github.com/davejduke/obvious/services/integration/internal/iam"
)

// ---------------------------------------------------------------------------
// Entra ID adapter tests
// ---------------------------------------------------------------------------

func TestEntraIDAdapter_Name(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{})
	if a.Name() != "entraid" {
		t.Errorf("expected name=entraid, got %s", a.Name())
	}
}

func TestEntraIDAdapter_MockHealth(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	status := a.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "entraid" {
		t.Errorf("expected connector=entraid, got %s", status.Connector)
	}
}

func TestEntraIDAdapter_MockSync_Users(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{
		TenantID: "test-tenant",
		MockMode: true,
	})

	snap, err := a.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Provider != "entraid" {
		t.Errorf("expected provider=entraid, got %s", snap.Provider)
	}
	if len(snap.Users) == 0 {
		t.Fatal("expected at least one user")
	}
	for _, u := range snap.Users {
		if u.ExternalID == "" {
			t.Error("expected non-empty ExternalID")
		}
		if u.Email == "" {
			t.Error("expected non-empty Email")
		}
		if u.Provider != "entraid" {
			t.Errorf("expected provider=entraid on user, got %s", u.Provider)
		}
	}
}

func TestEntraIDAdapter_MockSync_MFAFactors(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	snap, err := a.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// At least one user should have MFA factors enrolled.
	enrolled := 0
	for _, u := range snap.Users {
		if u.MFAEnrolled {
			enrolled++
			if len(u.MFAFactors) == 0 {
				t.Errorf("user %s is marked MFAEnrolled but has no factors", u.Email)
			}
			for _, f := range u.MFAFactors {
				if f.ID == "" || f.Type == "" {
					t.Errorf("factor on user %s is missing ID or Type", u.Email)
				}
			}
		}
	}
	if enrolled == 0 {
		t.Error("expected at least one MFA-enrolled user")
	}
}

func TestEntraIDAdapter_MockSync_AccessReviews(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())

	if len(snap.AccessReviews) == 0 {
		t.Error("expected at least one access review")
	}
	for _, ar := range snap.AccessReviews {
		if ar.ID == "" || ar.Decision == "" {
			t.Errorf("access review missing ID or Decision: %+v", ar)
		}
	}
}

func TestEntraIDAdapter_MockSync_Groups(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())

	if len(snap.GroupMemberships) == 0 {
		t.Error("expected at least one group")
	}
	for _, g := range snap.GroupMemberships {
		if g.GroupID == "" || g.GroupName == "" {
			t.Errorf("group missing ID or Name: %+v", g)
		}
	}
}

func TestEntraIDAdapter_MockSync_ConditionalAccessPolicies(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())

	if len(snap.ConditionalPolicies) == 0 {
		t.Error("expected at least one conditional access policy")
	}

	hasEnabled := false
	for _, p := range snap.ConditionalPolicies {
		if p.ID == "" || p.DisplayName == "" {
			t.Errorf("policy missing ID or DisplayName: %+v", p)
		}
		if p.State == "enabled" {
			hasEnabled = true
		}
	}
	if !hasEnabled {
		t.Error("expected at least one enabled conditional access policy")
	}
}

// ---------------------------------------------------------------------------
// Okta adapter tests
// ---------------------------------------------------------------------------

func TestOktaAdapter_Name(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{})
	if a.Name() != "okta" {
		t.Errorf("expected name=okta, got %s", a.Name())
	}
}

func TestOktaAdapter_MockHealth(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{MockMode: true})
	status := a.Health(context.Background())
	if !status.Healthy {
		t.Errorf("expected healthy in mock mode, got: %s", status.Message)
	}
	if status.Connector != "okta" {
		t.Errorf("expected connector=okta, got %s", status.Connector)
	}
}

func TestOktaAdapter_MockSync_Users(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{
		OrgURL:   "https://dev-12345.okta.com",
		MockMode: true,
	})
	snap, err := a.Sync(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if snap.Provider != "okta" {
		t.Errorf("expected provider=okta, got %s", snap.Provider)
	}
	if len(snap.Users) == 0 {
		t.Fatal("expected at least one user")
	}
	for _, u := range snap.Users {
		if u.ExternalID == "" {
			t.Error("expected non-empty ExternalID")
		}
		if u.Provider != "okta" {
			t.Errorf("expected provider=okta on user, got %s", u.Provider)
		}
	}
}

func TestOktaAdapter_MockSync_MFAFactors(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())

	enrolled := 0
	for _, u := range snap.Users {
		if u.MFAEnrolled {
			enrolled++
			if len(u.MFAFactors) == 0 {
				t.Errorf("user %s is marked MFAEnrolled but has no factors", u.Email)
			}
		}
	}
	if enrolled == 0 {
		t.Error("expected at least one MFA-enrolled user")
	}
}

func TestOktaAdapter_MockSync_AppAssignments(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())

	hasAssignments := false
	for _, u := range snap.Users {
		if len(u.AppAssignments) > 0 {
			hasAssignments = true
			for _, app := range u.AppAssignments {
				if app.AppID == "" || app.AppName == "" {
					t.Errorf("app assignment on user %s missing AppID or AppName", u.Email)
				}
			}
		}
	}
	if !hasAssignments {
		t.Error("expected at least one user with app assignments")
	}
}

func TestOktaAdapter_MockSync_SystemLogs(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())

	if len(snap.SystemLogs) == 0 {
		t.Error("expected at least one system log event")
	}
	for _, l := range snap.SystemLogs {
		if l.EventID == "" {
			t.Error("expected non-empty EventID")
		}
		if l.EventType == "" {
			t.Error("expected non-empty EventType")
		}
		if l.Outcome == "" {
			t.Error("expected non-empty Outcome")
		}
	}
}

func TestOktaAdapter_MockSync_SuspendedUserWithAppAccess(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())

	// The mock data includes grace@example.com who is Suspended with AWS SSO access.
	found := false
	for _, u := range snap.Users {
		if u.Status == iam.UserStatusSuspended && len(u.AppAssignments) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a suspended user with app assignments in mock data")
	}
}

// ---------------------------------------------------------------------------
// Evidence mapping tests
// ---------------------------------------------------------------------------

func TestMapSnapshotToEvidence_MFAEnrolled_FIDO2(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "entraid",
		CollectedAt: now,
		Users: []iam.IAMUser{
			{
				ID:          "u-001",
				ExternalID:  "ext-001",
				Email:       "alice@example.com",
				DisplayName: "Alice",
				Status:      iam.UserStatusActive,
				CreatedAt:   now.Add(-90 * 24 * time.Hour),
				MFAEnrolled: true,
				MFAFactors: []iam.MFAFactor{
					{ID: "f-001", Type: "fido2", Status: "active", Provider: "entraid"},
				},
				Provider: "entraid",
			},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	var mfaItem *iam.EvidenceItem
	for i := range items {
		if items[i].EvidenceType == iam.EvidenceTypeMFAEnrollment {
			mfaItem = &items[i]
			break
		}
	}
	if mfaItem == nil {
		t.Fatal("expected an MFA enrollment evidence item")
	}
	if mfaItem.QualityScore != 1.0 {
		t.Errorf("expected FIDO2 quality=1.0, got %.2f", mfaItem.QualityScore)
	}
	if mfaItem.Severity != "info" {
		t.Errorf("expected severity=info for score=1.0, got %s", mfaItem.Severity)
	}
}

func TestMapSnapshotToEvidence_MFANotEnrolled(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "okta",
		CollectedAt: now,
		Users: []iam.IAMUser{
			{
				ID:          "u-002",
				ExternalID:  "ext-002",
				Email:       "bob@example.com",
				Status:      iam.UserStatusActive,
				MFAEnrolled: false,
				Provider:    "okta",
			},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	var mfaItem *iam.EvidenceItem
	for i := range items {
		if items[i].EvidenceType == iam.EvidenceTypeMFAEnrollment {
			mfaItem = &items[i]
			break
		}
	}
	if mfaItem == nil {
		t.Fatal("expected an MFA enrollment evidence item for unenrolled user")
	}
	if mfaItem.QualityScore != 0.0 {
		t.Errorf("expected quality=0.0 for unenrolled user, got %.2f", mfaItem.QualityScore)
	}
	if mfaItem.Severity != "critical" {
		t.Errorf("expected severity=critical for score=0.0, got %s", mfaItem.Severity)
	}
}

func TestMapSnapshotToEvidence_SMSMFALowerScore(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "okta",
		CollectedAt: now,
		Users: []iam.IAMUser{
			{
				ExternalID:  "ext-sms",
				Email:       "sms@example.com",
				Status:      iam.UserStatusActive,
				MFAEnrolled: true,
				MFAFactors: []iam.MFAFactor{
					{ID: "f-sms", Type: "sms", Status: "active", Provider: "okta"},
				},
				Provider: "okta",
			},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeMFAEnrollment {
			if item.QualityScore >= 0.9 {
				t.Errorf("SMS MFA should score < 0.9, got %.2f", item.QualityScore)
			}
			return
		}
	}
	t.Error("expected MFA enrollment evidence item")
}

func TestMapSnapshotToEvidence_UserProvisioning_ActiveRecentLogin(t *testing.T) {
	now := time.Now().UTC()
	lastWeek := now.Add(-7 * 24 * time.Hour)
	snap := &iam.IAMSnapshot{
		Provider:    "entraid",
		CollectedAt: now,
		Users: []iam.IAMUser{
			{
				ExternalID:  "ext-active",
				Email:       "active@example.com",
				Status:      iam.UserStatusActive,
				LastLoginAt: &lastWeek,
				Provider:    "entraid",
			},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeUserProvisioning {
			if item.QualityScore < 0.9 {
				t.Errorf("recently active user should score >= 0.9, got %.2f", item.QualityScore)
			}
			return
		}
	}
	t.Error("expected user provisioning evidence item")
}

func TestMapSnapshotToEvidence_UserProvisioning_StaleActive(t *testing.T) {
	now := time.Now().UTC()
	oldLogin := now.Add(-120 * 24 * time.Hour) // 120 days ago
	snap := &iam.IAMSnapshot{
		Provider:    "entraid",
		CollectedAt: now,
		Users: []iam.IAMUser{
			{
				ExternalID:  "ext-stale",
				Email:       "stale@example.com",
				Status:      iam.UserStatusActive,
				LastLoginAt: &oldLogin,
				Provider:    "entraid",
			},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeUserProvisioning {
			if item.QualityScore > 0.7 {
				t.Errorf("stale active user should score <= 0.7, got %.2f", item.QualityScore)
			}
			return
		}
	}
	t.Error("expected user provisioning evidence item")
}

func TestMapSnapshotToEvidence_AccessReview_Deny(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "entraid",
		CollectedAt: now,
		AccessReviews: []iam.AccessReview{
			{ID: "rev-001", ReviewerID: "alice", ResourceID: "app-001", Decision: "deny", CompletedAt: now},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeAccessReview {
			if item.QualityScore != 1.0 {
				t.Errorf("deny decision should score 1.0, got %.2f", item.QualityScore)
			}
			return
		}
	}
	t.Error("expected access review evidence item")
}

func TestMapSnapshotToEvidence_AccessReview_Approve(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "entraid",
		CollectedAt: now,
		AccessReviews: []iam.AccessReview{
			{ID: "rev-002", ReviewerID: "alice", ResourceID: "app-002", Decision: "approve", CompletedAt: now},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeAccessReview {
			if item.QualityScore < 0.8 {
				t.Errorf("approve decision should score >= 0.8, got %.2f", item.QualityScore)
			}
			return
		}
	}
	t.Error("expected access review evidence item")
}

func TestMapSnapshotToEvidence_ConditionalAccess_Enabled(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "entraid",
		CollectedAt: now,
		ConditionalPolicies: []iam.ConditionalAccessPolicy{
			{ID: "cap-001", DisplayName: "Require MFA", State: "enabled", GrantControls: []string{"mfa"}},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeConditionalAccess {
			if item.QualityScore < 0.85 {
				t.Errorf("enabled policy should score >= 0.85, got %.2f", item.QualityScore)
			}
			if item.Severity != "info" {
				t.Errorf("enabled policy should have severity=info, got %s", item.Severity)
			}
			return
		}
	}
	t.Error("expected conditional access evidence item")
}

func TestMapSnapshotToEvidence_ConditionalAccess_Disabled(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "entraid",
		CollectedAt: now,
		ConditionalPolicies: []iam.ConditionalAccessPolicy{
			{ID: "cap-disabled", DisplayName: "Disabled Policy", State: "disabled"},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeConditionalAccess {
			if item.QualityScore > 0.2 {
				t.Errorf("disabled policy should score <= 0.2, got %.2f", item.QualityScore)
			}
			if item.Severity != "high" {
				t.Errorf("disabled policy should have severity=high, got %s", item.Severity)
			}
			return
		}
	}
	t.Error("expected conditional access evidence item")
}

func TestMapSnapshotToEvidence_SystemLog_Success(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "okta",
		CollectedAt: now,
		SystemLogs: []iam.SystemLogEvent{
			{
				EventID:     "log-001",
				PublishedAt: now.Add(-1 * time.Hour),
				EventType:   "user.authentication.auth_via_mfa",
				Severity:    "INFO",
				ActorID:     "user-001",
				ActorEmail:  "eve@example.com",
				Outcome:     "SUCCESS",
			},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeSystemLog {
			if item.QualityScore < 0.85 {
				t.Errorf("SUCCESS log should score >= 0.85, got %.2f", item.QualityScore)
			}
			return
		}
	}
	t.Error("expected system log evidence item")
}

func TestMapSnapshotToEvidence_SystemLog_Failure(t *testing.T) {
	now := time.Now().UTC()
	snap := &iam.IAMSnapshot{
		Provider:    "okta",
		CollectedAt: now,
		SystemLogs: []iam.SystemLogEvent{
			{
				EventID:     "log-002",
				PublishedAt: now,
				EventType:   "user.authentication.sso",
				Severity:    "ERROR",
				ActorID:     "user-003",
				ActorEmail:  "grace@example.com",
				Outcome:     "FAILURE",
			},
		},
	}

	items := iam.MapSnapshotToEvidence(snap)
	for _, item := range items {
		if item.EvidenceType == iam.EvidenceTypeSystemLog {
			if item.QualityScore > 0.3 {
				t.Errorf("ERROR FAILURE log should score <= 0.3, got %.2f", item.QualityScore)
			}
			return
		}
	}
	t.Error("expected system log evidence item")
}

func TestMapSnapshotToEvidence_EvidenceItemIDs_AreUnique(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())
	items := iam.MapSnapshotToEvidence(snap)

	seen := make(map[string]bool, len(items))
	for _, item := range items {
		if seen[item.ID] {
			t.Errorf("duplicate evidence item ID: %s", item.ID)
		}
		seen[item.ID] = true
	}
}

func TestMapSnapshotToEvidence_EntraIDFullMock(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())
	items := iam.MapSnapshotToEvidence(snap)

	if len(items) == 0 {
		t.Fatal("expected evidence items from Entra ID mock snapshot")
	}
	// Check coverage of evidence types
	typesSeen := make(map[iam.EvidenceType]int)
	for _, item := range items {
		typesSeen[item.EvidenceType]++
		if item.Provider == "" {
			t.Error("evidence item missing provider")
		}
		if item.CollectedAt.IsZero() {
			t.Error("evidence item missing CollectedAt")
		}
		if item.QualityScore < 0.0 || item.QualityScore > 1.0 {
			t.Errorf("evidence quality score out of range [0,1]: %.2f", item.QualityScore)
		}
	}
	expectedTypes := []iam.EvidenceType{
		iam.EvidenceTypeMFAEnrollment,
		iam.EvidenceTypeUserProvisioning,
		iam.EvidenceTypeAccessReview,
		iam.EvidenceTypeGroupMembership,
		iam.EvidenceTypeConditionalAccess,
	}
	for _, et := range expectedTypes {
		if typesSeen[et] == 0 {
			t.Errorf("expected evidence type %s to be present", et)
		}
	}
}

func TestMapSnapshotToEvidence_OktaFullMock(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{MockMode: true})
	snap, _ := a.Sync(context.Background())
	items := iam.MapSnapshotToEvidence(snap)

	if len(items) == 0 {
		t.Fatal("expected evidence items from Okta mock snapshot")
	}
	typesSeen := make(map[iam.EvidenceType]int)
	for _, item := range items {
		typesSeen[item.EvidenceType]++
		if item.QualityScore < 0.0 || item.QualityScore > 1.0 {
			t.Errorf("evidence quality score out of range [0,1]: %.2f", item.QualityScore)
		}
	}
	expectedTypes := []iam.EvidenceType{
		iam.EvidenceTypeMFAEnrollment,
		iam.EvidenceTypeUserProvisioning,
		iam.EvidenceTypeSystemLog,
	}
	for _, et := range expectedTypes {
		if typesSeen[et] == 0 {
			t.Errorf("expected evidence type %s to be present", et)
		}
	}
}

// ---------------------------------------------------------------------------
// Scheduler tests
// ---------------------------------------------------------------------------

func TestScheduler_SyncOnce(t *testing.T) {
	a := adapters.NewEntraIDAdapter(adapters.EntraIDConfig{MockMode: true})
	reg := iam.NewIAMRegistry()
	reg.Register(a)

	cfg := iam.SchedulerConfig{
		Interval: 10 * time.Minute, // long interval; we only test the initial sync
	}
	sched := iam.NewScheduler(reg, cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sched.Start(ctx)

	// Give the initial sync a moment to complete.
	deadline := time.Now().Add(2 * time.Second)
	var result iam.SyncResult
	var ok bool
	for time.Now().Before(deadline) {
		result, ok = sched.LastResult("entraid")
		if ok {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !ok {
		t.Fatal("scheduler never produced a result within 2s")
	}
	if result.Error != nil {
		t.Fatalf("unexpected sync error: %v", result.Error)
	}
	if result.Snapshot == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if len(result.Evidence) == 0 {
		t.Error("expected evidence items from scheduler sync")
	}
	sched.Stop()
}

func TestScheduler_CallsOnSyncCallback(t *testing.T) {
	a := adapters.NewOktaAdapter(adapters.OktaConfig{MockMode: true})
	reg := iam.NewIAMRegistry()
	reg.Register(a)

	called := make(chan iam.SyncResult, 1)
	cfg := iam.SchedulerConfig{
		Interval: 10 * time.Minute,
		OnSync: func(r iam.SyncResult) {
			select {
			case called <- r:
			default:
			}
		},
	}
	sched := iam.NewScheduler(reg, cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	sched.Start(ctx)
	defer sched.Stop()

	select {
	case r := <-called:
		if r.Provider != "okta" {
			t.Errorf("expected provider=okta in callback, got %s", r.Provider)
		}
	case <-time.After(2 * time.Second):
		t.Error("OnSync callback was never called")
	}
}

// ---------------------------------------------------------------------------
// IAM circuit breaker tests
// ---------------------------------------------------------------------------

// failingIAMConnector is a test double whose Sync always fails.
type failingIAMConnector struct{ name string }

func (f *failingIAMConnector) Name() string { return f.name }
func (f *failingIAMConnector) Sync(_ context.Context) (*iam.IAMSnapshot, error) {
	return nil, errors.New("upstream IAM error")
}
func (f *failingIAMConnector) Health(_ context.Context) connector.HealthStatus {
	return connector.HealthStatus{Healthy: false, Connector: f.name}
}

func TestIAMCircuitBreaker_Name(t *testing.T) {
	inner := &failingIAMConnector{name: "test-iam"}
	cb := iam.NewIAMCircuitBreaker(inner, connector.DefaultConfig())
	if cb.Name() != "test-iam" {
		t.Errorf("expected name=test-iam, got %s", cb.Name())
	}
}

func TestIAMCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cfg := connector.CircuitBreakerConfig{
		FailureThreshold: 3,
		RecoveryTimeout:  30 * time.Second,
	}
	inner := &failingIAMConnector{name: "failing-iam"}
	cb := iam.NewIAMCircuitBreaker(inner, cfg)
	ctx := context.Background()

	// First 3 calls fail but circuit stays closed.
	for i := 0; i < 3; i++ {
		_, err := cb.Sync(ctx)
		if errors.Is(err, connector.ErrCircuitOpen) {
			t.Fatalf("circuit opened too early at call %d", i+1)
		}
	}
	if cb.CircuitState() != connector.StateOpen {
		t.Errorf("expected StateOpen after threshold, got %s", cb.CircuitState())
	}

	// Next call should be short-circuited.
	_, err := cb.Sync(ctx)
	if !errors.Is(err, connector.ErrCircuitOpen) {
		t.Errorf("expected ErrCircuitOpen after threshold, got %v", err)
	}
}

func TestIAMCircuitBreaker_HalfOpenAfterTimeout(t *testing.T) {
	cfg := connector.CircuitBreakerConfig{
		FailureThreshold: 1,
		RecoveryTimeout:  10 * time.Millisecond,
	}
	inner := &failingIAMConnector{name: "failing-iam"}
	cb := iam.NewIAMCircuitBreaker(inner, cfg)
	ctx := context.Background()

	cb.Sync(ctx) //nolint:errcheck — trigger open
	cb.Sync(ctx) //nolint:errcheck — confirm open
	if cb.CircuitState() != connector.StateOpen {
		t.Fatal("circuit should be open")
	}

	time.Sleep(20 * time.Millisecond)
	if cb.CircuitState() != connector.StateHalfOpen {
		t.Errorf("expected StateHalfOpen after recovery timeout, got %s", cb.CircuitState())
	}
}

func TestIAMCircuitBreaker_HealthPassesThroughCircuitState(t *testing.T) {
	cfg := connector.CircuitBreakerConfig{FailureThreshold: 1, RecoveryTimeout: 30 * time.Second}
	inner := &failingIAMConnector{name: "iam-health-test"}
	cb := iam.NewIAMCircuitBreaker(inner, cfg)
	ctx := context.Background()

	// Open the circuit.
	cb.Sync(ctx) //nolint:errcheck
	cb.Sync(ctx) //nolint:errcheck

	status := cb.Health(ctx)
	// Message should mention the circuit state.
	if status.Message == "" {
		t.Error("expected health message to mention circuit state")
	}
}
