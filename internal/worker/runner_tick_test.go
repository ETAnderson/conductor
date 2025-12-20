package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/state"
)

func TestRunner_Tick_ClaimsAndCompletes_RunAndTenantInContext(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_test_complete_1"
	tenantID := uint64(1)

	err := st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "has_changes",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC().Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("InsertRun: %v", err)
	}

	calls := 0
	r := Runner{
		Store:       st,
		MaxPerClaim: 10,
		ProcessFn: func(ctx context.Context, job Job) error {
			calls++

			// Ensure job is correct
			if job.RunID != runID {
				t.Fatalf("expected job.run_id=%q got %q", runID, job.RunID)
			}
			if job.TenantID != tenantID {
				t.Fatalf("expected job.tenant_id=%d got %d", tenantID, job.TenantID)
			}

			// Ensure context was enriched (5.1)
			if got := tenantctx.TenantID(ctx); got != tenantID {
				t.Fatalf("expected ctx tenant=%d got %d", tenantID, got)
			}
			if got := RunID(ctx); got != runID {
				t.Fatalf("expected ctx run_id=%q got %q", runID, got)
			}

			return nil
		},
	}

	if err := r.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected ProcessFn called once, got %d", calls)
	}

	rec, ok, err := st.GetRun(context.Background(), tenantID, runID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if !ok {
		t.Fatalf("expected run to exist")
	}
	if rec.Status != "completed" {
		t.Fatalf("expected status=completed, got %q", rec.Status)
	}
}

func TestRunner_Tick_FailMarksFailed(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_test_fail_1"
	tenantID := uint64(1)

	err := st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "has_changes",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC().Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("InsertRun: %v", err)
	}

	r := Runner{
		Store:       st,
		MaxPerClaim: 10,
		ProcessFn: func(ctx context.Context, job Job) error {
			return errors.New("boom")
		},
	}

	if err := r.tick(context.Background()); err != nil {
		t.Fatalf("tick: %v", err)
	}

	rec, ok, err := st.GetRun(context.Background(), tenantID, runID)
	if err != nil {
		t.Fatalf("GetRun: %v", err)
	}
	if !ok {
		t.Fatalf("expected run to exist")
	}
	if rec.Status != "failed" {
		t.Fatalf("expected status=failed, got %q", rec.Status)
	}
}

func TestRunner_Tick_DoesNotReprocessAlreadyClaimedOrCompleted(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_test_no_reprocess_1"
	tenantID := uint64(1)

	err := st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "has_changes",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC().Add(-1 * time.Minute),
	})
	if err != nil {
		t.Fatalf("InsertRun: %v", err)
	}

	calls := 0
	r := Runner{
		Store:       st,
		MaxPerClaim: 10,
		ProcessFn: func(ctx context.Context, job Job) error {
			calls++
			return nil
		},
	}

	// First tick should process it
	if err := r.tick(context.Background()); err != nil {
		t.Fatalf("tick(1): %v", err)
	}

	// Second tick should find nothing claimable (it should be completed now)
	if err := r.tick(context.Background()); err != nil {
		t.Fatalf("tick(2): %v", err)
	}

	if calls != 1 {
		t.Fatalf("expected ProcessFn called once total, got %d", calls)
	}
}
