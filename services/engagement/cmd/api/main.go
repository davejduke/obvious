// Package main is the entry point for the engagement service.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	svclogging "github.com/davejduke/obvious/shared/logging"
	svcmetrics "github.com/davejduke/obvious/shared/metrics"

	"github.com/davejduke/obvious/services/engagement/internal/handler"
	"github.com/davejduke/obvious/services/engagement/internal/store"
)

func main() {
	logger := svclogging.New("engagement")

	// Start Prometheus metrics server on :9090.
	metricsSrv := svcmetrics.StartServer(os.Getenv("METRICS_PORT"))
	defer svcmetrics.StopServer(metricsSrv)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	env := os.Getenv("ENV")
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "engagement",
			"status":  "healthy",
			"version": "0.1.0",
		})
	})
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	engagementStore := store.NewEngagementStore()
	engagementHandler := handler.NewEngagementHandler(engagementStore)
	engagementHandler.RegisterRoutes(r)

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: svclogging.RequestIDMiddleware(svclogging.TraceContextMiddleware(svcmetrics.Middleware("engagement")(r))),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	ctx := context.Background()
	go func() {
		logger.Info(ctx, "server.start", map[string]any{"port": port})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "server.error", map[string]any{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error(ctx, "server.shutdown_error", map[string]any{"error": err.Error()})
	}
	logger.Info(ctx, "server.shutdown", nil)
}
