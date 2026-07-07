package handlers

// payment_reserve_refund_test.go — #611. InitiateRefund now reserves the refund UNDER A ROW
// LOCK (services.ReserveRefund) instead of an unlocked RemainingRefundable read + a separate
// claim + a stale `order.RefundAmount + amount` persist. These drive the GATEWAY branch via
// the razorpay httptest seam (the existing partial tests only exercise the to-wallet branch,
// since GetRazorpay()==nil there 503s before persist). They pin: a gateway refund records the
// RESERVED amount exactly once (no double read-modify-write), respects prior refunds (never
// over-refunds past the order total), and classifies full vs partial correctly.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/homechef/api/services"
)

// withGatewayRefundCapture points GetRazorpay at an httptest server answering the refund POST
// and captures how many real refunds were issued + the paise amount of the last one.
func withGatewayRefundCapture(t *testing.T) (calls *int, lastPaise *int) {
	t.Helper()
	var n int
	var paise int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && stringsHasRefundSuffix(r.URL.Path) {
			var body struct {
				Amount int `json:"amount"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			n++
			paise = body.Amount
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "rfnd_test", "status": "processed"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "rzp_test_key", "rzp_test_secret", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })
	return &n, &paise
}

func stringsHasRefundSuffix(p string) bool {
	const s = "/refund"
	return len(p) >= len(s) && p[len(p)-len(s):] == s
}

// A partial gateway refund issues exactly ONE real refund for the requested paise, records the
// reserved amount in refund_amount, and stays NON-terminal (payment_status reverts to completed,
// refunded_at NULL) so the chef's payout hold is releasable for the remainder.
func TestInitiateRefund_PartialGateway_RecordsReservedOnce(t *testing.T) {
	db := setupPayDB(t)
	calls, paise := withGatewayRefundCapture(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "partial goodwill", "amount": 200.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 1, *calls, "exactly one real gateway refund issued")
	require.Equal(t, 20000, *paise, "gateway refund for the reserved 200.00 (20000 paise)")

	s := loadRefundState(t, db, orderID)
	require.Equal(t, 200.0, s.RefundAmount, "refund_amount incremented by the reserved amount (once, not double)")
	require.Equal(t, "completed", s.PaymentStatus, "a partial reverts to completed")
	require.False(t, s.RefundedAt.Valid, "a partial does not stamp refunded_at")
}

// A full gateway refund is terminal: payment_status + status → refunded and refunded_at stamped.
func TestInitiateRefund_FullGateway_Terminal(t *testing.T) {
	db := setupPayDB(t)
	calls, paise := withGatewayRefundCapture(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "full refund", "amount": 500.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 1, *calls)
	require.Equal(t, 50000, *paise)

	s := loadRefundState(t, db, orderID)
	require.Equal(t, 500.0, s.RefundAmount)
	require.Equal(t, "refunded", s.PaymentStatus, "a full gateway refund is terminal")
	require.Equal(t, "refunded", s.Status)
	require.True(t, s.RefundedAt.Valid)
}

// #611 over-refund guard (single-threaded form): with ₹300 already refunded on a ₹500 order,
// a "full" refund reserves ONLY the remaining ₹200 under the lock — refund_amount lands
// exactly at the ₹500 total, never past it, and the gateway is asked for 200, not 500.
func TestInitiateRefund_Gateway_RespectsPriorRefund_NoOverRefund(t *testing.T) {
	db := setupPayDB(t)
	calls, paise := withGatewayRefundCapture(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")
	// A prior partial already refunded 300 and left the order non-terminal.
	require.NoError(t, db.Exec(`UPDATE orders SET refund_amount = 300 WHERE id = ?`, orderID.String()).Error)

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "the rest", "amount": 0.0}) // 0 = full remaining
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	require.Equal(t, 1, *calls)
	require.Equal(t, 20000, *paise, "reserves + refunds only the remaining 200, not the full 500")

	s := loadRefundState(t, db, orderID)
	require.Equal(t, 500.0, s.RefundAmount, "refund_amount reaches the total exactly — never over-refunds to 800")
	require.Equal(t, "refunded", s.PaymentStatus, "exhausting the remainder is terminal")
	require.True(t, s.RefundedAt.Valid)
}

// A request exceeding the remaining refundable is rejected up front (best-effort UX 400)
// before any gateway call — the reserve never runs, no money moves.
func TestInitiateRefund_Gateway_ExceedsRemaining_Rejected(t *testing.T) {
	db := setupPayDB(t)
	calls, _ := withGatewayRefundCapture(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")
	require.NoError(t, db.Exec(`UPDATE orders SET refund_amount = 400 WHERE id = ?`, orderID.String()).Error)

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "too much", "amount": 300.0}) // only 100 remains
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, 0, *calls, "no gateway refund issued when the request exceeds the remaining")

	s := loadRefundState(t, db, orderID)
	require.Equal(t, 400.0, s.RefundAmount, "refund_amount untouched")
	require.Equal(t, "completed", s.PaymentStatus)
}
