package google

import (
	"context"

	"github.com/ETAnderson/conductor/internal/channels"
)

type Channel struct{}

func (c Channel) Name() string { return "google" }

// Build is a skeleton: it does NOT call Google.
// For now, it just returns an "ok" outcome for each product ref.
func (c Channel) Build(ctx context.Context, tenantID uint64, products []channels.ProductRef) (channels.BuildResult, error) {
	_ = ctx
	_ = tenantID

	items := make([]channels.ProductOutcome, 0, len(products))
	for _, p := range products {
		items = append(items, channels.ProductOutcome{
			ProductKey: p.ProductKey,
			Status:     "ok",
			Message:    "queued_for_google_push (stub)",
		})
	}

	return channels.BuildResult{
		Channel:  c.Name(),
		Attempt:  1,
		OkCount:  len(items),
		ErrCount: 0,
		Items:    items,
	}, nil
}
