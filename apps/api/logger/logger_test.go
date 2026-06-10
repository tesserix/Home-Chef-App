package logger

import (
	"context"
	"testing"
)

func TestCorrelationIDRoundTrip(t *testing.T) {
	ctx := ContextWithCorrelationID(context.Background(), "abc-123")
	if got := CorrelationIDFromContext(ctx); got != "abc-123" {
		t.Errorf("CorrelationIDFromContext = %q, want abc-123", got)
	}
}

func TestTraceIDRoundTrip(t *testing.T) {
	ctx := ContextWithTraceID(context.Background(), "trace-xyz")
	if got := TraceIDFromContext(ctx); got != "trace-xyz" {
		t.Errorf("TraceIDFromContext = %q, want trace-xyz", got)
	}
}

func TestIDsFromEmptyContext(t *testing.T) {
	// A context variable left nil — exercises the defensive nil-guard without
	// the SA1012 literal-nil lint.
	var nilCtx context.Context
	if got := CorrelationIDFromContext(context.Background()); got != "" {
		t.Errorf("want empty correlation id, got %q", got)
	}
	if got := TraceIDFromContext(nilCtx); got != "" {
		t.Errorf("want empty trace id for nil ctx, got %q", got)
	}
}

func TestFromContextDoesNotPanic(t *testing.T) {
	// Before Init, FromContext should fall back to slog.Default() cleanly.
	var nilCtx context.Context
	_ = FromContext(context.Background())
	_ = FromContext(nilCtx)
}
