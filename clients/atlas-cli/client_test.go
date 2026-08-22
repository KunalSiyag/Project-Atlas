package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// gatewayishServer mimics the real inference-gateway JSON shapes the CLI depends on.
func gatewayishServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ok"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/readyz":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"status":"ready"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/generate":
			var req GenerateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if len(req.Messages) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"error": map[string]string{"code": "invalid_request", "message": "messages are required"},
				})
				return
			}
			last := req.Messages[len(req.Messages)-1].Content
			w.Header().Set("X-Request-ID", "req-test")
			w.Header().Set("X-Run-ID", "run-test")
			_ = json.NewEncoder(w).Encode(ChatResponse{
				Message:  Message{Role: "assistant", Content: "fake response to: " + last},
				Provider: "fake",
				Model:    req.Model,
				Usage:    Usage{InputTokens: len(req.Messages), OutputTokens: 1},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/conversations":
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(Conversation{
				ConversationID: "conv-1",
				Title:          "test",
				Status:         "open",
			})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/conversations/"):
			_ = json.NewEncoder(w).Encode(Conversation{
				ConversationID: "conv-1",
				Status:         "open",
				Messages: []ConversationMessage{
					{Role: "user", Content: "hi"},
					{Role: "assistant", Content: "fake response to: hi"},
				},
			})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/messages"):
			var body struct {
				Content string `json:"content"`
				Model   string `json:"model"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("X-Run-ID", "run-msg")
			_ = json.NewEncoder(w).Encode(ChatResponse{
				Message:  Message{Role: "assistant", Content: "fake response to: " + body.Content},
				Provider: "fake",
				Model:    body.Model,
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/fail":
			// unused
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]string{"code": "not_found", "message": "route not found"},
			})
		}
	}))
}

func TestClientGenerateReflectsUserMessage(t *testing.T) {
	server := gatewayishServer(t)
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.Generate(t.Context(), GenerateRequest{
		Messages: []Message{{Role: "user", Content: "planets"}},
		Model:    "fake-model",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "fake response to: planets" {
		t.Fatalf("content = %q", result.Message.Content)
	}
	if result.Provider != "fake" {
		t.Fatalf("provider = %q", result.Provider)
	}
	if result.RunID != "run-test" || result.RequestID != "req-test" {
		t.Fatalf("ids run=%q req=%q", result.RunID, result.RequestID)
	}
}

func TestClientHealthAndReady(t *testing.T) {
	server := gatewayishServer(t)
	defer server.Close()
	client := NewClient(server.URL)

	health, err := client.Health(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(health, "ok") {
		t.Fatalf("health = %s", health)
	}
	ready, err := client.Ready(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ready, "ready") {
		t.Fatalf("ready = %s", ready)
	}
}

func TestClientConversationRoundTrip(t *testing.T) {
	server := gatewayishServer(t)
	defer server.Close()
	client := NewClient(server.URL)

	conv, err := client.CreateConversation(t.Context(), "lesson")
	if err != nil {
		t.Fatal(err)
	}
	if conv.ConversationID != "conv-1" {
		t.Fatalf("id = %q", conv.ConversationID)
	}

	result, err := client.PostMessage(t.Context(), conv.ConversationID, "hello orbit", "fake", "", "fake-model", "act")
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "fake response to: hello orbit" {
		t.Fatalf("content = %q", result.Message.Content)
	}
	if result.RunID != "run-msg" {
		t.Fatalf("run = %q", result.RunID)
	}

	got, err := client.GetConversation(t.Context(), conv.ConversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Messages) < 2 {
		t.Fatalf("messages = %+v", got.Messages)
	}
}

func TestClientGetRunAndOpenAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/runs/"):
			_ = json.NewEncoder(w).Encode(InferenceRunView{
				RunID:         "run-99",
				Status:        "succeeded",
				ResolvedModel: "fake-model",
				Provider:      "fake",
				Result: &ChatResponse{
					Message: Message{Role: "assistant", Content: "stored answer"},
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/openapi.yaml":
			w.Header().Set("Content-Type", "application/yaml")
			_, _ = w.Write([]byte("openapi: 3.0.3\ninfo:\n  title: Atlas Inference Gateway\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL)
	view, err := client.GetRun(t.Context(), "run-99")
	if err != nil {
		t.Fatal(err)
	}
	if view.Status != "succeeded" || view.Result == nil || view.Result.Message.Content != "stored answer" {
		t.Fatalf("view = %+v", view)
	}
	doc, err := client.OpenAPI(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(doc, "Atlas Inference Gateway") {
		t.Fatalf("doc = %q", doc)
	}
}

func TestClientSurfacesNestedAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"code": "invalid_request", "message": "messages are required"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.Generate(t.Context(), GenerateRequest{
		Messages: []Message{{Role: "user", Content: "x"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "invalid_request") || !strings.Contains(err.Error(), "messages are required") {
		t.Fatalf("error = %v", err)
	}
}

func TestCmdGenerateUsesClient(t *testing.T) {
	server := gatewayishServer(t)
	defer server.Close()
	client := NewClient(server.URL)
	code := cmdGenerate(t.Context(), client, "auto", "", "fake-model", "from-cmd", false, false)
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
}

func TestCmdHealthUsesClient(t *testing.T) {
	server := gatewayishServer(t)
	defer server.Close()
	client := NewClient(server.URL)
	if code := cmdHealth(t.Context(), client, false); code != 0 {
		t.Fatalf("exit = %d", code)
	}
}

func TestBannerContainsAtlas(t *testing.T) {
	var b strings.Builder
	printBanner(&b, "http://localhost:8080", "fake-model", false)
	out := b.String()
	if !strings.Contains(out, "█████╗") || !strings.Contains(out, "gateway") {
		t.Fatalf("banner missing expected pieces:\n%s", out)
	}
	// The wordmark is drawn as block letters; ensure A T L A S pattern fragments exist.
	if !strings.Contains(atlasBanner, "██╗") {
		t.Fatal("atlasBanner constant incomplete")
	}
	if !strings.Contains(out, "http://localhost:8080") {
		t.Fatalf("banner missing URL:\n%s", out)
	}
}

func TestRunHelpAndVersion(t *testing.T) {
	if code := run([]string{"version"}); code != 0 {
		t.Fatalf("version exit %d", code)
	}
	if code := run([]string{"help"}); code != 0 {
		t.Fatalf("help exit %d", code)
	}
}
