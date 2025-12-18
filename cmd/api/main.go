package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ETAnderson/conductor/internal/config"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.NewStdLogger("api-service ")

	store := ingest.NewMemoryHashStore()
	proc := ingest.NewProcessor()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	idemStore := ingest.NewMemoryIdempotencyStore(24 * time.Hour)

	debugHandler := ingest.DebugUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	mux.Handle("/v1/debug/products:upsert", ingest.IdempotencyMiddleware{
		Store: idemStore,
		Next:  debugHandler,
	})

	mux.Handle("/v1/debug/products:upsert-bulk", ingest.IdempotencyMiddleware{
		Store: idemStore,
		Next: ingest.DebugBulkUpsertHandler{
			Processor:       proc,
			Store:           store,
			EnabledChannels: []string{"google"},
		},
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
