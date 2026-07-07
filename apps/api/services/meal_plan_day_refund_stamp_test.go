package services

// meal_plan_day_refund_stamp_test.go — #629. RefundDay refunds a meal-plan day on the
// mealplan-refund:<dayID> keyspace but the day's spawned shell Order otherwise still reads
// paid/pending/RefundAmount=0 — so a customer report-issue on that shell order lets
// RefundIssueToWallet pay them a SECOND time (disjoint issue:<id> key). RefundDay must stamp the
// linked shell Order refunded so ReportIssue refuses it and RemainingRefundable reads 0.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

func TestRefundDay_StampsLinkedShellOrderRefunded(t *testing.T) {
	escrowFlag(t, true)
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)

	cust := uuid.New()
	orderID := uuid.New()
	// The spawned per-day shell order: paid, still pending (undelivered), nothing refunded yet.
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, payment_status, total, refund_amount)
		VALUES (?,?,?,?,?,?)`, orderID.String(), cust.String(), "pending", "completed", 150.0, 0.0).Error)

	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, price, payout_hold_status)
		VALUES (?,?,?,?,?,?)`, dayID.String(), planID.String(), orderID.String(),
		string(models.MealPlanDayConfirmed), 120.0, string(models.PayoutHoldNone)).Error)

	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 120, Tax: 12, Total: 150,
		Days: []models.MealPlanDay{{ID: dayID, Price: 120}}}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 120, OrderID: &orderID,
		Status: models.MealPlanDayConfirmed}

	require.NoError(t, RefundDay(db, plan, day, "customer cancelled plan"))

	// The day was refunded to the wallet: perDayGross = 120 food + 12 GST + 18 delivery = 150.
	require.Equal(t, 150.0, walletBalance(t, db, cust))

	// And the linked shell Order is now stamped refunded, so a later report-issue on it can't
	// double-pay via the disjoint issue:<id> keyspace (#629): ReportIssue blocks on
	// status==refunded, and RemainingRefundable = Total(150) − refund_amount(150) = 0.
	var row struct {
		Status       string
		RefundAmount float64
		RefundedAt   *time.Time
	}
	require.NoError(t, db.Raw(`SELECT status, refund_amount, refunded_at FROM orders WHERE id = ?`,
		orderID.String()).Scan(&row).Error)
	require.Equal(t, string(models.OrderStatusRefunded), row.Status, "shell order marked refunded")
	require.Equal(t, 150.0, row.RefundAmount, "refund_amount = perDayGross → RemainingRefundable reads 0")
	require.NotNil(t, row.RefundedAt, "refunded_at stamped")
}

// A day with no linked shell order (OrderID nil — legacy/edge) must still refund cleanly without
// touching any order row.
func TestRefundDay_NilOrderID_NoOrderStamp(t *testing.T) {
	escrowFlag(t, true)
	t.Cleanup(func() { SetRazorpayClient(nil) })
	SetRazorpayClient(NewRazorpayTestClient("", "key", "secret", "whsec"))
	db := setupCrossguardDB(t)

	cust := uuid.New()
	planID := seedResolvePlan(t, db, cust, 120, 12)
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, payout_hold_status)
		VALUES (?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDayConfirmed), 120.0,
		string(models.PayoutHoldNone)).Error)
	plan := &models.MealPlan{ID: planID, CustomerID: cust, Subtotal: 120, Tax: 12, Total: 150,
		Days: []models.MealPlanDay{{ID: dayID, Price: 120}}}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 120, OrderID: nil,
		Status: models.MealPlanDayConfirmed}

	require.NoError(t, RefundDay(db, plan, day, "no shell order"))
	require.Equal(t, 150.0, walletBalance(t, db, cust), "day still refunded to wallet")
}
