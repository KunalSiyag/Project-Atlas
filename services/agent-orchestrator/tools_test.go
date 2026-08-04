package orchestrator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestToolSchemaValidation(t *testing.T) {
	_, err := RunTool(context.Background(), EchoTool{}, map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("expected schema error, got %v", err)
	}
	out, err := RunTool(context.Background(), EchoTool{}, map[string]any{"text": "hi"})
	if err != nil || out != "hi" {
		t.Fatalf("%q %v", out, err)
	}
}

func TestCalculatorAndTimeout(t *testing.T) {
	out, err := RunTool(context.Background(), CalculatorTool{}, map[string]any{"expression": "2+3"})
	if err != nil || out != "5" {
		t.Fatalf("%q %v", out, err)
	}
	slow := slowTool{}
	_, err = RunTool(context.Background(), slow, map[string]any{"text": "x"})
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("expected timeout, got %v", err)
	}
}

type slowTool struct{}

func (slowTool) Name() string                            { return "slow" }
func (slowTool) Dangerous() bool                         { return false }
func (slowTool) Timeout() time.Duration                  { return 20 * time.Millisecond }
func (slowTool) Schema() map[string]any                  { return map[string]any{} }
func (slowTool) Invoke(ctx context.Context, _ map[string]any) (string, error) {
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
	// Goal triggers danger tool
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
	// danger tool should have produced output
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
	// Force a plan with only danger tool + finalize without gateway for reject path before model
	// Use create then mutate steps
	run, err := svc.CreateRun(ctx, "danger only")
	if err != nil {
		t.Fatal(err)
	}
	// Start may fail if no gateway on model — set plan to only prepare + danger
	run.Steps = []Step{
		{StepID: "prepare", Name: "p", Kind: StepKindContext, Status: StepPending, MaxAttempts: 1},
		{StepID: "d", Name: "d", Kind: StepKindTool, DependsOn: []string{"prepare"}, Status: StepPending, MaxAttempts: 1,
			ToolName: "danger_echo", ToolInput: map[string]any{"text": "x"}, RequiresApproval: true},
	}
	_ = svc.Store.SaveRun(ctx, run)
	// Need gateway-less prepare - context doesn't need gateway
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
	out, err := RunTool(context.Background(), HTTPGetTool{Client: srv.Client()}, map[string]any{"url": srv.URL})
	if err != nil || !strings.Contains(out, "pong") {
		t.Fatalf("%q %v", out, err)
	}
}
