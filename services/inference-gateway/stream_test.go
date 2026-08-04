package inference

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChunkTextUTF8Safe(t *testing.T) {
	parts := chunkText("abcdefghij", 4)
	if got := strings.Join(parts, ""); got != "abcdefghij" {
		t.Fatalf("joined = %q", got)
	}
	if len(parts) != 3 {
		t.Fatalf("parts = %#v", parts)
	}
	// Multi-byte rune should not be split mid-rune.
	parts = chunkText("αβγδε", 2)
	joined := strings.Join(parts, "")
	if joined != "αβγδε" {
		t.Fatalf("joined unicode = %q", joined)
	}
}

func TestGatewayGenerateStreamSSE(t *testing.T) {
	server := httptest.NewServer(NewGateway(FakeProvider{ProviderName: "fake"}))
	defer server.Close()

	req, err := http.NewRequest(http.MethodPost, server.URL+"/v1/generate", strings.NewReader(
		`{"model":"fake-model","stream":true,"messages":[{"role":"user","content":"hello stream"}]}`,
	))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "stream-req-1")

	resp, err := server.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d body = %s", resp.StatusCode, body)
	}
	if !strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("content-type = %q", resp.Header.Get("Content-Type"))
	}
	if resp.Header.Get("X-Request-ID") != "stream-req-1" {
		t.Fatalf("request id = %q", resp.Header.Get("X-Request-ID"))
	}
	if resp.Header.Get("X-Run-ID") == "" {
		t.Fatal("expected X-Run-ID")
	}

	var deltas []string
	var done ChatResponse
	var sawDone bool
	scanner := bufio.NewScanner(resp.Body)
	var event string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			event = strings.TrimPrefix(line, "event: ")
			continue
		}
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			switch event {
			case "delta":
				var d StreamDelta
				if err := json.Unmarshal([]byte(payload), &d); err != nil {
					t.Fatalf("delta json: %v (%s)", err, payload)
				}
				deltas = append(deltas, d.Delta)
			case "done":
				if err := json.Unmarshal([]byte(payload), &done); err != nil {
					t.Fatalf("done json: %v", err)
				}
				sawDone = true
			case "error":
				t.Fatalf("unexpected error event: %s", payload)
			}
			event = ""
		}
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}
	if !sawDone {
		t.Fatal("expected done event")
	}
	joined := strings.Join(deltas, "")
	want := "fake response to: hello stream"
	if joined != want {
		t.Fatalf("deltas joined = %q, want %q (parts=%v)", joined, want, deltas)
	}
	if done.Message.Content != want {
		t.Fatalf("done content = %q", done.Message.Content)
	}
	if done.Provider != "fake" || done.Model != "fake-model" {
		t.Fatalf("done meta = %+v", done)
	}
	if len(deltas) < 2 {
		t.Fatalf("expected multiple deltas, got %d", len(deltas))
	}
}

func TestFakeProviderGenerateStream(t *testing.T) {
	var parts []string
	result, err := (FakeProvider{ProviderName: "fake"}).GenerateStream(context.Background(), ChatRequest{
		Model:    "m",
		Messages: []Message{{Role: "user", Content: "x"}},
	}, func(d StreamDelta) error {
		parts = append(parts, d.Delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(parts, "") != result.Message.Content {
		t.Fatalf("parts=%v result=%q", parts, result.Message.Content)
	}
}

func TestOllamaProviderGenerateStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input ollamaChatRequest
		_ = json.NewDecoder(r.Body).Decode(&input)
		if !input.Stream {
			t.Errorf("expected stream=true")
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"model":"gemma3","message":{"role":"assistant","content":"Hel"},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"model":"gemma3","message":{"role":"assistant","content":"lo"},"done":false}` + "\n"))
		_, _ = w.Write([]byte(`{"model":"gemma3","message":{"role":"assistant","content":""},"done":true,"done_reason":"stop","prompt_eval_count":2,"eval_count":2}` + "\n"))
	}))
	defer server.Close()

	var deltas []string
	result, err := (OllamaProvider{BaseURL: server.URL, Client: server.Client()}).GenerateStream(context.Background(), ChatRequest{
		Model:    "gemma3",
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, func(d StreamDelta) error {
		deltas = append(deltas, d.Delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "Hello" {
		t.Fatalf("content = %q", result.Message.Content)
	}
	if strings.Join(deltas, "") != "Hello" {
		t.Fatalf("deltas = %v", deltas)
	}
	if result.Usage.InputTokens != 2 || result.Usage.OutputTokens != 2 {
		t.Fatalf("usage = %+v", result.Usage)
	}
}

func TestOpenAIProviderGenerateStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input openAIChatRequest
		_ = json.NewDecoder(r.Body).Decode(&input)
		if !input.Stream {
			t.Errorf("expected stream=true")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"model\":\"gpt-4o-mini\",\"choices\":[{\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n")
		_, _ = io.WriteString(w, "data: {\"model\":\"gpt-4o-mini\",\"choices\":[{\"delta\":{\"content\":\"!\"},\"finish_reason\":\"stop\"}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	var deltas []string
	p := OpenAIProvider{BaseURL: server.URL + "/v1", Client: server.Client()}
	result, err := p.GenerateStream(context.Background(), ChatRequest{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: "user", Content: "yo"}},
	}, func(d StreamDelta) error {
		deltas = append(deltas, d.Delta)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Message.Content != "Hi!" {
		t.Fatalf("content = %q", result.Message.Content)
	}
	if strings.Join(deltas, "") != "Hi!" {
		t.Fatalf("deltas = %v", deltas)
	}
	if result.FinishReason != "stop" {
		t.Fatalf("finish = %q", result.FinishReason)
	}
}
