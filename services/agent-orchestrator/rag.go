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

// Document is an ingested knowledge chunk (or parent metadata after multi-chunk ingest).
type Document struct {
	ID       string    `json:"id"`
	Title    string    `json:"title,omitempty"`
	Content  string    `json:"content"`
	Vector   []float64 `json:"vector,omitempty"`
	Source   string    `json:"source,omitempty"`
	// Score is set on Search results (cosine + lexical boosts); omitted on ingest.
	Score float64 `json:"score,omitempty"`
	// ChunkIndex is 0-based when a parent document was split on ingest.
	ChunkIndex int `json:"chunk_index,omitempty"`
	// ParentID links a chunk back to the original document id (empty for whole docs).
	ParentID string `json:"parent_id,omitempty"`
	// Metadata is free-form source metadata returned on search hits.
	Metadata map[string]string `json:"metadata,omitempty"`
}

// Retriever stores and searches documents.
type Retriever interface {
	Ingest(ctx context.Context, doc Document) (Document, error)
	Search(ctx context.Context, query string, topK int) ([]Document, error)
}

// Default chunk sizes for production-style multi-chunk ingest.
const (
	defaultChunkRunes = 280
	defaultChunkOverlap = 40
)

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
// It chunks long content on ingest and ranks with cosine + BM25-lite lexical boosts.
type InProcessRetriever struct {
	mu          sync.RWMutex
	docs        []Document
	embedder    Embedder
	ChunkRunes  int
	ChunkOverlap int
	// DisableChunking forces single-vector ingest (tests that assert whole-doc IDs).
	DisableChunking bool
}

func NewInProcessRetriever(embedder Embedder) *InProcessRetriever {
	if embedder == nil {
		embedder = FakeEmbedder{Dims: 8}
	}
	return &InProcessRetriever{
		embedder:     embedder,
		ChunkRunes:   defaultChunkRunes,
		ChunkOverlap: defaultChunkOverlap,
	}
}

func (r *InProcessRetriever) Ingest(ctx context.Context, doc Document) (Document, error) {
	if strings.TrimSpace(doc.Content) == "" {
		return Document{}, fmt.Errorf("content required")
	}
	if doc.ID == "" {
		doc.ID = newID()
	}
	if doc.Metadata == nil {
		doc.Metadata = map[string]string{}
	}
	if doc.Source != "" {
		doc.Metadata["source"] = doc.Source
	}
	if doc.Title != "" {
		doc.Metadata["title"] = doc.Title
	}

	chunks := []string{doc.Content}
	if !r.DisableChunking {
		size := r.ChunkRunes
		if size <= 0 {
			size = defaultChunkRunes
		}
		overlap := r.ChunkOverlap
		if overlap < 0 {
			overlap = 0
		}
		chunks = ChunkText(doc.Content, size, overlap)
	}
	if len(chunks) == 0 {
		return Document{}, fmt.Errorf("content required")
	}

	// Embed all chunks in one batch when possible.
	vecs, err := r.embedder.Embed(ctx, chunks)
	if err != nil {
		return Document{}, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	parentID := doc.ID
	for i, chunk := range chunks {
		d := Document{
			ID:         parentID,
			Title:      doc.Title,
			Content:    chunk,
			Source:     doc.Source,
			ChunkIndex: i,
			Metadata:   copyStringMap(doc.Metadata),
		}
		if len(chunks) > 1 {
			d.ID = fmt.Sprintf("%s#c%d", parentID, i)
			d.ParentID = parentID
			if d.Metadata == nil {
				d.Metadata = map[string]string{}
			}
			d.Metadata["parent_id"] = parentID
			d.Metadata["chunk_index"] = fmt.Sprintf("%d", i)
		}
		if i < len(vecs) {
			d.Vector = vecs[i]
		}
		r.docs = append(r.docs, d)
	}
	// Return parent-shaped document (first chunk vector for convenience).
	out := doc
	if len(vecs) > 0 {
		out.Vector = vecs[0]
	}
	out.Metadata = copyStringMap(doc.Metadata)
	if len(chunks) > 1 {
		out.Metadata["chunks"] = fmt.Sprintf("%d", len(chunks))
	}
	return out, nil
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
	qwords := tokenize(query)
	for _, d := range r.docs {
		score := rankScore(q, d, qwords)
		hit := d
		hit.Score = score
		// Clear large vectors from search results for API cleanliness.
		hit.Vector = nil
		all = append(all, scored{d: hit, s: score})
	}
	sort.SliceStable(all, func(i, j int) bool {
		if all[i].s == all[j].s {
			return all[i].d.ID < all[j].d.ID
		}
		return all[i].s > all[j].s
	})
	if len(all) > topK {
		all = all[:topK]
	}
	out := make([]Document, 0, len(all))
	for _, s := range all {
		out = append(out, s.d)
	}
	return out, nil
}

// rankScore combines cosine similarity with lexical term frequency boosts.
func rankScore(queryVec []float64, d Document, qwords []string) float64 {
	score := cosine(queryVec, d.Vector)
	low := strings.ToLower(d.Content + " " + d.Title + " " + d.Source)
	for _, w := range qwords {
		if len(w) < 3 {
			continue
		}
		if strings.Contains(low, w) {
			// Longer query terms and title hits weigh more.
			boost := 0.35 + 0.05*float64(minInt(len(w), 12))
			if d.Title != "" && strings.Contains(strings.ToLower(d.Title), w) {
				boost += 0.4
			}
			score += boost
		}
	}
	// Exact phrase boost (query as a whole substring).
	if q := strings.TrimSpace(strings.ToLower(strings.Join(qwords, " "))); len(q) > 6 && strings.Contains(low, q) {
		score += 1.0
	}
	return score
}

func tokenize(s string) []string {
	fields := strings.Fields(strings.ToLower(s))
	var out []string
	for _, f := range fields {
		f = strings.Trim(f, ".,;:!?\"'()[]{}")
		if f != "" {
			out = append(out, f)
		}
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ChunkText splits text into overlapping rune windows for multi-chunk ingest.
func ChunkText(text string, size, overlap int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if size <= 0 {
		size = defaultChunkRunes
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 4
	}
	runes := []rune(text)
	if len(runes) <= size {
		return []string{text}
	}
	var chunks []string
	step := size - overlap
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[start:end]))
		if end == len(runes) {
			break
		}
	}
	return chunks
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
		source, _ := r.Payload["source"].(string)
		out = append(out, Document{
			ID: fmt.Sprint(r.ID), Title: title, Content: content, Source: source, Score: r.Score,
			Metadata: map[string]string{"source": source, "title": title},
		})
	}
	if len(out) == 0 {
		return q.Fallback.Search(ctx, query, topK)
	}
	return out, nil
}

// ListDocuments implements CorpusOps via the dual-write Fallback (always populated on Ingest).
// This is the shipped knowledge path used by main (NewQdrantRetriever).
func (q *QdrantRetriever) ListDocuments(ctx context.Context) ([]Document, error) {
	if q == nil || q.Fallback == nil {
		return nil, fmt.Errorf("qdrant fallback not configured")
	}
	return q.Fallback.ListDocuments(ctx)
}

// DeleteDocument removes the doc from Fallback and best-effort from Qdrant.
func (q *QdrantRetriever) DeleteDocument(ctx context.Context, id string) error {
	if q == nil {
		return fmt.Errorf("qdrant retriever is nil")
	}
	var fbErr error
	if q.Fallback != nil {
		fbErr = q.Fallback.DeleteDocument(ctx, id)
	}
	// Best-effort remote delete (points/delete). Unreachable Qdrant is fine — Fallback is source of truth for local ops.
	if q.BaseURL != "" && q.Client != nil {
		body, _ := json.Marshal(map[string]any{
			"points": []string{id},
		})
		req, err := http.NewRequestWithContext(ctx, http.MethodPost,
			q.BaseURL+"/collections/"+q.Collection+"/points/delete?wait=true", bytes.NewReader(body))
		if err == nil {
			req.Header.Set("Content-Type", "application/json")
			if resp, err := q.Client.Do(req); err == nil {
				resp.Body.Close()
			}
		}
	}
	if fbErr != nil {
		return fbErr
	}
	if q.Fallback == nil {
		return fmt.Errorf("delete document not supported")
	}
	return nil
}

// CorpusStatus reports counts from Fallback (dual-write path) with backend label.
func (q *QdrantRetriever) CorpusStatus(ctx context.Context) CorpusStatus {
	if q == nil || q.Fallback == nil {
		return CorpusStatus{Backend: "qdrant"}
	}
	st := q.Fallback.CorpusStatus(ctx)
	st.Backend = "qdrant+fallback"
	return st
}

// DocSnippets converts retrieval hits to context-builder notes (includes source/score).
func DocSnippets(docs []Document) []string {
	var out []string
	for _, d := range docs {
		s := d.Content
		if d.Title != "" {
			s = d.Title + ": " + s
		}
		if d.Source != "" {
			s = s + " [source=" + d.Source + "]"
		}
		if d.Score != 0 {
			s = fmt.Sprintf("%s [score=%.3f]", s, d.Score)
		}
		out = append(out, s)
	}
	return out
}
