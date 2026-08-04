package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestToolSchemaValidation(t *testing.T) {
	reg := DefaultToolRegistry()
	_, err := reg.Invoke(context.Background(), "echo", map[string]any{}, InvokeOptions{Approved: true})
	if err == nil {
		t.Fatal("expected schema error")
	}
	var te *ToolError
	if !errors.As(err, &te) || te.Class != ToolErrInvalidInput {
		t.Fatalf("want invalid_input ToolError, got %v", err)
	}
	out, err := reg.Invoke(context.Background(), "echo", map[string]any{"text": "hi"}, InvokeOptions{Approved: true})
	if err != nil || out != "hi" {
		t.Fatalf("%q %v", out, err)
	}
}

func TestRegistryInvokeCalculator(t *testing.T) {
	reg := DefaultToolRegistry()
	// Drive calculator only through shipped registry path (no reimplemented math).
	out, err := reg.Invoke(context.Background(), "calculator", map[string]any{"expression": "21+21"}, InvokeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if out != "42" {
		t.Fatalf("got %q want 42 from registry calculator", out)
	}
	// List/register surface
	names := map[string]bool{}
	for _, tool := range reg.List() {
		names[tool.Name()] = true
		if tool.Description() == "" {
			t.Fatalf("tool %s missing description", tool.Name())
		}
		if tool.InputSchema() == nil {
			t.Fatalf("tool %s missing schema", tool.Name())
		}
	}
	for _, n := range []string{"echo", "calculator", "http_get", "danger_echo"} {
		if !names[n] {
			t.Fatalf("missing built-in %s in List()", n)
		}
	}
}

func TestDangerRequiresApprovalViaRegistry(t *testing.T) {
	reg := DefaultToolRegistry()
	tool, ok := reg.Get("danger_echo")
	if !ok || !tool.RequiresApproval() {
		t.Fatal("danger_echo must require approval")
	}
	_, err := reg.Invoke(context.Background(), "danger_echo", map[string]any{"text": "x"}, InvokeOptions{Approved: false})
	if err == nil {
		t.Fatal("expected denied")
	}
	var te *ToolError
	if !errors.As(err, &te) || te.Class != ToolErrDenied {
		t.Fatalf("want denied, got %v", err)
	}
	out, err := reg.Invoke(context.Background(), "danger_echo", map[string]any{"text": "ship"}, InvokeOptions{Approved: true})
	if err != nil || !strings.Contains(out, "DANGER_OK:ship") {
		t.Fatalf("%q %v", out, err)
	}
}

func TestCalculatorAndTimeout(t *testing.T) {
	out, err := RunTool(context.Background(), CalculatorTool{}, map[string]any{"expression": "2+3"})
	if err != nil || out != "5" {
		t.Fatalf("%q %v", out, err)
	}
	reg := NewToolRegistry(slowTool{})
	_, err = reg.Invoke(context.Background(), "slow", map[string]any{"text": "x"}, InvokeOptions{Approved: true})
	if err == nil {
		t.Fatal("expected timeout")
	}
	var te *ToolError
	if !errors.As(err, &te) || te.Class != ToolErrTimeout {
		t.Fatalf("want timeout class, got %v", err)
	}
}

type slowTool struct{}

func (slowTool) Name() string                   { return "slow" }
func (slowTool) Description() string            { return "slow test tool" }
func (slowTool) RequiresApproval() bool         { return false }
func (slowTool) Timeout() time.Duration         { return 20 * time.Millisecond }
func (slowTool) InputSchema() map[string]any    { return map[string]any{} }
func (slowTool) Execute(ctx context.Context, _ map[string]any) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(500 * time.Millisecond):
		return "late", nil
	}
}

func TestToolStepAndApprovalFlow(t *testing.T) {
	gw := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":  map[string]string{"role": "assistant", "content": "fake response to: approved-run"},
			"provider": "fake", "model": "fake-model",
		})
	}))
	defer gw.Close()

	svc := NewService(ServiceConfig{
		Store:   NewMemoryStore(),
		Gateway: NewHTTPGatewayClient(gw.URL, "fake-model"),
		Tools:   DefaultToolRegistry(),
		Planner: NewRulePlanner(),
		IDFunc:  func() string { return "run-danger" },
	})
	ctx := context.Background()
	created, err := svc.CreateRun(ctx, "please danger echo this payload")
	if err != nil {
		t.Fatal(err)
	}
	hasDanger := false
	for _, s := range created.Steps {
		if s.Kind == StepKindTool && s.ToolName == "danger_echo" {
			hasDanger = true
		}
	}
	if !hasDanger {
		t.Fatalf("expected danger tool in plan: %+v", created.Steps)
	}

	run, err := svc.StartRun(ctx, created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusWaitingApproval {
		t.Fatalf("want waiting_approval, got %s (%s)", run.Status, run.Error)
	}

	run, err = svc.ApproveRun(ctx, created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusSucceeded {
		t.Fatalf("after approve want succeeded, got %s %s steps=%+v", run.Status, run.Error, run.Steps)
	}
	found := false
	for _, s := range run.Steps {
		if s.ToolName == "danger_echo" && strings.Contains(s.Output, "DANGER_OK:") {
			found = true
		}
	}
	if !found {
		t.Fatalf("danger tool output missing: %+v", run.Steps)
	}
}

func TestRejectApproval(t *testing.T) {
	svc := NewService(ServiceConfig{
		Store:  NewMemoryStore(),
		Tools:  DefaultToolRegistry(),
		IDFunc: func() string { return "run-rej" },
	})
	ctx := context.Background()
	run, err := svc.CreateRun(ctx, "danger only")
	if err != nil {
		t.Fatal(err)
	}
	run.Steps = []Step{
		{StepID: "prepare", Name: "p", Kind: StepKindContext, Status: StepPending, MaxAttempts: 1},
		{StepID: "d", Name: "d", Kind: StepKindTool, DependsOn: []string{"prepare"}, Status: StepPending, MaxAttempts: 1,
			ToolName: "danger_echo", ToolInput: map[string]any{"text": "x"}, RequiresApproval: true},
	}
	_ = svc.Store.SaveRun(ctx, run)
	run, err = svc.StartRun(ctx, "run-rej")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusWaitingApproval {
		t.Fatalf("%s", run.Status)
	}
	run, err = svc.RejectRun(ctx, "run-rej", "nope")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusFailed || run.Error != "nope" {
		t.Fatalf("%+v", run)
	}
}

func TestHTTPGetTool(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pong"))
	}))
	defer srv.Close()
	reg := NewToolRegistry(HTTPGetTool{Client: srv.Client()})
	out, err := reg.Invoke(context.Background(), "http_get", map[string]any{"url": srv.URL}, InvokeOptions{})
	if err != nil || !strings.Contains(out, "pong") {
		t.Fatalf("%q %v", out, err)
	}
}
