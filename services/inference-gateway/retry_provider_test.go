package inference

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryingProviderRetriesTransientErrors(t *testing.T) {
	provider := &sequenceProvider{errors: []error{
		&ProviderError{Class: "unavailable", Retryable: true, Err: errors.New("temporary")},
		&ProviderError{Class: "rate_limit", Retryable: true, Err: errors.New("slow down")},
		nil,
	}}
	wrapped := RetryingProvider{
		Provider: provider,
		Policy:   RetryPolicy{MaxAttempts: 3},
		Sleep: func(context.Context, time.Duration) error {
			return nil
		},
	}

	response, err := wrapped.Generate(context.Background(), ChatRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if response.Message.Content != "success" || provider.calls != 3 {
		t.Fatalf("response/calls = %+v/%d, want success/3", response, provider.calls)
	}
}

func TestRetryingProviderDoesNotRetryPermanentErrors(t *testing.T) {
	provider := &sequenceProvider{errors: []error{
		&ProviderError{Class: "invalid_request", Retryable: false, Err: errors.New("bad input")},
	}}
	wrapped := RetryingProvider{Provider: provider, Policy: RetryPolicy{MaxAttempts: 3}}

	_, err := wrapped.Generate(context.Background(), ChatRequest{})
	if err == nil || provider.calls != 1 {
		t.Fatalf("error/calls = %v/%d, want error/1", err, provider.calls)
	}
}

func TestRetryingProviderStopsAtAttemptLimit(t *testing.T) {
	provider := &sequenceProvider{errors: []error{
		&ProviderError{Class: "unavailable", Retryable: true, Err: errors.New("down")},
		&ProviderError{Class: "unavailable", Retryable: true, Err: errors.New("still down")},
		&ProviderError{Class: "unavailable", Retryable: true, Err: errors.New("still down")},
	}}
	wrapped := RetryingProvider{
		Provider: provider,
		Policy:   RetryPolicy{MaxAttempts: 2},
		Sleep: func(context.Context, time.Duration) error {
			return nil
		},
	}

	_, err := wrapped.Generate(context.Background(), ChatRequest{})
	if err == nil || provider.calls != 2 {
		t.Fatalf("error/calls = %v/%d, want error/2", err, provider.calls)
	}
}

type sequenceProvider struct {
	errors []error
	calls  int
}

func (p *sequenceProvider) Generate(context.Context, ChatRequest) (ChatResponse, error) {
	index := p.calls
	p.calls++
	if index < len(p.errors) && p.errors[index] != nil {
		return ChatResponse{}, p.errors[index]
	}
	return ChatResponse{Message: Message{Role: "assistant", Content: "success"}}, nil
}
