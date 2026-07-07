package handlers

// payout_hold_test.go — HTTP + delivery-hook checks for the payout hold state
// machine (#387). Delivering a meal-plan day parks the hold (no release); the
// customer confirm endpoint advances awaiting -> release_eligible, is owner-scoped,
// idempotent, and forced to disputed by an open OrderIssue. In-memory sqlite with
// hand-DDL'd tables (the models' gen_random_uuid() default can't run on sqlite).

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

func setupHoldHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE orders (id TEXT PRIMARY KEY, customer_id TEXT, status TEXT,
			razorpay_order_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			customer_confirmed_at DATETIME, refunded_at DATETIME,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT,
			status TEXT, payout_transfer_id TEXT DEFAULT '', payout_hold_status TEXT DEFAULT '',
			customer_confirmed_at DATETIME, delivered_at DATETIME, date DATETIME, refund_txn_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, status TEXT)`,
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, status TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT, value TEXT, type TEXT, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			subject TEXT, msg_id TEXT, aggregate_type TEXT, aggregate_id TEXT, payload TEXT,
			status TEXT, attempts INTEGER DEFAULT 0, last_error TEXT, next_retry_at DATETIME,
			created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func confirmRouter(userID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := NewPayoutHoldHandler()
	r.POST("/orders/:id/confirm-received", h.ConfirmOrderReceived)
	return r
}

// Delivering a meal-plan day that HAS a held transfer parks the hold and leaves
// the transfer UNCHANGED (release deferred) — with escrow ON and OFF.
func TestMarkMealPlanDayDelivered_ParksHold_NoRelease(t *testing.T) {
	for _, escrow := range []bool{true, false} {
		saved := config.AppConfig
		config.AppConfig = &config.Config{MealPlanEscrowEnabled: escrow}

		db := setupHoldHandlerDB(t)
		orderID := uuid.New()
		dayID := uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, payout_transfer_id, payout_hold_status) VALUES (?,?,?,?,?,?)`,
			dayID.String(), uuid.NewString(), orderID.String(), "confirmed", "tr_held_123", "").Error)

		services.MarkMealPlanDayDelivered(orderID)

		var day models.MealPlanDay
		require.NoError(t, db.First(&day, "id = ?", dayID).Error)
		require.Equal(t, models.MealPlanDayDelivered, day.Status, "escrow=%v", escrow)
		require.Equal(t, models.PayoutHoldAwaitingConfirmation, day.PayoutHoldStatus, "escrow=%v", escrow)
		require.Equal(t, "tr_held_123", day.PayoutTransferID, "release deferred — transfer untouched (escrow=%v)", escrow)

		config.AppConfig = saved
	}
}

// Owner confirm advances awaiting -> release_eligible and stamps the confirm time.
func TestConfirmOrderReceived_OwnerReleaseEligible(t *testing.T) {
	db := setupHoldHandlerDB(t)
	customerID := uuid.New()
	orderID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status) VALUES (?,?,?,?,?)`,
		orderID.String(), customerID.String(), "delivered", "order_rzp_1", string(models.PayoutHoldAwaitingConfirmation)).Error)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/confirm-received", nil)
	confirmRouter(customerID).ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var order models.Order
	require.NoError(t, db.First(&order, "id = ?", orderID).Error)
	require.Equal(t, models.PayoutHoldReleaseEligible, order.PayoutHoldStatus)
	require.NotNil(t, order.CustomerConfirmedAt)
}

// An open OrderIssue forces disputed (200 but not release_eligible).
func TestConfirmOrderReceived_OpenIssueDisputes(t *testing.T) {
	db := setupHoldHandlerDB(t)
	customerID := uuid.New()
	orderID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status) VALUES (?,?,?,?,?)`,
		orderID.String(), customerID.String(), "delivered", "order_rzp_1", string(models.PayoutHoldAwaitingConfirmation)).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, status) VALUES (?,?,?)`,
		uuid.NewString(), orderID.String(), string(models.IssuePending)).Error)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/confirm-received", nil)
	confirmRouter(customerID).ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var order models.Order
	require.NoError(t, db.First(&order, "id = ?", orderID).Error)
	require.Equal(t, models.PayoutHoldDisputed, order.PayoutHoldStatus)
}

// A different customer cannot confirm someone else's order.
func TestConfirmOrderReceived_NonOwner403(t *testing.T) {
	db := setupHoldHandlerDB(t)
	ownerID := uuid.New()
	orderID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status) VALUES (?,?,?,?,?)`,
		orderID.String(), ownerID.String(), "delivered", "order_rzp_1", string(models.PayoutHoldAwaitingConfirmation)).Error)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/confirm-received", nil)
	confirmRouter(uuid.New()).ServeHTTP(w, req) // a stranger
	require.Contains(t, []int{http.StatusForbidden, http.StatusNotFound}, w.Code)

	var order models.Order
	require.NoError(t, db.First(&order, "id = ?", orderID).Error)
	require.Equal(t, models.PayoutHoldAwaitingConfirmation, order.PayoutHoldStatus, "no state change for a non-owner")
}

// Re-confirming is an idempotent 200 with no double transition.
func TestConfirmOrderReceived_Idempotent(t *testing.T) {
	db := setupHoldHandlerDB(t)
	customerID := uuid.New()
	orderID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, customer_id, status, razorpay_order_id, payout_hold_status) VALUES (?,?,?,?,?)`,
		orderID.String(), customerID.String(), "delivered", "order_rzp_1", string(models.PayoutHoldAwaitingConfirmation)).Error)

	router := confirmRouter(customerID)
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/confirm-received", nil)
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	}
	var order models.Order
	require.NoError(t, db.First(&order, "id = ?", orderID).Error)
	require.Equal(t, models.PayoutHoldReleaseEligible, order.PayoutHoldStatus)
}
