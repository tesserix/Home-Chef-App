package handlers

// tips_cashfree_only_test.go — #1086 Phase 5. A tip is a NEW charge, so it is
// minted on Cashfree whatever gateway the order it thanks was stamped with.
//
// The old leg dispatched on the ORDER's payment_provider and sent everything
// else to Razorpay Route, which demanded a chef.razorpay_account_id that no chef
// on the platform has — the whole tip surface answered 409. A historical
// razorpay-stamped order can still be tipped; the tip just rides the gateway the
// platform actually charges on.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

// The retired INR gateway is no longer vocabulary in apps/api (#1132); a stored
// row still carries the string, which is what these tests exercise.
const retiredGatewayStamp = "razorpay"

// tipGateway stands in for Cashfree: it answers order-create and the verify
// path's GET /orders/:id/payments, and records every create body it receives.
func tipGateway(t *testing.T, payments string) *[]map[string]any {
	t.Helper()
	bodies := &[]map[string]any{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/payments") {
			_, _ = w.Write([]byte(payments))
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		*bodies = append(*bodies, body)
		orderID, _ := body["order_id"].(string)
		amount, _ := body["order_amount"].(float64)
		_, _ = w.Write([]byte(fmt.Sprintf(
			`{"cf_order_id":1,"order_id":%q,"payment_session_id":"sess_tip","order_status":"ACTIVE","order_amount":%.2f}`,
			orderID, amount)))
	}))
	t.Cleanup(srv.Close)
	services.SetCashfreeClient(
		services.NewCashfreeTestClient(srv.URL, "app_test", "secret_test", cfTestWebhookSecret, models.ChefModeLive))
	t.Cleanup(func() { services.SetCashfreeClient(nil) })
	return bodies
}

func setupTipChargeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupPayDB(t)
	require.NoError(t, db.Exec(`ALTER TABLE orders ADD COLUMN fulfillment_type TEXT DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE tips (mode text DEFAULT 'live', test_session_id text, cloned_from_id text,
		id TEXT PRIMARY KEY, order_id TEXT, customer_id TEXT, chef_user_id TEXT, rider_user_id TEXT,
		amount REAL DEFAULT 0, chef_amount REAL DEFAULT 0, rider_amount REAL DEFAULT 0,
		currency TEXT DEFAULT 'INR', status TEXT, gateway_order_id TEXT DEFAULT '',
		gateway_payment_id TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME)`).Error)
	return db
}

// tippableOrder seeds a delivered order stamped with orderProvider, whose chef
// has the given Cashfree vendor registration.
func tippableOrder(t *testing.T, db *gorm.DB, orderProvider, vendorID, vendorStatus string) (uuid.UUID, uuid.UUID) {
	t.Helper()
	chefUser := payUser(t, db, "chef")
	chefID := payChef(t, db, chefUser)
	require.NoError(t, db.Exec(
		`UPDATE chef_profiles SET cashfree_vendor_id = ?, cashfree_vendor_status = ? WHERE id = ?`,
		vendorID, vendorStatus, chefID.String()).Error)

	customerID := payUser(t, db, "customer")
	orderID := payOrder(t, db, customerID, chefID, "paid", 500, "order_legacy", "pay_legacy")
	require.NoError(t, db.Exec(
		`UPDATE orders SET status = ?, payment_provider = ?, fulfillment_type = ? WHERE id = ?`,
		models.OrderStatusDelivered, orderProvider, models.FulfillmentChefDelivery, orderID.String()).Error)
	return customerID, orderID
}

func createTip(customerID, orderID uuid.UUID, body any) *httptest.ResponseRecorder {
	return callAs(customerID, http.MethodPost, "/payments/order/"+orderID.String()+"/tip", func(r *gin.Engine) {
		r.POST("/payments/order/:orderId/tip", NewTipHandler().CreateOrderTip)
	}, body)
}

func seedPendingTip(t *testing.T, db *gorm.DB, customerID uuid.UUID, amount float64, gatewayOrderID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO tips (id, order_id, customer_id, amount, status, gateway_order_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), customerID.String(), amount, models.TipPending, gatewayOrderID,
		time.Now(), time.Now()).Error)
	return id
}

func verifyTip(customerID, tipID uuid.UUID) *httptest.ResponseRecorder {
	return callAs(customerID, http.MethodPost, "/payments/tip/"+tipID.String()+"/verify", func(r *gin.Engine) {
		r.POST("/payments/tip/:tipId/verify", NewTipHandler().VerifyTip)
	}, map[string]any{})
}

// The order is stamped with the retired INR gateway — a historical row. The tip
// is still a new charge, minted on Cashfree.
func TestCreateOrderTip_LegacyStampedOrderIsTippedThroughCashfree(t *testing.T) {
	db := setupTipChargeDB(t)
	bodies := tipGateway(t, `[]`)
	customerID, orderID := tippableOrder(t, db, retiredGatewayStamp, "hc_chef1", services.CashfreeVendorActive)

	w := createTip(customerID, orderID, map[string]any{"chefAmount": 60})

	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var body map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	require.Equal(t, models.PaymentProviderCashfree, body["provider"])
	require.NotEmpty(t, body["cashfreePaymentSessionId"])
	// cashfreeEnv is the name every other charge payload uses and the one the web
	// checkout reads; without it the sheet defaults to production against a
	// sandbox session id.
	require.Equal(t, "PRODUCTION", body["cashfreeEnv"])

	require.Len(t, *bodies, 1)
	created := (*bodies)[0]
	require.True(t, strings.HasPrefix(created["order_id"].(string), "tip-"))
	require.Equal(t, 60.0, created["order_amount"])
}

// A chef whose payout registration is not live gets the reason, not the blanket
// "This chef can't receive tips right now" the old leg gave everyone.
func TestCreateOrderTip_UnregisteredChefIsRefusedForTheRealReason(t *testing.T) {
	db := setupTipChargeDB(t)
	tipGateway(t, `[]`)
	customerID, orderID := tippableOrder(t, db, retiredGatewayStamp, "", "")

	w := createTip(customerID, orderID, map[string]any{"chefAmount": 60})

	require.Equal(t, http.StatusConflict, w.Code)
	require.Contains(t, w.Body.String(), "payout account isn't active")
}

// Verification asks the gateway, never the client: no gatewayPaymentId is
// required, and a legacy gateway order id does not route anywhere else.
func TestVerifyTip_LegacyOrderIDIsVerifiedAgainstCashfree(t *testing.T) {
	db := setupTipChargeDB(t)
	tipGateway(t, `[]`)
	customerID := payUser(t, db, "customer")
	tipID := seedPendingTip(t, db, customerID, 100, "order_legacy")

	w := verifyTip(customerID, tipID)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.NotContains(t, w.Body.String(), "gatewayPaymentId")
	require.Contains(t, w.Body.String(), "Payment not completed")
}

// #395·4's amount binding survives the gateway change: an under-amount capture
// must not settle the tip in full.
func TestVerifyTip_UnderAmountCaptureDoesNotSettleTheTip(t *testing.T) {
	db := setupTipChargeDB(t)
	tipGateway(t, `[{"cf_payment_id":1,"order_id":"tip-abc","payment_status":"SUCCESS","payment_amount":50.00}]`)
	customerID := payUser(t, db, "customer")
	tipID := seedPendingTip(t, db, customerID, 100, "tip-abc")

	w := verifyTip(customerID, tipID)

	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	require.Equal(t, string(models.TipPending), tipStatusOf(t, db, tipID))
}

func TestVerifyTip_CapturedPaymentSettlesTheTip(t *testing.T) {
	db := setupTipChargeDB(t)
	tipGateway(t, `[{"cf_payment_id":1,"order_id":"tip-abc","payment_status":"SUCCESS","payment_amount":100.00}]`)
	customerID := payUser(t, db, "customer")
	tipID := seedPendingTip(t, db, customerID, 100, "tip-abc")

	w := verifyTip(customerID, tipID)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, string(models.TipPaid), tipStatusOf(t, db, tipID))
}

func tipStatusOf(t *testing.T, db *gorm.DB, tipID uuid.UUID) string {
	t.Helper()
	var s string
	require.NoError(t, db.Raw(`SELECT status FROM tips WHERE id = ?`, tipID.String()).Scan(&s).Error)
	return s
}
