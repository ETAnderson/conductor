package tenantctx

import "context"

type ctxKeyTenantID struct{}

const DefaultTenantID uint64 = 1

func WithTenantID(ctx context.Context, tenantID uint64) context.Context {
	return context.WithValue(ctx, ctxKeyTenantID{}, tenantID)
}

func TenantID(ctx context.Context) uint64 {
	v := ctx.Value(ctxKeyTenantID{})
	if v == nil {
		return DefaultTenantID
	}

	if id, ok := v.(uint64); ok && id > 0 {
		return id
	}

	return DefaultTenantID
}
