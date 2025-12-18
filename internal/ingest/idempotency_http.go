package ingest

import (
	"bytes"
	"net/http"
)

const IdempotencyHeaderKey = "Idempotency-Key"

type IdempotencyMiddleware struct {
	Store *MemoryIdempotencyStore
	Next  http.Handler
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

	// Cache hit
	if rec, ok := m.Store.Get(key); ok {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(rec.StatusCode)
		_, _ = w.Write(rec.Body)
		return
	}

	// Cache miss: capture response
	cw := newCaptureWriter(w)
	m.Next.ServeHTTP(cw, r)

	m.Store.Set(key, IdempotencyRecord{
		StatusCode: cw.statusCode(),
		Body:       cw.bodyBytes(),
		CreatedAt:  nowUTC(),
	})
}

type captureWriter struct {
	w      http.ResponseWriter
	status int
	buf    bytes.Buffer
}

func newCaptureWriter(w http.ResponseWriter) *captureWriter {
	return &captureWriter{
		w:      w,
		status: 0,
	}
}

func (c *captureWriter) Header() http.Header {
	return c.w.Header()
}

func (c *captureWriter) WriteHeader(statusCode int) {
	c.status = statusCode
	c.w.WriteHeader(statusCode)
}

func (c *captureWriter) Write(b []byte) (int, error) {
	// Mirror to actual response and also buffer
	c.buf.Write(b)
	return c.w.Write(b)
}

func (c *captureWriter) statusCode() int {
	if c.status == 0 {
		return http.StatusOK
	}
	return c.status
}

func (c *captureWriter) bodyBytes() []byte {
	return c.buf.Bytes()
}
