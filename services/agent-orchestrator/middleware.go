package orchestrator

import (
	"net/http"
	"strings"
)

// RequireWorkspace rejects non-probe requests missing X-Workspace-ID.
// Used when ATLAS_REQUIRE_WORKSPACE=1 wraps the orchestrator handler.
// /healthz and /readyz always pass through without the header.
func RequireWorkspace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			next.ServeHTTP(w, r)
			return
		}
		if strings.TrimSpace(r.Header.Get("X-Workspace-ID")) == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"X-Workspace-ID required"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
