package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ETAnderson/conductor/internal/api/handlers"
	"github.com/ETAnderson/conductor/internal/api/middleware"
	"github.com/ETAnderson/conductor/internal/config"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/logging"
	"github.com/ETAnderson/conductor/internal/migrate"
	"github.com/ETAnderson/conductor/internal/state"
)

func main() {
	cfg := config.Load()

	logger := logging.NewStdLogger("api-service ")

	logger.Printf("ENV=%q PORT=%q STATE_BACKEND=%q RUN_MIGRATIONS=%v DB_DSN_set=%v",
		cfg.Env, cfg.Port, cfg.StateBackend, cfg.RunMigrations, cfg.MySQLDSN != "")

	if cfg.StateBackend == "" {
		cfg.StateBackend = "memory"
	}

	tenantID := uint64(1)

	proc := ingest.NewProcessor()

	factoryRes, err := state.NewStore(context.Background(), state.FactoryConfig{
		Backend:  cfg.StateBackend,
		MySQLDSN: cfg.MySQLDSN,
	})
	if err != nil {
		logger.Printf("state store init failed: %v", err)
		os.Exit(1)
	}

	store := factoryRes.Store

	if cfg.RunMigrations && factoryRes.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := migrate.ApplyDir(ctx, factoryRes.DB, "migrations"); err != nil {
			logger.Printf("migrations failed: %v", err)
			os.Exit(1)
		}
	}

	if factoryRes.DB != nil {
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
				_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "backend": cfg.StateBackend, "db_ok": false, "error": err.Error()})
				return
			}
			resp["db_ok"] = true
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	})

	debugUpsert := handlers.DebugUpsertHandler{
		Processor:       proc,
		Store:           store,
		TenantID:        tenantID,
		EnabledChannels: []string{"google"},
	}

	debugBulk := handlers.DebugBulkUpsertHandler{
		Processor:       proc,
		Store:           store,
		TenantID:        tenantID,
		EnabledChannels: []string{"google"},
	}

	mux.Handle("/v1/debug/products:upsert", middleware.IdempotencyMiddleware{
		Store:    store,
		TenantID: tenantID,
		Next:     debugUpsert,
	})

	mux.Handle("/v1/debug/products:upsert-bulk", middleware.IdempotencyMiddleware{
		Store:    store,
		TenantID: tenantID,
		Next:     debugBulk,
	})

	mux.Handle("/v1/debug/runs", handlers.DebugRunsHandler{
		Store:    store,
		TenantID: tenantID,
	})

	mux.Handle("/v1/debug/runs/", handlers.DebugRunDetailHandler{
		Store:    store,
		TenantID: tenantID,
	})

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           mux,
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
