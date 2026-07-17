package services

// unaccepted_order_cron_test.go — #694. The paid-but-unaccepted sweep.
//
// The hole: a customer pays, the money is captured, no chef accepts, and NOTHING
// ever happens. These pin that the sweep voids those orders once THE CHEF'S
// KITCHEN CLOSES for the meal they were placed for — refunding to the provider
// the customer actually paid with — and, just as importantly, that it never
// cancels an order it failed to refund, and never touches an order whose kitchen
// is still open.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedPaidPendingOrder is the exact stranded state: money captured, still waiting
// on a chef, for the meal at `serviceDay` (its hour decides lunch vs dinner via
// inferSlot).
//
// delivery_slot is left empty on purpose: setting it makes the void path call
// ReleaseSlot, whose SQL uses GREATEST — Postgres-only, and sqlite has no such
// function. That is an environment limit, not a behaviour the sweep needs to
// prove; slot windows and slot inference are pinned in
// order_accept_deadline_test.go, and these tests are about the void, the refund
// and the deadline.
func seedPaidPendingOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, _slot string, serviceDay, created time.Time) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-STRAND", CustomerID: uuid.New(), ChefID: chefID,
		Status: models.OrderStatusPending, PaymentStatus: models.PaymentCompleted,
		PaymentProvider: "razorpay", RazorpayPaymentID: "pay_strand", Total: 250,
	}
	require.NoError(t, db.Exec(`INSERT INTO orders
		(id, order_number, customer_id, chef_id, status, payment_status, payment_provider,
		 razorpay_payment_id, total, refund_amount, scheduled_for, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,0,?,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), chefID.String(),
		string(models.OrderStatusPending), string(models.PaymentCompleted), "razorpay",
		"pay_strand", 250.0, serviceDay, created, created).Error)
	return o
}

// setupUnacceptedDB is a chef whose lunch service ends at 14:00 and dinner at 22:00.
func setupUnacceptedDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db, chefID := setupDeadlineDB(t)
	seedCapacity(t, db, chefID, "14:00", "22:00")
	return db, chefID
}

// razorpayOK stands in for a working gateway.
func razorpayOK(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"id":"rfnd_sweep","status":"processed"}`))
	}))
	t.Cleanup(srv.Close)
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })
}

func orderRow(t *testing.T, db *gorm.DB, id uuid.UUID) (status, paymentStatus string, refund float64) {
	t.Helper()
	row := struct {
		Status        string
		PaymentStatus string
		RefundAmount  float64
	}{}
	require.NoError(t, db.Raw(
		`SELECT status, payment_status, refund_amount FROM orders WHERE id = ?`, id.String(),
	).Scan(&row).Error)
	return row.Status, row.PaymentStatus, row.RefundAmount
}

// THE bug: money captured, nobody accepted, and before #694 this order sat here
// forever with no refund and no timeout.
func TestUnacceptedSweep_VoidsAnOrderTheKitchenClosedOn(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	lunchDay := ist(2026, 7, 20, 12, 0)
	o := seedPaidPendingOrder(t, db, chefID, "lunch", lunchDay, ist(2026, 7, 20, 9, 0))

	// 14:05 — lunch service ended at 14:00 and nobody accepted.
	require.Equal(t, 1, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 20, 14, 5)))

	status, payment, refund := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusCancelled), status)
	require.Equal(t, string(models.PaymentRefunded), payment)
	require.Equal(t, 250.0, refund, "the customer gets their money back, not a shrug")
}

// The grace window must be respected — a chef who is 10 minutes slow has not
// abandoned anything.
func TestUnacceptedSweep_LeavesOrdersWhoseKitchenIsStillOpen(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	lunchDay := ist(2026, 7, 20, 12, 0)
	o := seedPaidPendingOrder(t, db, chefID, "lunch", lunchDay, ist(2026, 7, 20, 9, 0))

	// 13:30 — the chef still has half an hour of lunch service.
	require.Equal(t, 0, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 20, 13, 30)))

	status, _, refund := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status, "still the chef's to accept")
	require.Equal(t, 0.0, refund)
}

// THE bug the old 30-minutes-after-payment model caused: an ADVANCE order would
// have been voided and refunded half an hour after checkout, long before the chef
// could plausibly have looked at it.
func TestUnacceptedSweep_NeverVoidsAnAdvanceOrderEarly(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	placed := ist(2026, 7, 20, 10, 0)
	tomorrowDinner := ist(2026, 7, 21, 20, 0)
	o := seedPaidPendingOrder(t, db, chefID, "dinner", tomorrowDinner, placed)

	// Hours later on the day it was PLACED — the old rule would have killed it.
	require.Equal(t, 0, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 20, 18, 0)),
		"tomorrow's dinner is not abandoned because 8 hours passed today")
	status, _, refund := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status)
	require.Equal(t, 0.0, refund, "refunding this would be taking a booking and cancelling it unasked")

	// And it IS voided once TOMORROW's dinner service ends.
	require.Equal(t, 1, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 21, 22, 5)))
	_, _, refund = orderRow(t, db, o.ID)
	require.Equal(t, 250.0, refund)
}

// Only PAID + PENDING orders are the money hole. An accepted order is being
// cooked; an unpaid one is stale_order_cron's job.
func TestUnacceptedSweep_IgnoresOrdersThatAreNotStranded(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	lunchDay := ist(2026, 7, 20, 12, 0)
	placed := ist(2026, 7, 20, 9, 0)

	accepted := seedPaidPendingOrder(t, db, chefID, "lunch", lunchDay, placed)
	require.NoError(t, db.Exec(`UPDATE orders SET status = ? WHERE id = ?`,
		string(models.OrderStatusAccepted), accepted.ID.String()).Error)

	unpaid := seedPaidPendingOrder(t, db, chefID, "lunch", lunchDay, placed)
	require.NoError(t, db.Exec(`UPDATE orders SET payment_status = ? WHERE id = ?`,
		string(models.PaymentPending), unpaid.ID.String()).Error)

	require.Equal(t, 0, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 20, 14, 5)))

	s, _, _ := orderRow(t, db, accepted.ID)
	require.Equal(t, string(models.OrderStatusAccepted), s, "a chef is cooking this — never touch it")
	s, _, r := orderRow(t, db, unpaid.ID)
	require.Equal(t, string(models.OrderStatusPending), s)
	require.Equal(t, 0.0, r, "nothing was captured, so there is nothing to refund")
}

// The order that matters most: a gateway failure must NEVER leave a cancelled
// order the customer was not refunded for. That state looks resolved and is not.
func TestUnacceptedSweep_GatewayFailure_LeavesTheOrderPendingForRetry(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":{"description":"boom"}}`))
	}))
	defer srv.Close()
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })

	o := seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))

	require.Equal(t, 0, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 20, 14, 5)))

	status, payment, refund := orderRow(t, db, o.ID)
	require.Equal(t, string(models.OrderStatusPending), status,
		"cancelling an order we could not refund is the worst state there is — it LOOKS resolved")
	require.Equal(t, string(models.PaymentCompleted), payment, "the reservation was released for retry")
	require.Equal(t, 0.0, refund)
}

// A retry after a failure must refund exactly once.
func TestUnacceptedSweep_RetryAfterFailure_RefundsOnce(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	fail := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":{"description":"boom"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"rfnd_sweep","status":"processed"}`))
	}))
	defer srv.Close()
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })

	o := seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))
	after := ist(2026, 7, 20, 14, 5)
	require.Equal(t, 0, refundUnacceptedOrders(context.Background(), db, after))

	fail = false
	require.Equal(t, 1, refundUnacceptedOrders(context.Background(), db, after))

	_, _, refund := orderRow(t, db, o.ID)
	require.Equal(t, 250.0, refund, "refunded once, in full — not twice")
}

// A refunded order must not be swept again on the next tick.
func TestUnacceptedSweep_IsIdempotentAcrossTicks(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	o := seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))
	after := ist(2026, 7, 20, 14, 5)

	require.Equal(t, 1, refundUnacceptedOrders(context.Background(), db, after))
	require.Equal(t, 0, refundUnacceptedOrders(context.Background(), db, after),
		"it is cancelled now — a second tick must not see it at all")

	_, _, refund := orderRow(t, db, o.ID)
	require.Equal(t, 250.0, refund, "still exactly one refund")
}

// The customer must be told. Being refunded silently reads as "they took my
// money and cancelled my dinner".
func TestUnacceptedSweep_StagesTheVoidEvent(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))

	require.Equal(t, 1, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 20, 14, 5)))

	var n int64
	require.NoError(t, db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`,
		SubjectOrderVoided).Scan(&n).Error)
	require.Equal(t, int64(1), n,
		"orders.voided, staged in the same tx: nobody chose this, so the customer is owed "+
			"an apology as well as their money — and a crash must not swallow it")
}

// A meal-plan day / group order is refund-managed by its own escrow flow on a
// disjoint keyspace. The generic path must not touch it — refunding here would
// pay the customer twice.
func TestUnacceptedSweep_SkipsTypedEscrowOrders(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	razorpayOK(t)
	o := seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), o.ID.String(), "confirmed").Error)

	require.Equal(t, 0, refundUnacceptedOrders(context.Background(), db, ist(2026, 7, 20, 14, 5)))

	_, _, refund := orderRow(t, db, o.ID)
	require.Equal(t, 0.0, refund, "the meal-plan flow refunds this, on its own keyspace")
}
