package services

// stale_order_cron_test.go — #872 step 1. Pins the load-bearing rule of the
// gateway gate: an unknown answer from the gateway must NEVER result in a
// cancellation. Gateway error, gateway nil/unconfigured for that mode, an
// unrecognised provider — all skip the row and retry next tick. Only a gateway
// CONFIRMING "not captured" (or an order that never got a gateway order id at
// all) may reach the cancel transaction.
//
// Capacity/slot-release table assertions are intentionally NOT made here, for
// the same reason unaccepted_order_cron_test.go's seedPaidPendingOrder avoids
// them: ReleaseCapacity/ReleaseSlot's SQL uses Postgres' GREATEST, which sqlite
// has no equivalent for, so any order seeded with items or a delivery slot
// would fail the transaction outright, not release anything. These tests seed
// zero items and no delivery slot — same as that file — and assert on the
// order row's own field writes, which is what the cancel branch actually
// changed in #872.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// setupStaleOrderDB extends setupCancelRefundDB's orders table with
// razorpay_order_id — the column this cron reads to decide whether an order was
// ever handed to a gateway at all.
func setupStaleOrderDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupCancelRefundDB(t)
	require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN razorpay_order_id TEXT DEFAULT ''`).Error)
	return db
}

// seedStaleOrder is a payment_status=pending order created well past the
// 30-minute grace window, stamped with the given provider/gateway-order-id/mode.
// No items, no delivery slot — see the file header on why.
func seedStaleOrder(t *testing.T, db *gorm.DB, provider, gatewayOrderID, mode string, created time.Time) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-STALE", CustomerID: uuid.New(), ChefID: uuid.New(),
		Status: models.OrderStatusPending, PaymentStatus: models.PaymentPending,
		PaymentProvider: provider, RazorpayOrderID: gatewayOrderID, Total: 300, ModePartition: models.ModePartition{Mode: mode},
	}
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, payment_provider,
		 razorpay_order_id, total, mode, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), o.ChefID.String(),
		string(models.OrderStatusPending), string(models.PaymentPending), provider,
		gatewayOrderID, o.Total, mode, created, created).Error)
	return o
}

func staleOrderRow(t *testing.T, db *gorm.DB, id uuid.UUID) (status, paymentStatus, cancelReason string, cancelledAt *time.Time) {
	t.Helper()
	row := struct {
		Status        string
		PaymentStatus string
		CancelReason  string
		CancelledAt   *time.Time
	}{}
	require.NoError(t, db.Raw(
		`SELECT status, payment_status, cancel_reason, cancelled_at FROM orders WHERE id = ?`, id.String(),
	).Scan(&row).Error)
	return row.Status, row.PaymentStatus, row.CancelReason, row.CancelledAt
}

// withRazorpayServerFor points a mode's Razorpay slot at an httptest.Server and
// restores the previous occupant, mirroring withCashfreeServer (cashfree_test.go)
// for the mode Razorpay's own test helpers don't cover.
func withRazorpayServerFor(t *testing.T, mode string, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	prev := snapshotRazorpayClient(mode)
	t.Cleanup(func() { SetRazorpayClientFor(mode, prev) })
	SetRazorpayClientFor(mode, NewRazorpayTestClient(srv.URL, "rzp_test", "secret_test", "whsec_test"))
	return srv
}

const staleOrderGrace = 45 * time.Minute // safely past the 30-minute threshold

// ── The pure decision seam ──────────────────────────────────────────────────

// decideStaleOrderAction is the load-bearing rule of #872, exercised directly
// against every branch of the decision table: an unknown gateway answer must
// never reach "cancel".
func TestDecideStaleOrderAction_DecisionTable(t *testing.T) {
	for _, tc := range []struct {
		name              string
		hasGatewayOrderID bool
		state             gatewayPaymentState
		gatewayErr        error
		want              staleOrderAction
	}{
		{"no gateway order id ever stamped -> cancel, unasked", false, gatewayNoPayment, nil, staleOrderCancel},
		{"gateway error -> never cancel, skip", true, gatewayNoPayment, context.DeadlineExceeded, staleOrderSkipError},
		{"captured payment found -> skip, do not settle", true, gatewayCaptured, nil, staleOrderSkipCaptured},
		{"attempt still in flight -> skip, never cancel under a live charge", true, gatewayInFlight, nil, staleOrderSkipInFlight},
		{"gateway confirms every attempt is dead -> cancel", true, gatewayNoPayment, nil, staleOrderCancel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := decideStaleOrderAction(tc.hasGatewayOrderID, tc.state, tc.gatewayErr)
			require.Equal(t, tc.want, got)
		})
	}
}

// ── Scenario 1: captured payment -> stays pending, never settled ───────────

func TestStaleOrderSweep_CapturedPayment_StaysPendingAndIsNotSettled(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_captured", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"cf_payment_id":555,"order_id":"cf_order_captured","payment_status":"SUCCESS","payment_amount":300.00,"payment_group":"upi"}]`))
	})

	expired, skippedCaptured, _, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired)
	require.Equal(t, 1, skippedCaptured)
	require.Equal(t, 0, skippedError)

	status, paymentStatus, cancelReason, cancelledAt := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status, "a captured order is never cancelled")
	require.Equal(t, string(models.PaymentPending), paymentStatus, "not settled here — that's the reconcile cron's job")
	require.Empty(t, cancelReason)
	require.Nil(t, cancelledAt)
}

// ── Scenario 2: gateway confirms not captured -> cancelled exactly as before ─

func TestStaleOrderSweep_NotCaptured_CancelsExactlyAsBefore(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_notcaptured", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[]`))
	})

	expired, skippedCaptured, _, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 1, expired)
	require.Equal(t, 0, skippedCaptured)
	require.Equal(t, 0, skippedError)

	status, paymentStatus, cancelReason, cancelledAt := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusCancelled), status)
	require.Equal(t, string(models.PaymentFailed), paymentStatus)
	require.Equal(t, "payment not completed", cancelReason)
	require.NotNil(t, cancelledAt)
}

// ── Scenario 3: gateway fetch error -> never cancel, retried next tick ──────

func TestStaleOrderSweep_GatewayError_NeverCancelsAndIsRetried(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_erroring", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	})

	expired, skippedCaptured, _, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired)
	require.Equal(t, 0, skippedCaptured)
	require.Equal(t, 1, skippedError)

	status, _, cancelReason, cancelledAt := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status,
		"an unknown answer must never reach the cancel transaction")
	require.Empty(t, cancelReason)
	require.Nil(t, cancelledAt)

	// The still-failing gateway must not cancel it on a second tick either.
	expired, skippedCaptured, _, skippedError = runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired)
	require.Equal(t, 0, skippedCaptured)
	require.Equal(t, 1, skippedError)

	status, _, _, _ = staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status, "still retried, still not cancelled")
}

// ── Scenario 4: gateway not configured for the order's mode -> never cancel ─

func TestStaleOrderSweep_GatewayNotConfiguredForMode_NeverCancels(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_noconfig", models.ChefModeTest, now.Add(-staleOrderGrace))

	// Explicitly leave the TEST slot empty — no SetCashfreeClientFor(test, …)
	// call for this test — and restore whatever was there afterward so this
	// assertion can never leak into a sibling test.
	prev := snapshotCashfreeClient(models.ChefModeTest)
	SetCashfreeClientFor(models.ChefModeTest, nil)
	t.Cleanup(func() { SetCashfreeClientFor(models.ChefModeTest, prev) })

	expired, skippedCaptured, _, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired)
	require.Equal(t, 0, skippedCaptured)
	require.Equal(t, 1, skippedError)

	status, _, cancelReason, cancelledAt := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status,
		"an unconfigured gateway slot is an unknown answer, not a captured=false")
	require.Empty(t, cancelReason)
	require.Nil(t, cancelledAt)
}

// ── Scenario 5: no gateway order id ever stamped -> zero gateway calls ──────

func TestStaleOrderSweep_NoGatewayOrderID_CancelsWithZeroGatewayCalls(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "", models.ChefModeLive, now.Add(-staleOrderGrace))

	var hits int32
	// Installed deliberately, in the LIVE slot the order's own mode would use,
	// so a regression that asks the gateway anyway is caught even though a
	// client IS configured — an empty gateway order id must short-circuit
	// before any client lookup, not because none happens to be configured.
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&hits, 1)
		_, _ = w.Write([]byte(`[]`))
	})

	expired, skippedCaptured, _, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 1, expired)
	require.Equal(t, 0, skippedCaptured)
	require.Equal(t, 0, skippedError)
	require.Equal(t, int32(0), atomic.LoadInt32(&hits), "an unstamped order must never reach the gateway")

	status, paymentStatus, cancelReason, cancelledAt := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusCancelled), status)
	require.Equal(t, string(models.PaymentFailed), paymentStatus)
	require.Equal(t, "payment not completed", cancelReason)
	require.NotNil(t, cancelledAt)
}

// ── Scenario 6: provider routing — never cross-checked against the wrong gateway ─

// A Cashfree order is confirmed via the Cashfree server; the Razorpay slot is
// left unconfigured (would error if ever consulted), and a hit on it fails the
// test.
func TestStaleOrderSweep_CashfreeOrder_NeverRoutedToRazorpay(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_routing", models.ChefModeLive, now.Add(-staleOrderGrace))

	var razorpayHits, cashfreeHits int32
	withRazorpayServerFor(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&razorpayHits, 1)
		w.WriteHeader(http.StatusInternalServerError)
	})
	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&cashfreeHits, 1)
		_, _ = w.Write([]byte(`[{"cf_payment_id":9,"order_id":"cf_order_routing","payment_status":"SUCCESS","payment_amount":300.00,"payment_group":"upi"}]`))
	})

	expired, skippedCaptured, _, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired)
	require.Equal(t, 1, skippedCaptured)
	require.Equal(t, 0, skippedError)
	require.Equal(t, int32(0), atomic.LoadInt32(&razorpayHits), "a cashfree order must never hit the razorpay gateway")
	require.Equal(t, int32(1), atomic.LoadInt32(&cashfreeHits))

	status, _, _, _ := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status)
}

// #1086 — a Razorpay order is an unrecognised provider now. The sweep's
// backstop must hold: an unknown answer is never a cancel, and the retired
// gateway is never asked.
func TestStaleOrderSweep_RazorpayOrder_NeverCancelledAndNeverAsked(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "razorpay", "order_rzp_legacy", models.ChefModeLive, now.Add(-staleOrderGrace))

	var razorpayHits int32
	withRazorpayServerFor(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&razorpayHits, 1)
		_, _ = w.Write([]byte(`{"items":[]}`))
	})

	expired, _, _, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired, "an order the sweep cannot ask about is never cancelled")
	require.Equal(t, 1, skippedError)
	require.Equal(t, int32(0), atomic.LoadInt32(&razorpayHits), "the retired gateway must never be asked")

	status, _, _, _ := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status)
}

// ── Scenario 7: an attempt still in flight -> never cancel ─────────────────
//
// The defect this pins, observed on production sandbox orders HC26080400184814
// and HC26080400303019 on 4 Aug 2026: Cashfree held payment 5114933571626 at
// PENDING (bank OTP page open, is_captured false). SuccessfulPayment returns nil
// for PENDING exactly as it does for "no attempt at all", so the sweep read
// "nothing was captured" and cancelled. When that payment later resolves to
// SUCCESS the customer has been charged for a cancelled order — and the
// order-payment reconcile cron is forward-only by design (it excludes
// status=cancelled rows), so nothing recovers it.

func TestStaleOrderSweep_CashfreePendingPayment_NeverCancels(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_pending", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		// The exact shape Cashfree returned for the stranded order.
		_, _ = w.Write([]byte(`[{"cf_payment_id":5114933571626,"order_id":"cf_order_pending",
			"payment_status":"PENDING","payment_amount":393.05,"payment_group":"debit_card",
			"is_captured":false,"payment_message":"Simulated response message"}]`))
	})

	expired, skippedCaptured, skippedInFlight, skippedError := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired, "a live attempt must never be cancelled")
	require.Equal(t, 0, skippedCaptured)
	require.Equal(t, 1, skippedInFlight)
	require.Equal(t, 0, skippedError)

	status, paymentStatus, cancelReason, cancelledAt := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status)
	require.Equal(t, string(models.PaymentPending), paymentStatus)
	require.Empty(t, cancelReason)
	require.Nil(t, cancelledAt)
}

// A dead attempt still cancels — the fix must not turn the sweep into a no-op.
func TestStaleOrderSweep_CashfreeAllAttemptsDead_StillCancels(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_dead", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"cf_payment_id":1,"order_id":"cf_order_dead","payment_status":"FAILED","payment_amount":393.05},
			{"cf_payment_id":2,"order_id":"cf_order_dead","payment_status":"USER_DROPPED","payment_amount":393.05}
		]`))
	})

	expired, _, skippedInFlight, _ := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 1, expired)
	require.Equal(t, 0, skippedInFlight)

	status, paymentStatus, cancelReason, _ := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusCancelled), status)
	require.Equal(t, string(models.PaymentFailed), paymentStatus)
	require.Equal(t, "payment not completed", cancelReason)
}

// A mixed list — one dead attempt, one live retry — is in flight, not dead. The
// customer who failed once and is mid-way through a second attempt is precisely
// who must not have their order cancelled under them.
func TestStaleOrderSweep_CashfreeFailedThenPending_IsInFlight(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_retry", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
			{"cf_payment_id":1,"order_id":"cf_order_retry","payment_status":"FAILED","payment_amount":393.05},
			{"cf_payment_id":2,"order_id":"cf_order_retry","payment_status":"PENDING","payment_amount":393.05}
		]`))
	})

	expired, _, skippedInFlight, _ := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired)
	require.Equal(t, 1, skippedInFlight)

	status, _, _, _ := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status)
}

// Razorpay's `authorized` is money already held on the customer's card. It is
// not `captured`, so the old probe returned "" for it and cancelled — the worst
// version of this bug, since the hold is real money.
// An unrecognised gateway state reads as in-flight, never as dead. A status
// neither gateway has shipped yet must not be able to cancel an order.
func TestStaleOrderSweep_UnknownGatewayStatus_TreatedAsInFlight(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	seedStaleOrder(t, db, "cashfree", "cf_order_novel", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"cf_payment_id":9,"order_id":"cf_order_novel",
			"payment_status":"AWAITING_MANDATE_APPROVAL","payment_amount":393.05}]`))
	})

	expired, _, skippedInFlight, _ := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 0, expired)
	require.Equal(t, 1, skippedInFlight)
}

// NOT_ATTEMPTED is the ordinary abandoned checkout: a session was created and the
// customer never started. It MUST still cancel, or the chef's reserved capacity is
// held until the gateway session expires — Cashfree's 30-day default, since
// order_expiry_time is never set. Caught by a production canary
// (HC26080404499485) after the first version of the D-14 fix swept NOT_ATTEMPTED
// into the "unknown, therefore in flight" default.
func TestStaleOrderSweep_CashfreeNotAttempted_StillCancels(t *testing.T) {
	db := setupStaleOrderDB(t)
	now := time.Now()
	o := seedStaleOrder(t, db, "cashfree", "cf_order_unstarted", models.ChefModeLive, now.Add(-staleOrderGrace))

	withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"cf_payment_id":5114933580135,"order_id":"cf_order_unstarted",
			"payment_status":"NOT_ATTEMPTED","payment_amount":393.05,"is_captured":false}]`))
	})

	expired, _, skippedInFlight, _ := runStaleOrderScanWithDB(context.Background(), db, now)
	require.Equal(t, 1, expired, "an unstarted session is an abandoned checkout, not a live payment")
	require.Equal(t, 0, skippedInFlight)

	status, _, cancelReason, _ := staleOrderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusCancelled), status)
	require.Equal(t, "payment not completed", cancelReason)
}

// VOID and CANCELLED are likewise terminal — nothing can move money again.
func TestStaleOrderSweep_CashfreeVoidAndCancelled_StillCancel(t *testing.T) {
	for _, st := range []string{"VOID", "CANCELLED"} {
		t.Run(st, func(t *testing.T) {
			db := setupStaleOrderDB(t)
			now := time.Now()
			seedStaleOrder(t, db, "cashfree", "cf_order_"+st, models.ChefModeLive, now.Add(-staleOrderGrace))

			withCashfreeServer(t, models.ChefModeLive, func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`[{"cf_payment_id":1,"order_id":"cf_order_` + st + `",
					"payment_status":"` + st + `","payment_amount":393.05}]`))
			})

			expired, _, skippedInFlight, _ := runStaleOrderScanWithDB(context.Background(), db, now)
			require.Equal(t, 1, expired)
			require.Equal(t, 0, skippedInFlight)
		})
	}
}
