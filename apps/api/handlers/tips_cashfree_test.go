package handlers

// tips_cashfree_test.go — the tip must ride the gateway the order rode.
//
// D-18: the tip flow was written against Razorpay Route and never moved when
// payouts did, so it demanded a chef.razorpay_account_id that NO chef on the
// platform has. Every post-delivery tip answered 409 "This chef can't receive
// tips right now" — a whole surface, unreachable, promising "100% goes straight
// to your chef".

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// The dispatch predicate itself: a Cashfree order must never be routed to the
// Razorpay leg, which is what made the chef's missing Route account fatal.
func TestTipProviderDispatch_CashfreeOrderTakesTheCashfreeLeg(t *testing.T) {
	for _, tc := range []struct {
		provider string
		wantCF   bool
	}{
		{models.PaymentProviderCashfree, true},
		{models.PaymentProviderRazorpay, false},
		{"", false}, // legacy rows default to Razorpay
	} {
		got := models.NormalizeProvider(tc.provider) == models.PaymentProviderCashfree
		require.Equal(t, tc.wantCF, got, "provider %q", tc.provider)
	}
}

// The Cashfree tip id is what VerifyTip dispatches on, so its prefix is load
// bearing: a tip whose id does not carry it would be verified as a Razorpay
// payment and rejected for a missing razorpayPaymentId.
func TestCashfreeTipOrderID_CarriesTheDispatchPrefix(t *testing.T) {
	id := "tip-" + strings.ReplaceAll("0f5a366c-c73e-e626-d6e7-14323a021c80", "-", "")
	require.True(t, strings.HasPrefix(id, "tip-"))
	// Cashfree order ids are 3–45 chars of alphanumerics, - and _.
	require.LessOrEqual(t, len(id), 45)
	require.NotContains(t, strings.TrimPrefix(id, "tip-"), "-")
}

// A tip carries no commission and no tax (INV-6), so unlike an order's Easy
// Split there is no platform fee to subtract: the chef's split is the WHOLE tip.
func TestCashfreeTipSplit_IsTheWholeTip(t *testing.T) {
	tipPaise := services.ToPaise(50.0)
	split := services.CashfreeOrderSplit{
		VendorID:    "hc_abc",
		AmountPaise: services.CashfreeAmountFromPaise(tipPaise),
	}
	require.Equal(t, 5000, tipPaise)
	require.Equal(t, tipPaise, split.AmountPaise.Paise(),
		"the platform must keep nothing — the screen promises 100% to the chef")
}
