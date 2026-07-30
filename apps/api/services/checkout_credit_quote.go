package services

import (
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// checkout_credit_quote.go — loads the live balances and config for an order and
// runs the pure allocator.
//
// BOTH the /quote endpoint and payment creation call this. That is the whole
// point: the previous design had the client compute a payable from its own cached
// balance and post a rupee amount, while the server independently re-clamped
// against live state — so any drift showed the customer one figure and charged
// another. There is now exactly one computation.

// CreditRequest is the customer's intent. The amount fields are optional: a nil
// amount with UseX true means "apply as much as the ceilings allow".
type CreditRequest struct {
	UseWallet     bool     `json:"useWallet"`
	WalletAmount  *float64 `json:"walletAmount"`
	UseLoyalty    bool     `json:"useLoyalty"`
	LoyaltyPoints *float64 `json:"loyaltyPoints"`
}

// CreditFlags carries the server feature gates into the allocation.
type CreditFlags struct {
	WalletCheckoutEnabled  bool
	LoyaltyCheckoutEnabled bool
}

// BuildCreditQuote assembles CreditInputs from live state and allocates.
//
// Wallet and loyalty are rupee instruments settled through Razorpay Route, so a
// Stripe order — a chef settling in their own currency — takes no credit at all,
// and both rails report disabled rather than silently applying zero.
func BuildCreditQuote(db *gorm.DB, order *models.Order, userID uuid.UUID, req CreditRequest, flags CreditFlags) (CreditQuote, error) {
	inr := order.Currency == "" || strings.EqualFold(order.Currency, "INR")
	razorpay := !strings.EqualFold(order.PaymentProvider, "stripe")
	rupeeRail := inr && razorpay

	in := CreditInputs{
		SubtotalPaise: ToPaise(order.Subtotal),
		DiscountPaise: ToPaise(order.Discount),
		// The EFFECTIVE fee, not the charged one: when a chef lowered the delivery
		// fee at accept (#703) the difference was already refunded, so the higher
		// figure is not part of what this order can still redeem against.
		DeliveryFeePaise: ToPaise(order.EffectiveDeliveryFee()),
		PlatformFeePaise: ToPaise(order.PlatformFee),
		TaxPaise:         ToPaise(order.Tax),
		TotalPaise:       ToPaise(order.Total),
		Cfg:              GetLoyaltyConfig(db),
		WalletDisabled:   !rupeeRail || !flags.WalletCheckoutEnabled,
		LoyaltyDisabled:  !rupeeRail || !flags.LoyaltyCheckoutEnabled,
	}

	if !in.WalletDisabled {
		w, err := WalletBalance(db, userID)
		if err != nil {
			return CreditQuote{}, err
		}
		if w != nil {
			in.WalletBalancePaise = ToPaise(w.Balance)
		}
	}
	if !in.LoyaltyDisabled {
		acct, err := LoyaltyBalance(db, userID)
		if err != nil {
			return CreditQuote{}, err
		}
		if acct != nil {
			in.PointsBalance = acct.Balance
		}
		spent, err := MonthlyRedeemedPaise(db, userID)
		if err != nil {
			return CreditQuote{}, err
		}
		in.MonthlyRedeemedPaise = spent
	}

	// An opted-out rail is an explicit request for zero, which is distinct from
	// the rail being disabled: declined still reports enabled, so the UI shows the
	// row with its toggle off rather than hiding it.
	zeroPaise, zeroPoints := 0, float64(0)
	switch {
	case !req.UseWallet:
		in.RequestedWalletPaise = &zeroPaise
	case req.WalletAmount != nil:
		p := ToPaise(*req.WalletAmount)
		in.RequestedWalletPaise = &p
	}
	switch {
	case !req.UseLoyalty:
		in.RequestedPoints = &zeroPoints
	case req.LoyaltyPoints != nil:
		in.RequestedPoints = req.LoyaltyPoints
	}

	return PlanCheckoutCredit(in), nil
}
