package inference

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOllamaProviderMapsChatResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/chat" || request.Method != http.MethodPost {
			t.Fatalf("request = %s %s, want POST /api/chat", request.Method, request.URL.Path)
		}
		var input ollamaChatRequest
		if err := json.NewDecoder(request.Body).Decode(&input); err != nil {
			t.Fatal(err)
		}
		if input.Model != "gemma3" || input.Stream {
			t.Fatalf("request = %+v, want gemma3 and stream=false", input)
		}

		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"model":"gemma3","message":{"role":"assistant","content":"blue"},"done":true,"done_reason":"stop","prompt_eval_count":4,"eval_count":2}`))
	}))
	defer server.Close()

	result, err := (OllamaProvider{BaseURL: server.URL}).Generate(context.Background(), ChatRequest{
		Model:    "gemma3",
		Messages: []Message{{Role: "user", Content: "Why blue?"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "ollama" || result.Message.Content != "blue" {
		t.Fatalf("result = %+v, want Ollama response", result)
	}
	if result.Usage.InputTokens != 4 || result.Usage.OutputTokens != 2 {
		t.Fatalf("usage = %+v, want 4/2", result.Usage)
	}
}

func TestOllamaProviderReturnsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		http.Error(writer, "model unavailable", http.StatusNotFound)
	}))
	defer server.Close()

	_, err := (OllamaProvider{BaseURL: server.URL}).Generate(context.Background(), ChatRequest{Model: "missing"})
	if err == nil {
		t.Fatal("Generate() error = nil, want HTTP error")
	}
}
