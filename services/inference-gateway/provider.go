// Package inference contains the provider-independent boundary used by the
// inference gateway. Concrete model vendors belong behind these interfaces.
package inference

import "context"

// Message is one conversational input to a chat model.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the normalized request sent to a model provider.
type ChatRequest struct {
	Messages  []Message         `json:"messages"`
	TaskClass string            `json:"task_class,omitempty"`
	Model     string            `json:"model,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// Usage contains provider-reported token counts when available.
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ChatResponse is the normalized result returned by a model provider.
type ChatResponse struct {
	Message      Message `json:"message"`
	Usage        Usage   `json:"usage"`
	Provider     string  `json:"provider"`
	Model        string  `json:"model"`
	FinishReason string  `json:"finish_reason"`
}

// ModelProvider is the port used by the gateway to call any chat provider.
// The gateway knows this interface, not a vendor SDK.
type ModelProvider interface {
	Generate(ctx context.Context, request ChatRequest) (ChatResponse, error)
}

// ModelRouter resolves a logical task class to a provider and concrete model.
type ModelRouter interface {
	Resolve(taskClass, requestedModel string) (ModelProvider, string, error)
}
