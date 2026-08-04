package main

import (
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
	for _, c := range report.Cases {
		if c.ID == "rag-relevant-first" && c.Pass {
			found = true
		}
	}
	if !found {
		t.Fatal("missing rag-relevant-first case result")
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
