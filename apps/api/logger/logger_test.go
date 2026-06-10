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
	if got := CorrelationIDFromContext(context.Background()); got != "" {
		t.Errorf("want empty correlation id, got %q", got)
	}
	if got := TraceIDFromContext(nil); got != "" { //nolint:staticcheck // nil ctx tolerated
		t.Errorf("want empty trace id for nil ctx, got %q", got)
	}
}

func TestFromContextDoesNotPanic(t *testing.T) {
	// Before Init, FromContext should fall back to slog.Default() cleanly.
	_ = FromContext(context.Background())
	_ = FromContext(nil)
}
