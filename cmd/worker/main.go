package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ETAnderson/conductor/internal/config"
	"github.com/ETAnderson/conductor/internal/execute"
	"github.com/ETAnderson/conductor/internal/logging"
	"github.com/ETAnderson/conductor/internal/state"
	"github.com/ETAnderson/conductor/internal/worker"
)

func main() {
	cfg := config.Load()
	logger := logging.NewStdLogger("worker-service ")

	logger.Printf("ENV=%q STATE_BACKEND=%q DB_DSN_set=%v",
		cfg.Env, cfg.StateBackend, cfg.MySQLDSN != "")

	if cfg.StateBackend == "" {
		cfg.StateBackend = "memory"
	}

	factoryRes, err := state.NewStore(context.Background(), state.FactoryConfig{
		Backend:  cfg.StateBackend,
		MySQLDSN: cfg.MySQLDSN,
	})
	if err != nil {
		logger.Printf("state store init failed: %v", err)
		os.Exit(1)
	}

	store := factoryRes.Store

	exec := execute.Executor{
		Store: store,
		// OnExecute is intentionally nil for now (no external pushes yet).
		// The executor still validates run ownership and loads enqueued products.
	}

	r := worker.Runner{
		Store:       store,
		Executor:    exec,
		PollEvery:   1 * time.Second,
		MaxPerClaim: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Printf("starting (env=%s)", cfg.Env)

		err := r.Run(ctx)
		if err != nil && err != context.Canceled {
			logger.Printf("worker stopped: %v", err)
			os.Exit(1)
		}
	}()

	waitForShutdown(logger, cancel)
}

func waitForShutdown(logger interface{ Printf(string, ...any) }, cancel func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	<-sigCh
	logger.Printf("shutdown signal received")
	cancel()
	logger.Printf("shutdown complete")
}
