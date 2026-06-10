// Package tracing wires OpenTelemetry to Google Cloud Trace for auth-bff.
//
// Mirrors apps/api/tracing so login/session flows show up in the same Cloud
// Trace view as the API. Degrades gracefully: with no project (local dev) or
// no credentials it returns a no-op shutdown and the service boots normally.
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
		attribute.String("service.name", "auth-bff"),
		attribute.String("service.version", version),
		attribute.String("deployment.environment", env),
	))
	if err != nil {
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
