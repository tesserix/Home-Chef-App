package services

// cancellation_order_refund_typed_test.go — #544. RefundOrderForCancellation (the shared
// full-refund mover for customer CancelOrder + chef reject) must NEVER issue a generic refund
// on an order spawned by a typed escrow flow (meal-plan day / group order). Those are refunded
// through their own flows (RefundDay / participant refunds) on a DISJOINT idempotency keyspace,
// and their held chef payout is a DIRECT transfer the generic path can't reverse — so a generic
// refund would double-pay the customer (the moment a gateway id lands on the row) while the chef
// keeps the transfer. Mirrors the #394 guard already on InitiateRefund.

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// seedTypedRefundOrder seeds a paid Razorpay order that WOULD be generically refundable
// (payment_status=completed + a razorpay payment id set), so the only thing stopping the
// gateway call is the typed-order guard under test.
func seedTypedRefundOrder(t *testing.T, db *gorm.DB) *models.Order {
	t.Helper()
	o := &models.Order{
		ID: uuid.New(), OrderNumber: "ORD-MP", CustomerID: uuid.New(), ChefID: uuid.New(),
		Status: models.OrderStatusCancelled, PaymentStatus: models.PaymentCompleted,
		PaymentProvider: "razorpay", RazorpayPaymentID: "pay_typed", Total: 300,
	}
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		payment_provider, razorpay_payment_id, total, refund_amount) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		o.ID.String(), o.OrderNumber, o.CustomerID.String(), o.ChefID.String(), string(o.Status),
		string(models.PaymentCompleted), "razorpay", "pay_typed", 300.0, 0.0).Error)
	return o
}

// assertNoGenericRefund runs RefundOrderForCancellation with a gateway that flags any call, and
// asserts nothing moved: no gateway hit, payment_status stays completed, refund_amount stays 0.
func assertNoGenericRefund(t *testing.T, db *gorm.DB, o *models.Order) {
	t.Helper()
	gatewayCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayCalled = true
		_, _ = w.Write([]byte(`{"id":"rfnd_x","status":"processed"}`))
	}))
	defer srv.Close()
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	require.False(t, gatewayCalled, "a typed escrow order must NOT be refunded via the generic gateway path")
	ps, amt, rid, _ := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentCompleted), ps, "payment_status stays completed — no generic refund claim on a typed order")
	require.Equal(t, 0.0, amt, "no refund_amount written by the generic path")
	require.Empty(t, rid, "no gateway refund id recorded")
}

func TestRefundOrderForCancellation_SkipsTypedMealPlanDayOrder(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedTypedRefundOrder(t, db)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), o.ID.String(), "active").Error)

	assertNoGenericRefund(t, db, o)
}

func TestRefundOrderForCancellation_SkipsTypedGroupOrder(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedTypedRefundOrder(t, db)
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), o.ID.String(), "active").Error)

	assertNoGenericRefund(t, db, o)
}

// Non-regression: a PLAIN paid order (no meal_plan_days / group_orders back-ref) still refunds
// through the generic path — the guard must not block ordinary cancellations.
func TestRefundOrderForCancellation_PlainOrderStillRefunds(t *testing.T) {
	db := setupCancelRefundDB(t)
	o := seedTypedRefundOrder(t, db) // paid razorpay order, but no typed back-ref inserted

	gatewayCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gatewayCalled = true
		_, _ = w.Write([]byte(`{"id":"rfnd_ok","status":"processed"}`))
	}))
	defer srv.Close()
	SetRazorpayClient(NewRazorpayTestClient(srv.URL, "key", "secret", "whsec"))
	t.Cleanup(func() { SetRazorpayClient(nil) })

	require.NoError(t, RefundOrderForCancellation(o, "customer", "cancel"))

	require.True(t, gatewayCalled, "an ordinary paid order still refunds via the generic gateway path")
	ps, amt, rid, _ := loadRefund(t, db, o.ID)
	require.Equal(t, string(models.PaymentRefunded), ps)
	require.Equal(t, 300.0, amt)
	require.NotEmpty(t, rid)
}
