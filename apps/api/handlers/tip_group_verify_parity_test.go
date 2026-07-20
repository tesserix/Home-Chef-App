package handlers

// tip_group_verify_parity_test.go — #395·4. VerifyTip and VerifyGroupShare must
// bind the captured payment's AMOUNT and Checkout SIGNATURE to the tip/share,
// exactly like the main-order VerifyPayment (payment.go). Without these, an
// under-amount captured payment on the tip/share razorpay order would settle it in
// full, and a captured payment from another order could be reused. Driven end-to-end
// against an httptest gateway via services.SetRazorpayClient + NewRazorpayTestClient.

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/homechef/api/services"
)

const parityKeySecret = "rzp_test_secret_parity"

// checkoutSignature computes the Razorpay Checkout signature the client returns:
// HMAC-SHA256(order_id|payment_id, keySecret), matching VerifyPaymentSignature.
func checkoutSignature(orderID, paymentID string) string {
	mac := hmac.New(sha256.New, []byte(parityKeySecret))
	mac.Write([]byte(orderID + "|" + paymentID))
	return hex.EncodeToString(mac.Sum(nil))
}

// fetchPaymentServer stands in for Razorpay's GET /payments/:id, returning a canned
// captured payment with the given order id + amount (paise).
func fetchPaymentServer(t *testing.T, rzOrderID string, amountPaise int) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "pay_x", "status": "captured", "order_id": rzOrderID, "amount": amountPaise, "captured": true,
		})
	}))
	t.Cleanup(srv.Close)
	services.SetRazorpayClient(services.NewRazorpayTestClient(srv.URL, "rzp_test_key", parityKeySecret, ""))
	t.Cleanup(func() { services.SetRazorpayClient(nil) })
}

func setupTipDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE tips (id TEXT PRIMARY KEY, order_id TEXT, customer_id TEXT,
		chef_amount REAL DEFAULT 0, rider_amount REAL DEFAULT 0, amount REAL DEFAULT 0, currency TEXT DEFAULT 'INR',
		chef_user_id TEXT, rider_user_id TEXT, status TEXT DEFAULT 'pending',
		razorpay_order_id TEXT DEFAULT '', razorpay_payment_id TEXT DEFAULT '', created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT,
		aggregate_type TEXT, aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
		next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedTip(t *testing.T, db *gorm.DB, customerID uuid.UUID, amount float64, rzOrderID string) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO tips (id, order_id, customer_id, amount, status, razorpay_order_id, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		id.String(), uuid.NewString(), customerID.String(), amount, "pending", rzOrderID, time.Now(), time.Now()).Error)
	return id
}

func callVerifyTip(userID, tipID uuid.UUID, body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/payments/tip/:tipId/verify", NewTipHandler().VerifyTip)
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/payments/tip/"+tipID.String()+"/verify", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// An under-amount captured payment must NOT settle the tip in full.
func TestVerifyTip_UnderAmount_400(t *testing.T) {
	db := setupTipDB(t)
	cust := uuid.New()
	tipID := seedTip(t, db, cust, 100.0, "order_tip_1") // expects 10000 paise
	fetchPaymentServer(t, "order_tip_1", 5000)          // only ₹50 captured

	w := callVerifyTip(cust, tipID, map[string]any{"razorpayPaymentId": "pay_x", "razorpayOrderId": "order_tip_1"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM tips WHERE id = ?`, tipID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status, "under-amount payment must not mark the tip paid")
}

// A bad Checkout signature (when supplied) is rejected.
func TestVerifyTip_BadSignature_400(t *testing.T) {
	db := setupTipDB(t)
	cust := uuid.New()
	tipID := seedTip(t, db, cust, 100.0, "order_tip_2")
	fetchPaymentServer(t, "order_tip_2", 10000) // amount OK

	w := callVerifyTip(cust, tipID, map[string]any{
		"razorpayPaymentId": "pay_x", "razorpayOrderId": "order_tip_2", "razorpaySignature": "deadbeef",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var status string
	require.NoError(t, db.Raw(`SELECT status FROM tips WHERE id = ?`, tipID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status)
}

// Correct amount + valid signature settles the tip.
func TestVerifyTip_ValidAmountAndSignature_200(t *testing.T) {
	db := setupTipDB(t)
	cust := uuid.New()
	tipID := seedTip(t, db, cust, 100.0, "order_tip_3")
	fetchPaymentServer(t, "order_tip_3", 10000)

	w := callVerifyTip(cust, tipID, map[string]any{
		"razorpayPaymentId": "pay_x", "razorpayOrderId": "order_tip_3",
		"razorpaySignature": checkoutSignature("order_tip_3", "pay_x"),
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var status string
	require.NoError(t, db.Raw(`SELECT status FROM tips WHERE id = ?`, tipID.String()).Scan(&status).Error)
	require.Equal(t, "paid", status, "valid amount+signature marks the tip paid")
}

// The amount gate holds even with NO signature supplied (backward-compatible clients):
// the amount check is the hard, client-independent gate.
func TestVerifyTip_NoSignatureButAmountOK_200(t *testing.T) {
	db := setupTipDB(t)
	cust := uuid.New()
	tipID := seedTip(t, db, cust, 100.0, "order_tip_4")
	fetchPaymentServer(t, "order_tip_4", 10000)

	w := callVerifyTip(cust, tipID, map[string]any{"razorpayPaymentId": "pay_x", "razorpayOrderId": "order_tip_4"})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

// ── Group share ────────────────────────────────────────────────────────────

func setupGroupDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE group_orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, chef_id TEXT, order_id TEXT,
		status TEXT DEFAULT 'collecting', created_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_order_participants (display_name_enc text DEFAULT '', id TEXT PRIMARY KEY, group_order_id TEXT,
		user_id TEXT, role TEXT DEFAULT 'guest', display_name TEXT DEFAULT '', share_amount REAL DEFAULT 0,
		payment_status TEXT DEFAULT 'pending', razorpay_order_id TEXT DEFAULT '', razorpay_payment_id TEXT DEFAULT '',
		refund_txn_id TEXT, joined_at DATETIME, updated_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE group_order_items (id TEXT PRIMARY KEY, group_order_id TEXT)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, razorpay_account_id TEXT DEFAULT '')`).Error)
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

func seedGroupParticipant(t *testing.T, db *gorm.DB, userID uuid.UUID, share float64, rzOrderID string) uuid.UUID {
	t.Helper()
	groupID, chefID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, chef_id, status) VALUES (?,?,?)`,
		groupID.String(), chefID.String(), "collecting").Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id) VALUES (?)`, chefID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO group_order_participants
		(id, group_order_id, user_id, role, share_amount, payment_status, razorpay_order_id, joined_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?)`,
		uuid.NewString(), groupID.String(), userID.String(), "guest", share, "pending", rzOrderID, time.Now(), time.Now()).Error)
	return groupID
}

func callVerifyGroupShare(userID, groupID uuid.UUID, body any) *httptest.ResponseRecorder {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	r.POST("/group-orders/:id/pay/verify", NewGroupOrderHandler().VerifyGroupShare)
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/group-orders/"+groupID.String()+"/pay/verify", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// An under-amount captured payment must NOT mark a group share paid in full.
func TestVerifyGroupShare_UnderAmount_400(t *testing.T) {
	db := setupGroupDB(t)
	user := uuid.New()
	groupID := seedGroupParticipant(t, db, user, 200.0, "order_grp_1") // expects 20000 paise
	fetchPaymentServer(t, "order_grp_1", 5000)                         // only ₹50

	w := callVerifyGroupShare(user, groupID, map[string]any{"razorpayPaymentId": "pay_x", "razorpayOrderId": "order_grp_1"})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM group_order_participants WHERE group_order_id = ?`, groupID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status, "under-amount payment must not settle the share")
}

// A bad signature (when supplied) is rejected for the group share too.
func TestVerifyGroupShare_BadSignature_400(t *testing.T) {
	db := setupGroupDB(t)
	user := uuid.New()
	groupID := seedGroupParticipant(t, db, user, 200.0, "order_grp_2")
	fetchPaymentServer(t, "order_grp_2", 20000) // amount OK

	w := callVerifyGroupShare(user, groupID, map[string]any{
		"razorpayPaymentId": "pay_x", "razorpayOrderId": "order_grp_2", "razorpaySignature": "deadbeef",
	})
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
	var status string
	require.NoError(t, db.Raw(`SELECT payment_status FROM group_order_participants WHERE group_order_id = ?`, groupID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status)
}
