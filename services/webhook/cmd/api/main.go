// Package main is the entry point for the webhook delivery service.
package main

import (
	"context"
	"fmt"
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
	"github.com/davejduke/obvious/services/webhook/internal/handler"
	"github.com/davejduke/obvious/services/webhook/internal/store"
	"github.com/davejduke/obvious/services/webhook/internal/worker"
)

func main() {
	logger := svclogging.New("webhook")
	ctx := context.Background()

	// Start Prometheus metrics server on :9090.
	metricsSrv := svcmetrics.StartServer(os.Getenv("METRICS_PORT"))
	defer svcmetrics.StopServer(metricsSrv)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8088"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://aiauditor:aiauditor@localhost:5432/aiauditor"
	}

	dbCtx, dbCancel := context.WithTimeout(ctx, 15*time.Second)
	defer dbCancel()

	pool, err := pgxpool.New(dbCtx, dbURL)
	if err != nil {
		logger.Critical(ctx, "db.connect_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(dbCtx); err != nil {
		logger.Critical(ctx, "db.ping_failed", map[string]any{"error": err.Error()})
		os.Exit(1)
	}
	logger.Info(ctx, "db.connected", nil)

	// Wire dependencies.
	s := store.New(pool)
	h := handler.New(s)

	// Start background delivery worker.
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	wkr := worker.New(s)
	go wkr.Run(workerCtx)
	logger.Info(ctx, "worker.started", nil)

	// Build Chi router.
	r := chi.NewRouter()
	r.Use(svclogging.RequestIDMiddleware)
	r.Use(svclogging.TraceContextMiddleware)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(svcmetrics.Middleware("webhook"))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"webhook","status":"healthy","version":"0.1.0"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ready":true}`))
	})

	r.Route("/api/v1", func(api chi.Router) {
		h.Routes(api)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info(ctx, "server.start", map[string]any{"port": port})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "server.error", map[string]any{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	workerCancel()

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error(ctx, "server.shutdown_error", map[string]any{"error": err.Error()})
	}
	logger.Info(ctx, "server.shutdown", nil)
}

