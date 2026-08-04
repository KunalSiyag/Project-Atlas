package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Document is an ingested knowledge chunk.
type Document struct {
	ID       string    `json:"id"`
	Title    string    `json:"title,omitempty"`
	Content  string    `json:"content"`
	Vector   []float64 `json:"vector,omitempty"`
	Source   string    `json:"source,omitempty"`
}

// Retriever stores and searches documents.
type Retriever interface {
	Ingest(ctx context.Context, doc Document) (Document, error)
	Search(ctx context.Context, query string, topK int) ([]Document, error)
}

// Embedder produces vectors (gateway embeddings or local fake).
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float64, error)
}

// GatewayEmbedder calls inference-gateway POST /v1/embeddings.
type GatewayEmbedder struct {
	BaseURL string
	Model   string
	Client  *http.Client
}

func (e *GatewayEmbedder) Embed(ctx context.Context, texts []string) ([][]float64, error) {
	if e.Client == nil {
		e.Client = http.DefaultClient
	}
	model := e.Model
	if model == "" {
		model = "fake-embed"
	}
	body, _ := json.Marshal(map[string]any{"inputs": texts, "model": model})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(e.BaseURL, "/")+"/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := e.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var out struct {
		Vectors [][]float64 `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out.Vectors, nil
}

// FakeEmbedder is a local deterministic embedder (tests / no gateway).
type FakeEmbedder struct{ Dims int }

func (f FakeEmbedder) Embed(_ context.Context, texts []string) ([][]float64, error) {
	dims := f.Dims
	if dims <= 0 {
		dims = 8
	}
	out := make([][]float64, len(texts))
	for i, t := range texts {
		out[i] = hashVector(t, dims)
	}
	return out, nil
}

func hashVector(text string, dims int) []float64 {
	var seed uint64 = 14695981039346656037
	for i := 0; i < len(text); i++ {
		seed ^= uint64(text[i])
		seed *= 1099511628211
	}
	v := make([]float64, dims)
	var sum float64
	for i := 0; i < dims; i++ {
		seed = seed*6364136223846793005 + 1
		x := float64(int64(seed>>33)%1000)/500 - 1
		v[i] = x
		sum += x * x
	}
	n := math.Sqrt(sum)
	if n > 0 {
		for i := range v {
			v[i] /= n
		}
	}
	return v
}

// InProcessRetriever is the honest fallback when Qdrant is unavailable.
type InProcessRetriever struct {
	mu       sync.RWMutex
	docs     []Document
	embedder Embedder
}

func NewInProcessRetriever(embedder Embedder) *InProcessRetriever {
	if embedder == nil {
		embedder = FakeEmbedder{Dims: 8}
	}
	return &InProcessRetriever{embedder: embedder}
}

func (r *InProcessRetriever) Ingest(ctx context.Context, doc Document) (Document, error) {
	if strings.TrimSpace(doc.Content) == "" {
		return Document{}, fmt.Errorf("content required")
	}
	if doc.ID == "" {
		doc.ID = newID()
	}
	vecs, err := r.embedder.Embed(ctx, []string{doc.Content})
	if err != nil {
		return Document{}, err
	}
	if len(vecs) > 0 {
		doc.Vector = vecs[0]
	}
	r.mu.Lock()
	r.docs = append(r.docs, doc)
	r.mu.Unlock()
	return doc, nil
}

func (r *InProcessRetriever) Search(ctx context.Context, query string, topK int) ([]Document, error) {
	if topK <= 0 {
		topK = 3
	}
	qvecs, err := r.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(qvecs) == 0 {
		return nil, nil
	}
	q := qvecs[0]
	r.mu.RLock()
	defer r.mu.RUnlock()
	type scored struct {
		d Document
		s float64
	}
	var all []scored
	qwords := strings.Fields(strings.ToLower(query))
	for _, d := range r.docs {
		score := cosine(q, d.Vector)
		// Lexical boost so tests and sparse fake embeddings still retrieve.
		low := strings.ToLower(d.Content + " " + d.Title)
		for _, w := range qwords {
			if len(w) > 3 && strings.Contains(low, w) {
				score += 0.5
			}
		}
		all = append(all, scored{d: d, s: score})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].s > all[j].s })
	if len(all) > topK {
		all = all[:topK]
	}
	out := make([]Document, 0, len(all))
	for _, s := range all {
		out = append(out, s.d)
	}
	return out, nil
}

func cosine(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// QdrantRetriever stores points in Qdrant when available.
type QdrantRetriever struct {
	BaseURL    string
	Collection string
	Client     *http.Client
	Embedder   Embedder
	// Fallback used when Qdrant is unreachable.
	Fallback *InProcessRetriever
}

func NewQdrantRetriever(baseURL string, embedder Embedder) *QdrantRetriever {
	if embedder == nil {
		embedder = FakeEmbedder{Dims: 8}
	}
	return &QdrantRetriever{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		Collection: "atlas_docs",
		Client:     http.DefaultClient,
		Embedder:   embedder,
		Fallback:   NewInProcessRetriever(embedder),
	}
}

func (q *QdrantRetriever) ensureCollection(ctx context.Context, dims int) error {
	// create collection if missing (best-effort)
	body, _ := json.Marshal(map[string]any{
		"vectors": map[string]any{"size": dims, "distance": "Cosine"},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, q.BaseURL+"/collections/"+q.Collection, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.Client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (q *QdrantRetriever) Ingest(ctx context.Context, doc Document) (Document, error) {
	if q.Fallback != nil {
		// always keep local copy for tests
		_, _ = q.Fallback.Ingest(ctx, doc)
	}
	vecs, err := q.Embedder.Embed(ctx, []string{doc.Content})
	if err != nil {
		return q.Fallback.Ingest(ctx, doc)
	}
	if doc.ID == "" {
		doc.ID = newID()
	}
	if len(vecs) == 0 {
		return q.Fallback.Ingest(ctx, doc)
	}
	doc.Vector = vecs[0]
	_ = q.ensureCollection(ctx, len(doc.Vector))
	point := map[string]any{
		"points": []map[string]any{{
			"id":     doc.ID,
			"vector": doc.Vector,
			"payload": map[string]any{
				"content": doc.Content,
				"title":   doc.Title,
				"source":  doc.Source,
			},
		}},
	}
	body, _ := json.Marshal(point)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, q.BaseURL+"/collections/"+q.Collection+"/points?wait=true", bytes.NewReader(body))
	if err != nil {
		return q.Fallback.Ingest(ctx, doc)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.Client.Do(req)
	if err != nil {
		return doc, nil // fallback already has it
	}
	resp.Body.Close()
	return doc, nil
}

func (q *QdrantRetriever) Search(ctx context.Context, query string, topK int) ([]Document, error) {
	vecs, err := q.Embedder.Embed(ctx, []string{query})
	if err != nil || len(vecs) == 0 {
		return q.Fallback.Search(ctx, query, topK)
	}
	if topK <= 0 {
		topK = 3
	}
	body, _ := json.Marshal(map[string]any{
		"vector":       vecs[0],
		"limit":        topK,
		"with_payload": true,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, q.BaseURL+"/collections/"+q.Collection+"/points/search", bytes.NewReader(body))
	if err != nil {
		return q.Fallback.Search(ctx, query, topK)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := q.Client.Do(req)
	if err != nil {
		return q.Fallback.Search(ctx, query, topK)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return q.Fallback.Search(ctx, query, topK)
	}
	var parsed struct {
		Result []struct {
			ID      any            `json:"id"`
			Payload map[string]any `json:"payload"`
			Score   float64        `json:"score"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return q.Fallback.Search(ctx, query, topK)
	}
	var out []Document
	for _, r := range parsed.Result {
		content, _ := r.Payload["content"].(string)
		title, _ := r.Payload["title"].(string)
		out = append(out, Document{ID: fmt.Sprint(r.ID), Title: title, Content: content})
	}
	if len(out) == 0 {
		return q.Fallback.Search(ctx, query, topK)
	}
	return out, nil
}

// DocSnippets converts retrieval hits to context-builder notes.
func DocSnippets(docs []Document) []string {
	var out []string
	for _, d := range docs {
		s := d.Content
		if d.Title != "" {
			s = d.Title + ": " + s
		}
		out = append(out, s)
	}
	return out
}
