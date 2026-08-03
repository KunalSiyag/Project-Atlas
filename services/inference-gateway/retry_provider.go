package inference

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// ProviderError is a normalized provider failure. Retryable controls whether
// the gateway may safely try the same request again.
type ProviderError struct {
	Class     string
	Retryable bool
	Err       error
}

func (e *ProviderError) Error() string {
	if e.Err == nil {
		return e.Class
	}
	return fmt.Sprintf("%s: %v", e.Class, e.Err)
}

func (e *ProviderError) Unwrap() error { return e.Err }

// RetryPolicy bounds provider retries. MaxAttempts includes the first call.
type RetryPolicy struct {
	MaxAttempts int
	Backoff     time.Duration
}

// RetryingProvider decorates any provider with bounded, context-aware retries.
type RetryingProvider struct {
	Provider ModelProvider
	Policy   RetryPolicy
	Sleep    func(context.Context, time.Duration) error
}

func (p RetryingProvider) Generate(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if p.Provider == nil {
		return ChatResponse{}, errors.New("retry provider has no underlying provider")
	}
	maxAttempts := p.Policy.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	sleep := p.Sleep
	if sleep == nil {
		sleep = waitWithContext
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		attemptCtx, span := otel.Tracer(instrumentationName).Start(ctx, "inference.provider.attempt", trace.WithAttributes(attribute.Int("attempt", attempt)))
		response, err := p.Provider.Generate(attemptCtx, request)
		if err != nil {
			span.RecordError(err)
		}
		span.End()
		if err == nil {
			return response, nil
		}
		lastErr = err
		if !isRetryable(err) || attempt == maxAttempts {
			return ChatResponse{}, err
		}
		if err := sleep(ctx, p.Policy.Backoff); err != nil {
			return ChatResponse{}, err
		}
	}
	return ChatResponse{}, lastErr
}

func isRetryable(err error) bool {
	var providerErr *ProviderError
	return errors.As(err, &providerErr) && providerErr.Retryable
}

func waitWithContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
