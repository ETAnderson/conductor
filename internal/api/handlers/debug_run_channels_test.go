package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/state"
)

func TestDebugRunChannels_ListResultsAndItems(t *testing.T) {
	st := state.NewMemoryStore()
	tenantID := uint64(1)
	runID := "run_test_channels_1"

	// Seed run (required for tenant ownership check)
	_ = st.InsertRun(context.Background(), state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		Status:        "has_changes",
		PushTriggered: true,
		Received:      1,
		Valid:         1,
		Rejected:      0,
		Unchanged:     0,
		Enqueued:      1,
		CreatedAt:     time.Now().UTC(),
	})

	// Seed channel results + items
	_ = st.InsertRunChannelResult(context.Background(), state.RunChannelResultRecord{
		RunID:     runID,
		TenantID:  tenantID,
		Channel:   "google",
		Attempt:   1,
		OkCount:   1,
		ErrCount:  0,
		CreatedAt: time.Now().UTC(),
	})

	_ = st.InsertRunChannelItems(context.Background(), runID, "google", []state.RunChannelItemRecord{
		{RunID: runID, Channel: "google", ProductKey: "sku1", Status: "ok", Message: ""},
	})

	h := DebugRunChannelsHandler{Store: st}

	// GET /v1/debug/runs/{run}/channels
	req1 := httptest.NewRequest(http.MethodGet, "/v1/debug/runs/"+runID+"/channels", nil)
	req1 = req1.WithContext(tenantctx.WithTenantID(req1.Context(), tenantID))
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec1.Code, rec1.Body.String())
	}

	// GET /v1/debug/runs/{run}/channels/google?limit=1000
	req2 := httptest.NewRequest(http.MethodGet, "/v1/debug/runs/"+runID+"/channels/google?limit=1000", nil)
	req2 = req2.WithContext(tenantctx.WithTenantID(req2.Context(), tenantID))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestDebugRunChannels_RunNotFound(t *testing.T) {
	st := state.NewMemoryStore()
	tenantID := uint64(1)

	h := DebugRunChannelsHandler{Store: st}

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs/run_missing/channels", nil)
	req = req.WithContext(tenantctx.WithTenantID(req.Context(), tenantID))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d body=%s", rec.Code, rec.Body.String())
	}
}
