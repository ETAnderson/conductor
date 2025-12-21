package state

import (
	"context"
	"sort"
	"time"
)

func (s *MemoryStore) InsertRunChannelResult(ctx context.Context, rec RunChannelResultRecord) error {
	_ = ctx

	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.runChannelResults[rec.RunID] = append(s.runChannelResults[rec.RunID], rec)
	return nil
}

func (s *MemoryStore) InsertRunChannelItems(ctx context.Context, runID string, channel string, items []RunChannelItemRecord) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.runChannelItems[runID] == nil {
		s.runChannelItems[runID] = make(map[string][]RunChannelItemRecord)
	}

	cp := make([]RunChannelItemRecord, len(items))
	copy(cp, items)

	s.runChannelItems[runID][channel] = cp
	return nil
}

func (s *MemoryStore) ListRunChannelResults(ctx context.Context, tenantID uint64, runID string) ([]RunChannelResultRecord, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	// enforce tenant ownership via run lookup
	r, ok := s.runs[runID]
	if !ok || r.TenantID != tenantID {
		return []RunChannelResultRecord{}, nil
	}

	items := s.runChannelResults[runID]
	out := make([]RunChannelResultRecord, len(items))
	copy(out, items)

	sort.Slice(out, func(i, j int) bool {
		return out[i].Channel < out[j].Channel
	})

	return out, nil
}

func (s *MemoryStore) ListRunChannelItems(ctx context.Context, runID string, channel string, limit int) ([]RunChannelItemRecord, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	m := s.runChannelItems[runID]
	if m == nil {
		return []RunChannelItemRecord{}, nil
	}

	items := m[channel]
	out := make([]RunChannelItemRecord, len(items))
	copy(out, items)

	sort.Slice(out, func(i, j int) bool {
		return out[i].ProductKey < out[j].ProductKey
	})

	if limit <= 0 || limit > len(out) {
		return out, nil
	}
	return out[:limit], nil
}
