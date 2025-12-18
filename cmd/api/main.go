package main

import (
	"context"
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
	"github.com/ETAnderson/conductor/internal/state"
)

func main() {
	cfg := config.Load()
	logger := logging.NewStdLogger("api-service ")

	tenantID := uint64(1)

	store := state.NewMemoryStore()
	proc := ingest.NewProcessor()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
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
