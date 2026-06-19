package services

// promo.go — the promo-code discount + eligibility engine (#39).
//
// Pure (no DB, no gateway): the caller fetches the per-customer/order state into
// a PromoContext and these functions decide the discount and whether the code is
// allowed. Keeping the rules here makes them unit-testable and shared by the
// /promo/validate preview and the CreateOrder checkout path (no duplication).

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// Promo discount types.
const (
	PromoDiscountPercentage = "percentage"
	PromoDiscountFixed      = "fixed"
)

// PromoContext is the per-customer/order state the eligibility rules need. The
// caller resolves these from the DB; the rules themselves stay DB-free.
type PromoContext struct {
	Now            time.Time
	UserOrderCount int64     // orders this user has placed (for new/returning targeting)
	UserUsageCount int64     // times this user has already redeemed this code (per-user cap)
	OrderSubtotal  float64   // the base the discount + min-order check apply to
	ChefID         uuid.UUID // the order's chef; uuid.Nil when unknown (preview without a cart chef)
}

// Promo eligibility failures. Returned as values so the caller can surface the
// message and, for the min-order case, the threshold.
var (
	ErrPromoInactive        = errors.New("Invalid promo code")
	ErrPromoNotStarted      = errors.New("Promo code is not yet active")
	ErrPromoExpired         = errors.New("Promo code has expired")
	ErrPromoUsageLimit      = errors.New("Promo code usage limit reached")
	ErrPromoPerUserLimit    = errors.New("You have already used this promo code the maximum number of times")
	ErrPromoNewUsersOnly    = errors.New("This promo code is only for new users")
	ErrPromoReturningOnly   = errors.New("This promo code is only for returning users")
	ErrPromoWrongChef       = errors.New("This promo code isn't valid for this kitchen")
	ErrPromoBudgetExhausted = errors.New("This promo code is no longer available")
)

// ErrPromoMinOrder carries the unmet minimum so the API can return a richer body.
type ErrPromoMinOrder struct{ MinOrderAmount float64 }

func (e ErrPromoMinOrder) Error() string { return "Minimum order amount not met" }

// ComputePromoDiscount returns the discount for a promo against a base amount
// (the order subtotal). Percentage discounts honour MaxDiscount; every discount
// is clamped to [0, base] and rounded. Pure — no validity/eligibility checks.
func ComputePromoDiscount(promo *models.PromoCode, base float64) float64 {
	var discount float64
	if promo.DiscountType == PromoDiscountPercentage {
		discount = base * promo.DiscountValue / 100
		if promo.MaxDiscount > 0 && discount > promo.MaxDiscount {
			discount = promo.MaxDiscount
		}
	} else {
		discount = promo.DiscountValue
	}
	if discount > base {
		discount = base
	}
	if discount < 0 {
		discount = 0
	}
	return models.RoundAmount(discount)
}

// CheckPromoEligibility validates a promo against the customer/order context and
// the computed discount. Returns nil when the code may be applied, else a typed
// error describing why not. The budget check uses the computed discount so a
// redemption that would push BudgetSpent past BudgetCap is refused up front.
func CheckPromoEligibility(promo *models.PromoCode, ctx PromoContext, discount float64) error {
	if !promo.IsActive {
		return ErrPromoInactive
	}
	if ctx.Now.Before(promo.ValidFrom) {
		return ErrPromoNotStarted
	}
	if promo.ValidUntil != nil && ctx.Now.After(*promo.ValidUntil) {
		return ErrPromoExpired
	}
	if promo.UsageLimit > 0 && promo.UsageCount >= promo.UsageLimit {
		return ErrPromoUsageLimit
	}
	if promo.PerUserLimit > 0 && ctx.UserUsageCount >= int64(promo.PerUserLimit) {
		return ErrPromoPerUserLimit
	}
	if promo.MinOrderAmount > 0 && ctx.OrderSubtotal < promo.MinOrderAmount {
		return ErrPromoMinOrder{MinOrderAmount: promo.MinOrderAmount}
	}
	switch promo.ApplicableTo {
	case "new_users":
		if ctx.UserOrderCount > 0 {
			return ErrPromoNewUsersOnly
		}
	case "returning_users":
		if ctx.UserOrderCount == 0 {
			return ErrPromoReturningOnly
		}
	}
	// Chef-funded promos only apply to their funding chef's orders. Skip the
	// check in preview (ChefID unknown / no ChefID set on the promo).
	if promo.FundingSource == models.PromoFundingChef && promo.ChefID != nil && ctx.ChefID != uuid.Nil {
		if *promo.ChefID != ctx.ChefID {
			return ErrPromoWrongChef
		}
	}
	if promo.BudgetCap > 0 && models.RoundAmount(promo.BudgetSpent+discount) > promo.BudgetCap {
		return ErrPromoBudgetExhausted
	}
	return nil
}

// ChefFundedPortion returns how much of an applied discount the chef funds: the
// whole discount for a chef-funded promo, otherwise 0 (platform absorbs it).
func ChefFundedPortion(promo *models.PromoCode, discount float64) float64 {
	if promo != nil && promo.FundingSource == models.PromoFundingChef {
		return models.RoundAmount(discount)
	}
	return 0
}

// ClaimPromoRedemption atomically reserves one redemption of a promo, enforcing
// the global usage limit AND the budget cap inside a single conditional UPDATE so
// concurrent checkouts can never push usage_count past usage_limit or
// budget_spent past budget_cap. Returns true when the slot was claimed (the
// caller should then record the PromoCodeUsage row), false when the code is
// exhausted. Run inside the order transaction. This is the race-safe counterpart
// to the read-only CheckPromoEligibility budget/usage checks.
func ClaimPromoRedemption(tx *gorm.DB, promoID uuid.UUID, discount float64) (bool, error) {
	res := tx.Model(&models.PromoCode{}).
		Where("id = ? AND (usage_limit = 0 OR usage_count < usage_limit) AND (budget_cap = 0 OR budget_spent + ? <= budget_cap)",
			promoID, discount).
		Updates(map[string]interface{}{
			"usage_count":  gorm.Expr("usage_count + 1"),
			"budget_spent": gorm.Expr("budget_spent + ?", discount),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}
