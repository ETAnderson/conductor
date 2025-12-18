package handlers

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

type DebugBulkUpsertHandler struct {
	Processor ingest.Processor
	Store     state.Store

	TenantID uint64

	EnabledChannels []string
}

func (h DebugBulkUpsertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runID, err := ingest.NewRunID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "run_id_failed", "message": err.Error()})
		return
	}

	reader, err := wrapMaybeGzip(r.Body, r.Header.Get("Content-Encoding"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_encoding", "message": err.Error()})
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
		return h.Store.GetProductHash(r.Context(), h.TenantID, productKey)
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
			writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "processing_failed", "message": err.Error()})
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
				_ = h.Store.UpsertProductHash(r.Context(), h.TenantID, res.ProductKey, res.Hash)
			}
		case domain.ProductDispositionEnqueued:
			out.Summary.Enqueued++
			if res.Hash != "" {
				_ = h.Store.UpsertProductHash(r.Context(), h.TenantID, res.ProductKey, res.Hash)
			}
		}
	}

	if err := sc.Err(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "read_failed", "message": err.Error()})
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

	_ = h.Store.InsertRun(r.Context(), state.RunRecord{
		RunID:         runID,
		TenantID:      h.TenantID,
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
	})
	_ = h.Store.InsertRunProducts(r.Context(), runID, out.Products)

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
		return nil, io.ErrUnexpectedEOF
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
