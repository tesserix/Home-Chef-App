package services

// group_cancel_reverse_test.go — W-A (#456). ReverseGroupHoldForCancel drives a
// CANCELLED group's chef payout hold → reversed (via the guarded hold machine) and
// stamps settled, self-guarding on status==cancelled so it never reverses a delivered
// group and is safe to call on the retry/recovery path. Flags OFF ⇒ the money seam is
// a no-op; assertions are on the DB state advance.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func setGroupStatus(t *testing.T, db *gorm.DB, id uuid.UUID, status models.GroupOrderStatus) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE group_orders SET status = ? WHERE id = ?`, string(status), id.String()).Error)
}

// loadGroupHold lives in payout_disputed_fanout_test.go (same package) — reused here.

func loadGroupSettledAt(t *testing.T, db *gorm.DB, id uuid.UUID) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM group_orders WHERE id = ? AND payout_settled_at IS NOT NULL`, id.String()).Scan(&n).Error)
	return n == 1
}

func TestReverseGroupHoldForCancel_DrivesReversed(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	// The pre-delivery held-transfer case: hold=none but a cancelled group.
	for _, hold := range []models.PayoutHoldStatus{
		models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation,
		models.PayoutHoldReleaseEligible, models.PayoutHoldReleased,
	} {
		id := seedCrossGroup(t, db, hold, nil)
		setGroupStatus(t, db, id, models.GroupOrderCancelled)

		require.NoError(t, ReverseGroupHoldForCancel(db, id, "cancel"))
		require.Equal(t, models.PayoutHoldReversed, loadGroupHold(t, db, id), "%s → reversed", hold)
		require.True(t, loadGroupSettledAt(t, db, id), "settled stamped for %s", hold)
	}
}

func TestReverseGroupHoldForCancel_DeliveredGroupNoOp(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id := seedCrossGroup(t, db, models.PayoutHoldReleaseEligible, nil) // seed sets status=delivered

	require.NoError(t, ReverseGroupHoldForCancel(db, id, "cancel"))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadGroupHold(t, db, id),
		"a delivered (non-cancelled) group's payout is never reversed")
	require.False(t, loadGroupSettledAt(t, db, id))
}

func TestReverseGroupHoldForCancel_Idempotent(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id := seedCrossGroup(t, db, models.PayoutHoldAwaitingConfirmation, nil)
	setGroupStatus(t, db, id, models.GroupOrderCancelled)

	require.NoError(t, ReverseGroupHoldForCancel(db, id, "cancel"))
	require.NoError(t, ReverseGroupHoldForCancel(db, id, "cancel")) // second call (recovery/retry)
	require.Equal(t, models.PayoutHoldReversed, loadGroupHold(t, db, id))
}
