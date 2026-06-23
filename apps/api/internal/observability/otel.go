// Package observability wires OpenTelemetry traces + metrics to an OTLP
// collector over gRPC.
//
// It is intentionally self-contained and dependency-light: Init configures
// global Tracer/Meter providers exporting via OTLP gRPC and returns a single
// shutdown func that flushes and tears both down. It degrades gracefully — when
// OTEL_EXPORTER_OTLP_ENDPOINT is empty the whole stack is disabled and Init
// returns a no-op shutdown, so the service boots normally without a collector.
package observability

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// defaultEndpoint is the in-cluster OTLP gRPC collector address used when
// OTEL_EXPORTER_OTLP_ENDPOINT is unset.
const defaultEndpoint = "http://otel-collector.observability.svc.cluster.local:4317"

// Init configures global OpenTelemetry trace + metric providers exporting to an
// OTLP gRPC collector and returns a shutdown func (always non-nil — safe to
// defer even on error).
//
// The collector endpoint comes from OTEL_EXPORTER_OTLP_ENDPOINT, defaulting to
// the in-cluster collector. If that env var is explicitly set to an empty
// string the exporters are disabled and Init returns a no-op shutdown.
func Init(ctx context.Context, serviceName string) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }

	endpoint, set := os.LookupEnv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if !set {
		endpoint = defaultEndpoint
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		log.Println("observability: OTEL_EXPORTER_OTLP_ENDPOINT empty — OTLP traces/metrics disabled")
		return noop, nil
	}

	// otlptracegrpc/otlpmetricgrpc want a host:port without the scheme; an
	// http:// prefix means plaintext (insecure). https:// keeps TLS on.
	insecure := !strings.HasPrefix(endpoint, "https://")
	target := strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		// resource.New can return a partial resource together with a
		// schema-merge error; prefer a bare default over disabling telemetry.
		res = resource.Default()
	}

	traceOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(target)}
	metricOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(target)}
	if insecure {
		traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
		metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
	}

	traceExp, err := otlptracegrpc.New(ctx, traceOpts...)
	if err != nil {
		return noop, fmt.Errorf("otlp trace exporter: %w", err)
	}

	metricExp, err := otlpmetricgrpc.New(ctx, metricOpts...)
	if err != nil {
		// Don't leak the trace exporter if metrics fail to start.
		_ = traceExp.Shutdown(ctx)
		return noop, fmt.Errorf("otlp metric exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExp),
		sdktrace.WithResource(res),
	)
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp)),
	)

	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	log.Printf("observability: OTLP gRPC traces+metrics enabled (endpoint=%s, insecure=%t)", target, insecure)

	// shutdown flushes and tears down both providers. Best-effort: errors are
	// joined so a failure in one doesn't skip the other.
	shutdown := func(ctx context.Context) error {
		var errs []error
		if err := tp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("tracer provider shutdown: %w", err))
		}
		if err := mp.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("meter provider shutdown: %w", err))
		}
		if len(errs) == 0 {
			return nil
		}
		msgs := make([]string, 0, len(errs))
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		return fmt.Errorf("observability shutdown: %s", strings.Join(msgs, "; "))
	}
	return shutdown, nil
}
