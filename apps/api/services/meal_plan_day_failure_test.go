package services

// meal_plan_day_failure_test.go — #393 slice A (meal-plan DAY freeze). A terminally
// failed delivery on a per-day fulfilment order must not silently stall the plan: the
// per-day order is a shell with no razorpay_order_id (RecordDeliveryFailure skips it),
// so the day's money lives on the meal_plan_days row. This slice marks the day `failed`
// (a NON-terminal status — the plan waits for admin day-resolution) and FREEZES the
// day's payout hold to `disputed`. No money moves. Reuses the setupCrossguardDB harness
// (meal_plan_days + order_issues + outbox).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func loadDayStatus(t *testing.T, db *gorm.DB, dayID uuid.UUID) models.MealPlanDayStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&s).Error)
	return models.MealPlanDayStatus(s)
}

// ── SetMealPlanDayHoldDisputed ───────────────────────────────────────────────

func TestSetMealPlanDayHoldDisputed_FreezesFromPreTerminal(t *testing.T) {
	for _, from := range []models.PayoutHoldStatus{models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation} {
		t.Run(string(from), func(t *testing.T) {
			db := setupCrossguardDB(t)
			dayID := seedCrossDay(t, db, from, nil)

			err := db.Transaction(func(tx *gorm.DB) error {
				return SetMealPlanDayHoldDisputed(tx, dayID)
			})
			require.NoError(t, err)
			require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID))
			require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed), "emits payout.hold_disputed once")
		})
	}
}

func TestSetMealPlanDayHoldDisputed_NoOpOnSettled(t *testing.T) {
	// The #458 invariant: disputed must never un-settle a hold that already moved
	// (or is queued to move) real money.
	for _, from := range []models.PayoutHoldStatus{
		models.PayoutHoldReleaseEligible, models.PayoutHoldReleased,
		models.PayoutHoldReversed, models.PayoutHoldWithheld, models.PayoutHoldDisputed,
	} {
		t.Run(string(from), func(t *testing.T) {
			db := setupCrossguardDB(t)
			dayID := seedCrossDay(t, db, from, nil)

			err := db.Transaction(func(tx *gorm.DB) error {
				return SetMealPlanDayHoldDisputed(tx, dayID)
			})
			require.NoError(t, err)
			require.Equal(t, from, loadDayHold(t, db, dayID), "settled hold untouched")
			require.Equal(t, 0, countOutbox(t, db, SubjectHoldDisputed), "no event on no-op")
		})
	}
}

// ── MarkMealPlanDayFailed ────────────────────────────────────────────────────

func TestMarkMealPlanDayFailed_FreezesAndMarks(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	dayID := seedCrossDay(t, db, models.PayoutHoldNone, &orderID)
	setDayStatus(t, db, dayID, models.MealPlanDayPrepared)

	var froze bool
	err := db.Transaction(func(tx *gorm.DB) error {
		f, err := MarkMealPlanDayFailed(tx, orderID)
		froze = f
		return err
	})
	require.NoError(t, err)
	require.True(t, froze)
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID))
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayFailed), "emits meal_plans.day_failed once")
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldDisputed))
}

func TestMarkMealPlanDayFailed_Idempotent(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	dayID := seedCrossDay(t, db, models.PayoutHoldNone, &orderID)
	setDayStatus(t, db, dayID, models.MealPlanDayPrepared)

	var second bool
	for i := 0; i < 2; i++ {
		require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
			f, err := MarkMealPlanDayFailed(tx, orderID)
			if i == 1 {
				second = f
			}
			return err
		}))
	}
	require.False(t, second, "re-fire is a froze=false no-op")
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayFailed), "no duplicate day_failed event")
}

func TestMarkMealPlanDayFailed_NotADayOrder(t *testing.T) {
	db := setupCrossguardDB(t)
	var froze bool
	err := db.Transaction(func(tx *gorm.DB) error {
		f, err := MarkMealPlanDayFailed(tx, uuid.New()) // no day links to this order
		froze = f
		return err
	})
	require.NoError(t, err)
	require.False(t, froze)
}

func TestMarkMealPlanDayFailed_TerminalDayNoOp(t *testing.T) {
	// A late failure event must not overwrite an already-terminal day (e.g. a day that
	// was delivered, or refunded via day-skip) — the day keeps its terminal status and
	// its settled hold is untouched.
	for _, term := range []models.MealPlanDayStatus{
		models.MealPlanDayDelivered, models.MealPlanDayRefunded,
		models.MealPlanDayCancelled, models.MealPlanDaySkipped, models.MealPlanDayDeclined,
	} {
		t.Run(string(term), func(t *testing.T) {
			db := setupCrossguardDB(t)
			orderID := uuid.New()
			dayID := seedCrossDay(t, db, models.PayoutHoldAwaitingConfirmation, &orderID)
			setDayStatus(t, db, dayID, term)

			var froze bool
			require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
				f, err := MarkMealPlanDayFailed(tx, orderID)
				froze = f
				return err
			}))
			require.False(t, froze)
			require.Equal(t, term, loadDayStatus(t, db, dayID), "terminal status preserved")
			require.Equal(t, models.PayoutHoldAwaitingConfirmation, loadDayHold(t, db, dayID), "hold untouched")
			require.Equal(t, 0, countOutbox(t, db, SubjectMealPlanDayFailed))
		})
	}
}

// ── TerminalizeDeliveryFailure wiring ────────────────────────────────────────

func TestTerminalizeDeliveryFailure_DayLinkedNonGatewayFreezesDay(t *testing.T) {
	db := setupCrossguardDB(t)
	// A per-day shell order: linked to a meal-plan day, no razorpay_order_id.
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	require.NoError(t, db.Exec(`UPDATE orders SET razorpay_order_id = '' WHERE id = ?`, orderID.String()).Error)
	dayID := seedCrossDay(t, db, models.PayoutHoldNone, &orderID)
	setDayStatus(t, db, dayID, models.MealPlanDayPrepared)
	order := loadFailureOrder(t, db, orderID)

	froze, err := TerminalizeDeliveryFailure(db, order, models.FailureDriverNoShow, "courier", map[string]any{"delivery_id": "dl_1"})
	require.NoError(t, err)
	require.True(t, froze, "the day shell freezes even though the gateway path skips")
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID))
	require.Equal(t, models.PayoutHoldNone, loadOrderHold(t, db, orderID), "shell order hold untouched — money lives on the day")
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed), "one delivery.failed notification")
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayFailed), "one day_failed event")
	// No OrderIssue opened on the shell — the day, not the shell order, is the dispute.
	require.Equal(t, 0, countOrderIssues(t, db, orderID, models.IssueDeliveryFailed, models.IssuePending))
}

func TestTerminalizeDeliveryFailure_DayIdempotentNoDoubleNotify(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivering", nil)
	require.NoError(t, db.Exec(`UPDATE orders SET razorpay_order_id = '' WHERE id = ?`, orderID.String()).Error)
	dayID := seedCrossDay(t, db, models.PayoutHoldNone, &orderID)
	setDayStatus(t, db, dayID, models.MealPlanDayPrepared)
	order := loadFailureOrder(t, db, orderID)

	for i := 0; i < 2; i++ {
		_, err := TerminalizeDeliveryFailure(db, order, models.FailureDriverNoShow, "courier", nil)
		require.NoError(t, err)
	}
	require.Equal(t, 1, countOutbox(t, db, SubjectDeliveryFailed), "no duplicate notification on re-fire")
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayFailed))
}

// ── allDaysTerminal ──────────────────────────────────────────────────────────

func TestAllDaysTerminal_FailedDayNotTerminal(t *testing.T) {
	// A `failed` day is deliberately NON-terminal: the plan must stay open until an
	// admin resolves the day (slice B), so it can never auto-complete underneath a
	// still-disputed day.
	plan := &models.MealPlan{Days: []models.MealPlanDay{
		{Status: models.MealPlanDayDelivered},
		{Status: models.MealPlanDayFailed},
	}}
	require.False(t, allDaysTerminal(plan))

	plan.Days[1].Status = models.MealPlanDayDelivered
	require.True(t, allDaysTerminal(plan), "sanity: all-delivered plan is terminal")
}

// ── Freeze-integrity regressions (found by adversarial verify) ───────────────

// A late/replayed `delivered` event must not resurrect a frozen `failed` day back to
// delivered — that would orphan the disputed hold (stuck money) and let the plan
// silently complete. The delivered hook's guard must exclude `failed`.
func TestMarkMealPlanDayDelivered_DoesNotResurrectFailedDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	dayID := seedCrossDay(t, db, models.PayoutHoldNone, &orderID)
	setDayStatus(t, db, dayID, models.MealPlanDayPrepared)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		_, err := MarkMealPlanDayFailed(tx, orderID)
		return err
	}))
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID))

	// A stray delivered event now fires for the same order.
	MarkMealPlanDayDelivered(orderID)

	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID), "failed day not resurrected")
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID), "disputed hold not re-parked to awaiting")
	require.Equal(t, 0, countOutbox(t, db, SubjectMealPlanDayDelivered), "no delivered event emitted")
}

// A `failed` day's disputed hold must NOT be auto-cleared to release_eligible by an
// unrelated OrderIssue rejection on the shared shell order (only the admin
// day-resolution path may terminalize it). Contrast: a non-failed delivered disputed
// day IS cleared, proving the guard is specific to `failed`.
func TestReleaseDisputedHolds_FailedDayNotCleared(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldDisputed, &orderID)
	setDayStatus(t, db, dayID, models.MealPlanDayFailed)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID), "failed day stays disputed")

	// Control: a delivered disputed day on a cleared order DOES release.
	deliveredDay := seedCrossDay(t, db, models.PayoutHoldDisputed, &orderID) // status='delivered'
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return ReleaseDisputedHoldsForOrderIfCleared(tx, orderID)
	}))
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, deliveredDay), "delivered day clears")
}

// The money seam itself must refuse to pay out a `failed` day even if its hold somehow
// reached release_eligible — the definitive backstop.
func TestReleaseHold_BlocksFailedDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	orderID, _ := seedCrossOrder(t, db, models.PayoutHoldNone, "delivered", nil)
	dayID := seedCrossDay(t, db, models.PayoutHoldReleaseEligible, &orderID)
	setDayStatus(t, db, dayID, models.MealPlanDayFailed)

	err := ReleaseHold(db, aggTypeMealPlanDay, dayID)
	require.ErrorIs(t, err, ErrHoldNotEligible, "a failed day is never releasable")
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, dayID), "hold untouched")
}
