package config

import "os"

type Config struct {
	Port string
	Env  string
}

func Load() Config {
	return Config{
		Port: getenv("PORT", "8080"),
		Env:  getenv("ENV", "dev"),
	}
}

func getenv(key string, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}
