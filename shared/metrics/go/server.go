package metrics

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const defaultPort = "9090"

// StartServer starts a Prometheus metrics HTTP server on :port/metrics.
// It is non-blocking; the server runs in a goroutine.
// Returns the *http.Server so the caller can shut it down gracefully.
func StartServer(port string) *http.Server {
	if port == "" {
		port = defaultPort
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  30 * time.Second,
	}

	go func() {
		log.Printf("[metrics] Prometheus server listening on :%s/metrics", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[metrics] server error: %v", err)
		}
	}()

	return srv
}

// StopServer gracefully shuts down a metrics server started by StartServer.
func StopServer(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[metrics] shutdown error: %v", err)
	}
}

