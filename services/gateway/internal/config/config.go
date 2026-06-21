// Package config holds runtime configuration for the gateway service.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the API gateway.
type Config struct {
	Port           string
	RedisURL       string
	Env            string
	FrontendOrigin string

	// JWT / Identity
	IdentityJWKSURL   string // e.g. http://identity:8081/.well-known/jwks.json
	JWTPubKeyPath     string // fallback: load RSA public key from file
	JWKSCacheTTL      time.Duration

	// Rate-limit tier caps (requests per minute)
	RateLimitStandard    int
	RateLimitEnterprise  int
	RateLimitIntegration int

	// Upstream service base URLs
	IdentityURL   string
	ControlsURL   string
	EvidenceURL   string
	EngagementURL string
	IntegrationURL string
	AuditTrailURL string
	ReportingURL  string
	WebhooksURL   string
}

// Load reads config from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:           getenv("PORT", "8080"),
		RedisURL:       getenv("REDIS_URL", "redis://localhost:6379"),
		Env:            getenv("ENV", "development"),
		FrontendOrigin: getenv("FRONTEND_ORIGIN", "http://localhost:3000"),

		IdentityJWKSURL:   getenv("IDENTITY_JWKS_URL", "http://identity:8081/.well-known/jwks.json"),
		JWTPubKeyPath:     getenv("JWT_PUBLIC_KEY_PATH", ""),
		JWKSCacheTTL:      parseDuration(getenv("JWKS_CACHE_TTL", "5m"), 5*time.Minute),

		RateLimitStandard:    parseInt(getenv("RATE_LIMIT_STANDARD", "200"), 200),
		RateLimitEnterprise:  parseInt(getenv("RATE_LIMIT_ENTERPRISE", "1000"), 1000),
		RateLimitIntegration: parseInt(getenv("RATE_LIMIT_INTEGRATION", "5000"), 5000),

		IdentityURL:    getenv("IDENTITY_URL", "http://identity:8081"),
		ControlsURL:    getenv("CONTROLS_URL", "http://control-framework:8082"),
		EvidenceURL:    getenv("EVIDENCE_URL", "http://evidence:8083"),
		EngagementURL:  getenv("ENGAGEMENT_URL", "http://engagement:8084"),
		IntegrationURL: getenv("INTEGRATION_URL", "http://integration:8085"),
		AuditTrailURL:  getenv("AUDIT_TRAIL_URL", "http://audit-trail:8086"),
		ReportingURL:   getenv("REPORTING_URL", "http://reporting:8087"),
		WebhooksURL:    getenv("WEBHOOKS_URL", "http://webhooks:8088"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

func parseDuration(s string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return def
	}
	return d
}
