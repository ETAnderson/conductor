package state

import (
	"context"
	"database/sql"
)

func (s *MySQLStore) ClaimRuns(ctx context.Context, limit int) ([]RunClaim, error) {
	if limit <= 0 {
		limit = 10
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `
SELECT run_id, tenant_id
FROM runs
WHERE status = 'has_changes' AND push_triggered = 1
ORDER BY created_at ASC
LIMIT ?
FOR UPDATE
`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var claims []RunClaim
	for rows.Next() {
		var c RunClaim
		if err := rows.Scan(&c.RunID, &c.TenantID); err != nil {
			return nil, err
		}
		claims = append(claims, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Mark them processing
	for _, c := range claims {
		_, err := tx.ExecContext(ctx, `
UPDATE runs
SET status = 'processing'
WHERE run_id = ? AND tenant_id = ? AND status = 'has_changes'
`, c.RunID, c.TenantID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return claims, nil
}

func (s *MySQLStore) CompleteRun(ctx context.Context, tenantID uint64, runID string) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE runs
SET status = 'completed'
WHERE run_id = ? AND tenant_id = ?
`, runID, tenantID)
	return err
}

func (s *MySQLStore) FailRun(ctx context.Context, tenantID uint64, runID string, message string) error {
	_ = message
	_, err := s.db.ExecContext(ctx, `
UPDATE runs
SET status = 'failed'
WHERE run_id = ? AND tenant_id = ?
`, runID, tenantID)
	return err
}
