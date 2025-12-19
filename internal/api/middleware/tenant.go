package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
)

const TenantHeaderKey = "X-Tenant-ID"

type TenantMiddleware struct {
	Env  string // "dev" enables header override
	Next http.Handler
}

func (m TenantMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Next == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tenantID := tenantctx.DefaultTenantID

	// Only allow header override in dev
	if strings.EqualFold(strings.TrimSpace(m.Env), "dev") {
		raw := strings.TrimSpace(r.Header.Get(TenantHeaderKey))
		if raw != "" {
			v, err := strconv.ParseUint(raw, 10, 64)
			if err != nil || v == 0 {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"invalid_tenant_id","message":"X-Tenant-ID must be a positive integer"}`))
				return
			}
			tenantID = v
		}
	}

	ctx := tenantctx.WithTenantID(r.Context(), tenantID)
	m.Next.ServeHTTP(w, r.WithContext(ctx))
}
