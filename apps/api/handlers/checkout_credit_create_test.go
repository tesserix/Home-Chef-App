package handlers

// checkout_credit_create_test.go — payment creation is the ONLY authority on what
// the customer is charged.
//
// THE BUG THIS PINS: the client used to compute the payable from its own cached
// wallet balance and its own view of the total, then post a rupee amount. The
// server re-clamped that amount against live state independently. Whenever the two
// views drifted — a stale balance, a delivery fee recomputed server-side, a
// chef-adjusted fee — the screen rendered "To pay ₹0" while the server still
// minted a gateway charge. The client now sends INTENT only.

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// withGatewayOrderCapture points GetRazorpay at a stub answering the create-order
// POST, capturing the paise amount actually sent to the gateway.
func withGatewayOrderCapture(t *testing.T) *int {
	t.Helper()
	var paise int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body struct {
				Amount int `json:"amount"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			paise = body.Amount
			_ = json.NewEncoder(w).Encode(map[string]any{"id": "order_test", "status": "created"})
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "rzp_test_key", "rzp_test_secret", ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })
	return &paise
}

// seedCreditOrder builds an unpaid ₹1000 order (subtotal 900 + tax 100) for a
// customer holding the given wallet balance and loyalty points.
func seedCreditOrder(t *testing.T, db *gorm.DB, wallet, points float64) (uuid.UUID, uuid.UUID) {
	t.Helper()
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "pending", 1000, "", "")
	if wallet > 0 {
		_, err := services.CreditWallet(db, cust, wallet, models.WalletSourcePromo, nil,
			"seed", "seed-w-"+uuid.NewString(), nil)
		require.NoError(t, err)
	}
	if points > 0 {
		_, err := services.EarnLoyalty(db, cust, points, models.LoyaltySourceOrder, nil,
			"seed", "seed-p-"+uuid.NewString())
		require.NoError(t, err)
	}
	return cust, orderID
}

func createBody(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	return body
}

// The client's rupee figure must never win over the server's live computation.
func TestCreateOrderPayment_ClientAmountNeverOverridesServer(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	capturedPaise := withGatewayOrderCapture(t)
	// The wallet really holds ₹100; the client wrongly believes it holds ₹1200.
	cust, orderID := seedCreditOrder(t, db, 100, 0)

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"useWallet": true, "walletAmount": 1200.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	body := createBody(t, w)
	require.Equal(t, 100.0, body["walletApplied"], "clamped to the live balance")
	require.Equal(t, 900.0, body["payable"])
	require.Equal(t, 90000, *capturedPaise, "the gateway is charged the server's figure")
}

// Credit can no longer reach the fees, so the capture always covers fee + tax —
// which is precisely why a "₹0 outstanding" order can no longer reach the gateway
// with a mismatched amount.
func TestCreateOrderPayment_CaptureAlwaysCoversFeesAndTax(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	capturedPaise := withGatewayOrderCapture(t)
	cust, orderID := seedCreditOrder(t, db, 100000, 0)

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"useWallet": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	body := createBody(t, w)
	require.Equal(t, 900.0, body["walletApplied"], "the food subtotal, and no more")
	require.Equal(t, 100.0, body["payable"], "the tax, in cash")
	require.Equal(t, 10000, *capturedPaise)
}

// A legacy build posts a bare walletAmount with no useWallet flag.
func TestCreateOrderPayment_LegacyWalletAmountBodyStillWorks(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	withGatewayOrderCapture(t)
	cust, orderID := seedCreditOrder(t, db, 500, 0)

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"walletAmount": 500.0})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, 500.0, createBody(t, w)["walletApplied"])
}

// Points fund the order alongside the wallet and are stamped for the settle step.
func TestCreateOrderPayment_StampsLoyaltyForSettle(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	cust, orderID := seedCreditOrder(t, db, 0, 100000) // no wallet, lots of points
	withGatewayOrderCapture(t)

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"useWallet": true, "useLoyalty": true})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	body := createBody(t, w)
	require.Equal(t, 90.0, body["loyaltyApplied"], "10% of the 900.00 food subtotal")

	var o models.Order
	require.NoError(t, db.First(&o, "id = ?", orderID).Error)
	require.Equal(t, 90.0, o.LoyaltyApplied)
	require.Equal(t, 1800.0, o.LoyaltyPointsSpent, "90.00 / 0.05 per point")

	// Creating the payment must NOT yet burn the points — an abandoned checkout
	// would otherwise cost the customer their balance.
	acct, err := services.LoyaltyBalance(db, cust)
	require.NoError(t, err)
	require.Equal(t, 100000.0, acct.Balance, "points are debited on capture, not on create")
}

// Declining both rails charges the full total.
func TestCreateOrderPayment_NoCreditRequestedChargesFullTotal(t *testing.T) {
	db := setupPayDB(t)
	addWalletTables(t, db)
	quoteFlagsOn(t)
	capturedPaise := withGatewayOrderCapture(t)
	cust, orderID := seedCreditOrder(t, db, 5000, 100000)

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create",
		regCreate, map[string]any{"useWallet": false, "useLoyalty": false})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	body := createBody(t, w)
	require.Equal(t, 0.0, body["walletApplied"])
	require.Equal(t, 0.0, body["loyaltyApplied"])
	require.Equal(t, 100000, *capturedPaise)
}
