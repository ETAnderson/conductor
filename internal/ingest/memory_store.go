package ingest

import "sync"

type MemoryHashStore struct {
	mu     sync.RWMutex
	hashes map[string]string
}

func NewMemoryHashStore() *MemoryHashStore {
	return &MemoryHashStore{
		hashes: make(map[string]string),
	}
}

func (s *MemoryHashStore) Get(productKey string) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	h, ok := s.hashes[productKey]
	return h, ok, nil
}

func (s *MemoryHashStore) Set(productKey string, hash string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hashes[productKey] = hash
}
