package services

// wallet_split.go — funding allocation for wallet-at-checkout (#141).
//
// The customer is charged order.Total. When they apply wallet credit W, the
// gateway captures (Total - W) and the platform absorbs the redemption from its
// margin — and, when the redemption exceeds that margin, from its float. The chef
// is settled in full either way, on the Easy Split release or the weekly statement.
//
// The Route transfer specs this file used to carve out of the capture went with
// the transfer rail in #1086; what remains is the clamp, which decides both what
// the customer is charged and whether a gateway charge is needed at all.
//
// Pure (no DB, no gateway calls) and works in paise to stay exact.

// FundingPlan describes how a wallet-applied order is funded.
type FundingPlan struct {
	WalletAppliedPaise int  // credit actually applied (clamped to balance and order total)
	CapturePaise       int  // what the customer pays at the gateway (Total - WalletApplied)
	FullWallet         bool // CapturePaise == 0 → no gateway payment needed at all
}

// PlanWalletFunding clamps the requested wallet credit to [0, min(balance, total)].
func PlanWalletFunding(totalPaise, balancePaise, requestedPaise int) FundingPlan {
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

	return FundingPlan{
		WalletAppliedPaise: wallet,
		CapturePaise:       capture,
		FullWallet:         capture == 0,
	}
}
