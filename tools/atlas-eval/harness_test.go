package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	orchestrator "github.com/KunalSiyag/Project-Atlas/services/agent-orchestrator"
)

func TestRunSampleFixture(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "eval", "v1", "sample.json")
	ds, err := LoadDataset(root)
	if err != nil {
		t.Fatal(err)
	}
	report := RunDataset(ds)
	if !report.Passed {
		raw, _ := json.MarshalIndent(report, "", "  ")
		t.Fatalf("fixture failed:\n%s", raw)
	}
	if report.Total != len(ds.Cases) || report.PassedCount != report.Total {
		t.Fatalf("counts total=%d passed=%d", report.Total, report.PassedCount)
	}
	for _, c := range report.Cases {
		if !c.Pass || c.Score != 1.0 {
			t.Fatalf("case %+v", c)
		}
	}
	found := false
	toolByName := false
	for _, c := range report.Cases {
		if c.ID == "rag-relevant-first" && c.Pass {
			found = true
		}
		if c.ID == "tool-calculator-success" && c.Pass && c.Type == "tool_success" {
			toolByName = true
		}
	}
	if !found {
		t.Fatal("missing rag-relevant-first case result")
	}
	if !toolByName {
		t.Fatal("missing passing tool-calculator-success (tool success by name)")
	}
}

func TestRunExtendedFixture(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "eval", "v1", "extended.json")
	ds, err := LoadDataset(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(ds.Cases) < 5 {
		t.Fatalf("extended suite too small: %d cases", len(ds.Cases))
	}
	report := RunDataset(ds)
	if !report.Passed {
		raw, _ := json.MarshalIndent(report, "", "  ")
		t.Fatalf("extended fixture failed:\n%s", raw)
	}
	if report.Total != len(ds.Cases) || report.PassedCount != report.Total {
		t.Fatalf("counts total=%d passed=%d", report.Total, report.PassedCount)
	}
	// Must exercise real tool_success and rule_plan scorers
	var sawTool, sawPlan bool
	for _, c := range report.Cases {
		if c.Type == "tool_success" && c.Pass {
			sawTool = true
		}
		if c.Type == "rule_plan" && c.Pass {
			sawPlan = true
		}
	}
	if !sawTool || !sawPlan {
		t.Fatalf("extended suite must pass tool_success and rule_plan (tool=%v plan=%v)", sawTool, sawPlan)
	}
}

func TestToolSuccessByNameDrivesRegistry(t *testing.T) {
	// Drive shipped DefaultToolRegistry via scoreToolSuccess — not a reimplementation.
	c := Case{
		ID:   "unit-calc",
		Type: "tool_success",
		Input: json.RawMessage(`{"tool":"calculator","input":{"expression":"2*3"}}`),
		Expect: json.RawMessage(`{"equals":"6"}`),
	}
	pass, msg, err := scoreToolSuccess(c)
	if err != nil {
		t.Fatal(err)
	}
	if !pass {
		t.Fatalf("expected pass: %s", msg)
	}
	// Failure path: schema miss fails through real Invoke
	cBad := Case{
		ID:     "unit-bad",
		Type:   "tool_success",
		Input:  json.RawMessage(`{"tool":"calculator","input":{}}`),
		Expect: json.RawMessage(`{}`),
	}
	pass, msg, err = scoreToolSuccess(cBad)
	if err != nil {
		t.Fatal(err)
	}
	if pass {
		t.Fatal("expected schema failure for empty calculator input")
	}
	if msg == "" {
		t.Fatal("expected failure message")
	}
}

func TestLLMJudgeStubPassAndUnavailable(t *testing.T) {
	root := filepath.Join("..", "..", "fixtures", "eval", "v1", "sample_with_judge.json")
	ds, err := LoadDataset(root)
	if err != nil {
		t.Fatal(err)
	}
	// (a) stub judge pass
	report := RunDatasetWithOptions(ds, Options{
		Judge: orchestrator.StaticJudge{Pass: true, Score: 0.95, Rationale: "stub ok"},
	})
	if !report.Passed {
		raw, _ := json.MarshalIndent(report, "", "  ")
		t.Fatalf("judge fixture should pass with stub:\n%s", raw)
	}
	var judgeCase *CaseResult
	for i := range report.Cases {
		if report.Cases[i].ID == "judge-quality" {
			judgeCase = &report.Cases[i]
		}
	}
	if judgeCase == nil || !judgeCase.Pass || judgeCase.Score != 0.95 {
		t.Fatalf("judge case: %+v", judgeCase)
	}

	// (b) unavailable judge — no panic; case errors
	report2 := RunDatasetWithOptions(ds, Options{Judge: nil})
	if report2.Passed {
		t.Fatal("expected fail without judge")
	}
	for _, c := range report2.Cases {
		if c.ID == "judge-quality" {
			if c.Pass || c.Status != "error" {
				t.Fatalf("want error status for unavailable judge: %+v", c)
			}
		}
	}

	// failing judge client
	report3 := RunDatasetWithOptions(ds, Options{
		Judge: orchestrator.StaticJudge{Err: errors.New("gateway down")},
	})
	if report3.Passed {
		t.Fatal("expected fail when judge errors")
	}
}

// TestLLMJudgeGatewayProductPipeline drives shipped GatewayJudge through scoreLLMJudge / RunDatasetWithOptions.
func TestLLMJudgeGatewayProductPipeline(t *testing.T) {
	// Fake inference client returns structured judge JSON (real GatewayJudge.Judge path).
	gw := &judgeGatewayStub{content: `{"pass":true,"score":0.9,"rationale":"mentions Atlas planning"}`}
	judge := &orchestrator.GatewayJudge{Client: gw, Model: "fake-judge"}
	root := filepath.Join("..", "..", "fixtures", "eval", "v1", "sample_with_judge.json")
	ds, err := LoadDataset(root)
	if err != nil {
		t.Fatal(err)
	}
	report := RunDatasetWithOptions(ds, Options{Judge: judge})
	if !report.Passed {
		raw, _ := json.MarshalIndent(report, "", "  ")
		t.Fatalf("gateway judge pipeline failed:\n%s", raw)
	}
	if gw.calls < 1 {
		t.Fatal("GatewayJudge.Judge was not invoked")
	}
	for _, c := range report.Cases {
		if c.ID == "judge-quality" {
			if !c.Pass || c.Score < 0.9 {
				t.Fatalf("judge-quality: %+v", c)
			}
			return
		}
	}
	t.Fatal("missing judge-quality case")
}

type judgeGatewayStub struct {
	content string
	calls   int
}

func (j *judgeGatewayStub) Generate(ctx context.Context, req orchestrator.GenerateRequest) (orchestrator.GenerateResponse, error) {
	j.calls++
	return orchestrator.GenerateResponse{
		Message: orchestrator.Message{Role: "assistant", Content: j.content},
	}, nil
}

func TestWriteReportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	r := Report{
		Version: "1", Suite: "t", Passed: true, Total: 1, PassedCount: 1,
		Cases: []CaseResult{{ID: "a", Type: "string_contains", Pass: true, Score: 1.0, Status: "pass"}},
	}
	if err := WriteReport(path, r); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var got Report
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if !got.Passed || got.Cases[0].Score != 1.0 {
		t.Fatalf("%+v", got)
	}
}
