// Package main is the entry point for the audit planning service.
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

	svclogging "github.com/davejduke/obvious/shared/logging"
	svcmetrics "github.com/davejduke/obvious/shared/metrics"
	"github.com/davejduke/obvious/services/planning/internal/handler"
	"github.com/davejduke/obvious/services/planning/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("planning: %v", err)
	}
}

func run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8095"
	}

	logger := svclogging.New("planning")

	// Start Prometheus metrics server.
	metricsSrv := svcmetrics.StartServer(os.Getenv("METRICS_PORT"))
	defer svcmetrics.StopServer(metricsSrv)

	s := store.New()
	h := handler.New(s)

	r := chi.NewRouter()
	r.Use(svclogging.RequestIDMiddleware)
	r.Use(svclogging.TraceContextMiddleware)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Recoverer)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Org-ID")
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"planning","status":"healthy","version":"0.1.0"}`))
	})
	r.Get("/ready", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ready":true}`))
	})

	r.Route("/api/v1/planning", func(api chi.Router) {
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
		logger.Info(context.Background(), "server.start", map[string]any{"port": port})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(context.Background(), "server.error", map[string]any{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	logger.Info(context.Background(), "server.shutdown", nil)
	return nil
}
