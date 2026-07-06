package services

// cancellation_order_refund_test.go — #392. RefundOrderForCancellation issues exactly
// one full customer refund on a paid order, no-ops an unpaid or already-refunded one,
// and is concurrency-safe via the shared refunded_at claim. The wallet provider path
// exercises the whole flow with no external gateway (CreditWallet is DB-only).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// seedRazorpayWalletOrder is a gateway-charged order that ALSO consumed store credit at
// checkout (wallet_applied), so only (Total − WalletApplied) was captured at the gateway.
func seedRazorpayWalletOrder(t *testing.T, db *gorm.DB, total, walletApplied float64) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-W", CustomerID: uuid.New(), ChefID: uuid.New(),
		Status: models.OrderStatusCancelled, PaymentStatus: models.PaymentCompleted,
		PaymentProvider: "razorpay", RazorpayPaymentID: "pay_123", Total: total, WalletApplied: walletApplied,
	}
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		payment_provider, razorpay_payment_id, total, wallet_applied, refund_amount) VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), o.ChefID.String(), string(o.Status),
		string(models.PaymentCompleted), "razorpay", "pay_123", total, walletApplied, 0.0).Error)
	return o
}

// #609: the wallet-capture split re-credits the wallet-funded slice and refunds only the
// captured slice at the gateway, but the RESERVED ledger amount is the FULL remaining. Total
// money out (wallet re-credit + gateway) must equal the reserved refund_amount.
func TestRefundOrderForCancellation_WalletApplied_SplitConservesReservedTotal(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 120) // ₹300 total, ₹120 from wallet → ₹180 captured

	var gatewayAmt int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Amount int `json:"amount"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		gatewayAmt = body.Amount
		_, _ = w.Write([]byte(`{"id":"rfnd_test","status":"processed"}`))
	}))
	defer srv.Close()
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	require.Equal(t, 18000, gatewayAmt, "gateway refunds only the captured ₹180 (paise)")
	ps, amt, rid, _ := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentRefunded), ps)
	require.Equal(t, 300.0, amt, "refund_amount = the full reserved ₹300 (wallet 120 + gateway 180) — conserved")
	require.NotEmpty(t, rid)
	var bal float64
	db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal)
	require.Equal(t, 120.0, bal, "the wallet-funded portion is re-credited as store credit")
}

// #609: on a gateway failure the reservation must release the FULL reserved amount, not the
// reduced gateway share — else refund_amount would be left inflated by the wallet portion.
func TestRefundOrderForCancellation_WalletApplied_GatewayFailure_ReleasesFullReservation(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedRazorpayWalletOrder(t, db, 300, 120)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"description":"boom"}}`))
	}))
	defer srv.Close()
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })

	require.Error(t, RefundOrderForCancellation(o, "customer", "cancel"), "gateway failure surfaces")

	ps, amt, _, at := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentCompleted), ps, "claim reverted so a retry can re-refund")
	require.Nil(t, at, "refunded_at cleared")
	require.Equal(t, 0.0, amt, "the FULL reserved ₹300 is released, not just the ₹180 gateway share (#609)")
}

func setupCancelRefundDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT DEFAULT '', customer_id TEXT, chef_id TEXT,
			status TEXT, payment_status TEXT, payment_provider TEXT DEFAULT 'razorpay', razorpay_payment_id TEXT DEFAULT '',
			stripe_payment_intent_id TEXT DEFAULT '', total REAL DEFAULT 0, wallet_applied REAL DEFAULT 0, currency TEXT DEFAULT 'INR',
			refund_amount REAL DEFAULT 0, refund_id TEXT DEFAULT '', refund_reason TEXT, refund_initiated_by TEXT, refunded_at DATETIME,
			payout_hold_status TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0, currency TEXT DEFAULT 'INR',
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT, source TEXT, amount REAL,
			balance_after REAL, currency TEXT, order_id TEXT, reason TEXT, created_by TEXT, idempotency_key TEXT UNIQUE, created_at DATETIME)`,
		`CREATE TABLE order_items (id TEXT PRIMARY KEY, order_id TEXT, is_cancelled BOOLEAN DEFAULT 0,
			refund_amount REAL DEFAULT 0, subtotal REAL DEFAULT 0, created_at DATETIME)`,
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, status TEXT DEFAULT 'pending', created_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT,
			payload TEXT, status TEXT, attempts INT, last_error TEXT, next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
		`CREATE TABLE audit_logs (id TEXT PRIMARY KEY, user_id TEXT, action TEXT, entity_type TEXT, entity_id TEXT,
			old_value TEXT, new_value TEXT, ip_address TEXT, user_agent TEXT, correlation_id TEXT, created_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func seedWalletOrder(t *testing.T, db *gorm.DB, paymentStatus models.PaymentStatus, total, refundAmount float64) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-1", CustomerID: uuid.New(), ChefID: uuid.New(),
		Status: models.OrderStatusCancelled, PaymentStatus: paymentStatus, PaymentProvider: "wallet",
		Total: total, RefundAmount: refundAmount,
	}
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		payment_provider, total, refund_amount) VALUES (?,?,?,?,?,?,?,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), o.ChefID.String(), string(o.Status),
		string(paymentStatus), "wallet", total, refundAmount).Error)
	return o
}

func loadRefund(t *testing.T, db *gorm.DB, id uuid.UUID) (paymentStatus string, refundAmount float64, refundID string, refundedAt *time.Time) {
	t.Helper()
	row := struct {
		PaymentStatus string
		RefundAmount  float64
		RefundID      string
		RefundedAt    *time.Time
	}{}
	require.NoError(t, db.Raw(`SELECT payment_status, refund_amount, refund_id, refunded_at FROM orders WHERE id = ?`, id.String()).Scan(&row).Error)
	return row.PaymentStatus, row.RefundAmount, row.RefundID, row.RefundedAt
}

func TestRefundOrderForCancellation_PaidWalletOrderRefundsInFull(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 250, 0)

	require.NoError(t, RefundOrderForCancellation(o, "customer", "customer cancellation"))

	ps, amt, rid, at := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentRefunded), ps)
	require.Equal(t, 250.0, amt, "full refund of the remaining amount")
	require.NotEmpty(t, rid)
	require.NotNil(t, at)
	var bal float64
	require.NoError(t, db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal).Error)
	require.Equal(t, 250.0, bal, "customer wallet credited the refund")
}

func TestRefundOrderForCancellation_UnpaidNoOp(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentPending, 250, 0)

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	ps, amt, _, at := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentPending), ps, "unpaid order untouched")
	require.Equal(t, 0.0, amt)
	require.Nil(t, at)
}

func TestRefundOrderForCancellation_AlreadyRefundedNoOp(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 250, 0)
	now := time.Now()
	require.NoError(t, db.Exec(`UPDATE orders SET refunded_at = ? WHERE id = ?`, now, o.ID.String()).Error)

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	_, amt, _, _ := loadRefund(t, db, o.ID)
	require.Equal(t, 0.0, amt, "an already-refunded order is not re-refunded")
	var count int64
	db.Raw(`SELECT count(*) FROM wallet_txns`).Scan(&count)
	require.Equal(t, int64(0), count, "no wallet credit on a no-op")
}

func TestRefundOrderForCancellation_DoubleCallRefundsOnce(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 250, 0)

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))
	// second call (stale in-memory order still says completed/refunded_at nil) must no-op
	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	_, amt, _, _ := loadRefund(t, db, o.ID)
	require.Equal(t, 250.0, amt, "refund_amount stamped exactly once")
	var count int64
	db.Raw(`SELECT count(*) FROM wallet_txns`).Scan(&count)
	require.Equal(t, int64(1), count, "exactly one wallet credit — the atomic claim prevents a double refund")
}

// A sibling path that has claimed via payment_status (InitiateRefund-style) but not
// yet stamped refunded_at — the mid-gateway window — must exclude this refund. The
// two-column claim (payment_status='completed' AND refunded_at IS NULL) is what
// closes this (#392 verify CRITICAL #1).
func TestRefundOrderForCancellation_ExcludedByPaymentStatusClaim(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 250, 0)
	// Simulate InitiateRefund's claim: payment_status flipped, refunded_at still NULL.
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = ? WHERE id = ?`,
		string(models.PaymentRefunded), o.ID.String()).Error)

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	_, amt, _, _ := loadRefund(t, db, o.ID)
	require.Equal(t, 0.0, amt, "a payment_status=refunded order (sibling mid-flight) is not re-refunded")
	var count int64
	db.Raw(`SELECT count(*) FROM wallet_txns`).Scan(&count)
	require.Equal(t, int64(0), count, "no second gateway/wallet refund")
}

func TestRefundOrderForCancellation_PriorPartialRefundIsAdditive(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedWalletOrder(t, db, models.PaymentCompleted, 250, 100) // ₹100 already refunded

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	_, amt, _, _ := loadRefund(t, db, o.ID)
	require.Equal(t, 250.0, amt, "cumulative refund = prior 100 + remaining 150")
	var bal float64
	db.Raw(`SELECT balance FROM wallets WHERE user_id = ?`, o.CustomerID.String()).Scan(&bal)
	require.Equal(t, 150.0, bal, "only the remaining ₹150 is refunded now")
}
