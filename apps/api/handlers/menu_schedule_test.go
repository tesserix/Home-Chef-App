package handlers

import "testing"

func TestSanitizeWeekdays(t *testing.T) {
	// Valid, in order.
	got := sanitizeWeekdays([]int{1, 3, 5})
	if len(got) != 3 || got[0] != 1 || got[1] != 3 || got[2] != 5 {
		t.Fatalf("valid days should pass through in order; got %v", got)
	}

	// Drops out-of-range + de-dupes, preserves first-seen order.
	got = sanitizeWeekdays([]int{7, -1, 2, 2, 0, 9})
	if len(got) != 2 || got[0] != 2 || got[1] != 0 {
		t.Fatalf("expected [2 0] after dropping invalid/dupes; got %v", got)
	}

	// Empty stays empty (non-nil) = every day.
	if got := sanitizeWeekdays(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil input must yield empty non-nil array; got %v", got)
	}
}
