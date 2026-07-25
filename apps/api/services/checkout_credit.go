package services

import "math"

// checkout_credit.go — the single allocation authority for wallet + loyalty credit
// applied at checkout.
//
// The rule: credit may fund FOOD and DELIVERY only. The platform service fee and
// GST are always settled in real money — the platform remits GST to the government
// and the service fee is its revenue, so neither may be funded by a liability the
// platform itself issued.
//
// Pure: no DB, no gateway. Paise throughout to stay exact, matching wallet_split.go.
// The DB-backed assembler in checkout_credit_quote.go feeds it; handlers never
// reimplement any part of this arithmetic.

// CreditInputs is everything the allocation needs, in paise.
type CreditInputs struct {
	SubtotalPaise    int
	DiscountPaise    int
	DeliveryFeePaise int
	ServiceFeePaise  int
	TaxPaise         int
	TotalPaise       int

	WalletBalancePaise int
	PointsBalance      float64

	// nil means "auto" — apply as much as the ceilings allow. A non-nil value is
	// the customer's explicit slider/typed choice and is clamped, never expanded.
	RequestedWalletPaise *int
	RequestedPoints      *float64

	MonthlyRedeemedPaise int
	Cfg                  LoyaltyConfig

	// WalletDisabled / LoyaltyDisabled veto a rail regardless of balance — a
	// feature flag being off, or a non-INR (Stripe) order.
	WalletDisabled  bool
	LoyaltyDisabled bool
}

// CreditQuote is the authoritative breakdown. The client renders it verbatim and
// never recomputes any field.
type CreditQuote struct {
	RedeemableCapPaise int
	NonRedeemablePaise int

	WalletBalancePaise int
	WalletAppliedPaise int
	WalletMaxPaise     int

	PointsBalance       float64
	PointsAppliedPoints float64
	PointsAppliedPaise  int
	PointsMaxPoints     float64

	PayablePaise int

	// LoyaltyLimitReason tells the UI WHY the points row is capped, so it can
	// explain the limit without reimplementing the cap logic to guess.
	// "balance" | "per_order_cap" | "monthly_cap" | "order_covered" | "disabled"
	LoyaltyLimitReason string

	WalletEnabled  bool
	LoyaltyEnabled bool
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// PlanCheckoutCredit allocates wallet credit and loyalty points against an order.
//
// The ceiling is the MINIMUM of two expressions:
//
//	food + delivery      — encodes the product rule directly
//	total − fees − tax   — a structural guarantee that payable >= fees + tax holds
//	                       however Total is composed, now or later
//
// Two correct behaviours fall out of taking the minimum rather than trusting
// either branch alone. A tip is excluded, because it is in Total but not in the
// food branch. And an inclusive-tax order — where Tax is embedded in Subtotal
// rather than added to Total — still pays GST in cash, because there the second
// branch is the smaller one.
func PlanCheckoutCredit(in CreditInputs) CreditQuote {
	foodBranch := in.SubtotalPaise - in.DiscountPaise + in.DeliveryFeePaise
	feeBranch := in.TotalPaise - in.ServiceFeePaise - in.TaxPaise
	redeemable := foodBranch
	if feeBranch < redeemable {
		redeemable = feeBranch
	}
	if redeemable > in.TotalPaise {
		redeemable = in.TotalPaise
	}
	if redeemable < 0 {
		redeemable = 0
	}

	walletEnabled := !in.WalletDisabled
	loyaltyEnabled := !in.LoyaltyDisabled && in.Cfg.Enabled && in.Cfg.RedeemRate > 0

	q := CreditQuote{
		RedeemableCapPaise: redeemable,
		NonRedeemablePaise: in.TotalPaise - redeemable,
		WalletBalancePaise: in.WalletBalancePaise,
		PointsBalance:      in.PointsBalance,
		WalletEnabled:      walletEnabled,
		LoyaltyEnabled:     loyaltyEnabled,
	}

	// --- Wallet first. Wallet credit is a booked liability the platform has
	// already issued; loyalty points are a capped, expiring promotional currency.
	// Burning the liability first is both the owner's preference and the better
	// balance-sheet outcome, and it means a small order leaves the points intact.
	if walletEnabled {
		q.WalletMaxPaise = clampInt(in.WalletBalancePaise, 0, redeemable)
	}
	q.WalletAppliedPaise = q.WalletMaxPaise
	if in.RequestedWalletPaise != nil {
		q.WalletAppliedPaise = clampInt(*in.RequestedWalletPaise, 0, q.WalletMaxPaise)
	}

	// --- Loyalty second, against whatever the wallet left behind.
	room := redeemable - q.WalletAppliedPaise
	if room < 0 {
		room = 0
	}
	if !loyaltyEnabled {
		q.LoyaltyLimitReason = "disabled"
	} else {
		// Four ceilings; the binding one is reported so the UI can explain itself.
		maxPaise, reason := pointsToPaise(in.PointsBalance, in.Cfg.RedeemRate), "balance"
		if perOrder := int(math.Floor(float64(in.SubtotalPaise) * in.Cfg.MaxRedeemPct)); perOrder < maxPaise {
			maxPaise, reason = perOrder, "per_order_cap"
		}
		if monthly := ToPaise(in.Cfg.MonthlyRedeemCap) - in.MonthlyRedeemedPaise; monthly < maxPaise {
			maxPaise, reason = monthly, "monthly_cap"
		}
		if room < maxPaise {
			maxPaise, reason = room, "order_covered"
		}
		if maxPaise < 0 {
			maxPaise = 0
		}
		q.LoyaltyLimitReason = reason

		// Floor to whole points so the rupee figure is an exact multiple of the
		// redeem rate and the points ledger never carries a fraction.
		q.PointsMaxPoints = paiseToWholePoints(maxPaise, in.Cfg.RedeemRate)
		pts := q.PointsMaxPoints
		if in.RequestedPoints != nil {
			pts = math.Floor(*in.RequestedPoints)
			if pts < 0 {
				pts = 0
			}
			if pts > q.PointsMaxPoints {
				pts = q.PointsMaxPoints
			}
		}
		q.PointsAppliedPoints = pts
		q.PointsAppliedPaise = pointsToPaise(pts, in.Cfg.RedeemRate)
	}

	q.PayablePaise = in.TotalPaise - q.WalletAppliedPaise - q.PointsAppliedPaise
	if q.PayablePaise < 0 { // defence in depth; the clamps above already prevent it
		q.PayablePaise = 0
	}
	return q
}

// pointsToPaise converts points to paise, rounding DOWN so a redemption can never
// hand out more value than the points are worth.
func pointsToPaise(points, rate float64) int {
	if points <= 0 || rate <= 0 {
		return 0
	}
	return int(math.Floor(points * rate * 100))
}

// paiseToWholePoints is the inverse, rounding DOWN to a whole point.
func paiseToWholePoints(paise int, rate float64) float64 {
	if paise <= 0 || rate <= 0 {
		return 0
	}
	return math.Floor(float64(paise) / (rate * 100))
}
