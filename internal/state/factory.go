package state

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/ETAnderson/conductor/internal/db"
)

type FactoryConfig struct {
	Backend  string
	MySQLDSN string
}

type FactoryResult struct {
	Store Store
	DB    *sql.DB // only set for mysql
}

func NewStore(ctx context.Context, cfg FactoryConfig) (FactoryResult, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	if backend == "" {
		backend = "memory"
	}

	switch backend {
	case "memory":
		return FactoryResult{Store: NewMemoryStore()}, nil

	case "mysql":
		if strings.TrimSpace(cfg.MySQLDSN) == "" {
			return FactoryResult{}, errors.New("DB_DSN is required when STATE_BACKEND=mysql")
		}

		sqlDB, err := db.Open(db.Config{DSN: cfg.MySQLDSN})
		if err != nil {
			return FactoryResult{}, err
		}

		c, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		if err := sqlDB.PingContext(c); err != nil {
			_ = sqlDB.Close()
			return FactoryResult{}, err
		}

		return FactoryResult{
			Store: NewMySQLStore(sqlDB),
			DB:    sqlDB,
		}, nil

	default:
		return FactoryResult{}, errors.New("unknown STATE_BACKEND (use memory or mysql)")
	}
}
