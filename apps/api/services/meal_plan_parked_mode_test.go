package services

// meal_plan_parked_mode_test.go — a kitchen used as a sandbox must not keep
// fulfilling its LIVE meal plans.
//
// The chef's app renders the test partition while they are in test mode, so a
// live day order generated meanwhile is one nobody can see, cook or deliver —
// and 24h later sweepStuckDays would auto-refund the customer for a day that
// was never offered. Both halves are parked on the SAME rule: a plan is
// fulfilled only while its partition is the one the chef is currently working
// in. That makes the resume automatic on the way back to live.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupParkedPlanDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	createTablesFor(t, db,
		&models.MealPlan{}, &models.MealPlanDay{}, &models.ChefProfile{},
		&models.Order{}, &models.OrderItem{}, &models.Address{},
		&models.ChefSchedule{}, &models.OutboxEvent{})
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// seedParkedPlan writes a live plan with one due, addressed day for a chef in
// the given mode, and returns the day id.
func seedParkedPlan(t *testing.T, db *gorm.DB, chefMode string) (chefID, dayID uuid.UUID) {
	t.Helper()
	chefID, customerID := uuid.New(), uuid.New()
	require.NoError(t, db.Create(&models.ChefProfile{
		ID: chefID, UserID: uuid.New(), BusinessName: "Saffron Home Kitchen",
		Mode: chefMode,
	}).Error)
	require.NoError(t, db.Create(&models.Address{
		ID: uuid.New(), UserID: customerID, IsDefault: true,
		Line1: "1 Test Street", City: "Bengaluru", State: "KA", PostalCode: "560001",
	}).Error)

	planID := uuid.New()
	require.NoError(t, db.Create(&models.MealPlan{
		ID: planID, MealPlanNumber: "MP-PARK", CustomerID: customerID, ChefID: chefID,
		Status: models.MealPlanActive, Total: 240,
	}).Error)

	dayID = uuid.New()
	require.NoError(t, db.Create(&models.MealPlanDay{
		ID: dayID, MealPlanID: planID, Status: models.MealPlanDayConfirmed,
		Price: 120, Date: time.Now().AddDate(0, 0, -2),
	}).Error)
	return chefID, dayID
}

// seedLiveChefRow gives an older fixture's plan the chef row it always implies.
// The fulfilment gate compares a plan's mode against its chef's, so a plan whose
// chef cannot be read is parked — right in production, where an orphaned plan
// must not auto-refund, but not what those fixtures set out to exercise.
func seedLiveChefRow(t *testing.T, db *gorm.DB, chefID string) {
	t.Helper()
	require.NoError(t, db.Exec(
		`CREATE TABLE IF NOT EXISTS chef_profiles (id TEXT PRIMARY KEY, mode TEXT DEFAULT 'live')`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, mode) VALUES (?, 'live')`, chefID).Error)
}

func TestMealPlanFulfillment_TestModeKitchenDoesNotGenerateLiveDayOrders(t *testing.T) {
	db := setupParkedPlanDB(t)
	_, dayID := seedParkedPlan(t, db, models.ChefModeTest)

	generateDueDayOrders()

	var day models.MealPlanDay
	require.NoError(t, db.First(&day, "id = ?", dayID).Error)
	require.Nil(t, day.OrderID, "a sandboxed kitchen must not generate live day orders")

	var orders int64
	require.NoError(t, db.Model(&models.Order{}).Count(&orders).Error)
	require.Zero(t, orders)
}

// The control: the same fixture on a live kitchen DOES generate, so the test
// above is proving the mode gate rather than a broken fixture.
func TestMealPlanFulfillment_LiveKitchenStillGeneratesDayOrders(t *testing.T) {
	db := setupParkedPlanDB(t)
	_, dayID := seedParkedPlan(t, db, models.ChefModeLive)

	generateDueDayOrders()

	var day models.MealPlanDay
	require.NoError(t, db.First(&day, "id = ?", dayID).Error)
	require.NotNil(t, day.OrderID, "a live kitchen must keep fulfilling its plans")
}

// The stranding half. A parked day generates no order, so without the same gate
// the stuck-day sweep would read it as address-less and auto-refund a real
// customer 24h into the sandbox session.
func TestMealPlanFulfillment_TestModeKitchenLiveDaysAreNotAutoRefunded(t *testing.T) {
	escrowFlag(t, false)
	db := setupParkedPlanDB(t)
	_, dayID := seedParkedPlan(t, db, models.ChefModeTest)

	sweepStuckDays()

	var day models.MealPlanDay
	require.NoError(t, db.First(&day, "id = ?", dayID).Error)
	require.Equal(t, models.MealPlanDayConfirmed, day.Status,
		"a parked day must wait for the return to live, not be voided")
}
