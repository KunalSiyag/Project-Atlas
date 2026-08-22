package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CorpusOps extends retrieval with list/delete/status for knowledge corpus management.
type CorpusOps interface {
	ListDocuments(ctx context.Context) ([]Document, error)
	DeleteDocument(ctx context.Context, id string) error
	CorpusStatus(ctx context.Context) CorpusStatus
}

// CorpusStatus is a snapshot of the knowledge corpus.
type CorpusStatus struct {
	DocumentCount int    `json:"document_count"`
	ChunkCount    int    `json:"chunk_count"`
	Backend       string `json:"backend,omitempty"`
}

// SearchSLA holds SLA-oriented metrics for the last search call.
type SearchSLA struct {
	Query        string  `json:"query"`
	HitCount     int     `json:"hit_count"`
	LatencyMS    int64   `json:"latency_ms"`
	TopScore     float64 `json:"top_score,omitempty"`
	RecordedAt   string  `json:"recorded_at,omitempty"`
}

// MeteredRetriever wraps a Retriever and records search SLA metrics.
type MeteredRetriever struct {
	Inner      Retriever
	mu         sync.RWMutex
	lastSearch SearchSLA
}

// NewMeteredRetriever wraps inner (uses InProcessRetriever if nil).
func NewMeteredRetriever(inner Retriever) *MeteredRetriever {
	if inner == nil {
		inner = NewInProcessRetriever(FakeEmbedder{Dims: 8})
	}
	return &MeteredRetriever{Inner: inner}
}

func (m *MeteredRetriever) Ingest(ctx context.Context, doc Document) (Document, error) {
	return m.Inner.Ingest(ctx, doc)
}

func (m *MeteredRetriever) Search(ctx context.Context, query string, topK int) ([]Document, error) {
	start := time.Now()
	hits, err := m.Inner.Search(ctx, query, topK)
	elapsed := time.Since(start).Milliseconds()
	sla := SearchSLA{
		Query:      query,
		HitCount:   len(hits),
		LatencyMS:  elapsed,
		RecordedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	if len(hits) > 0 {
		sla.TopScore = hits[0].Score
	}
	m.mu.Lock()
	m.lastSearch = sla
	m.mu.Unlock()
	return hits, err
}

// LastSearchSLA returns metrics from the most recent Search call.
func (m *MeteredRetriever) LastSearchSLA() SearchSLA {
	if m == nil {
		return SearchSLA{}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.lastSearch
}

// ListDocuments lists corpus docs when the inner retriever supports CorpusOps
// (InProcessRetriever, QdrantRetriever via Fallback, etc.).
func (m *MeteredRetriever) ListDocuments(ctx context.Context) ([]Document, error) {
	if c, ok := m.Inner.(CorpusOps); ok {
		return c.ListDocuments(ctx)
	}
	return nil, fmt.Errorf("list documents not supported by backend")
}

// DeleteDocument deletes by id when supported.
func (m *MeteredRetriever) DeleteDocument(ctx context.Context, id string) error {
	if c, ok := m.Inner.(CorpusOps); ok {
		return c.DeleteDocument(ctx, id)
	}
	return fmt.Errorf("delete document not supported by backend")
}

// CorpusStatus returns counts when supported.
func (m *MeteredRetriever) CorpusStatus(ctx context.Context) CorpusStatus {
	if c, ok := m.Inner.(CorpusOps); ok {
		return c.CorpusStatus(ctx)
	}
	return CorpusStatus{Backend: "unknown"}
}

// ListDocuments returns parent-shaped documents currently in the corpus.
func (r *InProcessRetriever) ListDocuments(_ context.Context) ([]Document, error) {
	if r == nil {
		return nil, fmt.Errorf("retriever is nil")
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	// Collapse chunks: unique parent ids
	seen := map[string]Document{}
	order := []string{}
	for _, d := range r.docs {
		pid := d.ParentID
		if pid == "" {
			pid = d.ID
		}
		if _, ok := seen[pid]; !ok {
			out := d
			out.ID = pid
			out.ParentID = ""
			out.Vector = nil
			seen[pid] = out
			order = append(order, pid)
		}
	}
	out := make([]Document, 0, len(order))
	for _, id := range order {
		out = append(out, seen[id])
	}
	return out, nil
}

// DeleteDocument removes all chunks for a document id (or parent id).
func (r *InProcessRetriever) DeleteDocument(_ context.Context, id string) error {
	if r == nil {
		return fmt.Errorf("retriever is nil")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("document id required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	var kept []Document
	removed := 0
	for _, d := range r.docs {
		if d.ID == id || d.ParentID == id || strings.HasPrefix(d.ID, id+"#") {
			removed++
			continue
		}
		kept = append(kept, d)
	}
	if removed == 0 {
		return fmt.Errorf("document %q not found", id)
	}
	r.docs = kept
	return nil
}

// CorpusStatus returns document/chunk counts.
func (r *InProcessRetriever) CorpusStatus(_ context.Context) CorpusStatus {
	if r == nil {
		return CorpusStatus{Backend: "in_process"}
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	parents := map[string]bool{}
	for _, d := range r.docs {
		pid := d.ParentID
		if pid == "" {
			pid = d.ID
		}
		parents[pid] = true
	}
	return CorpusStatus{
		DocumentCount: len(parents),
		ChunkCount:    len(r.docs),
		Backend:       "in_process",
	}
}

func (h *Handler) listDocs(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Retriever == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "retriever not configured"})
		return
	}
	list, err := listCorpus(r.Context(), h.Service.Retriever)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": list})
}

func (h *Handler) deleteDoc(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Retriever == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "retriever not configured"})
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/knowledge/docs/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document id required"})
		return
	}
	if err := deleteCorpus(r.Context(), h.Service.Retriever, id); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deleted": id})
}

func (h *Handler) knowledgeStatus(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Retriever == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "retriever not configured"})
		return
	}
	st := corpusStatus(r.Context(), h.Service.Retriever)
	writeJSON(w, http.StatusOK, st)
}

func (h *Handler) searchKnowledge(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Retriever == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "retriever not configured"})
		return
	}
	var body struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	hits, err := h.Service.Retriever.Search(r.Context(), body.Query, body.TopK)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	var sla SearchSLA
	if m, ok := h.Service.Retriever.(*MeteredRetriever); ok {
		sla = m.LastSearchSLA()
	}
	writeJSON(w, http.StatusOK, map[string]any{"hits": hits, "sla": sla})
}

func listCorpus(ctx context.Context, r Retriever) ([]Document, error) {
	if m, ok := r.(*MeteredRetriever); ok {
		return m.ListDocuments(ctx)
	}
	if c, ok := r.(CorpusOps); ok {
		return c.ListDocuments(ctx)
	}
	return nil, fmt.Errorf("list documents not supported")
}

func deleteCorpus(ctx context.Context, r Retriever, id string) error {
	if m, ok := r.(*MeteredRetriever); ok {
		return m.DeleteDocument(ctx, id)
	}
	if c, ok := r.(CorpusOps); ok {
		return c.DeleteDocument(ctx, id)
	}
	return fmt.Errorf("delete not supported")
}

func corpusStatus(ctx context.Context, r Retriever) CorpusStatus {
	if m, ok := r.(*MeteredRetriever); ok {
		return m.CorpusStatus(ctx)
	}
	if c, ok := r.(CorpusOps); ok {
		return c.CorpusStatus(ctx)
	}
	return CorpusStatus{Backend: "unknown"}
}
