package orchestrator

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCorpusListDeleteAndSearchSLA(t *testing.T) {
	inner := NewInProcessRetriever(FakeEmbedder{Dims: 8})
	inner.DisableChunking = true
	svc := NewService(ServiceConfig{
		Store:     NewMemoryStore(),
		Retriever: inner,
	})
	// NewService wraps with MeteredRetriever
	m, ok := svc.Retriever.(*MeteredRetriever)
	if !ok {
		t.Fatalf("want MeteredRetriever, got %T", svc.Retriever)
	}
	ctx := context.Background()
	_, err := svc.IngestDocument(ctx, Document{ID: "doc-1", Title: "Atlas", Content: "vector search knowledge chunks"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.IngestDocument(ctx, Document{ID: "doc-2", Title: "Noise", Content: "unrelated pasta recipe"})
	if err != nil {
		t.Fatal(err)
	}
	list, err := m.ListDocuments(ctx)
	if err != nil || len(list) != 2 {
		t.Fatalf("list=%v err=%v", list, err)
	}
	hits, err := m.Search(ctx, "vector search knowledge", 2)
	if err != nil || len(hits) == 0 {
		t.Fatalf("search %v %v", hits, err)
	}
	sla := m.LastSearchSLA()
	if sla.HitCount != len(hits) || sla.LatencyMS < 0 || sla.Query == "" {
		t.Fatalf("sla=%+v", sla)
	}
	if err := m.DeleteDocument(ctx, "doc-2"); err != nil {
		t.Fatal(err)
	}
	st := m.CorpusStatus(ctx)
	if st.DocumentCount != 1 {
		t.Fatalf("status=%+v", st)
	}

	// HTTP path
	h := &Handler{Service: svc}
	req := httptest.NewRequest(http.MethodGet, "/v1/knowledge/docs", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("%d %s", rr.Code, rr.Body.String())
	}
	searchBody := `{"query":"vector knowledge","top_k":3}`
	reqS := httptest.NewRequest(http.MethodPost, "/v1/knowledge/search", bytes.NewBufferString(searchBody))
	reqS.Header.Set("Content-Type", "application/json")
	rrS := httptest.NewRecorder()
	h.ServeHTTP(rrS, reqS)
	if rrS.Code != http.StatusOK {
		t.Fatalf("search http %d %s", rrS.Code, rrS.Body.String())
	}
	var searchOut struct {
		Hits []Document `json:"hits"`
		SLA  SearchSLA  `json:"sla"`
	}
	if err := json.Unmarshal(rrS.Body.Bytes(), &searchOut); err != nil {
		t.Fatal(err)
	}
	if searchOut.SLA.HitCount != len(searchOut.Hits) {
		t.Fatalf("sla mismatch %+v", searchOut)
	}
	reqD := httptest.NewRequest(http.MethodDelete, "/v1/knowledge/docs/doc-1", nil)
	rrD := httptest.NewRecorder()
	h.ServeHTTP(rrD, reqD)
	if rrD.Code != http.StatusOK {
		t.Fatalf("delete %d %s", rrD.Code, rrD.Body.String())
	}
	reqSt := httptest.NewRequest(http.MethodGet, "/v1/knowledge/status", nil)
	rrSt := httptest.NewRecorder()
	h.ServeHTTP(rrSt, reqSt)
	var status CorpusStatus
	_ = json.Unmarshal(rrSt.Body.Bytes(), &status)
	if status.DocumentCount != 0 {
		t.Fatalf("status after delete %+v", status)
	}
}

// TestCorpusOpsViaQdrantRetrieverShippedPath matches main wiring:
// NewQdrantRetriever (unreachable Qdrant) dual-writes Fallback; NewService meters it;
// list/delete/status/search must work through Handler HTTP on that path.
func TestCorpusOpsViaQdrantRetrieverShippedPath(t *testing.T) {
	// Unreachable Qdrant URL — Fallback still holds dual-written docs (same as main).
	qdrant := NewQdrantRetriever("http://127.0.0.1:1", FakeEmbedder{Dims: 8})
	qdrant.Fallback.DisableChunking = true
	svc := NewService(ServiceConfig{
		Store:     NewMemoryStore(),
		Retriever: qdrant, // same type main uses
	})
	m, ok := svc.Retriever.(*MeteredRetriever)
	if !ok {
		t.Fatalf("want MeteredRetriever wrap, got %T", svc.Retriever)
	}
	if _, ok := m.Inner.(*QdrantRetriever); !ok {
		t.Fatalf("want inner *QdrantRetriever (main path), got %T", m.Inner)
	}

	ctx := context.Background()
	if _, err := svc.IngestDocument(ctx, Document{
		ID: "qd-1", Title: "Atlas RAG", Content: "vector search knowledge chunks for corpus ops",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.IngestDocument(ctx, Document{
		ID: "qd-2", Title: "Noise", Content: "unrelated pasta recipe",
	}); err != nil {
		t.Fatal(err)
	}

	h := &Handler{Service: svc}
	// List via shipped HTTP
	req := httptest.NewRequest(http.MethodGet, "/v1/knowledge/docs", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("list on Qdrant path: %d %s", rr.Code, rr.Body.String())
	}
	var listOut struct {
		Documents []Document `json:"documents"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &listOut); err != nil {
		t.Fatal(err)
	}
	if len(listOut.Documents) != 2 {
		t.Fatalf("list docs=%d body=%s", len(listOut.Documents), rr.Body.String())
	}

	// Status must not be backend=unknown with zero counts after ingest
	reqSt := httptest.NewRequest(http.MethodGet, "/v1/knowledge/status", nil)
	rrSt := httptest.NewRecorder()
	h.ServeHTTP(rrSt, reqSt)
	if rrSt.Code != http.StatusOK {
		t.Fatalf("status %d %s", rrSt.Code, rrSt.Body.String())
	}
	var st CorpusStatus
	if err := json.Unmarshal(rrSt.Body.Bytes(), &st); err != nil {
		t.Fatal(err)
	}
	if st.DocumentCount != 2 {
		t.Fatalf("status after ingest: %+v", st)
	}
	if st.Backend == "unknown" {
		t.Fatalf("backend unknown on Qdrant path: %+v", st)
	}

	// Search SLA through metered wrapper
	reqS := httptest.NewRequest(http.MethodPost, "/v1/knowledge/search",
		bytes.NewBufferString(`{"query":"vector search knowledge","top_k":2}`))
	reqS.Header.Set("Content-Type", "application/json")
	rrS := httptest.NewRecorder()
	h.ServeHTTP(rrS, reqS)
	if rrS.Code != http.StatusOK {
		t.Fatalf("search %d %s", rrS.Code, rrS.Body.String())
	}
	var searchOut struct {
		Hits []Document `json:"hits"`
		SLA  SearchSLA  `json:"sla"`
	}
	if err := json.Unmarshal(rrS.Body.Bytes(), &searchOut); err != nil {
		t.Fatal(err)
	}
	if len(searchOut.Hits) == 0 || searchOut.SLA.HitCount != len(searchOut.Hits) {
		t.Fatalf("search sla %+v", searchOut)
	}

	// Delete one doc
	reqD := httptest.NewRequest(http.MethodDelete, "/v1/knowledge/docs/qd-2", nil)
	rrD := httptest.NewRecorder()
	h.ServeHTTP(rrD, reqD)
	if rrD.Code != http.StatusOK {
		t.Fatalf("delete %d %s", rrD.Code, rrD.Body.String())
	}
	reqSt2 := httptest.NewRequest(http.MethodGet, "/v1/knowledge/status", nil)
	rrSt2 := httptest.NewRecorder()
	h.ServeHTTP(rrSt2, reqSt2)
	var st2 CorpusStatus
	_ = json.Unmarshal(rrSt2.Body.Bytes(), &st2)
	if st2.DocumentCount != 1 {
		t.Fatalf("status after delete: %+v", st2)
	}
}
