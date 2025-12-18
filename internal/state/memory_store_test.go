package state

import (
	"context"
	"testing"
	"time"

	"github.com/ETAnderson/conductor/internal/ingest"
)

func TestMemoryStore_ProductHashRoundTrip(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	err := s.UpsertProductHash(ctx, 1, "sku1", "abc")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	h, ok, err := s.GetProductHash(ctx, 1, "sku1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !ok || h != "abc" {
		t.Fatalf("unexpected value: ok=%v hash=%s", ok, h)
	}
}

func TestMemoryStore_IdempotencyTTL(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	keyHash := HashIdempotencyKey("k1")
	now := time.Now().UTC()

	err := s.PutIdempotency(ctx, 1, "/x", keyHash, IdempotencyRecord{
		StatusCode: 200,
		BodyJSON:   []byte(`{"ok":true}`),
		CreatedAt:  now,
		ExpiresAt:  now.Add(-1 * time.Second),
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	_, ok, err := s.GetIdempotency(ctx, 1, "/x", keyHash)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ok {
		t.Fatalf("expected expired record to be treated as missing")
	}
}

func TestWarningsToJSON_IsDeterministic(t *testing.T) {
	w := ingest.UnknownKeyWarning{UnknownKeys: []string{"b", "a"}}
	j := string(WarningsToJSON(w))
	if j != `{"unknown_keys":["a","b"]}` {
		t.Fatalf("unexpected json: %s", j)
	}
}
