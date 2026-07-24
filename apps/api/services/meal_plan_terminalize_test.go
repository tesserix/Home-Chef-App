package services

// meal_plan_terminalize_test.go — reject/cancel/expiry left a dead plan's days as
// `accepted` (the refund seam never sets day status). TerminalizeCancelledPlanDays
// drives every still-open day terminal: a refunded day → refunded, everything else →
// cancelled, already-terminal days untouched.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupTerminalizeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, status TEXT,
		refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`).Error)
	return db
}

func seedTermDay(t *testing.T, db *gorm.DB, planID uuid.UUID, status models.MealPlanDayStatus, refundTxn string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var rt any
	if refundTxn != "" {
		rt = refundTxn
	}
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, refund_txn_id) VALUES (?,?,?,?)`,
		id.String(), planID.String(), string(status), rt).Error)
	return id
}

func termDayStatus(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&s).Error)
	return s
}

func TestTerminalizeCancelledPlanDays(t *testing.T) {
	db := setupTerminalizeDB(t)
	planID := uuid.New()
	accepted := seedTermDay(t, db, planID, models.MealPlanDayAccepted, "")          // open → cancelled
	confirmed := seedTermDay(t, db, planID, models.MealPlanDayConfirmed, "")         // open → cancelled
	refunded := seedTermDay(t, db, planID, models.MealPlanDayConfirmed, "txn_123")   // has refund txn → refunded
	delivered := seedTermDay(t, db, planID, models.MealPlanDayDelivered, "")         // terminal → untouched
	skipped := seedTermDay(t, db, planID, models.MealPlanDaySkipped, "")             // terminal → untouched
	// A day on a DIFFERENT plan must not be touched.
	other := seedTermDay(t, db, uuid.New(), models.MealPlanDayAccepted, "")

	require.NoError(t, TerminalizeCancelledPlanDays(db, planID))

	require.Equal(t, string(models.MealPlanDayCancelled), termDayStatus(t, db, accepted), "open day → cancelled")
	require.Equal(t, string(models.MealPlanDayCancelled), termDayStatus(t, db, confirmed), "open day → cancelled")
	require.Equal(t, string(models.MealPlanDayRefunded), termDayStatus(t, db, refunded), "refunded-txn day → refunded")
	require.Equal(t, string(models.MealPlanDayDelivered), termDayStatus(t, db, delivered), "terminal untouched")
	require.Equal(t, string(models.MealPlanDaySkipped), termDayStatus(t, db, skipped), "terminal untouched")
	require.Equal(t, string(models.MealPlanDayAccepted), termDayStatus(t, db, other), "other plan untouched")

	// Idempotent — a second run changes nothing.
	require.NoError(t, TerminalizeCancelledPlanDays(db, planID))
	require.Equal(t, string(models.MealPlanDayCancelled), termDayStatus(t, db, accepted))
	require.Equal(t, string(models.MealPlanDayRefunded), termDayStatus(t, db, refunded))
}
