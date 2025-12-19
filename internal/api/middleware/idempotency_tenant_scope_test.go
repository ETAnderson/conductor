package middleware

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/state"
	"github.com/golang-jwt/jwt/v5"
)

func signRS256ForTenant(t *testing.T, priv *rsa.PrivateKey, tenantID uint64) string {
	t.Helper()

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"tenant_id": tenantID,
		"iss":       "conductor",
		"sub":       "test-client",
		"iat":       now.Unix(),
		"exp":       now.Add(10 * time.Minute).Unix(),
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}
	return s
}

func TestIdempotency_IsTenantScoped_SameKeyDifferentTenantDoesNotShareCache(t *testing.T) {
	store := state.NewMemoryStore()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	var calls int32
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)

		// Body differs per call so we can detect cache reuse.
		_, _ = w.Write([]byte(`{"ok":true,"call":` + itoa(int(n)) + `}`))
	})

	// Chain: TenantMiddleware (no-op in prod for X-Tenant-ID), AuthMiddleware (sets tenant), Idempotency
	var root http.Handler = next

	root = IdempotencyMiddleware{
		Store: store,
		Next:  root,
	}

	root = AuthMiddleware{
		Env:       "prod",
		PublicKey: &priv.PublicKey,
		Next:      root,
	}

	root = TenantMiddleware{
		Env:  "prod",
		Next: root,
	}

	const endpoint = "/v1/debug/products:upsert"
	const idemKey = "same-key"

	// Tenant 1 request #1
	tok1 := signRS256ForTenant(t, priv, 1)
	req1 := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(`[]`))
	req1.Header.Set("Authorization", "Bearer "+tok1)
	req1.Header.Set(IdempotencyHeaderKey, idemKey)
	rec1 := httptest.NewRecorder()
	root.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK {
		t.Fatalf("tenant1 req1 expected 200, got %d: %s", rec1.Code, rec1.Body.String())
	}

	// Tenant 1 request #2 (same key) => should be cached, calls stays 1
	req2 := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(`[]`))
	req2.Header.Set("Authorization", "Bearer "+tok1)
	req2.Header.Set(IdempotencyHeaderKey, idemKey)
	rec2 := httptest.NewRecorder()
	root.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("tenant1 req2 expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}

	// Tenant 2 request (same key) => should NOT hit tenant1 cache, calls becomes 2
	tok2 := signRS256ForTenant(t, priv, 2)
	req3 := httptest.NewRequest(http.MethodPost, endpoint, bytes.NewBufferString(`[]`))
	req3.Header.Set("Authorization", "Bearer "+tok2)
	req3.Header.Set(IdempotencyHeaderKey, idemKey)
	rec3 := httptest.NewRecorder()
	root.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK {
		t.Fatalf("tenant2 req expected 200, got %d: %s", rec3.Code, rec3.Body.String())
	}

	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected next called 2 times (tenant1 cached, tenant2 separate), got %d", got)
	}

	// Optional sanity: tenant1 second response should match first (cached)
	if rec2.Body.String() != rec1.Body.String() {
		t.Fatalf("expected tenant1 cached response to match; got1=%s got2=%s", rec1.Body.String(), rec2.Body.String())
	}

	// And tenant2 response should differ (fresh call)
	if rec3.Body.String() == rec1.Body.String() {
		t.Fatalf("expected tenant2 response to differ from tenant1 cached response; got=%s", rec3.Body.String())
	}
}

// tiny helper to avoid strconv import in test file
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + (n % 10))
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
