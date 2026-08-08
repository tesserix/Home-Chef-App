package services

import (
	"github.com/homechef/api/models"
)

// easy_split_refund.go — who bears a refund on a split order.
//
// With Easy Split the chef's net share leaves the platform on release, so a
// refund the platform bears alone is money it never held. Whether that happens
// is an ACCOUNT-LEVEL Cashfree setting we do not control from here: proportional
// vendor debiting is on by default, but it "can be turned off in your account",
// and with it off the whole refund lands on the merchant balance. Nothing
// recovers that — a fully-split chef has no platform-side payout for
// ApplyRecoveryDeduction to net against.
//
// Sending refund_splits explicitly "replaces the proportional default", so it is
// not additive and cannot double-debit. Stating the share ourselves makes the
// outcome the same under either account setting, and lets us control the parts
// the default gets wrong for us: the capture basis excludes store credit the
// gateway never saw, and multi-leg rounding is differenced rather than
// re-floored per leg.
//
// Cashfree, Easy Split FAQ: "By default, the balance of respective vendors is
// debited in the proportion of their commission for the order."

// BuildRefundSplits is the refund_splits for one refund on an order, or nil for
// a refund the platform bears alone.
//
// priorRefundedPaise is everything refunded on this order BEFORE this leg, and
// must be the same basis the caller used for its idempotency key — a handler
// holding a lock-reserved figure passes that, not order.RefundAmount, which was
// read before the reserve.
//
// nil when: the order was never split, the chef has no vendor to debit, or the
// vendor's share is already fully reversed. nil means "no refund_splits field",
// which is the pre-existing full-capture behaviour.
func BuildRefundSplits(order *models.Order, priorRefundedPaise, refundPaise int) []CashfreeVendorSplit {
	if order == nil || order.GatewaySplitPaise <= 0 || refundPaise <= 0 {
		return nil
	}
	vendorID := order.Chef.VendorID()
	if vendorID == "" {
		return nil
	}

	share := vendorRefundPaise(
		order.GatewaySplitPaise,
		gatewayCapturePaise(order),
		priorRefundedPaise,
		refundPaise,
	)
	if share <= 0 {
		return nil
	}
	// Cashfree rejects splits above the refund amount, and a rejected refund is a
	// customer who is never paid. Only reachable on a split that outruns its own
	// capture basis, but the cost of being wrong is the whole refund.
	if share > refundPaise {
		share = refundPaise
	}
	return []CashfreeVendorSplit{{
		VendorID:    vendorID,
		AmountPaise: CashfreeAmountFromPaise(share),
	}}
}

// gatewayCapturePaise is what the customer actually paid the gateway, which is
// the basis the split was calculated against — not the order total, which
// includes any store credit and loyalty the gateway never saw.
func gatewayCapturePaise(order *models.Order) int {
	return ToPaise(order.Total) - ToPaise(order.WalletApplied) - ToPaise(order.LoyaltyApplied)
}

// vendorRefundPaise is the vendor's portion of ONE refund leg.
//
// Computed as the difference between two cumulative floors rather than a
// proportion of this leg alone. Independent rounding per leg drifts: three
// legs of a ₹500 capture against a ₹380 share each round up a fraction of a
// paise and the vendor is debited more than they were ever paid. Differencing
// the running total makes the legs sum to exactly floor(refunded × share /
// capture) however many there are, and caps the series at the share itself.
//
// Flooring puts the sub-paise remainder on the platform, never the chef.
func vendorRefundPaise(splitPaise, capturePaise, priorRefundedPaise, refundPaise int) int {
	if capturePaise <= 0 {
		return 0
	}
	cumulative := func(refunded int) int {
		if refunded <= 0 {
			return 0
		}
		if refunded >= capturePaise {
			return splitPaise
		}
		return refunded * splitPaise / capturePaise
	}

	share := cumulative(priorRefundedPaise+refundPaise) - cumulative(priorRefundedPaise)
	if share < 0 {
		return 0
	}
	return share
}
