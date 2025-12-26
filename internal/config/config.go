package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env string `env:"ENV" default:"dev"`

	Port string `env:"PORT" default:"8080"`

	StateBackend string `env:"STATE_BACKEND" default:"memory"` // memory | mysql
	MySQLDSN     string `env:"DB_DSN" default:""`              // required when STATE_BACKEND=mysql

	// Optional: run migrations at startup (dev convenience)
	RunMigrations bool `env:"RUN_MIGRATIONS" default:"false"`
}

func LoadDotEnv() {
	// Loads .env into the process environment.
	// No-op if the file doesn't exist; does not override existing env vars.
	_ = godotenv.Load()
}

func Load() Config {
	cfg := Config{
		Env:           getenv("ENV", "dev"),
		Port:          getenv("PORT", "8080"),
		StateBackend:  getenv("STATE_BACKEND", "memory"),
		MySQLDSN:      getenv("DB_DSN", ""),
		RunMigrations: getenv("RUN_MIGRATIONS", "false") == "true",
	}
	return cfg
}

func getenv(key string, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
