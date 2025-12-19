package middleware

import (
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/golang-jwt/jwt/v5"
)

func signRS256(t *testing.T, priv *rsa.PrivateKey, tenantID uint64, ttl time.Duration) string {
	t.Helper()

	now := time.Now().UTC()

	claims := jwt.MapClaims{
		"tenant_id": tenantID,
		"iss":       "conductor",
		"sub":       "test-client",
		"iat":       now.Unix(),
		"exp":       now.Add(ttl).Unix(),
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token failed: %v", err)
	}

	return s
}

func TestAuthMiddleware_Prod_RejectsMissingToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next should not be called when token is missing")
	})

	h := AuthMiddleware{
		Env:       "prod",
		PublicKey: &priv.PublicKey,
		Next:      next,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthMiddleware_Prod_RejectsInvalidToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next should not be called when token is invalid")
	})

	h := AuthMiddleware{
		Env:       "prod",
		PublicKey: &priv.PublicKey,
		Next:      next,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs", nil)
	req.Header.Set("Authorization", "Bearer not-a-jwt")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthMiddleware_Prod_RejectsExpiredToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next should not be called when token is expired")
	})

	h := AuthMiddleware{
		Env:       "prod",
		PublicKey: &priv.PublicKey,
		Next:      next,
	}

	// ttl negative => already expired
	token := signRS256(t, priv, 1, -1*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthMiddleware_Prod_AllowsValidToken_InjectsTenant(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	nextCalls := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalls++

		got := tenantctx.TenantID(r.Context())
		if got != 42 {
			t.Fatalf("expected tenant=42, got %d", got)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	h := AuthMiddleware{
		Env:       "prod",
		PublicKey: &priv.PublicKey,
		Next:      next,
	}

	token := signRS256(t, priv, 42, 10*time.Minute)

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if nextCalls != 1 {
		t.Fatalf("expected next called once, got %d", nextCalls)
	}
}

func TestAuthAndTenantMiddleware_Dev_AllowsXTenantIDWithoutToken(t *testing.T) {
	// In dev, TenantMiddleware should set tenant from header, then AuthMiddleware should allow pass-through
	// when tenant is not default or Authorization is missing (per your implementation).

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := tenantctx.TenantID(r.Context())
		if got != 7 {
			t.Fatalf("expected tenant=7, got %d", got)
		}
		w.WriteHeader(http.StatusOK)
	})

	var root http.Handler = next

	root = AuthMiddleware{
		Env:       "dev",
		PublicKey: &priv.PublicKey,
		Next:      root,
	}

	root = TenantMiddleware{
		Env:  "dev",
		Next: root,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs", nil)
	req.Header.Set("X-Tenant-ID", "7")
	rec := httptest.NewRecorder()

	root.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestAuthAndTenantMiddleware_Prod_IgnoresXTenantIDWithoutToken(t *testing.T) {
	// In prod, X-Tenant-ID should not bypass JWT requirement.

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("keygen: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next should not be called when token is missing in prod")
	})

	var root http.Handler = next

	root = AuthMiddleware{
		Env:       "prod",
		PublicKey: &priv.PublicKey,
		Next:      root,
	}

	root = TenantMiddleware{
		Env:  "prod",
		Next: root,
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/debug/runs", nil)
	req.Header.Set("X-Tenant-ID", "7")
	rec := httptest.NewRecorder()

	root.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rec.Code, rec.Body.String())
	}
}
