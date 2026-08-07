package handlers

// payment_gateway_fee_test.go — #885. The gateway-fee levy wired into InitiateRefund's
// Cashfree branch: chef-initiated only (decision 1 — an admin-initiated refund is
// ambiguous fault and must never levy), Cashfree only, and sized off the amount actually
// sent to the gateway — which the wallet-at-checkout capping logic may have already
// lowered below the raw requested amount.
//
// Reuses addChefPenaltyTable / gatewayFeePenaltyRowsFor / withGatewayFeeLevyEnabled from
// chef_order_cancel_gateway_fee_test.go, withCashfreeRefundGateway from
// chef_order_cancel_refund_test.go, and cfPayOrder / callPay / regRefund (same package).

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitiateRefund_ChefInitiated_Cashfree_LevyGatewayFee(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	withCashfreeRefundGateway(t, http.StatusOK)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial goodwill", "amount": 100.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 1)
	require.Equal(t, 100.0, rows[0].BasisAmount, "the amount actually sent to Cashfree")
	require.Equal(t, 2.0, rows[0].Amount, "2% of 100")
}

func TestInitiateRefund_AdminInitiated_Cashfree_NoLevy(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	withCashfreeRefundGateway(t, http.StatusOK)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")
	admin := payUser(t, db, "admin")

	w := callPay(admin, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "ops refund", "amount": 100.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 0, "admin-initiated refunds never levy — decision 1's safer default")
}

func TestInitiateRefund_ChefInitiated_ToWallet_NoLevy(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	addWalletTables(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	withCashfreeRefundGateway(t, http.StatusOK)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "to wallet", "amount": 100.0, "toWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 0, "a wallet credit never reverses the gateway charge — no gateway fee incurred")
}

// A wallet-at-checkout split reduces the amount actually sent to Cashfree below the
// requested refund amount; the levy's basis must be the CAPPED gateway-bound amount.
func TestInitiateRefund_ChefInitiated_WalletAtCheckoutSplit_BasisIsGatewayPortionOnly(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	addWalletTables(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	withCashfreeRefundGateway(t, http.StatusOK)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")
	// wallet_applied=100 → the gateway only ever captured 400; a 500 full-refund request
	// caps the gateway leg at 400 and returns 100 as wallet credit.
	require.NoError(t, db.Exec(`UPDATE orders SET wallet_applied = 100 WHERE id = ?`, orderID.String()).Error)

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "full refund", "amount": 500.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 1)
	require.Equal(t, 400.0, rows[0].BasisAmount, "the CAPPED gateway-bound amount (500 total − 100 wallet), not the full 500 requested")
	require.Equal(t, 8.0, rows[0].Amount, "2% of 400")
}

func TestInitiateRefund_RepeatedPartialRefunds_Cashfree_LevyEach(t *testing.T) {
	db := setupPayDB(t)
	addChefPenaltyTable(t, db)
	withGatewayFeeLevyEnabled(t, 2, false)
	withCashfreeRefundGateway(t, http.StatusOK)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")

	w1 := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial 1", "amount": 100.0})
	require.Equal(t, http.StatusOK, w1.Code, w1.Body.String())

	w2 := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial 2", "amount": 50.0})
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())

	rows := gatewayFeePenaltyRowsFor(t, db, orderID)
	require.Len(t, rows, 2, "two distinct partial refunds each levy separately")
}

func TestInitiateRefund_GatewayFeeLevyFailure_DoesNotFailRefund(t *testing.T) {
	db := setupPayDB(t)
	// Deliberately DO NOT call addChefPenaltyTable — the levy insert will error.
	withGatewayFeeLevyEnabled(t, 2, false)
	withCashfreeRefundGateway(t, http.StatusOK)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-order")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial goodwill", "amount": 100.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String(), "the refund must succeed even though the levy insert errors")

	var body struct {
		RefundAmount float64 `json:"refundAmount"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, 100.0, body.RefundAmount)
}
