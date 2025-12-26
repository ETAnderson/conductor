package handlers

import (
	"net/http"
	"strings"

	"github.com/ETAnderson/conductor/internal/state"
)

type DebugRunSubroutesHandler struct {
	Store state.Store
}

func (h DebugRunSubroutesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// All routes here are GET-only today
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Expected base: /v1/debug/runs/{...}
	path := strings.TrimPrefix(r.URL.Path, "/v1/debug/runs/")
	path = strings.Trim(path, "/")

	// If there's no subpath after /runs/, treat as invalid (or delegate)
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error":   "invalid_run_id",
			"message": "run_id missing or invalid",
		})
		return
	}

	// Route to channels handler if the URL contains "/channels"
	// (it will validate run_id + tenant ownership internally)
	if strings.Contains(path, "/channels") {
		DebugRunChannelsHandler{Store: h.Store}.ServeHTTP(w, r)
		return
	}

	// Otherwise, route to runs handler (/v1/debug/runs/{run_id})
	DebugRunsHandler{Store: h.Store}.ServeHTTP(w, r)
}
