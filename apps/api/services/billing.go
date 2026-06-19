package services

import (
	"crypto/rand"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// PlanConfig holds parsed subscription settings for a country/role
type PlanConfig struct {
	Currency             string
	MonthlyPrice         float64
	QuarterlyPrice       float64
	YearlyPrice          float64
	TrialDays            int
	MinEarningsThreshold float64
	PaymentGateway       string
	GracePeriodDays      int

	// Premium tier (#44) — fully admin-configurable via PlatformSettings keys
	// `subscription.{CC}.{role}.premium_{monthly|quarterly|yearly}_price` and
	// `…premium_commission_rate`. Defaults below keep the feature working before
	// an admin sets anything.
	PremiumMonthlyPrice   float64
	PremiumQuarterlyPrice float64
	PremiumYearlyPrice    float64
	// PremiumCommissionRate is the platform take-rate for premium chefs (e.g.
	// 0.12 vs the standard 0.15). Applied in services/earnings.go.
	PremiumCommissionRate float64
}

// PriceFor returns the configured price for a tier + billing interval (#44).
func (c *PlanConfig) PriceFor(tier models.SubscriptionTier, interval models.BillingInterval) float64 {
	premium := tier == models.TierPremium
	switch interval {
	case models.BillingQuarterly:
		if premium {
			return c.PremiumQuarterlyPrice
		}
		return c.QuarterlyPrice
	case models.BillingYearly:
		if premium {
			return c.PremiumYearlyPrice
		}
		return c.YearlyPrice
	default: // monthly
		if premium {
			return c.PremiumMonthlyPrice
		}
		return c.MonthlyPrice
	}
}

// GetPlanSettings queries PlatformSettings for subscription configuration
func GetPlanSettings(countryCode string, subType models.SubscriberType) (*PlanConfig, error) {
	cc := strings.ToUpper(countryCode)
	role := string(subType)

	cfg := &PlanConfig{
		// Sensible defaults
		Currency:             "INR",
		MonthlyPrice:         499,
		QuarterlyPrice:       1299,
		YearlyPrice:          4499,
		TrialDays:            30,
		MinEarningsThreshold: 5000,
		PaymentGateway:       "razorpay",
		GracePeriodDays:      7,
		// Premium tier defaults (#44) — admin-overridable per the keys below.
		PremiumMonthlyPrice:   999,
		PremiumQuarterlyPrice: 2599,
		PremiumYearlyPrice:    8999,
		PremiumCommissionRate: 0.12,
	}

	if subType == models.SubscriberDriver {
		cfg.TrialDays = 15
		cfg.MonthlyPrice = 299
		cfg.QuarterlyPrice = 799
		cfg.YearlyPrice = 2999
		cfg.MinEarningsThreshold = 3000
		// Drivers have no premium tier today; keep premium == standard so a
		// stray lookup never under/over-charges.
		cfg.PremiumMonthlyPrice = cfg.MonthlyPrice
		cfg.PremiumQuarterlyPrice = cfg.QuarterlyPrice
		cfg.PremiumYearlyPrice = cfg.YearlyPrice
		cfg.PremiumCommissionRate = 0
	}

	// Try to load from PlatformSettings
	var settings []models.PlatformSettings
	prefixes := []string{
		fmt.Sprintf("subscription.%s.%s.", cc, role),
		fmt.Sprintf("subscription.%s.", cc),
	}

	for _, prefix := range prefixes {
		var found []models.PlatformSettings
		database.DB.Where("key LIKE ?", prefix+"%").Find(&found)
		settings = append(settings, found...)
	}

	// Also load global grace period
	var graceSetting models.PlatformSettings
	if err := database.DB.Where("key = ?", "subscription.grace_period_days").First(&graceSetting).Error; err == nil {
		if v, err := strconv.Atoi(graceSetting.Value); err == nil {
			cfg.GracePeriodDays = v
		}
	}

	// Parse settings into config
	for _, s := range settings {
		key := s.Key
		val := s.Value

		if strings.HasSuffix(key, ".currency") {
			cfg.Currency = val
		} else if strings.HasSuffix(key, ".premium_monthly_price") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.PremiumMonthlyPrice = v
			}
		} else if strings.HasSuffix(key, ".premium_quarterly_price") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.PremiumQuarterlyPrice = v
			}
		} else if strings.HasSuffix(key, ".premium_yearly_price") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.PremiumYearlyPrice = v
			}
		} else if strings.HasSuffix(key, ".premium_commission_rate") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.PremiumCommissionRate = v
			}
		} else if strings.HasSuffix(key, ".monthly_price") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.MonthlyPrice = v
			}
		} else if strings.HasSuffix(key, ".quarterly_price") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.QuarterlyPrice = v
			}
		} else if strings.HasSuffix(key, ".yearly_price") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.YearlyPrice = v
			}
		} else if strings.HasSuffix(key, ".trial_days") {
			if v, err := strconv.Atoi(val); err == nil {
				cfg.TrialDays = v
			}
		} else if strings.HasSuffix(key, ".min_earnings_threshold") {
			if v, err := strconv.ParseFloat(val, 64); err == nil {
				cfg.MinEarningsThreshold = v
			}
		} else if strings.HasSuffix(key, ".payment_gateway") {
			cfg.PaymentGateway = val
		}
	}

	return cfg, nil
}

// CreateTrialSubscription creates a new trial subscription for a user
func CreateTrialSubscription(userID uuid.UUID, subType models.SubscriberType, countryCode string) (*models.Subscription, error) {
	planCfg, err := GetPlanSettings(countryCode, subType)
	if err != nil {
		return nil, fmt.Errorf("failed to get plan settings: %w", err)
	}

	// Check if subscription already exists
	var existing models.Subscription
	if err := database.DB.Where("user_id = ? AND subscriber_type = ?", userID, subType).First(&existing).Error; err == nil {
		return &existing, nil
	}

	now := time.Now().UTC()
	trialEnd := now.AddDate(0, 0, planCfg.TrialDays)

	sub := models.Subscription{
		UserID:          userID,
		SubscriberType:  subType,
		CountryCode:     strings.ToUpper(countryCode),
		Currency:        planCfg.Currency,
		BillingInterval: models.BillingMonthly,
		Status:          models.SubStatusTrial,
		PlanAmount:      planCfg.MonthlyPrice,
		TrialStartsAt:   now,
		TrialEndsAt:     trialEnd,
		PaymentGateway:  planCfg.PaymentGateway,
	}

	if err := database.DB.Create(&sub).Error; err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Durable event publication via the transactional outbox.
	if err := EnqueueEvent(database.DB, SubjectSubscriptionCreated, "subscription.created", userID, map[string]interface{}{
		"subscription_id": sub.ID.String(),
		"subscriber_type": string(subType),
		"country_code":    sub.CountryCode,
		"trial_ends_at":   sub.TrialEndsAt.Format(time.RFC3339),
	}); err != nil {
		log.Printf("failed to enqueue subscription.created event: %v", err)
	}

	return &sub, nil
}

// RecordEarning creates an earnings ledger entry for the current billing cycle
func RecordEarning(userID, subscriptionID uuid.UUID, source models.EarningSource, amount float64, currency string, orderID, deliveryID *uuid.UUID) error {
	var sub models.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	// Determine cycle boundaries
	cycleStart, cycleEnd := getCycleBounds(&sub)

	entry := models.EarningsLedger{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		SubscriberType: sub.SubscriberType,
		CycleStart:     cycleStart,
		CycleEnd:       cycleEnd,
		Source:         source,
		OrderID:        orderID,
		DeliveryID:     deliveryID,
		Amount:         amount,
		Currency:       currency,
	}

	if err := database.DB.Create(&entry).Error; err != nil {
		return fmt.Errorf("failed to record earning: %w", err)
	}

	return nil
}

// CheckEarningsThreshold checks if earnings have crossed the threshold and generates invoice if needed
func CheckEarningsThreshold(subscriptionID uuid.UUID) (*models.SubscriptionInvoice, error) {
	var sub models.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return nil, fmt.Errorf("subscription not found: %w", err)
	}

	// Only active subscriptions get invoiced (trial users don't pay yet)
	if sub.Status != models.SubStatusActive {
		return nil, nil
	}

	// Get cycle bounds
	cycleStart, cycleEnd := getCycleBounds(&sub)

	// Check if there's already an invoice for this cycle
	var existingInvoice models.SubscriptionInvoice
	if err := database.DB.Where("subscription_id = ? AND period_start = ? AND period_end = ?",
		sub.ID, cycleStart, cycleEnd).First(&existingInvoice).Error; err == nil {
		return nil, nil // Invoice already exists
	}

	// Sum earnings for current cycle
	var cycleEarnings float64
	database.DB.Model(&models.EarningsLedger{}).
		Where("subscription_id = ? AND cycle_start = ? AND cycle_end = ?",
			sub.ID, cycleStart, cycleEnd).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&cycleEarnings)

	// Get threshold
	planCfg, err := GetPlanSettings(sub.CountryCode, sub.SubscriberType)
	if err != nil {
		return nil, err
	}

	if cycleEarnings >= planCfg.MinEarningsThreshold {
		invoice, err := GenerateInvoice(&sub, cycleEarnings)
		if err != nil {
			return nil, err
		}

		// Set BillingStartsAt if this is the first time
		if sub.BillingStartsAt == nil {
			now := time.Now().UTC()
			database.DB.Model(&sub).Update("billing_starts_at", &now)
		}

		// Durable event publication via the transactional outbox.
		if err := EnqueueEvent(database.DB, SubjectEarningsThresholdMet, "subscription.earnings.threshold_met", sub.UserID, map[string]interface{}{
			"subscription_id": sub.ID.String(),
			"cycle_earnings":  cycleEarnings,
			"threshold":       planCfg.MinEarningsThreshold,
			"invoice_id":      invoice.ID.String(),
		}); err != nil {
			log.Printf("failed to enqueue subscription.earnings.threshold_met event: %v", err)
		}

		return invoice, nil
	}

	return nil, nil
}

// GenerateInvoice creates a new invoice for a subscription. A one-time signup /
// trial promo (#269) is applied to the FIRST invoice: the discount + atomic
// redemption claim + invoice creation all commit together so the promo can never
// be half-applied.
func GenerateInvoice(sub *models.Subscription, cycleEarnings float64) (*models.SubscriptionInvoice, error) {
	cycleStart, cycleEnd := getCycleBounds(sub)
	taxRate := getTaxRate(sub.CountryCode)

	var invoice models.SubscriptionInvoice
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Apply the stored promo (no-op + 0 when none / ineligible / exhausted).
		discount := ApplySubscriptionPromoToInvoice(tx, sub, sub.PlanAmount)
		netAmount := models.RoundAmount(sub.PlanAmount - discount)
		taxAmount := models.RoundAmount(netAmount * taxRate)
		totalAmount := models.RoundAmount(netAmount + taxAmount)

		invoice = models.SubscriptionInvoice{
			SubscriptionID:       sub.ID,
			InvoiceNumber:        generateInvoiceNumber(),
			Status:               models.InvoicePending,
			Amount:               netAmount,
			Currency:             sub.Currency,
			TaxAmount:            taxAmount,
			TotalAmount:          totalAmount,
			PeriodStart:          cycleStart,
			PeriodEnd:            cycleEnd,
			EarningsAtGeneration: cycleEarnings,
			PaymentGateway:       sub.PaymentGateway,
			AttemptCount:         0,
			MaxAttempts:          3,
		}
		return tx.Create(&invoice).Error
	}); err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	// Durable event publication via the transactional outbox.
	if err := EnqueueEvent(database.DB, SubjectSubscriptionInvoiceCreated, "subscription.invoice.created", sub.UserID, map[string]interface{}{
		"subscription_id": sub.ID.String(),
		"invoice_id":      invoice.ID.String(),
		"invoice_number":  invoice.InvoiceNumber,
		"total_amount":    invoice.TotalAmount,
		"currency":        invoice.Currency,
	}); err != nil {
		log.Printf("failed to enqueue subscription.invoice.created event: %v", err)
	}

	return &invoice, nil
}

// CalculateProratedRefund calculates the refund for the remaining unused period
// as of now. Thin wrapper over calculateProratedRefundAt so the time-dependent
// math can be unit-tested deterministically with a fixed clock.
func CalculateProratedRefund(sub *models.Subscription) float64 {
	return calculateProratedRefundAt(sub, time.Now().UTC())
}

// calculateProratedRefundAt computes the prorated refund/credit for the unused
// remainder of the current billing period, evaluated at `now`:
//   - no period end, or period already over → 0
//   - yearly → remaining full months / 12 * plan amount
//   - monthly/quarterly → remaining days / total days * plan amount
func calculateProratedRefundAt(sub *models.Subscription, now time.Time) float64 {
	if sub.CurrentPeriodEnd == nil {
		return 0
	}

	if now.After(*sub.CurrentPeriodEnd) {
		return 0
	}

	if sub.BillingInterval == models.BillingYearly {
		// For yearly: remaining full months / 12 * yearly price
		remainingMonths := 0
		cursor := now
		for cursor.Before(*sub.CurrentPeriodEnd) {
			cursor = cursor.AddDate(0, 1, 0)
			if cursor.Before(*sub.CurrentPeriodEnd) || cursor.Equal(*sub.CurrentPeriodEnd) {
				remainingMonths++
			}
		}
		return models.RoundAmount(float64(remainingMonths) / 12.0 * sub.PlanAmount)
	}

	// For monthly/quarterly: remaining days / total days * plan amount
	totalDays := sub.CurrentPeriodEnd.Sub(*sub.CurrentPeriodStart).Hours() / 24
	remainingDays := sub.CurrentPeriodEnd.Sub(now).Hours() / 24

	if totalDays <= 0 {
		return 0
	}

	return models.RoundAmount(remainingDays / totalDays * sub.PlanAmount)
}

// TransitionTrialToActive transitions a trial subscription to active status
func TransitionTrialToActive(subscriptionID uuid.UUID) error {
	var sub models.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return fmt.Errorf("subscription not found: %w", err)
	}

	if sub.Status != models.SubStatusTrial {
		return fmt.Errorf("subscription is not in trial status")
	}

	now := time.Now().UTC()
	if now.Before(sub.TrialEndsAt) {
		return fmt.Errorf("trial has not ended yet")
	}

	periodEnd := addBillingInterval(now, sub.BillingInterval)

	updates := map[string]interface{}{
		"status":               models.SubStatusActive,
		"current_period_start": now,
		"current_period_end":   periodEnd,
	}

	if err := database.DB.Model(&sub).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to transition subscription: %w", err)
	}

	// Durable event publication via the transactional outbox.
	if err := EnqueueEvent(database.DB, SubjectSubscriptionActivated, "subscription.activated", sub.UserID, map[string]interface{}{
		"subscription_id":      sub.ID.String(),
		"current_period_start": now.Format(time.RFC3339),
		"current_period_end":   periodEnd.Format(time.RFC3339),
	}); err != nil {
		log.Printf("failed to enqueue subscription.activated event: %v", err)
	}

	return nil
}

// ---------- Helpers ----------

// getTaxRate returns the tax rate for a country code
func getTaxRate(countryCode string) float64 {
	rates := map[string]float64{
		"IN": 0.18, // India GST
		"AU": 0.10, // Australia GST
		"PK": 0.17, // Pakistan Sales Tax
		"BD": 0.15, // Bangladesh VAT
		"LK": 0.08, // Sri Lanka VAT
		"NP": 0.13, // Nepal VAT
	}
	if rate, ok := rates[strings.ToUpper(countryCode)]; ok {
		return rate
	}
	return 0.0
}

// generateInvoiceNumber generates a unique invoice number
func generateInvoiceNumber() string {
	now := time.Now().UTC()
	yearMonth := now.Format("200601")

	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based
		return fmt.Sprintf("FE3DR-INV-%s-%04d", yearMonth, now.UnixNano()%10000)
	}

	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 4)
	for i, v := range b {
		result[i] = chars[int(v)%len(chars)]
	}

	return fmt.Sprintf("FE3DR-INV-%s-%s", yearMonth, string(result))
}

// getCycleBounds returns the current billing cycle start and end dates
func getCycleBounds(sub *models.Subscription) (time.Time, time.Time) {
	if sub.CurrentPeriodStart != nil && sub.CurrentPeriodEnd != nil {
		return *sub.CurrentPeriodStart, *sub.CurrentPeriodEnd
	}

	// For trial subscriptions, use trial period as cycle
	return sub.TrialStartsAt, sub.TrialEndsAt
}

// addBillingInterval adds the billing interval to a time
func addBillingInterval(t time.Time, interval models.BillingInterval) time.Time {
	months := interval.IntervalMonths()
	return t.AddDate(0, months, 0)
}

// GetPlanAmount returns the plan amount for a given interval
func GetPlanAmount(cfg *PlanConfig, interval models.BillingInterval) float64 {
	switch interval {
	case models.BillingQuarterly:
		return cfg.QuarterlyPrice
	case models.BillingYearly:
		return cfg.YearlyPrice
	default:
		return cfg.MonthlyPrice
	}
}

// GetCycleEarnings returns total earnings for a subscription's current cycle
func GetCycleEarnings(subscriptionID uuid.UUID) (float64, float64, float64, float64) {
	var sub models.Subscription
	if err := database.DB.First(&sub, subscriptionID).Error; err != nil {
		return 0, 0, 0, 0
	}

	cycleStart, cycleEnd := getCycleBounds(&sub)

	var total, orderRev, deliveryFees, tips float64

	database.DB.Model(&models.EarningsLedger{}).
		Where("subscription_id = ? AND cycle_start = ? AND cycle_end = ?",
			subscriptionID, cycleStart, cycleEnd).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&total)

	database.DB.Model(&models.EarningsLedger{}).
		Where("subscription_id = ? AND cycle_start = ? AND cycle_end = ? AND source = ?",
			subscriptionID, cycleStart, cycleEnd, models.EarningOrderRevenue).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&orderRev)

	database.DB.Model(&models.EarningsLedger{}).
		Where("subscription_id = ? AND cycle_start = ? AND cycle_end = ? AND source = ?",
			subscriptionID, cycleStart, cycleEnd, models.EarningDeliveryFee).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&deliveryFees)

	database.DB.Model(&models.EarningsLedger{}).
		Where("subscription_id = ? AND cycle_start = ? AND cycle_end = ? AND source = ?",
			subscriptionID, cycleStart, cycleEnd, models.EarningTip).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&tips)

	return math.Round(total*100) / 100,
		math.Round(orderRev*100) / 100,
		math.Round(deliveryFees*100) / 100,
		math.Round(tips*100) / 100
}
