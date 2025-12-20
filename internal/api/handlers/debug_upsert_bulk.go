package handlers

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

type DebugBulkUpsertHandler struct {
	Processor ingest.Processor
	Store     state.Store

	EnabledChannels []string
}

func (h DebugBulkUpsertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantctx.TenantID(r.Context())

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.Store == nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "misconfigured",
			"message": "handler dependencies not configured",
		})
		return
	}

	runID, err := ingest.NewRunID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "run_id_failed",
			"message": err.Error(),
		})
		return
	}

	reader, err := wrapMaybeGzip(r.Body, r.Header.Get("Content-Encoding"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_encoding",
			"message": err.Error(),
		})
		return
	}
	defer reader.Close()

	unknown := make(map[string]struct{})

	out := ingest.ProcessOutput{
		Summary:  ingest.ProcessSummary{Received: 0},
		Products: make([]ingest.ProductProcessResult, 0, 1024),
	}

	sc := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 10*1024*1024)

	lookup := func(productKey string) (string, bool, error) {
		return h.Store.GetProductHash(r.Context(), tenantID, productKey)
	}

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}

		out.Summary.Received++

		prod, unk, err := ingest.ParseProductObjectAllowUnknown(line)
		if err != nil {
			out.Products = append(out.Products, ingest.ProductProcessResult{
				ProductKey:  "",
				Disposition: domain.ProductDispositionRejected,
				Reason:      "invalid_json_line",
				Issues: []ingest.ValidationIssue{
					{Path: "$", Code: "invalid_json", Message: err.Error()},
				},
			})
			out.Summary.Rejected++
			continue
		}

		for k := range unk {
			unknown[k] = struct{}{}
		}

		res, valid, err := h.Processor.ProcessProduct(prod, h.EnabledChannels, lookup)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{
				"error":   "processing_failed",
				"message": err.Error(),
			})
			return
		}

		out.Products = append(out.Products, res)

		if !valid {
			out.Summary.Rejected++
			continue
		}

		out.Summary.Valid++

		switch res.Disposition {
		case domain.ProductDispositionUnchanged:
			out.Summary.Unchanged++
			if res.Hash != "" {
				if err := h.Store.UpsertProductHash(r.Context(), tenantID, res.ProductKey, res.Hash); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_state_failed",
						"message": err.Error(),
						"product": res.ProductKey,
					})
					return
				}

				// Persist normalized canonical product doc (JSON) for valid products.
				b, err := json.Marshal(prod)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_doc_failed",
						"message": "failed to serialize product doc",
						"product": res.ProductKey,
					})
					return
				}

				if err := h.Store.UpsertProductDoc(r.Context(), tenantID, res.ProductKey, state.ProductDocRecord{
					ProductJSON: b,
				}); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_doc_failed",
						"message": err.Error(),
						"product": res.ProductKey,
					})
					return
				}
			}

		case domain.ProductDispositionEnqueued:
			out.Summary.Enqueued++
			if res.Hash != "" {
				if err := h.Store.UpsertProductHash(r.Context(), tenantID, res.ProductKey, res.Hash); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_state_failed",
						"message": err.Error(),
						"product": res.ProductKey,
					})
					return
				}

				// Persist normalized canonical product doc (JSON) for valid products.
				b, err := json.Marshal(prod)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_doc_failed",
						"message": "failed to serialize product doc",
						"product": res.ProductKey,
					})
					return
				}

				if err := h.Store.UpsertProductDoc(r.Context(), tenantID, res.ProductKey, state.ProductDocRecord{
					ProductJSON: b,
				}); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_doc_failed",
						"message": err.Error(),
						"product": res.ProductKey,
					})
					return
				}
			}
		}
	}

	if err := sc.Err(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "read_failed",
			"message": err.Error(),
		})
		return
	}

	warnings := ingest.UnknownKeyWarning{UnknownKeys: ingest.SortedUnknownKeys(unknown)}

	pushTriggered := out.Summary.Enqueued > 0
	status := domain.RunStatusCompleted
	if !pushTriggered && out.Summary.Rejected == 0 {
		status = domain.RunStatusNoChangeDetected
	} else if pushTriggered {
		status = domain.RunStatusHasChanges
	}

	// Persist run + run_products (do NOT ignore errors)
	runRec := state.RunRecord{
		RunID:         runID,
		TenantID:      tenantID,
		FeedID:        nil,
		Status:        string(status),
		PushTriggered: pushTriggered,
		Received:      out.Summary.Received,
		Valid:         out.Summary.Valid,
		Rejected:      out.Summary.Rejected,
		Unchanged:     out.Summary.Unchanged,
		Enqueued:      out.Summary.Enqueued,
		Warnings:      warnings,
		CreatedAt:     time.Now().UTC(),
	}

	if err := h.Store.InsertRun(r.Context(), runRec); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "persist_run_failed",
			"message": err.Error(),
			"run_id":  runID,
		})
		return
	}

	if err := h.Store.InsertRunProducts(r.Context(), runID, out.Products); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "persist_run_products_failed",
			"message": err.Error(),
			"run_id":  runID,
		})
		return
	}

	resp := RunResponse{
		RunID:         runID,
		Status:        status,
		PushTriggered: pushTriggered,
		Warnings:      warnings,
		Result:        out,
	}

	writeJSON(w, http.StatusOK, resp)
}

func wrapMaybeGzip(body io.ReadCloser, contentEncoding string) (io.ReadCloser, error) {
	enc := strings.ToLower(strings.TrimSpace(contentEncoding))
	if enc == "" {
		return body, nil
	}

	if enc != "gzip" {
		return nil, fmt.Errorf("unsupported Content-Encoding: %s", enc)
	}

	gr, err := gzip.NewReader(body)
	if err != nil {
		return nil, err
	}

	return readCloserChain{Reader: gr, Closers: []io.Closer{gr, body}}, nil
}

type readCloserChain struct {
	io.Reader
	Closers []io.Closer
}

func (r readCloserChain) Close() error {
	var firstErr error
	for _, c := range r.Closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
