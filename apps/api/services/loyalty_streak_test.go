package services

// loyalty_streak_test.go — meal-subscription adherence streaks (#40 / #291).
// Pins: a delivered day advances the streak; two slots on the same calendar day
// count once; the streak bonus lands exactly when the run hits the configured
// threshold (and never twice for the same milestone); a missed day resets the
// run; and the fulfillment-delivered transition (the gap this fills) both flips
// the fulfillment row and advances the streak.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// setupLoyaltyStreakDB extends the loyalty test DB with the meal-subscription
// fulfillment table so the delivered-transition path is exercisable.
func setupLoyaltyStreakDB(t *testing.T) *gorm.DB {
	db := setupLoyaltyDB(t)
	require.NoError(t, db.Exec(`CREATE TABLE meal_subscription_fulfillments (
		id text PRIMARY KEY, meal_subscription_id text, customer_id text, chef_id text,
		date datetime, slot text, dish_name text, price real, status text,
		order_id text, created_at datetime, updated_at datetime, deleted_at datetime
	)`).Error)
	return db
}

func day(d int) time.Time { return time.Date(2026, 6, d, 12, 0, 0, 0, time.UTC) }

func TestAdvanceStreak_IncrementsAndDedupesSameDay(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()

	s, _, err := AdvanceLoyaltyStreak(db, uid, day(1))
	require.NoError(t, err)
	require.Equal(t, 1, s)

	// Second slot, same calendar day → no double count.
	s, _, err = AdvanceLoyaltyStreak(db, uid, day(1).Add(6*time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, s)

	// Next day → 2.
	s, _, err = AdvanceLoyaltyStreak(db, uid, day(2))
	require.NoError(t, err)
	require.Equal(t, 2, s)

	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 2, acct.CurrentStreak)
	require.Equal(t, 2, acct.LongestStreak)
}

func TestStreakBonus_AwardedExactlyAtThreshold(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	setLoyaltySetting(t, db, "loyalty.streak_threshold", "3")
	setLoyaltySetting(t, db, "loyalty.streak_bonus", "30")

	var lastBonus float64
	for d := 1; d <= 3; d++ {
		_, bonus, err := AdvanceLoyaltyStreak(db, uid, day(d))
		require.NoError(t, err)
		lastBonus = bonus
	}
	require.Equal(t, 30.0, lastBonus) // bonus lands on the 3rd day

	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 3, acct.CurrentStreak)
	require.Equal(t, 30.0, acct.Balance) // only the streak bonus — no order points here

	// A 4th day does NOT re-award (not a multiple of 3 yet).
	_, bonus, err := AdvanceLoyaltyStreak(db, uid, day(4))
	require.NoError(t, err)
	require.Equal(t, 0.0, bonus)

	// Exactly one streak-bonus ledger row so far.
	var n int64
	db.Model(&models.LoyaltyTransaction{}).Where("user_id = ? AND source = ?", uid, models.LoyaltySourceStreak).Count(&n)
	require.Equal(t, int64(1), n)
}

func TestAdvanceStreak_GapBreaksTheRun(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()

	s, _, err := AdvanceLoyaltyStreak(db, uid, day(1))
	require.NoError(t, err)
	require.Equal(t, 1, s)
	s, _, err = AdvanceLoyaltyStreak(db, uid, day(2))
	require.NoError(t, err)
	require.Equal(t, 2, s)

	// A 3-day gap (day 2 → day 5, a missed day in between) breaks the run: the
	// next delivered day starts a fresh streak at 1, not 3.
	s, _, err = AdvanceLoyaltyStreak(db, uid, day(5))
	require.NoError(t, err)
	require.Equal(t, 1, s)

	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 1, acct.CurrentStreak)
	require.Equal(t, 2, acct.LongestStreak) // best run is preserved
}

func TestResetLoyaltyStreak(t *testing.T) {
	db := setupLoyaltyDB(t)
	uid := uuid.New()
	_, _, _ = AdvanceLoyaltyStreak(db, uid, day(1))
	_, _, _ = AdvanceLoyaltyStreak(db, uid, day(2))

	require.NoError(t, ResetLoyaltyStreak(db, uid))
	acct, _ := LoyaltyBalance(db, uid)
	require.Equal(t, 0, acct.CurrentStreak)
	require.Nil(t, acct.LastStreakDay)
	require.Equal(t, 2, acct.LongestStreak) // longest is preserved

	// After a reset, the next delivered day starts a fresh run at 1.
	s, _, err := AdvanceLoyaltyStreak(db, uid, day(4))
	require.NoError(t, err)
	require.Equal(t, 1, s)
}

func TestMarkMealFulfillmentDelivered_TransitionsAndAdvancesStreak(t *testing.T) {
	db := setupLoyaltyStreakDB(t)
	customerID := uuid.New()
	orderID := uuid.New()
	fulfillmentID := uuid.New()

	require.NoError(t, db.Exec(
		`INSERT INTO meal_subscription_fulfillments (id, meal_subscription_id, customer_id, chef_id, date, slot, status, order_id)
		 VALUES (?, ?, ?, ?, ?, 'lunch', 'placed', ?)`,
		fulfillmentID.String(), uuid.New().String(), customerID.String(), uuid.New().String(), day(1), orderID.String(),
	).Error)

	require.NoError(t, MarkMealFulfillmentDelivered(db, orderID))

	var status string
	db.Raw(`SELECT status FROM meal_subscription_fulfillments WHERE id = ?`, fulfillmentID.String()).Scan(&status)
	require.Equal(t, models.MealFulfillDelivered, status)

	acct, _ := LoyaltyBalance(db, customerID)
	require.Equal(t, 1, acct.CurrentStreak)

	// Idempotent: redelivered event does not re-advance the streak.
	require.NoError(t, MarkMealFulfillmentDelivered(db, orderID))
	acct, _ = LoyaltyBalance(db, customerID)
	require.Equal(t, 1, acct.CurrentStreak)
}

func TestMarkMealFulfillmentDelivered_NonSubscriptionOrderIsNoop(t *testing.T) {
	db := setupLoyaltyStreakDB(t)
	// An order with no matching fulfillment row (a normal one-off order).
	require.NoError(t, MarkMealFulfillmentDelivered(db, uuid.New()))
}
