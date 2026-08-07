package services

import "github.com/homechef/api/services/money"

// gateway_money.go — the gateway-neutral helpers every payment rail shares.
// They lived in razorpay.go until #1086 removed that client; nothing here was
// ever Razorpay-specific.

// isPlaceholderValue treats blank strings and the literal "placeholder"
// (used as a seed value by Helm bootstrap) as "not configured".
func isPlaceholderValue(v string) bool {
	return v == "" || v == "placeholder"
}

// ToPaise converts a rupee amount to integer paise, identically to
// ToMinor(amount, "INR") so every gateway path mints the same minor-unit value
// (#524/#396). New code should prefer the money package's Paise type directly.
func ToPaise(amount float64) int {
	return int(money.FromRupees(amount))
}

// FromPaise converts paise to a rupee amount.
func FromPaise(paise int) float64 {
	return money.Paise(paise).Rupees()
}

// ValidateCapturedPayment is the single hard gate every payment "verify" leg
// (order, catering deposit, featured-ad, tip, group share) MUST apply: a
// gateway-fetched payment has to be captured, belong to the EXPECTED gateway
// order, and cover the EXPECTED amount (paise). Status / OrderID / Amount come
// from the gateway, so the client can't forge them — without this binding, any
// captured payment on the merchant account (e.g. a ₹1 charge, or a payment from
// an unrelated order) could be replayed to settle a different object for free.
// Returns (true, "") when valid, else (false, reason).
func ValidateCapturedPayment(paymentStatus, paymentOrderID, expectedOrderID string, paymentAmountPaise, expectedPaise int) (bool, string) {
	if paymentStatus != "captured" {
		return false, "Payment not captured"
	}
	if expectedOrderID == "" {
		// The verify leg was called without first creating the gateway order.
		return false, "Start the payment first"
	}
	if paymentOrderID != expectedOrderID {
		return false, "Payment does not belong to this order"
	}
	if paymentAmountPaise < expectedPaise {
		return false, "Payment amount does not match the expected amount"
	}
	return true, ""
}
