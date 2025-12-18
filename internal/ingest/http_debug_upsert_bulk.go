package ingest

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/ETAnderson/conductor/internal/domain"
)

type DebugBulkUpsertHandler struct {
	Processor       Processor
	Store           *MemoryHashStore
	EnabledChannels []string
}

func (h DebugBulkUpsertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runID, err := NewRunID()
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

	out := ProcessOutput{
		Summary:  ProcessSummary{Received: 0},
		Products: make([]ProductProcessResult, 0, 1024),
	}

	sc := bufio.NewScanner(reader)
	// Increase scanner buffer for large product lines
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 10*1024*1024) // 10MB max line

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}

		out.Summary.Received++

		prod, unk, err := ParseProductObjectAllowUnknown(line)
		if err != nil {
			// Treat line-level JSON parse errors as rejected product
			out.Products = append(out.Products, ProductProcessResult{
				ProductKey:  "",
				Disposition: domain.ProductDispositionRejected,
				Reason:      "invalid_json_line",
				Issues: []ValidationIssue{
					{Path: "$", Code: "invalid_json", Message: err.Error()},
				},
			})
			out.Summary.Rejected++
			continue
		}

		for k := range unk {
			unknown[k] = struct{}{}
		}

		res, valid, err := h.Processor.ProcessProduct(prod, h.EnabledChannels, h.Store.Get)
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
		case domain.ProductDispositionEnqueued:
			out.Summary.Enqueued++

			// Simulate persisting “current canonical” state
			if res.Hash != "" {
				h.Store.Set(res.ProductKey, res.Hash)
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

	warnings := UnknownKeyWarning{
		UnknownKeys: setToSortedSlice(unknown),
	}

	pushTriggered := out.Summary.Enqueued > 0

	status := domain.RunStatusCompleted
	if !pushTriggered && out.Summary.Rejected == 0 {
		status = domain.RunStatusNoChangeDetected
	} else if pushTriggered {
		status = domain.RunStatusHasChanges
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
	// close in order
	var firstErr error
	for _, c := range r.Closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
