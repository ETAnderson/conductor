package worker

import (
	"context"
	"testing"
)

func TestWithRunID_RoundTrip(t *testing.T) {
	ctx := context.Background()

	if got := RunID(ctx); got != "" {
		t.Fatalf("expected empty run id, got %q", got)
	}

	ctx = WithRunID(ctx, "run_123")
	if got := RunID(ctx); got != "run_123" {
		t.Fatalf("expected run_123, got %q", got)
	}
}
