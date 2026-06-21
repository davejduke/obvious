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

// EntraIDConfig holds Azure Entra ID (Azure AD) connection settings.
type EntraIDConfig struct {
	// TenantID is the Azure AD tenant GUID.
	TenantID string
	// ClientID is the service principal application ID.
	ClientID string
	// ClientSecret is the service principal secret (populated from env/vault).
	ClientSecret string
	// BaseURL overrides the default Graph API endpoint (for sandbox / mock HTTP server).
	BaseURL string
	// MockMode returns deterministic synthetic data without making real API calls.
	MockMode bool
}

// EntraIDAdapter syncs IAM data from Microsoft Entra ID via the Graph API.
// It implements iam.IAMConnector.
type EntraIDAdapter struct {
	config EntraIDConfig
	client *http.Client
}

// NewEntraIDAdapter creates a ready-to-use Entra ID connector.
func NewEntraIDAdapter(cfg EntraIDConfig) *EntraIDAdapter {
	return &EntraIDAdapter{
		config: cfg,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name implements iam.IAMConnector.
func (a *EntraIDAdapter) Name() string { return "entraid" }

// Sync fetches a full IAM snapshot from Entra ID.
// In mock mode it returns deterministic synthetic data.
func (a *EntraIDAdapter) Sync(ctx context.Context) (*iam.IAMSnapshot, error) {
	if a.config.MockMode {
		return a.mockSnapshot(), nil
	}
	return a.fetchFromAPI(ctx)
}

// Health reports whether the Graph API endpoint is reachable.
func (a *EntraIDAdapter) Health(ctx context.Context) connector.HealthStatus {
	status := connector.HealthStatus{
		Connector:   a.Name(),
		LastChecked: time.Now().UTC(),
	}
	if a.config.MockMode {
		status.Healthy = true
		status.Message = "mock mode: healthy"
		return status
	}

	url := fmt.Sprintf("%s/v1.0/organization", a.graphBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		status.Message = "failed to build request: " + err.Error()
		return status
	}
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

// fetchFromAPI calls the Microsoft Graph API for all IAM data.
func (a *EntraIDAdapter) fetchFromAPI(ctx context.Context) (*iam.IAMSnapshot, error) {
	users, err := a.fetchUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("entraid: fetch users: %w", err)
	}

	reviews, err := a.fetchAccessReviews(ctx)
	if err != nil {
		return nil, fmt.Errorf("entraid: fetch access reviews: %w", err)
	}

	groups, err := a.fetchGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("entraid: fetch groups: %w", err)
	}

	policies, err := a.fetchConditionalAccessPolicies(ctx)
	if err != nil {
		return nil, fmt.Errorf("entraid: fetch conditional access policies: %w", err)
	}

	return &iam.IAMSnapshot{
		Provider:            a.Name(),
		CollectedAt:         time.Now().UTC(),
		Users:               users,
		AccessReviews:       reviews,
		GroupMemberships:    groups,
		ConditionalPolicies: policies,
	}, nil
}

func (a *EntraIDAdapter) fetchUsers(ctx context.Context) ([]iam.IAMUser, error) {
	url := fmt.Sprintf(
		"%s/v1.0/users?$select=id,displayName,mail,accountEnabled,createdDateTime",
		a.graphBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
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

	var result struct {
		Value []struct {
			ID              string `json:"id"`
			DisplayName     string `json:"displayName"`
			Mail            string `json:"mail"`
			AccountEnabled  bool   `json:"accountEnabled"`
			CreatedDateTime string `json:"createdDateTime"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	users := make([]iam.IAMUser, 0, len(result.Value))
	for _, u := range result.Value {
		status := iam.UserStatusActive
		if !u.AccountEnabled {
			status = iam.UserStatusInactive
		}
		created, _ := time.Parse(time.RFC3339, u.CreatedDateTime)
		users = append(users, iam.IAMUser{
			ID:          fmt.Sprintf("entraid-%s", u.ID),
			ExternalID:  u.ID,
			Email:       u.Mail,
			DisplayName: u.DisplayName,
			Status:      status,
			CreatedAt:   created,
			Provider:    a.Name(),
		})
	}
	return users, nil
}

func (a *EntraIDAdapter) fetchAccessReviews(ctx context.Context) ([]iam.AccessReview, error) {
	// GET identity governance access review instances with decisions
	// Full pagination omitted for brevity; real impl would walk @odata.nextLink
	url := fmt.Sprintf("%s/v1.0/identityGovernance/accessReviews/definitions?$top=50", a.graphBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// Simplified mapping — real impl would expand instances and decisions
	return []iam.AccessReview{}, nil
}

func (a *EntraIDAdapter) fetchGroups(ctx context.Context) ([]iam.GroupMembership, error) {
	url := fmt.Sprintf("%s/v1.0/groups?$select=id,displayName,description&$top=100", a.graphBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
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

	var result struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	groups := make([]iam.GroupMembership, 0, len(result.Value))
	for _, g := range result.Value {
		groups = append(groups, iam.GroupMembership{
			GroupID:     g.ID,
			GroupName:   g.DisplayName,
			Description: g.Description,
		})
	}
	return groups, nil
}

func (a *EntraIDAdapter) fetchConditionalAccessPolicies(ctx context.Context) ([]iam.ConditionalAccessPolicy, error) {
	url := fmt.Sprintf("%s/v1.0/identity/conditionalAccess/policies", a.graphBase())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
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

	var result struct {
		Value []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			State       string `json:"state"`
			GrantControls struct {
				BuiltInControls []string `json:"builtInControls"`
			} `json:"grantControls"`
		} `json:"value"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	policies := make([]iam.ConditionalAccessPolicy, 0, len(result.Value))
	for _, p := range result.Value {
		policies = append(policies, iam.ConditionalAccessPolicy{
			ID:            p.ID,
			DisplayName:   p.DisplayName,
			State:         p.State,
			GrantControls: p.GrantControls.BuiltInControls,
		})
	}
	return policies, nil
}

// ---------------------------------------------------------------------------
// Mock snapshot — deterministic, realistic Entra ID data
// ---------------------------------------------------------------------------

func (a *EntraIDAdapter) mockSnapshot() *iam.IAMSnapshot {
	now := time.Now().UTC()
	lastWeek := now.Add(-7 * 24 * time.Hour)
	lastYear := now.Add(-400 * 24 * time.Hour)

	users := []iam.IAMUser{
		{
			ID:          "entraid-user-001",
			ExternalID:  "aad-001-alice",
			Email:       "alice@example.com",
			DisplayName: "Alice Johnson",
			Status:      iam.UserStatusActive,
			CreatedAt:   now.Add(-365 * 24 * time.Hour),
			LastLoginAt: &lastWeek,
			MFAEnrolled: true,
			MFAFactors: []iam.MFAFactor{
				{ID: "aad-factor-001", Type: "fido2", Status: "active", Provider: "entraid"},
				{ID: "aad-factor-002", Type: "totp", Status: "active", Provider: "entraid"},
			},
			Groups:   []string{"Engineering", "Admins"},
			Provider: "entraid",
		},
		{
			ID:          "entraid-user-002",
			ExternalID:  "aad-002-bob",
			Email:       "bob@example.com",
			DisplayName: "Bob Smith",
			Status:      iam.UserStatusActive,
			CreatedAt:   now.Add(-180 * 24 * time.Hour),
			LastLoginAt: &lastWeek,
			MFAEnrolled: true,
			MFAFactors: []iam.MFAFactor{
				{ID: "aad-factor-003", Type: "sms", Status: "active", Provider: "entraid"},
			},
			Groups:   []string{"Engineering"},
			Provider: "entraid",
		},
		{
			// Inactive user with lingering app access — triggers high-risk provisioning evidence.
			ID:          "entraid-user-003",
			ExternalID:  "aad-003-charlie",
			Email:       "charlie@example.com",
			DisplayName: "Charlie Brown",
			Status:      iam.UserStatusInactive,
			CreatedAt:   now.Add(-500 * 24 * time.Hour),
			LastLoginAt: &lastYear,
			MFAEnrolled: false,
			AppAssignments: []iam.AppAssignment{
				{AppID: "app-001", AppName: "GitHub Enterprise", Status: "active"},
			},
			Provider: "entraid",
		},
		{
			// Active user who has never logged in — orphan risk.
			ID:          "entraid-user-004",
			ExternalID:  "aad-004-diana",
			Email:       "diana@example.com",
			DisplayName: "Diana Prince",
			Status:      iam.UserStatusActive,
			CreatedAt:   now.Add(-30 * 24 * time.Hour),
			MFAEnrolled: false,
			Provider:    "entraid",
		},
	}

	accessReviews := []iam.AccessReview{
		{
			ID:            "review-001",
			ReviewerID:    "aad-001-alice",
			ResourceID:    "app-001",
			Decision:      "deny",
			Justification: "User no longer needs access",
			CompletedAt:   now.Add(-48 * time.Hour),
		},
		{
			ID:          "review-002",
			ReviewerID:  "aad-001-alice",
			ResourceID:  "app-002",
			Decision:    "approve",
			CompletedAt: now.Add(-24 * time.Hour),
		},
	}

	groups := []iam.GroupMembership{
		{
			GroupID:     "grp-001",
			GroupName:   "Engineering",
			Description: "Engineering team",
			MemberIDs:   []string{"aad-001-alice", "aad-002-bob"},
		},
		{
			GroupID:     "grp-002",
			GroupName:   "Admins",
			Description: "Platform administrators",
			MemberIDs:   []string{"aad-001-alice"},
		},
		{
			GroupID:     "grp-003",
			GroupName:   "Finance",
			Description: "Finance department",
			MemberIDs:   []string{},
		},
	}

	policies := []iam.ConditionalAccessPolicy{
		{
			ID:            "cap-001",
			DisplayName:   "Require MFA for All Users",
			State:         "enabled",
			GrantControls: []string{"mfa"},
		},
		{
			ID:            "cap-002",
			DisplayName:   "Block Legacy Authentication",
			State:         "enabled",
			GrantControls: []string{"block"},
		},
		{
			ID:            "cap-003",
			DisplayName:   "Require Compliant Device for Admins",
			State:         "enabledForReportingButNotEnforced",
			GrantControls: []string{"compliantDevice", "mfa"},
		},
	}

	return &iam.IAMSnapshot{
		Provider:            a.Name(),
		CollectedAt:         now,
		Users:               users,
		AccessReviews:       accessReviews,
		GroupMemberships:    groups,
		ConditionalPolicies: policies,
	}
}

func (a *EntraIDAdapter) graphBase() string {
	if a.config.BaseURL != "" {
		return a.config.BaseURL
	}
	return "https://graph.microsoft.com"
}
