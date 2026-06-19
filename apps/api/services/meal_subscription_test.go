package services

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/homechef/api/models"
)

// meal_subscription_test.go — #280. The pure price + selection-validation +
// lifecycle logic for the customer tiffin subscription.

func TestComputeMealCycleAmount(t *testing.T) {
	// 2 slots (lunch+dinner) × 5 days × 1 week × ₹120 + ₹40 delivery = 1200 + 40.
	assert.Equal(t, 1240.0, ComputeMealCycleAmount(120, 2, 5, models.MealCadenceWeekly, 40))
	// Monthly = 4 weeks: 1 slot × 7 days × 4 × ₹100 + 0 = 2800.
	assert.Equal(t, 2800.0, ComputeMealCycleAmount(100, 1, 7, models.MealCadenceMonthly, 0))
	// Degenerate selections → 0.
	assert.Equal(t, 0.0, ComputeMealCycleAmount(120, 0, 5, models.MealCadenceWeekly, 40))
	assert.Equal(t, 0.0, ComputeMealCycleAmount(120, 2, 0, models.MealCadenceWeekly, 40))
}

func TestMealCycleWeeks(t *testing.T) {
	assert.Equal(t, 1, MealCycleWeeks(models.MealCadenceWeekly))
	assert.Equal(t, 4, MealCycleWeeks(models.MealCadenceMonthly))
}

func TestValidateMealSelection(t *testing.T) {
	cfg := &models.ChefSubscriptionConfig{
		Enabled:  true,
		Slots:    []string{"lunch", "dinner"},
		Cadences: []string{"weekly", "monthly"},
	}

	t.Run("valid selection", func(t *testing.T) {
		assert.NoError(t, ValidateMealSelection(cfg, []string{"lunch"}, []int64{1, 2, 3, 4, 5}, "weekly"))
	})
	t.Run("chef not configured / disabled", func(t *testing.T) {
		assert.ErrorIs(t, ValidateMealSelection(nil, []string{"lunch"}, []int64{1}, "weekly"), ErrMealSubNotConfigured)
		off := *cfg
		off.Enabled = false
		assert.ErrorIs(t, ValidateMealSelection(&off, []string{"lunch"}, []int64{1}, "weekly"), ErrMealSubNotConfigured)
	})
	t.Run("no days", func(t *testing.T) {
		assert.ErrorIs(t, ValidateMealSelection(cfg, []string{"lunch"}, nil, "weekly"), ErrMealSubNoDays)
	})
	t.Run("bad day", func(t *testing.T) {
		assert.ErrorIs(t, ValidateMealSelection(cfg, []string{"lunch"}, []int64{7}, "weekly"), ErrMealSubBadDay)
	})
	t.Run("slot not offered", func(t *testing.T) {
		lunchOnly := *cfg
		lunchOnly.Slots = []string{"lunch"}
		assert.ErrorIs(t, ValidateMealSelection(&lunchOnly, []string{"dinner"}, []int64{1}, "weekly"), ErrMealSubBadSlot)
	})
	t.Run("cadence not offered", func(t *testing.T) {
		weeklyOnly := *cfg
		weeklyOnly.Cadences = []string{"weekly"}
		assert.ErrorIs(t, ValidateMealSelection(&weeklyOnly, []string{"lunch"}, []int64{1}, "monthly"), ErrMealSubBadCadence)
	})
}

func TestMealSubLifecycleTransitions(t *testing.T) {
	assert.True(t, CanPauseMealSub(models.MealSubStatusActive))
	assert.False(t, CanPauseMealSub(models.MealSubStatusPaused))
	assert.True(t, CanResumeMealSub(models.MealSubStatusPaused))
	assert.False(t, CanResumeMealSub(models.MealSubStatusActive))

	assert.True(t, CanCancelMealSub(models.MealSubStatusActive))
	assert.True(t, CanCancelMealSub(models.MealSubStatusPaused))
	assert.True(t, CanCancelMealSub(models.MealSubStatusTrialing))
	assert.False(t, CanCancelMealSub(models.MealSubStatusCancelled))

	assert.True(t, MealSubGeneratesOrders(models.MealSubStatusActive))
	assert.False(t, MealSubGeneratesOrders(models.MealSubStatusPaused))
	assert.False(t, MealSubGeneratesOrders(models.MealSubStatusPastDue))
}
