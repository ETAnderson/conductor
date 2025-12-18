package ingest

import (
	"sync"
	"time"
)

type IdempotencyRecord struct {
	StatusCode int
	Body       []byte
	CreatedAt  time.Time
}

type MemoryIdempotencyStore struct {
	mu      sync.RWMutex
	records map[string]IdempotencyRecord
	ttl     time.Duration
}

func NewMemoryIdempotencyStore(ttl time.Duration) *MemoryIdempotencyStore {
	return &MemoryIdempotencyStore{
		records: make(map[string]IdempotencyRecord),
		ttl:     ttl,
	}
}

func (s *MemoryIdempotencyStore) Get(key string) (IdempotencyRecord, bool) {
	if key == "" {
		return IdempotencyRecord{}, false
	}

	s.mu.RLock()
	rec, ok := s.records[key]
	s.mu.RUnlock()

	if !ok {
		return IdempotencyRecord{}, false
	}

	// TTL enforcement
	if s.ttl > 0 && time.Since(rec.CreatedAt) > s.ttl {
		s.mu.Lock()
		delete(s.records, key)
		s.mu.Unlock()
		return IdempotencyRecord{}, false
	}

	return rec, true
}

func (s *MemoryIdempotencyStore) Set(key string, rec IdempotencyRecord) {
	if key == "" {
		return
	}

	s.mu.Lock()
	s.records[key] = rec
	s.mu.Unlock()
}
