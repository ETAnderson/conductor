package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ETAnderson/conductor/internal/api/tenantctx"
	"github.com/ETAnderson/conductor/internal/state"
)

type DebugRunChannelsHandler struct {
	Store state.Store
}

func (h DebugRunChannelsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	tenantID := tenantctx.TenantID(r.Context())

	// Expect: /v1/debug/runs/{runID}/channels
	path := strings.TrimPrefix(r.URL.Path, "/v1/debug/runs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[1] != "channels" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "invalid path"})
		return
	}
	runID := parts[0]

	_, ok, err := h.Store.GetRun(r.Context(), tenantID, runID)
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

	items, err := h.Store.ListRunChannelResults(r.Context(), tenantID, runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "list_failed",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

type DebugRunChannelItemsHandler struct {
	Store state.Store
}

func (h DebugRunChannelItemsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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

	// Expect: /v1/debug/runs/{runID}/channels/{channel}
	path := strings.TrimPrefix(r.URL.Path, "/v1/debug/runs/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 3 || parts[1] != "channels" {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not_found", "message": "invalid path"})
		return
	}
	runID := parts[0]
	channel := parts[2]

	limit := 1000
	if s := r.URL.Query().Get("limit"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			limit = v
		}
	}

	items, err := h.Store.ListRunChannelItems(r.Context(), runID, channel, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]any{
			"error":   "list_failed",
			"message": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"run_id":  runID,
		"channel": channel,
		"items":   items,
	})
}
