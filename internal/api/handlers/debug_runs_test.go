package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

func TestDebugRuns_ListAndDetail(t *testing.T) {
	st := state.NewMemoryStore()
	tenantID := uint64(1)

	// Seed a run
	const runID = "run_test_1"
	ctx := context.Background()

	_ = st.InsertRun(ctx, state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        string(domain.RunStatusHasChanges),
		PushTriggered: true,
		Received:      1,
		Valid:         1,
		Rejected:      0,
		Unchanged:     0,
		Enqueued:      1,
		Warnings:      ingest.UnknownKeyWarning{UnknownKeys: []string{"x"}},
		CreatedAt:     time.Now().UTC(),
	})

	_ = st.InsertRunProducts(ctx, runID, []ingest.ProductProcessResult{
		{
			ProductKey:  "sku1",
			Disposition: domain.ProductDispositionEnqueued,
			Reason:      "hash_changed",
			Hash:        "abc",
		},
	})

	// List endpoint
	listH := DebugRunsHandler{Store: st}
	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs?limit=10", nil)
	req = req.WithContext(tenantctx.WithTenantID(req.Context(), tenantID))
	rec := httptest.NewRecorder()
	listH.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var listResp struct {
		Items []state.RunRecord `json:"items"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(listResp.Items) != 1 || listResp.Items[0].RunID != runID {
		t.Fatalf("unexpected list response: %#v", listResp.Items)
	}

	// Detail endpoint
	detailH := DebugRunDetailHandler{Store: st}
	req2 := httptest.NewRequest(http.MethodGet, "/v1/debug/runs/"+runID, nil)
	req2 = req2.WithContext(tenantctx.WithTenantID(req2.Context(), tenantID))
	rec2 := httptest.NewRecorder()
	detailH.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var detailResp struct {
		Run      state.RunRecord               `json:"run"`
		Products []ingest.ProductProcessResult `json:"products"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &detailResp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if detailResp.Run.RunID != runID {
		t.Fatalf("expected run_id %s, got %s", runID, detailResp.Run.RunID)
	}
	if len(detailResp.Products) != 1 || detailResp.Products[0].ProductKey != "sku1" {
		t.Fatalf("unexpected products: %#v", detailResp.Products)
	}
}
