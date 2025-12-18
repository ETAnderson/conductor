package middleware

import (
	"bytes"
	"net/http"
	"time"

	"github.com/ETAnderson/conductor/internal/state"
)

const IdempotencyHeaderKey = "Idempotency-Key"

type IdempotencyMiddleware struct {
	Store    state.Store
	TenantID uint64
	Next     http.Handler
}

func (m IdempotencyMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Store == nil || m.Next == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	key := r.Header.Get(IdempotencyHeaderKey)
	if key == "" {
		m.Next.ServeHTTP(w, r)
		return
	}

	endpoint := r.URL.Path
	keyHash := state.HashIdempotencyKey(key)

	rec, ok, err := m.Store.GetIdempotency(r.Context(), m.TenantID, endpoint, keyHash)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(rec.StatusCode)
		_, _ = w.Write(rec.BodyJSON)
		return
	}

	cw := newCaptureWriter(w)
	m.Next.ServeHTTP(cw, r)

	_ = m.Store.PutIdempotency(r.Context(), m.TenantID, endpoint, keyHash, state.IdempotencyRecord{
		StatusCode: cw.statusCode(),
		BodyJSON:   cw.bodyBytes(),
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(24 * time.Hour),
	})
}

type captureWriter struct {
	w      http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func newCaptureWriter(w http.ResponseWriter) *captureWriter {
	return &captureWriter{w: w}
}

func (c *captureWriter) Header() http.Header { return c.w.Header() }

func (c *captureWriter) WriteHeader(code int) {
	c.status = code
	c.w.WriteHeader(code)
}

func (c *captureWriter) Write(b []byte) (int, error) {
	c.buf.Write(b)
	return c.w.Write(b)
}

func (c *captureWriter) statusCode() int {
	if c.status == 0 {
		return http.StatusOK
	}
	return c.status
}

func (c *captureWriter) bodyBytes() []byte { return c.buf.Bytes() }
