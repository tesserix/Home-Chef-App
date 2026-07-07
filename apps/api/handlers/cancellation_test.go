package handlers

// cancellation_test.go — the cancellation-with-arbitration money flow (#475/#477).
// Exercises the real refund path (wallet credit via services.CreditWallet) end to
// end: fast-path auto-refund, vendor-confirm tiered refund, and not-allowed. The
// gateway ("original") path needs Razorpay (nil in tests), so tests use the wallet
// destination; the refund MATH is separately proven in cancellation_refund_test.go.
// In-memory SQLite; hand-DDL (gen_random_uuid() can't run on SQLite).

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
	"github.com/homechef/api/models"
)

func setupCancelDB(t *testing.T) (*gorm.DB, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE chef_profiles (id text PRIMARY KEY, user_id text, business_name text, deleted_at datetime)`,
		`CREATE TABLE orders (id text PRIMARY KEY, order_number text, customer_id text, chef_id text,
			status text, payment_status text, payment_provider text DEFAULT '', razorpay_payment_id text DEFAULT '',
			subtotal real DEFAULT 0, delivery_fee real DEFAULT 0, service_fee real DEFAULT 0, tax real DEFAULT 0,
			total real DEFAULT 0, refund_amount real DEFAULT 0, refund_reason text, refunded_at datetime,
			currency text DEFAULT 'INR', created_at datetime, updated_at datetime, deleted_at datetime)`,
		`CREATE TABLE cancellation_requests (id text PRIMARY KEY, order_id text, customer_id text, chef_id text,
			status text, customer_reason text, vendor_reason text, refund_destination text,
			food_refund_paise integer DEFAULT 0, delivery_refund_paise integer DEFAULT 0, tax_refund_paise integer DEFAULT 0,
			refund_total_paise integer DEFAULT 0, vendor_kept_paise integer DEFAULT 0, platform_kept_paise integer DEFAULT 0,
			refund_executed integer DEFAULT 0, refund_ref text, disputed integer DEFAULT 0, dispute_reason text,
			admin_resolved_by text, admin_note text, vendor_respond_by datetime, resolved_at datetime,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE wallets (id text PRIMARY KEY, user_id text UNIQUE, balance real DEFAULT 0, currency text DEFAULT 'INR', created_at datetime, updated_at datetime)`,
		`CREATE TABLE wallet_txns (id text PRIMARY KEY, wallet_id text, user_id text, type text, source text,
			amount real, balance_after real, currency text, order_id text, reason text, created_by text,
			idempotency_key text UNIQUE, created_at datetime)`,
		`CREATE TABLE platform_settings (id text PRIMARY KEY, key text, value text, type text, updated_by text, updated_at datetime)`,
		`CREATE TABLE outbox_events (id text PRIMARY KEY, subject text, msg_id text, aggregate_type text, aggregate_id text,
			payload text, status text, attempts int, last_error text, next_retry_at datetime, created_at datetime, updated_at datetime, published_at datetime)`,
		// #544: RequestCancellation now checks TypedRefundOrderKind, which Counts these by order_id.
		`CREATE TABLE meal_plan_days (id text PRIMARY KEY, order_id text, status text, deleted_at datetime)`,
		`CREATE TABLE group_orders (id text PRIMARY KEY, order_id text, status text, deleted_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	custID, chefUserID, chefID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name) VALUES (?,?,?)`,
		chefID.String(), chefUserID.String(), "Dum Alooo").Error)
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db, custID, chefUserID, chefID
}

// ₹500 food, ₹40 delivery, ₹60 platform fee, ₹30 tax = ₹630 total.
func seedPaidOrder(t *testing.T, db *gorm.DB, custID, chefID uuid.UUID, status string) uuid.UUID {
	t.Helper()
	oid := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, customer_id, chef_id, status, payment_status,
		subtotal, delivery_fee, service_fee, tax, total) VALUES (?,?,?,?,?,?,500,40,60,30,630)`,
		oid.String(), "ORD-"+oid.String()[:6], custID.String(), chefID.String(), status, "completed").Error)
	return oid
}

func custPost(t *testing.T, userID uuid.UUID, path string, body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID) })
	h := NewCancellationHandler()
	r.POST("/orders/:id/cancel-request", h.RequestCancellation)
	r.POST("/chef/cancel-requests/:id/confirm", h.ConfirmCancellation)
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func walletBalance(t *testing.T, db *gorm.DB, userID uuid.UUID) float64 {
	var bal float64
	db.Raw(`SELECT COALESCE(balance,0) FROM wallets WHERE user_id = ?`, userID.String()).Scan(&bal)
	return bal
}

// Chef not accepted → full food refund, no vendor. ₹500 food@100% + ₹40 delivery +
// proportional tax (₹27) = ₹567 to the wallet; platform keeps the ₹60 fee (+ its ₹3 tax).
func TestRequestCancellation_FastPathFullRefund(t *testing.T) {
	db, custID, _, chefID := setupCancelDB(t)
	oid := seedPaidOrder(t, db, custID, chefID, "pending")

	w := custPost(t, custID, "/orders/"+oid.String()+"/cancel-request", map[string]any{"refundDestination": "wallet"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var cr models.CancellationRequest
	require.NoError(t, db.Where("order_id = ?", oid.String()).First(&cr).Error)
	require.Equal(t, models.CancelReqAutoRefunded, cr.Status)
	require.True(t, cr.RefundExecuted)
	require.Equal(t, 56700, cr.RefundTotalPaise) // ₹567
	require.Equal(t, 0, cr.VendorKeptPaise)      // 100% food refunded
	require.Equal(t, 63000, cr.RefundTotalPaise+cr.VendorKeptPaise+cr.PlatformKeptPaise, "money conserved")
	require.GreaterOrEqual(t, cr.PlatformKeptPaise, 6000, "platform fee always kept")
	require.InDelta(t, 567.0, walletBalance(t, db, custID), 0.001, "customer wallet credited the refund")

	var status string
	db.Raw(`SELECT status FROM orders WHERE id = ?`, oid.String()).Scan(&status)
	require.Equal(t, "cancelled", status)
}

// Order preparing → vendor confirms materials_purchased (40%). ₹200 food + ₹40
// delivery + ₹12 tax = ₹252; vendor keeps ₹300 of food; platform keeps the fee.
func TestConfirmCancellation_VendorTierRefund(t *testing.T) {
	db, custID, chefUserID, chefID := setupCancelDB(t)
	oid := seedPaidOrder(t, db, custID, chefID, "preparing")

	// Customer requests → pending_vendor.
	w := custPost(t, custID, "/orders/"+oid.String()+"/cancel-request", map[string]any{"refundDestination": "wallet"})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	// Grab the created request's id.
	var cr models.CancellationRequest
	require.NoError(t, db.Where("order_id = ?", oid.String()).First(&cr).Error)
	require.Equal(t, models.CancelReqPendingVendor, cr.Status)
	// SQLite has no gen_random_uuid → give the row a real id so confirm can target it.
	reqID := uuid.New()
	require.NoError(t, db.Exec(`UPDATE cancellation_requests SET id = ? WHERE order_id = ?`, reqID.String(), oid.String()).Error)

	// Vendor confirms with the materials tier.
	cw := custPost(t, chefUserID, "/chef/cancel-requests/"+reqID.String()+"/confirm", map[string]any{"reason": "materials_purchased"})
	require.Equal(t, http.StatusOK, cw.Code, cw.Body.String())

	require.NoError(t, db.First(&cr, "id = ?", reqID.String()).Error)
	require.Equal(t, models.CancelReqApproved, cr.Status)
	require.True(t, cr.RefundExecuted)
	require.Equal(t, 25200, cr.RefundTotalPaise) // ₹252
	require.Equal(t, 30000, cr.VendorKeptPaise)  // vendor keeps ₹300 (60% of food)
	require.Equal(t, 63000, cr.RefundTotalPaise+cr.VendorKeptPaise+cr.PlatformKeptPaise)
	require.InDelta(t, 252.0, walletBalance(t, db, custID), 0.001)
}

// #544: a meal-plan-day / group-order shell order must NOT open a generic cancellation request —
// it is refund-managed by its typed escrow flow on a disjoint keyspace. Refuse with 422 and open
// no request (so ExecuteCancellationRefund / the retry sweep never touch it).
func TestRequestCancellation_SkipsTypedOrder(t *testing.T) {
	for _, tc := range []struct {
		name, table string
	}{
		{"meal-plan-day", "meal_plan_days"},
		{"group-order", "group_orders"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, custID, _, chefID := setupCancelDB(t)
			oid := seedPaidOrder(t, db, custID, chefID, "pending")
			require.NoError(t, db.Exec(`INSERT INTO `+tc.table+` (id, order_id, status) VALUES (?,?,?)`,
				uuid.NewString(), oid.String(), "active").Error)

			w := custPost(t, custID, "/orders/"+oid.String()+"/cancel-request", map[string]any{"refundDestination": "wallet"})
			require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())

			var n int64
			db.Model(&models.CancellationRequest{}).Where("order_id = ?", oid.String()).Count(&n)
			require.Equal(t, int64(0), n, "no cancellation request opened on a typed escrow order")
			require.InDelta(t, 0.0, walletBalance(t, db, custID), 0.001, "no generic refund credited")
		})
	}
}

// A ready/dispatched order can't be cancelled here.
func TestRequestCancellation_NotAllowedWhenReady(t *testing.T) {
	db, custID, _, chefID := setupCancelDB(t)
	oid := seedPaidOrder(t, db, custID, chefID, "delivering")
	w := custPost(t, custID, "/orders/"+oid.String()+"/cancel-request", map[string]any{"refundDestination": "wallet"})
	require.Equal(t, http.StatusConflict, w.Code)
	var n int64
	db.Model(&models.CancellationRequest{}).Count(&n)
	require.Equal(t, int64(0), n, "no request created for a dispatched order")
}

func TestConfirmCancellation_RejectsBadReason(t *testing.T) {
	db, custID, chefUserID, chefID := setupCancelDB(t)
	oid := seedPaidOrder(t, db, custID, chefID, "preparing")
	reqID := uuid.New()
	respondBy := time.Now().Add(15 * time.Minute)
	require.NoError(t, db.Exec(`INSERT INTO cancellation_requests (id, order_id, customer_id, chef_id, status, refund_destination, vendor_respond_by)
		VALUES (?,?,?,?,?,?,?)`, reqID.String(), oid.String(), custID.String(), chefID.String(), "pending_vendor", "wallet", respondBy).Error)
	w := custPost(t, chefUserID, "/chef/cancel-requests/"+reqID.String()+"/confirm", map[string]any{"reason": "bogus"})
	require.Equal(t, http.StatusBadRequest, w.Code)
}
