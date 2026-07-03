package services

// payout_reconcile_cron_test.go — the released/reversed-but-unsettled payout
// reconcile (#459). The primary path (ReleaseHold/ReverseHold) flips the hold
// status in a committed tx and only THEN runs the money seam post-commit; if that
// seam fails the row is left released/reversed with money unmoved and settled_at
// NULL. The reconcile re-drives exactly those drift rows and stamps payout_settled_at
// on seam success. Everything is flag-gated: a pure DB no-op while both escrow flags
// are OFF. Runs on the same in-memory sqlite harness as payout_hold_test.go.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

// settledAtOf reads the raw payout_settled_at for an order (nil when NULL).
func settledAtOf(t *testing.T, db *gorm.DB, id uuid.UUID) *time.Time {
	t.Helper()
	var ts *time.Time
	require.NoError(t, db.Raw(`SELECT payout_settled_at FROM orders WHERE id = ?`, id.String()).Scan(&ts).Error)
	return ts
}

// withEscrowFlag sets config.AppConfig for the duration of fn (save/restore).
func withEscrowFlag(t *testing.T, orderFlag bool, fn func()) {
	t.Helper()
	saved := config.AppConfig
	defer func() { config.AppConfig = saved }()
	config.AppConfig = &config.Config{OrderPayoutAutoReleaseEnabled: orderFlag}
	fn()
}

// A released, unsettled row is left untouched when both escrow flags are off — the
// scan gate makes the whole sweep a pure DB no-op.
func TestPayoutReconcile_NoopWhenFlagsOff(t *testing.T) {
	db := setupHoldDB(t)
	id := seedReleasedOrder(t, db, models.PayoutHoldReleased, nil)

	// Default config.AppConfig is nil → both payoutMovementEnabled() and
	// MealPlanEscrowActive() are false → the scan returns before any query.
	saved := config.AppConfig
	config.AppConfig = nil
	defer func() { config.AppConfig = saved }()

	withSweepDB(t, db, func() { runPayoutReconcileScan(context.Background()) })

	require.Nil(t, settledAtOf(t, db, id), "flags-off scan must never stamp settled_at")
}

// A released, unsettled row is re-driven (seam no-ops nil in the harness) and
// stamped; a second scan must NOT re-pick the now-settled row.
func TestPayoutReconcile_ReleasedUnsettledIsDriven(t *testing.T) {
	db := setupHoldDB(t)
	id := seedReleasedOrder(t, db, models.PayoutHoldReleased, nil)

	withEscrowFlag(t, true, func() {
		// GetRazorpay() is nil in the harness → ReleaseOrderPayouts no-ops and
		// returns nil == "seam succeeded" → settleRelease stamps settled_at.
		withSweepDB(t, db, func() { runPayoutReconcileScan(context.Background()) })

		first := settledAtOf(t, db, id)
		require.NotNil(t, first, "released+unsettled row must be stamped on seam success")

		time.Sleep(2 * time.Millisecond)
		withSweepDB(t, db, func() { runPayoutReconcileScan(context.Background()) })
		second := settledAtOf(t, db, id)
		require.NotNil(t, second)
		require.WithinDuration(t, *first, *second, time.Millisecond, "an already-settled row must never be re-driven")
	})
}

// A released row already carrying settled_at is never re-driven (stamp unchanged).
func TestPayoutReconcile_ReleasedAlreadySettledSkipped(t *testing.T) {
	db := setupHoldDB(t)
	past := time.Now().Add(-48 * time.Hour).UTC().Truncate(time.Second)
	id := seedReleasedOrder(t, db, models.PayoutHoldReleased, &past)

	withEscrowFlag(t, true, func() {
		withSweepDB(t, db, func() { runPayoutReconcileScan(context.Background()) })
	})

	got := settledAtOf(t, db, id)
	require.NotNil(t, got)
	require.WithinDuration(t, past, got.UTC(), time.Second, "already-settled stamp must be untouched")
}

// A reversed-but-unsettled row is re-driven and stamped on seam success.
func TestPayoutReconcile_ReversedUnsettledIsDriven(t *testing.T) {
	db := setupHoldDB(t)
	id := seedReleasedOrder(t, db, models.PayoutHoldReversed, nil)

	withEscrowFlag(t, true, func() {
		withSweepDB(t, db, func() { runPayoutReconcileScan(context.Background()) })
	})

	require.NotNil(t, settledAtOf(t, db, id), "reversed+unsettled row must be stamped on seam success")
}

// stampPayoutSettled is a conditional UPDATE on settled_at IS NULL: it stamps once
// and a second call leaves the original timestamp unchanged.
func TestStampPayoutSettled_ConditionalOnce(t *testing.T) {
	db := setupHoldDB(t)
	id := seedReleasedOrder(t, db, models.PayoutHoldReleased, nil)

	require.NoError(t, stampPayoutSettled(db, aggTypeOrder, id))
	first := settledAtOf(t, db, id)
	require.NotNil(t, first, "first stamp must set settled_at")

	time.Sleep(2 * time.Millisecond)
	require.NoError(t, stampPayoutSettled(db, aggTypeOrder, id))
	second := settledAtOf(t, db, id)
	require.NotNil(t, second)
	require.WithinDuration(t, *first, *second, time.Millisecond, "second stamp must not overwrite the original timestamp")
}

// ReleaseDayPayout re-drive is idempotent: two calls both return nil (no error).
//
// The harness bound: GetRazorpay() is nil when MealPlanEscrowEnabled is off, so the
// call no-ops at the flag guard — this test asserts the no-error CONTRACT the
// reconcile depends on (a re-drive of a released day never errors), not gateway-level
// dedupe. The durable idempotency guard is payout_settled_at IS NULL (a settled day
// is never re-picked); the gateway "already released" tolerance is the second layer.
func TestReleaseDayPayout_IdempotentRedrive(t *testing.T) {
	db := setupHoldDB(t)
	day := &models.MealPlanDay{ID: uuid.New(), PayoutTransferID: "trf_redrive_1"}

	// Escrow flag off (default nil config) → ReleaseDayPayout is a no-op guard.
	require.NoError(t, ReleaseDayPayout(db, day))
	require.NoError(t, ReleaseDayPayout(db, day), "re-driving a released day must not error")
}
