package handlers

// payment_cashfree_test.go — the Cashfree create/verify/webhook legs driven
// end-to-end against an httptest gateway, with no live Cashfree and no GCP.
//
// The emphasis is on the gates that stop money going wrong, because those are the
// ones a "just add another provider" change quietly omits:
//
//   - verify must BIND the fetched payment to this order and its amount, so a
//     captured payment cannot be replayed onto a different order;
//   - a re-verify (or a verify racing the webhook) must not double-emit order.paid
//     or double-push the chef;
//   - the webhook must reject a forged signature and dedup a replay;
//   - the ids must land in the shared gateway-id columns with provider=cashfree.

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

const cfTestWebhookSecret = "whsec_cashfree_test"

// withCashfreeGateway points the live credential slot at an httptest.Server and
// restores whatever was there before.
func withCashfreeGateway(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	t.Cleanup(func() { services.SetCashfreeClient(nil) })
	services.SetCashfreeClient(
		services.NewCashfreeTestClient(srv.URL, "app_test", "secret_test", cfTestWebhookSecret, models.ChefModeLive))
}

// cfPayOrder inserts a Cashfree order: provider=cashfree with the gateway order
// id in the SHARED gateway_order_id column (models.GatewayOrderIDColumn).
func cfPayOrder(t *testing.T, db *gorm.DB, customerID, chefID uuid.UUID, paymentStatus string, total float64, cfOrderID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status, payment_provider,
			subtotal, tax, total, currency, gateway_order_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'pending', ?, 'cashfree', ?, ?, ?, 'INR', ?, ?, ?)`,
		id.String(), "HC-"+id.String()[:8], customerID.String(), chefID.String(), paymentStatus,
		total*0.9, total*0.1, total, cfOrderID, time.Now(), time.Now()).Error)
	return id
}

// cfPaymentsJSON is the GET /orders/{id}/payments body for one captured payment.
func cfPaymentsJSON(cfOrderID, cfPaymentID string, rupees float64) string {
	return fmt.Sprintf(`[{"cf_payment_id":%s,"order_id":%q,"payment_status":"SUCCESS",
		"payment_amount":%.2f,"payment_currency":"INR","payment_group":"upi"}]`,
		cfPaymentID, cfOrderID, rupees)
}

// cfGateway serves the two reads the verify leg makes.
func cfGateway(cfOrderID, cfPaymentID string, rupees float64, orderStatus string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/payments"):
			_, _ = w.Write([]byte(cfPaymentsJSON(cfOrderID, cfPaymentID, rupees)))
		default:
			_, _ = w.Write([]byte(fmt.Sprintf(
				`{"cf_order_id":1,"order_id":%q,"payment_session_id":"sess_x","order_status":%q,"order_amount":%.2f}`,
				cfOrderID, orderStatus, rupees)))
		}
	}
}

func orderColumn(t *testing.T, db *gorm.DB, id uuid.UUID, col string) string {
	t.Helper()
	var v *string
	require.NoError(t, db.Raw(`SELECT `+col+` FROM orders WHERE id = ?`, id.String()).Scan(&v).Error)
	if v == nil {
		return ""
	}
	return *v
}

// --- Verify ---

// The happy path: a captured Cashfree payment completes the order, stamps the
// cf_payment_id into the shared gateway-payment-id column, and records the method.
func TestVerifyCashfreePayment_CompletesOrder(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-1")

	withCashfreeGateway(t, cfGateway("cf-order-1", "9911", 500, "PAID"))

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-1"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
	require.Equal(t, "9911", orderColumn(t, db, orderID, models.GatewayPaymentIDColumn),
		"cf_payment_id lands in the shared gateway-payment-id column")
	require.Equal(t, "upi", orderColumn(t, db, orderID, "payment_method"))
	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"))
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder))
}

// A client that omits the order id still gets verified against the STAMPED id.
// The stamped value is the trustworthy one, so nothing is lost by not requiring
// the client to echo it — and a checkout that closed without returning it is
// common enough that failing would strand captured money.
func TestVerifyCashfreePayment_FallsBackToStampedOrderID(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-2")

	withCashfreeGateway(t, cfGateway("cf-order-2", "77", 500, "PAID"))

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
}

// A client-supplied order id that does not match the stamped one is rejected.
// Without this a customer could point their verify at somebody else's paid order.
func TestVerifyCashfreePayment_RejectsOrderIDMismatch(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-3")

	withCashfreeGateway(t, cfGateway("cf-order-3", "1", 500, "PAID"))

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-SOMEONE-ELSE"})

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "mismatch")
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

// THE REPLAY GATE. The gateway's own payment says it belongs to a DIFFERENT order,
// so it must not settle this one — even though the client asked for the right
// order id and the amount is sufficient. payment.order_id comes from Cashfree, so
// this is the check that makes a stolen-payment replay impossible.
func TestVerifyCashfreePayment_RejectsPaymentBelongingToAnotherOrder(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-4")

	// Same request path, but the gateway reports the payment against another order.
	withCashfreeGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/payments") {
			_, _ = w.Write([]byte(cfPaymentsJSON("cf-order-ELSEWHERE", "5", 500)))
			return
		}
		_, _ = w.Write([]byte(`{"order_id":"cf-order-4","order_status":"PAID","order_amount":500.00}`))
	})

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-4"})

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "does not belong to this order")
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

// A short payment must not settle a full order — the ₹1-charge-for-a-₹500-order attack.
func TestVerifyCashfreePayment_RejectsUnderpayment(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-5")

	withCashfreeGateway(t, cfGateway("cf-order-5", "6", 1.00, "PAID")) // ₹1 against a ₹500 order

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-5"})

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

// An order the customer part-funded with credit only CAPTURES the remainder, so
// the expected amount must subtract both credit rails. Omitting the loyalty term
// is what rejected every points-funded order with a false "amount does not match".
func TestVerifyCashfreePayment_AcceptsCreditReducedCapture(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-6")
	require.NoError(t, db.Exec(
		`UPDATE orders SET wallet_applied = 100, loyalty_applied = 50 WHERE id = ?`, orderID.String()).Error)

	// Gateway captured only 500 − 100 − 50 = ₹350.
	withCashfreeGateway(t, cfGateway("cf-order-6", "8", 350, "PAID"))

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-6"})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
}

// No captured payment ⇒ an honest "not completed", not a 500 and not a completion.
func TestVerifyCashfreePayment_RejectsWhenNothingCaptured(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-7")

	withCashfreeGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/payments") {
			_, _ = w.Write([]byte(`[{"cf_payment_id":1,"order_id":"cf-order-7","payment_status":"USER_DROPPED","payment_amount":500.00}]`))
			return
		}
		_, _ = w.Write([]byte(`{"order_id":"cf-order-7","order_status":"ACTIVE","order_amount":500.00}`))
	})

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-7"})

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "not completed")
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

// A re-verify must not re-fire the side effects. completeOrderPaymentTx's guarded
// UPDATE is what makes exactly one caller perform the transition.
func TestVerifyCashfreePayment_ReVerifyDoesNotDoubleEmit(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-8")

	withCashfreeGateway(t, cfGateway("cf-order-8", "42", 500, "PAID"))

	verify := func() *httptest.ResponseRecorder {
		return callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
			func(r *gin.Engine, h *PaymentHandler) {
				r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
			}, map[string]string{"cashfreeOrderId": "cf-order-8"})
	}
	require.Equal(t, http.StatusOK, verify().Code)
	require.Equal(t, http.StatusOK, verify().Code)

	require.Equal(t, int64(1), countOutbox(t, db, "orders.paid"), "still exactly one order.paid")
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder), "still exactly one chef push")
}

// A genuine Cashfree upstream failure (transport error, timeout, 5xx,
// unparseable response) must answer 502, not 400 — the fetch itself failed,
// so this is retryable, not a false "not paid" (#872 final item). The order
// must be left untouched so a retry finds it exactly as it was.
func TestVerifyCashfreePayment_UpstreamFetchFailure502(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-10")

	// The /payments fetch 500s — a genuine gateway error, not a 404-means-
	// nothing-yet response.
	withCashfreeGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/payments") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"internal error"}`))
			return
		}
		_, _ = w.Write([]byte(`{"order_id":"cf-order-10","order_status":"ACTIVE","order_amount":500.00}`))
	})

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-10"})

	require.Equal(t, http.StatusBadGateway, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "try again")
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

// A Cashfree gateway that isn't configured answers 503, not 400 — a
// server-side misconfiguration, not a payment outcome (#872 final item),
// matching the sibling checks in createCashfreePayment / verifyRazorpayPayment.
func TestVerifyCashfreePayment_GatewayNotConfigured503(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-order-11")

	t.Cleanup(func() { services.SetCashfreeClient(nil) })
	services.SetCashfreeClient(nil) // live slot, matching cfPayOrder's default mode='live'

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-11"})

	require.Equal(t, http.StatusServiceUnavailable, w.Code, w.Body.String())
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

// finishCashfreeFromGateway (the create-leg already-PAID recovery) shares the
// SAME upstream-failure treatment as verifyCashfreePayment — not just the
// verify leg (#872 final item). Called directly (same package) with a
// minimal in-memory order so the function is proven to return BEFORE it ever
// reaches database.DB.Transaction, which this test does not set up — a bug
// that reached it would panic rather than silently pass.
func TestFinishCashfreeFromGateway_UpstreamFetchFailure502(t *testing.T) {
	setupPayDB(t)

	withCashfreeGateway(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/payments") {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"internal error"}`))
			return
		}
		_, _ = w.Write([]byte(`{"order_id":"cf-order-recover-1","order_status":"PAID","order_amount":5.00}`))
	})

	order := &models.Order{
		ModePartition:   models.ModePartition{Mode: models.ChefModeLive},
		ID:              uuid.New(),
		OrderNumber:     "HC-recover-1",
		Total:           500,
		GatewayOrderID: "cf-order-recover-1",
	}
	cfOrder := &services.CashfreeOrderResponse{
		OrderID:     "cf-order-recover-1",
		OrderStatus: services.CashfreeOrderPaid,
	}

	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/payments/order/"+order.ID.String()+"/create", nil)

	NewPaymentHandler().finishCashfreeFromGateway(c, order, cfOrder)

	require.Equal(t, http.StatusBadGateway, rec.Code, rec.Body.String())
	require.NotEqual(t, models.PaymentCompleted, order.PaymentStatus,
		"the in-memory order must not have been mutated toward completed")
}

// A verify on somebody else's order must 404 (not 403) — the IDOR scope check,
// which returns 404 so the existence of other orders isn't leaked.
func TestVerifyCashfreePayment_ScopedToTheCustomer(t *testing.T) {
	db := setupPayDB(t)
	owner := payUser(t, db, "customer")
	attacker := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, owner, chef, "pending", 500, "cf-order-9")

	withCashfreeGateway(t, cfGateway("cf-order-9", "1", 500, "PAID"))

	w := callPay(attacker, http.MethodPost, "/payments/order/"+orderID.String()+"/verify",
		func(r *gin.Engine, h *PaymentHandler) {
			r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
		}, map[string]string{"cashfreeOrderId": "cf-order-9"})

	require.Equal(t, http.StatusNotFound, w.Code)
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

// --- Webhook ---

func cfWebhookRequest(t *testing.T, h *PaymentHandler, body []byte, sign bool, timestamp string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/webhooks/cashfree", h.CashfreeWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/cashfree", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(services.CashfreeWebhookTimestampHeader, timestamp)
	if sign {
		mac := hmac.New(sha256.New, []byte(cfTestWebhookSecret))
		mac.Write([]byte(timestamp))
		mac.Write(body)
		req.Header.Set(services.CashfreeWebhookSignatureHeader, base64.StdEncoding.EncodeToString(mac.Sum(nil)))
	} else {
		req.Header.Set(services.CashfreeWebhookSignatureHeader, "forged")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func cfSuccessWebhookBody(cfOrderID, cfPaymentID string, rupees float64) []byte {
	b, _ := json.Marshal(map[string]any{
		"type":       "PAYMENT_SUCCESS_WEBHOOK",
		"event_time": time.Now().Format(time.RFC3339),
		"data": map[string]any{
			"order":   map[string]any{"order_id": cfOrderID, "order_amount": rupees},
			"payment": map[string]any{"cf_payment_id": cfPaymentID, "payment_status": "SUCCESS", "payment_amount": rupees, "payment_group": "upi"},
		},
	})
	return b
}

// A genuine webhook completes the order — the durable path when the client's
// verify call is lost.
func TestCashfreeWebhook_PaymentSuccessCompletesOrder(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-wh-1")

	withCashfreeGateway(t, cfGateway("cf-wh-1", "555", 500, "PAID"))

	w := cfWebhookRequest(t, NewPaymentHandler(), cfSuccessWebhookBody("cf-wh-1", "555", 500), true, nowUnix())
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
	require.Equal(t, "555", orderColumn(t, db, orderID, models.GatewayPaymentIDColumn))
}

// A forged signature must be rejected outright and change nothing.
func TestCashfreeWebhook_RejectsForgedSignature(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-wh-2")

	withCashfreeGateway(t, cfGateway("cf-wh-2", "1", 500, "PAID"))

	w := cfWebhookRequest(t, NewPaymentHandler(), cfSuccessWebhookBody("cf-wh-2", "1", 500), false, nowUnix())
	require.Equal(t, http.StatusUnauthorized, w.Code)
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID), "a forged webhook must not touch the order")
}

// A replayed delivery is deduped by the processed_events claim, so the chef push
// and order.paid fire once. The signature proves authenticity, not freshness.
func TestCashfreeWebhook_ReplayIsDeduped(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-wh-3")

	withCashfreeGateway(t, cfGateway("cf-wh-3", "3", 500, "PAID"))

	body := cfSuccessWebhookBody("cf-wh-3", "3", 500)
	ts := nowUnix()
	first := cfWebhookRequest(t, NewPaymentHandler(), body, true, ts)
	second := cfWebhookRequest(t, NewPaymentHandler(), body, true, ts)

	require.Equal(t, http.StatusOK, first.Code)
	require.Equal(t, http.StatusOK, second.Code)
	require.Contains(t, second.Body.String(), "duplicate")
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
	require.Equal(t, int64(1), countOutbox(t, db, services.SubjectChefNewOrder),
		"the replay must not re-push the kitchen")

	// NOTE: the webhook path emits NO order.paid — it completes the order with a
	// bare guarded UPDATE rather than completeOrderPaymentTx. That is exact parity
	// with handlePaymentCaptured (the Razorpay webhook), which behaves the same way,
	// and it is asserted here so the parity is deliberate rather than accidental.
	//
	// It is also a pre-existing gap shared by BOTH gateways: an order completed only
	// by a webhook (client dropped before verify) never emits order.paid, so any
	// consumer of that event misses it. Fixing it means routing both webhooks through
	// completeOrderPaymentTx — a change to the Razorpay path too, deliberately not
	// bundled with adding a gateway.
	require.Equal(t, int64(0), countOutbox(t, db, "orders.paid"),
		"webhook completion emits no order.paid — parity with the Razorpay webhook")
}

// A failed/user-dropped webhook marks the order failed, leaving it retryable
// (failed is deliberately NOT in completionBlockedStatuses).
func TestCashfreeWebhook_UserDroppedMarksFailed(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "pending", 500, "cf-wh-4")

	withCashfreeGateway(t, cfGateway("cf-wh-4", "4", 500, "ACTIVE"))

	body, _ := json.Marshal(map[string]any{
		"type": "PAYMENT_USER_DROPPED_WEBHOOK",
		"data": map[string]any{
			"order":   map[string]any{"order_id": "cf-wh-4"},
			"payment": map[string]any{"cf_payment_id": "4", "payment_status": "USER_DROPPED", "payment_amount": 500.00},
		},
	})
	w := cfWebhookRequest(t, NewPaymentHandler(), body, true, nowUnix())
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "failed", paymentStatusOf(t, db, orderID))
}

// A late failure webhook must NOT undo a completed payment — events can arrive
// out of order, and downgrading a paid order would un-notify the kitchen.
func TestCashfreeWebhook_FailureDoesNotDowngradeCompleted(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-wh-5")

	withCashfreeGateway(t, cfGateway("cf-wh-5", "5", 500, "PAID"))

	body, _ := json.Marshal(map[string]any{
		"type": "PAYMENT_FAILED_WEBHOOK",
		"data": map[string]any{
			"order":   map[string]any{"order_id": "cf-wh-5"},
			"payment": map[string]any{"cf_payment_id": "5", "payment_status": "FAILED", "payment_amount": 500.00},
		},
	})
	require.Equal(t, http.StatusOK, cfWebhookRequest(t, NewPaymentHandler(), body, true, nowUnix()).Code)
	require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
}

// A late duplicate SUCCESS must not re-stamp a REFUNDED order back to completed —
// that would silently re-enable the chef payout on money already returned (#563).
func TestCashfreeWebhook_SuccessDoesNotResurrectRefundedOrder(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "refunded", 500, "cf-wh-6")

	withCashfreeGateway(t, cfGateway("cf-wh-6", "6", 500, "PAID"))

	w := cfWebhookRequest(t, NewPaymentHandler(), cfSuccessWebhookBody("cf-wh-6", "6", 500), true, nowUnix())
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "refunded", paymentStatusOf(t, db, orderID))
}

// A refund webhook records the refund id. A FULL refund (>= the captured amount)
// stamps refunded_at; a PARTIAL must leave it NULL, because refunded_at is the
// whole-order marker the payout guards block the entire chef payout on (#635).
func TestCashfreeWebhook_RefundStampsFullButNotPartial(t *testing.T) {
	for _, tc := range []struct {
		name           string
		refundRupees   float64
		wantRefundedAt bool
	}{
		{"partial leaves refunded_at NULL", 100, false},
		{"full stamps refunded_at", 500, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupPayDB(t)
			cust := payUser(t, db, "customer")
			chef := payChef(t, db, payUser(t, db, "chef"))
			orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-ref-1")

			withCashfreeGateway(t, cfGateway("cf-ref-1", "1", 500, "PAID"))

			body, _ := json.Marshal(map[string]any{
				"type": "REFUND_STATUS_WEBHOOK",
				"data": map[string]any{
					"refund": map[string]any{
						"cf_refund_id": 1, "refund_id": "rfnd_abc", "order_id": "cf-ref-1",
						"refund_status": "SUCCESS", "refund_amount": tc.refundRupees,
					},
				},
			})
			require.Equal(t, http.StatusOK, cfWebhookRequest(t, NewPaymentHandler(), body, true, nowUnix()).Code)

			require.Equal(t, "rfnd_abc", orderColumn(t, db, orderID, "refund_id"))
			refundedAt := orderColumn(t, db, orderID, "refunded_at")
			if tc.wantRefundedAt {
				require.NotEmpty(t, refundedAt, "a full refund stamps the whole-order marker")
			} else {
				require.Empty(t, refundedAt, "a partial refund must NOT stamp the whole-order marker")
			}
		})
	}
}

// A non-SUCCESS refund must not be stamped: PENDING/ONHOLD have not landed, and
// FAILED/CANCELLED never moved money.
func TestCashfreeWebhook_UnsettledRefundIsNotStamped(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := cfPayOrder(t, db, cust, chef, "completed", 500, "cf-ref-2")

	withCashfreeGateway(t, cfGateway("cf-ref-2", "1", 500, "PAID"))

	body, _ := json.Marshal(map[string]any{
		"type": "REFUND_STATUS_WEBHOOK",
		"data": map[string]any{
			"refund": map[string]any{
				"refund_id": "rfnd_pending", "order_id": "cf-ref-2",
				"refund_status": "PENDING", "refund_amount": 500.00,
			},
		},
	})
	require.Equal(t, http.StatusOK, cfWebhookRequest(t, NewPaymentHandler(), body, true, nowUnix()).Code)
	require.Empty(t, orderColumn(t, db, orderID, "refund_id"))
	require.Empty(t, orderColumn(t, db, orderID, "refunded_at"))
}

// --- Order id derivation ---

// The Cashfree order_id is derived from our own order UUID: deterministic (so a
// retry reuses the same gateway order and its live session) and within Cashfree's
// 3–45 alphanumeric/-/_ window.
func TestNextCashfreeOrderID_IsDeterministicAndReusesRetrySuffix(t *testing.T) {
	id := uuid.New()
	base := id.String()

	require.Equal(t, base, nextCashfreeOrderID(id, ""), "first attempt is the bare UUID")
	require.Equal(t, base, nextCashfreeOrderID(id, base), "an unchanged stamp is reused")
	require.Equal(t, base+"-r2", nextCashfreeOrderID(id, base+"-r2"), "an existing retry suffix is kept")
	require.Equal(t, base, nextCashfreeOrderID(id, "order_ABC123razorpay"),
		"another gateway's id is ignored, not extended")
	require.LessOrEqual(t, len(nextCashfreeOrderID(id, "")), 45)
}

// Bumping must advance the suffix monotonically and stay inside the length limit.
func TestBumpCashfreeOrderID_AdvancesMonotonically(t *testing.T) {
	base := uuid.New().String()
	first := bumpCashfreeOrderID(base)
	require.Equal(t, base+"-r2", first)
	require.Equal(t, base+"-r3", bumpCashfreeOrderID(first))
	require.Equal(t, base+"-r11", bumpCashfreeOrderID(base+"-r10"))
	require.LessOrEqual(t, len(bumpCashfreeOrderID(base+"-r10")), 45)
}

func nowUnix() string { return strconv.FormatInt(time.Now().Unix(), 10) }
