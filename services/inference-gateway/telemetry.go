package inference

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const instrumentationName = "github.com/KunalSiyag/Project-Atlas/services/inference-gateway"

// Telemetry contains the gateway's OpenTelemetry instruments. The API uses
// the configured global providers, which are no-op by default in local tests.
type Telemetry struct {
	tracer    trace.Tracer
	requests  metric.Int64Counter
	failures  metric.Int64Counter
	latencyMs metric.Float64Histogram
}

func NewTelemetry() *Telemetry {
	meter := otel.Meter(instrumentationName)
	requests, _ := meter.Int64Counter("atlas.inference.requests", metric.WithDescription("Inference requests started"))
	failures, _ := meter.Int64Counter("atlas.inference.failures", metric.WithDescription("Inference requests that failed"))
	latency, _ := meter.Float64Histogram("atlas.inference.latency_ms", metric.WithDescription("Inference request latency in milliseconds"), metric.WithUnit("ms"))
	return &Telemetry{
		tracer:    otel.Tracer(instrumentationName),
		requests:  requests,
		failures:  failures,
		latencyMs: latency,
	}
}

func (t *Telemetry) StartRequest(ctx context.Context, requestID string) (context.Context, trace.Span) {
	if t == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, "inference.request", trace.WithAttributes(attribute.String("request_id", requestID)))
}

func (t *Telemetry) RecordStarted(ctx context.Context, taskClass string) {
	if t != nil && t.requests != nil {
		t.requests.Add(ctx, 1, metric.WithAttributes(attribute.String("task_class", taskClass)))
	}
}

func (t *Telemetry) RecordCompleted(ctx context.Context, taskClass, provider string, started time.Time) {
	if t != nil && t.latencyMs != nil {
		t.latencyMs.Record(ctx, float64(time.Since(started).Microseconds())/1000, metric.WithAttributes(
			attribute.String("task_class", taskClass),
			attribute.String("provider", provider),
		))
	}
}

func (t *Telemetry) RecordFailure(ctx context.Context, taskClass string) {
	if t != nil && t.failures != nil {
		t.failures.Add(ctx, 1, metric.WithAttributes(attribute.String("task_class", taskClass)))
	}
}
