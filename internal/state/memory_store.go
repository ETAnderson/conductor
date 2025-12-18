package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"sync"
	"time"

	"github.com/ETAnderson/conductor/internal/ingest"
)

type MemoryStore struct {
	mu sync.RWMutex

	productHash map[uint64]map[string]string

	runs        map[string]RunRecord
	runProducts map[string][]ingest.ProductProcessResult

	idem map[uint64]map[string]map[string]IdempotencyRecord // tenant -> endpoint -> keyhash -> record
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		productHash: make(map[uint64]map[string]string),
		runs:        make(map[string]RunRecord),
		runProducts: make(map[string][]ingest.ProductProcessResult),
		idem:        make(map[uint64]map[string]map[string]IdempotencyRecord),
	}
}

func (s *MemoryStore) GetProductHash(ctx context.Context, tenantID uint64, productKey string) (string, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.productHash[tenantID]
	if !ok {
		return "", false, nil
	}
	h, ok := m[productKey]
	return h, ok, nil
}

func (s *MemoryStore) UpsertProductHash(ctx context.Context, tenantID uint64, productKey string, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.productHash[tenantID]
	if !ok {
		m = make(map[string]string)
		s.productHash[tenantID] = m
	}
	m[productKey] = hash
	return nil
}

func (s *MemoryStore) InsertRun(ctx context.Context, run RunRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runs[run.RunID] = run
	return nil
}

func (s *MemoryStore) InsertRunProducts(ctx context.Context, runID string, products []ingest.ProductProcessResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := make([]ingest.ProductProcessResult, len(products))
	copy(cp, products)
	s.runProducts[runID] = cp
	return nil
}

func (s *MemoryStore) GetIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string) (IdempotencyRecord, bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	te, ok := s.idem[tenantID]
	if !ok {
		return IdempotencyRecord{}, false, nil
	}
	ep, ok := te[endpoint]
	if !ok {
		return IdempotencyRecord{}, false, nil
	}
	rec, ok := ep[idemKeyHash]
	if !ok {
		return IdempotencyRecord{}, false, nil
	}

	if time.Now().UTC().After(rec.ExpiresAt) {
		return IdempotencyRecord{}, false, nil
	}

	return rec, true, nil
}

func (s *MemoryStore) PutIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string, rec IdempotencyRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	te, ok := s.idem[tenantID]
	if !ok {
		te = make(map[string]map[string]IdempotencyRecord)
		s.idem[tenantID] = te
	}
	ep, ok := te[endpoint]
	if !ok {
		ep = make(map[string]IdempotencyRecord)
		te[endpoint] = ep
	}
	ep[idemKeyHash] = rec
	return nil
}

// Helper for hashing idempotency keys deterministically
func HashIdempotencyKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// Helper for stable warnings storage (optional, nice for DB too)
func WarningsToJSON(w ingest.UnknownKeyWarning) []byte {
	keys := make([]string, len(w.UnknownKeys))
	copy(keys, w.UnknownKeys)
	sort.Strings(keys)
	b, _ := json.Marshal(ingest.UnknownKeyWarning{UnknownKeys: keys})
	return b
}
