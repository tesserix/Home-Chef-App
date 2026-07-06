package services

// meal_plan_day_resolve_test.go — #393 slice B (meal-plan DAY resolution). The admin
// confirms a concrete fault on a delivery-FAILED day (frozen `disputed` by slice A) and
// the hybrid money policy executes (owner policy): customer-fault → chef paid + day
// terminalized; platform/chef-fault → full day refund + hold reversed + day → refunded.
// Then the plan is re-checked for completion. The state transitions are tested
// deterministically (flags off = money-safe); one flags-on test proves the RefundDay
// wallet-credit wiring.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedResolvePlan inserts a meal_plans row with a known customer + snapshotted totals.
func seedResolvePlan(t *testing.T, db *gorm.DB, customerID uuid.UUID, subtotal, tax float64) uuid.UUID {
	t.Helper()
	planID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id, status, subtotal, tax)
		VALUES (?,?,?,?,?,?,?)`,
		planID.String(), "MP-"+planID.String()[:8], customerID.String(), uuid.NewString(),
		string(models.MealPlanActive), subtotal, tax).Error)
	return planID
}

// seedResolveDay inserts a meal_plan_days row under a plan with a chosen status/hold/transfer.
func seedResolveDay(t *testing.T, db *gorm.DB, planID uuid.UUID, status models.MealPlanDayStatus, hold models.PayoutHoldStatus, transferID string, price float64) uuid.UUID {
	t.Helper()
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days
		(id, meal_plan_id, order_id, status, payout_transfer_id, price, payout_hold_status, date)
		VALUES (?,?,?,?,?,?,?,?)`,
		dayID.String(), planID.String(), uuid.NewString(), string(status), transferID, price,
		string(hold), nil).Error)
	return dayID
}

func loadPlanStatus(t *testing.T, db *gorm.DB, planID uuid.UUID) models.MealPlanStatus {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM meal_plans WHERE id = ?`, planID.String()).Scan(&s).Error)
	return models.MealPlanStatus(s)
}

func dayRefundTxnSet(t *testing.T, db *gorm.DB, dayID uuid.UUID) bool {
	t.Helper()
	var n int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM meal_plan_days WHERE id = ? AND refund_txn_id IS NOT NULL`, dayID.String()).Scan(&n).Error)
	return n == 1
}

// resolveDay is a thin loader mirroring the handler: pass the DB-current plan+day struct.
func resolveDay(t *testing.T, db *gorm.DB, planID, dayID uuid.UUID, subtotal, tax, total, price float64, cust uuid.UUID) (*models.MealPlan, *models.MealPlanDay) {
	t.Helper()
	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: subtotal, Tax: tax, Total: total,
		Days: []models.MealPlanDay{{ID: dayID, Price: price}}}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: price, Status: models.MealPlanDayFailed}
	return plan, day
}

// ── customer-fault ───────────────────────────────────────────────────────────

func TestResolveMealPlanDayFailure_CustomerFault_PaysAndCompletes(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayFailed, models.PayoutHoldDisputed, "", 120)
	plan, day := resolveDay(t, db, planID, dayID, 120, 12, 150, 120, cust)

	require.NoError(t, ResolveMealPlanDayFailure(db, plan, day, models.FaultCustomer, uuid.New()))

	require.Equal(t, models.MealPlanDayDelivered, loadDayStatus(t, db, dayID), "chef paid → paid-terminal")
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, dayID), "hold enters the pay queue")
	require.Equal(t, models.MealPlanCompleted, loadPlanStatus(t, db, planID), "last day terminal → plan completes")
	require.Equal(t, 1, countOutbox(t, db, SubjectHoldReleaseEligible))
	require.False(t, dayRefundTxnSet(t, db, dayID), "no refund on customer fault")
}

func TestResolveMealPlanDayFailure_CustomerFault_PlanStaysOpenWithPendingDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 240, 24)
	failedDay := seedResolveDay(t, db, planID, models.MealPlanDayFailed, models.PayoutHoldDisputed, "", 120)
	seedResolveDay(t, db, planID, models.MealPlanDayConfirmed, models.PayoutHoldNone, "", 120) // still pending
	plan, day := resolveDay(t, db, planID, failedDay, 240, 24, 300, 120, cust)

	require.NoError(t, ResolveMealPlanDayFailure(db, plan, day, models.FaultCustomer, uuid.New()))

	require.Equal(t, models.MealPlanDayDelivered, loadDayStatus(t, db, failedDay))
	require.Equal(t, models.MealPlanActive, loadPlanStatus(t, db, planID), "plan stays open — a day is still pending")
}

// ── platform / chef fault ────────────────────────────────────────────────────

func TestResolveMealPlanDayFailure_PlatformFault_RefundsAndCompletes_FlagsOff(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayFailed, models.PayoutHoldDisputed, "", 120)
	plan, day := resolveDay(t, db, planID, dayID, 120, 12, 150, 120, cust)

	require.NoError(t, ResolveMealPlanDayFailure(db, plan, day, models.FaultPlatform, uuid.New()))

	require.Equal(t, models.MealPlanDayRefunded, loadDayStatus(t, db, dayID))
	// Flags off → RefundDay no-ops (no money); the resolver still drives the hold to a
	// terminal non-payable state so the disputed hold can never be cleared/paid later.
	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID))
	require.Equal(t, models.MealPlanCompleted, loadPlanStatus(t, db, planID))
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayRefunded))
	require.Equal(t, 0.0, walletBalance(t, db, cust), "no wallet credit when escrow is off")
}

func TestResolveMealPlanDayFailure_ChefFault_FlagsOn_CreditsWalletAndWithholds(t *testing.T) {
	escrowFlag(t, true)
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	// No held transfer (a failed day was never delivered) → RefundDay credits without a
	// gateway ReverseTransfer call.
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayFailed, models.PayoutHoldDisputed, "", 120)
	plan, day := resolveDay(t, db, planID, dayID, 120, 12, 150, 120, cust)

	require.NoError(t, ResolveMealPlanDayFailure(db, plan, day, models.FaultChef, uuid.New()))

	require.Equal(t, models.MealPlanDayRefunded, loadDayStatus(t, db, dayID))
	require.Equal(t, models.PayoutHoldWithheld, loadDayHold(t, db, dayID))
	require.True(t, dayRefundTxnSet(t, db, dayID), "refund txn stamped")
	// perDayGross = price + proportional food GST + per-day delivery = 120 + 12 + 18 = 150.
	require.Equal(t, 150.0, walletBalance(t, db, cust), "full day refund to wallet")
}

// ── guards ───────────────────────────────────────────────────────────────────

func TestResolveMealPlanDayFailure_Idempotent(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayFailed, models.PayoutHoldDisputed, "", 120)
	plan, day := resolveDay(t, db, planID, dayID, 120, 12, 150, 120, cust)

	require.NoError(t, ResolveMealPlanDayFailure(db, plan, day, models.FaultPlatform, uuid.New()))
	// Second call loses the claim (status no longer failed).
	err := ResolveMealPlanDayFailure(db, plan, day, models.FaultCustomer, uuid.New())
	require.ErrorIs(t, err, ErrIssueAlreadyHandled)
	require.Equal(t, models.MealPlanDayRefunded, loadDayStatus(t, db, dayID), "first resolution wins")
	require.Equal(t, 1, countOutbox(t, db, SubjectMealPlanDayRefunded), "no duplicate refund event")
}

func TestResolveMealPlanDayFailure_NotAFailedDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayDelivered, models.PayoutHoldAwaitingConfirmation, "", 120)
	plan := &models.MealPlan{ID: planID, CustomerID: cust}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Status: models.MealPlanDayDelivered}

	err := ResolveMealPlanDayFailure(db, plan, day, models.FaultPlatform, uuid.New())
	require.ErrorIs(t, err, ErrNotDeliveryFailure)
	require.Equal(t, models.MealPlanDayDelivered, loadDayStatus(t, db, dayID), "untouched")
}

// A refunded day must never pay the chef even if its hold somehow sat at
// release_eligible — the status-based release backstop blocks it (mirrors a refunded
// order via orderRefundBlocks). Regression for the Slice-B verify finding.
func TestReleaseHold_BlocksRefundedDay(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayRefunded, models.PayoutHoldReleaseEligible, "", 120)

	err := ReleaseHold(db, aggTypeMealPlanDay, dayID)
	require.ErrorIs(t, err, ErrHoldNotEligible, "a refunded day is never releasable")
	require.Equal(t, models.PayoutHoldReleaseEligible, loadDayHold(t, db, dayID), "hold untouched")
}

func TestResolveMealPlanDayFailure_Ambiguous(t *testing.T) {
	flagsOff(t)
	db := setupCrossguardDB(t)
	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := seedResolveDay(t, db, planID, models.MealPlanDayFailed, models.PayoutHoldDisputed, "", 120)
	plan, day := resolveDay(t, db, planID, dayID, 120, 12, 150, 120, cust)

	err := ResolveMealPlanDayFailure(db, plan, day, models.FaultAmbiguous, uuid.New())
	require.ErrorIs(t, err, ErrAmbiguousFault)
	require.Equal(t, models.MealPlanDayFailed, loadDayStatus(t, db, dayID), "still frozen — admin must decide")
	require.Equal(t, models.PayoutHoldDisputed, loadDayHold(t, db, dayID))
}
