package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// meal_subscription_orders_test.go — #282. Daily auto-order generation: the
// delivery-day decision, dish resolution, skip handling, idempotency, and the
// missed-day → credit + adherence accounting.

func TestIsMealDeliveryDay(t *testing.T) {
	// Days = Mon(1), Wed(3), Fri(5).
	sub := &models.MealSubscription{Days: []int64{1, 3, 5}}
	mon := nextWeekday(time.Monday)
	tue := nextWeekday(time.Tuesday)
	assert.True(t, IsMealDeliveryDay(sub, mon))
	assert.False(t, IsMealDeliveryDay(sub, tue))
}

func nextWeekday(wd time.Weekday) time.Time {
	// Normalize to UTC midnight: the sqlite test driver's date() comparison in
	// isMealDateSkipped normalizes a local-midnight (+offset) timestamp to the
	// previous UTC day, so a wall-clock-carrying time.Now() made the skip lookup
	// flaky by time-of-day. A UTC date is stable on every driver (Postgres prod
	// is unaffected — it stores proper timestamps).
	n := time.Now().UTC()
	d := time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
	for d.Weekday() != wd {
		d = d.AddDate(0, 0, 1)
	}
	return d
}

func TestCutoffPassed(t *testing.T) {
	ist, _ := time.LoadLocation("Asia/Kolkata")
	at := func(h, m int) time.Time { return time.Date(2026, 1, 1, h, m, 0, 0, ist) }
	assert.True(t, cutoffPassed(at(21, 30), "21:00", ist))  // after cutoff
	assert.False(t, cutoffPassed(at(20, 0), "21:00", ist))  // before cutoff
	assert.True(t, cutoffPassed(at(8, 0), "bad", ist))      // malformed → fail-open
}

func setupMealOrderDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	// Hand-rolled (AutoMigrate of these models chokes on sqlite's gen_random_uuid()).
	// The Order table is intentionally omitted: the order-persistence path reuses the
	// proven meal_plan generateDayOrder pattern; these tests cover the NEW decision /
	// skip / idempotency / missed-credit / adherence logic.
	stmts := []string{
		`CREATE TABLE weekly_menu_items (id TEXT PRIMARY KEY, chef_id TEXT, day_of_week INTEGER, slot TEXT, variant TEXT,
			name TEXT, description TEXT, price REAL, image_url TEXT, dietary_tags TEXT, allergens TEXT, menu_item_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_subscription_fulfillments (id TEXT PRIMARY KEY, meal_subscription_id TEXT, customer_id TEXT,
			chef_id TEXT, date DATETIME, slot TEXT, dish_name TEXT, price REAL, status TEXT DEFAULT 'scheduled',
			order_id TEXT, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE meal_subscription_skips (id TEXT PRIMARY KEY, meal_subscription_id TEXT, date DATETIME, created_at DATETIME)`,
		`CREATE TABLE meal_subscriptions (id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, cycle_amount REAL,
			credit_balance REAL DEFAULT 0, currency TEXT, status TEXT, updated_at DATETIME, deleted_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func seedDish(t *testing.T, db *gorm.DB, chefID uuid.UUID, weekday int, slot models.MealSlot, name string, price float64) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO weekly_menu_items (id, chef_id, day_of_week, slot, variant, name, price)
		VALUES (?, ?, ?, ?, 'veg', ?, ?)`, uuid.New().String(), chefID.String(), weekday, string(slot), name, price).Error)
}

func TestGenerateMealSubscriptionDay(t *testing.T) {
	t.Run("skipped date records a skipped fulfillment, no order", func(t *testing.T) {
		db := setupMealOrderDB(t)
		chef, cust := uuid.New(), uuid.New()
		date := nextWeekday(time.Tuesday)
		wd := int(date.Weekday())
		seedDish(t, db, chef, wd, models.MealSlotLunch, "Dal Rice", 120)
		sub := &models.MealSubscription{
			ID: uuid.New(), CustomerID: cust, ChefID: chef, Currency: "INR",
			Status: models.MealSubStatusActive, Days: []int64{int64(wd)}, Slots: []string{"lunch"}, Variant: models.MealVariantVeg,
		}
		require.NoError(t, db.Exec(`INSERT INTO meal_subscription_skips (id, meal_subscription_id, date) VALUES (?, ?, ?)`,
			uuid.New().String(), sub.ID.String(), date).Error)

		placed, err := GenerateMealSubscriptionDay(db, sub, date, models.Address{City: "X"})
		require.NoError(t, err)
		assert.Equal(t, 0, placed)
		var skipped int64
		db.Model(&models.MealSubscriptionFulfillment{}).Where("status = ?", models.MealFulfillSkipped).Count(&skipped)
		assert.Equal(t, int64(1), skipped)
	})

	t.Run("idempotent — already-fulfilled slots place nothing", func(t *testing.T) {
		db := setupMealOrderDB(t)
		chef, cust := uuid.New(), uuid.New()
		date := nextWeekday(time.Monday)
		wd := int(date.Weekday())
		sub := &models.MealSubscription{
			ID: uuid.New(), CustomerID: cust, ChefID: chef, Currency: "INR",
			Status: models.MealSubStatusActive, Days: []int64{int64(wd)}, Slots: []string{"lunch", "dinner"}, Variant: models.MealVariantVeg,
		}
		// Pre-seed both slots as already placed → re-run is a no-op.
		for _, slot := range []string{"lunch", "dinner"} {
			require.NoError(t, db.Exec(`INSERT INTO meal_subscription_fulfillments (id, meal_subscription_id, date, slot, status)
				VALUES (?, ?, ?, ?, 'placed')`, uuid.New().String(), sub.ID.String(), date, slot).Error)
		}
		placed, err := GenerateMealSubscriptionDay(db, sub, date, models.Address{City: "X"})
		require.NoError(t, err)
		assert.Equal(t, 0, placed)
	})

	t.Run("non-delivery day / non-active → nothing", func(t *testing.T) {
		db := setupMealOrderDB(t)
		mon := int64(nextWeekday(time.Monday).Weekday())
		active := &models.MealSubscription{ID: uuid.New(), Status: models.MealSubStatusActive, Days: []int64{mon}, Slots: []string{"lunch"}}
		// A day NOT in Days.
		placed, err := GenerateMealSubscriptionDay(db, active, nextWeekday(time.Sunday), models.Address{})
		require.NoError(t, err)
		assert.Equal(t, 0, placed)

		paused := &models.MealSubscription{ID: uuid.New(), Status: models.MealSubStatusPaused, Days: []int64{mon}, Slots: []string{"lunch"}}
		placed, err = GenerateMealSubscriptionDay(db, paused, nextWeekday(time.Monday), models.Address{})
		require.NoError(t, err)
		assert.Equal(t, 0, placed)
	})
}

func TestMarkMealFulfillmentMissedAndAdherence(t *testing.T) {
	db := setupMealOrderDB(t)
	subID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_subscriptions (id, cycle_amount, credit_balance, status)
		VALUES (?, 1000, 0, 'active')`, subID.String()).Error)
	fulfillID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_subscription_fulfillments (id, meal_subscription_id, date, slot, price, status)
		VALUES (?, ?, ?, 'lunch', 120, 'placed')`, fulfillID.String(), subID.String(), time.Now()).Error)

	require.NoError(t, MarkMealFulfillmentMissed(db, fulfillID))
	require.NoError(t, MarkMealFulfillmentMissed(db, fulfillID)) // idempotent

	var credit float64
	db.Raw(`SELECT credit_balance FROM meal_subscriptions WHERE id = ?`, subID.String()).Scan(&credit)
	assert.Equal(t, 120.0, credit, "missed day credited exactly once")

	ad := GetMealAdherence(db, subID)
	assert.Equal(t, int64(1), ad.Scheduled)
	assert.Equal(t, int64(1), ad.Missed)
}
