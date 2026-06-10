// Package tracing wires OpenTelemetry to Google Cloud Trace.
//
// Spans are created per HTTP request by the otelgin middleware; this package
// owns the TracerProvider + exporter lifecycle. It degrades gracefully: with
// no GCP project (local dev) or no credentials it returns a no-op shutdown and
// the service boots normally, just without traces.
package tracing

import (
	"context"
	"fmt"
	"log"

	texporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Init configures the global TracerProvider exporting to Cloud Trace and
// returns a shutdown func (always non-nil — safe to defer even on error).
//
// sampleRatio is the head-sampling rate for root spans (0.0–1.0); child spans
// inherit the parent's decision via ParentBased. Cloud Trace's free tier is
// generous (2.5M spans/mo) but sampling keeps cost predictable under load.
func Init(ctx context.Context, projectID, env, version string, sampleRatio float64) (func(context.Context) error, error) {
	noop := func(context.Context) error { return nil }

	if projectID == "" {
		log.Println("tracing: no GCP project configured — Cloud Trace disabled")
		return noop, nil
	}

	exporter, err := texporter.New(texporter.WithProjectID(projectID))
	if err != nil {
		return noop, fmt.Errorf("cloud trace exporter: %w", err)
	}

	if version == "" {
		version = "dev"
	}
	res, err := resource.New(ctx, resource.WithAttributes(
		attribute.String("service.name", "homechef-api"),
		attribute.String("service.version", version),
		attribute.String("deployment.environment", env),
	))
	if err != nil {
		// resource.New can return a partial resource + a schema-merge error;
		// prefer a bare default resource over disabling tracing entirely.
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio))),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	log.Printf("tracing: Cloud Trace enabled (project=%s, sample=%.2f)", projectID, sampleRatio)
	return tp.Shutdown, nil
}
