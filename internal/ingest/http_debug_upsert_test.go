package ingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ETAnderson/conductor/internal/domain"
)

func TestDebugUpsertHandler_FirstCallEnqueuesSecondCallUnchanged(t *testing.T) {
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

	// First call -> enqueued
	req1 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(body))
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Second call -> unchanged (same content, store has previous hash)
	req2 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(body))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	// Light assertion by decoding result
	var out ProcessOutput
	if err := jsonDecode(rec2.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if out.Summary.Unchanged != 1 {
		t.Fatalf("expected unchanged=1, got %#v", out.Summary)
	}
	if out.Products[0].Disposition != domain.ProductDispositionUnchanged {
		t.Fatalf("expected unchanged, got %s", out.Products[0].Disposition)
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

// local helper to avoid adding extra packages
func jsonDecode(b []byte, v any) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	return dec.Decode(v)
}
