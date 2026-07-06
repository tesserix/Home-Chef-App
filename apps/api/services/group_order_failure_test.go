package services

// group_order_failure_test.go — #393/#594. A failed/returned delivery on a GROUP order's
// consolidated Order currently freezes nothing: the consolidated order has no
// razorpay_order_id (RecordDeliveryFailure skips it) and no meal_plan_days row
// (MarkMealPlanDayFailed skips it), so TerminalizeDeliveryFailure returned froze=false and
// the group stranded — money-safe (hold stays none) but the participants are never
// refunded and no admin is alerted. Slice A mirrors the meal-plan-day freeze: mark the
// group `failed` (NON-terminal) + dispute its payout hold. No money moves.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func loadGroupStatus(t *testing.T, db *gorm.DB, id uuid.UUID) models.GroupOrderStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM group_orders WHERE id = ?`, id.String()).Scan(&s).Error)
	return models.GroupOrderStatus(s)
}

// ── SetGroupOrderHoldDisputed ────────────────────────────────────────────────

func TestSetGroupOrderHoldDisputed_FreezesFromPreTerminal(t *testing.T) {
	for _, from := range []models.PayoutHoldStatus{models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation} {
		t.Run(string(from), func(t *testing.T) {
			db := setupCrossguardDB(t)
			gid := seedCrossGroup(t, db, from, nil)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return SetGroupOrderHoldDisputed(tx, gid)
			}))
			require.Equal(t, models.PayoutHoldDisputed, loadGroupHold(t, db, gid))
			require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed))
		})
	}
}

func TestSetGroupOrderHoldDisputed_NoOpOnSettled(t *testing.T) {
	for _, from := range []models.PayoutHoldStatus{
		models.PayoutHoldReleaseEligible, models.PayoutHoldReleased,
		models.PayoutHoldReversed, models.PayoutHoldWithheld, models.PayoutHoldDisputed,
	} {
		t.Run(string(from), func(t *testing.T) {
			db := setupCrossguardDB(t)
			gid := seedCrossGroup(t, db, from, nil)
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				return SetGroupOrderHoldDisputed(tx, gid)
			}))
			require.Equal(t, from, loadGroupHold(t, db, gid), "settled hold untouched")
			require.Equal(t, 0, countOutbox(t, db, SubjectHoldDisputed))
		})
	}
}

// ── MarkGroupOrderFailed ─────────────────────────────────────────────────────

func TestMarkGroupOrderFailed_FreezesAndMarks(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	gid := seedCrossGroup(t, db, models.PayoutHoldNone, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderConfirmed)

	var froze bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		f, err := MarkGroupOrderFailed(tx, orderID)
		froze = f
		return err
	}))
	require.True(t, froze)
	require.Equal(t, models.GroupOrderFailed, loadGroupStatus(t, db, gid))
	require.Equal(t, models.PayoutHoldDisputed, loadGroupHold(t, db, gid))
	require.Equal(t, 1, countOutbox(t, db, SubjectGroupOrderFailed))
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed))
}

func TestMarkGroupOrderFailed_Idempotent(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	gid := seedCrossGroup(t, db, models.PayoutHoldNone, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderConfirmed)

	var second bool
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			f, err := MarkGroupOrderFailed(tx, orderID)
			if i == 1 {
				second = f
			}
			return err
		}))
	}
	require.False(t, second, "re-fire is a froze=false no-op")
	require.Equal(t, 1, countOutbox(t, db, SubjectGroupOrderFailed))
}

func TestMarkGroupOrderFailed_NotAGroupOrder(t *testing.T) {
	db := setupCrossguardDB(t)
	var froze bool
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		f, err := MarkGroupOrderFailed(tx, uuid.New())
		froze = f
		return err
	}))
	require.False(t, froze)
}

func TestMarkGroupOrderFailed_TerminalNoOp(t *testing.T) {
	for _, term := range []models.GroupOrderStatus{
		models.GroupOrderDelivered, models.GroupOrderCancelled, models.GroupOrderExpired,
	} {
		t.Run(string(term), func(t *testing.T) {
			db := setupCrossguardDB(t)
			orderID := uuid.New()
			gid := seedCrossGroup(t, db, models.PayoutHoldAwaitingConfirmation, &orderID)
			setGroupStatus(t, db, gid, term)

			var froze bool
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				f, err := MarkGroupOrderFailed(tx, orderID)
				froze = f
				return err
			}))
			require.False(t, froze)
			require.Equal(t, term, loadGroupStatus(t, db, gid), "terminal status preserved")
			require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadGroupHold(t, db, gid), "hold untouched")
		})
	}
}

// ── TerminalizeDeliveryFailure wiring ────────────────────────────────────────

func TestTerminalizeDeliveryFailure_GroupOrderFreezesGroup(t *testing.T) {
	db := setupCrossguardDB(t)
	// A group consolidated order: no razorpay_order_id, linked to a group order.
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	require.NoError(t, db.Exec(`UPDATE orders SET razorpay_order_id = '' WHERE id = ?`, orderID.String()).Error)
	gid := seedCrossGroup(t, db, models.PayoutHoldNone, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderConfirmed)
	order := loadFailureOrder(t, db, orderID)

	froze, err := TerminalizeDeliveryFailure(db, order, models.FailureDriverNoShow, "courier", map[string]any{"delivery_id": "dl_1"})
	require.NoError(t, err)
	require.True(t, froze, "the group freezes even though gateway + meal-plan-day paths skip")
	require.Equal(t, models.GroupOrderFailed, loadGroupStatus(t, db, gid))
	require.Equal(t, models.PayoutHoldDisputed, loadGroupHold(t, db, gid))
	require.Equal(t, models.PayoutHoldNone, loadOrderHold(t, db, orderID), "shell order hold untouched")
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed))
	require.Equal(t, 1, countOutbox(t, db, SubjectGroupOrderFailed))
	require.Equal(t, 0, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending), "no issue on the shell order")
}

// ── freeze-integrity regressions (found by adversarial verify; mirror #590) ──

// A late/replayed delivered event must not resurrect a frozen `failed` group back to
// delivered (which would orphan the disputed hold and hide it from resolution).
func TestMarkGroupOrderDelivered_DoesNotResurrectFailedGroup(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	gid := seedCrossGroup(t, db, models.PayoutHoldNone, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderConfirmed)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := MarkGroupOrderFailed(tx, orderID)
		return err
	}))
	require.Equal(t, models.GroupOrderFailed, loadGroupStatus(t, db, gid))

	MarkGroupOrderDelivered(orderID) // stray delivered event

	require.Equal(t, models.GroupOrderFailed, loadGroupStatus(t, db, gid), "failed group not resurrected")
	require.Equal(t, models.PayoutHoldDisputed, loadGroupHold(t, db, gid), "disputed hold not re-parked")
}

// A `failed` group's disputed hold must NOT be auto-cleared to release_eligible by an
// unrelated OrderIssue rejection on the shared consolidated order.
func TestReleaseDisputedGroupHolds_FailedGroupNotCleared(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	gid := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderFailed)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldDisputed, loadGroupHold(t, db, gid), "failed group stays disputed")

	// Control: a non-failed (delivered) disputed group on a cleared order DOES release.
	deliveredGroup := seedCrossGroup(t, db, models.PayoutHoldDisputed, &orderID) // status='delivered'
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadGroupHold(t, db, deliveredGroup), "delivered group clears")
}

// The money seam refuses to pay a `failed` group even if its hold reached release_eligible.
func TestReleaseHold_BlocksFailedGroup(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	gid := seedCrossGroup(t, db, models.PayoutHoldReleaseEligible, &orderID)
	setGroupStatus(t, db, gid, models.GroupOrderFailed)

	err := ReleaseHold(db, aggTypeGroupOrder, gid)
	require.ErrorIs(t, err, ErrHoldNotEligible, "a failed group is never releasable")
	require.Equal(t, models.PayoutHoldReleaseEligible, loadGroupHold(t, db, gid), "hold untouched")
}
