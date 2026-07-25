package services

import (
	"github.com/homechef/api/models"
)

// refund_funding_split.go — dividing a refund across the rails that funded the
// order.
//
// This matters because the on-demand refundable base is the WHOLE total (see
// RemainingRefundable) and the cancellation policy then applies a percentage — so
// a refund is routinely SMALLER than the credit applied, and the split rule is
// load-bearing rather than cosmetic.

// FundingSplit is one refund divided across the rails that paid for the order.
// The three parts always sum to exactly the requested refund.
type FundingSplit struct {
	WalletPaise  int // → wallet, automatically, instant
	LoyaltyPaise int // → wallet as rupees (owner decision), NOT restored as points
	CardPaise    int // → the existing customer choice: wallet or original method
}

// SplitRefundByFunding divides refundPaise pro-rata across wallet, loyalty and
// card. Each credit rail is capped at what it has not already been refunded, so
// repeated partial refunds can never return more than that rail ever funded.
//
// The CARD slice takes the remainder rather than being computed independently,
// which guarantees the parts sum to exactly refundPaise — three separate roundings
// would otherwise create or destroy a paise.
func SplitRefundByFunding(order *models.Order, refundPaise int) FundingSplit {
	if refundPaise <= 0 {
		return FundingSplit{}
	}
	totalPaise := ToPaise(order.Total)
	if totalPaise <= 0 {
		return FundingSplit{CardPaise: refundPaise}
	}

	share := func(funded, alreadyReturned float64) int {
		v := refundPaise * ToPaise(funded) / totalPaise
		remaining := ToPaise(funded) - ToPaise(alreadyReturned)
		if remaining < 0 {
			remaining = 0
		}
		if v > remaining {
			v = remaining
		}
		if v < 0 {
			v = 0
		}
		return v
	}

	s := FundingSplit{
		WalletPaise:  share(order.WalletApplied, order.WalletRefunded),
		LoyaltyPaise: share(order.LoyaltyApplied, order.LoyaltyRefunded),
	}
	s.CardPaise = refundPaise - s.WalletPaise - s.LoyaltyPaise
	if s.CardPaise < 0 {
		s.CardPaise = 0
	}
	return s
}
