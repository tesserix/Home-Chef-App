package services

// wallet_split.go — funding allocation for wallet-at-checkout (#141).
//
// The customer is charged order.Total. Razorpay Route carves the chef and driver
// payouts OUT of that single captured payment, so a payment-linked transfer can
// never exceed the captured amount. When a customer applies wallet credit W, the
// gateway only captures (Total - W); if W is larger than the platform's margin,
// the capture can no longer cover the chef + driver splits.
//
// The funding model (owner decision, #141): the chef and driver are ALWAYS settled
// in full. We fund each split from the captured payment as far as the capture
// allows, then top up the remainder via a DIRECT transfer from the platform's
// Razorpay balance. The platform absorbs the wallet redemption from its margin
// and — when the redemption exceeds that margin — from its float.
//
// This file is pure (no DB, no gateway calls) and works in paise to stay exact.

// Settlement is one party (chef or driver) that must be paid in full.
type Settlement struct {
	Account string            // Razorpay linked account; "" means withheld/absent → never transferred, stays with the platform
	Amount  int               // paise owed to this party in full
	Hold    bool              // on_hold until delivery confirmation
	Notes   map[string]string // transfer notes (purpose, order number)
}

// FundingPlan describes how a wallet-applied order is funded. For each settled
// account, PaymentTransfers + DirectTopUps sum to the full amount owed.
type FundingPlan struct {
	WalletAppliedPaise int            // credit actually applied (clamped to balance and order total)
	CapturePaise       int            // what the customer pays at the gateway (Total - WalletApplied)
	PaymentTransfers   []TransferSpec // Route transfers funded from the captured payment; their sum is always <= CapturePaise
	DirectTopUps       []TransferSpec // shortfall funded from the platform balance after capture
	FullWallet         bool           // CapturePaise == 0 → no gateway payment needed at all
}

// PlanWalletFunding clamps the requested wallet credit to [0, min(balance, total)]
// and splits each settlement into a payment-funded portion (bounded by the
// remaining capture) and a platform-funded top-up.
//
// Settlements are funded from the capture in the order given, so pass the chef
// first: the chef's (larger) food payout is then covered by real gateway money
// before the driver's, keeping it as a single payment-linked transfer whenever the
// capture allows (which lets Razorpay auto-reverse it on refund).
func PlanWalletFunding(totalPaise, balancePaise, requestedPaise int, settlements []Settlement) FundingPlan {
	wallet := requestedPaise
	if wallet < 0 {
		wallet = 0
	}
	if wallet > balancePaise {
		wallet = balancePaise
	}
	if wallet > totalPaise {
		wallet = totalPaise
	}
	capture := totalPaise - wallet

	plan := FundingPlan{
		WalletAppliedPaise: wallet,
		CapturePaise:       capture,
		FullWallet:         capture == 0,
	}

	remaining := capture
	for _, s := range settlements {
		// Withheld (FSSAI lockout) or absent account, or nothing owed: the
		// platform retains this slice — no transfer of any kind.
		if s.Account == "" || s.Amount <= 0 {
			continue
		}
		fromPayment := s.Amount
		if fromPayment > remaining {
			fromPayment = remaining
		}
		if fromPayment < 0 {
			fromPayment = 0
		}
		topUp := s.Amount - fromPayment
		remaining -= fromPayment

		if fromPayment > 0 {
			plan.PaymentTransfers = append(plan.PaymentTransfers, TransferSpec{
				Account:  s.Account,
				Amount:   fromPayment,
				Currency: "INR",
				Notes:    s.Notes,
				OnHold:   s.Hold,
			})
		}
		if topUp > 0 {
			plan.DirectTopUps = append(plan.DirectTopUps, TransferSpec{
				Account:  s.Account,
				Amount:   topUp,
				Currency: "INR",
				Notes:    s.Notes,
				OnHold:   s.Hold,
			})
		}
	}
	return plan
}
