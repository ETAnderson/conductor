package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ETAnderson/conductor/internal/api/auth"
	"github.com/ETAnderson/conductor/internal/api/handlers"
	"github.com/ETAnderson/conductor/internal/api/middleware"
	"github.com/ETAnderson/conductor/internal/config"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/logging"
	"github.com/ETAnderson/conductor/internal/migrate"
	"github.com/ETAnderson/conductor/internal/state"
)

func main() {
	config.LoadDotEnv()
	cfg := config.Load()

	logger := logging.NewStdLogger("api-service ")

	logger.Printf("ENV=%q PORT=%q STATE_BACKEND=%q RUN_MIGRATIONS=%v DB_DSN_set=%v",
		cfg.Env, cfg.Port, cfg.StateBackend, cfg.RunMigrations, cfg.MySQLDSN != "")

	if cfg.StateBackend == "" {
		cfg.StateBackend = "memory"
	}

	proc := ingest.NewProcessor()

	factoryRes, err := state.NewStore(context.Background(), state.FactoryConfig{
		Backend:  cfg.StateBackend,
		MySQLDSN: cfg.MySQLDSN,
	})
	if err != nil {
		logger.Printf("state store init failed: %v", err)
		os.Exit(1)
	}

	// Load RS256 public key for JWT verification.
	// In dev, allow missing key so you can keep using X-Tenant-ID + debug flows.
	pub, err := auth.LoadRSAPublicKeyFromPathEnv("JWT_PUBLIC_KEY_PATH")
	if err != nil {
		if cfg.Env != "dev" {
			logger.Printf("JWT public key load failed: %v", err)
			os.Exit(1)
		}
		pub = nil
	}

	store := factoryRes.Store
	logger.Printf("store_impl=%T", store)

	if cfg.RunMigrations && factoryRes.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := migrate.ApplyDir(ctx, factoryRes.DB, "migrations"); err != nil {
			logger.Printf("migrations failed: %v", err)
			os.Exit(1)
		}
	}

	// Dev bootstrap: ensure debug tenant exists so FK inserts succeed for runs/idempotency.
	if factoryRes.DB != nil && cfg.Env == "dev" {
		_, err := factoryRes.DB.Exec(`INSERT INTO tenants (tenant_id, name)
		VALUES (1, 'debug')
		ON DUPLICATE KEY UPDATE name=VALUES(name)`)
		if err != nil {
			logger.Printf("bootstrap tenant failed: %v", err)
			os.Exit(1)
		}
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		resp := map[string]any{"ok": true, "backend": cfg.StateBackend}

		if factoryRes.DB != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
			defer cancel()
			if err := factoryRes.DB.PingContext(ctx); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"ok":      false,
					"backend": cfg.StateBackend,
					"db_ok":   false,
					"error":   err.Error(),
				})
				return
			}
			resp["db_ok"] = true
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Handlers (tenant is resolved from request context now; no TenantID fields here)
	debugUpsert := handlers.DebugUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	debugBulk := handlers.DebugBulkUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	mux.Handle("/v1/debug/products:upsert", middleware.IdempotencyMiddleware{
		Store: store,
		Next:  debugUpsert,
	})

	mux.Handle("/v1/debug/products:upsert-bulk", middleware.IdempotencyMiddleware{
		Store: store,
		Next:  debugBulk,
	})

	mux.Handle("/v1/debug/runs", handlers.DebugRunsHandler{
		Store: store,
	})
	mux.Handle("/v1/debug/runs/", handlers.DebugRunSubroutesHandler{
		Store: store,
	})

	// Wrap handler chain (order matters!)
	var root http.Handler = mux

	// Tenant header (dev override / default tenant)
	root = middleware.TenantMiddleware{
		Env:  cfg.Env,
		Next: root,
	}

	// Auth (RS256 JWT)
	root = middleware.AuthMiddleware{
		Env:       cfg.Env,
		PublicKey: pub,
		Next:      root,
	}

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Printf("starting (env=%s) on %s", cfg.Env, server.Addr)

		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Printf("server error: %v", err)
			os.Exit(1)
		}
	}()

	waitForShutdown(logger, server)
}

func waitForShutdown(logger interface{ Printf(string, ...any) }, server *http.Server) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	logger.Printf("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)
	logger.Printf("shutdown complete")
}
