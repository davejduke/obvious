// Package config loads runtime configuration for the webhooks service.
package config

import (
	"fmt"
	"os"
)

// Config holds service configuration loaded from environment variables.
type Config struct {
	Port        string
	DatabaseURL string
	Env         string
	MetricsPort string
}

// Load reads configuration from the environment. Returns an error if any
// required variable is missing.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		Env:         os.Getenv("ENV"),
		MetricsPort: os.Getenv("METRICS_PORT"),
	}, nil
}

