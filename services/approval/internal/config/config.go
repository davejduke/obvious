// Package config loads runtime configuration for the approval service.
package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration for the approval service.
type Config struct {
	Port        string
	DatabaseURL string
	Env         string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		Env:         os.Getenv("ENV"),
	}, nil
}
