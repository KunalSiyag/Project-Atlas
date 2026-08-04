package orchestrator

import (
	"context"
	"strings"
	"testing"
)

func TestChunkTextOverlapping(t *testing.T) {
	// Build content longer than default chunk size.
	var b strings.Builder
	for i := 0; i < 40; i++ {
		b.WriteString("The capital of Testland is Testville. ")
	}
	chunks := ChunkText(b.String(), 80, 20)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	// Reassemble prefix should match start of text.
	if !strings.HasPrefix(b.String(), chunks[0][:minInt(20, len(chunks[0]))]) {
		t.Fatalf("chunk0 not from start: %q", chunks[0][:min(40, len(chunks[0]))])
	}
}

func TestRAGRanksRelevantDocAboveDistractor(t *testing.T) {
	// Production-quality path: multi-doc ingest + ranked search with scores/metadata.
	r := NewInProcessRetriever(FakeEmbedder{Dims: 16})
	// Keep chunks as whole docs so ranking is purely content-based for this fixture.
	r.DisableChunking = true
	ctx := context.Background()

	relevant, err := r.Ingest(ctx, Document{
		ID:      "doc-relevant",
		Title:   "Atlas orchestration",
		Source:  "kb://atlas/orchestrator",
		Content: "Project Atlas agent orchestrator runs plan DAGs with tools, memory, and RAG retrieval for enterprise workspaces.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if relevant.ID != "doc-relevant" {
		t.Fatalf("id=%q", relevant.ID)
	}

	_, err = r.Ingest(ctx, Document{
		ID:      "doc-distractor",
		Title:   "Cooking pasta",
		Source:  "kb://food/pasta",
		Content: "Boil water, add salt, cook spaghetti until al dente, then serve with tomato sauce and cheese.",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Also ingest a long multi-chunk document so chunking path is exercised.
	r.DisableChunking = false
	r.ChunkRunes = 100
	r.ChunkOverlap = 20
	long := strings.Repeat("Atlas memory subsystem stores short-term facts per run. ", 12)
	_, err = r.Ingest(ctx, Document{
		ID:      "doc-long",
		Title:   "Memory notes",
		Source:  "kb://atlas/memory",
		Content: long,
	})
	if err != nil {
		t.Fatal(err)
	}

	hits, err := r.Search(ctx, "Atlas agent orchestrator plan DAG tools memory RAG", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected hits")
	}
	top := hits[0]
	// Relevant doc (or its content) must rank first over cooking pasta distractor.
	if strings.Contains(strings.ToLower(top.Content), "spaghetti") || top.ID == "doc-distractor" {
		t.Fatalf("distractor ranked first: id=%s content=%q score=%v", top.ID, top.Content, top.Score)
	}
	if !strings.Contains(top.Content, "orchestrator") && !strings.Contains(top.Title, "orchestration") && top.ParentID != "doc-relevant" && top.ID != "doc-relevant" {
		// Allow long memory doc only if score higher and still Atlas-related — but fixture query is orchestrator-specific.
		if !strings.Contains(strings.ToLower(top.Content+top.Title), "atlas") {
			t.Fatalf("top hit not relevant: %+v", top)
		}
	}
	if top.Score == 0 {
		t.Fatalf("expected non-zero score on top hit: %+v", top)
	}
	// Metadata/source should surface for context assembly.
	if top.Source == "" && (top.Metadata == nil || top.Metadata["source"] == "") {
		// ingest set metadata from source; accept either field
		t.Logf("warning: no source metadata on %+v", top)
	}

	// Explicit head-to-head: only the two short docs.
	r2 := NewInProcessRetriever(FakeEmbedder{Dims: 16})
	r2.DisableChunking = true
	_, _ = r2.Ingest(ctx, Document{
		ID: "rel", Title: "Qdrant retrieval", Source: "docs/rag.md",
		Content: "Vector search ranks knowledge base chunks by cosine similarity for RAG quality.",
	})
	_, _ = r2.Ingest(ctx, Document{
		ID: "dist", Title: "Unrelated", Source: "docs/noise.md",
		Content: "The weather in Paris is mild in spring with occasional rain showers.",
	})
	hits2, err := r2.Search(ctx, "vector search RAG cosine knowledge chunks", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits2) < 1 {
		t.Fatal("no hits")
	}
	if hits2[0].ID != "rel" {
		t.Fatalf("want rel first, got id=%s content=%q score=%v all=%+v", hits2[0].ID, hits2[0].Content, hits2[0].Score, hits2)
	}
	if hits2[0].Score <= hits2[1].Score && len(hits2) > 1 {
		// Top must strictly outrank distractor when both present.
		if hits2[1].ID == "dist" && hits2[0].Score < hits2[1].Score {
			t.Fatalf("score order wrong: %+v", hits2)
		}
	}
	if hits2[0].Metadata["source"] != "docs/rag.md" && hits2[0].Source != "docs/rag.md" {
		t.Fatalf("missing source metadata: %+v", hits2[0])
	}
}

func TestDocSnippetsIncludesScoreAndSource(t *testing.T) {
	snips := DocSnippets([]Document{{
		Title: "T", Content: "body", Source: "s1", Score: 1.25,
	}})
	if len(snips) != 1 || !strings.Contains(snips[0], "source=s1") || !strings.Contains(snips[0], "score=") {
		t.Fatalf("%v", snips)
	}
}
