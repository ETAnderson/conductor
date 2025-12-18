package state

import (
	"context"
	"time"

	"github.com/ETAnderson/conductor/internal/ingest"
)

type RunRecord struct {
	RunID         string
	TenantID      uint64
	FeedID        *uint64
	Status        string
	PushTriggered bool

	Received  int
	Valid     int
	Rejected  int
	Unchanged int
	Enqueued  int

	Warnings  ingest.UnknownKeyWarning
	CreatedAt time.Time
}

type IdempotencyRecord struct {
	StatusCode int
	BodyJSON   []byte
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

type Store interface {
	// Canonical product state
	GetProductHash(ctx context.Context, tenantID uint64, productKey string) (hash string, ok bool, err error)
	UpsertProductHash(ctx context.Context, tenantID uint64, productKey string, hash string) error

	// Runs
	InsertRun(ctx context.Context, run RunRecord) error
	InsertRunProducts(ctx context.Context, runID string, products []ingest.ProductProcessResult) error

	// Idempotency cache
	GetIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string) (IdempotencyRecord, bool, error)
	PutIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string, rec IdempotencyRecord) error
}
