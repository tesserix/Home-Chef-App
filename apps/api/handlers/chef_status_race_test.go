package handlers

// chef_status_race_test.go — #534 Fix 3. The chef order-status UPDATE was
// optimistic-unsafe: it persisted `WHERE id = ?` only, so a concurrent
// cancel/reject/refund landing in the load→persist window was blindly overwritten —
// a chef's in-flight `delivered` would resurrect a just-cancelled order and fire the
// delivered payout side-effects (MarkGroupOrderDelivered / MarkMealPlanDayDelivered).
// The fix guards the UPDATE on the read status (priorStatus) and 409s when it raced.

import (
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func TestUpdateOrderStatus_RacedToCancelled_409(t *testing.T) {
	db, userID, chefID := setupChefOrderDB(t)
	orderID, custID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, fulfillment_type,
		 subtotal, total, currency, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		orderID.String(), "ORD-"+orderID.String()[:8], custID.String(), chefID.String(),
		"delivering", "completed", "chef_delivery", 100.0, 100.0, "INR", time.Now(), time.Now()).Error)

	// Simulate a concurrent host/customer cancel landing between the handler's load
	// and its persist UPDATE: a one-shot Before-update hook flips the row to
	// cancelled via a fresh session just before the handler writes.
	injected := false
	require.NoError(t, db.Callback().Update().Before("gorm:update").Register("inject_cancel", func(tx *gorm.DB) {
		if injected {
			return
		}
		injected = true
		tx.Session(&gorm.Session{NewDB: true}).Exec(
			`UPDATE orders SET status = ? WHERE id = ?`, string(models.OrderStatusCancelled), orderID.String())
	}))

	w := postStatus(t, userID, orderID, "delivered")
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	// The optimistic guard aborted the persist tx: the order is NOT resurrected to
	// delivered. (The injected cancel shares the single sqlite connection with the
	// handler tx, so on the 409 rollback the row reverts to its pre-tx status rather
	// than staying cancelled — a harness artifact; the money-safe invariant is
	// simply that a raced update never lands `delivered`.)
	var status string
	require.NoError(t, db.Raw(`SELECT status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.NotEqual(t, string(models.OrderStatusDelivered), status,
		"a raced order must not be resurrected to delivered")

	var deliveredEvents int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, "orders.delivered").
		Scan(&deliveredEvents).Error)
	require.Equal(t, int64(0), deliveredEvents, "no delivered side-effects staged when the update raced")
}
