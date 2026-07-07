package handlers

// chef_order_cancel_refund_test.go — #576. The two chef-side refund paths that issued
// the Razorpay CreateRefund BEFORE any atomic claim:
//
//   - CancelOrderItem (per-line): the item flip had no `is_cancelled = false` guard, so
//     two concurrent cancels of the SAME line both passed the stale preloaded check and
//     both incremented order.RefundAmount → ledger inflated 2×.
//   - RefundOrder (post-delivery goodwill): no atomic claim at all (only checked
//     status == delivered), so it raced a concurrent InitiateRefund on order.RefundAmount.
//
// Both now serialize on an atomic claim (mirroring claimRefundForProcessing / the
// CancelOrder CAS), which also unblocks threading the #574 gateway idempotency keys
// (a stable key is only safe once a same-order double-submit can't double-count locally).
//
// Concurrency is exercised deterministically (the sqlite :memory: harness can't run true
// concurrent txns faithfully): the per-line CAS is proven on the extracted claim helper,
// and the goodwill claim by driving RefundOrder against an order a concurrent refund has
// already claimed (payment_status left non-completed).

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
)

// callChefCancel drives a ChefOrderCancelHandler route with the user id injected the way
// the auth middleware would (loadChefForUser reads it).
func callChefCancel(userID uuid.UUID, method, path string, register func(*gin.Engine, *ChefOrderCancelHandler), body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	register(r, &ChefOrderCancelHandler{})

	var reader *bytes.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		reader = bytes.NewReader(buf)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func regGoodwillRefund(r *gin.Engine, h *ChefOrderCancelHandler) {
	r.POST("/chef/orders/:orderId/refund", h.RefundOrder)
}

// withRefundGateway points GetRazorpay at an httptest server that answers the refund POST
// and records the X-Refund-Idempotency header + how many real refunds were issued.
func withRefundGateway(t *testing.T) (gotKey *string, refundCalls *int) {
	t.Helper()
	var key string
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/refund") {
			key = r.Header.Get("X-Refund-Idempotency")
			calls++
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "rfnd_test"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "rzp_test_key", "rzp_test_secret", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })
	return &key, &calls
}

func markDeliveredPaid(t *testing.T, orderID uuid.UUID, payID string) {
	t.Helper()
	require.NoError(t, database.DB.Exec(
		`UPDATE orders SET status = 'delivered', payment_status = 'completed', razorpay_payment_id = ? WHERE id = ?`,
		payID, orderID.String()).Error)
}

func refundStateOf(t *testing.T, orderID uuid.UUID) (paymentStatus string, refundAmount float64, refundedAtValid bool) {
	t.Helper()
	var row struct {
		PaymentStatus string
		RefundAmount  float64
		RefundedAt    *time.Time
	}
	require.NoError(t, database.DB.Raw(
		`SELECT payment_status, refund_amount, refunded_at FROM orders WHERE id = ?`, orderID.String()).Scan(&row).Error)
	return row.PaymentStatus, row.RefundAmount, row.RefundedAt != nil
}

// ── CancelOrderItem per-line CAS (deterministic) ──

func TestClaimOrderItemForCancel_SerializesConcurrentDuplicates(t *testing.T) {
	db := setupPayDB(t)
	// setupPayDB's order_items is the minimal RemainingRefundable shape; the per-line
	// claim also writes the cancel bookkeeping columns.
	for _, col := range []string{"cancelled_reason TEXT DEFAULT ''", "cancelled_at DATETIME", "refund_id TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE order_items ADD COLUMN `+col).Error)
	}
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x") // payment_status=completed
	itemID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,0,0)`,
		itemID.String(), orderID.String()).Error)
	now := time.Now().UTC()

	// The claim runs BEFORE the gateway refund — the loser never reaches CreateRefund.
	_, won1, err1 := reserveOrderItemForCancel(db, orderID, itemID, "customer_request", now)
	require.NoError(t, err1)
	require.True(t, won1, "the winner flips the line is_cancelled=false → true")

	_, won2, err2 := reserveOrderItemForCancel(db, orderID, itemID, "customer_request", now)
	require.NoError(t, err2)
	require.False(t, won2, "a concurrent duplicate per-line cancel loses the CAS → never reaches the gateway, never double-counts")

	var cancelled bool
	require.NoError(t, db.Raw(`SELECT is_cancelled FROM order_items WHERE id = ?`, itemID.String()).Row().Scan(&cancelled))
	require.True(t, cancelled, "the winning claim marks the line cancelled")
}

// #620: a per-line cancel must LOSE when an order-level refund has claimed the order
// (payment_status != completed) — otherwise it issues a second real gateway refund on top of
// the order-level one and double-counts refund_amount (customer over-refund).
func TestClaimOrderItemForCancel_LosesToInFlightOrderRefund(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_reason TEXT DEFAULT ''", "cancelled_at DATETIME", "refund_id TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE order_items ADD COLUMN `+col).Error)
	}
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	itemID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,0,0)`,
		itemID.String(), orderID.String()).Error)
	// An order-level refund (InitiateRefund/CancelOrder) has claimed the order.
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded' WHERE id = ?`, orderID.String()).Error)

	_, won, err := reserveOrderItemForCancel(db, orderID, itemID, "customer_request", time.Now().UTC())
	require.ErrorIs(t, err, errOrderRefundInProgress, "the line claim reports the order-refund conflict")
	require.False(t, won)

	var cancelled bool
	require.NoError(t, db.Raw(`SELECT is_cancelled FROM order_items WHERE id = ?`, itemID.String()).Row().Scan(&cancelled))
	require.False(t, cancelled, "the line is NOT cancelled — no second refund is issued")
}

// ── RefundOrder goodwill claim + gateway key ──

// A goodwill refund on an order a concurrent refund has ALREADY claimed (payment_status
// no longer 'completed') must lose the claim → 409, issuing no gateway refund and not
// touching refund_amount. Pre-fix (no claim) it would issue a second real refund and
// double-count.
func TestRefundOrder_ConcurrentRefundAlreadyClaimed_409(t *testing.T) {
	db := setupPayDB(t)
	_, refundCalls := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	markDeliveredPaid(t, orderID, "pay_x")
	// A concurrent InitiateRefund already won the claim (completed→refunded).
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = 'refunded' WHERE id = ?`, orderID.String()).Error)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/refund", regGoodwillRefund,
		map[string]any{"amount": 100.0, "reason": "goodwill"})
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Equal(t, 0, *refundCalls, "no second gateway refund issued when a refund is already in progress")

	_, amt, _ := refundStateOf(t, orderID)
	require.Equal(t, 0.0, amt, "refund_amount not double-counted")
}

// A PARTIAL goodwill refund takes the claim, reverts payment_status→completed so the
// order stays non-terminal (releasable + repeatable, #549), and threads the gateway
// idempotency key.
func TestRefundOrder_PartialGoodwill_ClaimRevertedAndKeyed(t *testing.T) {
	db := setupPayDB(t)
	gotKey, refundCalls := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	markDeliveredPaid(t, orderID, "pay_x")

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/refund", regGoodwillRefund,
		map[string]any{"amount": 100.0, "reason": "partial goodwill"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 1, *refundCalls)
	require.NotEmpty(t, *gotKey, "the goodwill refund threads a gateway idempotency key (#574/#576)")

	status, amt, refundedAt := refundStateOf(t, orderID)
	require.Equal(t, "completed", status, "a partial goodwill refund reverts the claim → order stays non-terminal")
	require.False(t, refundedAt, "a partial refund must NOT stamp refunded_at (would forfeit the whole hold)")
	require.Equal(t, 100.0, amt)
}

// A FULL goodwill refund keeps the claim (terminal), stamps refunded_at, and keys the gateway.
func TestRefundOrder_FullGoodwill_TerminalAndKeyed(t *testing.T) {
	db := setupPayDB(t)
	gotKey, _ := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	markDeliveredPaid(t, orderID, "pay_x")

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/refund", regGoodwillRefund,
		map[string]any{"amount": 500.0, "reason": "full goodwill"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.NotEmpty(t, *gotKey, "full goodwill refund threads a gateway idempotency key")

	status, amt, refundedAt := refundStateOf(t, orderID)
	require.Equal(t, "refunded", status, "a full goodwill refund is terminal (claim not reverted)")
	require.True(t, refundedAt, "a full refund stamps refunded_at")
	require.Equal(t, 500.0, amt)
}

// A duplicate full goodwill refund (client retry after the order is already fully
// refunded) is idempotent — no second gateway refund, no double-count.
func TestRefundOrder_AfterFullRefund_IdempotentNoDoubleRefund(t *testing.T) {
	db := setupPayDB(t)
	_, refundCalls := withRefundGateway(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_o", "pay_x")
	markDeliveredPaid(t, orderID, "pay_x")
	// Already fully refunded.
	require.NoError(t, db.Exec(`UPDATE orders SET refund_amount = 500, refunded_at = ?, payment_status = 'refunded' WHERE id = ?`,
		time.Now().UTC(), orderID.String()).Error)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/refund", regGoodwillRefund,
		map[string]any{"amount": 100.0, "reason": "retry"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String()) // remaining<=0 → idempotent 200
	require.Equal(t, 0, *refundCalls, "no gateway refund when nothing remains refundable")
}
