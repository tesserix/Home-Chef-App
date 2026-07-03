package services

// payout_auto_confirm_cron_test.go — the auto-confirm sweep (#387 follow-up). A
// delivered, dispute-free hold that has sat in awaiting_customer_confirmation past
// the confirm window is advanced to release_eligible with no customer tap; a hold
// with an open issue lands in disputed; a hold still inside the window is left
// alone. Every genuine transition emits its NATS event via the outbox. Runs on the
// same in-memory sqlite harness as payout_hold_test.go (setupHoldDB).

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// seedAwaitingOrder inserts an awaiting_customer_confirmation regular order whose
// delivered_at is `age` before now and customer_confirmed_at is NULL.
func seedAwaitingOrder(t *testing.T, db *gorm.DB, age time.Duration) uuid.UUID {
	t.Helper()
	id := uuid.New()
	delivered := time.Now().Add(-age)
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status, delivered_at) VALUES (?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), "delivered", "order_rzp_123",
		string(models.PayoutHoldAwaitingConfirmation), delivered).Error)
	return id
}

// withSweepDB swaps database.DB to the sqlite harness for the duration of fn.
func withSweepDB(t *testing.T, db *gorm.DB, fn func()) {
	t.Helper()
	orig := database.DB
	database.DB = db
	defer func() { database.DB = orig }()
	fn()
}

// Test D: a dispute-free order past the window is auto-advanced to release_eligible
// and emits exactly one hold_release_eligible event.
func TestPayoutAutoConfirm_PastWindowReleaseEligible(t *testing.T) {
	db := setupHoldDB(t)
	id := seedAwaitingOrder(t, db, 30*time.Hour) // past the default 24h window

	withSweepDB(t, db, func() { runPayoutAutoConfirmScan(context.Background()) })

	require.Equal(t, models.PayoutHoldReleaseEligible, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// Test E: an order still inside the window is left untouched — no transition, no event.
func TestPayoutAutoConfirm_InsideWindowUntouched(t *testing.T) {
	db := setupHoldDB(t)
	id := seedAwaitingOrder(t, db, 2*time.Hour) // recent, well inside 24h

	withSweepDB(t, db, func() { runPayoutAutoConfirmScan(context.Background()) })

	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 0, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// Test F: a past-window order with an open pending OrderIssue is advanced to
// disputed (never release_eligible) and emits hold_disputed.
func TestPayoutAutoConfirm_PastWindowOpenIssueDisputes(t *testing.T) {
	db := setupHoldDB(t)
	id := seedAwaitingOrder(t, db, 30*time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), id.String(), string(models.IssuePending)).Error)

	withSweepDB(t, db, func() { runPayoutAutoConfirmScan(context.Background()) })

	require.Equal(t, models.PayoutHoldDisputed, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed))
	require.Equal(t, 0, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// Test G: a past-window meal-plan day is advanced to release_eligible + event.
func TestPayoutAutoConfirm_MealPlanDayPastWindow(t *testing.T) {
	db := setupHoldDB(t)
	dayID := uuid.New()
	delivered := time.Now().Add(-30 * time.Hour)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, payout_hold_status, delivered_at) VALUES (?,?,?,?,?)`,
		dayID.String(), uuid.NewString(), "delivered",
		string(models.PayoutHoldAwaitingConfirmation), delivered).Error)

	withSweepDB(t, db, func() { runPayoutAutoConfirmScan(context.Background()) })

	var day models.MealPlanDay
	require.NoError(t, db.First(&day, "id = ?", dayID).Error)
	require.Equal(t, models.PayoutHoldReleaseEligible, day.PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// Test H: a 48h window override leaves an order delivered 30h ago awaiting.
func TestPayoutAutoConfirm_WindowOverrideRespected(t *testing.T) {
	db := setupHoldDB(t)
	require.NoError(t, db.Exec(`INSERT INTO platform_settings (id, key, value, type) VALUES (?,?,?,?)`,
		uuid.NewString(), "payout.customer_confirm_window_hours", "48", "number").Error)
	id := seedAwaitingOrder(t, db, 30*time.Hour) // inside the widened 48h window

	withSweepDB(t, db, func() { runPayoutAutoConfirmScan(context.Background()) })

	require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 0, countOutbox(t, db, SubjectHoldReleaseEligible))
}

// Test I: running the scan twice advances the hold once and emits exactly one event.
func TestPayoutAutoConfirm_IdempotentAcrossRuns(t *testing.T) {
	db := setupHoldDB(t)
	id := seedAwaitingOrder(t, db, 30*time.Hour)

	withSweepDB(t, db, func() {
		runPayoutAutoConfirmScan(context.Background())
		runPayoutAutoConfirmScan(context.Background())
	})

	require.Equal(t, models.PayoutHoldReleaseEligible, loadOrder(t, db, id).PayoutHoldStatus)
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))
}
