package services

// subscription_promo.go — applying a promo / trial-discount code to a PLATFORM
// subscription (#269). Reuses the order promo engine (ComputePromoDiscount,
// CheckPromoEligibility, ClaimPromoRedemption, UserPromoRedemptions) so there's
// one source of truth for the discount math + caps.
//
// Rules specific to subscriptions:
//   - The code must be PLATFORM-funded — the subscription fee is platform revenue,
//     so chef-funded codes make no sense here and are rejected.
//   - It is applied ONCE, to the subscription's FIRST generated invoice, then the
//     stored code is cleared (a signup / trial incentive, not every cycle).
//   - The eligibility "base" is the plan amount (so min-spend + percentage caps
//     work against the subscription price).

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// ErrPromoNotForSubscriptions is returned when a chef-funded code is used on a
// subscription (only platform-funded codes apply to subscription fees).
var ErrPromoNotForSubscriptions = errors.New("This promo code can't be used for subscriptions")

// promoContextFor builds the eligibility context for a user against a base amount,
// resolving the per-user usage + order counts the rules need. Shared by the
// subscription preview (read) and the at-invoice application.
func promoContextFor(db *gorm.DB, promo *models.PromoCode, userID uuid.UUID, base float64) PromoContext {
	var userUsageCount int64
	if promo.PerUserLimit > 0 {
		db.Model(&models.PromoCodeUsage{}).
			Where("promo_code_id = ? AND user_id = ?", promo.ID, userID).
			Count(&userUsageCount)
	}
	var userOrderCount int64
	if promo.ApplicableTo != "all" {
		db.Model(&models.Order{}).Where("customer_id = ?", userID).Count(&userOrderCount)
	}
	return PromoContext{
		Now:            time.Now(),
		UserOrderCount: userOrderCount,
		UserUsageCount: userUsageCount,
		OrderSubtotal:  base,
	}
}

// ValidateSubscriptionPromo validates a code for a subscription plan amount and
// returns the previewed discount + the promo. Used at signup (ChoosePlan) to
// confirm + store the code; the real discount is recomputed and atomically
// claimed at invoice time so the budget isn't consumed by a trial that never bills.
func ValidateSubscriptionPromo(db *gorm.DB, code string, userID uuid.UUID, planAmount float64) (float64, *models.PromoCode, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, nil, errors.New("Promo code is required")
	}
	var promo models.PromoCode
	if err := db.Where("code = ? AND is_active = ?", code, true).First(&promo).Error; err != nil {
		return 0, nil, ErrPromoInactive
	}
	if promo.FundingSource == models.PromoFundingChef {
		return 0, nil, ErrPromoNotForSubscriptions
	}
	discount := ComputePromoDiscount(&promo, planAmount)
	if err := CheckPromoEligibility(&promo, promoContextFor(db, &promo, userID, planAmount), discount); err != nil {
		return 0, nil, err
	}
	return discount, &promo, nil
}

// ApplySubscriptionPromoToInvoice applies the subscription's stored promo to its
// first invoice: recomputes the discount against planAmount, re-validates and
// atomically claims it (usage + budget race-safe), records the redemption against
// the subscription, and clears the stored code so it never applies twice. Returns
// the discount actually applied — 0 when there's no promo, it's no longer
// eligible, or its budget is exhausted (the caller then bills full price). MUST
// run inside the invoice-generation transaction.
func ApplySubscriptionPromoToInvoice(tx *gorm.DB, sub *models.Subscription, planAmount float64) float64 {
	if sub.PromoCodeID == nil {
		return 0
	}
	promoID := *sub.PromoCodeID

	// One-time, race-safe: atomically flip the stored code to NULL and only the
	// tx that wins that conditional UPDATE proceeds to apply the discount. Invoice
	// generation runs in fire-and-forget goroutines per delivery, so two could race
	// for the same first invoice; this guarantees the promo is applied (and its
	// budget consumed) exactly once. A rolled-back tx undoes the clear too.
	claim := tx.Model(&models.Subscription{}).
		Where("id = ? AND promo_code_id = ?", sub.ID, promoID).
		Update("promo_code_id", nil)
	if claim.Error != nil || claim.RowsAffected != 1 {
		return 0
	}
	sub.PromoCodeID = nil

	var promo models.PromoCode
	if err := tx.First(&promo, "id = ?", promoID).Error; err != nil {
		return 0
	}
	if promo.FundingSource == models.PromoFundingChef {
		return 0
	}
	discount := ComputePromoDiscount(&promo, planAmount)
	if discount <= 0 {
		return 0
	}
	if err := CheckPromoEligibility(&promo, promoContextFor(tx, &promo, sub.UserID, planAmount), discount); err != nil {
		return 0
	}
	claimed, err := ClaimPromoRedemption(tx, promo.ID, discount)
	if err != nil || !claimed {
		return 0
	}
	usage := models.PromoCodeUsage{
		PromoCodeID:    promo.ID,
		UserID:         sub.UserID,
		SubscriptionID: &sub.ID,
		Discount:       discount,
	}
	if err := tx.Create(&usage).Error; err != nil {
		return 0
	}
	return discount
}

// ValidateSubscriptionPromoDB is the database.DB convenience wrapper for handlers.
func ValidateSubscriptionPromoDB(code string, userID uuid.UUID, planAmount float64) (float64, *models.PromoCode, error) {
	return ValidateSubscriptionPromo(database.DB, code, userID, planAmount)
}
