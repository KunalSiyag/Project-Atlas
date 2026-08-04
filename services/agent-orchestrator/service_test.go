package orchestrator

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStartRunCallsGatewayAndSucceeds(t *testing.T) {
	var gotBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/generate" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		msgs, _ := gotBody["messages"].([]any)
		// echo last user-ish content via fake style
		content := "fake response to: goal-text"
		if len(msgs) > 0 {
			content = "fake response to: ship it"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": content},
			"provider": "fake",
			"model":    "fake-model",
		})
	}))
	defer server.Close()

	svc := NewService(ServiceConfig{
		Store:   NewMemoryStore(),
		Gateway: NewHTTPGatewayClient(server.URL, "fake-model"),
		Builder: NewContextBuilder(),
		Model:   "fake-model",
		IDFunc:  func() string { return "run-gw-1" },
	})
	ctx := context.Background()
	created, err := svc.CreateRun(ctx, "ship it")
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Steps) < 3 {
		t.Fatalf("expected real plan steps, got %+v", created.Steps)
	}
	// ensure deps exist
	if len(created.Steps[1].DependsOn) == 0 {
		t.Fatal("infer step should depend on prepare")
	}

	run, err := svc.StartRun(ctx, created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusSucceeded {
		t.Fatalf("status=%s error=%s steps=%+v", run.Status, run.Error, run.Steps)
	}
	if !strings.Contains(run.Result, "fake response to:") {
		t.Fatalf("result=%q", run.Result)
	}
	if gotBody == nil {
		t.Fatal("gateway was not called")
	}
}

func TestStartRunGatewayFailureMarksFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"code":"provider_error","message":"boom"}}`))
	}))
	defer server.Close()

	svc := NewService(ServiceConfig{
		Store:   NewMemoryStore(),
		Gateway: NewHTTPGatewayClient(server.URL, "fake-model"),
		IDFunc:  func() string { return "run-fail-1" },
	})
	// only 1 attempt on infer for faster fail — mutate after create
	ctx := context.Background()
	created, err := svc.CreateRun(ctx, "will fail")
	if err != nil {
		t.Fatal(err)
	}
	// lower retries
	created.Steps[1].MaxAttempts = 1
	_ = svc.Store.SaveRun(ctx, created)

	run, err := svc.StartRun(ctx, created.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusFailed {
		t.Fatalf("want failed got %s (%s)", run.Status, run.Error)
	}
	if run.Error == "" {
		t.Fatal("expected error message")
	}
}

func TestPlanDependencyOrderAndRetry(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("busy"))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": "fake response to: retry-goal"},
			"provider": "fake", "model": "fake-model",
		})
	}))
	defer server.Close()

	store := NewMemoryStore()
	svc := NewService(ServiceConfig{
		Store:   store,
		Gateway: NewHTTPGatewayClient(server.URL, "fake-model"),
		IDFunc:  func() string { return "run-retry" },
	})
	ctx := context.Background()
	run, err := svc.CreateRun(ctx, "retry-goal")
	if err != nil {
		t.Fatal(err)
	}
	// infer has MaxAttempts=2 by default
	run, err = svc.StartRun(ctx, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusSucceeded {
		t.Fatalf("%s %s", run.Status, run.Error)
	}
	if calls.Load() < 2 {
		t.Fatalf("expected retry, calls=%d", calls.Load())
	}
	// dependency: prepare succeeded before infer
	var prepareDone, inferDone time.Time
	// use events order
	events, _ := store.ListEvents(ctx, "run-retry")
	prepareOK, inferStart := false, false
	for _, e := range events {
		if e.StepID == "prepare" && e.Type == "step_succeeded" {
			prepareOK = true
		}
		if e.StepID == "infer" && e.Type == "step_started" {
			if !prepareOK {
				t.Fatal("infer started before prepare succeeded")
			}
			inferStart = true
		}
	}
	if !inferStart {
		t.Fatal("infer never started")
	}
	_ = prepareDone
	_ = inferDone
}

func TestCancelPreventsLaterSteps(t *testing.T) {
	// Gateway blocks until cancelled path — use a step that never gets to model if we cancel while pending after prepare.
	// Simpler: cancel while pending before start → cancelled immediately.
	svc := NewService(ServiceConfig{
		Store:  NewMemoryStore(),
		IDFunc: func() string { return "run-cancel" },
	})
	ctx := context.Background()
	run, err := svc.CreateRun(ctx, "cancel me")
	if err != nil {
		t.Fatal(err)
	}
	run, err = svc.CancelRun(ctx, run.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusCancelled {
		t.Fatalf("status=%s", run.Status)
	}
	for _, st := range run.Steps {
		if st.Status != StepCancelled {
			t.Fatalf("step %s status %s", st.StepID, st.Status)
		}
	}
}

func TestCancelDuringExecution(t *testing.T) {
	// Inject a gateway that cancels the run on first generate attempt via side channel store.
	store := NewMemoryStore()
	var svc *Service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request cancel mid-flight
		_, _ = svc.CancelRun(context.Background(), "run-mid")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message": map[string]string{"role": "assistant", "content": "should-not-matter-if-cancelled-after"},
			"provider": "fake", "model": "m",
		})
	}))
	defer server.Close()
	svc = NewService(ServiceConfig{
		Store:   store,
		Gateway: NewHTTPGatewayClient(server.URL, "m"),
		IDFunc:  func() string { return "run-mid" },
	})
	ctx := context.Background()
	_, err := svc.CreateRun(ctx, "mid cancel")
	if err != nil {
		t.Fatal(err)
	}
	run, err := svc.StartRun(ctx, "run-mid")
	if err != nil {
		t.Fatal(err)
	}
	// After cancel requested during infer, either cancelled or succeeded depending on timing.
	// We assert cancel flag path works: if still running steps after cancel, finalize may skip.
	if run.Status != StatusCancelled && run.Status != StatusSucceeded {
		t.Fatalf("unexpected status %s err=%s", run.Status, run.Error)
	}
	// Stronger cancel test: executor checks cancel between steps — cancel after prepare by custom executor hooks.
	// Use blocking approach: custom InferenceClient
	client := &cancelClient{store: store, runID: "run-block", cancelAfter: 0}
	svc2 := NewService(ServiceConfig{
		Store:   store,
		Gateway: client,
		IDFunc:  func() string { return "run-block" },
	})
	// fresh store
	store2 := NewMemoryStore()
	client.store = store2
	svc2 = NewService(ServiceConfig{
		Store:   store2,
		Gateway: client,
		IDFunc:  func() string { return "run-block" },
	})
	client.svc = svc2
	_, _ = svc2.CreateRun(ctx, "block")
	run, err = svc2.StartRun(ctx, "run-block")
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusCancelled {
		t.Fatalf("want cancelled, got %s (%s) steps=%+v", run.Status, run.Error, run.Steps)
	}
	// finalize should not succeed
	for _, st := range run.Steps {
		if st.StepID == "finalize" && st.Status == StepSucceeded {
			t.Fatal("finalize should not succeed after cancel")
		}
	}
}

type cancelClient struct {
	store       RunStore
	svc         *Service
	runID       string
	cancelAfter int
	n           int
}

func (c *cancelClient) Generate(ctx context.Context, req GenerateRequest) (GenerateResponse, error) {
	c.n++
	// Request cancel after model step so finalize is skipped on the next executor loop.
	if c.svc != nil {
		_, _ = c.svc.CancelRun(ctx, c.runID)
	}
	var r GenerateResponse
	r.Message.Role = "assistant"
	r.Message.Content = "fake response to: cancelled-mid"
	r.Provider = "fake"
	return r, nil
}

func TestHTTPStartPath(t *testing.T) {
	serverGW := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"message":  map[string]string{"role": "assistant", "content": "fake response to: http-goal"},
			"provider": "fake", "model": "fake-model",
		})
	}))
	defer serverGW.Close()

	svc := NewService(ServiceConfig{
		Store:   NewMemoryStore(),
		Gateway: NewHTTPGatewayClient(serverGW.URL, "fake-model"),
		IDFunc:  func() string { return "http-1" },
	})
	api := httptest.NewServer(&Handler{Service: svc})
	defer api.Close()

	resp, err := api.Client().Post(api.URL+"/v1/runs", "application/json", strings.NewReader(`{"goal":"http-goal"}`))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create %d", resp.StatusCode)
	}
	resp, err = api.Client().Post(api.URL+"/v1/runs/http-1/start", "application/json", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("start %d", resp.StatusCode)
	}
	var run Run
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Status != StatusSucceeded || !strings.Contains(run.Result, "fake response to:") {
		t.Fatalf("%+v", run)
	}
}
