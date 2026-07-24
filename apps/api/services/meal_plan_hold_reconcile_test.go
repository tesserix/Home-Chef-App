package services

// meal_plan_hold_reconcile_test.go — #395·3 GAP 2 backstop. Since ConfirmMealPlanAdvance
// now holds payouts OUTSIDE the confirm tx, a crash between the two leaves a confirmed,
// captured plan whose days aren't held. This sweep completes the hold; it must be
// idempotent (a re-run over already-held days creates no new transfers).

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// transfersOnlyStub serves POST /transfers (the hold) — the reconcile does no gateway
// reads, only HoldChefPayouts writes.
func transfersOnlyStub(t *testing.T, creates *int32) {
	t.Helper()
	withRazorpayTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/transfers") {
			n := atomic.AddInt32(creates, 1)
			_ = json.NewEncoder(w).Encode(map[string]any{"id": fmt.Sprintf("trf_%d", n), "on_hold": true})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
}

// seedConfirmedPlan inserts a confirmed, captured plan (+chef) with N confirmed days
// whose payout_transfer_id is `transferID` ("" = unheld).
func seedConfirmedPlan(t *testing.T, db *gorm.DB, nDays int, transferID string) uuid.UUID {
	t.Helper()
	planID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans
		(id, meal_plan_number, customer_id, chef_id, status, razorpay_order_id, escrow_payment_id, total, currency)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		planID.String(), "MP-held", uuid.NewString(), chefID.String(), string(models.MealPlanConfirmed),
		"order_h", "pay_h", 352.0, "INR").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, razorpay_account_id) VALUES (?,?,?)`,
		chefID.String(), uuid.NewString(), "acc_chef_h").Error)
	for i := 0; i < nDays; i++ {
		require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, payout_transfer_id) VALUES (?,?,?,?,?)`,
			uuid.NewString(), planID.String(), string(models.MealPlanDayConfirmed), 160.0, transferID).Error)
	}
	return planID
}

// A confirmed, captured plan with unheld days → the sweep completes one hold per day.
func TestReconcileUnheldMealPlans_HoldsConfirmedCapturedPlan(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	transfersOnlyStub(t, &creates)
	seedConfirmedPlan(t, db, 2, "") // 2 unheld confirmed days

	n := reconcileUnheldMealPlans(db, time.Now())
	require.Equal(t, 1, n, "one plan (re)held")
	require.Equal(t, int32(2), atomic.LoadInt32(&creates), "one transfer per unheld payable day")
}

// A plan whose days are already held → not selected, no new transfers.
func TestReconcileUnheldMealPlans_AlreadyHeld_NoOp(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	transfersOnlyStub(t, &creates)
	seedConfirmedPlan(t, db, 2, "trf_existing") // days already carry a transfer id

	n := reconcileUnheldMealPlans(db, time.Now())
	require.Equal(t, 0, n, "nothing to hold")
	require.Equal(t, int32(0), atomic.LoadInt32(&creates))
}

// Escrow OFF → no-op.
func TestReconcileUnheldMealPlans_EscrowOff_NoOp(t *testing.T) {
	escrowFlag(t, false)
	db := setupConfirmAdvanceDB(t)
	var creates int32
	transfersOnlyStub(t, &creates)
	seedConfirmedPlan(t, db, 2, "")

	n := reconcileUnheldMealPlans(db, time.Now())
	require.Equal(t, 0, n)
	require.Equal(t, int32(0), atomic.LoadInt32(&creates))
}
