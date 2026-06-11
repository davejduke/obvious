// Package config loads runtime configuration for the audit-trail service.
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
}

// Load reads configuration from the environment. It returns an error if any
// required variable is missing.
func Load() (*Config, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		Env:         os.Getenv("ENV"),
	}, nil
}

