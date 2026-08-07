package services

// order_payment_reconcile_cron_test.go — #872 step 2, Task 4. Pins the
// belt-and-suspenders behind step 1's stale-order gateway gate: a
// captured-but-unconfirmed order is settled (not merely skipped) within the
// grace window, through the SAME SettleCashfreeOrder
// core the HTTP verify legs use, and the cancelled-order backfill boundary is
// enforced structurally by the query, never merely by convention.
//
// Reuses setupCancelRefundDB, seedStaleOrder, withCashfreeServer,
// withRazorpayServerFor from stale_order_cron_test.go (same package) — extended
// with chef_profiles/deliveries/delivery_partners so Preload("Chef") and
// Preload("Delivery.DeliveryPartner") (SettleOrderWallet's requirement) don't
// error on a missing table. Every seeded order in this file has
// wallet_applied=loyalty_applied=loyalty_points_spent=0 (seedStaleOrder's
// defaults), so SettleOrderWallet's credit-settlement branch never runs and no
// wallet/loyalty tables are needed.

import (
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// setupReconcileDB extends setupCancelRefundDB with the tables
// Preload("Chef").Preload("Delivery.DeliveryPartner") queries against. Empty in
// every test here — no order's wallet/loyalty credit is set, so
// SettleOrderWallet's provider-split branch (the only reader of Chef/Delivery
// Route account fields) never runs.
func setupReconcileDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupCancelRefundDB(t)
	// setupCancelRefundDB's orders table (the base setupCancelRefundDB extends)
	// has no payment_method column — CompleteCashfreeOrderTx
	// stamp it alongside payment_status=completed.
	require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN payment_method TEXT DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
		payment_provider TEXT DEFAULT 'razorpay', razorpay_account_id TEXT DEFAULT '',
		payout_country TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE deliveries (
		id TEXT PRIMARY KEY, order_id TEXT, delivery_partner_id TEXT, status TEXT DEFAULT 'pending',
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_partners (
		id TEXT PRIMARY KEY, user_id TEXT, razorpay_account_id TEXT DEFAULT '',
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	return db
}

func reconcilePaymentRow(t *testing.T, db *gorm.DB, id interface{ String() string }) (paymentStatus, gatewayPaymentID string) {
	t.Helper()
	row := struct {
		PaymentStatus     string
		RazorpayPaymentID string
	}{}
	require.NoError(t, db.Raw(
		`SELECT payment_status, razorpay_payment_id FROM orders WHERE id = ?`, id.String(),
	).Scan(&row).Error)
	return row.PaymentStatus, row.RazorpayPaymentID
}

const reconcileGraceElapsed = 10 * time.Minute // safely past the 5-minute grace

// ── Scenario 1: Cashfree captured + amount matches → settled, idempotent ────

func TestOrderPaymentReconcile_Cashfree_CapturedAndMatches_Settles(t *testing.T) {
	db := setupReconcileDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_match", models.ChefModeLive, now.Add(-reconcileGraceElapsed))
	o.Total = 300

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"cf_payment_id":777,"order_id":"cf_order_match","payment_status":"SUCCESS","payment_amount":300.00,"payment_group":"upi"}]`))
	})

	settled := reconcileOrderPayments(db, now)
	require.Equal(t, 1, settled)

	paymentStatus, gatewayPaymentID := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentCompleted), paymentStatus)
	require.Equal(t, "777", gatewayPaymentID, "cf_payment_id stamped into the shared gateway-payment-id column")

	// Idempotency (query-level): the row is now payment_status=completed, so a
	// second scan never re-selects it.
	settled = reconcileOrderPayments(db, now)
	require.Equal(t, 0, settled, "already settled — not re-selected on the next tick")
	paymentStatus2, gatewayPaymentID2 := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, paymentStatus, paymentStatus2)
	require.Equal(t, gatewayPaymentID, gatewayPaymentID2)
}

// ── Scenario 3: Cashfree captured payment bound to a DIFFERENT order → rejected ─

func TestOrderPaymentReconcile_Cashfree_PaymentBoundToDifferentOrder_NeverSettles(t *testing.T) {
	db := setupReconcileDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_mine", models.ChefModeLive, now.Add(-reconcileGraceElapsed))
	o.Total = 300

	// The gateway record's own order_id does not match the seeded order's
	// stamped gateway order id — proves ValidateCapturedPayment's order-id
	// binding check, not just the URL scoping, is what protects this.
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"cf_payment_id":888,"order_id":"cf_order_someone_elses","payment_status":"SUCCESS","payment_amount":300.00,"payment_group":"upi"}]`))
	})

	settled := reconcileOrderPayments(db, now)
	require.Equal(t, 0, settled)

	paymentStatus, _ := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentPending), paymentStatus)
}

// ── Scenario 5: Cashfree gateway HTTP 500 → skipped, retried next tick ──────

func TestOrderPaymentReconcile_Cashfree_GatewayError_SkippedAndRetried(t *testing.T) {
	db := setupReconcileDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_erroring", models.ChefModeLive, now.Add(-reconcileGraceElapsed))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	})

	settled := reconcileOrderPayments(db, now)
	require.Equal(t, 0, settled)
	paymentStatus, _ := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentPending), paymentStatus)

	// Retried on the next tick against the same still-failing gateway.
	settled = reconcileOrderPayments(db, now)
	require.Equal(t, 0, settled)
	paymentStatus, _ = reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentPending), paymentStatus)
}

// ── Scenario 7: grace window — inside grace, zero gateway hits ──────────────

func TestOrderPaymentReconcile_InsideGraceWindow_NeverAsksTheGateway(t *testing.T) {
	db := setupReconcileDB(t)
	now := time.Now()
	// Created just now — well inside the 5-minute grace window.
	o := seedStaleOrder(t, db, "cashfree", "cf_order_fresh", models.ChefModeLive, now)

	var hits int32
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`[{"cf_payment_id":999,"order_id":"cf_order_fresh","payment_status":"SUCCESS","payment_amount":300.00,"payment_group":"upi"}]`))
	})

	settled := reconcileOrderPayments(db, now)
	require.Equal(t, 0, settled, "inside the grace window — the SQL filter excludes it before any gateway call")
	require.Equal(t, int32(0), atomic.LoadInt32(&hits), "zero gateway hits proves the created_at filter, not just a would-have-rejected check")

	paymentStatus, _ := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentPending), paymentStatus)
}

// ── Scenario 8: provider routing, both directions ────────────────────────────

func TestOrderPaymentReconcile_CashfreeOrder_NeverRoutedToRazorpay(t *testing.T) {
	db := setupReconcileDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_routing", models.ChefModeLive, now.Add(-reconcileGraceElapsed))
	o.Total = 300

	var razorpayHits, cashfreeHits int32
	withRazorpayServerFor(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&razorpayHits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&cashfreeHits, 1)
		_, _ = w.Write([]byte(`[{"cf_payment_id":1010,"order_id":"cf_order_routing","payment_status":"SUCCESS","payment_amount":300.00,"payment_group":"upi"}]`))
	})

	settled := reconcileOrderPayments(db, now)
	require.Equal(t, 1, settled)
	require.Equal(t, int32(0), atomic.LoadInt32(&razorpayHits), "a cashfree order must never hit the razorpay gateway")
	require.Equal(t, int32(1), atomic.LoadInt32(&cashfreeHits))

	paymentStatus, _ := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentCompleted), paymentStatus)
}

// ── Scenario 9: the historical cancelled-order backfill boundary ────────────
//
// A row shaped exactly like the pre-step-1 stale cron's wrongful-cancel target
// must never be selected, let alone settled — that backfill is a separate,
// owner-gated REFUND task, not this cron's job. The query predicate
// (payment_status='pending' AND status NOT IN (cancelled, ...)) must exclude
// it before any gateway call is made.
func TestOrderPaymentReconcile_CancelledOrder_NeverSelectedZeroGatewayHits(t *testing.T) {
	db := setupReconcileDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "razorpay", "order_rzp_cancelled_backfill", models.ChefModeLive, now.Add(-reconcileGraceElapsed))
	require.NoError(t, db.Exec(
		`UPDATE orders SET status = ?, payment_status = ?, cancel_reason = ? WHERE id = ?`,
		string(models.OrderStatusCancelled), string(models.PaymentFailed), "payment not completed", o.ID.String(),
	).Error)

	var hits int32
	withRazorpayServerFor(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`{"items":[{"id":"pay_x","order_id":"order_rzp_cancelled_backfill","status":"captured","amount":30000,"method":"upi"}]}`))
	})

	settled := reconcileOrderPayments(db, now)
	require.Equal(t, 0, settled)
	require.Equal(t, int32(0), atomic.LoadInt32(&hits), "excluded by the query predicate before any gateway call — never merely skipped after asking")

	paymentStatus, _ := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentFailed), paymentStatus, "untouched — not this cron's job")
}

// Note: a "top-level bail when nothing is configured at all" scenario is
// deliberately NOT exercised here. Calling GetRazorpayFor(live) with an empty
// cache and no Secret Manager falls into a pre-existing dev-fallback branch
// (services/razorpay.go's fetchRazorpayFromSM) that dereferences the
// package-global config.AppConfig — nil in this package's test binary unless
// another test happens to have set it. That is a latent gap in a file outside
// this task's scope (services/razorpay.go isn't in files_modified), not
// something this cron introduces, so it is left for a separate fix rather than
// patched here. Every scenario above that touches Razorpay seeds the live slot
// via withRazorpayServerFor specifically to stay clear of that branch, exactly
// as the sibling stale_order_cron_test.go's own coverage does.

// ── #1086: a Razorpay order is not this cron's business any more ────────────
//
// No order can be captured on Razorpay since #1101, so there is nothing left to
// discover — and asking would be worse than useless: the answer could only
// settle an order against a rail the platform no longer operates.
func TestOrderPaymentReconcile_RazorpayOrder_LeftAloneWithZeroGatewayHits(t *testing.T) {
	db := setupReconcileDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "razorpay", "order_rzp_legacy", models.ChefModeLive, now.Add(-reconcileGraceElapsed))
	o.Total = 300

	var razorpayHits int32
	withRazorpayServerFor(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&razorpayHits, 1)
		_, _ = w.Write([]byte(`{"items":[{"id":"pay_legacy","order_id":"order_rzp_legacy","status":"captured","amount":30000,"method":"upi"}]}`))
	})
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})

	settled := reconcileOrderPayments(db, now)
	require.Equal(t, 0, settled)
	require.Equal(t, int32(0), atomic.LoadInt32(&razorpayHits), "the retired gateway must never be asked")

	paymentStatus, gatewayPaymentID := reconcilePaymentRow(t, db, o.ID)
	require.Equal(t, string(models.PaymentPending), paymentStatus, "a legacy order is left exactly as it was")
	require.Empty(t, gatewayPaymentID)
}
