package execute

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

func TestExecutor_Execute_LoadsRunAndFiltersEnqueued(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_exec_skel_1"
	tenantID := uint64(1)

	_ = st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "processing",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC(),
	})

	_ = st.InsertRunProducts(context.Background(), runID, []ingest.ProductProcessResult{
		{
			ProductKey:  "sku1",
			Disposition: domain.ProductDispositionEnqueued,
			Reason:      "hash_changed",
			Hash:        "aaa",
		},
		{
			ProductKey:  "sku2",
			Disposition: domain.ProductDispositionUnchanged,
			Reason:      "no_change_detected",
			Hash:        "bbb",
		},
		{
			ProductKey:  "sku3",
			Disposition: domain.ProductDispositionEnqueued,
			Reason:      "hash_changed",
			Hash:        "ccc",
		},
	})

	var gotRun state.RunRecord
	var gotEnqueued []ingest.ProductProcessResult

	ex := Executor{
		Store: st,
		OnExecute: func(ctx context.Context, run state.RunRecord, enq []ingest.ProductProcessResult) error {
			gotRun = run
			gotEnqueued = append([]ingest.ProductProcessResult(nil), enq...)
			return nil
		},
	}

	if err := ex.Execute(context.Background(), runID, tenantID); err != nil {
		t.Fatalf("Execute returned err: %v", err)
	}

	if gotRun.RunID != runID {
		t.Fatalf("expected run_id=%q got %q", runID, gotRun.RunID)
	}
	if gotRun.TenantID != tenantID {
		t.Fatalf("expected tenant_id=%d got %d", tenantID, gotRun.TenantID)
	}

	if len(gotEnqueued) != 2 {
		t.Fatalf("expected 2 enqueued products, got %d", len(gotEnqueued))
	}
	if gotEnqueued[0].ProductKey != "sku1" || gotEnqueued[1].ProductKey != "sku3" {
		t.Fatalf("unexpected enqueued order/values: %+v", gotEnqueued)
	}
}

func TestExecutor_Execute_RejectsWrongTenant(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_exec_skel_2"

	_ = st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      1,
		Status:        "processing",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC(),
	})

	ex := Executor{Store: st}

	err := ex.Execute(context.Background(), runID, 2)
	if !errors.Is(err, ErrRunNotFound) {
		t.Fatalf("expected ErrRunNotFound, got %v", err)
	}
}

func TestExecutor_Execute_PropagatesHookError(t *testing.T) {
	st := state.NewMemoryStore()

	runID := "run_exec_skel_3"
	tenantID := uint64(1)

	_ = st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "processing",
		PushTriggered: true,
		CreatedAt:     time.Now().UTC(),
	})

	want := errors.New("executor hook failed")

	ex := Executor{
		Store: st,
		OnExecute: func(ctx context.Context, run state.RunRecord, enq []ingest.ProductProcessResult) error {
			return want
		},
	}

	err := ex.Execute(context.Background(), runID, tenantID)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v got %v", want, err)
	}
}
