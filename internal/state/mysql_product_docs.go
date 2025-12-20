package state

import (
	"context"
	"database/sql"
)

func (s *MySQLStore) GetProductDoc(ctx context.Context, tenantID uint64, productKey string) (ProductDocRecord, bool, error) {
	var rec ProductDocRecord

	row := s.db.QueryRowContext(ctx, `
SELECT product_json, created_at, updated_at
FROM product_docs
WHERE tenant_id = ? AND product_key = ?
`, tenantID, productKey)

	err := row.Scan(&rec.ProductJSON, &rec.CreatedAt, &rec.UpdatedAt)
	if err == sql.ErrNoRows {
		return ProductDocRecord{}, false, nil
	}
	if err != nil {
		return ProductDocRecord{}, false, err
	}

	return rec, true, nil
}

func (s *MySQLStore) UpsertProductDoc(ctx context.Context, tenantID uint64, productKey string, rec ProductDocRecord) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO product_docs (tenant_id, product_key, product_json)
VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE product_json = VALUES(product_json)
`, tenantID, productKey, rec.ProductJSON)

	return err
}
