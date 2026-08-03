package inference

import (
	"context"
	"errors"
)

// FakeProvider is a deterministic provider for local development and tests.
// It deliberately does not imitate model intelligence; it proves the port,
// request validation, and response normalization work without network access.
type FakeProvider struct {
	ProviderName string
}

func (p FakeProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if err := ctx.Err(); err != nil {
		return ChatResponse{}, err
	}
	if len(request.Messages) == 0 {
		return ChatResponse{}, errors.New("at least one message is required")
	}
	if request.Model == "" {
		return ChatResponse{}, errors.New("model is required")
	}

	last := request.Messages[len(request.Messages)-1]
	return ChatResponse{
		Message: Message{
			Role:    "assistant",
			Content: "fake response to: " + last.Content,
		},
		Usage: Usage{
			InputTokens:  len(request.Messages),
			OutputTokens: 1,
		},
		Provider:     p.ProviderName,
		Model:        request.Model,
		FinishReason: "stop",
	}, nil
}
