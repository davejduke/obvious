// Package config holds runtime configuration for the identity service.
package config

import (
	"os"
	"strconv"
)

// Config holds all runtime configuration.
type Config struct {
	Port           string
	DatabaseURL    string
	RedisURL       string
	JWTPrivKeyPath string
	JWTPubKeyPath  string
	AccessTokenTTL int // minutes
	RefreshTokenTTL int // days
	Env            string
}

// Load reads config from environment variables with defaults.
func Load() *Config {
	accessTTL, _ := strconv.Atoi(getenv("ACCESS_TOKEN_TTL_MINUTES", "15"))
	refreshTTL, _ := strconv.Atoi(getenv("REFRESH_TOKEN_TTL_DAYS", "7"))
	return &Config{
		Port:            getenv("PORT", "8081"),
		DatabaseURL:     getenv("DATABASE_URL", "postgres://aiauditor:aiauditor@localhost:5432/aiauditor?sslmode=disable"),
		RedisURL:        getenv("REDIS_URL", "redis://localhost:6379"),
		JWTPrivKeyPath:  getenv("JWT_PRIVATE_KEY_PATH", "/run/secrets/jwt_private.pem"),
		JWTPubKeyPath:   getenv("JWT_PUBLIC_KEY_PATH", "/run/secrets/jwt_public.pem"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		Env:             getenv("ENV", "development"),
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
