// Package main is the entry point for the identity service.
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
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

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

	// ── PostgreSQL ──────────────────────────────────────────────────────────────
	ctx := context.Background()
	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("[identity] connect postgres: %v", err)
	}
	defer dbPool.Close()
	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("[identity] ping postgres: %v", err)
	}
	log.Println("[identity] PostgreSQL connected")

	// ── Redis ───────────────────────────────────────────────────────────────────
	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("[identity] parse redis url: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("[identity] ping redis: %v", err)
	}
	log.Println("[identity] Redis connected")
	defer redisClient.Close()

	// ── JWT Manager ─────────────────────────────────────────────────────────────
	accessTTL := time.Duration(cfg.AccessTokenTTL) * time.Minute
	refreshTTL := time.Duration(cfg.RefreshTokenTTL) * 24 * time.Hour

	jwtManager, err := jwtpkg.NewManager(
		cfg.JWTPrivKeyPath, cfg.JWTPubKeyPath,
		"identity.aiauditor", accessTTL, refreshTTL,
	)
	if err != nil {
		log.Fatalf("[identity] init jwt manager: %v", err)
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
		log.Printf("[identity] listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[identity] server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Fatalf("[identity] forced shutdown: %v", err)
	}
	log.Println("[identity] server exited")
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
