package worker

import (
	"context"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/state"
)

func TestRunner_DefaultsAndStopsOnContext(t *testing.T) {
	r := Runner{
		Store: state.NewMemoryStore(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := r.Run(ctx)
	if err == nil {
		t.Fatalf("expected context error, got nil")
	}
}
