package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	orchestrator "github.com/KunalSiyag/Project-Atlas/services/agent-orchestrator"
)

// Dataset is the v1 fixture root.
type Dataset struct {
	Version     string `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Cases       []Case `json:"cases"`
}

// Case is one deterministic evaluation case.
type Case struct {
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Description string          `json:"description,omitempty"`
	Input       json.RawMessage `json:"input"`
	Expect      json.RawMessage `json:"expect"`
}

// CaseResult is one scored case in the report.
type CaseResult struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	Pass    bool    `json:"pass"`
	Score   float64 `json:"score"`
	Message string  `json:"message,omitempty"`
}

// Report is the machine-readable harness output.
type Report struct {
	Version     string       `json:"version"`
	Suite       string       `json:"suite"`
	GeneratedAt string       `json:"generated_at"`
	Passed      bool         `json:"passed"`
	Total       int          `json:"total"`
	PassedCount int          `json:"passed_count"`
	FailedCount int          `json:"failed_count"`
	Cases       []CaseResult `json:"cases"`
}

// LoadDataset reads a v1 fixture file.
func LoadDataset(path string) (Dataset, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Dataset{}, err
	}
	var ds Dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return Dataset{}, fmt.Errorf("decode dataset: %w", err)
	}
	if ds.Version != "1" {
		return Dataset{}, fmt.Errorf("unsupported fixture version %q (want 1)", ds.Version)
	}
	if len(ds.Cases) == 0 {
		return Dataset{}, fmt.Errorf("dataset has no cases")
	}
	return ds, nil
}

// RunDataset scores every case with shipped orchestrator code paths.
func RunDataset(ds Dataset) Report {
	report := Report{
		Version:     "1",
		Suite:       ds.Name,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Total:       len(ds.Cases),
		Cases:       make([]CaseResult, 0, len(ds.Cases)),
	}
	for _, c := range ds.Cases {
		cr := scoreCase(c)
		report.Cases = append(report.Cases, cr)
		if cr.Pass {
			report.PassedCount++
		} else {
			report.FailedCount++
		}
	}
	report.Passed = report.FailedCount == 0
	return report
}

func scoreCase(c Case) CaseResult {
	cr := CaseResult{ID: c.ID, Type: c.Type, Score: 0}
	var err error
	var pass bool
	var msg string
	switch strings.ToLower(strings.TrimSpace(c.Type)) {
	case "rule_plan":
		pass, msg, err = scoreRulePlan(c)
	case "plan_json":
		pass, msg, err = scorePlanJSON(c)
	case "rag_rank":
		pass, msg, err = scoreRAGRank(c)
	case "string_contains":
		pass, msg, err = scoreStringContains(c)
	default:
		err = fmt.Errorf("unknown case type %q", c.Type)
	}
	if err != nil {
		cr.Message = err.Error()
		cr.Pass = false
		cr.Score = 0
		return cr
	}
	cr.Pass = pass
	cr.Message = msg
	if pass {
		cr.Score = 1.0
	} else {
		cr.Score = 0
	}
	return cr
}

func scoreRulePlan(c Case) (bool, string, error) {
	var in struct {
		Goal string `json:"goal"`
	}
	var exp struct {
		HasTool  string   `json:"has_tool"`
		MinSteps int      `json:"min_steps"`
		Kinds    []string `json:"kinds"`
	}
	if err := json.Unmarshal(c.Input, &in); err != nil {
		return false, "", err
	}
	if err := json.Unmarshal(c.Expect, &exp); err != nil {
		return false, "", err
	}
	steps, err := orchestrator.NewRulePlanner().Plan(context.Background(), in.Goal)
	if err != nil {
		return false, err.Error(), nil
	}
	if exp.MinSteps > 0 && len(steps) < exp.MinSteps {
		return false, fmt.Sprintf("steps=%d want >= %d", len(steps), exp.MinSteps), nil
	}
	kinds := map[string]bool{}
	tools := map[string]bool{}
	for _, s := range steps {
		kinds[s.Kind] = true
		if s.ToolName != "" {
			tools[s.ToolName] = true
		}
	}
	if exp.HasTool != "" && !tools[exp.HasTool] {
		return false, fmt.Sprintf("missing tool %q", exp.HasTool), nil
	}
	for _, k := range exp.Kinds {
		if !kinds[k] {
			return false, fmt.Sprintf("missing kind %q", k), nil
		}
	}
	return true, fmt.Sprintf("steps=%d", len(steps)), nil
}

func scorePlanJSON(c Case) (bool, string, error) {
	var in struct {
		PlanJSON string `json:"plan_json"`
	}
	var exp struct {
		Valid bool `json:"valid"`
	}
	if err := json.Unmarshal(c.Input, &in); err != nil {
		return false, "", err
	}
	if err := json.Unmarshal(c.Expect, &exp); err != nil {
		return false, "", err
	}
	steps, err := orchestrator.ParsePlanJSON(in.PlanJSON)
	valid := err == nil
	if valid {
		if vErr := orchestrator.ValidatePlan(steps); vErr != nil {
			valid = false
			err = vErr
		}
	}
	if valid != exp.Valid {
		if err != nil {
			return false, fmt.Sprintf("valid=%v want %v (%v)", valid, exp.Valid, err), nil
		}
		return false, fmt.Sprintf("valid=%v want %v", valid, exp.Valid), nil
	}
	return true, fmt.Sprintf("valid=%v", valid), nil
}

func scoreRAGRank(c Case) (bool, string, error) {
	var in struct {
		Query string `json:"query"`
		TopK  int    `json:"top_k"`
		Docs  []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			Content string `json:"content"`
			Source  string `json:"source"`
		} `json:"docs"`
	}
	var exp struct {
		TopID string `json:"top_id"`
	}
	if err := json.Unmarshal(c.Input, &in); err != nil {
		return false, "", err
	}
	if err := json.Unmarshal(c.Expect, &exp); err != nil {
		return false, "", err
	}
	r := orchestrator.NewInProcessRetriever(orchestrator.FakeEmbedder{Dims: 16})
	r.DisableChunking = true
	ctx := context.Background()
	for _, d := range in.Docs {
		if _, err := r.Ingest(ctx, orchestrator.Document{
			ID: d.ID, Title: d.Title, Content: d.Content, Source: d.Source,
		}); err != nil {
			return false, "", err
		}
	}
	hits, err := r.Search(ctx, in.Query, in.TopK)
	if err != nil {
		return false, err.Error(), nil
	}
	if len(hits) == 0 {
		return false, "no hits", nil
	}
	top := hits[0]
	if top.ID != exp.TopID {
		return false, fmt.Sprintf("top_id=%q want %q score=%v", top.ID, exp.TopID, top.Score), nil
	}
	return true, fmt.Sprintf("top_id=%s score=%.3f", top.ID, top.Score), nil
}

func scoreStringContains(c Case) (bool, string, error) {
	var in struct {
		Text string `json:"text"`
	}
	var exp struct {
		Contains []string `json:"contains"`
	}
	if err := json.Unmarshal(c.Input, &in); err != nil {
		return false, "", err
	}
	if err := json.Unmarshal(c.Expect, &exp); err != nil {
		return false, "", err
	}
	for _, needle := range exp.Contains {
		if !strings.Contains(in.Text, needle) {
			return false, fmt.Sprintf("missing %q", needle), nil
		}
	}
	return true, "ok", nil
}

// WriteReport writes the report JSON to path.
func WriteReport(path string, report Report) error {
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}
