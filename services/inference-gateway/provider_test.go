package inference

import (
	"context"
	"strings"
	"testing"
)

func TestFakeProviderNormalizesAResponse(t *testing.T) {
	provider := FakeProvider{ProviderName: "fake"}

	response, err := provider.Generate(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "Explain ports."}},
		Model:    "fake-model",
		Metadata: map[string]string{"run_id": "run-1"},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if response.Message.Role != "assistant" {
		t.Fatalf("role = %q, want assistant", response.Message.Role)
	}
	if !strings.Contains(response.Message.Content, "Explain ports.") {
		t.Fatalf("content = %q, want original user content", response.Message.Content)
	}
	if response.Provider != "fake" || response.Model != "fake-model" {
		t.Fatalf("provider/model = %q/%q, want fake/fake-model", response.Provider, response.Model)
	}
	if response.FinishReason != "stop" {
		t.Fatalf("finish reason = %q, want stop", response.FinishReason)
	}
}

func TestFakeProviderRejectsInvalidRequests(t *testing.T) {
	provider := FakeProvider{ProviderName: "fake"}
	cases := []ChatRequest{
		{Model: "fake-model"},
		{Messages: []Message{{Role: "user", Content: "hello"}}},
	}

	for _, request := range cases {
		if _, err := provider.Generate(context.Background(), request); err == nil {
			t.Fatalf("Generate(%+v) error = nil, want validation error", request)
		}
	}
}

func TestFakeProviderHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := (FakeProvider{ProviderName: "fake"}).Generate(ctx, ChatRequest{
		Messages: []Message{{Role: "user", Content: "hello"}},
		Model:    "fake-model",
	})
	if err != context.Canceled {
		t.Fatalf("error = %v, want context canceled", err)
	}
}
