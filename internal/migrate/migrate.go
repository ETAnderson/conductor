package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func ApplyDir(ctx context.Context, db *sql.DB, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	files := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasSuffix(strings.ToLower(name), ".sql") {
			files = append(files, filepath.Join(dir, name))
		}
	}

	sort.Strings(files)

	if err := ensureSchemaMigrations(ctx, db); err != nil {
		return err
	}

	for _, path := range files {
		name := filepath.Base(path)

		applied, err := isApplied(ctx, db, name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("migration %s failed: %w", name, err)
		}

		if err := markApplied(ctx, db, name); err != nil {
			return err
		}
	}

	return nil
}

func ensureSchemaMigrations(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
  name VARCHAR(255) NOT NULL,
  applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (name)
) ENGINE=InnoDB;
`)
	return err
}

func isApplied(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var v string
	err := db.QueryRowContext(ctx, `SELECT name FROM schema_migrations WHERE name = ?`, name).Scan(&v)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func markApplied(ctx context.Context, db *sql.DB, name string) error {
	_, err := db.ExecContext(ctx, `INSERT INTO schema_migrations (name) VALUES (?)`, name)
	return err
}
