// telemetry/otel.go — OpenTelemetry TracerProvider bootstrap.
//
// InitProvider sets the global OTel TracerProvider. The returned shutdown
// function MUST be called from the SIGTERM handler to flush buffered spans
// before the pod exits. Losing the crash-path spans is the "silent P0" the
// doctrine calls out — so Shutdown is chained into the existing graceful
// teardown sequence in main.go, not deferred in isolation.
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// ShutdownFunc flushes pending spans. context deadline controls how long
// the flush may block. Called from the SIGTERM handler.
type ShutdownFunc func(ctx context.Context) error

// InitProvider bootstraps the OTel TracerProvider with a batched span
// processor. Today the exporter writes to stdout (JSON) — swap to OTLP
// when a collector is deployed. The provider is registered as the global
// default so otel.Tracer("backend-go") works anywhere.
//
// Environment knobs:
//
//	OTEL_SERVICE_NAME   — defaults to "backend-go"
//	OTEL_EXPORTER       — "stdout" (default) | "none" (no-op for tests)
//	OTEL_SAMPLE_RATE    — reserved for Phase 6 sampling config
func InitProvider(ctx context.Context) (ShutdownFunc, error) {
	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "backend-go"
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
		),
		resource.WithOS(),
		resource.WithProcess(),
	)
	if err != nil {
		return nil, fmt.Errorf("telemetry.InitProvider: build resource: %w", err)
	}

	// ── Exporter selection ──────────────────────────────────────────────
	exporterKind := os.Getenv("OTEL_EXPORTER")
	if exporterKind == "" {
		exporterKind = "stdout"
	}

	var exporter sdktrace.SpanExporter
	switch exporterKind {
	case "none":
		// No-op: useful in test / local-dev where span noise is unwanted.
		// The provider still propagates context — it just doesn't export.
		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithSampler(sdktrace.NeverSample()),
		)
		otel.SetTracerProvider(tp)
		return tp.Shutdown, nil

	case "stdout":
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("telemetry.InitProvider: stdout exporter: %w", err)
		}

	default:
		return nil, fmt.Errorf("telemetry.InitProvider: unknown OTEL_EXPORTER %q (supported: stdout, none)", exporterKind)
	}

	// BatchSpanProcessor: flushes every 5 s or when 512 spans accumulate —
	// whichever comes first. MaxExportBatchSize caps a single export call
	// so a burst doesn't OOM the exporter.
	bsp := sdktrace.NewBatchSpanProcessor(exporter,
		sdktrace.WithBatchTimeout(5*time.Second),
		sdktrace.WithMaxQueueSize(2048),
		sdktrace.WithMaxExportBatchSize(512),
	)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
		// Default sampler: AlwaysSample. Phase 6 will swap to
		// TraceIDRatioBased(rate) from OTEL_SAMPLE_RATE env var.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	slog.Info("telemetry.otel.init", "exporter", exporterKind, "service", serviceName)

	// Return a shutdown function that flushes the batch processor.
	// The caller (main.go SIGTERM handler) MUST invoke this with a
	// deadline context — if the pod is being killed, we get N seconds
	// to flush, not infinity.
	return tp.Shutdown, nil
}
