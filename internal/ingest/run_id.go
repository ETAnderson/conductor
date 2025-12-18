package ingest

import (
	"crypto/rand"
	"encoding/hex"
)

// NewRunID creates a random run id suitable for logs + API responses.
// Format: "run_" + 16 bytes hex (32 chars)
func NewRunID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return "run_" + hex.EncodeToString(b), nil
}
