// Package main is the entry point for the AIAUDITOR search service.
//
// On startup it:
//  1. Connects to PostgreSQL and OpenSearch
//  2. Ensures index mappings exist
//  3. Runs a bulk reindex to catch up on existing data
//  4. Starts the CDC LISTEN/NOTIFY pipeline
//  5. Serves the search HTTP API on PORT (default 8089)
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/davejduke/obvious/services/search/internal/cdc"
	"github.com/davejduke/obvious/services/search/internal/config"
	"github.com/davejduke/obvious/services/search/internal/handler"
	"github.com/davejduke/obvious/services/search/internal/indexer"
	"github.com/davejduke/obvious/services/search/internal/opensearch"
	_ "github.com/lib/pq"
)

func main() {
	cfg := config.Load()

	log.Printf("[search] starting (port=%s opensearch=%s)", cfg.Port, cfg.OpenSearchURL)

	// Open PostgreSQL connection pool
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[search] open db: %v", err)
	}
	defer db.Close()
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Wait for DB
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		log.Printf("[search] waiting for postgres... (%d/30)", i+1)
		time.Sleep(2 * time.Second)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[search] postgres unavailable: %v", err)
	}
	log.Println("[search] postgres connected")

	// Build OpenSearch client
	osClient := opensearch.New(cfg.OpenSearchURL)

	// Build indexer
	ix := indexer.New(db, osClient)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Ensure indices exist (waits for OpenSearch to be ready)
	if err := ix.Startup(ctx); err != nil {
		log.Fatalf("[search] index setup: %v", err)
	}
	log.Println("[search] indices ready")

	// Initial bulk reindex
	if err := ix.BulkReindex(ctx); err != nil {
		log.Printf("[search] bulk reindex warning: %v", err)
	}

	// Start CDC pipeline in background
	cdcListener := cdc.New(cfg.DatabaseURL, ix.HandleEvent)
	go func() {
		for {
			if err := cdcListener.Start(ctx); err != nil {
				if ctx.Err() != nil {
					return // context cancelled, clean shutdown
				}
				log.Printf("[search] CDC pipeline error: %v; restarting in 10s", err)
				time.Sleep(10 * time.Second)
			}
		}
	}()

	// HTTP server
	h := handler.New(osClient)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      h.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[search] listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[search] server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[search] shutting down...")
	cancel()
	cdcListener.Close()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[search] forced shutdown: %v", err)
	}
	log.Println("[search] server exited")
}

