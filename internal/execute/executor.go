package execute

import (
	"context"
	"errors"
	"fmt"

	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

type Executor struct {
	Store state.Store

	// ProductLimit controls how many run products we load for execution.
	// If <= 0, defaults to 100000 (safe for now; optimize later).
	ProductLimit int

	// OnExecute is a test seam + future channel orchestration hook.
	// It receives the run record plus ONLY the enqueued products for this run.
	// If nil, execution is a no-op (but still validates tenant/run ownership).
	OnExecute func(ctx context.Context, run state.RunRecord, enqueued []ingest.ProductProcessResult) error
}

var ErrRunNotFound = errors.New("run not found")

// Execute implements worker.RunExecutor.
// It validates run ownership, loads run products, filters to enqueued items,
// and invokes the orchestration hook (OnExecute).
func (e Executor) Execute(ctx context.Context, runID string, tenantID uint64) error {
	if e.Store == nil {
		return errors.New("store is nil")
	}
	if runID == "" {
		return errors.New("runID is required")
	}
	if tenantID == 0 {
		return errors.New("tenantID is required")
	}

	limit := e.ProductLimit
	if limit <= 0 {
		limit = 100000
	}

	run, ok, err := e.Store.GetRun(ctx, tenantID, runID)
	if err != nil {
		return fmt.Errorf("get run failed: %w", err)
	}
	if !ok {
		return ErrRunNotFound
	}

	products, err := e.Store.ListRunProducts(ctx, runID, limit)
	if err != nil {
		return fmt.Errorf("list run products failed: %w", err)
	}

	enqueued := make([]ingest.ProductProcessResult, 0, len(products))
	for _, p := range products {
		if p.Disposition == domain.ProductDispositionEnqueued {
			enqueued = append(enqueued, p)
		}
	}

	if e.OnExecute != nil {
		if err := e.OnExecute(ctx, run, enqueued); err != nil {
			return err
		}
	}

	return nil
}
