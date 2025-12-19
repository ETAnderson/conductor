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

func (s *MySQLStore) ListRuns(ctx context.Context, tenantID uint64, limit int) ([]RunRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, tenant_id, feed_id, status, push_triggered,
       received, valid, rejected, unchanged, enqueued,
       warnings_json, created_at
FROM runs
WHERE tenant_id = ?
ORDER BY created_at DESC
LIMIT ?`, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RunRecord, 0, limit)

	for rows.Next() {
		var r RunRecord
		var feedID sql.NullInt64
		var push int
		var warningsBytes []byte
		var created time.Time

		err := rows.Scan(
			&r.RunID,
			&r.TenantID,
			&feedID,
			&r.Status,
			&push,
			&r.Received,
			&r.Valid,
			&r.Rejected,
			&r.Unchanged,
			&r.Enqueued,
			&warningsBytes,
			&created,
		)
		if err != nil {
			return nil, err
		}

		if feedID.Valid {
			v := uint64(feedID.Int64)
			r.FeedID = &v
		}

		r.PushTriggered = push == 1
		r.CreatedAt = created.UTC()

		if len(warningsBytes) > 0 {
			_ = json.Unmarshal(warningsBytes, &r.Warnings)
		}

		out = append(out, r)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *MySQLStore) GetRun(ctx context.Context, tenantID uint64, runID string) (RunRecord, bool, error) {
	var r RunRecord
	var feedID sql.NullInt64
	var push int
	var warningsBytes []byte
	var created time.Time

	err := s.db.QueryRowContext(ctx, `
SELECT run_id, tenant_id, feed_id, status, push_triggered,
       received, valid, rejected, unchanged, enqueued,
       warnings_json, created_at
FROM runs
WHERE tenant_id = ? AND run_id = ?`, tenantID, runID).
		Scan(
			&r.RunID,
			&r.TenantID,
			&feedID,
			&r.Status,
			&push,
			&r.Received,
			&r.Valid,
			&r.Rejected,
			&r.Unchanged,
			&r.Enqueued,
			&warningsBytes,
			&created,
		)

	if err == sql.ErrNoRows {
		return RunRecord{}, false, nil
	}
	if err != nil {
		return RunRecord{}, false, err
	}

	if feedID.Valid {
		v := uint64(feedID.Int64)
		r.FeedID = &v
	}
	r.PushTriggered = push == 1
	r.CreatedAt = created.UTC()

	if len(warningsBytes) > 0 {
		_ = json.Unmarshal(warningsBytes, &r.Warnings)
	}

	return r, true, nil
}

func (s *MySQLStore) ListRunProducts(ctx context.Context, runID string, limit int) ([]ingest.ProductProcessResult, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT product_key, disposition, reason, normalized_hash, issues_json
FROM run_products
WHERE run_id = ?
ORDER BY product_key ASC
LIMIT ?`, runID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]ingest.ProductProcessResult, 0, limit)

	for rows.Next() {
		var p ingest.ProductProcessResult
		var reason sql.NullString
		var hash sql.NullString
		var issuesBytes []byte

		err := rows.Scan(&p.ProductKey, &p.Disposition, &reason, &hash, &issuesBytes)
		if err != nil {
			return nil, err
		}

		if reason.Valid {
			p.Reason = reason.String
		}
		if hash.Valid {
			p.Hash = hash.String
		}
		if len(issuesBytes) > 0 {
			_ = json.Unmarshal(issuesBytes, &p.Issues)
		}

		out = append(out, p)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return out, nil
}
