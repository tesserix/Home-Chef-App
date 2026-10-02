package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/services"
	"github.com/stretchr/testify/require"
)

type stripeGatewayStub struct {
	intent      *services.StripePaymentIntent
	created     []*services.StripePaymentIntentRequest
	onFetch     func()
	settlements []string
}

func (s *stripeGatewayStub) CreatePaymentIntent(_ context.Context, req *services.StripePaymentIntentRequest) (*services.StripePaymentIntent, error) {
	s.created = append(s.created, req)
	return s.intent, nil
}
func (s *stripeGatewayStub) FetchPaymentIntent(context.Context, string) (*services.StripePaymentIntent, error) {
	if s.onFetch != nil {
		s.onFetch()
	}
	return s.intent, nil
}
func (s *stripeGatewayStub) SetPaymentIntentSettlement(_ context.Context, id, account string) (*services.StripePaymentIntent, error) {
	s.settlements = append(s.settlements, id+":"+account)
	s.intent.OnBehalfOf = account
	return s.intent, nil
}
func (s *stripeGatewayStub) GetPublishableKey() string { return "pk_test_fixture" }
func (s *stripeGatewayStub) IsTestMode() bool          { return true }

func TestStripePlannedMarketRejectsLiveCheckout(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	require.NoError(t, db.Exec("UPDATE chef_profiles SET payout_country = 'AU', stripe_account_id = 'acct_vendor', stripe_charges_enabled = 1 WHERE id = ?", chef).Error)
	require.NoError(t, db.Exec("UPDATE orders SET currency = 'AUD', mode = 'live', commission_rate = 10 WHERE id = ?", orderID).Error)
	stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_live", Amount: 5000, Currency: "aud", Status: "requires_payment_method", ClientSecret: "fixture"}}
	response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/create", func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regCreate(r, h) }, nil)
	require.Equal(t, 503, response.Code, response.Body.String())
	require.Empty(t, stub.created)
}

func TestStripeVerifyDoesNotReportSuccessWhenCommitFails(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	setStripeIntent(t, db, orderID, "pi_verify")
	require.NoError(t, db.Exec("UPDATE orders SET currency = 'NZD', mode = 'test' WHERE id = ?", orderID).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_outbox BEFORE INSERT ON outbox_events BEGIN SELECT RAISE(FAIL, 'outbox unavailable'); END`).Error)
	stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_verify", Amount: 5000, AmountReceived: 5000, Currency: "nzd", Status: "succeeded"}}
	response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", func(r *gin.Engine, h *PaymentHandler) {
		h.stripe = stub
		regVerify(r, h)
	}, map[string]string{"stripePaymentIntentId": "pi_verify"})
	require.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
}

func TestStripeVerifyAPIAuthenticatesAndCompletesExactlyOnce(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	other := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	setStripeIntent(t, db, orderID, "pi_verified")
	require.NoError(t, db.Exec("UPDATE orders SET currency = 'AUD', mode = 'test' WHERE id = ?", orderID).Error)
	stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_verified", Amount: 5000, AmountReceived: 5000, Currency: "aud", Status: "succeeded"}}
	register := func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regVerify(r, h) }
	path := "/payments/order/" + orderID.String() + "/verify"
	body := map[string]string{"stripePaymentIntentId": "pi_verified"}
	response := callPay(other, http.MethodPost, path, register, body)
	require.Equal(t, 404, response.Code)
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
	for i := 0; i < 2; i++ {
		response = callPay(customer, http.MethodPost, path, register, body)
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), `"status":"completed"`)
	}
	require.EqualValues(t, 1, countOutbox(t, db, "orders.paid"))
	require.EqualValues(t, 1, countOutbox(t, db, services.SubjectChefNewOrder))
}

func TestStripeVerifyDoesNotReportCompletedAfterConcurrentRefund(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	setStripeIntent(t, db, orderID, "pi_refund_race")
	stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_refund_race", Amount: 5000, AmountReceived: 5000, Currency: "inr", Status: "succeeded", Livemode: true}, onFetch: func() {
		require.NoError(t, db.Exec("UPDATE orders SET payment_status = 'refunded' WHERE id = ?", orderID).Error)
	}}
	response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regVerify(r, h) }, map[string]string{"stripePaymentIntentId": "pi_refund_race"})
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
	require.Equal(t, "refunded", paymentStatusOf(t, db, orderID))
	require.Zero(t, countOutbox(t, db, "orders.paid"))
}

func TestStripeCreateDoesNotExposeIntentWhenPersistenceFails(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	require.NoError(t, db.Exec("UPDATE chef_profiles SET payment_provider = 'stripe', payout_country = 'AU', stripe_account_id = 'acct_vendor', stripe_charges_enabled = 1 WHERE id = ?", chef).Error)
	require.NoError(t, db.Exec("UPDATE orders SET currency = 'AUD', mode = 'test', commission_rate = 10 WHERE id = ?", orderID).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_intent BEFORE UPDATE OF stripe_payment_intent_id ON orders BEGIN SELECT RAISE(FAIL, 'write unavailable'); END`).Error)
	stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_created", Amount: 5000, Currency: "aud", Status: "requires_payment_method", ClientSecret: "fixture_secret"}}
	response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/create", func(r *gin.Engine, h *PaymentHandler) {
		h.stripe = stub
		regCreate(r, h)
	}, nil)
	require.Equal(t, http.StatusInternalServerError, response.Code, response.Body.String())
	require.NotContains(t, response.Body.String(), "fixture_secret")
	require.Empty(t, orderColumn(t, db, orderID, "stripe_payment_intent_id"))
}

func TestStripeCreateReusesStoredIntent(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	require.NoError(t, db.Exec("UPDATE chef_profiles SET payout_country = 'AU', stripe_account_id = 'acct_vendor', stripe_charges_enabled = 1 WHERE id = ?", chef).Error)
	require.NoError(t, db.Exec("UPDATE orders SET currency = 'AUD', mode = 'test', commission_rate = 10 WHERE id = ?", orderID).Error)
	stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_created", Amount: 5000, Currency: "aud", Status: "requires_payment_method", ClientSecret: "fixture_secret"}}
	register := func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regCreate(r, h) }
	for i := 0; i < 2; i++ {
		response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/create", register, nil)
		require.Equal(t, 200, response.Code, response.Body.String())
		require.Contains(t, response.Body.String(), `"stripePaymentIntentId":"pi_created"`)
	}
	require.Len(t, stub.created, 1, "retry fetches the stored intent, never creates another")
	require.Equal(t, "fe3dr-order-"+orderID.String(), stub.created[0].IdempotencyKey)
}

func TestStripeCreateRejectsUnsupportedCreditWithoutCharging(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	require.NoError(t, db.Exec("UPDATE chef_profiles SET payout_country = 'AU', stripe_account_id = 'acct_vendor', stripe_charges_enabled = 1 WHERE id = ?", chef).Error)
	require.NoError(t, db.Exec("UPDATE orders SET currency = 'AUD', mode = 'test', commission_rate = 10 WHERE id = ?", orderID).Error)
	stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_credit", Amount: 5000, Currency: "aud", Status: "requires_payment_method", ClientSecret: "fixture_secret"}}
	response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/create", func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regCreate(r, h) }, map[string]bool{"useWallet": true})
	require.Equal(t, http.StatusUnprocessableEntity, response.Code, response.Body.String())
	require.Empty(t, stub.created)
}

func TestStripeSuccessRejectsUnderpayment(t *testing.T) {
	db := setupPayDB(t)
	customer := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
	setStripeIntent(t, db, orderID, "pi_underpaid")
	require.NoError(t, db.Exec("UPDATE orders SET currency = 'AUD' WHERE id = ?", orderID).Error)
	payload, err := json.Marshal(services.StripePaymentIntent{
		ID: "pi_underpaid", Livemode: true, Amount: 100, Currency: "aud", Status: "succeeded",
	})
	require.NoError(t, err)
	(&PaymentHandler{}).handleStripePaymentSucceeded(t.Context(), payload)
	require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
	require.Zero(t, countOutbox(t, db, "orders.paid"))
}

func TestStripeSuccessCommitsPaymentAndOutboxTogether(t *testing.T) {
	for _, failOutbox := range []bool{false, true} {
		t.Run(map[bool]string{false: "duplicate emits once", true: "outbox failure rolls back"}[failOutbox], func(t *testing.T) {
			db := setupPayDB(t)
			customer := payUser(t, db, "customer")
			chef := payChef(t, db, payUser(t, db, "chef"))
			orderID := payOrder(t, db, customer, chef, "pending", 500, "", "")
			setStripeIntent(t, db, orderID, "pi_atomic")
			if failOutbox {
				require.NoError(t, db.Exec(`CREATE TRIGGER reject_outbox BEFORE INSERT ON outbox_events BEGIN SELECT RAISE(FAIL, 'outbox unavailable'); END`).Error)
			}
			h := &PaymentHandler{}
			h.handleStripePaymentSucceeded(t.Context(), stripeIntentPayload(t, "pi_atomic"))
			h.handleStripePaymentSucceeded(t.Context(), stripeIntentPayload(t, "pi_atomic"))
			if failOutbox {
				require.Equal(t, "pending", paymentStatusOf(t, db, orderID))
				require.Zero(t, countOutbox(t, db, services.SubjectChefNewOrder))
			} else {
				require.Equal(t, "completed", paymentStatusOf(t, db, orderID))
				require.EqualValues(t, 1, countOutbox(t, db, "orders.paid"))
				require.EqualValues(t, 1, countOutbox(t, db, services.SubjectChefNewOrder))
				require.EqualValues(t, 1, countOutbox(t, db, services.SubjectPaymentSuccess))
			}
		})
	}
}

func TestStripeApplicationFeePreservesSharedCommissionBasis(t *testing.T) {
	for _, tc := range []struct{ country, currency string }{{"AU", "AUD"}, {"NZ", "NZD"}} {
		t.Run(tc.country, func(t *testing.T) {
			db := setupPayDB(t)
			customer := payUser(t, db, "customer")
			chef := payChef(t, db, payUser(t, db, "chef"))
			orderID := payOrder(t, db, customer, chef, "pending", 120, "", "")
			require.NoError(t, db.Exec("UPDATE chef_profiles SET payment_provider='stripe', payout_country=?, stripe_account_id='acct_vendor', stripe_charges_enabled=1 WHERE id=?", tc.country, chef).Error)
			require.NoError(t, db.Exec("UPDATE orders SET currency=?, mode='test', subtotal=100, tax=10, tax_food=10, chef_tip=5, commission_rate=0.06 WHERE id=?", tc.currency, orderID).Error)
			stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_allocation", Amount: 12000, Currency: tc.currency, Status: "requires_payment_method", ClientSecret: "fixture"}}
			response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/create", func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regCreate(r, h) }, nil)
			require.Equal(t, http.StatusOK, response.Code, response.Body.String())
			require.Len(t, stub.created, 1)
			require.Equal(t, 12000, stub.created[0].Amount)
			require.Equal(t, 1100, stub.created[0].ApplicationFeeCents, "vendor gets 100 + 10 food GST + 5 tip - 6 commission = 109; platform holds remainder")
			require.Equal(t, "acct_vendor", stub.created[0].DestinationAccount)
		})
	}
}

func TestStripeRetryRepairsUnconfirmedDestinationSettlement(t *testing.T) {
	for _, tc := range []struct {
		name, status, settlement string
		wantCode, updates        int
	}{
		{"unconfirmed", "requires_payment_method", "", 200, 1},
		{"awaiting confirmation", "requires_confirmation", "", 200, 1},
		{"already correct", "requires_payment_method", "acct_nz_vendor", 200, 0},
		{"wrong merchant", "requires_payment_method", "acct_other", 502, 0},
		{"processing", "processing", "", 200, 0},
		{"requires action", "requires_action", "", 200, 0},
		{"succeeded", "succeeded", "", 200, 0},
		{"canceled", "canceled", "", 409, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupPayDB(t)
			customer := payUser(t, db, "customer")
			chef := payChef(t, db, payUser(t, db, "chef"))
			orderID := payOrder(t, db, customer, chef, "pending", 50, "", "")
			require.NoError(t, db.Exec("UPDATE chef_profiles SET payout_country='NZ', stripe_account_id='acct_nz_vendor', stripe_charges_enabled=1 WHERE id=?", chef).Error)
			require.NoError(t, db.Exec("UPDATE orders SET currency='NZD', mode='test', stripe_payment_intent_id='pi_saved', payment_provider='stripe' WHERE id=?", orderID).Error)
			stub := &stripeGatewayStub{intent: &services.StripePaymentIntent{ID: "pi_saved", Amount: 5000, Currency: "nzd", Status: tc.status, OnBehalfOf: tc.settlement, ClientSecret: "fixture"}}
			response := callPay(customer, http.MethodPost, "/payments/order/"+orderID.String()+"/create", func(r *gin.Engine, h *PaymentHandler) { h.stripe = stub; regCreate(r, h) }, nil)
			require.Equal(t, tc.wantCode, response.Code, response.Body.String())
			require.Len(t, stub.settlements, tc.updates)
			if tc.updates > 0 {
				require.Equal(t, "pi_saved:acct_nz_vendor", stub.settlements[0])
			}
			require.Empty(t, stub.created)
		})
	}
}
