package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// order_issue_test.go — #37. Guardrails on the report endpoint: one open report
// per order (no duplicate refunds / no issue-rate inflation from replayed
// requests), enforced server-side regardless of the UI's disabled button.

func setupOrderIssueHandlerDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, meal_plan_day_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, photo_urls TEXT, affected_item_ids TEXT,
			requested_amount REAL DEFAULT 0, refund_amount REAL DEFAULT 0, status TEXT DEFAULT 'pending',
			resolved_by TEXT, resolved_at DATETIME, refund_txn_id TEXT, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, customer_id TEXT, chef_id TEXT, payment_status TEXT,
			status TEXT, subtotal REAL, tax REAL, total REAL, refund_amount REAL DEFAULT 0,
			refund_reason TEXT, refund_initiated_by TEXT, refunded_at DATETIME, delivered_at DATETIME,
			payout_hold_status TEXT DEFAULT '', customer_confirmed_at DATETIME, created_at DATETIME,
			updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE order_items (id TEXT PRIMARY KEY, order_id TEXT, name TEXT, price REAL, quantity INTEGER,
			subtotal REAL, is_cancelled BOOLEAN DEFAULT 0, refund_amount REAL DEFAULT 0,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE chef_profiles (address_line1_enc text DEFAULT '', address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, user_id TEXT, issue_count INTEGER DEFAULT 0, deleted_at DATETIME)`,
		// #498: AdminRejectIssue now fans the disputed-hold clear out to any meal-plan-day
		// / group-order linked to the order, so those tables must exist (empty here).
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, order_id TEXT, status TEXT,
			payout_hold_status TEXT DEFAULT '', payout_settled_at DATETIME, refund_txn_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE group_orders (delivery_address_line1_enc text DEFAULT '', delivery_address_line2_enc text DEFAULT '', id TEXT PRIMARY KEY, order_id TEXT, status TEXT,
			payout_hold_status TEXT DEFAULT '')`,
		`CREATE TABLE platform_settings (id TEXT PRIMARY KEY, key TEXT UNIQUE, value TEXT, type TEXT, updated_by TEXT, updated_at DATETIME)`,
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT, msg_id TEXT, aggregate_type TEXT,
			aggregate_id TEXT, payload TEXT, status TEXT, attempts INT, last_error TEXT,
			next_retry_at DATETIME, created_at DATETIME, updated_at DATETIME, published_at DATETIME)`,
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

	t.Run("report after the 48h delivery window is rejected (422)", func(t *testing.T) {
		db := setupOrderIssueHandlerDB(t)
		customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New()
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount, delivered_at)
			 VALUES (?, ?, ?, 'completed', 'delivered', 400, 0, 400, 0, ?)`,
			orderID.String(), customer.String(), chefID.String(), time.Now().Add(-49*time.Hour)).Error)
		require.NoError(t, db.Exec(
			`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
			chefID.String(), uuid.New().String()).Error)

		w := reportIssueReq(t, customer, orderID, "reason=missing_item")
		assert.Equal(t, http.StatusUnprocessableEntity, w.Code)

		var issues int64
		db.Raw(`SELECT count(*) FROM order_issues WHERE order_id = ?`, orderID.String()).Scan(&issues)
		assert.Equal(t, int64(0), issues, "no issue row created for an out-of-window report")
	})

	t.Run("report within the 48h delivery window is allowed", func(t *testing.T) {
		db := setupOrderIssueHandlerDB(t)
		customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New()
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount, delivered_at)
			 VALUES (?, ?, ?, 'completed', 'delivered', 400, 0, 400, 0, ?)`,
			orderID.String(), customer.String(), chefID.String(), time.Now().Add(-1*time.Hour)).Error)
		require.NoError(t, db.Exec(
			`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
			chefID.String(), uuid.New().String()).Error)

		w := reportIssueReq(t, customer, orderID, "reason=missing_item")
		assert.Equal(t, http.StatusCreated, w.Code)
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

	// #622 (reverse direction): an item already cancelled+refunded via CancelOrderItem must be
	// EXCLUDED from the issue's refundable computation — otherwise RefundIssueToWallet returns
	// that line's money a second time (fully automatic on ordinary multi-item orders).
	t.Run("an already-cancelled affected item is excluded from the requested refund", func(t *testing.T) {
		db := setupOrderIssueHandlerDB(t)
		customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New()
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount)
			 VALUES (?, ?, ?, 'completed', 'delivered', 1000, 0, 1000, 0)`,
			orderID.String(), customer.String(), chefID.String()).Error)
		require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
			chefID.String(), uuid.New().String()).Error)
		itemA, itemB := uuid.New(), uuid.New()
		require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, subtotal, is_cancelled) VALUES (?,?,?,1)`,
			itemA.String(), orderID.String(), 500).Error) // A already cancelled
		require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, subtotal, is_cancelled) VALUES (?,?,?,0)`,
			itemB.String(), orderID.String(), 500).Error) // B still live

		// Report naming BOTH A and B.
		w := reportIssueReq(t, customer, orderID, "reason=damaged&affectedItemIds="+itemA.String()+"&affectedItemIds="+itemB.String())
		assert.Equal(t, http.StatusCreated, w.Code, w.Body.String())

		var requested float64
		var affected string
		require.NoError(t, db.Raw(`SELECT requested_amount, affected_item_ids FROM order_issues WHERE order_id = ?`,
			orderID.String()).Row().Scan(&requested, &affected))
		assert.Equal(t, 500.0, requested, "only the LIVE item B (500) is refundable; the cancelled A is excluded (not 1000)")
		assert.NotContains(t, affected, itemA.String(), "the cancelled item is dropped from affected_item_ids")
		assert.Contains(t, affected, itemB.String())
	})
}

// #618 slice 2: reporting a quality issue on a delivered meal-plan DAY (reported against
// the day's shell order) must (a) link the issue to the day and (b) freeze the day's
// payout hold to `disputed` so a pending report can't release the day before it's resolved.
func TestReportIssue_MealPlanDay_FreezesHoldAndLinksDay(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	customer, orderID, chefID, dayID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 120, 0, 120, 0)`,
		orderID.String(), customer.String(), chefID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
		chefID.String(), uuid.New().String()).Error)
	// The delivered day parked awaiting customer confirmation, linked to the shell order.
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, payout_hold_status)
		VALUES (?, ?, ?, ?, ?)`, dayID.String(), uuid.New().String(), orderID.String(),
		string(models.MealPlanDayDelivered), string(models.PayoutHoldAwaitingConfirmation)).Error)

	// No affected items → requested 0 → stays pending (no auto-refund) so we isolate the freeze.
	w := reportIssueReq(t, customer, orderID, "reason=quality_issue")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var linkedDay string
	require.NoError(t, db.Raw(`SELECT meal_plan_day_id FROM order_issues WHERE order_id = ?`, orderID.String()).Scan(&linkedDay).Error)
	assert.Equal(t, dayID.String(), linkedDay, "issue links to the reported meal-plan day")

	var hold string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM meal_plan_days WHERE id = ?`, dayID.String()).Scan(&hold).Error)
	assert.Equal(t, string(models.PayoutHoldDisputed), hold, "reporting freezes the day's payout hold to disputed")
}

// #618 slice 2: a quality issue on a delivery-FAILED meal-plan day is rejected. That day
// is handled by the admin delivery-failure resolver (RefundDay, a disjoint wallet key), so
// allowing this path's refund too would double-refund the customer. No issue row is created.
func TestReportIssue_FailedMealPlanDay_Rejected(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	customer, orderID, chefID, dayID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 120, 0, 120, 0)`,
		orderID.String(), customer.String(), chefID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
		chefID.String(), uuid.New().String()).Error)
	// A failed day (frozen disputed by the delivery-failure path) linked to the shell.
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, order_id, status, payout_hold_status)
		VALUES (?, ?, ?, ?, ?)`, dayID.String(), uuid.New().String(), orderID.String(),
		string(models.MealPlanDayFailed), string(models.PayoutHoldDisputed)).Error)

	w := reportIssueReq(t, customer, orderID, "reason=quality_issue")
	assert.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	var issues int64
	db.Raw(`SELECT count(*) FROM order_issues WHERE order_id = ?`, orderID.String()).Scan(&issues)
	assert.Equal(t, int64(0), issues, "no issue row is created for a failed day")
}

// A normal food order (no linked meal-plan day) reports fine with a nil day link and no
// day-hold side effects — the day lookup is a benign record-not-found.
func TestReportIssue_NonMealPlanOrder_NoDayLink(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 400, 0, 400, 0)`,
		orderID.String(), customer.String(), chefID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, issue_count) VALUES (?, ?, 0)`,
		chefID.String(), uuid.New().String()).Error)

	w := reportIssueReq(t, customer, orderID, "reason=quality_issue")
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	var linkedDay *string
	require.NoError(t, db.Raw(`SELECT meal_plan_day_id FROM order_issues WHERE order_id = ?`, orderID.String()).Scan(&linkedDay).Error)
	assert.Nil(t, linkedDay, "a normal order has no linked meal-plan day")
}

// rejectIssueReq POSTs an admin reject for the given issue.
func rejectIssueReq(t *testing.T, adminID, issueID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", adminID); c.Next() })
	r.POST("/admin/issues/:issueId/reject", NewOrderIssueHandler().AdminRejectIssue)
	req := httptest.NewRequest(http.MethodPost, "/admin/issues/"+issueID.String()+"/reject", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestAdminRejectIssue_DrivesDisputedHoldToReleaseEligible — rejecting the dispute
// (#458) advances the order's disputed payout hold back to release_eligible in the
// same transaction, so the chef's legitimately-earned payout leaves the dead-end.
func TestAdminRejectIssue_DrivesDisputedHoldToReleaseEligible(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	admin, customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, payout_hold_status, customer_confirmed_at)
		 VALUES (?, ?, ?, 'completed', 'delivered', 400, ?, ?)`,
		orderID.String(), customer.String(), chefID.String(), string(models.PayoutHoldDisputed), time.Now()).Error)
	issueID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, status)
		 VALUES (?, ?, ?, ?, 'quality_issue', 'pending')`,
		issueID.String(), orderID.String(), chefID.String(), customer.String()).Error)

	w := rejectIssueReq(t, admin, issueID)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var status, hold string
	require.NoError(t, db.Raw(`SELECT status FROM order_issues WHERE id = ?`, issueID.String()).Scan(&status).Error)
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM orders WHERE id = ?`, orderID.String()).Scan(&hold).Error)
	require.Equal(t, string(models.IssueRejected), status)
	require.Equal(t, string(models.PayoutHoldReleaseEligible), hold, "rejecting the dispute releases the held payout")
}

// TestAdminRejectIssue_MultiIssue_StaysDisputedUntilLastCleared — an order can carry
// several pending issues; rejecting one must keep the hold disputed while another is
// still pending, and only release once the last one clears (#458 NOT EXISTS guard).
func TestAdminRejectIssue_MultiIssue_StaysDisputedUntilLastCleared(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	admin, customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, payout_hold_status, customer_confirmed_at)
		 VALUES (?, ?, ?, 'completed', 'delivered', 400, ?, ?)`,
		orderID.String(), customer.String(), chefID.String(), string(models.PayoutHoldDisputed), time.Now()).Error)
	issue1, issue2 := uuid.New(), uuid.New()
	for _, iid := range []uuid.UUID{issue1, issue2} {
		require.NoError(t, db.Exec(
			`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, status)
			 VALUES (?, ?, ?, ?, 'quality_issue', 'pending')`,
			iid.String(), orderID.String(), chefID.String(), customer.String()).Error)
	}

	// Reject the first — the second is still pending, so the hold must stay disputed.
	require.Equal(t, http.StatusOK, rejectIssueReq(t, admin, issue1).Code)
	require.Equal(t, string(models.PayoutHoldDisputed), orderHold(t, db, orderID), "still disputed while issue2 pending")

	// Reject the second — now nothing is pending, the hold releases.
	require.Equal(t, http.StatusOK, rejectIssueReq(t, admin, issue2).Code)
	require.Equal(t, string(models.PayoutHoldReleaseEligible), orderHold(t, db, orderID), "released once the last issue cleared")

	// Rejecting an already-handled issue is a 409 and changes nothing.
	require.Equal(t, http.StatusConflict, rejectIssueReq(t, admin, issue2).Code)
}

func orderHold(t *testing.T, db *gorm.DB, orderID uuid.UUID) string {
	t.Helper()
	var hold string
	require.NoError(t, db.Raw(`SELECT payout_hold_status FROM orders WHERE id = ?`, orderID.String()).Scan(&hold).Error)
	return hold
}

// resolveIssueReq POSTs an admin assisted-refund resolution with the given amount.
func resolveIssueReq(t *testing.T, adminID, issueID uuid.UUID, amount float64) *httptest.ResponseRecorder {
	return resolveIssueReqPolicy(t, adminID, issueID, amount, "")
}

// resolveIssueReqPolicy posts an admin resolve with an explicit fault policy (#618).
// An empty policy omits the field (falls back to the configured default).
func resolveIssueReqPolicy(t *testing.T, adminID, issueID uuid.UUID, amount float64, policy string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", adminID); c.Next() })
	r.POST("/admin/issues/:issueId/resolve", NewOrderIssueHandler().AdminResolveIssue)
	body := fmt.Sprintf(`{"amount": %g}`, amount)
	if policy != "" {
		body = fmt.Sprintf(`{"amount": %g, "faultPolicy": %q}`, amount, policy)
	}
	req := httptest.NewRequest(http.MethodPost, "/admin/issues/"+issueID.String()+"/resolve", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// #618: a platform-goodwill resolution that would FULLY refund the order is rejected
// (422) before any money moves — goodwill can't preserve the chef payout on a full
// refund. The admin lowers the amount or resolves with clawback.
func TestAdminResolveIssue_GoodwillFullRefund_Returns422(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	admin, customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 300, 0)`,
		orderID.String(), customer.String(), chefID.String()).Error)
	issueID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, status)
		 VALUES (?, ?, ?, ?, 'quality_issue', 'pending')`,
		issueID.String(), orderID.String(), chefID.String(), customer.String()).Error)

	// Full amount (== remaining) + goodwill → rejected before any credit.
	w := resolveIssueReqPolicy(t, admin, issueID, 300, "platform_goodwill")
	require.Equal(t, http.StatusUnprocessableEntity, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM order_issues WHERE id = ?`, issueID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status, "issue untouched — no money moved on a rejected goodwill-full")
	var refund float64
	require.NoError(t, db.Raw(`SELECT refund_amount FROM orders WHERE id = ?`, orderID.String()).Scan(&refund).Error)
	require.Equal(t, 0.0, refund)
}

// #618: an invalid fault policy is a 400 (before any lookup / money).
func TestAdminResolveIssue_InvalidFaultPolicy_Returns400(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	admin, customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 300, 0)`,
		orderID.String(), customer.String(), chefID.String()).Error)
	issueID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, status)
		 VALUES (?, ?, ?, ?, 'quality_issue', 'pending')`,
		issueID.String(), orderID.String(), chefID.String(), customer.String()).Error)

	w := resolveIssueReqPolicy(t, admin, issueID, 100, "chef_pays_double")
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())
}

// TestAdminResolveIssue_FullyRefunded_PreCapRejects — the pre-cap (now RemainingRefundable, #624)
// rejects an assisted refund on an order with nothing left, before any money moves.
func TestAdminResolveIssue_FullyRefunded_PreCapRejects(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	admin, customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 300, 300)`,
		orderID.String(), customer.String(), chefID.String()).Error)
	issueID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, status)
		 VALUES (?, ?, ?, ?, 'quality_issue', 'pending')`,
		issueID.String(), orderID.String(), chefID.String(), customer.String()).Error)

	w := resolveIssueReq(t, admin, issueID, 100)
	require.Equal(t, http.StatusBadRequest, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM order_issues WHERE id = ?`, issueID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status, "issue untouched when there's nothing to refund")
}

// TestAdminResolveIssue_AllAffectedCancelled_Returns409 — #624 end-to-end through the handler: an
// affected line was cancelled after the report (its money already returned), another line keeps
// RemainingRefundable positive so the pre-cap passes, but RefundIssueToWallet's lock-time
// exclusion zeroes the credit → the handler maps ErrNothingToRefund to 409, no double refund.
func TestAdminResolveIssue_AllAffectedCancelled_Returns409(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	admin, customer, orderID, chefID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	itemA, itemC := uuid.New(), uuid.New()
	// A(200)+C(200)=400; chef cancelled A → total=200, refund_amount=200, A per-line refund 200. C live.
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 200, 200)`,
		orderID.String(), customer.String(), chefID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		itemA.String(), orderID.String(), true, 200).Error)
	require.NoError(t, db.Exec(`INSERT INTO order_items (id, order_id, is_cancelled, refund_amount) VALUES (?,?,?,?)`,
		itemC.String(), orderID.String(), false, 0).Error)
	issueID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, status, affected_item_ids, requested_amount)
		 VALUES (?, ?, ?, ?, 'missing_item', 'pending', ?, 200)`,
		issueID.String(), orderID.String(), chefID.String(), customer.String(), "{"+itemA.String()+"}").Error)

	w := resolveIssueReq(t, admin, issueID, 200)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM order_issues WHERE id = ?`, issueID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status, "nothing resolved — the affected line was already refunded via the cancel")
}

// TestDeliveryFailureIssue_NotResolvableViaGenericPath — a delivery_failed issue
// carries the fault-based RTO money policy (#393), so it must be EXCLUDED from the
// generic issues list and REJECTED by the generic resolve/reject paths (a flat
// refund there would mis-handle a customer-fault RTO, which pays the chef with no
// refund). It's resolvable only via the delivery-failure fault resolver.
func TestDeliveryFailureIssue_NotResolvableViaGenericPath(t *testing.T) {
	db := setupOrderIssueHandlerDB(t)
	admin, orderID, chefID, issueID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, customer_id, chef_id, payment_status, status, subtotal, tax, total, refund_amount)
		 VALUES (?, ?, ?, 'completed', 'delivered', 400, 0, 400, 0)`,
		orderID.String(), uuid.New().String(), chefID.String()).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO order_issues (id, order_id, chef_id, reason, status, requested_amount)
		 VALUES (?, ?, ?, 'delivery_failed', 'pending', 0)`,
		issueID.String(), orderID.String(), chefID.String()).Error)

	// Excluded from the generic issues list.
	gin.SetMode(gin.TestMode)
	rList := gin.New()
	rList.GET("/admin/issues", NewOrderIssueHandler().AdminListIssues)
	reqList := httptest.NewRequest(http.MethodGet, "/admin/issues?status=pending", nil)
	recList := httptest.NewRecorder()
	rList.ServeHTTP(recList, reqList)
	require.Equal(t, http.StatusOK, recList.Code)
	assert.NotContains(t, recList.Body.String(), issueID.String(), "delivery_failed excluded from generic list")

	// Generic resolve + reject both refuse it (409); the issue stays pending.
	require.Equal(t, http.StatusConflict, resolveIssueReq(t, admin, issueID, 100).Code)
	require.Equal(t, http.StatusConflict, rejectIssueReq(t, admin, issueID).Code)

	var status string
	require.NoError(t, db.Raw(`SELECT status FROM order_issues WHERE id = ?`, issueID.String()).Scan(&status).Error)
	require.Equal(t, "pending", status)
}
