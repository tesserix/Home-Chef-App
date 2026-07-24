package handlers

// meal_plan_resume_test.go — abandoned-payment re-entry. Under capture-at-approval the
// customer's Approve mints a Razorpay advance order and hands the app checkout. If they
// back out of Razorpay without paying, re-tapping Approve used to 409 ("no longer
// awaiting your approval") because the mint guard sees razorpay_order_id already set —
// stranding a minted-but-unpaid advance with no way back into checkout. Re-approve must
// instead RESUME: return the SAME order so the app re-launches checkout. A plan whose
// advance is already captured (escrow_payment_id set) is NOT resumable — that's a
// genuine conflict, never a re-checkout of a paid plan.

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

func setEscrow(t *testing.T, on bool) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{MealPlanEscrowEnabled: on}
	t.Cleanup(func() { config.AppConfig = prev })
}

// Re-approving a plan that already minted an advance order but was NOT paid resumes the
// SAME order (200 + razorpayOrderId), does not 409, and does not re-mint or confirm.
func TestApproveMealPlan_ResumesUnpaidMintedOrder(t *testing.T) {
	setEscrow(t, true)
	db := setupOrchestrationDB(t)
	chefID, _ := seedOrchChef(t, db)
	cust := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanAwaitingCustomer, cust, chefID)
	seedOrchDay(t, db, planID, models.MealPlanDayAccepted, 160)
	seedOrchDay(t, db, planID, models.MealPlanDayAccepted, 160)
	// A prior approve minted the advance order; the customer backed out unpaid.
	require.NoError(t, db.Exec(`UPDATE meal_plans SET razorpay_order_id = ?, escrow_payment_id = '', total = 352 WHERE id = ?`,
		"order_existing", planID.String()).Error)

	w := finalizeReq(t, cust, planID, "approve")

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, "order_existing", resp["razorpayOrderId"], "resumed the SAME order — no 409, no re-mint")
	require.NotNil(t, resp["mealPlan"], "the plan is returned so the app can read the checkout amount")
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"), "still awaiting payment, not confirmed")
	require.Equal(t, "order_existing", planField(t, db, planID, "razorpay_order_id"), "order id unchanged")
}

// A plan whose advance is already captured (escrow_payment_id set) is NOT resumable —
// re-approve is a genuine conflict, so a paid plan can't be re-checked-out.
func TestApproveMealPlan_PaidPlan_NotResumed(t *testing.T) {
	setEscrow(t, true)
	db := setupOrchestrationDB(t)
	chefID, _ := seedOrchChef(t, db)
	cust := uuid.New()
	planID := seedOrchPlan(t, db, models.MealPlanAwaitingCustomer, cust, chefID)
	seedOrchDay(t, db, planID, models.MealPlanDayAccepted, 160)
	require.NoError(t, db.Exec(`UPDATE meal_plans SET razorpay_order_id = ?, escrow_payment_id = ? WHERE id = ?`,
		"order_paid", "pay_captured", planID.String()).Error)

	w := finalizeReq(t, cust, planID, "approve")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
}
