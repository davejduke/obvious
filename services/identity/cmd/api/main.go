// Package main is the entry point for the identity service.
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
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	svclogging "github.com/davejduke/obvious/shared/logging"
	svcmetrics "github.com/davejduke/obvious/shared/metrics"
	"github.com/davejduke/obvious/services/identity/internal/config"
	"github.com/davejduke/obvious/services/identity/internal/handlers"
	idmw "github.com/davejduke/obvious/services/identity/internal/middleware"
	"github.com/davejduke/obvious/services/identity/internal/repository"
	"github.com/davejduke/obvious/services/identity/internal/service"
	jwtpkg "github.com/davejduke/obvious/services/identity/pkg/jwt"
	"github.com/davejduke/obvious/services/identity/pkg/rbac"
)

func main() {
	cfg := config.Load()

	logger := svclogging.New("identity")

	// Start Prometheus metrics server on :9090.
	metricsSrv := svcmetrics.StartServer(os.Getenv("METRICS_PORT"))
	defer svcmetrics.StopServer(metricsSrv)

	// ── PostgreSQL ──────────────────────────────────────────────────────────────
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Critical(context.Background(), "db.connect_failed", map[string]any{"error": err.Error()})
os.Exit(1)
	}
	defer dbPool.Close()
	if err := dbPool.Ping(ctx); err != nil {
		logger.Critical(context.Background(), "db.ping_failed", map[string]any{"error": err.Error()})
os.Exit(1)
	}
	logger.Info(context.Background(), "db.connected", map[string]any{"db": "postgres"})

	// ── Redis ───────────────────────────────────────────────────────────────────
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Critical(context.Background(), "redis.parse_url_failed", map[string]any{"error": err.Error()})
os.Exit(1)
	}
	redisClient := redis.NewClient(redisOpts)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Critical(context.Background(), "redis.ping_failed", map[string]any{"error": err.Error()})
os.Exit(1)
	}
	logger.Info(context.Background(), "db.connected", map[string]any{"db": "redis"})
	defer redisClient.Close()

	// ── JWT Manager ─────────────────────────────────────────────────────────────
	accessTTL := time.Duration(cfg.AccessTokenTTL) * time.Minute
	refreshTTL := time.Duration(cfg.RefreshTokenTTL) * 24 * time.Hour

	jwtManager, err := jwtpkg.NewManager(
		cfg.JWTPrivKeyPath, cfg.JWTPubKeyPath,
		"identity.aiauditor", accessTTL, refreshTTL,
	)
	if err != nil {
		logger.Critical(context.Background(), "jwt.init_failed", map[string]any{"error": err.Error()})
os.Exit(1)
	}

	// ── Dependencies ────────────────────────────────────────────────────────────
	repo := repository.New(dbPool)
	sessions := service.NewSessionStore(redisClient, refreshTTL)
	rbacChecker := rbac.NewChecker()

	authHandler := handlers.NewAuthHandler(repo, jwtManager, sessions, refreshTTL)
	userHandler := handlers.NewUserHandler(repo)
	orgHandler := handlers.NewOrgHandler(repo)
	roleHandler := handlers.NewRoleHandler(repo)

	authenticator := idmw.NewAuthenticator(jwtManager)

	// ── Router ──────────────────────────────────────────────────────────────────
	r := chi.NewRouter()
	r.Use(svclogging.RequestIDMiddleware)
	r.Use(svclogging.TraceContextMiddleware)
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(corsMiddleware)

	// Public health endpoints
	r.Get("/health", healthHandler)
	r.Get("/ready", readyHandler(dbPool, redisClient))

	// Auth endpoints (public)
	r.Post("/auth/register", authHandler.Register)
	r.Post("/auth/login", authHandler.Login)
	r.Post("/auth/refresh", authHandler.Refresh)

	// Auth-required routes
	r.Group(func(r chi.Router) {
		r.Use(authenticator.Authenticate)

		r.Post("/auth/logout", authHandler.Logout)

		// Users
		r.Get("/users", idmw.RequirePermission(rbacChecker, rbac.ResourceUser, rbac.ActionRead)(
			http.HandlerFunc(userHandler.List),
		).ServeHTTP)
		r.Get("/users/{id}", idmw.RequirePermission(rbacChecker, rbac.ResourceUser, rbac.ActionRead)(
			http.HandlerFunc(userHandler.Get),
		).ServeHTTP)
		r.Put("/users/{id}", idmw.RequirePermission(rbacChecker, rbac.ResourceUser, rbac.ActionWrite)(
			http.HandlerFunc(userHandler.Update),
		).ServeHTTP)
		r.Delete("/users/{id}", idmw.RequirePermission(rbacChecker, rbac.ResourceUser, rbac.ActionManage)(
			http.HandlerFunc(userHandler.Delete),
		).ServeHTTP)
		r.Post("/users/{id}/roles", idmw.RequirePermission(rbacChecker, rbac.ResourceUser, rbac.ActionManage)(
			http.HandlerFunc(roleHandler.AssignToUser),
		).ServeHTTP)

		// Organizations
		r.Post("/organizations", orgHandler.Create)
		r.Get("/organizations", idmw.RequirePermission(rbacChecker, rbac.ResourceOrg, rbac.ActionRead)(
			http.HandlerFunc(orgHandler.List),
		).ServeHTTP)
		r.Get("/organizations/{id}", idmw.RequirePermission(rbacChecker, rbac.ResourceOrg, rbac.ActionRead)(
			http.HandlerFunc(orgHandler.Get),
		).ServeHTTP)
		r.Put("/organizations/{id}", idmw.RequirePermission(rbacChecker, rbac.ResourceOrg, rbac.ActionWrite)(
			http.HandlerFunc(orgHandler.Update),
		).ServeHTTP)
		r.Post("/organizations/{id}/members", idmw.RequirePermission(rbacChecker, rbac.ResourceUser, rbac.ActionManage)(
			http.HandlerFunc(orgHandler.AddMember),
		).ServeHTTP)

		// Roles
		r.Get("/roles", roleHandler.List)
	})

	// ── Server ──────────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Info(context.Background(), "server.start", map[string]any{"port": cfg.Port})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(context.Background(), "server.error", map[string]any{"error": err.Error()})
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		logger.Error(context.Background(), "server.shutdown_error", map[string]any{"error": err.Error()})
	}
	logger.Info(context.Background(), "server.shutdown", nil)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"service":"identity","status":"healthy","version":"0.1.0"}`)
}

func readyHandler(db *pgxpool.Pool, rc *redis.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		dbOK := db.Ping(ctx) == nil
		redisOK := rc.Ping(ctx).Err() == nil
		ready := dbOK && redisOK
		status := http.StatusOK
		if !ready {
			status = http.StatusServiceUnavailable
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		fmt.Fprintf(w, `{"ready":%v,"db":%v,"redis":%v}`, ready, dbOK, redisOK)
	}
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type,X-Request-ID")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
