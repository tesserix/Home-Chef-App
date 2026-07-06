package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupDeliveryFailureQueueDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE order_issues (id TEXT PRIMARY KEY, order_id TEXT, chef_id TEXT, customer_id TEXT,
			reason TEXT, description TEXT, status TEXT DEFAULT 'pending', created_at DATETIME)`,
		`CREATE TABLE orders (id TEXT PRIMARY KEY, order_number TEXT, total REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', deleted_at DATETIME)`,
		`CREATE TABLE meal_plan_days (id TEXT PRIMARY KEY, meal_plan_id TEXT, status TEXT, price REAL DEFAULT 0,
			payout_hold_status TEXT DEFAULT '', date DATETIME, updated_at DATETIME)`,
		`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, meal_plan_number TEXT, customer_id TEXT, chef_id TEXT)`,
		`CREATE TABLE group_orders (id TEXT PRIMARY KEY, host_id TEXT, chef_id TEXT, status TEXT,
			subtotal REAL DEFAULT 0, tax REAL DEFAULT 0, payout_hold_status TEXT DEFAULT '', updated_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	prev := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = prev })
	return db
}

type deliveryFailureResp struct {
	OrderIssues  []orderDeliveryFailureRow `json:"orderIssues"`
	MealPlanDays []dayDeliveryFailureRow   `json:"mealPlanDays"`
	GroupOrders  []groupDeliveryFailureRow `json:"groupOrders"`
	Count        int                       `json:"count"`
}

func TestAdminListDeliveryFailures(t *testing.T) {
	db := setupDeliveryFailureQueueDB(t)

	// --- Order issues: one pending delivery_failed (SHOWS), plus two that must NOT show.
	order := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO orders (id, order_number, total, payout_hold_status)
		VALUES (?,?,?,?)`, order.String(), "ORD-77", 250.0, string(models.PayoutHoldDisputed)).Error)
	issue, cust, chef := uuid.New(), uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, chef_id, customer_id, reason, description, status)
		VALUES (?,?,?,?,?,?,?)`, issue.String(), order.String(), chef.String(), cust.String(),
		string(models.IssueDeliveryFailed),
		"delivery failed: reason=customer_unavailable suggested_fault=customer reported_by=courier",
		string(models.IssuePending)).Error)
	// wrong reason → excluded
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, reason, status) VALUES (?,?,?,?)`,
		uuid.NewString(), order.String(), string(models.IssueMissingItem), string(models.IssuePending)).Error)
	// already resolved → excluded
	require.NoError(t, db.Exec(`INSERT INTO order_issues (id, order_id, reason, status) VALUES (?,?,?,?)`,
		uuid.NewString(), order.String(), string(models.IssueDeliveryFailed), string(models.IssueResolved)).Error)

	// --- Meal-plan days: one failed (SHOWS), one delivered (excluded).
	plan := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, meal_plan_number, customer_id, chef_id) VALUES (?,?,?,?)`,
		plan.String(), "MP-9", cust.String(), chef.String()).Error)
	failedDay := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, payout_hold_status)
		VALUES (?,?,?,?,?)`, failedDay.String(), plan.String(), string(models.MealPlanDayFailed), 150.0,
		string(models.PayoutHoldDisputed)).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price) VALUES (?,?,?,?)`,
		uuid.NewString(), plan.String(), string(models.MealPlanDayDelivered), 150.0).Error)

	// --- Group orders: one failed (SHOWS), one delivered (excluded).
	failedGroup := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, host_id, chef_id, status, subtotal, tax, payout_hold_status)
		VALUES (?,?,?,?,?,?,?)`, failedGroup.String(), uuid.NewString(), chef.String(),
		string(models.GroupOrderFailed), 400.0, 20.0, string(models.PayoutHoldDisputed)).Error)
	require.NoError(t, db.Exec(`INSERT INTO group_orders (id, status) VALUES (?,?)`,
		uuid.NewString(), string(models.GroupOrderDelivered)).Error)

	// --- Call the handler.
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/admin/delivery-failures", NewOrderIssueHandler().AdminListDeliveryFailures)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/delivery-failures", nil))
	require.Equal(t, http.StatusOK, w.Code)

	var resp deliveryFailureResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))

	require.Len(t, resp.OrderIssues, 1, "only the pending delivery_failed issue shows")
	require.Equal(t, issue.String(), resp.OrderIssues[0].IssueID)
	require.Equal(t, "ORD-77", resp.OrderIssues[0].OrderNumber)
	require.Equal(t, 250.0, resp.OrderIssues[0].Total)
	require.Equal(t, string(models.PayoutHoldDisputed), resp.OrderIssues[0].HoldStatus)
	require.Equal(t, "customer", resp.OrderIssues[0].SuggestedFault, "suggested fault parsed from the description")
	require.Equal(t, "customer_unavailable", resp.OrderIssues[0].Reason)
	require.Equal(t, "courier", resp.OrderIssues[0].ReportedBy)

	require.Len(t, resp.MealPlanDays, 1, "only the failed day shows")
	require.Equal(t, failedDay.String(), resp.MealPlanDays[0].DayID)
	require.Equal(t, "MP-9", resp.MealPlanDays[0].MealPlanNumber)
	require.Equal(t, 150.0, resp.MealPlanDays[0].Price)

	require.Len(t, resp.GroupOrders, 1, "only the failed group shows")
	require.Equal(t, failedGroup.String(), resp.GroupOrders[0].GroupID)
	require.Equal(t, 400.0, resp.GroupOrders[0].Subtotal)

	require.Equal(t, 3, resp.Count)
}
