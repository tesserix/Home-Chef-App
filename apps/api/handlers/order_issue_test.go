package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
)

// order_issue_test.go — #37. Guardrails on the report endpoint: one open report
// per order (no duplicate refunds / no issue-rate inflation from replayed
// requests), enforced server-side regardless of the UI's disabled button.

func setupOrderIssueHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, photo_urls TEXT, affected_item_ids TEXT,
			requested_amount REAL DEFAULT 0, refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending',
			resolved_by TEXT, resolved_at DATETIME, refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE orders (id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, payment_status TEXT,
			status TEXT, subtotal REAL, tax REAL, total REAL, refund_amount REAL DEFAULT 0,
			refund_reason TEXT, refund_initiated_by TEXT, refunded_at DATETIME, created_at DATETIME,
			updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE order_items (id TEXT PRIMARY KEY, order_id TEXT, name TEXT, price REAL, quantity INTEGER,
			subtotal REAL, created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT, issue_count INTEGER DEFAULT 0, deleted_at DATETIME)`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT, type TEXT, updated_by TEXT, updated_at DATETIME)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

// reportIssueReq posts an urlencoded report (no photo) as the given customer.
func reportIssueReq(t *testing.T, customerID, orderID uuid.UUID, form string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", customerID); c.Next() })
	r.POST("/orders/:id/report-issue", NewOrderIssueHandler().ReportIssue)
	req := httptest.NewRequest(http.MethodPost, "/orders/"+orderID.String()+"/report-issue", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestReportIssueGuards(t *testing.T) {
	t.Run("second report on the same order is blocked (409) and doesn't re-bump issue rate", func(t *testing.T) {
		db := setupOrderIssueHandlerDB(t)
		customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New()
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount)
			 VALUES (?, ?, ?, 'completed', 'delivered', 400, 0, 400, 0)`,
			orderID.String(), customer.String(), chefID.String()).Error)
		require.NoError(t, db.Exec(
			`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
			chefID.String(), uuid.New().String()).Error)

		// First report (no affected items → requested 0 → stays pending).
		w1 := reportIssueReq(t, customer, orderID, "reason=missing_item")
		assert.Equal(t, http.StatusCreated, w1.Code)

		// Second report on the same order is rejected.
		w2 := reportIssueReq(t, customer, orderID, "reason=quality_issue")
		assert.Equal(t, http.StatusConflict, w2.Code)

		// Exactly one issue row and the chef's issue_count bumped exactly once.
		var issues, issueCount int64
		db.Raw(`SELECT count(*) FROM order_issues WHERE order_id = ?`, orderID.String()).Scan(&issues)
		db.Raw(`SELECT issue_count FROM chef_profiles WHERE id = ?`, chefID.String()).Scan(&issueCount)
		assert.Equal(t, int64(1), issues)
		assert.Equal(t, int64(1), issueCount)
	})

	t.Run("report on an unpaid order is rejected", func(t *testing.T) {
		db := setupOrderIssueHandlerDB(t)
		customer, orderID := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, refund_amount)
			 VALUES (?, ?, ?, 'pending', 'pending', 400, 0)`,
			orderID.String(), customer.String(), uuid.New().String()).Error)

		w := reportIssueReq(t, customer, orderID, "reason=missing_item")
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("cannot report on another customer's order", func(t *testing.T) {
		db := setupOrderIssueHandlerDB(t)
		owner, attacker, orderID := uuid.New(), uuid.New(), uuid.New()
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, refund_amount)
			 VALUES (?, ?, ?, 'completed', 'delivered', 400, 0)`,
			orderID.String(), owner.String(), uuid.New().String()).Error)

		w := reportIssueReq(t, attacker, orderID, "reason=missing_item")
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}
