package ingest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type countingHandler struct {
	count int
	next  http.Handler
}

func (c *countingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.count++
	c.next.ServeHTTP(w, r)
}

func TestIdempotencyMiddleware_CachesResponse(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Hour)

	// Underlying debug handler
	hashStore := NewMemoryHashStore()
	proc := NewProcessor()
	debug := DebugUpsertHandler{
		Processor:       proc,
		Store:           hashStore,
		EnabledChannels: []string{"google"},
	}

	counter := &countingHandler{next: debug}

	mw := IdempotencyMiddleware{
		Store: store,
		Next:  counter,
	}

	body := `[
  {
    "product_key": "sku1",
    "title": "Test Product",
    "description": "Test Description",
    "link": "https://example.com/p/sku1",
    "image_link": "https://example.com/p/sku1.jpg",
    "condition": "new",
    "availability": "in_stock",
    "price": { "amount_decimal": "19.99", "currency": "USD" },
    "channel": { "google": { "control": { "state": "active" } } }
  }
]`

	req1 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(body))
	req1.Header.Set(IdempotencyHeaderKey, "abc123")
	rec1 := httptest.NewRecorder()

	mw.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}

	req2 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(body))
	req2.Header.Set(IdempotencyHeaderKey, "abc123")
	rec2 := httptest.NewRecorder()

	mw.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	if counter.count != 1 {
		t.Fatalf("expected underlying handler to be called once, got %d", counter.count)
	}

	// Responses should be identical bytes
	if rec1.Body.String() != rec2.Body.String() {
		t.Fatalf("expected identical cached response bodies")
	}

	// Ensure it still decodes as RunResponse
	var rr RunResponse
	if err := json.NewDecoder(bytes.NewReader(rec2.Body.Bytes())).Decode(&rr); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if rr.RunID == "" {
		t.Fatalf("expected run_id")
	}
}

func TestIdempotencyMiddleware_NoHeaderPassThrough(t *testing.T) {
	store := NewMemoryIdempotencyStore(1 * time.Hour)

	called := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	mw := IdempotencyMiddleware{
		Store: store,
		Next:  next,
	}

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	rec := httptest.NewRecorder()

	mw.ServeHTTP(rec, req)

	if called != 1 {
		t.Fatalf("expected called=1, got %d", called)
	}
}
