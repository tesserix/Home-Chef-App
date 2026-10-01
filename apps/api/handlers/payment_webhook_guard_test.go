package handlers

// payment_webhook_guard_test.go — #563. The Stripe webhook handlers were UNCONDITIONAL
// updates keyed only on the payment-intent id, so a duplicate/out-of-order delivery could
// re-stamp a refunded/completed order (payment_intent.succeeded on a refunded order →
// completed re-enables the chef payout on refunded money). Both are now guarded, and the
// the retired gateway capture guard was tightened to also exclude `refunded`.

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setStripeIntent(t *testing.T, db *gorm.DB, orderID uuid.UUID, piID string) {
	t.Helper()
	require.NoError(t, db.Exec(`UPDATE orders SET stripe_payment_intent_id = ?, payment_provider = 'stripe', gateway_order_id = '' WHERE id = ?`,
		piID, orderID.String()).Error)
}

func stripeIntentPayload(t *testing.T, piID string) json.RawMessage {
	t.Helper()
	b, err := json.Marshal(services.StripePaymentIntent{ID: piID, Amount: 50000, AmountReceived: 50000, Livemode: true, Currency: "inr", Status: "succeeded"})
	require.NoError(t, err)
	return b
}

// A payment_intent.succeeded replay on a REFUNDED order must not re-complete it (#563).
func TestHandleStripePaymentSucceeded_DoesNotReStampRefunded(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "refunded", 500, "", "")
	setStripeIntent(t, db, orderID, "pi_ref")

	(&PaymentHandler{}).handleStripePaymentSucceeded(t.Context(), stripeIntentPayload(t, "pi_ref"))

	require.Equal(t, string(models.PaymentRefunded), paymentStatusOf(t, db, orderID),
		"a refunded order survives a duplicate payment_intent.succeeded")
	require.Equal(t, int64(0), countOutbox(t, db, services.SubjectChefNewOrder), "no chef push on the no-op")
}

// A pending order IS completed by the webhook (the legitimate first delivery).
func TestHandleStripePaymentSucceeded_CompletesPending(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "", "")
	setStripeIntent(t, db, orderID, "pi_ok")

	(&PaymentHandler{}).handleStripePaymentSucceeded(t.Context(), stripeIntentPayload(t, "pi_ok"))

	require.Equal(t, string(models.PaymentCompleted), paymentStatusOf(t, db, orderID))
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder), "chef notified once on the transition")
}

// Retry-after-decline at the webhook level: a payment_intent.succeeded on a FAILED order
// (the customer retried after a decline) DOES complete it — `failed` is not blocked.
func TestHandleStripePaymentSucceeded_CompletesFailedOnRetry(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "failed", 500, "", "")
	setStripeIntent(t, db, orderID, "pi_retry")

	(&PaymentHandler{}).handleStripePaymentSucceeded(t.Context(), stripeIntentPayload(t, "pi_retry"))

	require.Equal(t, string(models.PaymentCompleted), paymentStatusOf(t, db, orderID),
		"a retry after a decline completes the order")
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder))
}

// A capture on a FAILED order (the customer retried and it went through)
// completes it too.
func TestHandleCashfreePaymentSuccess_CompletesFailedOnRetry(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "failed", 500, "cf_cap_retry")

	require.NoError(t, (&PaymentHandler{}).handleCashfreePaymentSuccess(
		capturedPayload("cf_cap_retry", "8801", 500), models.ChefModeLive))

	require.Equal(t, string(models.PaymentCompleted), paymentStatusOf(t, db, orderID))
}

// payment_intent.payment_failed must not overwrite a COMPLETED order (#563).
func TestHandleStripePaymentFailed_DoesNotOverwriteCompleted(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "completed", 500, "", "")
	setStripeIntent(t, db, orderID, "pi_done")

	(&PaymentHandler{}).handleStripePaymentFailed(stripeIntentPayload(t, "pi_done"))

	require.Equal(t, string(models.PaymentCompleted), paymentStatusOf(t, db, orderID),
		"a completed order survives a stray payment_intent.payment_failed")
}

// A capture on a REFUNDED order must not re-stamp it completed (tightened guard).
func TestHandleCashfreePaymentSuccess_DoesNotReStampRefunded(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "refunded", 500, "cf_cap_ref")
	// The guarded UPDATE matches nothing, so the handler falls through to the
	// FSSAI fallback — a real query that needs its table present.
	addFssaiRequestsTable(t, db)

	require.NoError(t, (&PaymentHandler{}).handleCashfreePaymentSuccess(
		capturedPayload("cf_cap_ref", "8802", 500), models.ChefModeLive))

	require.Equal(t, string(models.PaymentRefunded), paymentStatusOf(t, db, orderID))
}
