package middleware

import (
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/ETAnderson/conductor/internal/api/auth"
	"github.com/ETAnderson/conductor/internal/api/tenantctx"
)

type AuthMiddleware struct {
	Env       string
	PublicKey *rsa.PublicKey
	Next      http.Handler
}

func (m AuthMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Next == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// In dev, if tenant was explicitly set via X-Tenant-ID (TenantMiddleware),
	// allow it as a fallback to avoid blocking local testing tooling.
	if strings.EqualFold(strings.TrimSpace(m.Env), "dev") {
		if tenantctx.TenantID(r.Context()) != tenantctx.DefaultTenantID || strings.TrimSpace(r.Header.Get("Authorization")) == "" {
			m.Next.ServeHTTP(w, r)
			return
		}
	}

	// In non-dev (or if Authorization is present in dev), require a valid Bearer token.
	authz := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authz, "Bearer ") {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized","message":"missing bearer token"}`))
		return
	}

	tokenString := strings.TrimSpace(strings.TrimPrefix(authz, "Bearer "))
	if tokenString == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized","message":"empty bearer token"}`))
		return
	}

	claims, err := auth.ParseAndValidateRS256(tokenString, m.PublicKey)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"unauthorized","message":"invalid token"}`))
		return
	}

	ctx := tenantctx.WithTenantID(r.Context(), claims.TenantID)
	m.Next.ServeHTTP(w, r.WithContext(ctx))
}
