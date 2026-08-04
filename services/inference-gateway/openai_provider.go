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
	Stream   bool            `json:"stream,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message      openAIMessage `json:"message"`
		Delta        openAIMessage `json:"delta"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    any    `json:"code"`
	} `json:"error"`
}

func (p OpenAIProvider) name() string {
	if p.ProviderName == "" {
		return "openai"
	}
	return p.ProviderName
}

func (p OpenAIProvider) baseURL() string {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if base == "" {
		return "https://api.openai.com/v1"
	}
	return base
}

func (p OpenAIProvider) client() *http.Client {
	if p.Client == nil {
		return http.DefaultClient
	}
	return p.Client
}

func (p OpenAIProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if strings.TrimSpace(request.Model) == "" {
		return ChatResponse{}, fmt.Errorf("model is required")
	}
	if len(request.Messages) == 0 {
		return ChatResponse{}, fmt.Errorf("at least one message is required")
	}

	msgs := make([]openAIMessage, 0, len(request.Messages))
	for _, m := range request.Messages {
		msgs = append(msgs, openAIMessage{Role: m.Role, Content: m.Content})
	}
	body, err := json.Marshal(openAIChatRequest{Model: request.Model, Messages: msgs})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal openai request: %w", err)
	}

	endpoint := p.baseURL() + "/chat/completions"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create openai request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	if key := strings.TrimSpace(p.APIKey); key != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+key)
	}

	httpResponse, err := p.client().Do(httpRequest)
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
	usage := Usage{}
	if parsed.Usage != nil {
		usage = Usage{InputTokens: parsed.Usage.PromptTokens, OutputTokens: parsed.Usage.CompletionTokens}
	}
	return ChatResponse{
		Message: Message{
			Role:    nonEmptyRole(choice.Message.Role, "assistant"),
			Content: choice.Message.Content,
		},
		Usage:        usage,
		Provider:     p.name(),
		Model:        model,
		FinishReason: choice.FinishReason,
	}, nil
}

// GenerateStream consumes OpenAI-compatible SSE chat.completion.chunk events.
func (p OpenAIProvider) GenerateStream(ctx context.Context, request ChatRequest, emit StreamEmit) (ChatResponse, error) {
	if strings.TrimSpace(request.Model) == "" {
		return ChatResponse{}, fmt.Errorf("model is required")
	}
	if len(request.Messages) == 0 {
		return ChatResponse{}, fmt.Errorf("at least one message is required")
	}

	msgs := make([]openAIMessage, 0, len(request.Messages))
	for _, m := range request.Messages {
		msgs = append(msgs, openAIMessage{Role: m.Role, Content: m.Content})
	}
	body, err := json.Marshal(openAIChatRequest{Model: request.Model, Messages: msgs, Stream: true})
	if err != nil {
		return ChatResponse{}, fmt.Errorf("marshal openai stream request: %w", err)
	}

	endpoint := p.baseURL() + "/chat/completions"
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("create openai stream request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "text/event-stream")
	if key := strings.TrimSpace(p.APIKey); key != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+key)
	}

	httpResponse, err := p.client().Do(httpRequest)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("call openai stream: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < http.StatusOK || httpResponse.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(httpResponse.Body, 4096))
		class, retryable := classifyHTTPStatus(httpResponse.StatusCode)
		return ChatResponse{}, &ProviderError{
			Class:     class,
			Retryable: retryable,
			Err:       fmt.Errorf("openai stream HTTP %d: %s", httpResponse.StatusCode, strings.TrimSpace(string(responseBody))),
		}
	}

	var (
		content strings.Builder
		model   = request.Model
		finish  string
		usage   Usage
		name    = p.name()
	)
	scanner := bufio.NewScanner(httpResponse.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return ChatResponse{}, err
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk openAIChatResponse
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return ChatResponse{}, fmt.Errorf("decode openai stream chunk: %w", err)
		}
		if chunk.Model != "" {
			model = chunk.Model
		}
		if chunk.Usage != nil {
			usage = Usage{InputTokens: chunk.Usage.PromptTokens, OutputTokens: chunk.Usage.CompletionTokens}
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]
		if choice.FinishReason != "" {
			finish = choice.FinishReason
		}
		delta := choice.Delta.Content
		if delta != "" {
			content.WriteString(delta)
			if err := emit(StreamDelta{Delta: delta, Provider: name, Model: model}); err != nil {
				return ChatResponse{}, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return ChatResponse{}, fmt.Errorf("read openai stream: %w", err)
	}
	if finish == "" {
		finish = "stop"
	}
	return ChatResponse{
		Message:      Message{Role: "assistant", Content: content.String()},
		Usage:        usage,
		Provider:     name,
		Model:        model,
		FinishReason: finish,
	}, nil
}

func nonEmptyRole(role, fallback string) string {
	if strings.TrimSpace(role) == "" {
		return fallback
	}
	return role
}
