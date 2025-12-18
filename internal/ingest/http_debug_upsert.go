package ingest

import (
	"encoding/json"
	"io"
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

type RunResponse struct {
	RunID         string            `json:"run_id"`
	Status        domain.RunStatus  `json:"status"`
	PushTriggered bool              `json:"push_triggered"`
	Warnings      UnknownKeyWarning `json:"warnings,omitempty"`
	Result        ProcessOutput     `json:"result"`
}

func (h DebugUpsertHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "read_failed",
			"message": err.Error(),
		})
		return
	}

	parsed, err := ParseProductsAllowUnknown(bodyBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_json",
			"message": err.Error(),
		})
		return
	}

	out, err := h.Processor.ProcessProducts(parsed.Products, h.EnabledChannels, h.Store.Get)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "processing_failed",
			"message": err.Error(),
		})
		return
	}

	// Update store for valid products (simulates persisting current state)
	for _, pr := range out.Products {
		if pr.Hash == "" {
			continue
		}

		switch pr.Disposition {
		case domain.ProductDispositionEnqueued, domain.ProductDispositionUnchanged:
			h.Store.Set(pr.ProductKey, pr.Hash)
		}
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
