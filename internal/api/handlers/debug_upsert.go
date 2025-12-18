package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ETAnderson/conductor/internal/domain"
	"github.com/ETAnderson/conductor/internal/ingest"
	"github.com/ETAnderson/conductor/internal/state"
)

type DebugUpsertHandler struct {
	Processor ingest.Processor
	Store     state.Store

	TenantID uint64

	EnabledChannels []string
}

type RunResponse struct {
	RunID         string                   `json:"run_id"`
	Status        domain.RunStatus         `json:"status"`
	PushTriggered bool                     `json:"push_triggered"`
	Warnings      ingest.UnknownKeyWarning `json:"warnings,omitempty"`
	Result        ingest.ProcessOutput     `json:"result"`
}

func (h DebugUpsertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	runID, err := ingest.NewRunID()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "run_id_failed", "message": err.Error()})
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "read_failed", "message": err.Error()})
		return
	}

	parsed, err := ingest.ParseProductsAllowUnknown(bodyBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}

	lookup := func(productKey string) (string, bool, error) {
		return h.Store.GetProductHash(r.Context(), h.TenantID, productKey)
	}

	out, err := h.Processor.ProcessProducts(parsed.Products, h.EnabledChannels, lookup)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "processing_failed", "message": err.Error()})
		return
	}

	// Persist canonical state for valid products
	for _, pr := range out.Products {
		if pr.Hash == "" {
			continue
		}
		switch pr.Disposition {
		case domain.ProductDispositionEnqueued, domain.ProductDispositionUnchanged:
			_ = h.Store.UpsertProductHash(r.Context(), h.TenantID, pr.ProductKey, pr.Hash)
		}
	}

	pushTriggered := out.Summary.Enqueued > 0

	status := domain.RunStatusCompleted
	if !pushTriggered && out.Summary.Rejected == 0 {
		status = domain.RunStatusNoChangeDetected
	} else if pushTriggered {
		status = domain.RunStatusHasChanges
	}

	// Persist run + run_products (for reporting)
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
		Warnings:      parsed.Warnings,
		CreatedAt:     time.Now().UTC(),
	})
	_ = h.Store.InsertRunProducts(r.Context(), runID, out.Products)

	resp := RunResponse{
		RunID:         runID,
		Status:        status,
		PushTriggered: pushTriggered,
		Warnings:      parsed.Warnings,
		Result:        out,
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
