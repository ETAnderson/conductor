package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

func TestDebugUpsert_StateWired_FirstEnqueuedSecondUnchanged(t *testing.T) {
	store := state.NewMemoryStore()
	proc := ingest.NewProcessor()

	h := DebugUpsertHandler{
		Processor:       proc,
		Store:           store,
		TenantID:        1,
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
    "channel": { "google": { "control": { "state": "active" } } },
    "unknown_field": 1
  }
]`

	// First call -> enqueued
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
	if !resp1.PushTriggered || resp1.Status != domain.RunStatusHasChanges {
		t.Fatalf("unexpected run: push=%v status=%s", resp1.PushTriggered, resp1.Status)
	}
	if len(resp1.Warnings.UnknownKeys) == 0 {
		t.Fatalf("expected unknown key warnings")
	}

	// Second call -> unchanged (because canonical hash persisted in state)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(body))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	var resp2 RunResponse
	_ = json.NewDecoder(bytes.NewReader(rec2.Body.Bytes())).Decode(&resp2)

	if resp2.PushTriggered || resp2.Status != domain.RunStatusNoChangeDetected {
		t.Fatalf("expected no_change_detected, got push=%v status=%s", resp2.PushTriggered, resp2.Status)
	}
	if resp2.Result.Summary.Unchanged != 1 {
		t.Fatalf("expected unchanged=1, got %#v", resp2.Result.Summary)
	}
}
