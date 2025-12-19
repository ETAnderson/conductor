package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/state"
)

// HTTP header used for idempotent requests
const IdempotencyHeaderKey = "Idempotency-Key"

type IdempotencyMiddleware struct {
	Store state.Store
	Next  http.Handler
}

func (m IdempotencyMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Next == nil || m.Store == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		// continue
	default:
		m.Next.ServeHTTP(w, r)
		return
	}

	idemKey := strings.TrimSpace(r.Header.Get(IdempotencyHeaderKey))
	if idemKey == "" {
		m.Next.ServeHTTP(w, r)
		return
	}

	endpoint := strings.TrimSpace(r.URL.Path)
	if endpoint == "" {
		endpoint = "/"
	}

	tenantID := tenantctx.TenantID(r.Context())
	keyHash := sha256Hex(idemKey)

	// Cache hit?
	rec, ok, err := m.Store.GetIdempotency(r.Context(), tenantID, endpoint, keyHash)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"idempotency_lookup_failed"}`))
		return
	}

	if ok {
		// Return cached response (body only)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		status := rec.StatusCode
		if status == 0 {
			status = http.StatusOK
		}

		w.WriteHeader(status)
		_, _ = w.Write(rec.BodyJSON)
		return
	}

	// Ensure downstream can read the body (we may have already consumed it if we add future logic)
	if r.Body != nil {
		reqBody, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(reqBody))
	}

	// Record downstream response
	rr := httptest.NewRecorder()
	m.Next.ServeHTTP(rr, r)

	// Copy recorded response to the real writer
	for k, vals := range rr.Header() {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	status := rr.Code
	if status == 0 {
		status = http.StatusOK
	}

	w.WriteHeader(status)
	_, _ = w.Write(rr.Body.Bytes())

	// Cache body (and status) only
	respRec := state.IdempotencyRecord{
		StatusCode: status,
		BodyJSON:   rr.Body.Bytes(),
		CreatedAt:  time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(24 * time.Hour),
	}

	// If caching fails, do not fail the request; response has already been written.
	_ = m.Store.PutIdempotency(r.Context(), tenantID, endpoint, keyHash, respRec)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
