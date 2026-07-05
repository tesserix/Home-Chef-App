package services

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupOrderSourceDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, table := range []string{"meal_plan_days", "meal_subscription_fulfillments", "group_orders"} {
		require.NoError(t, db.Exec(`CREATE TABLE `+table+` (id TEXT PRIMARY KEY, order_id TEXT, deleted_at DATETIME)`).Error)
	}
	return db
}

func link(t *testing.T, db *gorm.DB, table string, orderID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO `+table+` (id, order_id) VALUES (?, ?)`, uuid.NewString(), orderID.String()).Error)
}

func TestClassifyOrderSources(t *testing.T) {
	db := setupOrderSourceDB(t)

	alacarte := uuid.New()
	mealPlan := uuid.New()
	subscription := uuid.New()
	group := uuid.New()
	link(t, db, "meal_plan_days", mealPlan)
	link(t, db, "meal_subscription_fulfillments", subscription)
	link(t, db, "group_orders", group)

	got := ClassifyOrderSources(db, []uuid.UUID{alacarte, mealPlan, subscription, group})

	require.Equal(t, models.OrderSourceAlacarte, got[alacarte], "an unlinked order is à-la-carte")
	require.Equal(t, models.OrderSourceMealPlan, got[mealPlan])
	require.Equal(t, models.OrderSourceSubscription, got[subscription])
	require.Equal(t, models.OrderSourceGroup, got[group])
}

func TestClassifyOrderSources_Empty(t *testing.T) {
	db := setupOrderSourceDB(t)
	require.Empty(t, ClassifyOrderSources(db, nil))
}

// Only the ids asked for are classified — a link for an unrelated order doesn't
// leak into the result.
func TestClassifyOrderSources_ScopedToInput(t *testing.T) {
	db := setupOrderSourceDB(t)
	asked := uuid.New()
	other := uuid.New()
	link(t, db, "meal_plan_days", other) // not in the input set

	got := ClassifyOrderSources(db, []uuid.UUID{asked})
	require.Len(t, got, 1)
	require.Equal(t, models.OrderSourceAlacarte, got[asked])
}
