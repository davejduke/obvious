// Package auth provides authentication provider implementations for the
// Connector SDK. Three auth types are supported:
//
//   - API Key  — adds a static token via a configurable HTTP header (or query param).
//   - OAuth 2.0 Client Credentials — fetches and automatically refreshes bearer
//     tokens using the RFC 6749 client_credentials grant.
//   - Certificate (mTLS) — configures the caller's http.Client with a client TLS
//     certificate for mutual TLS authentication.
package auth

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// AuthProvider interface
// ---------------------------------------------------------------------------

// AuthProvider augments an outgoing HTTP request with credentials.
// Implementations must be safe for concurrent use.
type AuthProvider interface {
	// AddAuth mutates req to inject credentials (headers, query params, etc.).
	// ctx is forwarded to any token-refresh network calls.
	AddAuth(ctx context.Context, req *http.Request) error

	// Scheme returns a short human-readable name for logging (e.g. "apikey").
	Scheme() string
}

// ---------------------------------------------------------------------------
// 1. API Key
// ---------------------------------------------------------------------------

// APIKeyLocation specifies where the key is injected.
type APIKeyLocation string

const (
	// APIKeyHeader injects the key as an HTTP header (default).
	APIKeyHeader APIKeyLocation = "header"
	// APIKeyQuery injects the key as a URL query parameter.
	APIKeyQuery APIKeyLocation = "query"
)

// APIKeyConfig holds settings for API key authentication.
type APIKeyConfig struct {
	// Key is the secret token value.
	Key string

	// HeaderName is the HTTP header to set when Location is APIKeyHeader.
	// Defaults to "Authorization" with a "Bearer " prefix when empty.
	HeaderName string

	// Prefix is prepended to Key in the header value (e.g. "Bearer ", "Token ").
	// Ignored when Location is APIKeyQuery.
	Prefix string

	// Location controls where the key is placed. Defaults to APIKeyHeader.
	Location APIKeyLocation

	// QueryParam is the query parameter name when Location is APIKeyQuery.
	QueryParam string
}

// APIKeyAuth implements AuthProvider using a static API key.
type APIKeyAuth struct {
	cfg APIKeyConfig
}

// NewAPIKeyAuth returns an AuthProvider that injects a static API key.
func NewAPIKeyAuth(cfg APIKeyConfig) *APIKeyAuth {
	if cfg.Location == "" {
		cfg.Location = APIKeyHeader
	}
	if cfg.Location == APIKeyHeader && cfg.HeaderName == "" {
		cfg.HeaderName = "Authorization"
		if cfg.Prefix == "" {
			cfg.Prefix = "Bearer "
		}
	}
	if cfg.Location == APIKeyQuery && cfg.QueryParam == "" {
		cfg.QueryParam = "api_key"
	}
	return &APIKeyAuth{cfg: cfg}
}

// AddAuth implements AuthProvider.
func (a *APIKeyAuth) AddAuth(_ context.Context, req *http.Request) error {
	switch a.cfg.Location {
	case APIKeyQuery:
		q := req.URL.Query()
		q.Set(a.cfg.QueryParam, a.cfg.Key)
		req.URL.RawQuery = q.Encode()
	default: // header
		req.Header.Set(a.cfg.HeaderName, a.cfg.Prefix+a.cfg.Key)
	}
	return nil
}

// Scheme implements AuthProvider.
func (a *APIKeyAuth) Scheme() string { return "apikey" }

// ---------------------------------------------------------------------------
// 2. OAuth 2.0 Client Credentials
// ---------------------------------------------------------------------------

// OAuth2Config holds settings for the client_credentials grant.
type OAuth2Config struct {
	// TokenURL is the IdP token endpoint (e.g. https://auth.example.com/token).
	TokenURL string

	// ClientID is the OAuth 2.0 application client identifier.
	ClientID string

	// ClientSecret is the OAuth 2.0 application client secret.
	ClientSecret string

	// Scopes lists the requested OAuth scopes (space-separated or individual).
	Scopes []string

	// ExpiryDelta is how early to refresh before actual expiry. Defaults to 30s.
	ExpiryDelta time.Duration

	// HTTPClient is used for token refresh requests. Uses http.DefaultClient if nil.
	HTTPClient *http.Client
}

// oauth2Token holds a cached access token.
type oauth2Token struct {
	accessToken string
	expiry      time.Time
}

// expired reports whether the token should be refreshed.
func (t *oauth2Token) expired(delta time.Duration) bool {
	if t.accessToken == "" {
		return true
	}
	return time.Now().Add(delta).After(t.expiry)
}

// OAuth2ClientCredentials implements AuthProvider using the OAuth 2.0
// client_credentials grant (RFC 6749 §4.4). Tokens are cached and
// automatically refreshed when they approach expiry.
type OAuth2ClientCredentials struct {
	cfg    OAuth2Config
	mu     sync.Mutex
	cached *oauth2Token
}

// NewOAuth2ClientCredentials returns an AuthProvider for the OAuth 2.0
// client_credentials grant.
func NewOAuth2ClientCredentials(cfg OAuth2Config) *OAuth2ClientCredentials {
	if cfg.ExpiryDelta == 0 {
		cfg.ExpiryDelta = 30 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	return &OAuth2ClientCredentials{cfg: cfg}
}

// AddAuth implements AuthProvider. It refreshes the token if needed.
func (o *OAuth2ClientCredentials) AddAuth(ctx context.Context, req *http.Request) error {
	token, err := o.token(ctx)
	if err != nil {
		return fmt.Errorf("oauth2: fetch token: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// Scheme implements AuthProvider.
func (o *OAuth2ClientCredentials) Scheme() string { return "oauth2_client_credentials" }

// token returns a valid access token, refreshing if necessary.
func (o *OAuth2ClientCredentials) token(ctx context.Context) (string, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.cached != nil && !o.cached.expired(o.cfg.ExpiryDelta) {
		return o.cached.accessToken, nil
	}

	fresh, err := o.fetchToken(ctx)
	if err != nil {
		return "", err
	}
	o.cached = fresh
	return fresh.accessToken, nil
}

// fetchToken performs the client_credentials token exchange.
func (o *OAuth2ClientCredentials) fetchToken(ctx context.Context) (*oauth2Token, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", o.cfg.ClientID)
	form.Set("client_secret", o.cfg.ClientSecret)
	if len(o.cfg.Scopes) > 0 {
		form.Set("scope", strings.Join(o.cfg.Scopes, " "))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.cfg.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := o.cfg.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token endpoint returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var payload struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}
	if payload.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned empty access_token")
	}

	expiry := time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return &oauth2Token{accessToken: payload.AccessToken, expiry: expiry}, nil
}

// ---------------------------------------------------------------------------
// 3. Certificate (mTLS)
// ---------------------------------------------------------------------------

// CertificateConfig holds settings for mutual TLS authentication.
type CertificateConfig struct {
	// CertPEM is the PEM-encoded client certificate.
	CertPEM []byte

	// KeyPEM is the PEM-encoded private key for the client certificate.
	KeyPEM []byte

	// InsecureSkipVerify disables server certificate validation.
	// Only for testing — never set true in production.
	InsecureSkipVerify bool
}

// CertificateAuth implements AuthProvider using a client TLS certificate.
// It configures the provided *http.Client's Transport with the mTLS cert.
// Call ConfigureClient before making requests.
type CertificateAuth struct {
	cfg  CertificateConfig
	cert tls.Certificate
}

// NewCertificateAuth parses the certificate and key and returns a CertificateAuth.
func NewCertificateAuth(cfg CertificateConfig) (*CertificateAuth, error) {
	cert, err := tls.X509KeyPair(cfg.CertPEM, cfg.KeyPEM)
	if err != nil {
		return nil, fmt.Errorf("certificate auth: parse key pair: %w", err)
	}
	return &CertificateAuth{cfg: cfg, cert: cert}, nil
}

// ConfigureClient attaches the client certificate to client's TLS config.
// Call once when constructing the HTTP client for the connector.
func (c *CertificateAuth) ConfigureClient(client *http.Client) {
	tlsCfg := &tls.Config{
		Certificates:       []tls.Certificate{c.cert},
		InsecureSkipVerify: c.cfg.InsecureSkipVerify, //nolint:gosec // caller opt-in
	}
	if t, ok := client.Transport.(*http.Transport); ok {
		t.TLSClientConfig = tlsCfg
	} else {
		client.Transport = &http.Transport{TLSClientConfig: tlsCfg}
	}
}

// AddAuth implements AuthProvider. For mTLS the credentials are in the TLS
// handshake, not the request itself, so this is a no-op.
func (c *CertificateAuth) AddAuth(_ context.Context, _ *http.Request) error {
	return nil
}

// Scheme implements AuthProvider.
func (c *CertificateAuth) Scheme() string { return "certificate" }
