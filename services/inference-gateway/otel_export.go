package inference

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.41.0"
)

// TelemetryShutdown flushes and stops OpenTelemetry providers installed by InstallTelemetry.
type TelemetryShutdown func(context.Context) error

// InstallTelemetry configures global OpenTelemetry providers from Config.
// exporter "none" or empty leaves the global no-op providers (safe for tests).
// "console" writes metrics and spans to stdout; "otlp" uses OTLP/HTTP.
func InstallTelemetry(ctx context.Context, config Config) (TelemetryShutdown, error) {
	mode := strings.ToLower(strings.TrimSpace(config.OTELExporter))
	if mode == "" || mode == "none" || mode == "off" || mode == "disabled" {
		return func(context.Context) error { return nil }, nil
	}

	serviceName := config.OTELServiceName
	if serviceName == "" {
		serviceName = "atlas-inference-gateway"
	}
	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	var (
		traceExporter sdktrace.SpanExporter
		metricReader  sdkmetric.Reader
	)

	switch mode {
	case "console", "stdout":
		traceExporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("stdout trace exporter: %w", err)
		}
		metricExp, err := stdoutmetric.New()
		if err != nil {
			return nil, fmt.Errorf("stdout metric exporter: %w", err)
		}
		metricReader = sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(200*time.Millisecond))
	case "otlp":
		endpoint := strings.TrimSpace(config.OTELEndpoint)
		if endpoint == "" {
			endpoint = "localhost:4318"
		}
		// Allow host:port or full URL; otlptracehttp accepts WithEndpoint without scheme.
		endpoint = strings.TrimPrefix(endpoint, "https://")
		endpoint = strings.TrimPrefix(endpoint, "http://")
		traceExporter, err = otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("otlp trace exporter: %w", err)
		}
		metricExp, err := otlpmetrichttp.New(ctx,
			otlpmetrichttp.WithEndpoint(endpoint),
			otlpmetrichttp.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("otlp metric exporter: %w", err)
		}
		metricReader = sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(1*time.Second))
	default:
		return nil, fmt.Errorf("unsupported ATLAS_OTEL_EXPORTER %q (use none, console, or otlp)", config.OTELExporter)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter, sdktrace.WithBatchTimeout(100*time.Millisecond)),
		sdktrace.WithResource(res),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(metricReader),
		sdkmetric.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	return func(shutdownCtx context.Context) error {
		var first error
		if err := tp.Shutdown(shutdownCtx); err != nil && first == nil {
			first = err
		}
		if err := mp.Shutdown(shutdownCtx); err != nil && first == nil {
			first = err
		}
		return first
	}, nil
}

// InstallTelemetryFromEnv is a convenience for main: load config fields that affect OTEL only.
func InstallTelemetryFromEnv(ctx context.Context) (TelemetryShutdown, error) {
	return InstallTelemetry(ctx, Config{
		OTELExporter:    valueOrDefault("ATLAS_OTEL_EXPORTER", "none"),
		OTELEndpoint:    valueOrDefault("ATLAS_OTEL_ENDPOINT", "localhost:4318"),
		OTELServiceName: valueOrDefault("ATLAS_OTEL_SERVICE_NAME", "atlas-inference-gateway"),
	})
}

// force flush helper used in tests — re-read env after Install for verification scripts.
func otelExporterEnv() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv("ATLAS_OTEL_EXPORTER")))
}
