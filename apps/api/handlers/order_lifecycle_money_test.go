package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/models"
)

// order_lifecycle_money_test.go — the automatable core of #8 (full order
// lifecycle E2E). The gateway / push / invoice-PDF / auto-email / NATS legs of
// #8 need live infra and are covered by the on-device runbook; the money
// invariants — per-line partial refund → order recompute → whole-order cancel →
// earnings reflect the refund — are pure and deterministic, so this walks the
// whole lifecycle through the real helpers (lineRefundAmount, recomputeOrderTotals,
// computeOrderBreakdown) in one scenario and pins the invariants end-to-end.
//
// Scenario: a 2-line order, ₹600 + ₹400 (subtotal 1000, tax 50 = 5%, delivery
// 40, tip 20). The customer flow: chef can't fulfil line B → partial refund;
// then the whole order is cancelled; then earnings are computed for the
// survived-then-refunded state.
func TestOrderLifecycle_RefundRecomputeAndEarnings(t *testing.T) {
	const (
		origSubtotal = 1000.0
		origTax      = 50.0
		deliveryFee  = 40.0
		serviceFee   = 0.0
		tip          = 20.0
		discount     = 0.0
		lineA        = 600.0
		lineB        = 400.0
	)

	items := []models.OrderItem{
		{ID: uuid.New(), Subtotal: lineA},
		{ID: uuid.New(), Subtotal: lineB},
	}

	// ── Step 1: per-line "can't fulfil" on line B ──────────────────────────
	// Refund = line subtotal + its proportional tax share, against the ORIGINAL
	// subtotal/tax so concurrent cancels split tax consistently.
	refundB := lineRefundAmount(lineB, origSubtotal, origTax)
	if refundB != 420.0 { // 400 + 50*(400/1000)
		t.Fatalf("line B refund: got %.2f, want 420.00", refundB)
	}

	// Line B is now cancelled; the order recomputes around the survivor (A).
	items[1].IsCancelled = true
	sub1, tax1, total1 := recomputeOrderTotals(items, origSubtotal, origTax, deliveryFee, serviceFee, tip, discount)
	if sub1 != 600.0 {
		t.Fatalf("subtotal after line B: got %.2f, want 600.00", sub1)
	}
	if tax1 != 30.0 { // 50 * (600/1000)
		t.Fatalf("tax after line B: got %.2f, want 30.00", tax1)
	}
	if total1 != 690.0 { // 600 + 40 + 30 + 20
		t.Fatalf("total after line B: got %.2f, want 690.00", total1)
	}
	// Line A must be untouched — the order continues for the other item.
	if items[0].IsCancelled {
		t.Fatal("line A must still be active after line B is cancelled")
	}

	// ── Step 2: whole-order cancel (cancel the remaining line A) ────────────
	// Full refund of what's still owed, computed against the CURRENT (post-step-1)
	// subtotal/tax so the two per-line refunds together return exactly the
	// original item money plus the original tax — nothing lost, nothing double-paid.
	refundA := lineRefundAmount(lineA, sub1, tax1)
	if refundA != 630.0 { // 600 + 30*(600/600)
		t.Fatalf("line A refund: got %.2f, want 630.00", refundA)
	}
	if refundB+refundA != origSubtotal+origTax { // 420 + 630 == 1000 + 50
		t.Fatalf("total refunded %.2f must equal original subtotal+tax %.2f",
			refundB+refundA, origSubtotal+origTax)
	}

	// With every line cancelled the item money is zero; fees + tip still remain
	// on the order (they aren't line-scoped).
	items[0].IsCancelled = true
	sub2, tax2, total2 := recomputeOrderTotals(items, sub1, tax1, deliveryFee, serviceFee, tip, discount)
	if sub2 != 0 || tax2 != 0 {
		t.Fatalf("all-cancelled subtotal/tax: got %.2f/%.2f, want 0/0", sub2, tax2)
	}
	if total2 != 60.0 { // 0 + 40 delivery + 20 tip
		t.Fatalf("all-cancelled total: got %.2f, want 60.00 (fees + tip remain)", total2)
	}

	// ── Step 3: earnings reflect the refund ─────────────────────────────────
	// Had only line B been refunded (order delivered with line A), the chef's
	// earnings read the RECOMPUTED item revenue (600), not the original 1000 —
	// so the refund flows through to the payout, not just the customer's card.
	row := earningsOrderRow{
		OrderID:       uuid.New(),
		OrderNumber:   "HC-LIFECYCLE",
		CompletedAt:   time.Now(),
		ItemRevenue:   sub1, // survived subtotal after the partial refund
		Tax:           tax1, // #390: the SURVIVING food GST also flows to the chef
		DeliveryFee:   deliveryFee,
		ChefTip:       tip,
		DeliveryState: "Maharashtra",
	}
	got := computeOrderBreakdown(row, "Maharashtra", 0)
	if got.ItemRevenue != 600.0 {
		t.Fatalf("earnings item revenue: got %.2f, want 600.00 (must reflect the refund)", got.ItemRevenue)
	}
	if got.PlatformCommission != 36.0 { // 0.06 * 600, not 0.06 * 1000
		t.Fatalf("earnings commission: got %.2f, want 36.00 (6%% of the refunded-down revenue)", got.PlatformCommission)
	}
	// #390: gross = itemRevenue 600 + surviving tax 30 + chef tip 20 = 650. The
	// delivery fee (40) is the driver's and is excluded from the chef's gross.
	if got.Gross != 650.0 {
		t.Fatalf("earnings gross: got %.2f, want 650.00", got.Gross)
	}
}
