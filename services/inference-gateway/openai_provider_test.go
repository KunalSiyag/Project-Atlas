package inference

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIProviderGenerateSuccess(t *testing.T) {
	var gotAuth string
	var gotPath string
	var gotBody openAIChatRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "gpt-4o-mini",
			"choices": []map[string]any{
				{
					"message":       map[string]string{"role": "assistant", "content": "hello from openai-style"},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 5},
		})
	}))
	defer server.Close()

	p := OpenAIProvider{
		BaseURL: server.URL + "/v1",
		APIKey:  "sk-test",
		Client:  server.Client(),
	}
	result, err := p.Generate(t.Context(), ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "hello from openai-style" {
		t.Fatalf("content = %q", result.Message.Content)
	}
	if result.Provider != "openai" {
		t.Fatalf("provider = %q", result.Provider)
	}
	if result.Usage.OutputTokens != 5 {
		t.Fatalf("usage = %+v", result.Usage)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if gotPath != "/v1/chat/completions" {
		t.Fatalf("path = %q", gotPath)
	}
	if len(gotBody.Messages) != 1 || gotBody.Messages[0].Content != "hi" {
		t.Fatalf("request body = %+v", gotBody)
	}
}

func TestOpenAIProviderAuthFailureClass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{"message": "invalid api key", "type": "invalid_request_error"},
		})
	}))
	defer server.Close()

	p := OpenAIProvider{BaseURL: server.URL, Client: server.Client()}
	_, err := p.Generate(t.Context(), ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	pe, ok := err.(*ProviderError)
	if !ok {
		t.Fatalf("type %T: %v", err, err)
	}
	if pe.Class != "auth" || pe.Retryable {
		t.Fatalf("provider error = %+v", pe)
	}
	if !strings.Contains(pe.Error(), "invalid api key") {
		t.Fatalf("message = %v", pe)
	}
}
