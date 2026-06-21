// Package main is the entry point for the integration gateway service.
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

	"github.com/davejduke/obvious/services/integration/internal/adapters"
	"github.com/davejduke/obvious/services/integration/internal/connector"
	"github.com/davejduke/obvious/services/integration/internal/grc"
	"github.com/davejduke/obvious/services/integration/internal/handler"
	"github.com/davejduke/obvious/services/integration/internal/vulnconnector"
)

func main() {
	logger := svclogging.New("integration")

	// Start Prometheus metrics server on :9090.
	metricsSrv := svcmetrics.StartServer(os.Getenv("METRICS_PORT"))
	defer svcmetrics.StopServer(metricsSrv)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	env := os.Getenv("ENV")
	if env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "integration",
			"status":  "healthy",
			"version": "0.1.0",
		})
	})
	r.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ready": true})
	})

	// ── SIEM connector registry ────────────────────────────────────────────
	reg := connector.NewRegistry()

	// Sentinel adapter with circuit breaker
	sentinelAdapter := adapters.NewSentinelAdapter(adapters.SentinelConfig{
		WorkspaceID:  os.Getenv("SENTINEL_WORKSPACE_ID"),
		TenantID:     os.Getenv("AZURE_TENANT_ID"),
		ClientID:     os.Getenv("AZURE_CLIENT_ID"),
		ClientSecret: os.Getenv("AZURE_CLIENT_SECRET"),
		MockMode:     os.Getenv("MOCK_MODE") != "false",
	})
	reg.Register(connector.NewCircuitBreaker(sentinelAdapter, connector.DefaultConfig()))

	// Splunk adapter with circuit breaker
	splunkAdapter := adapters.NewSplunkAdapter(adapters.SplunkConfig{
		BaseURL:     os.Getenv("SPLUNK_BASE_URL"),
		Token:       os.Getenv("SPLUNK_TOKEN"),
		SavedSearch: os.Getenv("SPLUNK_SAVED_SEARCH"),
		MockMode:    os.Getenv("MOCK_MODE") != "false",
	})
	reg.Register(connector.NewCircuitBreaker(splunkAdapter, connector.DefaultConfig()))

	// ── GRC outbound connector registry ───────────────────────────────────
	grcReg := grc.NewRegistry()

	// ServiceNow GRC adapter (all connectors use mock/stub — no live credentials)
	serviceNowAdapter := adapters.NewServiceNowAdapter(adapters.ServiceNowConfig{
		BaseURL:  os.Getenv("SERVICENOW_BASE_URL"),
		Username: os.Getenv("SERVICENOW_USERNAME"),
		Password: os.Getenv("SERVICENOW_PASSWORD"),
		MockMode: os.Getenv("MOCK_MODE") != "false",
	})
	grcReg.Register(serviceNowAdapter)

	// ── Vulnerability/endpoint connector registry ──────────────────────────
	vulnReg := vulnconnector.NewVulnRegistry()

	qualysAdapter := adapters.NewQualysAdapter(adapters.QualysConfig{
		BaseURL:  os.Getenv("QUALYS_BASE_URL"),
		Username: os.Getenv("QUALYS_USERNAME"),
		Password: os.Getenv("QUALYS_PASSWORD"),
		MockMode: os.Getenv("MOCK_MODE") != "false",
	})
	vulnReg.Register(qualysAdapter)

	tenableAdapter := adapters.NewTenableAdapter(adapters.TenableConfig{
		BaseURL:   os.Getenv("TENABLE_BASE_URL"),
		AccessKey: os.Getenv("TENABLE_ACCESS_KEY"),
		SecretKey: os.Getenv("TENABLE_SECRET_KEY"),
		MockMode:  os.Getenv("MOCK_MODE") != "false",
	})
	vulnReg.Register(tenableAdapter)

	crowdStrikeAdapter := adapters.NewCrowdStrikeAdapter(adapters.CrowdStrikeConfig{
		BaseURL:      os.Getenv("CROWDSTRIKE_BASE_URL"),
		ClientID:     os.Getenv("CROWDSTRIKE_CLIENT_ID"),
		ClientSecret: os.Getenv("CROWDSTRIKE_CLIENT_SECRET"),
		MockMode:     os.Getenv("MOCK_MODE") != "false",
	})
	vulnReg.Register(crowdStrikeAdapter)

	// ── Routes ────────────────────────────────────────────────────────────
	integrationHandler := handler.NewIntegrationHandlerFull(reg, grcReg, vulnReg)
	integrationHandler.RegisterRoutes(r)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      svclogging.RequestIDMiddleware(svclogging.TraceContextMiddleware(svcmetrics.Middleware("integration")(r))),
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
