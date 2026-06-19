package services

// meal_subscription_billing.go — the customer meal-subscription billing math +
// invoice generation (#281). Flat delivery fee; a skipped day (before cutoff) or a
// missed day (chef no-show) credits the NEXT cycle (owner decision). The pure math
// is unit-tested; the Razorpay recurring rail (CreateSubscription/UPI-Autopay
// mandate) lives in razorpay.go and is owner-tested against the live gateway.

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// PerDayMealCost is what one delivery DAY costs the customer: the per-meal price ×
// the number of meal slots they take (lunch and/or dinner). This is the unit a
// skip/missed day is credited at.
func PerDayMealCost(perMealPrice float64, numSlots int) float64 {
	if perMealPrice < 0 || numSlots <= 0 {
		return 0
	}
	return models.RoundAmount(perMealPrice * float64(numSlots))
}

// ComputeMealCycleCharge applies the carried credit balance to the gross cycle
// amount: the customer is charged max(0, cycle − credit); any credit beyond the
// cycle carries forward. Returns (chargeNow, remainingCredit).
func ComputeMealCycleCharge(cycleAmount, creditBalance float64) (charge, remainingCredit float64) {
	if creditBalance < 0 {
		creditBalance = 0
	}
	if cycleAmount < 0 {
		cycleAmount = 0
	}
	charge = cycleAmount - creditBalance
	if charge < 0 {
		remainingCredit = -charge
		charge = 0
	}
	return models.RoundAmount(charge), models.RoundAmount(remainingCredit)
}

// mealInvoiceTaxRate returns the subscription tax rate for a country (reuses the
// existing tax config). 0 when unknown so generation never fails.
func mealInvoiceTaxRate(countryCode string) float64 {
	cfg, err := GetTaxConfig(countryCode)
	if err != nil || cfg == nil {
		return 0
	}
	return cfg.SubscriptionPercent / 100
}

// GenerateMealCycleInvoice creates the next-cycle invoice for a subscription,
// applying the carried credit balance, freezing the amounts, and zeroing the
// consumed credit on the subscription — all in one transaction. Returns the
// invoice. The caller then charges it via the gateway (or the webhook marks paid).
func GenerateMealCycleInvoice(db *gorm.DB, sub *models.MealSubscription, periodStart, periodEnd time.Time) (*models.MealSubscriptionInvoice, error) {
	charge, remainingCredit := ComputeMealCycleCharge(sub.CycleAmount, sub.CreditBalance)
	creditApplied := models.RoundAmount(sub.CreditBalance - remainingCredit)
	// Tiffin subscriptions are India-domiciled; tax on the net charge.
	tax := models.RoundAmount(charge * mealInvoiceTaxRate("IN"))

	var invoice models.MealSubscriptionInvoice
	err := db.Transaction(func(tx *gorm.DB) error {
		invoice = models.MealSubscriptionInvoice{
			MealSubscriptionID: sub.ID,
			InvoiceNumber:      mealInvoiceNumber(),
			Status:             models.MealInvoiceStatusPending,
			CycleAmount:        models.RoundAmount(sub.CycleAmount),
			CreditApplied:      creditApplied,
			Amount:             charge,
			TaxAmount:          tax,
			TotalAmount:        models.RoundAmount(charge + tax),
			Currency:           sub.Currency,
			PeriodStart:        periodStart,
			PeriodEnd:          periodEnd,
		}
		if err := tx.Create(&invoice).Error; err != nil {
			return err
		}
		// Consume the applied credit (carry only the remainder).
		if creditApplied > 0 {
			if err := tx.Model(&models.MealSubscription{}).Where("id = ?", sub.ID).
				Update("credit_balance", remainingCredit).Error; err != nil {
				return err
			}
			sub.CreditBalance = remainingCredit
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

// CreditMealSubscription adds a skip/missed-day credit to the subscription's
// balance (applied to the next cycle). Race-safe increment.
func CreditMealSubscription(db *gorm.DB, subID uuid.UUID, amount float64) error {
	if amount <= 0 {
		return nil
	}
	return db.Model(&models.MealSubscription{}).Where("id = ?", subID).
		Update("credit_balance", gorm.Expr("credit_balance + ?", models.RoundAmount(amount))).Error
}

func mealInvoiceNumber() string {
	return fmt.Sprintf("MS-%d-%s", time.Now().Unix(), uuid.NewString()[:8])
}

// ActivateMealSubscriptionOnCharge handles a successful recurring charge from the
// Razorpay `subscription.charged` webhook for a meal subscription: it flips a
// trialing/past_due sub to active, rolls the billing period forward, and generates
// the cycle invoice (marked paid with the gateway payment id). Idempotent per
// (subscription, period): a duplicate webhook for the same cycle is a no-op.
// Returns the invoice (nil if it was a duplicate / no matching subscription).
func ActivateMealSubscriptionOnCharge(db *gorm.DB, gatewaySubID, paymentID string) (*models.MealSubscriptionInvoice, error) {
	var sub models.MealSubscription
	if err := db.Where("gateway_sub_id = ?", gatewaySubID).First(&sub).Error; err != nil {
		return nil, nil // not a meal subscription (or unknown) — caller tries other tables
	}
	// Idempotency keys on the gateway PAYMENT id — a redelivered webhook repeats it
	// (the billing period advances after each real charge, so it can't be the key).
	if paymentID != "" {
		var existing int64
		db.Model(&models.MealSubscriptionInvoice{}).Where("gateway_payment_id = ?", paymentID).Count(&existing)
		if existing > 0 {
			return nil, nil
		}
	}

	now := time.Now()
	periodStart := now
	if sub.CurrentPeriodEnd != nil && sub.CurrentPeriodEnd.After(now) {
		periodStart = *sub.CurrentPeriodEnd
	}
	periodEnd := AddMealCadence(periodStart, sub.Cadence)

	inv, err := GenerateMealCycleInvoice(db, &sub, periodStart, periodEnd)
	if err != nil {
		return nil, err
	}
	paidAt := now
	db.Model(&models.MealSubscriptionInvoice{}).Where("id = ?", inv.ID).
		Updates(map[string]any{"status": models.MealInvoiceStatusPaid, "gateway_payment_id": paymentID, "paid_at": &paidAt})
	db.Model(&models.MealSubscription{}).Where("id = ?", sub.ID).Updates(map[string]any{
		"status":               models.MealSubStatusActive,
		"current_period_start": &periodStart,
		"current_period_end":   &periodEnd,
	})
	return inv, nil
}

// HaltMealSubscriptionOnFailure handles a `subscription.halted`/`past_due` webhook
// for a meal subscription: stops order generation by moving it to past_due.
// Returns true if a meal subscription matched.
func HaltMealSubscriptionOnFailure(db *gorm.DB, gatewaySubID string) bool {
	res := db.Model(&models.MealSubscription{}).
		Where("gateway_sub_id = ? AND status IN ?", gatewaySubID, []string{models.MealSubStatusActive, models.MealSubStatusTrialing}).
		Update("status", models.MealSubStatusPastDue)
	return res.RowsAffected > 0
}
