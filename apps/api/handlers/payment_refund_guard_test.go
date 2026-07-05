package handlers

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// #394: a meal-plan day spawns a regular Order (reachable by the generic refund
// endpoint) but refunds through the escrow-aware RefundDay (key
// mealplan-refund:<dayID>). The generic endpoint keys on refund:<orderID> and
// bypasses Order.RefundAmount, so the same day could be refunded once via each
// keyspace — customer over-credited — and the generic path never reverses the
// held DIRECT transfer, so the chef keeps the money too. The generic endpoint
// must refuse these orders (422) and point at the meal-plan refund flow.

func linkMealPlanDay(t *testing.T, db *gorm.DB, orderID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id) VALUES (?, ?)`,
		uuid.New().String(), orderID.String()).Error)
}

func linkGroupOrder(t *testing.T, db *gorm.DB, orderID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, order_id) VALUES (?, ?)`,
		uuid.New().String(), orderID.String()).Error)
}

func TestInitiateRefund_MealPlanDayOrder_422(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	// A completed day-order with a gateway payment id — would otherwise be a
	// perfectly refundable order; the guard must fire before any refund happens.
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")
	linkMealPlanDay(t, db, orderID)

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "day not needed", "toWallet": true})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 for meal-plan day order, got %d (%s)", w.Code, w.Body.String())
	}

	// And nothing was refunded on the order.
	var refunded float64
	require.NoError(t, db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&refunded).Error)
	if refunded != 0 {
		t.Fatalf("guard must not partially refund; refund_amount=%v", refunded)
	}
}

func TestInitiateRefund_GroupOrder_422(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")
	linkGroupOrder(t, db, orderID)

	admin := payUser(t, db, "admin")
	w := callPay(admin, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "cancel group", "toWallet": true})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422 for group order, got %d (%s)", w.Code, w.Body.String())
	}
}

// A plain order (not linked from any typed flow) must pass the guard unchanged.
// It stops at the missing-gateway-payment 400, proving the guard let it through.
func TestInitiateRefund_PlainOrder_PassesGuard(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "") // no payment id
	admin := payUser(t, db, "admin")

	w := callPay(admin, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "ops"})
	if w.Code == http.StatusUnprocessableEntity {
		t.Fatalf("plain order must not be blocked by the typed-flow guard, got 422 (%s)", w.Body.String())
	}
}
