package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Config struct {
	DSN string
}

func Open(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, err
	}

	// Conservative defaults (tune later)
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(20)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

func Ping(ctx context.Context, db *sql.DB) error {
	c, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return db.PingContext(c)
}
