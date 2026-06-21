// Package main is the entry point for the AIAUDITOR webhooks service.
//
// The webhooks service is responsible for:
//   - CRUD management of per-organisation webhook subscriptions
//   - HMAC-SHA256 payload signing (X-AIAUDITOR-Signature header)
//   - Asynchronous event dispatch to subscriber endpoints
//   - 3-attempt exponential backoff retry (30 s / 5 m / 30 m)
//   - Dead-letter logging for undeliverable events
//   - Admin health dashboard for delivery monitoring
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"

	svclogging "github.com/davejduke/obvious/shared/logging"
	svcmetrics "github.com/davejduke/obvious/shared/metrics"
	"github.com/davejduke/obvious/services/webhooks/internal/config"
	"github.com/davejduke/obvious/services/webhooks/internal/delivery"
	"github.com/davejduke/obvious/services/webhooks/internal/handler"
	"github.com/davejduke/obvious/services/webhooks/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("webhooks: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger := svclogging.New("webhooks")

	// Start Prometheus metrics server.
	metricsSrv := svcmetrics.StartServer(cfg.MetricsPort)
	defer svcmetrics.StopServer(metricsSrv)

	// Connect to PostgreSQL.
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("parse database URL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("database ping: %w", err)
	}
	logger.Info(context.Background(), "database.connected", nil)

	// Build service layer.
	s := store.New(pool)
	w := delivery.New(s)
	h := handler.New(s, w)

	// Start background retry worker.
	stopRetry := w.StartRetryWorker(context.Background())
	defer stopRetry()
	logger.Info(context.Background(), "retry_worker.started", nil)

	// Build Chi router.
	r := chi.NewRouter()
	r.Use(svclogging.RequestIDMiddleware)
	r.Use(svclogging.TraceContextMiddleware)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)

	// CORS for dev / inter-service access.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Org-ID")
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	// Health check (no auth required).
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"service":"webhooks","status":"healthy","version":"0.1.0"}`)
	})

	// Mount all routes at /api/v1/webhooks.
	r.Route("/api/v1/webhooks", func(r chi.Router) {
		h.Routes(r)
	})

	// Serve.
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info(context.Background(), "server.listening", map[string]any{"port": cfg.Port})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[webhooks] server error: %v", err)
		}
	}()

	// Graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info(context.Background(), "server.shutdown", nil)

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}
	logger.Info(context.Background(), "server.exited", nil)
	return nil
}

