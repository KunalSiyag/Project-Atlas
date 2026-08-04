package orchestrator

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRulePlannerDynamicPlans(t *testing.T) {
	p := NewRulePlanner()
	steps, err := p.Plan(context.Background(), "calculate 12+30 for me")
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) < 4 {
		t.Fatalf("expected multi-step plan, got %d", len(steps))
	}
	kinds := map[string]bool{}
	for _, s := range steps {
		kinds[s.Kind] = true
		if s.Kind == StepKindTool && s.ToolName == "calculator" {
			if s.ToolInput["expression"] != "12+30" {
				t.Fatalf("expr=%v", s.ToolInput)
			}
		}
	}
	if !kinds[StepKindTool] || !kinds[StepKindModel] {
		t.Fatalf("kinds=%v", kinds)
	}
	if err := ValidatePlan(steps); err != nil {
		t.Fatal(err)
	}
}

// stubGateway returns fixed model content or an error — drives LLMPlanner.
type stubGateway struct {
	content string
	err     error
	calls   int
}

func (s *stubGateway) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	s.calls++
	if s.err != nil {
		return GenerateResponse{}, s.err
	}
	var r GenerateResponse
	r.Message.Role = "assistant"
	r.Message.Content = s.content
	return r, nil
}

func TestLLMPlannerUsesModelJSONPlan(t *testing.T) {
	// Valid multi-step plan from gateway JSON (not rule-keyword based).
	jsonPlan := `[
		{"step_id":"prepare","name":"prepare_context","kind":"context","depends_on":[]},
		{"step_id":"tool_calc","name":"calculator","kind":"tool","depends_on":["prepare"],"tool_name":"calculator","tool_input":{"expression":"7+8"}},
		{"step_id":"infer","name":"call_inference","kind":"model","depends_on":["tool_calc"]},
		{"step_id":"finalize","name":"return_answer","kind":"finalize","depends_on":["infer"]}
	]`
	gw := &stubGateway{content: "```json\n" + jsonPlan + "\n```"}
	p := NewLLMPlanner(gw, "fake-model", NewRulePlanner())
	steps, err := p.Plan(context.Background(), "please sum seven and eight")
	if err != nil {
		t.Fatal(err)
	}
	if gw.calls != 1 {
		t.Fatalf("gateway calls=%d want 1", gw.calls)
	}
	if err := ValidatePlan(steps); err != nil {
		t.Fatal(err)
	}
	// Must be the model plan (tool_calc with 7+8), not pure rule defaults.
	found := false
	for _, s := range steps {
		if s.StepID == "tool_calc" && s.ToolName == "calculator" {
			found = true
			if s.ToolInput["expression"] != "7+8" {
				t.Fatalf("expected model tool_input expression 7+8, got %v", s.ToolInput)
			}
		}
	}
	if !found {
		t.Fatalf("expected model-provided tool_calc step, got %+v", steps)
	}
	if len(steps) != 4 {
		t.Fatalf("len=%d steps=%+v", len(steps), steps)
	}
}

func TestLLMPlannerFallsBackWhenGatewayFails(t *testing.T) {
	gw := &stubGateway{err: errors.New("gateway down")}
	p := NewLLMPlanner(gw, "fake-model", NewRulePlanner())
	// Goal has calculator keywords so rule fallback is non-trivial.
	steps, err := p.Plan(context.Background(), "calculate 21+21 please")
	if err != nil {
		t.Fatal(err)
	}
	if gw.calls != 1 {
		t.Fatalf("gateway calls=%d", gw.calls)
	}
	if err := ValidatePlan(steps); err != nil {
		t.Fatal(err)
	}
	// Rule planner should have produced a calculator tool step.
	hasCalc := false
	for _, s := range steps {
		if s.Kind == StepKindTool && s.ToolName == "calculator" {
			hasCalc = true
			if s.ToolInput["expression"] != "21+21" {
				t.Fatalf("rule expr=%v", s.ToolInput)
			}
		}
	}
	if !hasCalc {
		t.Fatalf("fallback plan missing calculator: %+v", steps)
	}
}

func TestLLMPlannerFallsBackOnInvalidJSON(t *testing.T) {
	gw := &stubGateway{content: "not a plan at all"}
	p := NewLLMPlanner(gw, "m", NewRulePlanner())
	steps, err := p.Plan(context.Background(), "calculate 1+1")
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePlan(steps); err != nil {
		t.Fatal(err)
	}
}

func TestParsePlanJSONExtractsArray(t *testing.T) {
	raw := `Here you go:\n[{"step_id":"prepare","name":"p","kind":"context"},{"step_id":"infer","name":"i","kind":"model","depends_on":["prepare"]},{"step_id":"finalize","name":"f","kind":"finalize","depends_on":["infer"]}]`
	steps, err := ParsePlanJSON(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidatePlan(steps); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePlanRejectsCycle(t *testing.T) {
	err := ValidatePlan([]Step{
		{StepID: "a", Kind: StepKindContext, DependsOn: []string{"b"}, MaxAttempts: 1},
		{StepID: "b", Kind: StepKindModel, DependsOn: []string{"a"}, MaxAttempts: 1},
	})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("got %v", err)
	}
}

func TestValidatePlanRejectsBadKind(t *testing.T) {
	err := ValidatePlan([]Step{{StepID: "a", Kind: "nope", MaxAttempts: 1}})
	if err == nil {
		t.Fatal("expected error")
	}
}
