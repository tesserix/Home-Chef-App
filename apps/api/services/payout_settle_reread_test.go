package services

// payout_settle_reread_test.go — #508. The money seam is dispatched by the hold's
// FRESH status (settlePayout re-reads), not by the transition the calling goroutine
// won. Flags OFF here, so the seams are no-ops and every assertion is on the DB
// state advance — the point is WHICH status the dispatcher acts on.

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func settleReadSettledAt(t *testing.T, db *gorm.DB, id uuid.UUID) *time.Time {
	t.Helper()
	var nt sql.NullTime
	require.NoError(t, db.Raw(`SELECT payout_settled_at FROM orders WHERE id = ?`, id.String()).Scan(&nt).Error)
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

func TestSettlePayout_ReleasedStampsSettled(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleased, "delivered", nil)

	require.NoError(t, settlePayout(db, aggTypeOrder, id))
	require.NotNil(t, settleReadSettledAt(t, db, id), "a released hold is settled")
}

func TestSettlePayout_ReversedStampsSettled(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReversed, "delivered", nil)

	require.NoError(t, settlePayout(db, aggTypeOrder, id))
	require.NotNil(t, settleReadSettledAt(t, db, id), "a reversed hold is settled")
}

// The observable heart of #508: the release wrapper on a NON-terminal row must NOT
// settle it. The OLD settleRelease ran releaseMoney + stamped unconditionally; the
// new one re-reads, sees a non-terminal status, and does nothing.
func TestSettleRelease_NonTerminalNotSettled(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleaseEligible, "delivered", nil)

	require.NoError(t, settleRelease(db, aggTypeOrder, id))
	require.Nil(t, settleReadSettledAt(t, db, id), "a release_eligible (non-terminal) hold is never settled by the seam")
}

// The race the fix closes: this goroutine "won" a release (calls settleRelease), but
// a concurrent refund has flipped the row to reversed. The dispatcher must act on the
// FRESH `reversed` status — settle it as a reversal, leaving it reversed, never
// re-releasing it. (With flags off the seam is a no-op; the assertion is that the
// release wrapper handled a reversed row correctly and did not error or un-reverse it.)
func TestSettleRelease_DispatchesFreshReversedStatus(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleased, "delivered", nil)
	// A concurrent refund flips released → reversed after this goroutine committed its release.
	require.NoError(t, db.Exec(`UPDATE orders SET payout_hold_status = ? WHERE id = ?`,
		string(models.PayoutHoldReversed), id.String()).Error)

	require.NoError(t, settleRelease(db, aggTypeOrder, id))
	require.Equal(t, models.PayoutHoldReversed, loadOrderHold(t, db, id), "the row stays reversed — no re-release")
	require.NotNil(t, settleReadSettledAt(t, db, id), "settled once, as a reversal")
}

func TestSettlePayout_AlreadySettledNoOp(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	id, _ := seedCrossOrder(t, db, models.PayoutHoldReleased, "delivered", nil)
	require.NoError(t, settlePayout(db, aggTypeOrder, id))
	first := settleReadSettledAt(t, db, id)
	require.NotNil(t, first)

	// Second call must not re-run the seam nor re-stamp.
	require.NoError(t, settlePayout(db, aggTypeOrder, id))
	require.Equal(t, first.UnixNano(), settleReadSettledAt(t, db, id).UnixNano(), "settled stamp is not moved by a re-drive")
}

// settlePayout dispatches identically across all three aggregate types.
func TestSettlePayout_MealPlanDayAndGroup(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)

	dayID := seedCrossDay(t, db, models.PayoutHoldReversed, nil)
	require.NoError(t, settlePayout(db, aggTypeMealPlanDay, dayID))
	var dayAt sql.NullTime
	require.NoError(t, db.Raw(`SELECT payout_settled_at FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&dayAt).Error)
	require.True(t, dayAt.Valid, "reversed meal-plan day settled")

	groupID := seedCrossGroup(t, db, models.PayoutHoldReleased, nil)
	require.NoError(t, settlePayout(db, aggTypeGroupOrder, groupID))
	var grpAt sql.NullTime
	require.NoError(t, db.Raw(`SELECT payout_settled_at FROM group_orders WHERE id = ?`, groupID.String()).Scan(&grpAt).Error)
	require.True(t, grpAt.Valid, "released group order settled")
}

func TestIsAlreadyReversedErr(t *testing.T) {
	require.False(t, isAlreadyReversedErr(nil))
	for _, msg := range []string{"transfer already reversed", "Fully Reversed", "not_reversible"} {
		require.True(t, isAlreadyReversedErr(errorString(msg)), msg)
	}
	require.False(t, isAlreadyReversedErr(errorString("network timeout")))
}

type errorString string

func (e errorString) Error() string { return string(e) }
