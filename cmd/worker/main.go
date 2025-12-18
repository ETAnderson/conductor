package main

import (
	"time"

	"github.com/ETAnderson/conductor/internal/config"
	"github.com/ETAnderson/conductor/internal/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.NewStdLogger("worker-service ")

	logger.Printf("starting (env=%s)", cfg.Env)

	for {
		time.Sleep(30 * time.Second)
		logger.Printf("alive")
	}
}
