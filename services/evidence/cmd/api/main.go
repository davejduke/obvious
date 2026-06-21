// Package main is the entry point for the evidence service.
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

	"github.com/davejduke/obvious/services/evidence/internal/handler"

	svclogging "github.com/davejduke/obvious/shared/logging"
	svcmetrics "github.com/davejduke/obvious/shared/metrics"
	"github.com/davejduke/obvious/services/evidence/internal/repository"
)

func main() {
	logger := svclogging.New("evidence")

	// Start Prometheus metrics server on :9090.
	metricsSrv := svcmetrics.StartServer(os.Getenv("METRICS_PORT"))
	defer svcmetrics.StopServer(metricsSrv)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}

	// Wire up in-memory repository (swap for postgres repository in production)
	repo := repository.NewMemory()
	h := handler.New(repo)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler: svclogging.RequestIDMiddleware(svclogging.TraceContextMiddleware(svcmetrics.Middleware("evidence")(h.Router()))),
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
