package ingest

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDebugBulkUpsertHandler_NDJSON_EnqueueThenUnchanged(t *testing.T) {
	store := NewMemoryHashStore()
	proc := NewProcessor()

	h := DebugBulkUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	ndjson := "" +
		`{"product_key":"sku1","title":"T1","description":"D1","link":"https://e.com/p/sku1","image_link":"https://e.com/i1.jpg","condition":"new","availability":"in_stock","price":{"amount_decimal":"1.00","currency":"USD"},"channel":{"google":{"control":{"state":"active"}}},"extra":"x"}` + "\n" +
		`{"product_key":"sku2","title":"T2","description":"D2","link":"https://e.com/p/sku2","image_link":"https://e.com/i2.jpg","condition":"new","availability":"in_stock","price":{"amount_decimal":"2.00","currency":"USD"},"channel":{"google":{"control":{"state":"active"}}}}` + "\n"

	// First call -> enqueued=2
	req1 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert-bulk", bytes.NewBufferString(ndjson))
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}

	var resp1 RunResponse
	if err := json.NewDecoder(bytes.NewReader(rec1.Body.Bytes())).Decode(&resp1); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp1.Result.Summary.Enqueued != 2 {
		t.Fatalf("expected enqueued=2, got %#v", resp1.Result.Summary)
	}
	if len(resp1.Warnings.UnknownKeys) == 0 {
		t.Fatalf("expected unknown key warning")
	}

	// Second call with same NDJSON -> unchanged=2
	req2 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert-bulk", bytes.NewBufferString(ndjson))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	var resp2 RunResponse
	if err := json.NewDecoder(bytes.NewReader(rec2.Body.Bytes())).Decode(&resp2); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp2.Result.Summary.Unchanged != 2 {
		t.Fatalf("expected unchanged=2, got %#v", resp2.Result.Summary)
	}
}

func TestDebugBulkUpsertHandler_GzipNDJSON_Works(t *testing.T) {
	store := NewMemoryHashStore()
	proc := NewProcessor()

	h := DebugBulkUpsertHandler{
		Processor:       proc,
		Store:           store,
		EnabledChannels: []string{"google"},
	}

	ndjson := `{"product_key":"sku1","title":"T1","description":"D1","link":"https://e.com/p/sku1","image_link":"https://e.com/i1.jpg","condition":"new","availability":"in_stock","price":{"amount_decimal":"1.00","currency":"USD"},"channel":{"google":{"control":{"state":"active"}}}}` + "\n"

	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, _ = zw.Write([]byte(ndjson))
	_ = zw.Close()

	req := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert-bulk", bytes.NewReader(gz.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp RunResponse
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if resp.Result.Summary.Enqueued != 1 && resp.Result.Summary.Unchanged != 1 {
		t.Fatalf("expected 1 processed, got %#v", resp.Result.Summary)
	}
}
