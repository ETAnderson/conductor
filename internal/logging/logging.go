package logging

import (
	"log"
	"os"
)

func NewStdLogger(prefix string) *log.Logger {
	return log.New(os.Stdout, prefix, log.LstdFlags|log.LUTC)
}
