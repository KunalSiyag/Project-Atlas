package orchestrator

import (
	"context"
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
