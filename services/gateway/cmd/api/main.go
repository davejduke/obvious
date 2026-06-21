// Package main is the entry point for the AIAUDITOR API Gateway service.
//
// The gateway is the single ingress point for the AIAUDITOR platform.
// It handles:
//   - JWT RS256 validation against the Identity Service JWKS
//   - RBAC enforcement (7-persona permission matrix, §4.3)
//   - Per-organisation sliding-window rate limiting backed by Redis
//     (Standard 200/min, Enterprise 1000/min, Integration 5000/min)
//   - Reverse-proxy routing to all 7 backend services
//   - Cursor-based pagination header injection
//   - CORS for the frontend origin
//   - Graceful shutdown on SIGINT/SIGTERM
package main

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/davejduke/obvious/services/gateway/internal/config"
	gatewaymw "github.com/davejduke/obvious/services/gateway/internal/middleware"
	"github.com/davejduke/obvious/services/gateway/internal/proxy"
)

func main() {
	cfg := config.Load()
	log.Printf("[gateway] starting on port %s (env=%s)", cfg.Port, cfg.Env)

	// ── Redis ───────────────────────────────────────────────────────────────────
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("[gateway] parse redis url: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("[gateway] warning: redis ping failed (%v) — rate limiter will fail open", err)
	} else {
		log.Println("[gateway] Redis connected")
	}
	defer redisClient.Close()

	// ── JWT Verifier ────────────────────────────────────────────────────────────
	var verifier gatewaymw.Verifier
	if cfg.JWTPubKeyPath != "" {
		// Static key mode (testing / environments without live Identity Service).
		pub, err := loadRSAPublicKey(cfg.JWTPubKeyPath)
		if err != nil {
			log.Fatalf("[gateway] load public key: %v", err)
		}
		verifier = gatewaymw.NewStaticKeyVerifier(pub)
		log.Printf("[gateway] JWT: static key from %s", cfg.JWTPubKeyPath)
	} else {
		// JWKS mode — fetch keys from Identity Service.
		verifier = gatewaymw.NewJWKSVerifier(cfg.IdentityJWKSURL, cfg.JWKSCacheTTL)
		log.Printf("[gateway] JWT: JWKS from %s (cache TTL=%s)", cfg.IdentityJWKSURL, cfg.JWKSCacheTTL)
	}

	// ── Middleware ──────────────────────────────────────────────────────────────
	auth := gatewaymw.NewAuthenticator(verifier)
	rl := gatewaymw.NewRateLimiter(
		redisClient,
		cfg.RateLimitStandard,
		cfg.RateLimitEnterprise,
		cfg.RateLimitIntegration,
	)

	log.Printf("[gateway] rate limits: standard=%d/min enterprise=%d/min integration=%d/min",
		cfg.RateLimitStandard, cfg.RateLimitEnterprise, cfg.RateLimitIntegration)

	// ── Router ──────────────────────────────────────────────────────────────────
	svcCfg := proxy.ServiceConfig{
		IdentityURL:    cfg.IdentityURL,
		ControlsURL:    cfg.ControlsURL,
		EvidenceURL:    cfg.EvidenceURL,
		EngagementURL:  cfg.EngagementURL,
		IntegrationURL: cfg.IntegrationURL,
		AuditTrailURL:  cfg.AuditTrailURL,
		ReportingURL:   cfg.ReportingURL,
	}
	router := proxy.Build(svcCfg, auth, rl, cfg.FrontendOrigin)

	// ── Server ──────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[gateway] listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[gateway] server error: %v", err)
		}
	}()

	// ── Graceful shutdown ────────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[gateway] shutdown signal received")

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("[gateway] forced shutdown: %v", err)
	}
	log.Println("[gateway] server exited cleanly")
}

// loadRSAPublicKey reads and parses an RSA public key PEM file.
func loadRSAPublicKey(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	pub, err := gojwt.ParseRSAPublicKeyFromPEM(b)
	if err != nil {
		return nil, fmt.Errorf("parse PEM: %w", err)
	}
	return pub, nil
}
