package services

// checkout_credit_quote_test.go — the DB-backed assembler. The point of this
// layer is that ONE function loads the live balances and allocates, and both the
// /quote endpoint and payment creation call it, so the figure a customer is shown
// and the figure they are charged cannot diverge.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// bothRails is the normal server state: wallet and loyalty checkout both enabled.
var bothRails = CreditFlags{WalletCheckoutEnabled: true, LoyaltyCheckoutEnabled: true}

// Wallet and loyalty are rupee instruments settled through Razorpay Route, so a
// chef settling via Stripe in their own currency takes no credit at all.
func TestBuildCreditQuote_StripeOrdersTakeNoCredit(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 500, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "USD",
		PaymentProvider: "stripe", Subtotal: 100, Total: 118}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: true, UseLoyalty: true}, bothRails)
	require.NoError(t, err)
	require.False(t, q.WalletEnabled)
	require.False(t, q.LoyaltyEnabled)
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, ToPaise(118), q.PayablePaise)
}

// Opting a rail out is an explicit zero — distinct from the rail being disabled,
// which hides the row entirely.
func TestBuildCreditQuote_OptedOutRailContributesNothing(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 500, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "INR",
		PaymentProvider: "razorpay", Subtotal: 1000, ServiceFee: 50, Tax: 50, Total: 1100}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: false, UseLoyalty: false}, bothRails)
	require.NoError(t, err)
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, ToPaise(1100), q.PayablePaise)
	require.True(t, q.WalletEnabled, "still offered — the customer simply declined it")
}

func TestBuildCreditQuote_AppliesLiveWalletBalanceCappedAtFood(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 5000, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "INR",
		PaymentProvider: "razorpay", Subtotal: 1000, DeliveryFee: 100,
		ServiceFee: 50, Tax: 50, Total: 1200}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: true}, bothRails)
	require.NoError(t, err)
	require.Equal(t, ToPaise(1100), q.WalletAppliedPaise, "food + delivery only")
	require.Equal(t, ToPaise(100), q.PayablePaise, "service fee + tax, in cash")
}

// A server flag being off vetoes the rail even with a funded balance.
func TestBuildCreditQuote_ServerFlagOffDisablesTheRail(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 5000, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "INR",
		PaymentProvider: "razorpay", Subtotal: 1000, ServiceFee: 50, Tax: 50, Total: 1100}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: true},
		CreditFlags{WalletCheckoutEnabled: false, LoyaltyCheckoutEnabled: true})
	require.NoError(t, err)
	require.False(t, q.WalletEnabled)
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, ToPaise(1100), q.PayablePaise)
}

// A chef who lowered the delivery fee at accept (#703) already had the difference
// refunded, so the redeemable base must follow the EFFECTIVE fee, not the charged one.
func TestBuildCreditQuote_UsesEffectiveDeliveryFee(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := CreditWallet(db, uid, 5000, models.WalletSourcePromo, nil, "seed", "w-1", nil)
	require.NoError(t, err)

	final := 40.0
	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "INR",
		PaymentProvider: "razorpay", Subtotal: 1000, DeliveryFee: 100,
		DeliveryFeeFinal: &final, ServiceFee: 50, Tax: 50, Total: 1140}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: true}, bothRails)
	require.NoError(t, err)
	require.Equal(t, ToPaise(1040), q.WalletAppliedPaise, "food + the fee actually charged")
	require.Equal(t, ToPaise(100), q.PayablePaise)
}

// Points spend against the order, capped at 10% of the food subtotal.
func TestBuildCreditQuote_SpendsPointsWhenWalletIsEmpty(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, err := EarnLoyalty(db, uid, 100000, models.LoyaltySourceOrder, nil, "seed", "seed-1")
	require.NoError(t, err)

	order := &models.Order{ID: uuid.New(), CustomerID: uid, Currency: "INR",
		PaymentProvider: "razorpay", Subtotal: 1000, ServiceFee: 50, Tax: 50, Total: 1100}
	q, err := BuildCreditQuote(db, order, uid, CreditRequest{UseWallet: true, UseLoyalty: true}, bothRails)
	require.NoError(t, err)
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, ToPaise(100), q.PointsAppliedPaise, "10% of the 1000.00 subtotal")
	require.Equal(t, "per_order_cap", q.LoyaltyLimitReason)
	require.Equal(t, ToPaise(1000), q.PayablePaise)
}
