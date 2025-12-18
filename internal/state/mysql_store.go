package state

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/ETAnderson/conductor/internal/ingest"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(db *sql.DB) *MySQLStore {
	return &MySQLStore{db: db}
}

func (s *MySQLStore) GetProductHash(ctx context.Context, tenantID uint64, productKey string) (string, bool, error) {
	var h string
	err := s.db.QueryRowContext(
		ctx,
		`SELECT normalized_hash FROM product_state WHERE tenant_id = ? AND product_key = ?`,
		tenantID, productKey,
	).Scan(&h)

	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return h, true, nil
}

func (s *MySQLStore) UpsertProductHash(ctx context.Context, tenantID uint64, productKey string, hash string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO product_state (tenant_id, product_key, normalized_hash)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE normalized_hash = VALUES(normalized_hash)`,
		tenantID, productKey, hash,
	)
	return err
}

func (s *MySQLStore) InsertRun(ctx context.Context, run RunRecord) error {
	wb, err := json.Marshal(run.Warnings)
	if err != nil {
		return err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO runs (
			run_id, tenant_id, feed_id, status, push_triggered,
			received, valid, rejected, unchanged, enqueued,
			warnings_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.RunID, run.TenantID, run.FeedID, run.Status, run.PushTriggered,
		run.Received, run.Valid, run.Rejected, run.Unchanged, run.Enqueued,
		wb, run.CreatedAt.UTC(),
	)
	return err
}

func (s *MySQLStore) InsertRunProducts(ctx context.Context, runID string, products []ingest.ProductProcessResult) error {
	// Simple row-by-row insert (optimize to bulk insert later)
	for _, p := range products {
		issues, err := json.Marshal(p.Issues)
		if err != nil {
			return err
		}

		_, err = s.db.ExecContext(
			ctx,
			`INSERT INTO run_products (run_id, product_key, disposition, reason, normalized_hash, issues_json)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			runID, p.ProductKey, p.Disposition, p.Reason, p.Hash, issues,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *MySQLStore) GetIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string) (IdempotencyRecord, bool, error) {
	var status int
	var body []byte
	var created time.Time
	var expires time.Time

	err := s.db.QueryRowContext(
		ctx,
		`SELECT status_code, response_body_json, created_at, expires_at
		 FROM idempotency
		 WHERE tenant_id = ? AND endpoint = ? AND idem_key_hash = ?`,
		tenantID, endpoint, idemKeyHash,
	).Scan(&status, &body, &created, &expires)

	if err == sql.ErrNoRows {
		return IdempotencyRecord{}, false, nil
	}
	if err != nil {
		return IdempotencyRecord{}, false, err
	}

	if time.Now().UTC().After(expires.UTC()) {
		return IdempotencyRecord{}, false, nil
	}

	return IdempotencyRecord{
		StatusCode: status,
		BodyJSON:   body,
		CreatedAt:  created.UTC(),
		ExpiresAt:  expires.UTC(),
	}, true, nil
}

func (s *MySQLStore) PutIdempotency(ctx context.Context, tenantID uint64, endpoint string, idemKeyHash string, rec IdempotencyRecord) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO idempotency (tenant_id, endpoint, idem_key_hash, status_code, response_body_json, created_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   status_code = VALUES(status_code),
		   response_body_json = VALUES(response_body_json),
		   created_at = VALUES(created_at),
		   expires_at = VALUES(expires_at)`,
		tenantID, endpoint, idemKeyHash, rec.StatusCode, rec.BodyJSON, rec.CreatedAt.UTC(), rec.ExpiresAt.UTC(),
	)
	return err
}
