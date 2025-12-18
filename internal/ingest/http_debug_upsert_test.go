package ingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ETAnderson/conductor/internal/domain"
)

func TestDebugUpsertHandler_RunSemantics_FirstEnqueuedSecondUnchanged(t *testing.T) {
	store := NewMemoryHashStore()
	proc := NewProcessor()

	h := DebugUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	body := `[
  {
    "product_key": "sku1",
    "title": "Test",
    "description": "Desc",
    "link": "https://example.com/p/sku1",
    "image_link": "https://example.com/p/sku1.jpg",
    "condition": "new",
    "availability": "in_stock",
    "price": { "amount_decimal": "19.99", "currency": "USD" },
    "channel": { "google": { "control": { "state": "active" } } }
  }
]`

	// First call -> enqueued, push_triggered=true, status=has_changes
	req1 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(body))
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}

	var resp1 RunResponse
	if err := json.NewDecoder(bytes.NewReader(rec1.Body.Bytes())).Decode(&resp1); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp1.RunID == "" {
		t.Fatalf("expected run_id")
	}
	if !resp1.PushTriggered {
		t.Fatalf("expected push_triggered=true")
	}
	if resp1.Status != domain.RunStatusHasChanges {
		t.Fatalf("expected status=has_changes, got %s", resp1.Status)
	}
	if resp1.Result.Summary.Enqueued != 1 {
		t.Fatalf("expected enqueued=1, got %#v", resp1.Result.Summary)
	}

	// Second call -> unchanged, push_triggered=false, status=no_change_detected
	req2 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(body))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var resp2 RunResponse
	if err := json.NewDecoder(bytes.NewReader(rec2.Body.Bytes())).Decode(&resp2); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp2.RunID == "" {
		t.Fatalf("expected run_id")
	}
	if resp2.PushTriggered {
		t.Fatalf("expected push_triggered=false")
	}
	if resp2.Status != domain.RunStatusNoChangeDetected {
		t.Fatalf("expected status=no_change_detected, got %s", resp2.Status)
	}
	if resp2.Result.Summary.Unchanged != 1 {
		t.Fatalf("expected unchanged=1, got %#v", resp2.Result.Summary)
	}
	if resp2.Result.Products[0].Disposition != domain.ProductDispositionUnchanged {
		t.Fatalf("expected unchanged disposition, got %s", resp2.Result.Products[0].Disposition)
	}
}

func TestDebugUpsertHandler_InvalidJson(t *testing.T) {
	store := NewMemoryHashStore()
	proc := NewProcessor()

	h := DebugUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(`{`))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDebugUpsertHandler_MethodNotAllowed(t *testing.T) {
	store := NewMemoryHashStore()
	proc := NewProcessor()

	h := DebugUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/products:upsert", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
}
