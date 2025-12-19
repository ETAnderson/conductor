package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/state"
)

func TestIdempotencyMiddleware_CachesResponseViaStateStore(t *testing.T) {
	store := state.NewMemoryStore()

	calls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"run_id":"run_x","status":"has_changes"}`))
	})

	mw := IdempotencyMiddleware{
		Store: store,
		Next:  next,
	}

	req1 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(`[]`))
	req1 = req1.WithContext(tenantctx.WithTenantID(req1.Context(), 1))
	req1.Header.Set(IdempotencyHeaderKey, "abc123")
	rec1 := httptest.NewRecorder()
	mw.ServeHTTP(rec1, req1)

	req2 := httptest.NewRequest(http.MethodPost, "/v1/debug/products:upsert", bytes.NewBufferString(`[]`))
	req2 = req2.WithContext(tenantctx.WithTenantID(req2.Context(), 1))
	req2.Header.Set(IdempotencyHeaderKey, "abc123")
	rec2 := httptest.NewRecorder()
	mw.ServeHTTP(rec2, req2)

	if calls != 1 {
		t.Fatalf("expected underlying handler called once, got %d", calls)
	}
	if rec1.Body.String() != rec2.Body.String() {
		t.Fatalf("expected cached response match")
	}
}
