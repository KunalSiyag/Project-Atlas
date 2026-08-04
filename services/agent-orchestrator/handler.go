package orchestrator

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Handler exposes HTTP control surface for the orchestrator.
type Handler struct {
	Service *Service
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Optional workspace isolation: require X-Workspace-ID when ATLAS_REQUIRE_WORKSPACE=1 is set by main via middleware — handled here lightly.
	if r.Method == http.MethodGet && r.URL.Path == "/healthz" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "agent-orchestrator"})
		return
	}
	if r.Method == http.MethodGet && r.URL.Path == "/readyz" {
		if h.Service == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready", "service": "agent-orchestrator"})
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/v1/runs" {
		h.createRun(w, r)
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/v1/knowledge/ingest" {
		h.ingest(w, r)
		return
	}
	if r.Method == http.MethodPost && r.URL.Path == "/v1/memory" {
		h.putMemory(w, r)
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/runs/") && strings.HasSuffix(r.URL.Path, "/start") {
		h.startRun(w, r)
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/runs/") && strings.HasSuffix(r.URL.Path, "/cancel") {
		h.cancelRun(w, r)
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/runs/") && strings.HasSuffix(r.URL.Path, "/approve") {
		h.approveRun(w, r)
		return
	}
	if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v1/runs/") && strings.HasSuffix(r.URL.Path, "/reject") {
		h.rejectRun(w, r)
		return
	}
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/runs/") {
		h.getRun(w, r)
		return
	}
	http.NotFound(w, r)
}

func (h *Handler) createRun(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "not_ready"})
		return
	}
	var body struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		body.Goal = ""
	}
	run, err := h.Service.CreateRun(r.Context(), body.Goal)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, run)
}

func (h *Handler) getRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/v1/runs/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "run id required"})
		return
	}
	run, err := h.Service.GetRun(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) startRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/runs/"), "/start")
	run, err := h.Service.StartRun(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) cancelRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/runs/"), "/cancel")
	run, err := h.Service.CancelRun(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) approveRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/runs/"), "/approve")
	run, err := h.Service.ApproveRun(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) rejectRun(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v1/runs/"), "/reject")
	var body struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	run, err := h.Service.RejectRun(r.Context(), id, body.Reason)
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, run)
}

func (h *Handler) ingest(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "not_ready"})
		return
	}
	var doc Document
	if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	out, err := h.Service.IngestDocument(r.Context(), doc)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (h *Handler) putMemory(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "not_ready"})
		return
	}
	var fact MemoryFact
	if err := json.NewDecoder(r.Body).Decode(&fact); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if err := h.Service.PutMemory(r.Context(), fact); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, fact)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
