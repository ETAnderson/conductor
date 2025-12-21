package state

import (
	"context"
)

func (s *MySQLStore) InsertRunChannelResult(ctx context.Context, rec RunChannelResultRecord) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO run_channel_results (run_id, tenant_id, channel, attempt, ok_count, err_count)
VALUES (?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  attempt = VALUES(attempt),
  ok_count = VALUES(ok_count),
  err_count = VALUES(err_count)
`, rec.RunID, rec.TenantID, rec.Channel, rec.Attempt, rec.OkCount, rec.ErrCount)

	return err
}

func (s *MySQLStore) InsertRunChannelItems(ctx context.Context, runID string, channel string, items []RunChannelItemRecord) error {
	// simple v1 approach: delete then insert (run+channel scope is small per run)
	_, err := s.db.ExecContext(ctx, `
DELETE FROM run_channel_items
WHERE run_id = ? AND channel = ?
`, runID, channel)
	if err != nil {
		return err
	}

	if len(items) == 0 {
		return nil
	}

	// batched inserts
	for _, it := range items {
		_, err := s.db.ExecContext(ctx, `
INSERT INTO run_channel_items (run_id, channel, product_key, status, message)
VALUES (?, ?, ?, ?, ?)
`, it.RunID, it.Channel, it.ProductKey, it.Status, it.Message)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *MySQLStore) ListRunChannelResults(ctx context.Context, tenantID uint64, runID string) ([]RunChannelResultRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, tenant_id, channel, attempt, ok_count, err_count, created_at
FROM run_channel_results
WHERE tenant_id = ? AND run_id = ?
ORDER BY channel ASC
`, tenantID, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RunChannelResultRecord, 0, 8)
	for rows.Next() {
		var r RunChannelResultRecord
		if err := rows.Scan(&r.RunID, &r.TenantID, &r.Channel, &r.Attempt, &r.OkCount, &r.ErrCount, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}

	return out, rows.Err()
}

func (s *MySQLStore) ListRunChannelItems(ctx context.Context, runID string, channel string, limit int) ([]RunChannelItemRecord, error) {
	if limit <= 0 {
		limit = 1000
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT run_id, channel, product_key, status, message
FROM run_channel_items
WHERE run_id = ? AND channel = ?
ORDER BY product_key ASC
LIMIT ?
`, runID, channel, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RunChannelItemRecord, 0, limit)
	for rows.Next() {
		var it RunChannelItemRecord
		if err := rows.Scan(&it.RunID, &it.Channel, &it.ProductKey, &it.Status, &it.Message); err != nil {
			return nil, err
		}
		out = append(out, it)
	}

	return out, rows.Err()
}
