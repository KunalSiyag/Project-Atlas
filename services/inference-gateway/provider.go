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
	Provider  string            `json:"provider,omitempty"`
	APIKey    string            `json:"api_key,omitempty"`
	Model     string            `json:"model,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	// Stream requests incremental output. The HTTP layer uses SSE when true;
	// providers may implement StreamingModelProvider for native token streams.
	Stream bool `json:"stream,omitempty"`
}

// StreamDelta is one incremental content fragment during a streaming generate.
type StreamDelta struct {
	Delta    string `json:"delta"`
	Provider string `json:"provider,omitempty"`
	Model    string `json:"model,omitempty"`
}

// StreamEmit writes one delta to the client. Returning a non-nil error aborts the stream.
type StreamEmit func(StreamDelta) error

// StreamingModelProvider is an optional extension for native token streaming.
// Providers that do not implement it still support stream=true via a synthesized
// post-generate chunking path in streamGenerate.
type StreamingModelProvider interface {
	GenerateStream(ctx context.Context, request ChatRequest, emit StreamEmit) (ChatResponse, error)
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
	Resolve(taskClass, requestedProvider, apiKey, requestedModel string) (ModelProvider, string, error)
}
