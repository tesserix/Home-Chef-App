package payouts

import (
	"testing"
	"time"
)

// #741 — the per-order release decision. Pure: every input is passed in so the
// decision recorded on a blocked order can be re-derived later.

var govNow = time.Date(2026, 7, 22, 12, 0, 0, 0, time.UTC)

func clean() ReleaseInput {
	return ReleaseInput{
		Now:                 govNow,
		DeliveredAt:         govNow.Add(-3 * time.Hour),
		Maturation:          2 * time.Hour,
		AutomationEnabled:   true,
		SettlementActivated: true,
		RefundOpen:          false,
		RecoveryBalance:     Zero(CurrencyINR),
		DeliveredOrderCount: 10,
		RampOrders:          3,
		OrderTotal:          Money{Minor: 50_000, Currency: CurrencyINR},
		ReviewAbove:         Money{Minor: 500_000, Currency: CurrencyINR},
	}
}

func hasReason(d ReleaseDecision, r BlockReason) bool {
	for _, got := range d.Reasons {
		if got == r {
			return true
		}
	}
	return false
}

func TestDecideRelease_ReleasesACleanMaturedOrder(t *testing.T) {
	d := DecideRelease(clean())
	if !d.Release {
		t.Fatalf("clean order must release, blocked by %v", d.Reasons)
	}
	if len(d.Reasons) != 0 {
		t.Fatalf("reasons = %v, want none", d.Reasons)
	}
}

func TestDecideRelease_HoldsUntilMaturation(t *testing.T) {
	in := clean()
	in.DeliveredAt = govNow.Add(-time.Hour) // only 1h of a 2h window
	d := DecideRelease(in)
	if d.Release {
		t.Fatal("an order inside the maturation window must not release")
	}
	if !hasReason(d, BlockNotMatured) {
		t.Fatalf("reasons = %v, want %q", d.Reasons, BlockNotMatured)
	}
}

func TestDecideRelease_MaturationBoundaryIsInclusive(t *testing.T) {
	// Off-by-one here silently delays every payout by a full sweep cycle.
	in := clean()
	in.DeliveredAt = govNow.Add(-2 * time.Hour) // exactly matured
	if !DecideRelease(in).Release {
		t.Fatal("an order maturing exactly now is releasable")
	}
}

func TestDecideRelease_BlocksOnEachGuardrail(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*ReleaseInput)
		want   BlockReason
	}{
		{"automation off", func(in *ReleaseInput) { in.AutomationEnabled = false }, BlockAutomationOff},
		{"settlement not activated", func(in *ReleaseInput) { in.SettlementActivated = false }, BlockSettlementNotActivated},
		{"refund open", func(in *ReleaseInput) { in.RefundOpen = true }, BlockRefundOpen},
		{"recovery balance", func(in *ReleaseInput) {
			in.RecoveryBalance = Money{Minor: 25_000, Currency: CurrencyINR}
		}, BlockRecoveryBalance},
		{"new chef ramp", func(in *ReleaseInput) { in.DeliveredOrderCount = 1 }, BlockNewChefRamp},
		{"above review threshold", func(in *ReleaseInput) {
			in.OrderTotal = Money{Minor: 900_000, Currency: CurrencyINR}
		}, BlockAboveReviewThreshold},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := clean()
			tc.mutate(&in)
			d := DecideRelease(in)
			if d.Release {
				t.Fatal("must not release")
			}
			if !hasReason(d, tc.want) {
				t.Fatalf("reasons = %v, want %q", d.Reasons, tc.want)
			}
		})
	}
}

func TestDecideRelease_ReportsEveryReasonAtOnce(t *testing.T) {
	// Short-circuiting makes an admin fix one problem, retry, and immediately
	// hit the next. They must see the whole picture in one pass.
	in := clean()
	in.SettlementActivated = false
	in.RefundOpen = true
	in.DeliveredOrderCount = 0

	d := DecideRelease(in)
	if len(d.Reasons) < 3 {
		t.Fatalf("reasons = %v, want all three reported together", d.Reasons)
	}
}

func TestDecideRelease_MarksHardBlocksAsNotOverridable(t *testing.T) {
	// An admin forcing a release to an unactivated account strands the money
	// rather than delivering it; paying against an open refund may settle
	// before the claw-back lands.
	if BlockSettlementNotActivated.Overridable() {
		t.Fatal("settlement activation is not an admin decision")
	}
	if BlockRefundOpen.Overridable() {
		t.Fatal("an open refund is not an admin decision")
	}
	if !BlockRecoveryBalance.Overridable() {
		t.Fatal("a recovery balance is an admin judgement call")
	}
	if !BlockNewChefRamp.Overridable() || !BlockAboveReviewThreshold.Overridable() {
		t.Fatal("ramp and threshold holds exist to be reviewed and released")
	}
}

func TestDecideRelease_ZeroRampOrdersDisablesTheRampCheck(t *testing.T) {
	// The ramp is runtime-tunable; setting it to 0 must turn it off rather
	// than block every chef forever.
	in := clean()
	in.RampOrders = 0
	in.DeliveredOrderCount = 0
	if hasReason(DecideRelease(in), BlockNewChefRamp) {
		t.Fatal("a zero ramp must disable the check")
	}
}

func TestDecideRelease_ZeroThresholdDisablesTheReviewCheck(t *testing.T) {
	in := clean()
	in.ReviewAbove = Zero(CurrencyINR)
	in.OrderTotal = Money{Minor: 10_000_000, Currency: CurrencyINR}
	if hasReason(DecideRelease(in), BlockAboveReviewThreshold) {
		t.Fatal("a zero threshold must disable the check")
	}
}

func TestDecideRelease_BlocksWhenTheThresholdComparisonFails(t *testing.T) {
	// An uncomparable amount must block rather than slip through: a Money
	// built without a Currency would otherwise skip the review gate entirely.
	in := clean()
	in.OrderTotal = Money{Minor: 900_000} // no currency set
	d := DecideRelease(in)
	if d.Release {
		t.Fatal("an uncomparable order total must not auto-release")
	}
	if !hasReason(d, BlockAboveReviewThreshold) {
		t.Fatalf("reasons = %v, want %q", d.Reasons, BlockAboveReviewThreshold)
	}
}
