package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSampleFixture(t *testing.T) {
	// Resolve sample relative to repo root from this package.
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
	// Every case must have a score field of 1.0 on pass.
	for _, c := range report.Cases {
		if !c.Pass || c.Score != 1.0 {
			t.Fatalf("case %+v", c)
		}
		if c.ID == "" || c.Type == "" {
			t.Fatalf("missing id/type: %+v", c)
		}
	}
	// Known fixture expectation: rag case top_id rel must pass.
	found := false
	for _, c := range report.Cases {
		if c.ID == "rag-relevant-first" {
			found = true
			if !c.Pass {
				t.Fatal("rag-relevant-first must pass")
			}
		}
	}
	if !found {
		t.Fatal("missing rag-relevant-first case result")
	}
}

func TestWriteReportRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	r := Report{
		Version: "1", Suite: "t", Passed: true, Total: 1, PassedCount: 1,
		Cases: []CaseResult{{ID: "a", Type: "string_contains", Pass: true, Score: 1.0}},
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
