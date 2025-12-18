package ingest

import (
	"encoding/json"
	"net/http"

	"github.com/ETAnderson/conductor/internal/domain"
)

type DebugUpsertHandler struct {
	Processor Processor
	Store     *MemoryHashStore

	// For now we hardcode enabled channels to prove flow.
	// Later this comes from feed config in DB.
	EnabledChannels []string
}

func (h DebugUpsertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var products []domain.Product
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // strict for now in debug endpoint
	if err := dec.Decode(&products); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_json",
			"message": err.Error(),
		})
		return
	}

	out, err := h.Processor.ProcessProducts(products, h.EnabledChannels, h.Store.Get)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "processing_failed",
			"message": err.Error(),
		})
		return
	}

	// Update store for enqueued/valid products (simulates persisting current state)
	for _, pr := range out.Products {
		if pr.Disposition == domain.ProductDispositionEnqueued && pr.Hash != "" {
			h.Store.Set(pr.ProductKey, pr.Hash)
		}
		if pr.Disposition == domain.ProductDispositionUnchanged && pr.Hash != "" {
			// keep it set (idempotent)
			h.Store.Set(pr.ProductKey, pr.Hash)
		}
	}

	writeJSON(w, http.StatusOK, out)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
