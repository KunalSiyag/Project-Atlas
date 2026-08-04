// Command atlas-eval runs deterministic offline evaluation fixtures.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	orchestrator "github.com/KunalSiyag/Project-Atlas/services/agent-orchestrator"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("atlas-eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataset := fs.String("dataset", "", "path to v1 fixture JSON")
	out := fs.String("out", "eval-report.json", "report output path")
	judgeStub := fs.String("judge-stub", "", "optional static judge: pass|fail (for llm_judge cases without a live model)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*dataset) == "" {
		fmt.Fprintln(os.Stderr, "usage: atlas-eval -dataset fixtures/eval/v1/sample.json -out report.json [-judge-stub pass]")
		return 2
	}
	ds, err := LoadDataset(*dataset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		return 1
	}
	opts := Options{}
	switch strings.ToLower(strings.TrimSpace(*judgeStub)) {
	case "pass":
		opts.Judge = orchestrator.StaticJudge{Pass: true, Score: 1.0, Rationale: "stub-pass"}
	case "fail":
		opts.Judge = orchestrator.StaticJudge{Pass: false, Score: 0.0, Rationale: "stub-fail"}
	case "":
		// no judge
	default:
		fmt.Fprintln(os.Stderr, "invalid -judge-stub (use pass|fail)")
		return 2
	}
	report := RunDatasetWithOptions(ds, opts)
	if err := WriteReport(*out, report); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		return 1
	}
	summary, _ := json.Marshal(map[string]any{
		"suite": report.Suite, "passed": report.Passed,
		"passed_count": report.PassedCount, "failed_count": report.FailedCount, "out": *out,
	})
	fmt.Println(string(summary))
	if !report.Passed {
		return 1
	}
	return 0
}
