package state

import (
	"context"
	"time"
)

func (s *MemoryStore) GetProductDoc(ctx context.Context, tenantID uint64, productKey string) (ProductDocRecord, bool, error) {
	_ = ctx

	s.mu.RLock()
	defer s.mu.RUnlock()

	tm, ok := s.productDocs[tenantID]
	if !ok {
		return ProductDocRecord{}, false, nil
	}

	rec, ok := tm[productKey]
	return rec, ok, nil
}

func (s *MemoryStore) UpsertProductDoc(ctx context.Context, tenantID uint64, productKey string, rec ProductDocRecord) error {
	_ = ctx

	now := time.Now().UTC()

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.productDocs == nil {
		s.productDocs = make(map[uint64]map[string]ProductDocRecord)
	}
	if s.productDocs[tenantID] == nil {
		s.productDocs[tenantID] = make(map[string]ProductDocRecord)
	}

	existing, ok := s.productDocs[tenantID][productKey]
	if ok {
		rec.CreatedAt = existing.CreatedAt
		rec.UpdatedAt = now
	} else {
		rec.CreatedAt = now
		rec.UpdatedAt = now
	}

	s.productDocs[tenantID][productKey] = rec
	return nil
}
