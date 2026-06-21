package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/davejduke/obvious/services/integration/internal/connector"
	"github.com/davejduke/obvious/services/integration/internal/iam"
)

// OktaConfig holds Okta connection settings.
type OktaConfig struct {
	// OrgURL is the Okta org base URL, e.g. https://dev-12345.okta.com
	OrgURL string
	// APIToken is the Okta Management API token (SSWS).
	APIToken string
	// MockMode returns deterministic synthetic data without real API calls.
	MockMode bool
}

// OktaAdapter syncs IAM data from Okta via the Okta Management API.
// It implements iam.IAMConnector.
type OktaAdapter struct {
	config OktaConfig
	client *http.Client
}

// NewOktaAdapter creates a ready-to-use Okta connector.
func NewOktaAdapter(cfg OktaConfig) *OktaAdapter {
	return &OktaAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements iam.IAMConnector.
func (a *OktaAdapter) Name() string { return "okta" }

// Sync fetches a full IAM snapshot from Okta.
// In mock mode it returns deterministic synthetic data.
func (a *OktaAdapter) Sync(ctx context.Context) (*iam.IAMSnapshot, error) {
	if a.config.MockMode {
		return a.mockSnapshot(), nil
	}
	return a.fetchFromAPI(ctx)
}

// Health reports whether the Okta API is reachable.
func (a *OktaAdapter) Health(ctx context.Context) connector.HealthStatus {
	status := connector.HealthStatus{
		Connector:   a.Name(),
		LastChecked: time.Now().UTC(),
	}
	if a.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}

	url := fmt.Sprintf("%s/api/v1/org", a.orgBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
	req.Header.Set("Authorization", "SSWS "+a.config.APIToken)
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

// fetchFromAPI calls the Okta Management API for all IAM data.
func (a *OktaAdapter) fetchFromAPI(ctx context.Context) (*iam.IAMSnapshot, error) {
	users, err := a.fetchUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("okta: fetch users: %w", err)
	}

	systemLogs, err := a.fetchSystemLogs(ctx)
	if err != nil {
		return nil, fmt.Errorf("okta: fetch system logs: %w", err)
	}

	return &iam.IAMSnapshot{
		Provider:    a.Name(),
		CollectedAt: time.Now().UTC(),
		Users:       users,
		SystemLogs:  systemLogs,
	}, nil
}

func (a *OktaAdapter) fetchUsers(ctx context.Context) ([]iam.IAMUser, error) {
	url := fmt.Sprintf("%s/api/v1/users?limit=200&filter=status+eq+\"ACTIVE\"", a.orgBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "SSWS "+a.config.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var result []struct {
		ID      string `json:"id"`
		Status  string `json:"status"` // ACTIVE | INACTIVE | DEPROVISIONED | SUSPENDED | RECOVERY | LOCKED_OUT
		Profile struct {
			Login     string `json:"login"`
			Email     string `json:"email"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
		} `json:"profile"`
		Created   string `json:"created"`
		LastLogin string `json:"lastLogin"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	users := make([]iam.IAMUser, 0, len(result))
	for _, u := range result {
		status := oktaStatusToNormal(u.Status)
		created, _ := time.Parse(time.RFC3339, u.Created)
		var lastLogin *time.Time
		if u.LastLogin != "" {
			t, err := time.Parse(time.RFC3339, u.LastLogin)
			if err == nil {
				lastLogin = &t
			}
		}
		users = append(users, iam.IAMUser{
			ID:          fmt.Sprintf("okta-%s", u.ID),
			ExternalID:  u.ID,
			Email:       u.Profile.Email,
			DisplayName: fmt.Sprintf("%s %s", u.Profile.FirstName, u.Profile.LastName),
			Status:      status,
			CreatedAt:   created,
			LastLoginAt: lastLogin,
			Provider:    a.Name(),
		})
	}
	return users, nil
}

func (a *OktaAdapter) fetchSystemLogs(ctx context.Context) ([]iam.SystemLogEvent, error) {
	url := fmt.Sprintf("%s/api/v1/logs?limit=100&sortOrder=DESCENDING", a.orgBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "SSWS "+a.config.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}

	var result []struct {
		UUID      string `json:"uuid"`
		Published string `json:"published"`
		EventType string `json:"eventType"`
		Severity  string `json:"severity"`
		Actor     struct {
			ID          string `json:"id"`
			AlternateID string `json:"alternateId"`
		} `json:"actor"`
		Target []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"target"`
		Outcome struct {
			Result string `json:"result"`
		} `json:"outcome"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	logs := make([]iam.SystemLogEvent, 0, len(result))
	for _, e := range result {
		published, _ := time.Parse(time.RFC3339, e.Published)
		var targetID, targetType string
		if len(e.Target) > 0 {
			targetID = e.Target[0].ID
			targetType = e.Target[0].Type
		}
		logs = append(logs, iam.SystemLogEvent{
			EventID:     e.UUID,
			PublishedAt: published,
			EventType:   e.EventType,
			Severity:    e.Severity,
			ActorID:     e.Actor.ID,
			ActorEmail:  e.Actor.AlternateID,
			TargetID:    targetID,
			TargetType:  targetType,
			Outcome:     e.Outcome.Result,
		})
	}
	return logs, nil
}

// ---------------------------------------------------------------------------
// Mock snapshot — deterministic, realistic Okta data
// ---------------------------------------------------------------------------

func (a *OktaAdapter) mockSnapshot() *iam.IAMSnapshot {
	now := time.Now().UTC()
	yesterday := now.Add(-24 * time.Hour)
	lastMonth := now.Add(-45 * 24 * time.Hour)

	users := []iam.IAMUser{
		{
			ID:          "okta-user-001",
			ExternalID:  "okta-001-eve",
			Email:       "eve@example.com",
			DisplayName: "Eve Martinez",
			Status:      iam.UserStatusActive,
			CreatedAt:   now.Add(-200 * 24 * time.Hour),
			LastLoginAt: &yesterday,
			MFAEnrolled: true,
			MFAFactors: []iam.MFAFactor{
				{ID: "okta-factor-001", Type: "push", Status: "active", Provider: "okta"},
				{ID: "okta-factor-002", Type: "totp", Status: "active", Provider: "okta"},
			},
			AppAssignments: []iam.AppAssignment{
				{AppID: "okta-app-001", AppName: "Slack", Status: "active"},
				{AppID: "okta-app-002", AppName: "Jira", Status: "active"},
			},
			Groups:   []string{"developers", "all-users"},
			Provider: "okta",
		},
		{
			// Active user with stale last login — medium risk.
			ID:          "okta-user-002",
			ExternalID:  "okta-002-frank",
			Email:       "frank@example.com",
			DisplayName: "Frank Wilson",
			Status:      iam.UserStatusActive,
			CreatedAt:   now.Add(-100 * 24 * time.Hour),
			LastLoginAt: &lastMonth,
			MFAEnrolled: true,
			MFAFactors: []iam.MFAFactor{
				{ID: "okta-factor-003", Type: "sms", Status: "active", Provider: "okta"},
			},
			AppAssignments: []iam.AppAssignment{
				{AppID: "okta-app-001", AppName: "Slack", Status: "active"},
			},
			Groups:   []string{"all-users"},
			Provider: "okta",
		},
		{
			// Suspended user with AWS SSO access — high risk.
			ID:          "okta-user-003",
			ExternalID:  "okta-003-grace",
			Email:       "grace@example.com",
			DisplayName: "Grace Lee",
			Status:      iam.UserStatusSuspended,
			CreatedAt:   now.Add(-600 * 24 * time.Hour),
			LastLoginAt: &lastMonth,
			MFAEnrolled: false,
			AppAssignments: []iam.AppAssignment{
				{AppID: "okta-app-003", AppName: "AWS SSO", Status: "active"},
			},
			Groups:   []string{"all-users"},
			Provider: "okta",
		},
		{
			// New user — no MFA yet.
			ID:          "okta-user-004",
			ExternalID:  "okta-004-henry",
			Email:       "henry@example.com",
			DisplayName: "Henry Brown",
			Status:      iam.UserStatusActive,
			CreatedAt:   now.Add(-14 * 24 * time.Hour),
			MFAEnrolled: false,
			AppAssignments: []iam.AppAssignment{
				{AppID: "okta-app-001", AppName: "Slack", Status: "active"},
			},
			Groups:   []string{"all-users"},
			Provider: "okta",
		},
	}

	systemLogs := []iam.SystemLogEvent{
		{
			EventID:     "okta-log-001",
			PublishedAt: now.Add(-1 * time.Hour),
			EventType:   "user.authentication.auth_via_mfa",
			Severity:    "INFO",
			ActorID:     "okta-001-eve",
			ActorEmail:  "eve@example.com",
			TargetID:    "okta-app-001",
			TargetType:  "AppInstance",
			Outcome:     "SUCCESS",
		},
		{
			EventID:     "okta-log-002",
			PublishedAt: now.Add(-2 * time.Hour),
			EventType:   "user.session.start",
			Severity:    "INFO",
			ActorID:     "okta-002-frank",
			ActorEmail:  "frank@example.com",
			Outcome:     "SUCCESS",
		},
		{
			EventID:     "okta-log-003",
			PublishedAt: now.Add(-3 * time.Hour),
			EventType:   "user.authentication.sso",
			Severity:    "WARN",
			ActorID:     "okta-003-grace",
			ActorEmail:  "grace@example.com",
			TargetID:    "okta-app-003",
			TargetType:  "AppInstance",
			Outcome:     "FAILURE",
		},
		{
			EventID:     "okta-log-004",
			PublishedAt: now.Add(-4 * time.Hour),
			EventType:   "user.account.lock",
			Severity:    "ERROR",
			ActorID:     "system",
			ActorEmail:  "system",
			TargetID:    "okta-003-grace",
			TargetType:  "User",
			Outcome:     "SUCCESS",
		},
		{
			EventID:     "okta-log-005",
			PublishedAt: now.Add(-5 * time.Hour),
			EventType:   "policy.evaluate_sign_on",
			Severity:    "INFO",
			ActorID:     "okta-001-eve",
			ActorEmail:  "eve@example.com",
			TargetID:    "okta-app-002",
			TargetType:  "AppInstance",
			Outcome:     "ALLOW",
		},
	}

	return &iam.IAMSnapshot{
		Provider:    a.Name(),
		CollectedAt: now,
		Users:       users,
		SystemLogs:  systemLogs,
	}
}

// oktaStatusToNormal maps Okta status strings to normalised UserStatus.
func oktaStatusToNormal(status string) iam.UserStatus {
	switch status {
	case "ACTIVE":
		return iam.UserStatusActive
	case "INACTIVE", "RECOVERY", "LOCKED_OUT":
		return iam.UserStatusInactive
	case "SUSPENDED":
		return iam.UserStatusSuspended
	case "DEPROVISIONED":
		return iam.UserStatusDeprovisioned
	default:
		return iam.UserStatusInactive
	}
}

func (a *OktaAdapter) orgBase() string {
	if a.config.OrgURL != "" {
		return a.config.OrgURL
	}
	return "https://your-org.okta.com"
}
