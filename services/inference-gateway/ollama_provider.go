package inference

import (
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
	DoneReason   string  `json:"done_reason"`
	PromptTokens int     `json:"prompt_eval_count"`
	OutputTokens int     `json:"eval_count"`
}

func (p OllamaProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(p.BaseURL) == "" {
		return ChatResponse{}, fmt.Errorf("ollama base URL is required")
	}
	if p.Client == nil {
		p.Client = http.DefaultClient
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

	httpResponse, err := p.Client.Do(httpRequest)
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
