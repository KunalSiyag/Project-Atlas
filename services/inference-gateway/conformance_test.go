package inference

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Provider conformance: same ChatRequest/ChatResponse contract for fake and OpenAI-style adapters.
func TestChatProviderConformance(t *testing.T) {
	openAIServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req openAIChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		last := ""
		if len(req.Messages) > 0 {
			last = req.Messages[len(req.Messages)-1].Content
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": req.Model,
			"choices": []map[string]any{
				{"message": map[string]string{"role": "assistant", "content": "echo:" + last}, "finish_reason": "stop"},
			},
			"usage": map[string]int{"prompt_tokens": 1, "completion_tokens": 1},
		})
	}))
	defer openAIServer.Close()

	cases := []struct {
		name     string
		provider ModelProvider
		model    string
	}{
		{name: "fake", provider: FakeProvider{ProviderName: "fake"}, model: "fake-model"},
		{
			name: "openai-style",
			provider: OpenAIProvider{
				BaseURL: openAIServer.URL,
				APIKey:  "test",
				Client:  openAIServer.Client(),
			},
			model: "gpt-test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name+"/success", func(t *testing.T) {
			// Use factory-equivalent construction for fake; OpenAI uses same struct production uses.
			result, err := tc.provider.Generate(t.Context(), ChatRequest{
				Model:    tc.model,
				Messages: []Message{{Role: "user", Content: "conformance"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if result.Message.Content == "" {
				t.Fatal("empty assistant content")
			}
			if result.Provider == "" {
				t.Fatal("empty provider")
			}
			if result.Model == "" {
				t.Fatal("empty model")
			}
			// fake prefixes; openai-style mock echoes
			if tc.name == "fake" && result.Message.Content != "fake response to: conformance" {
				t.Fatalf("fake content = %q", result.Message.Content)
			}
			if tc.name == "openai-style" && result.Message.Content != "echo:conformance" {
				t.Fatalf("openai content = %q", result.Message.Content)
			}
		})

		t.Run(tc.name+"/empty_messages_fails", func(t *testing.T) {
			_, err := tc.provider.Generate(t.Context(), ChatRequest{Model: tc.model, Messages: nil})
			if err == nil {
				t.Fatal("expected error for empty messages")
			}
		})
	}

	t.Run("openai-style/rate_limit_class", func(t *testing.T) {
		limited := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"slow down"}}`))
		}))
		defer limited.Close()
		p := OpenAIProvider{BaseURL: limited.URL, Client: limited.Client()}
		_, err := p.Generate(t.Context(), ChatRequest{
			Model:    "gpt-test",
			Messages: []Message{{Role: "user", Content: "x"}},
		})
		pe, ok := err.(*ProviderError)
		if !ok || pe.Class != "rate_limit" || !pe.Retryable {
			t.Fatalf("got %v (%T)", err, err)
		}
	})
}

func TestConfiguredProviderFactoryIncludesOpenAI(t *testing.T) {
	p, err := NewConfiguredProvider(Config{ProviderName: "openai", OpenAIBaseURL: "http://example", OpenAIAPIKey: "k"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := p.(OpenAIProvider); !ok {
		t.Fatalf("type %T", p)
	}
}
