package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// Planner builds an executable step graph from a goal.
type Planner interface {
	Plan(ctx context.Context, goal string) ([]Step, error)
}

// LLMPlanner asks the inference gateway for a JSON plan, validates it, and
// falls back to Fallback (default RulePlanner) when the model is unavailable
// or returns an invalid plan.
type LLMPlanner struct {
	Gateway  InferenceClient
	Model    string
	Fallback Planner
	// Prompts optional file-backed registry for system/user plan templates.
	Prompts *PromptRegistry
}

// NewLLMPlanner constructs a hybrid planner. Fallback defaults to RulePlanner.
func NewLLMPlanner(gateway InferenceClient, model string, fallback Planner) *LLMPlanner {
	if fallback == nil {
		fallback = NewRulePlanner()
	}
	return &LLMPlanner{Gateway: gateway, Model: model, Fallback: fallback}
}

// WithPrompts attaches a prompt registry for plan system/user templates.
func (p *LLMPlanner) WithPrompts(reg *PromptRegistry) *LLMPlanner {
	if p != nil {
		p.Prompts = reg
	}
	return p
}

func (p *LLMPlanner) Plan(ctx context.Context, goal string) ([]Step, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}
	if p.Gateway != nil {
		steps, err := planFromModel(ctx, p.Gateway, p.Model, goal, p.Prompts)
		if err == nil {
			if err := ValidatePlan(steps); err == nil {
				return steps, nil
			}
		}
		// Model failed or invalid plan — deterministic fallback.
	}
	if p.Fallback == nil {
		return NewRulePlanner().Plan(ctx, goal)
	}
	return p.Fallback.Plan(ctx, goal)
}

// RulePlanner produces multi-step plans from goal keywords (deterministic, no LLM required).
type RulePlanner struct {
	// Gateway optional: if set and goal asks for "json plan", can call model (optional path).
	Gateway InferenceClient
	Model   string
	Prompts *PromptRegistry
}

func NewRulePlanner() *RulePlanner { return &RulePlanner{} }

var calcRe = regexp.MustCompile(`(?i)(calculate|compute|math|\d+\s*[\+\-\*/]\s*\d+)`)
var httpRe = regexp.MustCompile(`(?i)(https?://\S+|fetch url|http get)`)
var dangerRe = regexp.MustCompile(`(?i)(danger|approve|side[- ]?effect)`)
var echoRe = regexp.MustCompile(`(?i)\becho\b`)

func (p *RulePlanner) Plan(ctx context.Context, goal string) ([]Step, error) {
	goal = strings.TrimSpace(goal)
	if goal == "" {
		return nil, fmt.Errorf("goal is required")
	}

	// Optional: model JSON plan when goal starts with "plan:" (legacy path).
	// Prefer LLMPlanner for always-on model planning with rule fallback.
	if strings.HasPrefix(strings.ToLower(goal), "plan:") && p.Gateway != nil {
		steps, err := planFromModel(ctx, p.Gateway, p.Model, strings.TrimSpace(goal[5:]), p.Prompts)
		if err == nil {
			if err := ValidatePlan(steps); err != nil {
				return nil, err
			}
			return steps, nil
		}
		// fall through to rules on model failure
	}

	steps := []Step{
		{StepID: "prepare", Name: "prepare_context", Kind: StepKindContext, Status: StepPending, MaxAttempts: 1},
	}
	prev := "prepare"

	if calcRe.MatchString(goal) {
		expr := extractExpression(goal)
		steps = append(steps, Step{
			StepID: "tool_calc", Name: "calculator", Kind: StepKindTool, DependsOn: []string{prev},
			Status: StepPending, MaxAttempts: 1, ToolName: "calculator",
			ToolInput: map[string]any{"expression": expr},
		})
		prev = "tool_calc"
	}
	if echoRe.MatchString(goal) {
		steps = append(steps, Step{
			StepID: "tool_echo", Name: "echo", Kind: StepKindTool, DependsOn: []string{prev},
			Status: StepPending, MaxAttempts: 1, ToolName: "echo",
			ToolInput: map[string]any{"text": goal},
		})
		prev = "tool_echo"
	}
	if httpRe.MatchString(goal) {
		url := extractURL(goal)
		if url == "" {
			url = "https://example.com"
		}
		steps = append(steps, Step{
			StepID: "tool_http", Name: "http_get", Kind: StepKindTool, DependsOn: []string{prev},
			Status: StepPending, MaxAttempts: 1, ToolName: "http_get",
			ToolInput: map[string]any{"url": url},
		})
		prev = "tool_http"
	}
	if dangerRe.MatchString(goal) {
		steps = append(steps, Step{
			StepID: "tool_danger", Name: "danger_echo", Kind: StepKindTool, DependsOn: []string{prev},
			Status: StepPending, MaxAttempts: 1, ToolName: "danger_echo",
			ToolInput: map[string]any{"text": goal}, RequiresApproval: true,
		})
		prev = "tool_danger"
	}

	steps = append(steps,
		Step{
			StepID: "infer", Name: "call_inference", Kind: StepKindModel, DependsOn: []string{prev},
			Status: StepPending, MaxAttempts: 2,
		},
		Step{
			StepID: "finalize", Name: "return_answer", Kind: StepKindFinalize, DependsOn: []string{"infer"},
			Status: StepPending, MaxAttempts: 1,
		},
	)
	if err := ValidatePlan(steps); err != nil {
		return nil, err
	}
	return steps, nil
}

// planFromModel asks the gateway for a JSON step array and normalizes defaults.
// When prompts is non-nil, system/user text come from the file-backed registry.
func planFromModel(ctx context.Context, gateway InferenceClient, model, goal string, prompts *PromptRegistry) ([]Step, error) {
	if gateway == nil {
		return nil, fmt.Errorf("gateway is required")
	}
	vars := map[string]string{"goal": goal}
	system := defaultPlanSystemPrompt
	user := defaultPlanUserPrompt(goal)
	if prompts != nil {
		if s, _, err := prompts.ResolveAndRender(PromptPlanSystem, "", vars); err == nil && strings.TrimSpace(s) != "" {
			system = s
		}
		if u, _, err := prompts.ResolveAndRender(PromptPlanUser, "", vars); err == nil && strings.TrimSpace(u) != "" {
			user = u
		}
	}
	resp, err := gateway.Generate(ctx, GenerateRequest{
		Messages: []Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Model:     model,
		TaskClass: "plan",
	})
	if err != nil {
		return nil, err
	}
	return ParsePlanJSON(resp.Message.Content)
}

const defaultPlanSystemPrompt = "You are an Atlas plan compiler. Output valid JSON array only. No prose."

func defaultPlanUserPrompt(goal string) string {
	return `Return ONLY a JSON array of plan steps for the user goal.
Each step object fields: step_id (string), name (string), kind (one of context|model|tool|finalize),
depends_on (array of step_id), tool_name (for tool steps), tool_input (object), max_attempts (int optional).
Include at least one context step, one model or tool step, and a finalize step.
Goal: ` + goal
}

// ParsePlanJSON extracts and normalizes a step array from model output.
// Exported so tests and eval can exercise the same validation path.
func ParsePlanJSON(content string) ([]Step, error) {
	raw := strings.TrimSpace(content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```JSON")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	// Tolerate leading prose by locating the first JSON array.
	if i := strings.Index(raw, "["); i >= 0 {
		if j := strings.LastIndex(raw, "]"); j > i {
			raw = raw[i : j+1]
		}
	}
	var steps []Step
	if err := json.Unmarshal([]byte(raw), &steps); err != nil {
		return nil, fmt.Errorf("invalid plan JSON: %w", err)
	}
	for i := range steps {
		if steps[i].Status == "" {
			steps[i].Status = StepPending
		}
		if steps[i].MaxAttempts == 0 {
			steps[i].MaxAttempts = 1
		}
	}
	return steps, nil
}

// ValidatePlan ensures unique ids, valid kinds, and acyclic deps.
func ValidatePlan(steps []Step) error {
	if len(steps) == 0 {
		return fmt.Errorf("plan is empty")
	}
	ids := map[string]bool{}
	for _, s := range steps {
		if s.StepID == "" {
			return fmt.Errorf("step missing step_id")
		}
		if ids[s.StepID] {
			return fmt.Errorf("duplicate step_id %q", s.StepID)
		}
		ids[s.StepID] = true
		switch s.Kind {
		case StepKindContext, StepKindModel, StepKindTool, StepKindFinalize:
		default:
			return fmt.Errorf("step %q has invalid kind %q", s.StepID, s.Kind)
		}
		if s.Kind == StepKindTool && s.ToolName == "" {
			return fmt.Errorf("tool step %q missing tool_name", s.StepID)
		}
		for _, d := range s.DependsOn {
			if !ids[d] && d != s.StepID {
				// dep may appear later — second pass
			}
		}
	}
	// deps exist
	for _, s := range steps {
		for _, d := range s.DependsOn {
			if !ids[d] {
				return fmt.Errorf("step %q depends on unknown %q", s.StepID, d)
			}
			if d == s.StepID {
				return fmt.Errorf("step %q depends on itself", s.StepID)
			}
		}
	}
	// cycle detect via DFS
	state := map[string]int{} // 0=unseen 1=visiting 2=done
	var visit func(string) error
	adj := map[string][]string{}
	for _, s := range steps {
		adj[s.StepID] = append([]string{}, s.DependsOn...)
	}
	visit = func(id string) error {
		if state[id] == 1 {
			return fmt.Errorf("cycle detected at %q", id)
		}
		if state[id] == 2 {
			return nil
		}
		state[id] = 1
		for _, d := range adj[id] {
			if err := visit(d); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for _, s := range steps {
		if err := visit(s.StepID); err != nil {
			return err
		}
	}
	return nil
}

func extractExpression(goal string) string {
	re := regexp.MustCompile(`(\d+(?:\.\d+)?)\s*([\+\-\*/])\s*(\d+(?:\.\d+)?)`)
	m := re.FindStringSubmatch(goal)
	if len(m) == 4 {
		return m[1] + m[2] + m[3]
	}
	return "1+1"
}

func extractURL(goal string) string {
	re := regexp.MustCompile(`https?://\S+`)
	return re.FindString(goal)
}
