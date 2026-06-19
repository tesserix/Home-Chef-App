package services

// meal_subscription.go — the customer tiffin meal-subscription engine (#2/#3).
// Pure price + selection-validation + lifecycle logic kept here (DB-free, unit
// tested); the DB-touching helpers wrap it. The Razorpay recurring rail + the
// daily auto-order cron land in later phases (#281/#282) on top of this.

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

var (
	ErrMealSubNotConfigured = errors.New("this chef does not offer meal subscriptions")
	ErrMealSubNoMenu        = errors.New("the chef must publish a weekly menu first")
	ErrMealSubBadSlot       = errors.New("selected meal slot is not offered by this chef")
	ErrMealSubBadCadence    = errors.New("selected cadence is not offered by this chef")
	ErrMealSubNoDays        = errors.New("select at least one delivery day")
	ErrMealSubBadDay        = errors.New("invalid delivery day")
)

// MealCycleWeeks is the number of weeks billed per cycle for a cadence.
func MealCycleWeeks(cadence string) int {
	if cadence == models.MealCadenceMonthly {
		return 4
	}
	return 1
}

// ComputeMealCycleAmount is the per-cycle charge: perMeal × slots × days × weeks
// + flat delivery. The single source of truth for the live price preview and the
// frozen subscription amount. Returns 0 for a degenerate selection.
func ComputeMealCycleAmount(perMealPrice float64, numSlots, numDays int, cadence string, deliveryFee float64) float64 {
	if perMealPrice < 0 || numSlots <= 0 || numDays <= 0 {
		return 0
	}
	meals := numSlots * numDays * MealCycleWeeks(cadence)
	amount := perMealPrice*float64(meals) + deliveryFee
	if amount < 0 {
		amount = 0
	}
	return models.RoundAmount(amount)
}

// ValidateMealSelection checks a customer's selection against a chef's offer.
// Returns nil when the selection is valid.
func ValidateMealSelection(cfg *models.ChefSubscriptionConfig, slots []string, days []int64, cadence string) error {
	if cfg == nil || !cfg.Enabled {
		return ErrMealSubNotConfigured
	}
	if len(days) == 0 {
		return ErrMealSubNoDays
	}
	for _, d := range days {
		if d < 0 || d > 6 {
			return ErrMealSubBadDay
		}
	}
	if len(slots) == 0 {
		return ErrMealSubBadSlot
	}
	for _, s := range slots {
		if !sliceHas(cfg.Slots, s) {
			return ErrMealSubBadSlot
		}
	}
	if !sliceHas(cfg.Cadences, cadence) {
		return ErrMealSubBadCadence
	}
	return nil
}

func sliceHas(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

// Lifecycle transitions — what's allowed from a given status.
func CanPauseMealSub(status string) bool  { return status == models.MealSubStatusActive }
func CanResumeMealSub(status string) bool { return status == models.MealSubStatusPaused }
func CanCancelMealSub(status string) bool {
	switch status {
	case models.MealSubStatusActive, models.MealSubStatusPaused,
		models.MealSubStatusTrialing, models.MealSubStatusPastDue:
		return true
	}
	return false
}

// MealSubGeneratesOrders reports whether the daily auto-order cron should generate
// for a subscription in this status (active only; paused/past_due/cancelled don't).
func MealSubGeneratesOrders(status string) bool { return status == models.MealSubStatusActive }

// GetChefSubscriptionConfig loads a chef's meal-subscription offer config, or nil.
func GetChefSubscriptionConfig(db *gorm.DB, chefID uuid.UUID) *models.ChefSubscriptionConfig {
	var cfg models.ChefSubscriptionConfig
	if err := db.Where("chef_id = ?", chefID).First(&cfg).Error; err != nil {
		return nil
	}
	return &cfg
}

// ChefHasPublishedWeeklyMenu guards subscription enablement on a published menu (#1).
func ChefHasPublishedWeeklyMenu(db *gorm.DB, chefID uuid.UUID) bool {
	var n int64
	db.Model(&models.WeeklyMenu{}).Where("chef_id = ? AND is_published = ?", chefID, true).Count(&n)
	return n > 0
}

// AddMealCadence advances a time by one billing cycle (1 week / 1 month).
func AddMealCadence(t time.Time, cadence string) time.Time {
	if cadence == models.MealCadenceMonthly {
		return t.AddDate(0, 1, 0)
	}
	return t.AddDate(0, 0, 7)
}
