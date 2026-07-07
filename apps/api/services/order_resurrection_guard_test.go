package services

// order_resurrection_guard_test.go — #631. A late/replayed courier or 3PL "delivered" event must
// not RESURRECT an already-refunded order/day: (A) flipping the shell Order back to `delivered`
// re-enters it into the weekly statement (which selects delivered orders); (B) re-marking a
// refunded meal-plan day delivered re-parks its hold on money already returned → chef over-pay.
// These guards mirror the existing asymmetric-mirror day/group guards (#534/#393/#590).

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/models"
)

// (A) A refunded shell order whose 3PL delivery was still in flight must stay refunded when the
// courier later reports `delivered` — otherwise it folds into the chef's weekly statement.
func TestHandleShadowfaxWebhook_DeliveredDoesNotResurrectRefundedOrder(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "shadowfax", `{}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "SFX-R1", "HCSF-R1")
	require.NoError(t, db.Exec(`UPDATE orders SET status = ? WHERE id = ?`,
		string(models.OrderStatusRefunded), orderID.String()).Error)

	require.NoError(t, NewProviderService().HandleProviderWebhook("shadowfax",
		[]byte(`{"order_id":"HCSF-R1","status":"delivered"}`)))

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, string(models.OrderStatusRefunded), status,
		"a late 3PL delivered webhook must not resurrect a refunded order to delivered")
}

// (B) A refunded meal-plan day whose STATUS was not terminalized (RefundUndeliveredDays /
// RefundDeclinedDays refund via RefundDay but leave the day status non-terminal) must not be
// re-marked delivered — that would re-park its hold and make refunded money payable again.
func TestMarkMealPlanDayDelivered_DoesNotResurrectRefundedDay(t *testing.T) {
	db := setupCrossguardDB(t)
	orderID := uuid.New()
	dayID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, price, payout_hold_status, refund_txn_id)
		VALUES (?,?,?,?,?,?,?)`, dayID.String(), uuid.NewString(), orderID.String(),
		string(models.MealPlanDayConfirmed), 120.0, string(models.PayoutHoldWithheld), uuid.NewString()).Error)

	MarkMealPlanDayDelivered(orderID)

	var status, hold string
	require.NoError(t, db.Raw(`SELECT status, payout_hold_status FROM meal_plan_days WHERE id = ?`,
		dayID.String()).Row().Scan(&status, &hold))
	require.Equal(t, string(models.MealPlanDayConfirmed), status, "a refunded day is not resurrected to delivered")
	require.Equal(t, string(models.PayoutHoldWithheld), hold, "the refunded day's hold is not re-parked to awaiting")
}

// Non-regression: a normal in-flight (non-refunded) order is still marked delivered by the webhook.
func TestHandleShadowfaxWebhook_DeliveredMarksActiveOrderDelivered(t *testing.T) {
	db := setupProviderFailureDB(t)
	pid := seedProvider(t, db, "shadowfax", `{}`)
	orderID, _ := seed3PLOrderAndDelivery(t, db, pid, "SFX-OK", "HCSF-OK") // seeded status=delivering

	require.NoError(t, NewProviderService().HandleProviderWebhook("shadowfax",
		[]byte(`{"order_id":"HCSF-OK","status":"delivered"}`)))

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, string(models.OrderStatusDelivered), status, "an active order is still marked delivered")
}
