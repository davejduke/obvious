// Package middleware provides Chi-compatible HTTP middleware for the API gateway.
package middleware

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
)

// ContextKey is the type used for context keys in this package.
type ContextKey string

const (
	// ClaimsKey is the context key where gateway JWT claims are stored.
	ClaimsKey ContextKey = "gateway_claims"
)

// Claims represents the JWT claims issued by the Identity Service.
type Claims struct {
	OrgID       string `json:"org_id"`
	Email       string `json:"email"`
	DisplayName string `json:"name"`
	Persona     string `json:"persona"`
	gojwt.RegisteredClaims
}

// OrgTier derives the rate-limit tier from org metadata.
// The tier is stored as a field on the Organisation record; the gateway
// receives it via a custom JWT claim "org_tier" or falls back to "standard".
type OrgTier string

const (
	TierStandard    OrgTier = "standard"
	TierEnterprise  OrgTier = "enterprise"
	TierIntegration OrgTier = "integration"
)

// GatewayClaims extends standard Claims with gateway-specific data.
type GatewayClaims struct {
	Claims
	OrgTier OrgTier `json:"org_tier,omitempty"`
}

// jwkKey is a single key from a JWKS response.
type jwkKey struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

// jwksResponse is the structure returned by the JWKS endpoint.
type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

// JWKSVerifier fetches and caches JWKS from the Identity Service.
type JWKSVerifier struct {
	url      string
	cacheTTL time.Duration
	client   *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	fetchedAt time.Time
}

// NewJWKSVerifier creates a new JWKSVerifier.
func NewJWKSVerifier(jwksURL string, cacheTTL time.Duration) *JWKSVerifier {
	return &JWKSVerifier{
		url:      jwksURL,
		cacheTTL: cacheTTL,
		client:   &http.Client{Timeout: 10 * time.Second},
		keys:     make(map[string]*rsa.PublicKey),
	}
}

// StaticKeyVerifier uses a pre-loaded RSA public key (for testing / fallback).
type StaticKeyVerifier struct {
	key *rsa.PublicKey
}

// NewStaticKeyVerifier creates a verifier from an already-parsed RSA public key.
func NewStaticKeyVerifier(key *rsa.PublicKey) *StaticKeyVerifier {
	return &StaticKeyVerifier{key: key}
}

// Verifier is the interface satisfied by both JWKS and static key verifiers.
type Verifier interface {
	VerifyToken(tokenStr string) (*GatewayClaims, error)
}

// VerifyToken verifies an RS256 token using the JWKS endpoint.
func (j *JWKSVerifier) VerifyToken(tokenStr string) (*GatewayClaims, error) {
	tok, err := gojwt.ParseWithClaims(tokenStr, &GatewayClaims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("gateway/auth: unexpected signing method: %v", t.Header["alg"])
		}
		kid, _ := t.Header["kid"].(string)
		pub, err := j.getKey(kid)
		if err != nil {
			return nil, err
		}
		return pub, nil
	})
	if err != nil {
		return nil, fmt.Errorf("gateway/auth: %w", err)
	}
	claims, ok := tok.Claims.(*GatewayClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("gateway/auth: invalid token claims")
	}
	return claims, nil
}

// getKey returns the RSA public key for the given kid, refreshing the cache if needed.
func (j *JWKSVerifier) getKey(kid string) (*rsa.PublicKey, error) {
	j.mu.RLock()
	if time.Since(j.fetchedAt) < j.cacheTTL {
		k, ok := j.keys[kid]
		j.mu.RUnlock()
		if ok {
			return k, nil
		}
		// Key not found in cache but cache is fresh — return an error.
		return nil, fmt.Errorf("gateway/auth: key id %q not found in JWKS", kid)
	}
	j.mu.RUnlock()

	// Cache is stale — refresh.
	if err := j.fetchKeys(); err != nil {
		return nil, err
	}

	j.mu.RLock()
	defer j.mu.RUnlock()
	k, ok := j.keys[kid]
	if !ok {
		// Try "default" key (no kid) when token has no kid.
		k, ok = j.keys[""]
	}
	if !ok {
		return nil, fmt.Errorf("gateway/auth: key id %q not found in JWKS after refresh", kid)
	}
	return k, nil
}

// fetchKeys fetches and parses the JWKS endpoint.
func (j *JWKSVerifier) fetchKeys() error {
	resp, err := j.client.Get(j.url)
	if err != nil {
		return fmt.Errorf("gateway/auth: fetch JWKS: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("gateway/auth: read JWKS: %w", err)
	}
	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("gateway/auth: parse JWKS: %w", err)
	}
	newKeys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		pub, err := rsaFromJWK(k.N, k.E)
		if err != nil {
			continue
		}
		newKeys[k.Kid] = pub
	}
	j.mu.Lock()
	j.keys = newKeys
	j.fetchedAt = time.Now()
	j.mu.Unlock()
	return nil
}

// rsaFromJWK constructs an *rsa.PublicKey from base64url-encoded N and E.
func rsaFromJWK(n, e string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	eInt := 0
	for _, b := range eBytes {
		eInt = eInt<<8 | int(b)
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: eInt,
	}, nil
}

// VerifyToken verifies a token using the static RSA public key.
func (s *StaticKeyVerifier) VerifyToken(tokenStr string) (*GatewayClaims, error) {
	tok, err := gojwt.ParseWithClaims(tokenStr, &GatewayClaims{}, func(t *gojwt.Token) (any, error) {
		if _, ok := t.Method.(*gojwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("gateway/auth: unexpected signing method: %v", t.Header["alg"])
		}
		return s.key, nil
	})
	if err != nil {
		return nil, fmt.Errorf("gateway/auth: %w", err)
	}
	claims, ok := tok.Claims.(*GatewayClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("gateway/auth: invalid token claims")
	}
	return claims, nil
}

// Authenticator is Chi middleware that validates JWT Bearer tokens.
type Authenticator struct {
	verifier Verifier
}

// NewAuthenticator creates an Authenticator with the given verifier.
func NewAuthenticator(v Verifier) *Authenticator {
	return &Authenticator{verifier: v}
}

// Authenticate validates the Bearer token and injects claims into the request context.
func (a *Authenticator) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeUnauthorized(w, "missing authorization header")
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			writeUnauthorized(w, "invalid authorization header format")
			return
		}
		claims, err := a.verifier.VerifyToken(parts[1])
		if err != nil {
			writeUnauthorized(w, "invalid or expired token")
			return
		}
		ctx := context.WithValue(r.Context(), ClaimsKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// ClaimsFromContext extracts GatewayClaims from the request context.
func ClaimsFromContext(ctx context.Context) *GatewayClaims {
	v := ctx.Value(ClaimsKey)
	if v == nil {
		return nil
	}
	c, _ := v.(*GatewayClaims)
	return c
}

// ── RBAC ─────────────────────────────────────────────────────────────────────

// Permission is a (resource, action) pair.
type Permission struct {
	Resource string
	Action   string
}

// personaPermissions mirrors the identity service RBAC matrix (§4.3).
// Duplicated here intentionally — the gateway must enforce RBAC independently.
var personaPermissions = map[string][]Permission{
	"internal_auditor": {
		{"engagement", "read"}, {"engagement", "write"},
		{"finding", "read"}, {"finding", "write"},
		{"control", "read"},
		{"evidence", "read"}, {"evidence", "write"},
		{"work_paper", "read"}, {"work_paper", "write"},
		{"report", "read"},
		{"dashboard", "read"},
		{"audit_trail", "read"},
	},
	"cae": {
		{"engagement", "manage"}, {"finding", "manage"},
		{"control", "manage"}, {"evidence", "manage"},
		{"work_paper", "manage"}, {"report", "manage"},
		{"dashboard", "manage"}, {"audit_trail", "read"},
		{"user", "manage"}, {"org", "read"}, {"org", "write"},
	},
	"audit_committee": {
		{"report", "read"}, {"dashboard", "read"},
		{"finding", "read"}, {"engagement", "read"},
	},
	"auditee_ciso": {
		{"finding", "read"}, {"finding", "write"},
		{"engagement", "read"}, {"control", "read"},
		{"evidence", "read"}, {"evidence", "write"},
		{"dashboard", "read"},
	},
	"cosourced_provider": {
		{"engagement", "read"},
		{"finding", "read"}, {"finding", "write"},
		{"evidence", "read"}, {"evidence", "write"},
		{"work_paper", "read"}, {"work_paper", "write"},
	},
	"ptwg_member": {
		{"work_paper", "read"}, {"work_paper", "write"},
		{"control", "read"}, {"finding", "read"}, {"report", "read"},
	},
	"beta_tester": {
		{"engagement", "read"}, {"finding", "read"}, {"control", "read"},
		{"evidence", "read"}, {"report", "read"},
		{"dashboard", "read"}, {"work_paper", "read"},
	},
}

// Can returns true if the persona can perform action on resource.
func Can(persona, resource, action string) bool {
	perms, ok := personaPermissions[persona]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p.Resource == resource {
			if p.Action == action || p.Action == "manage" {
				return true
			}
		}
	}
	return false
}

// RequirePermission returns Chi middleware that enforces an RBAC permission check.
func RequirePermission(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil {
				writeUnauthorized(w, "authentication required")
				return
			}
			if !Can(claims.Persona, resource, action) {
				writeForbidden(w, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"code":"UNAUTHORIZED","message":"` + msg + `"}`)) //nolint:errcheck
}

func writeForbidden(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte(`{"code":"FORBIDDEN","message":"` + msg + `"}`)) //nolint:errcheck
}
