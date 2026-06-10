// Package logger is the structured-logging foundation for homechef-api.
//
// It wraps log/slog with a JSON handler and threads a correlation ID (and,
// once OpenTelemetry is wired, a trace ID) through context so every log line
// emitted during a request can be tied back to that request — and to its
// distributed trace — in Cloud Logging.
package logger

import (
	"context"
	"log/slog"
	"os"
)

type ctxKey string

const (
	correlationIDKey ctxKey = "correlation_id"
	traceIDKey       ctxKey = "trace_id"
)

// base is the process-wide logger set by Init. Falls back to slog.Default()
// if Init was never called (e.g. in tests).
var base *slog.Logger

// Init configures the global JSON logger. Call once at startup, before any
// request handling. env tags every line so prod/staging logs are separable.
func Init(env string) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	base = slog.New(handler).With("service", "homechef-api", "env", env)
	slog.SetDefault(base)
}

// L returns the process logger (or slog's default before Init).
func L() *slog.Logger {
	if base == nil {
		return slog.Default()
	}
	return base
}

// FromContext returns a logger enriched with whatever correlation/trace IDs
// are present on ctx. Use this inside request-scoped code so log lines carry
// the request's identity.
func FromContext(ctx context.Context) *slog.Logger {
	l := L()
	if ctx == nil {
		return l
	}
	if cid, ok := ctx.Value(correlationIDKey).(string); ok && cid != "" {
		l = l.With("correlation_id", cid)
	}
	if tid, ok := ctx.Value(traceIDKey).(string); ok && tid != "" {
		l = l.With("trace_id", tid)
	}
	return l
}

// ContextWithCorrelationID returns a child context carrying the correlation ID.
func ContextWithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationIDFromContext extracts the correlation ID, or "" if unset.
func CorrelationIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if cid, ok := ctx.Value(correlationIDKey).(string); ok {
		return cid
	}
	return ""
}

// ContextWithTraceID returns a child context carrying the trace ID. Populated
// by the OpenTelemetry middleware so logs and audit rows can join to traces.
func ContextWithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// TraceIDFromContext extracts the trace ID, or "" if unset.
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if tid, ok := ctx.Value(traceIDKey).(string); ok {
		return tid
	}
	return ""
}
