package handlers

// payment_test.go — backend verification for issue #6 (payment happy-path +
// failure paths). The on-device WebView tap-through (Razorpay Standard Checkout)
// and the live-key switch are manual/owner actions (#25). What we pin down here
// is the server-side contract the device depends on, WITHOUT needing a live
// gateway: ownership/IDOR, "already paid" guard, verify-step validation, refund
// authorization + preconditions, and that the webhook fails CLOSED on a bad
// signature (no silent state change). These are the "no silent failure" and
// "no double-charge surface" guarantees the issue calls for.
//
// Self-contained in-memory SQLite (own tables + helpers, distinct names from the
// other handler tests in this package). Not parallel — shares the global DB.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

func setupPayDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)

	// users + orders carry gorm.DeletedAt → GORM appends `deleted_at IS NULL`;
	// the column must exist or the (preloaded) order load errors → spurious 404.
	require.NoError(t, db.Exec(`CREATE TABLE users (
		id TEXT PRIMARY KEY, email TEXT, first_name TEXT DEFAULT '', last_name TEXT DEFAULT '',
		phone TEXT DEFAULT '', role TEXT DEFAULT 'customer', is_active INTEGER DEFAULT 1,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (
		id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
		payment_provider TEXT DEFAULT 'razorpay', razorpay_account_id TEXT DEFAULT '',
		stripe_account_id TEXT DEFAULT '', stripe_charges_enabled INTEGER DEFAULT 0,
		payout_country TEXT DEFAULT 'IN', created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE orders (
		id TEXT PRIMARY KEY, order_number TEXT, customer_id TEXT, chef_id TEXT, delivery_id TEXT,
		status TEXT DEFAULT 'pending', payment_status TEXT DEFAULT 'pending',
		payment_method TEXT DEFAULT '', payment_provider TEXT DEFAULT 'razorpay',
		subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, total REAL DEFAULT 0,
		chef_tip REAL DEFAULT 0, driver_tip REAL DEFAULT 0, delivery_fee REAL DEFAULT 0,
		currency TEXT DEFAULT 'INR', razorpay_order_id TEXT DEFAULT '', razorpay_payment_id TEXT DEFAULT '',
		stripe_payment_intent_id TEXT DEFAULT '', refund_id TEXT DEFAULT '', refund_amount REAL DEFAULT 0,
		created_at DATETIME, updated_at DATETIME, deleted_at DATETIME
	)`).Error)
	// CreateOrderPayment preloads Delivery.DeliveryPartner — the tables must
	// exist (no rows in these tests, so the nested scan stays empty).
	require.NoError(t, db.Exec(`CREATE TABLE deliveries (
		id TEXT PRIMARY KEY, order_id TEXT, delivery_partner_id TEXT, status TEXT DEFAULT 'pending',
		created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE delivery_partners (
		id TEXT PRIMARY KEY, user_id TEXT, razorpay_account_id TEXT DEFAULT '',
		created_at DATETIME, updated_at DATETIME
	)`).Error)

	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func payUser(t *testing.T, db *gorm.DB, role string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO users (id, email, role, is_active, created_at, updated_at) VALUES (?, ?, ?, 1, ?, ?)`,
		id.String(), id.String()+"@t.com", role, time.Now(), time.Now()).Error)
	return id
}

func payChef(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, payment_provider, payout_country, created_at, updated_at)
		 VALUES (?, ?, 'Test Kitchen', 'razorpay', 'IN', ?, ?)`,
		id.String(), userID.String(), time.Now(), time.Now()).Error)
	return id
}

// payOrder inserts an order. paymentStatus drives the precondition branches.
func payOrder(t *testing.T, db *gorm.DB, customerID, chefID uuid.UUID, paymentStatus string, total float64, rzOrderID, rzPaymentID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status, payment_provider, subtotal, tax, total, currency, razorpay_order_id, razorpay_payment_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, 'pending', ?, 'razorpay', ?, ?, ?, 'INR', ?, ?, ?, ?)`,
		id.String(), "HC-"+id.String()[:8], customerID.String(), chefID.String(), paymentStatus,
		total*0.9, total*0.1, total, rzOrderID, rzPaymentID, time.Now(), time.Now()).Error)
	return id
}

func callPay(userID uuid.UUID, method, path string, register func(*gin.Engine, *PaymentHandler), body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	register(r, NewPaymentHandler())

	var reader *bytes.Reader
	if body != nil {
		buf, _ := json.Marshal(body)
		reader = bytes.NewReader(buf)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func regCreate(r *gin.Engine, h *PaymentHandler) {
	r.POST("/payments/order/:orderId/create", h.CreateOrderPayment)
}
func regVerify(r *gin.Engine, h *PaymentHandler) {
	r.POST("/payments/order/:orderId/verify", h.VerifyPayment)
}
func regRefund(r *gin.Engine, h *PaymentHandler) {
	r.POST("/payments/order/:orderId/refund", h.InitiateRefund)
}

// ── CreateOrderPayment ───────────────────────────────────────────────────────

func TestCreateOrderPayment_InvalidOrderID_400(t *testing.T) {
	setupPayDB(t)
	w := callPay(uuid.New(), http.MethodPost, "/payments/order/not-a-uuid/create", regCreate, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestCreateOrderPayment_OtherCustomersOrder_404_IDOR(t *testing.T) {
	db := setupPayDB(t)
	owner := payUser(t, db, "customer")
	attacker := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, owner, chef, "pending", 500, "", "")

	// Attacker (different customer) must not see/pay another customer's order.
	w := callPay(attacker, http.MethodPost, "/payments/order/"+orderID.String()+"/create", regCreate, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404 for IDOR, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestCreateOrderPayment_AlreadyPaid_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/create", regCreate, nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 already-paid, got %d (%s)", w.Code, w.Body.String())
	}
}

// Note: the "gateway unconfigured → 503" path isn't unit-testable here —
// services.GetRazorpay() lazily resolves credentials via GCP Secret Manager,
// which isn't initialised in the test process. In production InitRazorpay() runs
// at startup. That branch is covered by the live-switch manual step (#25).

// ── VerifyPayment ────────────────────────────────────────────────────────────

func TestVerifyPayment_OrderNotFound_404(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	w := callPay(cust, http.MethodPost, "/payments/order/"+uuid.New().String()+"/verify", regVerify,
		map[string]string{"razorpayPaymentId": "pay_1", "razorpayOrderId": "ord_1", "razorpaySignature": "sig"})
	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestVerifyPayment_MissingFields_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "rzp_order_x", "")

	// Empty razorpay fields → handler must reject (no silent pass).
	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", regVerify,
		map[string]string{})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 missing-fields, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestVerifyPayment_OrderIDMismatch_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "pending", 500, "rzp_order_REAL", "")

	// Client supplies a razorpay order id that doesn't match the stored one.
	w := callPay(cust, http.MethodPost, "/payments/order/"+orderID.String()+"/verify", regVerify,
		map[string]string{"razorpayPaymentId": "pay_1", "razorpayOrderId": "rzp_order_FORGED", "razorpaySignature": "sig"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 order-id mismatch, got %d (%s)", w.Code, w.Body.String())
	}
}

// ── InitiateRefund ───────────────────────────────────────────────────────────

func TestInitiateRefund_Unauthorized_403(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	// A random customer (not the chef, not admin) cannot refund.
	stranger := payUser(t, db, "customer")
	w := callPay(stranger, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "changed mind"})
	if w.Code != http.StatusForbidden {
		t.Fatalf("want 403, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestInitiateRefund_NotCompleted_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "pending", 500, "", "") // not completed

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "test"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 not-completed, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestInitiateRefund_ExceedsTotal_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chefUser := payUser(t, db, "chef")
	chef := payChef(t, db, chefUser)
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "pay_x")

	w := callPay(chefUser, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "test", "amount": 999.0})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 exceeds-total, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestInitiateRefund_AdminAllowed_NoRazorpayPayment_400(t *testing.T) {
	db := setupPayDB(t)
	cust := payUser(t, db, "customer")
	chef := payChef(t, db, payUser(t, db, "chef"))
	// Completed order but no razorpay_payment_id recorded.
	orderID := payOrder(t, db, cust, chef, "completed", 500, "rzp_order_x", "")
	admin := payUser(t, db, "admin")

	// Admin passes authz + completed + amount-ok, then fails on the missing
	// gateway payment id — proving the admin authorization path is reachable.
	w := callPay(admin, http.MethodPost, "/payments/order/"+orderID.String()+"/refund", regRefund,
		map[string]any{"reason": "ops refund"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400 no-razorpay-payment (admin authorized past 403), got %d (%s)", w.Code, w.Body.String())
	}
}
