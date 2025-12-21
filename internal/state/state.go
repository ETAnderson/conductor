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

type RunClaim struct {
	RunID    string
	TenantID uint64
}

type IdempotencyRecord struct {
	StatusCode int
	BodyJSON   []byte
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

type ProductDocRecord struct {
	ProductJSON []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Store interface {
	// Canonical product state
	GetProductHash(ctx context.Context, tenantID uint64, productKey string) (hash string, ok bool, err error)
	UpsertProductHash(ctx context.Context, tenantID uint64, productKey string, hash string) error

	// Runs (write)
	InsertRun(ctx context.Context, run RunRecord) error
	InsertRunProducts(ctx context.Context, runID string, products []ingest.ProductProcessResult) error

	// Idempotency cache
	GetIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string) (IdempotencyRecord, bool, error)
	PutIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string, rec IdempotencyRecord) error

	// Runs (read/debug)
	ListRuns(ctx context.Context, tenantID uint64, limit int) ([]RunRecord, error)
	GetRun(ctx context.Context, tenantID uint64, runID string) (RunRecord, bool, error)
	ListRunProducts(ctx context.Context, runID string, limit int) ([]ingest.ProductProcessResult, error)

	// Worker queue (runs)
	ClaimRuns(ctx context.Context, limit int) ([]RunClaim, error)
	CompleteRun(ctx context.Context, tenantID uint64, runID string) error
	FailRun(ctx context.Context, tenantID uint64, runID string, message string) error

	// Canonical product docs (normalized JSON)
	GetProductDoc(ctx context.Context, tenantID uint64, productKey string) (ProductDocRecord, bool, error)
	UpsertProductDoc(ctx context.Context, tenantID uint64, productKey string, rec ProductDocRecord) error

	// Channel results (write)
	InsertRunChannelResult(ctx context.Context, rec RunChannelResultRecord) error
	InsertRunChannelItems(ctx context.Context, runID string, channel string, items []RunChannelItemRecord) error

	// Channel results (read/debug)
	ListRunChannelResults(ctx context.Context, tenantID uint64, runID string) ([]RunChannelResultRecord, error)
	ListRunChannelItems(ctx context.Context, runID string, channel string, limit int) ([]RunChannelItemRecord, error)
}

type RunChannelResultRecord struct {
	RunID     string
	TenantID  uint64
	Channel   string
	Attempt   int
	OkCount   int
	ErrCount  int
	CreatedAt time.Time
}

type RunChannelItemRecord struct {
	RunID      string
	Channel    string
	ProductKey string
	Status     string
	Message    string
}
