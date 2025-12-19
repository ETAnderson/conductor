package state

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStore_ClaimRuns_MarksProcessingAndIsOrdered(t *testing.T) {
	st := NewMemoryStore()

	now := time.Now().UTC()

	// claimable
	_ = st.InsertRun(context.Background(), RunRecord{
		RunID:         "run1",
		TenantID:      1,
		Status:        "has_changes",
		PushTriggered: true,
		CreatedAt:     now.Add(-2 * time.Minute),
	})
	_ = st.InsertRun(context.Background(), RunRecord{
		RunID:         "run2",
		TenantID:      1,
		Status:        "has_changes",
		PushTriggered: true,
		CreatedAt:     now.Add(-1 * time.Minute),
	})

	// not claimable
	_ = st.InsertRun(context.Background(), RunRecord{
		RunID:         "run3",
		TenantID:      1,
		Status:        "no_change_detected",
		PushTriggered: false,
		CreatedAt:     now,
	})

	claims, err := st.ClaimRuns(context.Background(), 10)
	if err != nil {
		t.Fatalf("ClaimRuns err: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("expected 2 claims, got %d", len(claims))
	}
	if claims[0].RunID != "run1" || claims[1].RunID != "run2" {
		t.Fatalf("expected run1 then run2, got %+v", claims)
	}

	// Ensure they are marked processing and not re-claimable
	claims2, err := st.ClaimRuns(context.Background(), 10)
	if err != nil {
		t.Fatalf("ClaimRuns(2) err: %v", err)
	}
	if len(claims2) != 0 {
		t.Fatalf("expected 0 claims after processing mark, got %d", len(claims2))
	}
}
