package orchestrator

import (
	"context"
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
