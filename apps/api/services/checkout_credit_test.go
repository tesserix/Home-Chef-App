package services

// checkout_credit_test.go — the allocation rules for wallet + loyalty credit at
// checkout. The load-bearing assertion is that credit NEVER funds the platform
// service fee or GST: the platform remits GST to the government and the service
// fee is its revenue, so neither may be paid with a liability the platform issued.

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func creditTestCfg() LoyaltyConfig {
	return LoyaltyConfig{Enabled: true, RedeemRate: 0.05, MinRedeem: 500,
		MaxRedeemPct: 0.10, MonthlyRedeemCap: 300, ExpiryDays: 365}
}

// The owner's rule, on the exact order from the reported screenshot.
func TestPlanCheckoutCredit_NeverFundsFeesOrTax(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 258000, DeliveryFeePaise: 20000, PlatformFeePaise: 12874,
		TaxPaise: 14544, TotalPaise: 305418,
		WalletBalancePaise: 1000000, PointsBalance: 100000, Cfg: creditTestCfg(),
	})
	require.Equal(t, 278000, q.RedeemableCapPaise, "food + delivery only")
	require.Equal(t, 27418, q.PayablePaise, "service fee + tax, in cash")
	require.GreaterOrEqual(t, q.PayablePaise, 12874+14544)
}

// Wallet is consumed before loyalty; on a small order the points survive entirely.
func TestPlanCheckoutCredit_WalletFirstLeavesPointsUntouched(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 34000, TotalPaise: 34000,
		WalletBalancePaise: 50000, PointsBalance: 100000, Cfg: creditTestCfg(),
	})
	require.Equal(t, 34000, q.WalletAppliedPaise)
	require.Equal(t, 0, q.PointsAppliedPaise)
	require.Equal(t, float64(0), q.PointsAppliedPoints)
	require.Equal(t, 0, q.PayablePaise)
	require.Equal(t, "order_covered", q.LoyaltyLimitReason)
}

// A tip is a gratuity for the chef or rider, not food — credit must never absorb it.
func TestPlanCheckoutCredit_TipIsNotRedeemable(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 110000, // 10000 paise tip
		WalletBalancePaise: 500000, Cfg: creditTestCfg(),
	})
	require.Equal(t, 100000, q.RedeemableCapPaise)
	require.Equal(t, 10000, q.PayablePaise, "the tip stays in cash")
}

// Inclusive-tax regime: Tax lives INSIDE Subtotal and is not added to Total, so the
// food branch would over-count. The Total−PlatformFee−Tax branch must win.
func TestPlanCheckoutCredit_InclusiveTaxStillPaidInCash(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, PlatformFeePaise: 5000,
		TaxPaise: 4762, TotalPaise: 105000,
		WalletBalancePaise: 500000, Cfg: creditTestCfg(),
	})
	require.Equal(t, 95238, q.RedeemableCapPaise)
	require.Equal(t, 9762, q.PayablePaise)
	require.GreaterOrEqual(t, q.PayablePaise, 5000+4762)
}

// The 10%-of-subtotal per-order cap finally binds — it never has before.
func TestPlanCheckoutCredit_PerOrderLoyaltyCapBinds(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		PointsBalance: 100000, Cfg: creditTestCfg(),
	})
	require.Equal(t, 10000, q.PointsAppliedPaise, "10% of 1000.00")
	require.Equal(t, "per_order_cap", q.LoyaltyLimitReason)
}

func TestPlanCheckoutCredit_MonthlyCapBinds(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 1000000, TotalPaise: 1000000,
		PointsBalance: 1000000, MonthlyRedeemedPaise: 28000, Cfg: creditTestCfg(),
	})
	require.Equal(t, 2000, q.PointsAppliedPaise, "300.00 cap − 280.00 already used")
	require.Equal(t, "monthly_cap", q.LoyaltyLimitReason)
}

// MinRedeem is a redeem-to-wallet dust rule; at checkout any balance is spendable.
func TestPlanCheckoutCredit_MinRedeemWaivedAtCheckout(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		PointsBalance: 180, Cfg: creditTestCfg(),
	})
	require.Equal(t, 900, q.PointsAppliedPaise, "180 pts x 0.05")
	require.Equal(t, float64(180), q.PointsAppliedPoints)
	require.Equal(t, "balance", q.LoyaltyLimitReason)
}

// An explicit slider/typed value is clamped, never expanded.
func TestPlanCheckoutCredit_ExplicitRequestsAreClamped(t *testing.T) {
	w, p := 5000, float64(100)
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		WalletBalancePaise: 90000, PointsBalance: 100000,
		RequestedWalletPaise: &w, RequestedPoints: &p, Cfg: creditTestCfg(),
	})
	require.Equal(t, 5000, q.WalletAppliedPaise, "not expanded to the max")
	require.Equal(t, 500, q.PointsAppliedPaise, "100 pts honoured, not maxed")
}

// Dialling the wallet down must NOT silently expand loyalty into the gap — the
// customer reducing one rail is not consent to spend more of the other.
func TestPlanCheckoutCredit_LoweringWalletDoesNotExpandExplicitLoyalty(t *testing.T) {
	w, p := 0, float64(100)
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		WalletBalancePaise: 100000, PointsBalance: 100000,
		RequestedWalletPaise: &w, RequestedPoints: &p, Cfg: creditTestCfg(),
	})
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, 500, q.PointsAppliedPaise, "still exactly the 100 points asked for")
}

func TestPlanCheckoutCredit_DisabledConfigSpendsNoPoints(t *testing.T) {
	c := creditTestCfg()
	c.Enabled = false
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		PointsBalance: 100000, Cfg: c,
	})
	require.Equal(t, 0, q.PointsAppliedPaise)
	require.False(t, q.LoyaltyEnabled)
	require.Equal(t, "disabled", q.LoyaltyLimitReason)
}

func TestPlanCheckoutCredit_VetoedRailsContributeNothing(t *testing.T) {
	q := PlanCheckoutCredit(CreditInputs{
		SubtotalPaise: 100000, TotalPaise: 100000,
		WalletBalancePaise: 100000, PointsBalance: 100000,
		WalletDisabled: true, LoyaltyDisabled: true, Cfg: creditTestCfg(),
	})
	require.Equal(t, 0, q.WalletAppliedPaise)
	require.Equal(t, 0, q.PointsAppliedPaise)
	require.Equal(t, 100000, q.PayablePaise)
	require.False(t, q.WalletEnabled)
	require.False(t, q.LoyaltyEnabled)
}

// The invariants must hold for EVERY input, including adversarial ones: discounts
// exceeding the subtotal, fees exceeding the total, empty and enormous balances.
func TestPlanCheckoutCredit_InvariantsHoldUnderFuzz(t *testing.T) {
	r := rand.New(rand.NewSource(20260725))
	for i := 0; i < 20000; i++ {
		sub := r.Intn(500000)
		del := r.Intn(50000)
		svc := r.Intn(30000)
		tax := r.Intn(30000)
		tip := r.Intn(20000)
		disc := r.Intn(sub + 1000) // may exceed the subtotal
		total := sub + del + svc + tax + tip - disc
		if total < 0 {
			total = 0
		}
		q := PlanCheckoutCredit(CreditInputs{
			SubtotalPaise: sub, DiscountPaise: disc, DeliveryFeePaise: del,
			PlatformFeePaise: svc, TaxPaise: tax, TotalPaise: total,
			WalletBalancePaise:   r.Intn(1000000),
			PointsBalance:        float64(r.Intn(200000)),
			MonthlyRedeemedPaise: r.Intn(40000),
			Cfg:                  creditTestCfg(),
		})
		require.Equal(t, total, q.WalletAppliedPaise+q.PointsAppliedPaise+q.PayablePaise,
			"credit + payable must reconstitute the total exactly")
		require.LessOrEqual(t, q.WalletAppliedPaise+q.PointsAppliedPaise, q.RedeemableCapPaise)
		require.GreaterOrEqual(t, q.PayablePaise, 0)
		require.GreaterOrEqual(t, q.WalletAppliedPaise, 0)
		require.GreaterOrEqual(t, q.PointsAppliedPaise, 0)
		if svc+tax <= total {
			require.GreaterOrEqual(t, q.PayablePaise, svc+tax,
				"fees and tax are never funded by credit")
		}
	}
}
