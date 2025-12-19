package worker

import (
	"context"
	"errors"
	"time"

	"github.com/ETAnderson/conductor/internal/state"
)

type Runner struct {
	Store       state.Store
	PollEvery   time.Duration
	ClaimTTL    time.Duration
	MaxPerClaim int
	ProcessFn   func(ctx context.Context, job Job) error
}

type Job struct {
	RunID    string
	TenantID uint64
	// For v1 we treat a "job" as a run-level unit. Later we can claim per-product.
}

func (r Runner) Run(ctx context.Context) error {
	if r.Store == nil {
		return errors.New("store is nil")
	}
	if r.PollEvery <= 0 {
		r.PollEvery = 500 * time.Millisecond
	}
	if r.ClaimTTL <= 0 {
		r.ClaimTTL = 30 * time.Second
	}
	if r.MaxPerClaim <= 0 {
		r.MaxPerClaim = 10
	}
	if r.ProcessFn == nil {
		r.ProcessFn = func(context.Context, Job) error { return nil }
	}

	ticker := time.NewTicker(r.PollEvery)
	defer ticker.Stop()

	// one immediate pass
	if err := r.tick(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := r.tick(ctx); err != nil {
				return err
			}
		}
	}
}

func (r Runner) tick(ctx context.Context) error {
	claims, err := r.Store.ClaimRuns(ctx, r.MaxPerClaim)
	if err != nil {
		return err
	}

	for _, c := range claims {
		job := Job{
			RunID:    c.RunID,
			TenantID: c.TenantID,
		}

		if err := r.ProcessFn(ctx, job); err != nil {
			_ = r.Store.FailRun(ctx, c.TenantID, c.RunID, err.Error())
			continue
		}

		_ = r.Store.CompleteRun(ctx, c.TenantID, c.RunID)
	}

	return nil
}
