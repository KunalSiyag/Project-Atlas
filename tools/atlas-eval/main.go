// Command atlas-eval runs deterministic offline evaluation fixtures.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("atlas-eval", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	dataset := fs.String("dataset", "", "path to v1 fixture JSON")
	out := fs.String("out", "eval-report.json", "report output path")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*dataset) == "" {
		fmt.Fprintln(os.Stderr, "usage: atlas-eval -dataset fixtures/eval/v1/sample.json -out report.json")
		return 2
	}
	ds, err := LoadDataset(*dataset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		return 1
	}
	report := RunDataset(ds)
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
