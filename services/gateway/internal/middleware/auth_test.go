package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// generateTestKey creates a 2048-bit RSA key for tests.
func generateTestKey(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return priv, &priv.PublicKey
}

// signTestToken creates a signed RS256 JWT for testing.
func signTestToken(t *testing.T, priv *rsa.PrivateKey, claims GatewayClaims) string {
	t.Helper()
	tok := gojwt.NewWithClaims(gojwt.SigningMethodRS256, &claims)
	signed, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestAuthenticator_ValidToken(t *testing.T) {
	priv, pub := generateTestKey(t)
	verifier := NewStaticKeyVerifier(pub)
	auth := NewAuthenticator(verifier)

	claims := GatewayClaims{
		Claims: Claims{
			OrgID:   "org-1",
			Email:   "alice@example.com",
			Persona: "internal_auditor",
			RegisteredClaims: gojwt.RegisteredClaims{
				Subject:   "user-1",
				ExpiresAt: gojwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  gojwt.NewNumericDate(time.Now()),
			},
		},
		OrgTier: TierStandard,
	}
	token := signTestToken(t, priv, claims)

	var capturedClaims *GatewayClaims
	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedClaims = ClaimsFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}
	if capturedClaims == nil {
		t.Fatal("claims not set in context")
	}
	if capturedClaims.OrgID != "org-1" {
		t.Errorf("unexpected org_id: %s", capturedClaims.OrgID)
	}
}

func TestAuthenticator_MissingHeader(t *testing.T) {
	_, pub := generateTestKey(t)
	auth := NewAuthenticator(NewStaticKeyVerifier(pub))

	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rr.Code)
	}
}

func TestAuthenticator_ExpiredToken(t *testing.T) {
	priv, pub := generateTestKey(t)
	auth := NewAuthenticator(NewStaticKeyVerifier(pub))

	claims := GatewayClaims{
		Claims: Claims{
			OrgID:   "org-1",
			Persona: "internal_auditor",
			RegisteredClaims: gojwt.RegisteredClaims{
				Subject:   "user-1",
				ExpiresAt: gojwt.NewNumericDate(time.Now().Add(-1 * time.Minute)),
				IssuedAt:  gojwt.NewNumericDate(time.Now().Add(-16 * time.Minute)),
			},
		},
	}
	token := signTestToken(t, priv, claims)

	handler := auth.Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 got %d", rr.Code)
	}
}

func TestRequirePermission_Allowed(t *testing.T) {
	priv, pub := generateTestKey(t)
	auth := NewAuthenticator(NewStaticKeyVerifier(pub))

	claims := GatewayClaims{
		Claims: Claims{
			OrgID:   "org-1",
			Persona: "cae",
			RegisteredClaims: gojwt.RegisteredClaims{
				Subject:   "user-1",
				ExpiresAt: gojwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  gojwt.NewNumericDate(time.Now()),
			},
		},
	}
	token := signTestToken(t, priv, claims)

	handler := auth.Authenticate(RequirePermission("engagement", "manage")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 got %d", rr.Code)
	}
}

func TestRequirePermission_Denied(t *testing.T) {
	priv, pub := generateTestKey(t)
	auth := NewAuthenticator(NewStaticKeyVerifier(pub))

	// audit_committee can only read reports/dashboards — not manage engagements
	claims := GatewayClaims{
		Claims: Claims{
			OrgID:   "org-1",
			Persona: "audit_committee",
			RegisteredClaims: gojwt.RegisteredClaims{
				Subject:   "user-1",
				ExpiresAt: gojwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
				IssuedAt:  gojwt.NewNumericDate(time.Now()),
			},
		},
	}
	token := signTestToken(t, priv, claims)

	handler := auth.Authenticate(RequirePermission("engagement", "write")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Errorf("expected 403 got %d", rr.Code)
	}
}

func TestCan_AllPersonas(t *testing.T) {
	tests := []struct {
		persona  string
		resource string
		action   string
		want     bool
	}{
		{"internal_auditor", "engagement", "read", true},
		{"internal_auditor", "engagement", "manage", false},
		{"cae", "engagement", "manage", true},
		{"cae", "user", "manage", true},
		{"audit_committee", "report", "read", true},
		{"audit_committee", "report", "write", false},
		{"auditee_ciso", "finding", "write", true},
		{"auditee_ciso", "audit_trail", "read", false},
		{"cosourced_provider", "evidence", "write", true},
		{"cosourced_provider", "report", "read", false},
		{"ptwg_member", "work_paper", "write", true},
		{"ptwg_member", "engagement", "write", false},
		{"beta_tester", "engagement", "read", true},
		{"beta_tester", "engagement", "write", false},
		{"unknown_persona", "report", "read", false},
	}
	for _, tt := range tests {
		got := Can(tt.persona, tt.resource, tt.action)
		if got != tt.want {
			t.Errorf("Can(%q, %q, %q) = %v, want %v", tt.persona, tt.resource, tt.action, got, tt.want)
		}
	}
}
