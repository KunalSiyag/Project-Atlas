package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"
)

// streamGenerate prefers a native StreamingModelProvider; otherwise synthesizes
// deltas from a full Generate result so every provider can serve stream=true.
func streamGenerate(ctx context.Context, provider ModelProvider, request ChatRequest, emit StreamEmit) (ChatResponse, error) {
	if provider == nil {
		return ChatResponse{}, fmt.Errorf("provider is not configured")
	}
	// Native streaming skips RetryingProvider retries (unsafe after the first delta).
	inner := unwrapProvider(provider)
	if sp, ok := inner.(StreamingModelProvider); ok {
		return sp.GenerateStream(ctx, request, emit)
	}
	// Full Generate keeps retries when provider is a RetryingProvider.
	return synthesizeStream(ctx, provider, request, emit)
}

func unwrapProvider(provider ModelProvider) ModelProvider {
	if retry, ok := provider.(RetryingProvider); ok && retry.Provider != nil {
		return retry.Provider
	}
	return provider
}

func synthesizeStream(ctx context.Context, provider ModelProvider, request ChatRequest, emit StreamEmit) (ChatResponse, error) {
	// Clear stream flag so non-streaming Generate implementations stay simple.
	request.Stream = false
	result, err := provider.Generate(ctx, request)
	if err != nil {
		return ChatResponse{}, err
	}
	for _, part := range chunkText(result.Message.Content, 12) {
		if err := ctx.Err(); err != nil {
			return ChatResponse{}, err
		}
		if err := emit(StreamDelta{Delta: part, Provider: result.Provider, Model: result.Model}); err != nil {
			return ChatResponse{}, err
		}
	}
	return result, nil
}

// chunkText splits s into UTF-8-safe pieces of at most maxRunes runes.
// Empty content yields no chunks (caller still emits the done event).
func chunkText(s string, maxRunes int) []string {
	if s == "" {
		return nil
	}
	if maxRunes < 1 {
		maxRunes = 1
	}
	var parts []string
	var b strings.Builder
	count := 0
	for _, r := range s {
		b.WriteRune(r)
		count++
		if count >= maxRunes {
			parts = append(parts, b.String())
			b.Reset()
			count = 0
		}
	}
	if b.Len() > 0 {
		parts = append(parts, b.String())
	}
	// Ensure we never drop incomplete runes (range already rune-safe).
	_ = utf8.ValidString(s)
	return parts
}

type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSEWriter(w http.ResponseWriter) (*sseWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("response writer does not support flushing")
	}
	return &sseWriter{w: w, flusher: flusher}, nil
}

func (s *sseWriter) begin() {
	h := s.w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")
	s.w.WriteHeader(http.StatusOK)
	s.flusher.Flush()
}

func (s *sseWriter) event(name string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", name, data); err != nil {
		return err
	}
	s.flusher.Flush()
	return nil
}
