package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/state"
)

type DebugRunsHandler struct {
	Store state.Store
}

func (h DebugRunsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantctx.TenantID(r.Context())

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	runs, err := h.Store.ListRuns(r.Context(), tenantID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "list_runs_failed",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": runs,
	})
}

type DebugRunDetailHandler struct {
	Store state.Store
}

func (h DebugRunDetailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := tenantctx.TenantID(r.Context())

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Expected path: /v1/debug/runs/{run_id}
	runID := strings.TrimPrefix(r.URL.Path, "/v1/debug/runs/")
	runID = strings.TrimSpace(runID)
	if runID == "" || strings.Contains(runID, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_run_id",
			"message": "run_id missing or invalid",
		})
		return
	}

	run, ok, err := h.Store.GetRun(r.Context(), tenantID, runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "get_run_failed",
			"message": err.Error(),
		})
		return
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"error":   "not_found",
			"message": "run not found",
		})
		return
	}

	limit := 500
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	if limit <= 0 {
		limit = 500
	}
	if limit > 2000 {
		limit = 2000
	}

	products, err := h.Store.ListRunProducts(r.Context(), runID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "list_run_products_failed",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"run":      run,
		"products": products,
	})
}
