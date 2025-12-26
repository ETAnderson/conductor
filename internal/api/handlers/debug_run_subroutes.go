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
	// If request targets channels subroutes, send to channels handler.
	// Otherwise, send to existing runs handler (list/detail).
	if strings.Contains(r.URL.Path, "/channels") {
		DebugRunChannelsHandler{Store: h.Store}.ServeHTTP(w, r)
		return
	}

	// Fall back to existing handler (it already supports list+detail)
	DebugRunsHandler{Store: h.Store}.ServeHTTP(w, r)
}
