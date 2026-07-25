package handlers

// Regression: a loyalty-funded order captures only (Total − LoyaltyApplied) at
// the gateway (loyalty points are applied at checkout exactly like wallet credit
// and shrink plan.CapturePaise). The synchronous verify amount check must
// subtract the loyalty slice too — omitting it rejected every points-paid order
// with a false "Payment amount does not match the order total" 400, so the client
// never received a synchronous confirmation and hung on "Confirming your payment…".

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// A capture of exactly (Total − LoyaltyApplied) must verify (200) and mark the
// order completed. Before the fix expectedPaise omitted loyalty, so this 400'd.
func TestVerifyPayment_LoyaltyFundedOrder_Verifies_200(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 1000, "rzp_ord_loyalty", "")
	// ₹100 of the ₹1000 order paid with loyalty points at checkout → the gateway
	// order was created for, and captured, ₹900 (90000 paise).
	require.NoError(t, db.Exec(
		`UPDATE orders SET loyalty_applied = 100, loyalty_points_spent = 2000 WHERE id = ?`,
		orderID.String()).Error)

	fetchPaymentServer(t, "rzp_ord_loyalty", 90000) // ₹900 captured — matches Total − loyalty

	// No signature supplied: the amount/binding checks are the hard gate under test.
	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", regVerify,
		map[string]string{"razorpayPaymentId": "pay_x", "razorpayOrderId": "rzp_ord_loyalty"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, "completed", status, "a valid loyalty-funded capture must complete the order")
}

// Guardrail: a genuinely under-amount capture (less than Total − loyalty − wallet)
// must still be rejected — the fix must not turn into a blanket pass.
func TestVerifyPayment_LoyaltyOrder_UnderCapture_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 1000, "rzp_ord_short", "")
	require.NoError(t, db.Exec(
		`UPDATE orders SET loyalty_applied = 100 WHERE id = ?`, orderID.String()).Error)

	fetchPaymentServer(t, "rzp_ord_short", 50000) // only ₹500 captured vs ₹900 expected

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", regVerify,
		map[string]string{"razorpayPaymentId": "pay_x", "razorpayOrderId": "rzp_ord_short"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM orders WHERE id = ?`, orderID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status, "an under-amount capture must not complete the order")
}
