package handlers

// chef_order_totals_test.go — unit tests for the order cancel/refund money math
// (partial coverage of #8: per-line partial-refund amount + order
// subtotal/tax/total recompute, and paise rounding). The device-driven E2E
// items in #8 (push delivery, gateway refund timing, invoice auto-email) are
// out of scope here and remain manual QA.

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/homechef/api/models"
)

func approx(t *testing.T, got, want float64, msg string) {
	t.Helper()
	assert.InDelta(t, want, got, 1e-9, msg)
}

// ── lineRefundAmount ─────────────────────────────────────────────────────────

func TestLineRefundAmount_IncludesProportionalTaxShare(t *testing.T) {
	// Order: subtotal 1000, tax 100 (10%). A line worth 250 should refund its
	// 250 subtotal + its 25% share of the 100 tax = 25. Total 275.
	got := lineRefundAmount(250, 1000, 100)
	approx(t, got, 275, "line refund = subtotal + proportional tax share")
}

func TestLineRefundAmount_WholeOrderLine(t *testing.T) {
	// A single line that is the whole order gets all the tax back.
	got := lineRefundAmount(800, 800, 64)
	approx(t, got, 864, "sole line refunds full subtotal + full tax")
}

func TestLineRefundAmount_ZeroOrderSubtotal_NoTaxShare(t *testing.T) {
	// Guard: never divide by zero. With no order subtotal, refund is just the
	// line subtotal (no tax share to allocate).
	got := lineRefundAmount(120, 0, 50)
	approx(t, got, 120, "zero order subtotal → no tax share")
}

func TestLineRefundAmount_NoTax(t *testing.T) {
	got := lineRefundAmount(300, 900, 0)
	approx(t, got, 300, "no order tax → refund is just the line subtotal")
}

// ── recomputeOrderTotals ─────────────────────────────────────────────────────

func TestRecomputeOrderTotals_ExcludesCancelledLine(t *testing.T) {
	// Order had two lines (600 + 400 = 1000 subtotal), tax 100 (10%),
	// delivery 50, service 20, tip 30, discount 10. Cancel the 400 line.
	items := []models.OrderItem{
		{Subtotal: 600, IsCancelled: false},
		{Subtotal: 400, IsCancelled: true},
	}
	subtotal, tax, total := recomputeOrderTotals(items, 1000, 100, 50, 20, 30, 10)

	approx(t, subtotal, 600, "subtotal drops to the surviving line")
	// Tax scales with surviving subtotal: 100 * 600/1000 = 60.
	approx(t, tax, 60, "tax scales proportionally to surviving subtotal")
	// Total = 600 + 50 + 20 + 60 + 30 - 10 = 750.
	approx(t, total, 750, "total recomputed with fees/tip/discount preserved")
}

func TestRecomputeOrderTotals_NoCancellations_Unchanged(t *testing.T) {
	items := []models.OrderItem{
		{Subtotal: 600, IsCancelled: false},
		{Subtotal: 400, IsCancelled: false},
	}
	subtotal, tax, total := recomputeOrderTotals(items, 1000, 100, 50, 20, 30, 10)
	approx(t, subtotal, 1000, "subtotal unchanged when nothing cancelled")
	approx(t, tax, 100, "tax unchanged")
	// 1000 + 50 + 20 + 100 + 30 - 10 = 1190.
	approx(t, total, 1190, "total unchanged")
}

func TestRecomputeOrderTotals_AllCancelled_FeesAndTipRemain(t *testing.T) {
	// Every line cancelled → subtotal + tax go to zero, but the non-line
	// charges (delivery, service, tip) and discount still apply. The handler
	// deliberately does NOT auto-flip status here; a whole-order cancel is the
	// explicit path.
	items := []models.OrderItem{
		{Subtotal: 600, IsCancelled: true},
		{Subtotal: 400, IsCancelled: true},
	}
	subtotal, tax, total := recomputeOrderTotals(items, 1000, 100, 50, 20, 30, 10)
	approx(t, subtotal, 0, "all lines cancelled → zero subtotal")
	approx(t, tax, 0, "zero subtotal → zero tax")
	// 0 + 50 + 20 + 0 + 30 - 10 = 90.
	approx(t, total, 90, "non-line charges survive a full line cancellation")
}

func TestRecomputeOrderTotals_ZeroCurrentSubtotal_NoTax(t *testing.T) {
	items := []models.OrderItem{{Subtotal: 0, IsCancelled: false}}
	subtotal, tax, total := recomputeOrderTotals(items, 0, 100, 10, 0, 0, 0)
	approx(t, subtotal, 0, "zero subtotal in, zero out")
	approx(t, tax, 0, "guard divide-by-zero → zero tax")
	approx(t, total, 10, "only the delivery fee remains")
}

// ── roundPaise ───────────────────────────────────────────────────────────────

func TestRoundPaise_RoundsRupeesToPaise(t *testing.T) {
	cases := []struct {
		rupees float64
		paise  float64
	}{
		{1, 100},
		{0, 0},
		{12.34, 1234},
		{99.99, 9999},
		{0.001, 0},   // 0.1 paise → 0
		{0.999, 100}, // 99.9 paise → 100
		{2.999, 300}, // 299.9 paise → 300
		{2.5, 250},   // exact
		{-2.5, -250}, // negative branch (round away from zero)
		{-12.34, -1234},
	}
	for _, c := range cases {
		got := roundPaise(c.rupees)
		if math.Abs(got-c.paise) > 1e-9 {
			t.Errorf("roundPaise(%v) = %v, want %v", c.rupees, got, c.paise)
		}
	}
}

// ── CancelReason validation (input boundary for the cancel handlers) ─────────

func TestCancelReason_IsValid(t *testing.T) {
	valid := []models.CancelReason{
		models.CancelReason("out_of_ingredient"),
		models.CancelReason("equipment_failure"),
		models.CancelReason("customer_request"),
		models.CancelReason("other"),
	}
	for _, r := range valid {
		assert.Truef(t, r.IsValid(), "%q should be a valid cancel reason", r)
	}
	for _, r := range []models.CancelReason{"", "lol", "OUT_OF_INGREDIENT"} {
		assert.Falsef(t, r.IsValid(), "%q should be rejected", r)
	}
}
