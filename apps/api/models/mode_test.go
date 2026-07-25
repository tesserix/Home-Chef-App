package models

import (
	"testing"
	"time"
)

func TestNormalizeMode(t *testing.T) {
	cases := map[string]string{
		"live":    ChefModeLive,
		"test":    ChefModeTest,
		"":        ChefModeLive,
		"LIVE":    ChefModeLive,
		"TEST":    ChefModeTest,
		" test ":  ChefModeTest,
		"garbage": ChefModeLive,
	}
	for in, want := range cases {
		if got := NormalizeMode(in); got != want {
			t.Fatalf("NormalizeMode(%q) = %q, want %q", in, got, want)
		}
	}
}

// The fail-safe direction: anything we don't recognise must read as live, so a
// column-default glitch can never hide a real kitchen or route a real payment
// through sandbox credentials.
func TestIsTestModeIsConservative(t *testing.T) {
	for _, in := range []string{"", "live", "garbage", "tes", "testing"} {
		if IsTestMode(in) {
			t.Fatalf("IsTestMode(%q) must be false", in)
		}
	}
	if !IsTestMode("test") || !IsTestMode("TEST") {
		t.Fatal(`IsTestMode must accept "test" case-insensitively`)
	}
}

func TestChefBornTestVsFlipped(t *testing.T) {
	born := ChefProfile{Mode: ChefModeTest}
	if !born.IsBornTest() {
		t.Fatal("a chef that has never been live is born-test")
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	flipped := ChefProfile{Mode: ChefModeTest, FirstLiveAt: &now}
	if flipped.IsBornTest() {
		t.Fatal("a chef that has been live is not born-test")
	}
	if !born.IsTestMode() {
		t.Fatal("a test-mode chef reports test mode")
	}
	if (&ChefProfile{Mode: ChefModeLive}).IsTestMode() {
		t.Fatal("a live chef is not in test mode")
	}
}
