package worker

import (
	"context"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
)

type ctxKey string

const runIDKey ctxKey = "worker_run_id"

// WithRunID stores the run ID on the context.
func WithRunID(ctx context.Context, runID string) context.Context {
	if runID == "" {
		return ctx
	}
	return context.WithValue(ctx, runIDKey, runID)
}

// RunID reads the run ID from context.
func RunID(ctx context.Context) string {
	v := ctx.Value(runIDKey)
	s, _ := v.(string)
	return s
}

// WithTenant stores tenant ID on context using the shared tenantctx package.
func WithTenant(ctx context.Context, tenantID uint64) context.Context {
	return tenantctx.WithTenantID(ctx, tenantID)
}
