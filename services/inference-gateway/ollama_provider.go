package inference

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// OllamaProvider adapts Ollama's /api/chat endpoint to ModelProvider.
type OllamaProvider struct {
	BaseURL string
	Client  *http.Client
}

type ollamaChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ollamaChatResponse struct {
	Model        string  `json:"model"`
	Message      Message `json:"message"`
	Done         bool    `json:"done"`
	DoneReason   string  `json:"done_reason"`
	PromptTokens int     `json:"prompt_eval_count"`
	OutputTokens int     `json:"eval_count"`
}

func (p OllamaProvider) client() *http.Client {
	if p.Client == nil {
		return http.DefaultClient
	}
	return p.Client
}

func (p OllamaProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(p.BaseURL) == "" {
		return ChatResponse{}, fmt.Errorf("ollama base URL is required")
	}

	body, err := json.Marshal(ollamaChatRequest{
		Model:    request.Model,
		Messages: request.Messages,
		Stream:   false,
	})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal Ollama request: %w", err)
	}

	endpoint := strings.TrimRight(p.BaseURL, "/") + "/api/chat"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create Ollama request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := p.client().Do(httpRequest)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("call Ollama: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read Ollama response: %w", err)
	}
	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		class, retryable := classifyHTTPStatus(httpResponse.StatusCode)
		return ChatResponse{}, &ProviderError{
			Class:     class,
			Retryable: retryable,
			Err:       fmt.Errorf("Ollama returned HTTP %d: %s", httpResponse.StatusCode, strings.TrimSpace(string(responseBody))),
		}
	}

	var ollamaResponse ollamaChatResponse
	if err := json.Unmarshal(responseBody, &ollamaResponse); err != nil {
		return ChatResponse{}, fmt.Errorf("decode Ollama response: %w", err)
	}

	model := ollamaResponse.Model
	if model == "" {
		model = request.Model
	}
	return ChatResponse{
		Message:      ollamaResponse.Message,
		Usage:        Usage{InputTokens: ollamaResponse.PromptTokens, OutputTokens: ollamaResponse.OutputTokens},
		Provider:     "ollama",
		Model:        model,
		FinishReason: ollamaResponse.DoneReason,
	}, nil
}

// GenerateStream consumes Ollama NDJSON stream chunks and emits content deltas.
func (p OllamaProvider) GenerateStream(ctx context.Context, request ChatRequest, emit StreamEmit) (ChatResponse, error) {
	if strings.TrimSpace(p.BaseURL) == "" {
		return ChatResponse{}, fmt.Errorf("ollama base URL is required")
	}
	if strings.TrimSpace(request.Model) == "" {
		return ChatResponse{}, fmt.Errorf("model is required")
	}
	if len(request.Messages) == 0 {
		return ChatResponse{}, fmt.Errorf("at least one message is required")
	}

	body, err := json.Marshal(ollamaChatRequest{
		Model:    request.Model,
		Messages: request.Messages,
		Stream:   true,
	})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal Ollama stream request: %w", err)
	}

	endpoint := strings.TrimRight(p.BaseURL, "/") + "/api/chat"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create Ollama stream request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := p.client().Do(httpRequest)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("call Ollama stream: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(httpResponse.Body, 4096))
		class, retryable := classifyHTTPStatus(httpResponse.StatusCode)
		return ChatResponse{}, &ProviderError{
			Class:     class,
			Retryable: retryable,
			Err:       fmt.Errorf("Ollama returned HTTP %d: %s", httpResponse.StatusCode, strings.TrimSpace(string(responseBody))),
		}
	}

	var (
		content strings.Builder
		model   = request.Model
		usage   Usage
		finish  string
	)
	scanner := bufio.NewScanner(httpResponse.Body)
	// Ollama chunks are small; allow larger lines for safety.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return ChatResponse{}, err
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var chunk ollamaChatResponse
		if err := json.Unmarshal([]byte(line), &chunk); err != nil {
			return ChatResponse{}, fmt.Errorf("decode Ollama stream chunk: %w", err)
		}
		if chunk.Model != "" {
			model = chunk.Model
		}
		delta := chunk.Message.Content
		if delta != "" {
			content.WriteString(delta)
			if err := emit(StreamDelta{Delta: delta, Provider: "ollama", Model: model}); err != nil {
				return ChatResponse{}, err
			}
		}
		if chunk.Done {
			finish = chunk.DoneReason
			if finish == "" {
				finish = "stop"
			}
			usage = Usage{InputTokens: chunk.PromptTokens, OutputTokens: chunk.OutputTokens}
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatResponse{}, fmt.Errorf("read Ollama stream: %w", err)
	}
	if finish == "" {
		finish = "stop"
	}
	return ChatResponse{
		Message:      Message{Role: "assistant", Content: content.String()},
		Usage:        usage,
		Provider:     "ollama",
		Model:        model,
		FinishReason: finish,
	}, nil
}

func classifyHTTPStatus(status int) (string, bool) {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return "auth", false
	case http.StatusTooManyRequests:
		return "rate_limit", true
	case http.StatusRequestTimeout:
		return "timeout", true
	case http.StatusBadRequest:
		return "invalid_request", false
	default:
		if status >= http.StatusInternalServerError {
			return "unavailable", true
		}
		return "provider_error", false
	}
}
