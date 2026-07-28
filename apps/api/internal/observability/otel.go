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
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
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

	// The exporters want a host:port without the scheme; an http:// prefix
	// means plaintext (insecure). https:// keeps TLS on.
	insecure := !strings.HasPrefix(endpoint, "https://")
	target := strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")

	// Choose gRPC or HTTP.
	//
	// This is not a preference — gRPC does not work from this namespace.
	// homechef runs under Istio ambient mesh, so ztunnel intercepts outbound
	// traffic and attempts HBONE to the destination; the observability
	// namespace is outside the mesh and cannot answer, and the HTTP/2 preface
	// is reset before the connection is established:
	//
	//   traces export: rpc error: code = Unavailable ...
	//   "error reading server preface: connection reset by peer"
	//
	// Every trace was being dropped. devai is also ambient and exports fine
	// over OTLP/HTTP on 4318, which is the evidence this follows. mark8ly uses
	// gRPC successfully only because it is NOT in the mesh.
	//
	// Inferred from the port so the deployment decides by pointing at 4317 or
	// 4318, with OTEL_EXPORTER_OTLP_PROTOCOL as an explicit override for the
	// case where the ports are ever remapped.
	useHTTP := strings.HasSuffix(target, ":4318")
	switch strings.TrimSpace(strings.ToLower(os.Getenv("OTEL_EXPORTER_OTLP_PROTOCOL"))) {
	case "http/protobuf", "http":
		useHTTP = true
	case "grpc":
		useHTTP = false
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		// resource.New can return a partial resource together with a
		// schema-merge error; prefer a bare default over disabling telemetry.
		res = resource.Default()
	}

	var (
		traceExp  sdktrace.SpanExporter
		metricExp sdkmetric.Exporter
	)

	if useHTTP {
		traceOpts := []otlptracehttp.Option{otlptracehttp.WithEndpoint(target)}
		metricOpts := []otlpmetrichttp.Option{otlpmetrichttp.WithEndpoint(target)}
		if insecure {
			traceOpts = append(traceOpts, otlptracehttp.WithInsecure())
			metricOpts = append(metricOpts, otlpmetrichttp.WithInsecure())
		}

		traceExp, err = otlptracehttp.New(ctx, traceOpts...)
		if err != nil {
			return noop, fmt.Errorf("otlp trace exporter (http): %w", err)
		}

		metricExp, err = otlpmetrichttp.New(ctx, metricOpts...)
		if err != nil {
			// Don't leak the trace exporter if metrics fail to start.
			_ = traceExp.Shutdown(ctx)
			return noop, fmt.Errorf("otlp metric exporter (http): %w", err)
		}
	} else {
		traceOpts := []otlptracegrpc.Option{otlptracegrpc.WithEndpoint(target)}
		metricOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(target)}
		if insecure {
			traceOpts = append(traceOpts, otlptracegrpc.WithInsecure())
			metricOpts = append(metricOpts, otlpmetricgrpc.WithInsecure())
		}

		traceExp, err = otlptracegrpc.New(ctx, traceOpts...)
		if err != nil {
			return noop, fmt.Errorf("otlp trace exporter: %w", err)
		}

		metricExp, err = otlpmetricgrpc.New(ctx, metricOpts...)
		if err != nil {
			// Don't leak the trace exporter if metrics fail to start.
			_ = traceExp.Shutdown(ctx)
			return noop, fmt.Errorf("otlp metric exporter: %w", err)
		}
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
