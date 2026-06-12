// Package jwt provides RS256 JWT issuance and verification for the identity service.
package jwt

import (
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager handles JWT signing and verification.
type Manager struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// AccessClaims are embedded in the short-lived access token.
type AccessClaims struct {
	OrgID       string `json:"org_id"`
	Email       string `json:"email"`
	DisplayName string `json:"name"`
	Persona     string `json:"persona"`
	gojwt.RegisteredClaims
}

// RefreshClaims are embedded in the long-lived refresh token.
type RefreshClaims struct {
	OrgID string `json:"org_id"`
	gojwt.RegisteredClaims
}

// NewManager creates a Manager from PEM file paths.
func NewManager(privPath, pubPath, issuer string, accessTTL, refreshTTL time.Duration) (*Manager, error) {
	privBytes, err := os.ReadFile(privPath)
	if err != nil {
		return nil, fmt.Errorf("jwt: read private key: %w", err)
	}
	priv, err := gojwt.ParseRSAPrivateKeyFromPEM(privBytes)
	if err != nil {
		return nil, fmt.Errorf("jwt: parse private key: %w", err)
	}

	pubBytes, err := os.ReadFile(pubPath)
	if err != nil {
		return nil, fmt.Errorf("jwt: read public key: %w", err)
	}
	pub, err := gojwt.ParseRSAPublicKeyFromPEM(pubBytes)
	if err != nil {
		return nil, fmt.Errorf("jwt: parse public key: %w", err)
	}

	return &Manager{
		privateKey: priv,
		publicKey:  pub,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

// NewManagerFromKeys creates a Manager from in-memory RSA keys (for testing).
func NewManagerFromKeys(priv *rsa.PrivateKey, pub *rsa.PublicKey, issuer string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		privateKey: priv,
		publicKey:  pub,
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}
}

// IssueAccessToken signs a new RS256 access token.
func (m *Manager) IssueAccessToken(userID, orgID, email, displayName, persona string) (string, error) {
	now := time.Now()
	claims := AccessClaims{
		OrgID:       orgID,
		Email:       email,
		DisplayName: displayName,
		Persona:     persona,
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.accessTTL)),
			ID:        uuid.New().String(),
		},
	}
	tok := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	return tok.SignedString(m.privateKey)
}

// IssueRefreshToken signs a new RS256 refresh token.
func (m *Manager) IssueRefreshToken(userID, orgID string) (string, string, error) {
	now := time.Now()
	jti := uuid.New().String()
	claims := RefreshClaims{
		OrgID: orgID,
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  gojwt.NewNumericDate(now),
			ExpiresAt: gojwt.NewNumericDate(now.Add(m.refreshTTL)),
			ID:        jti,
		},
	}
	tok := gojwt.NewWithClaims(gojwt.SigningMethodRS256, claims)
	signed, err := tok.SignedString(m.privateKey)
	return signed, jti, err
}

// VerifyAccessToken parses and validates an access token.
func (m *Manager) VerifyAccessToken(tokenStr string) (*AccessClaims, error) {
	tok, err := gojwt.ParseWithClaims(tokenStr, &AccessClaims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return m.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt: verify: %w", err)
	}
	claims, ok := tok.Claims.(*AccessClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("jwt: invalid token claims")
	}
	return claims, nil
}

// VerifyRefreshToken parses and validates a refresh token.
func (m *Manager) VerifyRefreshToken(tokenStr string) (*RefreshClaims, error) {
	tok, err := gojwt.ParseWithClaims(tokenStr, &RefreshClaims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("jwt: unexpected signing method: %v", t.Header["alg"])
		}
		return m.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("jwt: verify refresh: %w", err)
	}
	claims, ok := tok.Claims.(*RefreshClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("jwt: invalid refresh token")
	}
	return claims, nil
}

// PublicKey returns the RSA public key (used by other services for verification).
func (m *Manager) PublicKey() *rsa.PublicKey {
	return m.publicKey
}
