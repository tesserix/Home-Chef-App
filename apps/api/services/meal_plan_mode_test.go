package services

// meal_plan_mode_test.go — #963. A meal-plan day order must inherit the plan's
// live/test partition.
//
// This is a money-safety guarantee, not bookkeeping. `models/mode.go` promises
// that test rows are "excluded from every real-money and reporting path" and
// that mode is "a snapshot taken at creation ... so a refund on an order paid in
// test mode still routes to the test gateway". A day order that defaults to
// `live` under a `test` plan breaks both, and `NormalizeMode`'s deliberate
// "anything not test becomes live" asymmetry makes the omission silent.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupDayOrderModeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	createTablesFor(t, db,
		&models.Order{}, &models.OrderItem{}, &models.MealPlanDay{},
		&models.ChefProfile{}, &models.OutboxEvent{})
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func TestGenerateDayOrder_InheritsPlanMode(t *testing.T) {
	session := uuid.New()
	cases := []struct {
		name       string
		planMode   string
		wantMode   string
		wantIsTest bool
	}{
		{"test plan spawns TEST orders", models.ChefModeTest, models.ChefModeTest, true},
		{"live plan spawns live orders", models.ChefModeLive, models.ChefModeLive, false},
		{"empty mode falls back to live", "", models.ChefModeLive, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := setupDayOrderModeDB(t)
			chefID := uuid.New()
			require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name) VALUES (?,?,?)`,
				chefID.String(), uuid.New().String(), "Amma Ka Kitchen").Error)

			plan := models.MealPlan{
				ID: uuid.New(), CustomerID: uuid.New(), ChefID: chefID, Currency: "INR",
			}
			plan.Mode = tc.planMode
			if tc.wantIsTest {
				plan.TestSessionID = &session
			}
			day := models.MealPlanDay{
				ID: uuid.New(), MealPlanID: plan.ID, Date: time.Now(),
				DishName: "Bisi Bele Bath", Price: 140,
			}
			require.NoError(t, db.Exec(
				`INSERT INTO meal_plan_days (id, meal_plan_id, date, dish_name, price, status) VALUES (?,?,?,?,?,?)`,
				day.ID.String(), plan.ID.String(), day.Date, day.DishName, day.Price, "scheduled").Error)

			require.NoError(t, generateDayOrder(&plan, &day, models.Address{City: "Bengaluru", Country: "IN"}))

			var order models.Order
			require.NoError(t, db.First(&order, "chef_id = ?", chefID.String()).Error)
			assert.Equal(t, tc.wantMode, order.Mode,
				"a %q plan must not spawn a %q order", tc.planMode, order.Mode)

			if tc.wantIsTest {
				require.NotNil(t, order.TestSessionID,
					"test session must ride along so a session purge collects the order")
				assert.Equal(t, session, *order.TestSessionID)
			}
		})
	}
}
