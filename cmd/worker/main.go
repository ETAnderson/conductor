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

	"github.com/ETAnderson/conductor/internal/channels"
	"github.com/ETAnderson/conductor/internal/channels/google"
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

	reg := channels.NewRegistry(
		google.Channel{Store: store},
	)

	exec := execute.Executor{
		Store:           store,
		Registry:        reg,
		EnabledChannels: []string{"google"},
		OnChannelResult: func(ctx context.Context, run state.RunRecord, res channels.BuildResult) error {
			rec := state.RunChannelResultRecord{
				RunID:     run.RunID,
				TenantID:  run.TenantID,
				Channel:   res.Channel,
				Attempt:   res.Attempt,
				OkCount:   res.OkCount,
				ErrCount:  res.ErrCount,
				CreatedAt: time.Now().UTC(),
			}

			if err := store.InsertRunChannelResult(ctx, rec); err != nil {
				return err
			}

			items := make([]state.RunChannelItemRecord, 0, len(res.Items))
			for _, it := range res.Items {
				items = append(items, state.RunChannelItemRecord{
					RunID:      run.RunID,
					Channel:    res.Channel,
					ProductKey: it.ProductKey,
					Status:     it.Status,
					Message:    it.Message,
				})
			}

			return store.InsertRunChannelItems(ctx, run.RunID, res.Channel, items)
		},
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
