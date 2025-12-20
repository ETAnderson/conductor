package worker

import "context"

type RunExecutor interface {
	Execute(ctx context.Context, runID string, tenantID uint64) error
}
