package state

import (
	"context"
	"sort"
)

func (s *MemoryStore) ClaimRuns(ctx context.Context, limit int) ([]RunClaim, error) {
	_ = ctx

	if limit <= 0 {
		limit = 10
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var candidates []RunRecord
	for _, r := range s.runs {
		if r.Status == "has_changes" && r.PushTriggered && r.TenantID != 0 {
			candidates = append(candidates, r)
		}
	}

	// Oldest first (stable-ish order for consistent processing)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].CreatedAt.Before(candidates[j].CreatedAt)
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	out := make([]RunClaim, 0, len(candidates))
	for _, r := range candidates {
		// Mark claimed
		r.Status = "processing"
		s.runs[r.RunID] = r

		out = append(out, RunClaim{
			RunID:    r.RunID,
			TenantID: r.TenantID,
		})
	}

	return out, nil
}

func (s *MemoryStore) CompleteRun(ctx context.Context, tenantID uint64, runID string) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.runs[runID]
	if !ok {
		return nil
	}
	if r.TenantID != tenantID {
		return nil
	}

	r.Status = "completed"
	s.runs[runID] = r
	return nil
}

func (s *MemoryStore) FailRun(ctx context.Context, tenantID uint64, runID string, message string) error {
	_ = ctx
	_ = message

	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.runs[runID]
	if !ok {
		return nil
	}
	if r.TenantID != tenantID {
		return nil
	}

	r.Status = "failed"
	s.runs[runID] = r
	return nil
}
