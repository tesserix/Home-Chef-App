package handlers

// chef_order_cancel_deferred_test.go — #766-followup. ChefOrderCancelHandler.CancelOrder
// must never hard-block a chef's cancel on the synchronous the retired gateway refund: the order
// always flips to cancelled + the full-refund obligation is reserved (payment_status /
// refunded_at / refund_amount), regardless of whether the gateway call can complete right
// now. When it can't (no gateway configured, or CreateRefund errors), the handler defers by
// stamping a "pending:gateway-retry:<paise>" sentinel into refund_id and still returns 200
// — services.RetryDeferredCancelRefunds (deferred_cancel_refund_test.go, services package)
// is what re-issues the gateway call later.

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
)

func regChefCancelOrder(r *gin.Engine, h *ChefOrderCancelHandler) {
	r.POST("/chef/orders/:orderId/cancel", h.CancelOrder)
}

// pinSingleConn caps the harness's sqlite :memory: pool at one connection. CancelOrder
// fires a background goroutine (services.CancelOrderDelivery) that queries database.DB
// concurrently with the handler's own remaining reads; without this pin, database/sql may
// open a SECOND pooled connection, which for a bare ":memory:" DSN (no shared cache) is a
// completely separate, empty database — producing a flaky "no such table: orders". Same
// idiom as services/webhook_dedup_test.go's TestClaimWebhookEvent_Concurrent.
func pinSingleConn(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
}

// markPreparing flips a freshly-created payOrder row into a cancellable mid-prep state
// (payOrder always inserts status='pending', which cancellableStatuses rejects).
func markPreparing(t *testing.T, orderID uuid.UUID) {
	t.Helper()
	require.NoError(t, database.DB.Exec(`UPDATE orders SET status = 'preparing' WHERE id = ?`, orderID.String()).Error)
}

// chefCancelStateOf reads back the columns the deferral contract cares about.
func chefCancelStateOf(t *testing.T, orderID uuid.UUID) (status, refundID string, refundAmount float64, refundedAtValid bool) {
	t.Helper()
	var r struct {
		Status       string
		RefundID     string
		RefundAmount float64
		RefundedAtN  int64
	}
	require.NoError(t, database.DB.Raw(
		`SELECT status, refund_id, refund_amount, (refunded_at IS NOT NULL) AS refunded_at_n FROM orders WHERE id = ?`,
		orderID.String()).Scan(&r).Error)
	return r.Status, r.RefundID, r.RefundAmount, r.RefundedAtN == 1
}

// TestCancelOrder_GatewayFailure_CancelsAndDefersRefund — a chef cancel against a paid,
// mid-prep order whose gateway refund call FAILS must still cancel the order (200), reserve
// the full refund (refunded_at set, refund_amount == total), and leave a deferred-retry
// sentinel in refund_id instead of hard-blocking with a 502.
func TestCancelOrder_GatewayFailure_CancelsAndDefersRefund(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	withCashfreeRefundGateway(t, http.StatusInternalServerError)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf_o")
	markPreparing(t, orderID)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "customer_request"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status, "the order must cancel even though the gateway refund failed")
	require.True(t, strings.HasPrefix(refundID, "pending:gateway-retry:"), "a deferred sentinel must be recorded, got %q", refundID)
	paise, err := strconv.Atoi(strings.TrimPrefix(refundID, "pending:gateway-retry:"))
	require.NoError(t, err)
	require.Equal(t, 50000, paise, "the sentinel must encode the full owed amount in paise (₹500 → 50000)")
	require.True(t, refundedAt, "refunded_at must be set — the reservation guarantees the refund and blocks the chef payout")
	require.Equal(t, 500.0, refundAmount, "the full order total is reserved as owed, gateway outcome notwithstanding")
}

// TestCancelOrder_GatewayNil_CancelsAndDefers — same contract when the gateway is
// entirely unconfigured, the other early-block condition being replaced.
func TestCancelOrder_GatewayNil_CancelsAndDefers(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	// A gateway lookup with no cached client falls through to a live Secret Manager fetch,
	// which needs a non-nil config.AppConfig for its dev-fallback check — set an empty one
	// so the fetch fails cleanly (no real credentials) and the lookup returns nil, instead
	// of panicking on a nil config in this test binary.
	pinSingleConn(t, db)
	prevCfg := config.AppConfig
	config.AppConfig = &config.Config{Environment: "test"}
	t.Cleanup(func() { config.AppConfig = prevCfg })
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 300, "cf_o")
	markPreparing(t, orderID)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "out_of_ingredient"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status)
	require.True(t, strings.HasPrefix(refundID, "pending:gateway-retry:"), "got %q", refundID)
	require.True(t, refundedAt)
	require.Equal(t, 300.0, refundAmount)
}

// TestCancelOrder_GatewaySuccess_NoSentinel — the happy-path guard: when the gateway
// refund succeeds, refund_id is the REAL gateway id, never the deferred sentinel, and the
// gateway is called exactly once.
func TestCancelOrder_GatewaySuccess_NoSentinel(t *testing.T) {
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

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "equipment_failure"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 1, gw.calls)

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status)
	require.Equal(t, gw.key, refundID, "the real gateway id must be recorded, never a deferred sentinel")
	require.True(t, refundedAt)
	require.Equal(t, 200.0, refundAmount)
}

// TestCancelOrder_GatewaySuccess_PersistsCancelledThenReplacesSentinel — #766-followup. The
// crash-safety reorder: CancelOrder must persist status=cancelled + the deferred sentinel
// BEFORE calling the gateway, then (on success) guard-replace the sentinel with the real
// refund id. The synchronous call can't observe the intermediate sentinel, but asserting the
// correct FINAL state — real id, no sentinel, gateway called exactly once — proves the
// reserve → persist-sentinel → gateway → replace sequence actually ran end to end (a handler
// that skipped the sentinel persist, or never replaced it, would fail one of these asserts).
func TestCancelOrder_GatewaySuccess_PersistsCancelledThenReplacesSentinel(t *testing.T) {
	db := setupPayDB(t)
	for _, col := range []string{"cancelled_at DATETIME", "cancel_reason TEXT DEFAULT ''"} {
		require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN `+col).Error)
	}
	pinSingleConn(t, db)
	gw := withCashfreeRefundGateway(t, http.StatusOK)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := cfPayOrder(t, db, cust, chef, "completed", 750, "cf_o")
	markPreparing(t, orderID)

	w := callChefCancel(chefUser, http.MethodPost, "/chef/orders/"+orderID.String()+"/cancel", regChefCancelOrder,
		map[string]any{"reason": "customer_request"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 1, gw.calls, "the gateway must be called exactly once")

	status, refundID, refundAmount, refundedAt := chefCancelStateOf(t, orderID)
	require.Equal(t, "cancelled", status, "the reordered flow still lands the order cancelled")
	require.Equal(t, gw.key, refundID, "the sentinel persisted before the gateway call must be replaced by the real id")
	require.False(t, strings.HasPrefix(refundID, "pending:gateway-retry:"), "no sentinel must survive a successful gateway call")
	require.True(t, refundedAt, "refunded_at was stamped by the reservation, unaffected by the reorder")
	require.Equal(t, 750.0, refundAmount)
}
