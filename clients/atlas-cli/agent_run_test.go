package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentRunAgainstOrchestrator(t *testing.T) {
	var state struct {
		status string
		steps  []OrchStep
	}
	state.status = "pending"
	state.steps = []OrchStep{
		{StepID: "prepare", Name: "prepare", Kind: "context", Status: "pending"},
		{StepID: "infer", Name: "infer", Kind: "model", Status: "pending"},
		{StepID: "finalize", Name: "finalize", Kind: "finalize", Status: "pending"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/healthz":
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/runs":
			state.status = "pending"
			_ = json.NewEncoder(w).Encode(OrchRun{RunID: "r1", Goal: "g", Status: "pending", Steps: state.steps})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			state.status = "succeeded"
			state.steps = []OrchStep{
				{StepID: "prepare", Status: "succeeded", Kind: "context"},
				{StepID: "infer", Status: "succeeded", Kind: "model", Output: "fake response to: g"},
				{StepID: "finalize", Status: "succeeded", Kind: "finalize", Output: "fake response to: g"},
			}
			_ = json.NewEncoder(w).Encode(OrchRun{RunID: "r1", Goal: "g", Status: "succeeded", Steps: state.steps, Result: "fake response to: g"})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/runs/"):
			_ = json.NewEncoder(w).Encode(OrchRun{RunID: "r1", Goal: "g", Status: state.status, Steps: state.steps, Result: "fake response to: g"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	code := cmdAgentRun(t.Context(), NewOrchestratorClient(srv.URL), "calculate 1+1", false, false)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
}
