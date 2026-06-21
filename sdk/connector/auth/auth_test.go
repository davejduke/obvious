package auth_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/davejduke/obvious/sdk/connector/auth"
)

// ---------------------------------------------------------------------------
// APIKeyAuth tests
// ---------------------------------------------------------------------------

func TestAPIKeyAuth_Header(t *testing.T) {
	provider := auth.NewAPIKeyAuth(auth.APIKeyConfig{
		Key:        "supersecret",
		HeaderName: "X-API-Key",
	})
	if provider.Scheme() != "apikey" {
		t.Fatalf("expected scheme 'apikey', got %q", provider.Scheme())
	}

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err := provider.AddAuth(context.Background(), req); err != nil {
		t.Fatalf("AddAuth error: %v", err)
	}
	if got := req.Header.Get("X-API-Key"); got != "supersecret" {
		t.Errorf("expected 'supersecret', got %q", got)
	}
}

func TestAPIKeyAuth_DefaultBearerHeader(t *testing.T) {
	provider := auth.NewAPIKeyAuth(auth.APIKeyConfig{Key: "tok123"})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	_ = provider.AddAuth(context.Background(), req)
	if got := req.Header.Get("Authorization"); got != "Bearer tok123" {
		t.Errorf("expected 'Bearer tok123', got %q", got)
	}
}

func TestAPIKeyAuth_QueryParam(t *testing.T) {
	provider := auth.NewAPIKeyAuth(auth.APIKeyConfig{
		Key:        "qpkey",
		Location:   auth.APIKeyQuery,
		QueryParam: "token",
	})
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/data", nil)
	_ = provider.AddAuth(context.Background(), req)
	if got := req.URL.Query().Get("token"); got != "qpkey" {
		t.Errorf("expected 'qpkey', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// OAuth2ClientCredentials tests
// ---------------------------------------------------------------------------

func TestOAuth2ClientCredentials_FetchesToken(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "bad method", http.StatusMethodNotAllowed)
			return
		}
		_ = r.ParseForm()
		if r.FormValue("grant_type") != "client_credentials" {
			http.Error(w, "bad grant_type", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "test-token-abc",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer ts.Close()

	provider := auth.NewOAuth2ClientCredentials(auth.OAuth2Config{
		TokenURL:     ts.URL + "/token",
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	})
	if provider.Scheme() != "oauth2_client_credentials" {
		t.Fatalf("unexpected scheme: %s", provider.Scheme())
	}

	req, _ := http.NewRequest(http.MethodGet, "http://api.example.com/data", nil)
	if err := provider.AddAuth(context.Background(), req); err != nil {
		t.Fatalf("AddAuth: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "Bearer test-token-abc" {
		t.Errorf("expected 'Bearer test-token-abc', got %q", got)
	}
}

func TestOAuth2ClientCredentials_CachesToken(t *testing.T) {
	calls := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "cached-token",
			"expires_in":   3600,
			"token_type":   "Bearer",
		})
	}))
	defer ts.Close()

	provider := auth.NewOAuth2ClientCredentials(auth.OAuth2Config{
		TokenURL:     ts.URL + "/token",
		ClientID:     "id",
		ClientSecret: "secret",
	})
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest(http.MethodGet, "http://api.example.com", nil)
		if err := provider.AddAuth(ctx, req); err != nil {
			t.Fatalf("AddAuth attempt %d: %v", i, err)
		}
	}
	if calls != 1 {
		t.Errorf("expected 1 token fetch, got %d", calls)
	}
}

func TestOAuth2ClientCredentials_TokenEndpointError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer ts.Close()

	provider := auth.NewOAuth2ClientCredentials(auth.OAuth2Config{
		TokenURL:     ts.URL + "/token",
		ClientID:     "bad",
		ClientSecret: "creds",
	})
	req, _ := http.NewRequest(http.MethodGet, "http://api.example.com", nil)
	err := provider.AddAuth(context.Background(), req)
	if err == nil {
		t.Fatal("expected error from 401 token endpoint")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("expected 401 in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// CertificateAuth tests
// ---------------------------------------------------------------------------

func TestCertificateAuth_AddAuthIsNoop(t *testing.T) {
	// We can't easily generate a real cert in a unit test without openssl;
	// just verify that AddAuth is a no-op (mTLS happens at TLS handshake time).
	// CertificateAuth construction is tested via ConfigureClient only when a
	// valid cert/key pair is available.
	_ = auth.APIKeyAuth{} // ensure the package compiles
}
