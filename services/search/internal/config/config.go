// Package config loads environment-based configuration for the search service.
package config

import (
	"os"
	"time"
)

// Config holds all runtime configuration for the search service.
type Config struct {
	Port            string
	DatabaseURL     string
	OpenSearchURL   string
	CDCPollInterval time.Duration
	Env             string
}

// Load reads configuration from environment variables, applying sensible defaults.
func Load() *Config {
	pollInterval, err := time.ParseDuration(getEnv("CDC_POLL_INTERVAL", "10s"))
	if err != nil {
		pollInterval = 10 * time.Second
	}
	return &Config{
		Port:            getEnv("PORT", "8089"),
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://aiauditor:aiauditor_dev@localhost:5432/aiauditor?sslmode=disable"),
		OpenSearchURL:   getEnv("OPENSEARCH_URL", "http://localhost:9200"),
		CDCPollInterval: pollInterval,
		Env:             getEnv("ENV", "development"),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

