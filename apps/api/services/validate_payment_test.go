package services

// validate_payment_test.go — the shared payment-verify gate (security audit).
// Two peripheral verify legs (catering deposit, featured-ad) previously skipped
// the order+amount binding the core order path enforces, letting a captured ₹1
// payment settle a different object for free. ValidateCapturedPayment is now the
// single hard gate; these pin its exact contract.

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCapturedPayment(t *testing.T) {
	const order = "order_ABC123"
	const due = 50000 // ₹500 in paise

	cases := []struct {
		name        string
		status      string
		payOrderID  string
		expOrderID  string
		payPaise    int
		expPaise    int
		wantOK      bool
		wantReasonC string // substring the reason must contain (when !ok)
	}{
		{"happy path", "captured", order, order, due, due, true, ""},
		{"overpaid is fine", "captured", order, order, due + 100, due, true, ""},
		{"not captured", "authorized", order, order, due, due, false, "not captured"},
		{"failed status", "failed", order, order, due, due, false, "not captured"},
		// The reported exploit #1: verify called without create → no stored order id.
		{"no expected order (verify without create)", "captured", order, "", due, due, false, "Start the payment first"},
		// The reported exploit: reuse a captured payment from a DIFFERENT order.
		{"order mismatch (payment reuse)", "captured", "order_OTHER", order, due, due, false, "does not belong"},
		// The ₹1-charge exploit: captured but underpays.
		{"underpaid (₹1 reuse)", "captured", order, order, 100, due, false, "does not match"},
		{"one paise short", "captured", order, order, due - 1, due, false, "does not match"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, reason := ValidateCapturedPayment(tc.status, tc.payOrderID, tc.expOrderID, tc.payPaise, tc.expPaise)
			require.Equal(t, tc.wantOK, ok, "reason: %q", reason)
			if !tc.wantOK {
				require.Contains(t, reason, tc.wantReasonC)
			} else {
				require.Empty(t, reason)
			}
		})
	}
}
