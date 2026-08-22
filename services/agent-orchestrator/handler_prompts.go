package orchestrator

import (
	"encoding/json"
	"net/http"
	"strings"
)

// workspaceFromRequest reads X-Workspace-ID (required for multi-tenant prompt admin).
func workspaceFromRequest(r *http.Request) (string, bool) {
	ws := strings.TrimSpace(r.Header.Get("X-Workspace-ID"))
	if ws == "" {
		return "", false
	}
	return ws, true
}

func (h *Handler) publishPrompt(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Prompts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "prompt store not configured"})
		return
	}
	ws, ok := workspaceFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "X-Workspace-ID required"})
		return
	}
	var entry PromptEntry
	if err := json.NewDecoder(r.Body).Decode(&entry); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	entry.WorkspaceID = ws
	if err := h.Service.Prompts.Publish(r.Context(), ws, entry); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	// Merge into process registry memory when present (no second store write).
	if h.Service.PromptReg != nil {
		h.Service.PromptReg.mu.Lock()
		h.Service.PromptReg.upsertLocked(entry)
		h.Service.PromptReg.mu.Unlock()
	}
	// Re-read so response matches durable store (workspace filled).
	if got, err := h.Service.Prompts.Get(r.Context(), ws, entry.ID, entry.Version); err == nil {
		entry = got
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (h *Handler) listPrompts(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Prompts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "prompt store not configured"})
		return
	}
	ws, ok := workspaceFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "X-Workspace-ID required"})
		return
	}
	list, err := h.Service.Prompts.List(r.Context(), ws)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if list == nil {
		list = []PromptEntry{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspace_id": ws, "prompts": list})
}

func (h *Handler) getPrompt(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Prompts == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "prompt store not configured"})
		return
	}
	ws, ok := workspaceFromRequest(r)
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "X-Workspace-ID required"})
		return
	}
	// Path: /v1/prompts/{id}
	id := strings.TrimPrefix(r.URL.Path, "/v1/prompts/")
	if id == "" || strings.Contains(id, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "prompt id required"})
		return
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	if version == "" {
		// Highest active for this workspace: list and resolve.
		list, err := h.Service.Prompts.List(r.Context(), ws)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		reg := NewPromptRegistry(nil)
		for _, e := range list {
			if e.ID == id {
				reg.upsertLocked(e)
			}
		}
		entry, err := reg.Resolve(id, "")
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		entry.WorkspaceID = ws
		writeJSON(w, http.StatusOK, entry)
		return
	}
	entry, err := h.Service.Prompts.Get(r.Context(), ws, id, version)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, entry)
}
