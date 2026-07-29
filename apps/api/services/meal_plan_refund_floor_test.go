package services

// meal_plan_refund_floor_test.go — server-side floor enforcement (#834).
//
// The floor is the whole point of the tier model, and it is enforced ONLY here: both web and
// mobile hit the same endpoint, so a client-side constraint constrains nothing. These tests
// pin that a chef cannot go below the day's floor, and — the subtler property — that the floor
// is the one PINNED WHEN THE REQUEST WAS RAISED, not one recomputed off the current clock. A
// chef who sits on a decision until the meal is imminent must not thereby earn a cheaper band.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedFloorDay inserts a pending_chef day carrying a pinned floor.
func seedFloorDay(t *testing.T, db *gorm.DB, custID uuid.UUID, floor int) (planID, dayID uuid.UUID) {
	t.Helper()
	planID, dayID = uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, customer_id, meal_plan_number, escrow_payment_id, subtotal, tax, total)
		VALUES (?,?,?,?,?,?,?)`, planID.String(), custID.String(), "MP-FLOOR", "pay_test", 320.0, 32.0, 372.0).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, commission_rate, payout_hold_status, refund_stage, refund_floor_percent)
		VALUES (?,?,?,?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDaySkipRequested), 160.0, 0.15,
		string(models.PayoutHoldDisputed), string(models.MPRefundPendingChef), floor).Error)
	return
}

func dayRefundPercent(t *testing.T, db *gorm.DB, id uuid.UUID) int {
	t.Helper()
	var p *int
	require.NoError(t, db.Raw(`SELECT refund_percent FROM meal_plan_days WHERE id = ?`, id.String()).Scan(&p).Error)
	if p == nil {
		return -1
	}
	return *p
}

// Below the floor is REJECTED — no state change, no money.
func TestChefDecide_BelowFloor_Rejected(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedFloorDay(t, db, u, 75)

	err := ChefDecideMealPlanRefund(db, dayID, 50, false)
	require.ErrorIs(t, err, ErrRefundBelowFloor)
	require.Contains(t, err.Error(), "75%", "the error must carry the floor so a client can correct it")

	_, stage, _, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MPRefundPendingChef), stage, "a rejected decision leaves the day awaiting the chef")
	require.Equal(t, 0.0, v2WalletBalance(t, db, u))
}

// Exactly AT the floor is allowed — the floor is inclusive.
func TestChefDecide_AtFloor_Accepted(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedFloorDay(t, db, u, 75)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, 75, false))
	_, stage, choice, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MPRefundPendingCustomer), stage)
	require.Equal(t, 75, dayRefundPercent(t, db, dayID))
	require.Equal(t, string(models.RefundProportionPartial), choice,
		"75% has no legacy enum value, so the coarse label is 'partial'")

	// The customer then picks wallet: 75% of the 186 gross = 139.50.
	require.NoError(t, CustomerChooseMealPlanRefundMedium(db, dayID, u, models.RefundDestinationWallet))
	require.Equal(t, 139.5, v2WalletBalance(t, db, u))
}

// Anything between the floor and 100 is allowed — the point of replacing the enum.
func TestChefDecide_AnyPercentAboveFloor(t *testing.T) {
	for _, pct := range []int{76, 80, 90, 99, 100} {
		v2EscrowOn(t)
		db := setupV2RefundDB(t)
		u := uuid.New()
		_, dayID := seedFloorDay(t, db, u, 75)
		require.NoError(t, ChefDecideMealPlanRefund(db, dayID, pct, false), "percent %d", pct)
		require.Equal(t, pct, dayRefundPercent(t, db, dayID))
	}
}

// A zero floor (the sub-2h band) still permits refusing outright — and that resolves the day.
func TestChefDecide_ZeroFloor_AllowsNone(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedFloorDay(t, db, u, 0)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, 0, false))
	status, stage, _, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDaySkipped), status)
	require.Equal(t, string(models.MPRefundResolved), stage)
	require.Equal(t, 0.0, v2WalletBalance(t, db, u))
}

// Out of range is a validation error, distinct from a floor violation (400 vs 422 upstream).
func TestChefDecide_OutOfRange(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	_, dayID := seedFloorDay(t, db, uuid.New(), 0)
	require.ErrorIs(t, ChefDecideMealPlanRefund(db, dayID, 101, false), ErrInvalidRefundChoice)
	require.ErrorIs(t, ChefDecideMealPlanRefund(db, dayID, -1, false), ErrInvalidRefundChoice)
}

// THE point of pinning: the floor travels with the day, so a chef deciding late is still bound
// by the band that applied when the customer asked. A recomputed-at-decision-time floor would
// read 0 here and let the chef refund nothing.
func TestChefDecide_FloorIsPinnedNotRecomputed(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	// The request was raised >12h out but under the alternative policy — floor 75 — and the
	// chef only acts now, when the meal is minutes away (a live tier lookup would say 0).
	_, dayID := seedFloorDay(t, db, u, 75)
	withTiers(t, DefaultMealPlanRefundTiers())

	require.ErrorIs(t, ChefDecideMealPlanRefund(db, dayID, 0, false), ErrRefundBelowFloor,
		"a delayed decision must be priced against the pinned floor, not the current clock")
}

// A pre-v3 day carries no pinned floor; it keeps the old freedom rather than being
// retroactively bound by a rule that did not exist when the request was raised.
func TestChefDecide_LegacyDayNoPinnedFloor(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedV2FlowRow(t, db, u) // no refund_floor_percent
	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, 0, false))
	status, _, _, _ := v2DayRow(t, db, dayID)
	require.Equal(t, string(models.MealPlanDaySkipped), status)
}

// RouteMealPlanDayToChef pins the floor; AgreeMealPlanDayRefundAuto records the agreed amount.
func TestRouteAndAgree_PinTheFloor(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error { return RouteMealPlanDayToChef(tx, day, 50) }))
	var floor int
	require.NoError(t, db.Raw(`SELECT refund_floor_percent FROM meal_plan_days WHERE id = ?`, day.ID.String()).Scan(&floor).Error)
	require.Equal(t, 50, floor)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return AgreeMealPlanDayRefundAuto(tx, plan, day, 100)
	}))
	require.Equal(t, 100, dayRefundPercent(t, db, day.ID))
	_, stage, _, _ := v2DayRow(t, db, day.ID)
	require.Equal(t, string(models.MPRefundPendingCustomer), stage)
}

// Double-submit: a second decision on a day that has already moved on is rejected, so a
// retried request cannot re-price a refund the customer has already been offered.
func TestChefDecide_DoubleSubmit(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	_, dayID := seedFloorDay(t, db, u, 75)

	require.NoError(t, ChefDecideMealPlanRefund(db, dayID, 100, false))
	require.ErrorIs(t, ChefDecideMealPlanRefund(db, dayID, 75, false), ErrRefundStageMismatch)
	require.Equal(t, 100, dayRefundPercent(t, db, dayID), "the first decision stands")
}
