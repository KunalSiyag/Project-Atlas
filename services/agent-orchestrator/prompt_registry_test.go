package orchestrator

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoPromptsDir(t *testing.T) string {
	t.Helper()
	// Walk up from CWD (package dir when go test runs) to find prompts/.
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "prompts")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("prompts/ directory not found from ", wd)
	return ""
}

func TestPromptRegistryLoadResolveRender(t *testing.T) {
	reg, err := LoadPromptDir(repoPromptsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	// Highest active version of system prompt (v1, not retired v0).
	entry, err := reg.Resolve(PromptPlanSystem, "")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Version != "1" {
		t.Fatalf("version=%q want 1 (active highest)", entry.Version)
	}
	text, err := entry.Render(map[string]string{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "ATLAS_PLAN_COMPILER_V1") {
		t.Fatalf("rendered system missing distinctive phrase: %q", text)
	}

	// User prompt requires goal.
	_, _, err = reg.ResolveAndRender(PromptPlanUser, "", nil)
	if err == nil || !strings.Contains(err.Error(), "missing required") {
		t.Fatalf("expected missing required var error, got %v", err)
	}
	user, _, err := reg.ResolveAndRender(PromptPlanUser, "", map[string]string{"goal": "calculate 2+2"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(user, "calculate 2+2") {
		t.Fatalf("user render=%q", user)
	}

	// Explicit version
	old, err := reg.Resolve(PromptPlanSystem, "0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(old.Text, "OLD_RETIRED") {
		t.Fatalf("%q", old.Text)
	}
}

func TestPromptRegistryHotReload(t *testing.T) {
	dir := t.TempDir()
	writePromptJSON(t, dir, "hello.json", PromptEntry{
		ID: "test.hotreload", Version: "1", Status: "active",
		Text: "VERSION_ONE",
	})
	reg, err := LoadPromptDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	e, err := reg.Resolve("test.hotreload", "")
	if err != nil {
		t.Fatal(err)
	}
	if e.Text != "VERSION_ONE" {
		t.Fatalf("got %q", e.Text)
	}
	// Mutate file on disk and Reload without recreating the process/registry pointer.
	writePromptJSON(t, dir, "hello.json", PromptEntry{
		ID: "test.hotreload", Version: "1", Status: "active",
		Text: "VERSION_TWO_RELOADED",
	})
	if err := reg.Reload(); err != nil {
		t.Fatal(err)
	}
	e, err = reg.Resolve("test.hotreload", "")
	if err != nil {
		t.Fatal(err)
	}
	if e.Text != "VERSION_TWO_RELOADED" {
		t.Fatalf("after reload got %q want VERSION_TWO_RELOADED", e.Text)
	}
	text, err := e.Render(nil)
	if err != nil || text != "VERSION_TWO_RELOADED" {
		t.Fatalf("render %q %v", text, err)
	}
}

func TestPromptRegistryPublishMemoryStoreResolve(t *testing.T) {
	store := NewMemoryPromptStore()
	reg := NewPromptRegistry(store)
	ctx := context.Background()
	// File-less registry: publish is the only source.
	err := reg.Publish(ctx, PromptEntry{
		ID: "test.published", Version: "3", Status: "active",
		Text: "hello {{name}}",
		Variables: []PromptVar{{Name: "name", Required: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	// Same logical id/version as file prompts.
	entry, err := reg.Resolve("test.published", "3")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Text != "hello {{name}}" {
		t.Fatalf("%+v", entry)
	}
	// Highest active without version.
	entry, err = reg.Resolve("test.published", "")
	if err != nil {
		t.Fatal(err)
	}
	if entry.Version != "3" {
		t.Fatalf("version=%s", entry.Version)
	}
	out, _, err := reg.ResolveAndRender("test.published", "", map[string]string{"name": "atlas"})
	if err != nil || out != "hello atlas" {
		t.Fatalf("%q %v", out, err)
	}
	// Durable store round-trip via LoadFromStore on a fresh registry.
	reg2 := NewPromptRegistry(store)
	if err := reg2.LoadFromStore(ctx); err != nil {
		t.Fatal(err)
	}
	got, err := reg2.Resolve("test.published", "3")
	if err != nil || got.Text != "hello {{name}}" {
		t.Fatalf("%+v %v", got, err)
	}
}

func TestPromptRegistryPublishPostgres(t *testing.T) {
	url := postgresTestURL()
	ctx := context.Background()
	pg, err := OpenPostgresPromptStore(ctx, url)
	if err != nil {
		t.Skipf("postgres unavailable for prompt publish: %v", err)
	}
	defer pg.Close()

	// Isolate key per run.
	id := "test.pg.prompt." + newID()[:8]
	reg := NewPromptRegistry(pg)
	entry := PromptEntry{
		ID: id, Version: "1", Status: "active",
		Text: "PG_PUBLISHED {{goal}}",
		Variables: []PromptVar{{Name: "goal", Required: true}},
	}
	if err := reg.Publish(ctx, entry); err != nil {
		t.Fatal(err)
	}
	// Resolve via same registry (in-memory merge after publish).
	got, err := reg.Resolve(id, "1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != entry.Text {
		t.Fatalf("resolve after publish: %+v", got)
	}
	// New store connection + LoadFromStore (restart simulation).
	pg2, err := OpenPostgresPromptStore(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer pg2.Close()
	reg2 := NewPromptRegistry(pg2)
	if err := reg2.LoadFromStore(ctx); err != nil {
		t.Fatal(err)
	}
	got2, err := reg2.Resolve(id, "")
	if err != nil {
		t.Fatal(err)
	}
	if got2.Version != "1" || got2.Text != entry.Text {
		t.Fatalf("%+v", got2)
	}
	text, err := got2.Render(map[string]string{"goal": "ship"})
	if err != nil || text != "PG_PUBLISHED ship" {
		t.Fatalf("%q %v", text, err)
	}
	// Direct store Get path (workspace-scoped).
	row, err := pg2.Get(ctx, "default", id, "1")
	if err != nil || row.Text != entry.Text {
		t.Fatalf("store Get: %+v %v", row, err)
	}
}

func writePromptJSON(t *testing.T, dir, name string, e PromptEntry) {
	t.Helper()
	raw, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
		t.Fatal(err)
	}
}

func postgresTestURL() string {
	url := strings.TrimSpace(os.Getenv("ATLAS_ORCHESTRATOR_DATABASE_URL"))
	if url == "" {
		url = strings.TrimSpace(os.Getenv("ATLAS_DATABASE_URL"))
	}
	if url == "" {
		url = "postgres://atlas:atlas-dev-password@127.0.0.1:5432/atlas?sslmode=disable"
	}
	return url
}

func TestLLMPlannerUsesRegistrySystemPrompt(t *testing.T) {
	reg, err := LoadPromptDir(repoPromptsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	var sawSystem string
	gw := &stubGateway{content: `[
		{"step_id":"prepare","name":"prepare_context","kind":"context"},
		{"step_id":"infer","name":"call_inference","kind":"model","depends_on":["prepare"]},
		{"step_id":"finalize","name":"return_answer","kind":"finalize","depends_on":["infer"]}
	]`}
	// Capture system message via custom gateway
	cap := &captureSystemGateway{inner: gw, onReq: func(req GenerateRequest) {
		for _, m := range req.Messages {
			if m.Role == "system" {
				sawSystem = m.Content
			}
		}
	}}
	p := NewLLMPlanner(cap, "fake-model", NewRulePlanner()).WithPrompts(reg)
	steps, err := p.Plan(context.Background(), "summarize the workspace")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePlan(steps); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sawSystem, "ATLAS_PLAN_COMPILER_V1") {
		t.Fatalf("planner did not use registry system prompt: %q", sawSystem)
	}
}

type captureSystemGateway struct {
	inner InferenceClient
	onReq func(GenerateRequest)
}

func (c *captureSystemGateway) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	if c.onReq != nil {
		c.onReq(req)
	}
	return c.inner.Generate(ctx, req)
}
