package handlers

// chef_order_cancel_gateway_fee_test.go — #885. The gateway-fee levy wired into
// CancelOrder, CancelOrderItem and RefundOrder: a Cashfree refund issued because the CHEF
// is at fault raises a best-effort ChefPenalty(kind=gateway_fee), sized off the amount
// actually sent to Cashfree — never the order's original total — and never able to affect
// the customer's refund even when the levy itself fails to record.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/services"
)

// addChefPenaltyTable creates the chef_penalties table with the exact same DDL as
// services/chef_penalty_test.go's setupPenaltyDB — copied rather than imported across
// packages.
func addChefPenaltyTable(t *testing.T, db *gorm.DB) {
	t.Helper()
	require.NoError(t, db.Exec(`CREATE TABLE chef_penalties (id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT,
		kind TEXT, status TEXT, source_key TEXT UNIQUE, order_id TEXT, reference TEXT, currency TEXT,
		basis_amount REAL, rate_percent REAL, amount REAL, lead_hours REAL, reason TEXT,
		deducted_statement_id TEXT, deducted_at DATETIME, waived_by TEXT, waived_at DATETIME,
		waive_reason TEXT, occurred_at DATETIME, created_at DATETIME, updated_at DATETIME)`).Error)
}

// gatewayFeePenaltyRowsFor returns the gateway_fee ChefPenalty rows on an order, in
// creation order.
func gatewayFeePenaltyRowsFor(t *testing.T, db *gorm.DB, orderID uuid.UUID) []struct {
	BasisAmount float64
	Amount      float64
} {
	t.Helper()
	var rows []struct {
		BasisAmount float64
		Amount      float64
	}
	require.NoError(t, db.Raw(
		`SELECT basis_amount, amount FROM chef_penalties WHERE order_id = ? AND kind = 'gateway_fee' ORDER BY created_at`,
		orderID.String()).Scan(&rows).Error)
	return rows
}

// withGatewayFeeLevyEnabled installs a platform policy with the gateway-fee levy on at a
// fixed rate (no grace, no stacking unless alsoCancelLate asks for it too), for the
// duration of the test. Persisted through services.SavePlatformPolicy so both this
// package and the services package read the same live config.
func withGatewayFeeLevyEnabled(t *testing.T, percent float64, alsoCancelLate bool) {
	t.Helper()
	p := services.DefaultPlatformPolicy()
	p.GatewayFeeLevyEnabled = true
	p.GatewayFeeLevyPercent = percent
	p.GatewayFeeLevyGraceEnabled = false
	p.GatewayFeeLevyStackWithCancelLevy = false
	if alsoCancelLate {
		p.ChefCancelPenaltyEnabled = true
		p.ChefCancelPenaltyPercent = 6
		p.ChefCancelPenaltyLeadHours = 4
		p.ChefCancelPenaltyGraceCount = 0
	} else {
		p.ChefCancelPenaltyEnabled = false
	}
	require.NoError(t, services.SavePlatformPolicy(p, nil))
	t.Cleanup(services.InvalidatePlatformPolicy)
}

// cfRefundGateway serves POST /orders/{id}/refunds with a SUCCESS refund response and
// counts how many times it was called.
func cfRefundGateway(refundCalls *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/refunds") {
			*refundCalls++
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"cf_refund_id":%d,"refund_id":"cfrfnd_test_%d","order_id":"cf-order","refund_status":"SUCCESS","refund_amount":100.00}`,
				*refundCalls, *refundCalls)))
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func regChefCancelItem(r *gin.Engine, h *ChefOrderCancelHandler) {
	r.POST("/chef/orders/:orderId/items/:itemId/cancel", h.CancelOrderItem)
}

func TestCancelOrder_Cashfree_LevyGatewayFee(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	var calls int
	withCashfreeGateway(t, cfRefundGateway(&calls))
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 250, "cf-order")
	markPreparing(t, orderID)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "customer_request"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 1, calls)

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 1)
	require.Equal(t, 250.0, rows[0].BasisAmount, "the full card-bound refund amount")
	require.Equal(t, 5.0, rows[0].Amount, "2% of 250")
}

func TestCancelOrder_Cashfree_SkipsWhenCancelLateAlsoLevied(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	withGatewayFeeLevyEnabled(t, 2, true)
	var calls int
	withCashfreeGateway(t, cfRefundGateway(&calls))
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 250, "cf-order")
	markPreparing(t, orderID) // no ScheduledFor → lead=0h, inside the 4h cancel-late window

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "customer_request"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var cancelLateCount, gatewayFeeCount int64
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM chef_penalties WHERE order_id = ? AND kind = 'cancel_late'`,
		orderID.String()).Scan(&cancelLateCount).Error)
	require.NoError(t, db.Raw(`SELECT COUNT(*) FROM chef_penalties WHERE order_id = ? AND kind = 'gateway_fee'`,
		orderID.String()).Scan(&gatewayFeeCount).Error)
	require.Equal(t, int64(1), cancelLateCount, "cancel_late levied")
	require.Equal(t, int64(0), gatewayFeeCount, "no stacking by default")
}

func TestCancelOrder_Razorpay_NoGatewayFeeLevy(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	_, _ = withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 250, "rzp_o", "pay_x")
	markPreparing(t, orderID)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "customer_request"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 0, "razorpay must not levy the Cashfree-only gateway fee")
}

func TestCancelOrderItem_Cashfree_LevyGatewayFee(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	for _, col := range []string{"cancelled_reason TEXT DEFAULT ''", "cancelled_at DATETIME", "refund_id TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE order_items ADD COLUMN `+col).Error)
	}
	withGatewayFeeLevyEnabled(t, 2, false)
	var calls int
	withCashfreeGateway(t, cfRefundGateway(&calls))
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")
	require.NoError(t, db.Exec(`UPDATE orders SET status = 'preparing', subtotal = 450, tax = 50 WHERE id = ?`, orderID.String()).Error)
	itemID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, subtotal, refund_amount) VALUES (?,?,0,225,0)`,
		itemID.String(), orderID.String()).Error)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/items/"+itemID.String()+"/cancel",
		regChefCancelItem, map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 1)
	require.Equal(t, 250.0, rows[0].BasisAmount, "basis is the line's refund amount (subtotal 225 + proportional tax 25)")
	require.Equal(t, 5.0, rows[0].Amount, "2% of 250")
}

func TestCancelOrderItem_TwoLines_LevyTwice(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	for _, col := range []string{"cancelled_reason TEXT DEFAULT ''", "cancelled_at DATETIME", "refund_id TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE order_items ADD COLUMN `+col).Error)
	}
	withGatewayFeeLevyEnabled(t, 2, false)
	var calls int
	withCashfreeGateway(t, cfRefundGateway(&calls))
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")
	require.NoError(t, db.Exec(`UPDATE orders SET status = 'preparing', subtotal = 450, tax = 50 WHERE id = ?`, orderID.String()).Error)
	item1, item2 := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, subtotal, refund_amount) VALUES (?,?,0,225,0)`,
		item1.String(), orderID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, subtotal, refund_amount) VALUES (?,?,0,225,0)`,
		item2.String(), orderID.String()).Error)

	w1 := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/items/"+item1.String()+"/cancel",
		regChefCancelItem, map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w1.Code, w1.Body.String())

	w2 := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/items/"+item2.String()+"/cancel",
		regChefCancelItem, map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 2, "each line cancel raises its own gateway_fee levy")
}

func TestRefundOrder_Cashfree_RepeatedGoodwill_LevyEach(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	var calls int
	withCashfreeGateway(t, cfRefundGateway(&calls))
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")
	require.NoError(t, db.Exec(`UPDATE orders SET status = 'delivered' WHERE id = ?`, orderID.String()).Error)

	w1 := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/refund", regGoodwillRefund,
		map[string]any{"amount": 100.0, "reason": "goodwill 1"})
	require.Equal(t, http.StatusOK, w1.Code, w1.Body.String())

	w2 := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/refund", regGoodwillRefund,
		map[string]any{"amount": 50.0, "reason": "goodwill 2"})
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 2, "each distinct partial goodwill refund levies separately")
	require.Equal(t, 100.0, rows[0].BasisAmount)
	require.Equal(t, 50.0, rows[1].BasisAmount)
}

func TestCancelOrder_GatewayFeeLevyFailure_DoesNotFailCancel(t *testing.T) {
	db := setupPayDB(t)
	// Deliberately DO NOT call addChefPenaltyTable — the levy's insert will error.
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	var calls int
	withCashfreeGateway(t, cfRefundGateway(&calls))
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 250, "cf-order")
	markPreparing(t, orderID)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "customer_request"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String(), "the cancel must succeed even though the levy insert errors")

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status)
	require.NotEmpty(t, refundID)
	require.True(t, refundedAt)
	require.Equal(t, 250.0, refundAmount, "the customer refund is unaffected by the levy failure")
}
