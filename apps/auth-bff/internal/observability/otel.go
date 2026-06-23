// Package observability wires auth-bff to an in-cluster OpenTelemetry
// Collector over OTLP/gRPC, exporting both traces and metrics.
//
// This is independent of the Cloud Trace exporter in internal/tracing: it
// targets the cluster collector (otel-collector.observability) addressed via
// OTEL_EXPORTER_OTLP_ENDPOINT. It degrades gracefully — when the endpoint env
// is empty (e.g. local dev) Init is a no-op and returns a no-op shutdown.
package observability

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// DefaultEndpoint is the in-cluster OTLP/gRPC collector address used as the
// documented default for OTEL_EXPORTER_OTLP_ENDPOINT (set by the Helm chart).
const DefaultEndpoint = "http://otel-collector.observability.svc.cluster.local:4317"

// Init configures a global batch TracerProvider and periodic MeterProvider that
// export over OTLP/gRPC to the in-cluster collector, and returns a shutdown
// func (always non-nil — safe to defer even on error).
//
// The collector endpoint comes from OTEL_EXPORTER_OTLP_ENDPOINT. When that env
// var is empty, Init no-ops (OTLP export disabled) and returns a no-op shutdown.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }

	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		log.Println("observability: OTEL_EXPORTER_OTLP_ENDPOINT unset — OTLP export disabled")
		return noop, nil
	}

	// otlp*grpc.WithEndpoint wants host:port, not a scheme. Strip http(s):// and
	// derive whether TLS is required from the scheme.
	target, insecure := parseEndpoint(endpoint)

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		res = resource.Default()
	}

	// --- Traces ---
	traceOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(target)}
	if insecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
	}
	traceExp, err := otlptracegrpc.New(ctx, traceOpts...)
	if err != nil {
		return noop, fmt.Errorf("otlp trace exporter: %w", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)

	// --- Metrics ---
	metricOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(target)}
	if insecure {
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	}
	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		// Don't leak the already-started trace provider.
		_ = tp.Shutdown(ctx)
		return noop, fmt.Errorf("otlp metric exporter: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp,
			sdkmetric.WithInterval(30*time.Second),
		)),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	log.Printf("observability: OTLP/gRPC enabled (service=%s, endpoint=%s)", serviceName, target)

	shutdown := func(ctx context.Context) error {
		var firstErr error
		if err := tp.Shutdown(ctx); err != nil {
			firstErr = err
		}
		if err := mp.Shutdown(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		return firstErr
	}
	return shutdown, nil
}

// parseEndpoint splits an OTLP endpoint into a host:port target and a flag
// indicating whether the connection should be plaintext. A missing scheme or
// http:// implies insecure; https:// implies TLS.
func parseEndpoint(endpoint string) (target string, insecure bool) {
	switch {
	case strings.HasPrefix(endpoint, "http://"):
		return strings.TrimPrefix(endpoint, "http://"), true
	case strings.HasPrefix(endpoint, "https://"):
		return strings.TrimPrefix(endpoint, "https://"), false
	default:
		// No scheme — assume plaintext in-cluster gRPC.
		return endpoint, true
	}
}
