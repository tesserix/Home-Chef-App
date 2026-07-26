package services

// deferred_cancel_refund_test.go — #766-followup. RetryDeferredCancelRefunds is the sweep
// that finds cancelled orders whose chef-cancel gateway refund was deferred (a durable
// "pending:gateway-retry:<paise>" sentinel in refund_id — see deferred_cancel_refund.go)
// and re-issues the SAME idempotency-keyed refund until it lands. Own in-memory sqlite
// harness (hand-DDL'd, mirroring stuck_refund_reconcile_test.go's setupStuckRefundDB) plus
// a Razorpay stub injected via SetRazorpayClient/NewRazorpayTestClient, matching the
// gateway_idempotency_test.go pattern for capturing the outbound amount + idempotency key.

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupDeferredCancelRefundDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE orders (mode text DEFAULT 'live', test_session_id text, cloned_from_id text, delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '',
		id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT, chef_id TEXT,
		status TEXT, payment_status TEXT, payment_provider TEXT DEFAULT 'razorpay', total REAL DEFAULT 0,
		razorpay_payment_id TEXT DEFAULT '', refund_amount REAL DEFAULT 0, refund_id TEXT DEFAULT '', refund_reason TEXT,
		refund_initiated_by TEXT, refunded_at DATETIME, payout_hold_status TEXT DEFAULT '',
		razorpay_order_id TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// seedDeferredCancelOrder inserts a cancelled order carrying a deferred-refund sentinel —
// the shape RetryDeferredCancelRefunds' query matches (status=cancelled, refund_id LIKE
// the sentinel prefix, razorpay_payment_id set, updated_at past the grace).
func seedDeferredCancelOrder(t *testing.T, db *gorm.DB, refundID string, updatedAt time.Time) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		total, razorpay_payment_id, refund_amount, refund_id, refunded_at, updated_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), "ORD-D", uuid.NewString(), uuid.NewString(), string(models.OrderStatusCancelled), string(models.PaymentRefunded),
		3054.0, "pay_test123", 3054.0, refundID, time.Now(), updatedAt).Error)
	return id
}

func deferredRefundIDOf(t *testing.T, db *gorm.DB, id uuid.UUID) string {
	t.Helper()
	var refundID string
	require.NoError(t, db.Raw(`SELECT refund_id FROM orders WHERE id = ?`, id.String()).Scan(&refundID).Error)
	return refundID
}

// withDeferredRefundGateway wires GetRazorpay at a WORKING gateway that records the
// refunded amount, the X-Refund-Idempotency header, and how many refund calls it saw.
func withDeferredRefundGateway(t *testing.T) (gotAmount *int, gotKey *string, calls *int) {
	t.Helper()
	var amount int
	var key string
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/refund") {
			n++
			key = r.Header.Get(headerRefundIdempotency)
			body, _ := io.ReadAll(r.Body)
			var req RefundRequest
			_ = json.Unmarshal(body, &req)
			amount = req.Amount
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"rfnd_healed","entity":"refund","status":"processed"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "rzp_test_key", "rzp_test_secret", ""))
	t.Cleanup(func() { SetRazorpayClient(nil) })
	return &amount, &key, &n
}

// TestRetryDeferredCancelRefunds_IssuesGatewayRefund — a deferred sentinel past the grace
// is re-issued: the gateway is called once with the exact encoded paise and the SAME
// idempotency key CancelOrder would have used, and refund_id is replaced by the real
// gateway id (no longer a sentinel).
func TestRetryDeferredCancelRefunds_IssuesGatewayRefund(t *testing.T) {
	db := setupDeferredCancelRefundDB(t)
	gotAmount, gotKey, calls := withDeferredRefundGateway(t)
	orderID := seedDeferredCancelOrder(t, db, "pending:gateway-retry:305400", time.Now().Add(-time.Hour))

	require.Equal(t, 1, RetryDeferredCancelRefunds())
	require.Equal(t, 1, *calls)
	require.Equal(t, 305400, *gotAmount)
	require.Equal(t, normalizeIdempotencyKey(RefundFullIdempotencyKey(orderID)), *gotKey,
		"the retry must reuse the SAME stable idempotency key CancelOrder used — dedups a lost-response success instead of double-refunding")

	require.Equal(t, "rfnd_healed", deferredRefundIDOf(t, db, orderID), "the sentinel must be replaced by the real gateway id")
}

// TestRetryDeferredCancelRefunds_Idempotent — once healed, a second sweep must not find the
// row again (it no longer matches the sentinel LIKE clause) and must issue no further
// gateway call.
func TestRetryDeferredCancelRefunds_Idempotent(t *testing.T) {
	db := setupDeferredCancelRefundDB(t)
	_, _, calls := withDeferredRefundGateway(t)
	orderID := seedDeferredCancelOrder(t, db, "pending:gateway-retry:305400", time.Now().Add(-time.Hour))

	require.Equal(t, 1, RetryDeferredCancelRefunds())
	require.Equal(t, 1, *calls)

	require.Equal(t, 0, RetryDeferredCancelRefunds(), "an already-healed order must not be re-picked")
	require.Equal(t, 1, *calls, "no second gateway refund is ever issued")
	require.Equal(t, "rfnd_healed", deferredRefundIDOf(t, db, orderID))
}
