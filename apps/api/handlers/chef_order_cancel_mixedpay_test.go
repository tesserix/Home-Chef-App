package handlers

// chef_order_cancel_mixedpay_test.go — money-critical bugfix. A live E2E test found that
// ChefOrderCancelHandler.CancelOrder refunded the FULL order Total to Razorpay even when
// the order was part-funded by wallet/loyalty checkout credits: the gateway only ever
// CAPTURED (Total − WalletApplied − LoyaltyApplied), so the refund call failed with
// "refund amount greater than amount captured", the cancel deferred it, and the retry
// cron kept re-sending the SAME over-large amount forever — the customer was never
// refunded (confirmed live: order 64c560c1 tried to refund 48191 paise, only 32766 was
// captured). These tests pin the fix: the wallet + loyalty slices are credited back to
// the customer's wallet instantly, and only the CARD slice is sent to the gateway.

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
)

// railRefundedColumnsOf reads back the wallet_refunded/loyalty_refunded bookkeeping the
// split writes — the ledger record of what each credit rail has been given back.
func railRefundedColumnsOf(t *testing.T, orderID uuid.UUID) (walletRefunded, loyaltyRefunded float64) {
	t.Helper()
	var r struct {
		WalletRefunded  float64
		LoyaltyRefunded float64
	}
	require.NoError(t, database.DB.Raw(
		`SELECT wallet_refunded, loyalty_refunded FROM orders WHERE id = ?`, orderID.String()).Scan(&r).Error)
	return r.WalletRefunded, r.LoyaltyRefunded
}

// TestCancelOrder_MixedPayment_GatewayGetsCapturedOnly is the exact live-bug scenario:
// ₹481.91 total, ₹150.40 wallet + ₹3.85 loyalty applied at checkout, so Razorpay only
// captured ₹327.66 (32766 paise). The gateway must be asked to refund exactly that —
// never the full ₹481.91 — and the wallet+loyalty slices must land back in the wallet.
func TestCancelOrder_MixedPayment_GatewayGetsCapturedOnly(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	addWalletTables(t, db)
	pinSingleConn(t, db)
	gw := withCashfreeRefundGateway(t, http.StatusOK)

	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 481.91, "cf_o")
	markPreparing(t, orderID)
	require.NoError(t, db.Exec(`UPDATE orders SET wallet_applied = 150.40, loyalty_applied = 3.85 WHERE id = ?`,
		orderID.String()).Error)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "customer_request"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 1, gw.calls, "the gateway must be called exactly once")
	require.Equal(t, 32766, gw.amountPaise,
		"the gateway must only be asked to refund what it actually captured (₹327.66 = 32766 paise), not the full ₹481.91 total (48191 paise) — that mismatch is the live bug")

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status)
	require.Equal(t, gw.key, refundID, "the real gateway id — the gateway call succeeded with the correct amount, no deferred sentinel")
	require.True(t, refundedAt)
	require.Equal(t, 481.91, refundAmount, "the reservation still records the FULL total owed across every rail")

	require.InDelta(t, 154.25, walletBalance(t, db, cust), 0.001,
		"the wallet slice (₹150.40) + the loyalty slice (₹3.85), both credited instantly")

	walletRefunded, loyaltyRefunded := railRefundedColumnsOf(t, orderID)
	require.InDelta(t, 150.40, walletRefunded, 0.001)
	require.InDelta(t, 3.85, loyaltyRefunded, 0.001)
}

// TestCancelOrder_PureCard_Unchanged pins the non-regression: a pure-card order (no
// wallet/loyalty applied) still refunds its full total to the gateway exactly as before.
func TestCancelOrder_PureCard_Unchanged(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	gw := withCashfreeRefundGateway(t, http.StatusOK)

	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 200, "cf_o")
	markPreparing(t, orderID)
	// wallet_applied / loyalty_applied stay at their column default of 0.

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "equipment_failure"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 1, gw.calls)
	require.Equal(t, 20000, gw.amountPaise, "a pure-card order still refunds the full total to the gateway — unchanged behavior")

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status)
	require.Equal(t, gw.key, refundID)
	require.True(t, refundedAt)
	require.Equal(t, 200.0, refundAmount)

	walletRefunded, loyaltyRefunded := railRefundedColumnsOf(t, orderID)
	require.Zero(t, walletRefunded, "no wallet credit on a pure-card cancel")
	require.Zero(t, loyaltyRefunded, "no loyalty credit on a pure-card cancel")
}

// TestCancelOrder_FullyCreditFunded_NoGateway covers the edge where wallet + loyalty
// cover the ENTIRE total: the card slice is zero, so there is nothing left to refund
// through the gateway at all — no gateway call, no sentinel, order still cancels cleanly.
func TestCancelOrder_FullyCreditFunded_NoGateway(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	addWalletTables(t, db)
	pinSingleConn(t, db)
	gw := withCashfreeRefundGateway(t, http.StatusOK)

	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 100, "cf_o")
	markPreparing(t, orderID)
	require.NoError(t, db.Exec(`UPDATE orders SET wallet_applied = 80, loyalty_applied = 20 WHERE id = ?`,
		orderID.String()).Error)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 0, gw.calls, "no card slice remains — the gateway must not be called at all")

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status)
	require.Empty(t, refundID, "no gateway call means no real id and no deferred sentinel")
	require.True(t, refundedAt)
	require.Equal(t, 100.0, refundAmount)

	require.InDelta(t, 100.0, walletBalance(t, db, cust), 0.001, "wallet + loyalty together made the customer fully whole")

	walletRefunded, loyaltyRefunded := railRefundedColumnsOf(t, orderID)
	require.InDelta(t, 80.0, walletRefunded, 0.001)
	require.InDelta(t, 20.0, loyaltyRefunded, 0.001)
}
