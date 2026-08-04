package inference

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestInstallTelemetryNoneIsNoop(t *testing.T) {
	shutdown, err := InstallTelemetry(context.Background(), Config{OTELExporter: "none"})
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestInstallTelemetryConsoleAndRecord(t *testing.T) {
	// Capture stdout-ish by using manual reader after install console —
	// instead assert SDK path via manual meter provider is not required:
	// Install console and fire instruments through Telemetry helpers, then force flush.
	var buf bytes.Buffer
	// Use a manual setup that mirrors InstallTelemetry console path with a capturing metric reader.
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	prev := otel.GetMeterProvider()
	otel.SetMeterProvider(mp)
	defer otel.SetMeterProvider(prev)

	tp := sdktrace.NewTracerProvider()
	prevTP := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(prevTP)

	tel := NewTelemetry()
	ctx, span := tel.StartRequest(context.Background(), "req-otel")
	tel.RecordStarted(ctx, "act")
	tel.RecordCompleted(ctx, "act", "fake", time.Now().Add(-5*time.Millisecond))
	span.End()

	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if strings.Contains(m.Name, "atlas.inference") {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("expected atlas.inference metrics, got %+v", rm)
	}
	_ = buf
}

func TestInstallTelemetryRejectsUnknown(t *testing.T) {
	_, err := InstallTelemetry(context.Background(), Config{OTELExporter: "banana"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestInstallTelemetryConsoleStarts(t *testing.T) {
	shutdown, err := InstallTelemetry(context.Background(), Config{
		OTELExporter:    "console",
		OTELServiceName: "atlas-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	tel := NewTelemetry()
	ctx, span := tel.StartRequest(context.Background(), "r1")
	tel.RecordStarted(ctx, "act")
	span.End()
	// Give periodic reader a moment; shutdown flushes.
	time.Sleep(250 * time.Millisecond)
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
