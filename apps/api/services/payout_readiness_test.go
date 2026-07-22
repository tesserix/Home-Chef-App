package services

import (
	"testing"
	"time"
)

func TestParsePayoutGateLevel(t *testing.T) {
	cases := []struct {
		raw  string
		want PayoutGateLevel
		ok   bool
	}{
		{"off", PayoutGateOff, true},
		{"method_on_file", PayoutGateMethodOnFile, true},
		{"verified", PayoutGateVerified, true},
		{"  METHOD_ON_FILE  ", PayoutGateMethodOnFile, true}, // tolerate admin typing
		{"", "", false},
		{"yes", "", false},
		{"true", "", false},
	}
	for _, tc := range cases {
		got, ok := ParsePayoutGateLevel(tc.raw)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Fatalf("ParsePayoutGateLevel(%q) = %q, %v; want %q, %v", tc.raw, got, ok, tc.want, tc.ok)
		}
	}
}

func TestVerifiedLevelIsDowngradedUntilVerificationExists(t *testing.T) {
	// Selecting `verified` before #740 ships would block every chef on the
	// platform, because nothing can be verified yet. Downgrade rather than
	// obey — an admin typo must not be able to halt all trading.
	got := resolveGateLevel(PayoutGateVerified, false)
	if got != PayoutGateMethodOnFile {
		t.Fatalf("resolveGateLevel(verified, unsupported) = %q, want method_on_file", got)
	}
	// Once verification exists the same input is honoured verbatim.
	if got := resolveGateLevel(PayoutGateVerified, true); got != PayoutGateVerified {
		t.Fatalf("resolveGateLevel(verified, supported) = %q, want verified", got)
	}
}

func TestEvaluatePayoutReadiness(t *testing.T) {
	cases := []struct {
		name         string
		level        PayoutGateLevel
		methodOnFile bool
		verified     bool
		wantReady    bool
		wantReason   string
	}{
		{"gate off lets anyone trade", PayoutGateOff, false, false, true, ""},
		{"method on file satisfies method gate", PayoutGateMethodOnFile, true, false, true, ""},
		{"no method fails method gate", PayoutGateMethodOnFile, false, false, false, ReasonPayoutMethodMissing},
		{"method but unverified fails verified gate", PayoutGateVerified, true, false, false, ReasonPayoutMethodUnverified},
		{"verified satisfies verified gate", PayoutGateVerified, true, true, true, ""},
		{"no method fails verified gate on the missing reason", PayoutGateVerified, false, false, false, ReasonPayoutMethodMissing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ready, reason := EvaluatePayoutReadiness(tc.level, tc.methodOnFile, tc.verified)
			if ready != tc.wantReady || reason != tc.wantReason {
				t.Fatalf("got (%v, %q), want (%v, %q)", ready, reason, tc.wantReady, tc.wantReason)
			}
		})
	}
}

func TestMissingBeatsUnverified(t *testing.T) {
	// A chef with nothing on file should be told to add a method, not told it
	// is unverified — the wrong message sends them to the wrong screen.
	_, reason := EvaluatePayoutReadiness(PayoutGateVerified, false, false)
	if reason != ReasonPayoutMethodMissing {
		t.Fatalf("reason = %q, want %q", reason, ReasonPayoutMethodMissing)
	}
}

func TestPayoutGraceActive(t *testing.T) {
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		raw  string
		want bool
	}{
		{"unset means no grace", "", false},
		{"future timestamp is grace", "2026-08-02T00:00:00Z", true},
		{"past timestamp has expired", "2026-07-31T00:00:00Z", false},
		{"exact boundary has expired", "2026-08-01T12:00:00Z", false},
		{"garbage is not grace", "soon", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := PayoutGraceActive(tc.raw, now); got != tc.want {
				t.Fatalf("PayoutGraceActive(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestGarbageGraceValueDoesNotSilentlyDisableTheGate(t *testing.T) {
	// A malformed grace setting must fail closed (gate enforced), not open.
	// Failing open would mean a typo silently lets unpayable chefs trade.
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	if PayoutGraceActive("not-a-date", now) {
		t.Fatal("a malformed grace value must not grant grace")
	}
}

func TestReadinessIsReportedEvenDuringGrace(t *testing.T) {
	// Grace suppresses enforcement, not reporting: the vendor app still needs
	// to know it should nag, and admin still needs the unpayable count.
	r := buildReadiness(PayoutGateMethodOnFile, false, false, true)
	if r.Enforced {
		t.Fatal("grace must suppress enforcement")
	}
	if r.Ready {
		t.Fatal("readiness must still report not-ready during grace")
	}
	if r.ReasonCode != ReasonPayoutMethodMissing {
		t.Fatalf("ReasonCode = %q, want %q", r.ReasonCode, ReasonPayoutMethodMissing)
	}
}

func TestBlocksOnlyWhenNotReadyAndEnforced(t *testing.T) {
	cases := []struct {
		name      string
		ready     bool
		grace     bool
		level     PayoutGateLevel
		wantBlock bool
	}{
		{"ready, enforced", true, false, PayoutGateMethodOnFile, false},
		{"not ready, enforced", false, false, PayoutGateMethodOnFile, true},
		{"not ready, in grace", false, true, PayoutGateMethodOnFile, false},
		{"not ready, gate off", false, false, PayoutGateOff, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			methodOnFile := tc.ready
			r := buildReadiness(tc.level, methodOnFile, false, tc.grace)
			if r.Blocks() != tc.wantBlock {
				t.Fatalf("Blocks() = %v, want %v (readiness %+v)", r.Blocks(), tc.wantBlock, r)
			}
		})
	}
}
