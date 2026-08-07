package services

// meal_plan_advance_reconcile_test.go — #395·3. The last-line reconcile that recovers a
// meal-plan advance whose client verify AND payment.captured webhook were both lost:
// money captured at the gateway, plan stuck awaiting_customer, chef payout never held.
// It asks the gateway whether the plan's advance order was paid and, if so, confirms via
// ConfirmMealPlanAdvance. Also covers the grace window (don't race a just-approved plan)
// and the unpaid case (leave an abandoned plan for the expiry sweep).

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedStuckAdvancePlan inserts an awaiting_customer plan with an advance order + accepted
// days, blank escrow_payment_id, and the given updated_at (to drive the grace window).
func seedStuckAdvancePlan(t *testing.T, db *gorm.DB, orderID string, dayPrices []float64, updatedAt time.Time) uuid.UUID {
	t.Helper()
	planID, chefID, custID := uuid.New(), uuid.New(), uuid.New()
	total := 32.0
	for _, p := range dayPrices {
		total += p
	}
	require.NoError(t, db.Exec(`INSERT INTO meal_plans
		(id, meal_plan_number, customer_id, chef_id, status, gateway_order_id, escrow_payment_id, subtotal, tax, total, currency, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		planID.String(), "MP-stuck", custID.String(), chefID.String(), string(models.MealPlanAwaitingCustomer),
		orderID, "", total-32, 25.6, total, "INR", updatedAt).Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		chefID.String(), uuid.NewString(), "acc_chef_rec").Error)
	for _, p := range dayPrices {
		require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price) VALUES (?,?,?,?)`,
			uuid.NewString(), planID.String(), string(models.MealPlanDayAccepted), p).Error)
	}
	return planID
}

// A stuck, captured plan idle past the grace window → reconciled: confirmed.
func TestReconcileMealPlanAdvances_ConfirmsStuckCapturedPlan(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	withCashfreeOrderPayments(t, "order_rec1", 35200, CashfreePaymentSuccess)
	planID := seedStuckAdvancePlan(t, db, "order_rec1", []float64{160, 160}, time.Now().Add(-time.Hour))

	n := reconcileMealPlanAdvances(db, time.Now())
	require.Equal(t, 1, n, "the stuck captured plan was confirmed")
	require.Equal(t, string(models.MealPlanConfirmed), planField(t, db, planID, "status"))
	require.Equal(t, "4242", planField(t, db, planID, "escrow_payment_id"))
}

// An order with no captured payment → left alone (an abandoned plan expires elsewhere).
func TestReconcileMealPlanAdvances_UnpaidLeftAlone(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	withCashfreeOrderPayments(t, "order_rec2", 35200, "") // no captured payment on the order
	planID := seedStuckAdvancePlan(t, db, "order_rec2", []float64{160, 160}, time.Now().Add(-time.Hour))

	n := reconcileMealPlanAdvances(db, time.Now())
	require.Equal(t, 0, n)
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"), "unpaid plan untouched")
}

// A freshly-approved plan (updated within the grace window) is NOT swept — the client
// verify + webhook get first crack; the reconcile must not race them.
func TestReconcileMealPlanAdvances_GraceWindow_SkipsFreshPlan(t *testing.T) {
	escrowFlag(t, true)
	db := setupConfirmAdvanceDB(t)
	withCashfreeOrderPayments(t, "order_rec3", 35200, CashfreePaymentSuccess)
	planID := seedStuckAdvancePlan(t, db, "order_rec3", []float64{160, 160}, time.Now()) // just approved

	n := reconcileMealPlanAdvances(db, time.Now())
	require.Equal(t, 0, n, "fresh plan is inside the grace window")
	require.Equal(t, string(models.MealPlanAwaitingCustomer), planField(t, db, planID, "status"))
}
