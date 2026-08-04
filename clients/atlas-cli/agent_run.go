package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// cmdAgentRun creates an orchestrator run, starts it, polls until terminal, prints steps.
func cmdAgentRun(ctx context.Context, orch *OrchestratorClient, goal string, color bool, autoApprove bool) int {
	if err := orch.Health(ctx); err != nil {
		fmt.Fprintln(os.Stderr, colorize(color, red, "error: "+err.Error()))
		fmt.Fprintln(os.Stderr, colorize(color, dim, "  start orchestrator: make orch-run"))
		return 1
	}
	fmt.Fprintln(os.Stdout, colorize(color, bold+cyan, "  ATLAS  agent run"))
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "goal"), goal)
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "orchestrator"), orch.BaseURL)

	created, err := orch.CreateRun(ctx, goal)
	if err != nil {
		fmt.Fprintln(os.Stderr, colorize(color, red, "error create: "+err.Error()))
		return 1
	}
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "run_id"), created.RunID)
	printOrchSteps(created, color)

	run, err := orch.StartRun(ctx, created.RunID)
	if err != nil {
		fmt.Fprintln(os.Stderr, colorize(color, red, "error start: "+err.Error()))
		return 1
	}

	// Poll for waiting_approval / terminal (start is often sync, but keep poll for resume).
	deadline := time.Now().Add(3 * time.Minute)
	for {
		printOrchSteps(run, color)
		switch run.Status {
		case "succeeded":
			fmt.Fprintln(os.Stdout, colorize(color, green, "  ✓  succeeded"))
			if run.Result != "" {
				fmt.Fprintln(os.Stdout, colorize(color, bold+magenta, "  result ›"))
				fmt.Fprintln(os.Stdout, "  "+run.Result)
			}
			return 0
		case "failed", "cancelled":
			fmt.Fprintln(os.Stdout, colorize(color, red, "  ✗  "+run.Status+": "+run.Error))
			return 1
		case "waiting_approval":
			fmt.Fprintln(os.Stdout, colorize(color, yellow, "  ⏳ waiting_approval"))
			if autoApprove {
				fmt.Fprintln(os.Stdout, colorize(color, dim, "  auto-approving…"))
				run, err = orch.ApproveRun(ctx, run.RunID)
				if err != nil {
					fmt.Fprintln(os.Stderr, colorize(color, red, "error approve: "+err.Error()))
					return 1
				}
				continue
			}
			fmt.Fprintln(os.Stdout, colorize(color, dim, "  approve: curl -X POST "+orch.BaseURL+"/v1/runs/"+run.RunID+"/approve"))
			// keep polling in case external approve happens
		}
		if time.Now().After(deadline) {
			fmt.Fprintln(os.Stderr, colorize(color, red, "error: timed out waiting for run"))
			return 1
		}
		select {
		case <-ctx.Done():
			return 1
		case <-time.After(400 * time.Millisecond):
		}
		run, err = orch.GetRun(ctx, created.RunID)
		if err != nil {
			fmt.Fprintln(os.Stderr, colorize(color, red, "error poll: "+err.Error()))
			return 1
		}
	}
}

func printOrchSteps(run OrchRun, color bool) {
	fmt.Fprintf(os.Stdout, "  %s  %s\n", colorize(color, dim, "status"), colorize(color, statusColor(color, run.Status), run.Status))
	for _, s := range run.Steps {
		line := fmt.Sprintf("    • %-12s %-10s %s", s.StepID, s.Status, s.Kind)
		if s.ToolName != "" {
			line += " tool=" + s.ToolName
		}
		if s.Error != "" {
			line += " err=" + s.Error
		}
		fmt.Fprintln(os.Stdout, colorize(color, dim, line))
		if s.Output != "" && (s.Status == "succeeded" || strings.HasPrefix(s.Output, "DANGER")) {
			out := s.Output
			if len(out) > 120 {
				out = out[:117] + "..."
			}
			fmt.Fprintln(os.Stdout, colorize(color, dim, "      → "+out))
		}
	}
}
