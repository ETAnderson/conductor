package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "read_failed",
			"message": err.Error(),
		})
		return
	}

	parsed, err := ingest.ParseProductsAllowUnknown(bodyBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_json",
			"message": err.Error(),
		})
		return
	}

	lookup := func(productKey string) (string, bool, error) {
		return h.Store.GetProductHash(r.Context(), tenantID, productKey)
	}

	out, err := h.Processor.ProcessProducts(parsed.Products, h.EnabledChannels, lookup)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "processing_failed",
			"message": err.Error(),
		})
		return
	}

	// Build a lookup from product_key -> normalized payload so we can persist canonical JSON docs
	// for valid products (enqueued/unchanged) without storing rejected payloads.
	payloadByKey := make(map[string]any, len(parsed.Products))
	for _, p := range parsed.Products {
		// Assumes parsed.Products items have ProductKey
		payloadByKey[p.ProductKey] = p
	}

	// Persist canonical product state for relevant dispositions
	for _, pr := range out.Products {
		if pr.Hash == "" {
			continue
		}

		switch pr.Disposition {
		case domain.ProductDispositionEnqueued, domain.ProductDispositionUnchanged:
			if err := h.Store.UpsertProductHash(r.Context(), tenantID, pr.ProductKey, pr.Hash); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]any{
					"error":   "persist_product_state_failed",
					"message": err.Error(),
					"product": pr.ProductKey,
				})
				return
			}

			// Persist normalized canonical product doc (JSON) for valid products.
			// This enables channel mappers (google v0) to read full product fields.
			if payload, ok := payloadByKey[pr.ProductKey]; ok {
				b, err := json.Marshal(payload)
				if err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_doc_failed",
						"message": "failed to serialize product doc",
						"product": pr.ProductKey,
					})
					return
				}

				if err := h.Store.UpsertProductDoc(r.Context(), tenantID, pr.ProductKey, state.ProductDocRecord{
					ProductJSON: b,
				}); err != nil {
					writeJSON(w, http.StatusInternalServerError, map[string]any{
						"error":   "persist_product_doc_failed",
						"message": err.Error(),
						"product": pr.ProductKey,
					})
					return
				}
			}
		}
	}

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
		Warnings:      parsed.Warnings,
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
		Warnings:      parsed.Warnings,
		Result:        out,
	}

	// Always encode response last so we only return success if persistence succeeded
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
