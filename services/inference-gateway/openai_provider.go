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

// OpenAIProvider implements ModelProvider against an OpenAI-compatible
// Chat Completions HTTP API (OpenAI, many local proxies, Azure-compatible gateways).
type OpenAIProvider struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
	// ProviderName is reported in ChatResponse.Provider (default "openai").
	ProviderName string
}

type openAIChatRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (p OpenAIProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(request.Model) == "" {
		return ChatResponse{}, fmt.Errorf("model is required")
	}
	if len(request.Messages) == 0 {
		return ChatResponse{}, fmt.Errorf("at least one message is required")
	}
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	if p.Client == nil {
		p.Client = http.DefaultClient
	}
	name := p.ProviderName
	if name == "" {
		name = "openai"
	}

	msgs := make([]openAIMessage, 0, len(request.Messages))
	for _, m := range request.Messages {
		msgs = append(msgs, openAIMessage{Role: m.Role, Content: m.Content})
	}
	body, err := json.Marshal(openAIChatRequest{Model: request.Model, Messages: msgs})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal openai request: %w", err)
	}

	endpoint := base + "/chat/completions"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create openai request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(p.APIKey); key != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+key)
	}

	httpResponse, err := p.Client.Do(httpRequest)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("call openai: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBody, err := io.ReadAll(httpResponse.Body)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("read openai response: %w", err)
	}

	var parsed openAIChatResponse
	_ = json.Unmarshal(responseBody, &parsed)

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		class, retryable := classifyHTTPStatus(httpResponse.StatusCode)
		msg := strings.TrimSpace(string(responseBody))
		if parsed.Error != nil && parsed.Error.Message != "" {
			msg = parsed.Error.Message
		}
		return ChatResponse{}, &ProviderError{
			Class:     class,
			Retryable: retryable,
			Err:       fmt.Errorf("openai HTTP %d: %s", httpResponse.StatusCode, msg),
		}
	}
	if len(parsed.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("openai response missing choices")
	}
	choice := parsed.Choices[0]
	model := parsed.Model
	if model == "" {
		model = request.Model
	}
	return ChatResponse{
		Message: Message{
			Role:    nonEmptyRole(choice.Message.Role, "assistant"),
			Content: choice.Message.Content,
		},
		Usage: Usage{
			InputTokens:  parsed.Usage.PromptTokens,
			OutputTokens: parsed.Usage.CompletionTokens,
		},
		Provider:     name,
		Model:        model,
		FinishReason: choice.FinishReason,
	}, nil
}

func nonEmptyRole(role, fallback string) string {
	if strings.TrimSpace(role) == "" {
		return fallback
	}
	return role
}
