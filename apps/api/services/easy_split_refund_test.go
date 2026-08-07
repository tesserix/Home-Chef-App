package services

import (
	"testing"

	"github.com/homechef/api/models"
)

// A split order: ₹500 captured, ₹380 of it settled to the chef's vendor at
// capture. The platform holds the remaining ₹120 (commission, GST, TDS, fees).
func splitOrder(totalRupees float64, splitPaise int) *models.Order {
	o := &models.Order{Total: totalRupees, GatewaySplitPaise: splitPaise}
	o.Chef.CashfreeVendorID = "hc_vendor1"
	return o
}

func TestBuildRefundSplitsFullRefundReversesTheWholeVendorShare(t *testing.T) {
	order := splitOrder(500, 38000)

	splits := BuildRefundSplits(order, 0, 50000)

	if len(splits) != 1 {
		t.Fatalf("want 1 refund split, got %d", len(splits))
	}
	if splits[0].VendorID != "hc_vendor1" {
		t.Errorf("vendor id = %q, want hc_vendor1", splits[0].VendorID)
	}
	if got := splits[0].AmountPaise.Paise(); got != 38000 {
		t.Errorf("vendor refund = %d paise, want the whole 38000 share", got)
	}
}

func TestBuildRefundSplitsPartialRefundIsProRata(t *testing.T) {
	order := splitOrder(500, 38000)

	// Half the capture refunded → half the vendor's share comes back.
	splits := BuildRefundSplits(order, 0, 25000)

	if len(splits) != 1 {
		t.Fatalf("want 1 refund split, got %d", len(splits))
	}
	if got := splits[0].AmountPaise.Paise(); got != 19000 {
		t.Errorf("vendor refund = %d paise, want 19000", got)
	}
}

// The reason the builder works on cumulative totals rather than per-call
// proportions: independent rounding on each leg can drift past the share.
func TestBuildRefundSplitsSuccessivePartialsNeverExceedTheShare(t *testing.T) {
	order := splitOrder(500, 38000)

	// Three refunds of ₹166.67, ₹166.67, ₹166.66 — together the full capture.
	legs := []int{16667, 16667, 16666}
	total, refunded := 0, 0
	for _, leg := range legs {
		splits := BuildRefundSplits(order, refunded, leg)
		if len(splits) != 1 {
			t.Fatalf("want 1 refund split per leg, got %d", len(splits))
		}
		total += splits[0].AmountPaise.Paise()
		refunded += leg
	}

	if total != 38000 {
		t.Errorf("vendor refunded %d paise across three legs, want exactly the 38000 share", total)
	}
}

func TestBuildRefundSplitsClampsAtTheVendorShare(t *testing.T) {
	order := splitOrder(500, 38000)
	order.RefundAmount = 500 // already fully refunded

	splits := BuildRefundSplits(order, 50000, 5000)

	if len(splits) != 0 {
		t.Fatalf("nothing is left of the vendor's share; want no split, got %+v", splits)
	}
}

func TestBuildRefundSplitsNonSplitOrderSendsNoSplits(t *testing.T) {
	order := splitOrder(500, 0)

	if splits := BuildRefundSplits(order, 0, 50000); splits != nil {
		t.Errorf("a full-capture order must send no refund_splits, got %+v", splits)
	}
}

func TestBuildRefundSplitsWithoutAVendorSendsNoSplits(t *testing.T) {
	order := splitOrder(500, 38000)
	order.Chef.CashfreeVendorID = ""

	if splits := BuildRefundSplits(order, 0, 50000); splits != nil {
		t.Errorf("no vendor to debit; want no refund_splits, got %+v", splits)
	}
}

func TestBuildRefundSplitsNilOrderIsSafe(t *testing.T) {
	if splits := BuildRefundSplits(nil, 0, 50000); splits != nil {
		t.Errorf("want no refund_splits for a nil order, got %+v", splits)
	}
}

// Credit-funded orders are never split (BuildOrderSplit refuses them), so a
// capture basis that ignored credit would only ever be wrong on data that
// cannot exist. Assert the guard anyway — it is one line and it is the
// assumption the pro-rata maths rests on.
func TestBuildRefundSplitsIgnoresOrdersFundedPartlyByCredit(t *testing.T) {
	order := splitOrder(500, 38000)
	order.WalletApplied = 100

	splits := BuildRefundSplits(order, 0, 40000)

	if len(splits) != 1 {
		t.Fatalf("want 1 refund split, got %d", len(splits))
	}
	// Capture was ₹400, not ₹500: a full refund of the capture must still
	// reverse the whole share rather than 4/5ths of it.
	if got := splits[0].AmountPaise.Paise(); got != 38000 {
		t.Errorf("vendor refund = %d paise, want the whole 38000 share", got)
	}
}

// A handler that reserved its refund under a row lock holds a prior-refunded
// figure the loaded order struct does not: order.RefundAmount was read before
// the reserve. The basis has to come from the caller, or the leg is priced
// against the wrong point on the cumulative curve.
func TestBuildRefundSplitsUsesTheCallersPriorRefundedBasis(t *testing.T) {
	order := splitOrder(500, 38001)

	// ₹400 already refunded under the lock; the loaded row still says nothing is.
	splits := BuildRefundSplits(order, 40000, 10000)

	if len(splits) != 1 {
		t.Fatalf("want 1 refund split, got %d", len(splits))
	}
	// 38001 - floor(40000×38001/50000) = 38001 - 30400 = 7601. Reading the stale
	// zero off the order would price this leg at 7600.
	if got := splits[0].AmountPaise.Paise(); got != 7601 {
		t.Errorf("vendor refund = %d paise, want 7601", got)
	}
}

// Cashfree rejects a refund whose splits exceed the refund amount, and a
// rejected refund is a customer who is never paid. The split can outrun the leg
// on inconsistent data — a capture basis smaller than the recorded split, which
// is what a wallet credit applied after capture looks like.
func TestBuildRefundSplitsNeverExceedsTheRefundItself(t *testing.T) {
	order := splitOrder(500, 38000)
	order.WalletApplied = 200 // capture basis ₹300, below the ₹380 recorded split

	splits := BuildRefundSplits(order, 0, 30000)

	if len(splits) != 1 {
		t.Fatalf("want 1 refund split, got %d", len(splits))
	}
	if got := splits[0].AmountPaise.Paise(); got > 30000 {
		t.Errorf("vendor refund = %d paise, must not exceed the %d paise refund", got, 30000)
	}
}

func TestVendorRefundPaiseRoundsInThePlatformsFavour(t *testing.T) {
	// ₹100 capture, ₹33.33 vendor share, ₹10 refunded → 3.333 paise per rupee.
	// 333 is the floor; 334 would hand the vendor a paise it never received.
	if got := vendorRefundPaise(3333, 10000, 0, 1000); got != 333 {
		t.Errorf("vendor refund = %d paise, want 333 (floored)", got)
	}
}
