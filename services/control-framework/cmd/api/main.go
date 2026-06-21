// Package main is the entry point for the control-framework service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	svclogging "github.com/davejduke/obvious/shared/logging"
	svcmetrics "github.com/davejduke/obvious/shared/metrics"
	"github.com/davejduke/obvious/services/control-framework/internal/handlers"
	"github.com/davejduke/obvious/services/control-framework/internal/middleware"
	"github.com/davejduke/obvious/services/control-framework/internal/repository"
	"github.com/davejduke/obvious/services/control-framework/internal/service"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := svclogging.New("control-framework")

	// Start Prometheus metrics server on :9090.
	metricsSrv := svcmetrics.StartServer(os.Getenv("METRICS_PORT"))
	defer svcmetrics.StopServer(metricsSrv)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	// Build repository (use MemoryRepository if no DATABASE_URL)
	var repo repository.Repository
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			logger.Critical(context.Background(), "db.connect_failed", map[string]any{"error": err.Error()})
os.Exit(1)
		}
		defer pool.Close()
		repo = repository.NewPostgresRepository(pool)
		logger.Info(context.Background(), "db.connected", nil)
	} else {
		repo = repository.NewMemoryRepository()
		logger.Warn(context.Background(), "db.using_memory", map[string]any{"reason": "no DATABASE_URL"})
	}

	svc := service.New(repo)

	// Handlers
	fwHandler := handlers.NewFrameworkHandler(svc)
	ctrlHandler := handlers.NewControlHandler(svc)

	// Router
	r := chi.NewRouter()
	r.Use(svclogging.RequestIDMiddleware)
	r.Use(svclogging.TraceContextMiddleware)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(svcmetrics.Middleware("control-framework"))

	// Health / ready (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"control-framework","status":"healthy","version":"0.1.0"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ready":true}`))
	})

	// NIS 2 domains reference (public)
	r.Get("/frameworks/nis2/domains", fwHandler.GetNIS2Domains)

	// Authenticated API routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.DevAuth)

		// Framework routes
		r.Get("/frameworks", fwHandler.ListFrameworks)
		r.Post("/frameworks", fwHandler.CreateFramework)
		r.Get("/frameworks/{id}", fwHandler.GetFramework)

		// Control routes
		r.Get("/controls", ctrlHandler.ListControls)
		r.Post("/controls", ctrlHandler.CreateControl)
		r.Get("/controls/{id}", ctrlHandler.GetControl)
		r.Put("/controls/{id}", ctrlHandler.UpdateControl)
		r.Get("/controls/{id}/mappings", ctrlHandler.GetControlMappings)
		r.Get("/controls/{id}/evidence-requirements", ctrlHandler.GetEvidenceRequirements)
		r.Post("/controls/{id}/assess", ctrlHandler.AssessControl)
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info(context.Background(), "server.start", map[string]any{"port": port})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(context.Background(), "server.error", map[string]any{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error(context.Background(), "server.shutdown_error", map[string]any{"error": err.Error()})
	}
	logger.Info(context.Background(), "server.shutdown", nil)
}
