package handlers

// refund_request_dedup_http_test.go — #600. A client retry-after-success of an identical partial
// refund must NOT issue a second real refund. The chef goodwill RefundOrder + the customer/admin
// InitiateRefund both claim the submission (services.ClaimRefundRequest) before the reserve —
// keyed by an Idempotency-Key header when present, else the (order+amount+reason) fallback within
// a short window.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
)

// callGoodwillRefund posts to RefundOrder with an optional Idempotency-Key header.
func callGoodwillRefund(userID uuid.UUID, orderID string, body any, idemKey string) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/chef/orders/:orderId/refund", (&ChefOrderCancelHandler{}).RefundOrder)

	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/chef/orders/"+orderID+"/refund", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	if idemKey != "" {
		req.Header.Set("Idempotency-Key", idemKey)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// #600 regression: an identical goodwill refund resubmitted (no Idempotency-Key) is deduped by the
// order+amount+reason window — no second real gateway refund, refund_amount unchanged.
func TestRefundOrder_IdenticalResubmit_DedupedNoDoubleRefund(t *testing.T) {
	db := setupPayDB(t)
	_, refundCalls := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	markDeliveredPaid(t, orderID, "pay_x")

	body := map[string]any{"amount": 100.0, "reason": "goodwill"}
	w1 := callGoodwillRefund(chefUser, orderID.String(), body, "")
	require.Equal(t, http.StatusOK, w1.Code, w1.Body.String())
	require.Equal(t, 1, *refundCalls)
	_, amt1, _ := refundStateOf(t, orderID)
	require.Equal(t, 100.0, amt1)

	// The chef app re-submits the identical refund (its first HTTP response was dropped).
	w2 := callGoodwillRefund(chefUser, orderID.String(), body, "")
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	require.Equal(t, 1, *refundCalls, "#600: the duplicate issues NO second real refund")
	_, amt2, _ := refundStateOf(t, orderID)
	require.Equal(t, 100.0, amt2, "refund_amount is not double-counted")
}

// A retry carrying the SAME Idempotency-Key dedups exactly.
func TestRefundOrder_SameIdempotencyKey_Deduped(t *testing.T) {
	db := setupPayDB(t)
	_, refundCalls := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	markDeliveredPaid(t, orderID, "pay_x")

	body := map[string]any{"amount": 100.0, "reason": "goodwill"}
	require.Equal(t, http.StatusOK, callGoodwillRefund(chefUser, orderID.String(), body, "idem-1").Code)
	require.Equal(t, 1, *refundCalls)

	w2 := callGoodwillRefund(chefUser, orderID.String(), body, "idem-1")
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	require.Equal(t, 1, *refundCalls, "same Idempotency-Key → deduped")
}

// After the dedup window, a genuinely-intended identical repeat refund is allowed.
func TestRefundOrder_IdenticalRepeatAfterWindow_Allowed(t *testing.T) {
	db := setupPayDB(t)
	_, refundCalls := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	markDeliveredPaid(t, orderID, "pay_x")

	body := map[string]any{"amount": 100.0, "reason": "goodwill"}
	require.Equal(t, http.StatusOK, callGoodwillRefund(chefUser, orderID.String(), body, "").Code)
	require.Equal(t, 1, *refundCalls)

	// Age the claim past the window (no time travel in sqlite).
	require.NoError(t, database.DB.Exec(
		`UPDATE processed_events SET processed_at = datetime('now','-1 hour') WHERE consumer = 'refund-request'`).Error)

	w2 := callGoodwillRefund(chefUser, orderID.String(), body, "")
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	require.Equal(t, 2, *refundCalls, "a genuine identical repeat after the window issues a real refund")
	_, amt, _ := refundStateOf(t, orderID)
	require.Equal(t, 200.0, amt)
}

// InitiateRefund (customer/admin path) dedups an identical partial to-wallet resubmit.
func TestInitiateRefund_IdenticalResubmit_Deduped(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")

	body := map[string]any{"reason": "goodwill", "amount": 100.0, "toWallet": true}
	w1 := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund, body)
	require.Equal(t, http.StatusOK, w1.Code, w1.Body.String())

	w2 := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund, body)
	require.Equal(t, http.StatusOK, w2.Code, w2.Body.String())
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
	require.Equal(t, true, resp["duplicate"], "the identical resubmit is reported as a duplicate")

	var balance float64
	require.NoError(t, db.Raw(`SELECT COALESCE(balance,0) FROM wallets WHERE user_id = ?`, cust.String()).Scan(&balance).Error)
	require.Equal(t, 100.0, balance, "the wallet is credited once, not twice")
	s := loadRefundState(t, db, orderID)
	require.Equal(t, 100.0, s.RefundAmount, "refund_amount not double-counted")
}
