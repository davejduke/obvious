package jwt_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	jwtpkg "github.com/davejduke/obvious/services/identity/pkg/jwt"
	"github.com/google/uuid"
)

func newTestManager(t *testing.T) *jwtpkg.Manager {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return jwtpkg.NewManagerFromKeys(priv, &priv.PublicKey, "test.issuer", 15*time.Minute, 7*24*time.Hour)
}

func TestIssueAndVerifyAccessToken(t *testing.T) {
	m := newTestManager(t)
	userID := uuid.New().String()
	orgID := uuid.New().String()

	tok, err := m.IssueAccessToken(userID, orgID, "alice@example.com", "Alice", "internal_auditor")
	if err != nil {
		t.Fatalf("IssueAccessToken: %v", err)
	}
	if tok == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := m.VerifyAccessToken(tok)
	if err != nil {
		t.Fatalf("VerifyAccessToken: %v", err)
	}
	if claims.Subject != userID {
		t.Errorf("sub: got %s, want %s", claims.Subject, userID)
	}
	if claims.OrgID != orgID {
		t.Errorf("org_id: got %s, want %s", claims.OrgID, orgID)
	}
	if claims.Persona != "internal_auditor" {
		t.Errorf("persona: got %s, want internal_auditor", claims.Persona)
	}
	if claims.Email != "alice@example.com" {
		t.Errorf("email: got %s, want alice@example.com", claims.Email)
	}
}

func TestIssueAndVerifyRefreshToken(t *testing.T) {
	m := newTestManager(t)
	userID := uuid.New().String()
	orgID := uuid.New().String()

	tok, jti, err := m.IssueRefreshToken(userID, orgID)
	if err != nil {
		t.Fatalf("IssueRefreshToken: %v", err)
	}
	if tok == "" || jti == "" {
		t.Fatal("expected non-empty token and jti")
	}

	claims, err := m.VerifyRefreshToken(tok)
	if err != nil {
		t.Fatalf("VerifyRefreshToken: %v", err)
	}
	if claims.Subject != userID {
		t.Errorf("sub: got %s, want %s", claims.Subject, userID)
	}
	if claims.ID != jti {
		t.Errorf("jti: got %s, want %s", claims.ID, jti)
	}
}

func TestExpiredAccessToken(t *testing.T) {
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	// Create manager with 0 TTL (immediately expired)
	m := jwtpkg.NewManagerFromKeys(priv, &priv.PublicKey, "test", -1*time.Second, time.Hour)
	tok, err := m.IssueAccessToken("u1", "o1", "test@test.com", "Test", "beta_tester")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	_, err = m.VerifyAccessToken(tok)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestInvalidSignature(t *testing.T) {
	m1 := newTestManager(t)
	m2 := newTestManager(t)
	// Issue with m1, verify with m2 (different keys)
	tok, _ := m1.IssueAccessToken("u1", "o1", "t@t.com", "T", "cae")
	_, err := m2.VerifyAccessToken(tok)
	if err == nil {
		t.Fatal("expected signature verification error, got nil")
	}
}
