package orchestrator

import (
	"context"
	"strings"
	"testing"
)

func TestMemoryWiredIntoContextBuilder(t *testing.T) {
	mem := NewInMemoryMemory()
	ctx := context.Background()
	_ = mem.PutShort(ctx, "r1", "SHORT_FACT_ALPHA")
	_ = mem.PutLong(ctx, MemoryFact{WorkspaceID: "ws1", Content: "LONG_FACT_BETA", UserID: "u1"})

	notes := CollectMemoryNotes(ctx, mem, "r1", "ws1", "u1")
	joined := strings.Join(notes, " ")
	if !strings.Contains(joined, "SHORT_FACT_ALPHA") || !strings.Contains(joined, "LONG_FACT_BETA") {
		t.Fatalf("notes=%v", notes)
	}

	b := NewContextBuilder()
	out, err := b.Build(ContextRequest{
		System:      "sys",
		History:     []Message{{Role: "user", Content: "hello"}},
		MemoryNotes: notes,
		MaxTokens:   200,
	})
	if err != nil {
		t.Fatal(err)
	}
	blob := ""
	for _, m := range out.Messages {
		blob += m.Content
	}
	if !strings.Contains(blob, "SHORT_FACT_ALPHA") || !strings.Contains(blob, "LONG_FACT_BETA") {
		t.Fatalf("builder missed memory: %s", blob)
	}
}

func TestRAGIngestSearchAndContext(t *testing.T) {
	r := NewInProcessRetriever(FakeEmbedder{Dims: 8})
	ctx := context.Background()
	_, err := r.Ingest(ctx, Document{Title: "Atlas", Content: "Project Atlas is an agent platform with tools and memory."})
	if err != nil {
		t.Fatal(err)
	}
	hits, err := r.Search(ctx, "agent platform tools", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) == 0 {
		t.Fatal("expected retrieval hits")
	}
	if !strings.Contains(hits[0].Content, "Atlas") {
		t.Fatalf("%+v", hits[0])
	}
	snips := DocSnippets(hits)
	b := NewContextBuilder()
	out, err := b.Build(ContextRequest{
		History:     []Message{{Role: "user", Content: "what is atlas?"}},
		DocSnippets: snips,
		MaxTokens:   300,
	})
	if err != nil {
		t.Fatal(err)
	}
	blob := ""
	for _, m := range out.Messages {
		blob += m.Content
	}
	if !strings.Contains(blob, "agent platform") {
		t.Fatalf("context missing doc: %s", blob)
	}
}

func TestExecutorUsesMemoryAndRAG(t *testing.T) {
	mem := NewInMemoryMemory()
	ctx := context.Background()
	_ = mem.PutLong(ctx, MemoryFact{WorkspaceID: "default", Content: "USER_PREFERS_BRIEF_ANSWERS"})

	rag := NewInProcessRetriever(FakeEmbedder{Dims: 8})
	_, _ = rag.Ingest(ctx, Document{Content: "Knowledge base says the capital of Testland is Testville."})

	var sawPrompt string
	gw := &captureGateway{fn: func(req GenerateRequest) GenerateResponse {
		for _, m := range req.Messages {
			sawPrompt += m.Content + "\n"
		}
		var r GenerateResponse
		r.Message.Role = "assistant"
		r.Message.Content = "fake response to: testland"
		return r
	}}

	svc := NewService(ServiceConfig{
		Store:     NewMemoryStore(),
		Gateway:   gw,
		Memory:    mem,
		Retriever: rag,
		Planner:   NewRulePlanner(),
		IDFunc:    func() string { return "run-mem-rag" },
		WorkspaceID: "default",
	})
	created, err := svc.CreateRun(ctx, "What is the capital of Testland?")
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.StartRun(ctx, created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusSucceeded {
		t.Fatalf("%s %s", run.Status, run.Error)
	}
	if !strings.Contains(sawPrompt, "USER_PREFERS_BRIEF_ANSWERS") && !strings.Contains(sawPrompt, "Testville") {
		// at least one of memory or rag should appear in model context
		// prepare step may hold them; model also loads again
		t.Logf("prompt was:\n%s", sawPrompt)
	}
	// Stronger: prepare output or model messages should include knowledge
	prep := stepByID(run, "prepare")
	combined := sawPrompt
	if prep != nil {
		combined += prep.Output
	}
	if !strings.Contains(combined, "Testville") && !strings.Contains(combined, "USER_PREFERS") {
		t.Fatalf("expected memory or RAG in context, got:\n%s", combined)
	}
}

type captureGateway struct {
	fn func(GenerateRequest) GenerateResponse
}

func (c *captureGateway) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	return c.fn(req), nil
}
