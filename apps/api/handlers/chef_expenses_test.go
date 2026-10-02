package handlers

// chef_expenses_test.go — pins the expense-book CRUD (ownership, validation,
// FY summary bucketing) and the FY statement aggregation the vendor uses for
// GST/ITR filing: earnings re-derived from delivered orders (Easy Split
// included) minus self-declared expenses.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	"github.com/homechef/api/services"
)

func setupExpenseDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_profiles (
			id            TEXT PRIMARY KEY,
			user_id       TEXT NOT NULL,
			business_name TEXT,
			state         TEXT,
			city          TEXT,
			postal_code   TEXT,
			pan_number    TEXT,
			gstin         TEXT,
			payout_country TEXT DEFAULT 'IN'
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE chef_expenses (
			id           TEXT PRIMARY KEY,
			chef_id      TEXT NOT NULL,
			user_id      TEXT NOT NULL,
			category     TEXT NOT NULL,
			amount       REAL NOT NULL,
			currency     TEXT DEFAULT 'INR',
			note         TEXT,
			order_id     TEXT,
			expense_date DATETIME NOT NULL,
			receipt_url  TEXT,
			created_at   DATETIME,
			updated_at   DATETIME
		)
	`).Error)
	require.NoError(t, db.Exec(`
		CREATE TABLE orders (
			id                     TEXT PRIMARY KEY,
			order_number           TEXT,
			chef_id                TEXT,
			status                 TEXT,
			delivered_at           DATETIME,
			subtotal               REAL DEFAULT 0,
			tax                    REAL DEFAULT 0,
			-- Per-supply split. The FY statement reads these to credit the chef the
			-- FOOD GST only; without them the query cannot run at all.
			tax_food               REAL DEFAULT 0,
			tax_service            REAL DEFAULT 0,
			chef_funded_discount   REAL DEFAULT 0,
			delivery_fee           REAL DEFAULT 0,
			delivery_fee_final     REAL,
			fulfillment_type       TEXT DEFAULT 'delivery',
			chef_tip               REAL DEFAULT 0,
			driver_tip             REAL DEFAULT 0,
			delivery_address_state TEXT,
			commission_rate        REAL DEFAULT 0,
			gateway_split_paise    INTEGER DEFAULT 0,
			deleted_at             DATETIME
		)
	`).Error)
	database.DB = db
	return db
}

func seedExpenseChef(t *testing.T, db *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, state) VALUES (?, ?, ?, ?)`,
		chefID.String(), userID.String(), "Test Kitchen", "Karnataka",
	).Error)
	return chefID
}

func expenseReq(t *testing.T, userID uuid.UUID, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", userID); c.Next() })
	h := NewChefExpensesHandler()
	tax := NewChefTaxHandler()
	r.GET("/chef/expenses", h.ListExpenses)
	r.POST("/chef/expenses", h.CreateExpense)
	r.GET("/chef/expenses/summary", h.GetExpenseSummary)
	r.PUT("/chef/expenses/:id", h.UpdateExpense)
	r.DELETE("/chef/expenses/:id", h.DeleteExpense)
	r.GET("/chef/tax/fy-statement", tax.GetFYStatement)

	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestExpenseCRUD(t *testing.T) {
	db := setupExpenseDB(t)
	userID := uuid.New()
	chefID := seedExpenseChef(t, db, userID)

	// Create.
	w := expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
		"category":    "gas",
		"amount":      850.505,
		"note":        "LPG refill",
		"expenseDate": "2025-06-10",
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	var created struct {
		Expense models.ChefExpense `json:"expense"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
	assert.Equal(t, models.ExpenseGas, created.Expense.Category)
	assert.Equal(t, 850.51, created.Expense.Amount) // rounded to paise
	assert.Equal(t, chefID, created.Expense.ChefID)

	// List returns it.
	w = expenseReq(t, userID, http.MethodGet, "/chef/expenses", nil)
	require.Equal(t, http.StatusOK, w.Code)
	var list struct {
		Expenses []models.ChefExpense `json:"expenses"`
		Total    int64                `json:"total"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list.Expenses, 1)
	assert.EqualValues(t, 1, list.Total)

	// Update.
	w = expenseReq(t, userID, http.MethodPut, "/chef/expenses/"+created.Expense.ID.String(), gin.H{
		"category":    "utensils",
		"amount":      1200.0,
		"note":        "New kadai",
		"expenseDate": "2025-06-12",
	})
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var updated struct {
		Expense models.ChefExpense `json:"expense"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &updated))
	assert.Equal(t, models.ExpenseUtensils, updated.Expense.Category)
	assert.Equal(t, 1200.0, updated.Expense.Amount)

	// Delete.
	w = expenseReq(t, userID, http.MethodDelete, "/chef/expenses/"+created.Expense.ID.String(), nil)
	require.Equal(t, http.StatusOK, w.Code)
	w = expenseReq(t, userID, http.MethodGet, "/chef/expenses", nil)
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	assert.Len(t, list.Expenses, 0)
}

func TestExpenseValidationAndOwnership(t *testing.T) {
	db := setupExpenseDB(t)
	userID := uuid.New()
	seedExpenseChef(t, db, userID)

	// Unknown category.
	w := expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
		"category": "yacht", "amount": 10.0, "expenseDate": "2025-06-10",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Future date.
	future := time.Now().AddDate(0, 0, 2).Format("2006-01-02")
	w = expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
		"category": "gas", "amount": 10.0, "expenseDate": future,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Negative amount.
	w = expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
		"category": "gas", "amount": -5.0, "expenseDate": "2025-06-10",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)

	// Another chef cannot touch my rows.
	w = expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
		"category": "gas", "amount": 100.0, "expenseDate": "2025-06-10",
	})
	require.Equal(t, http.StatusCreated, w.Code)
	var created struct {
		Expense models.ChefExpense `json:"expense"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))

	otherUser := uuid.New()
	seedExpenseChef(t, db, otherUser)
	w = expenseReq(t, otherUser, http.MethodDelete, "/chef/expenses/"+created.Expense.ID.String(), nil)
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = expenseReq(t, otherUser, http.MethodPut, "/chef/expenses/"+created.Expense.ID.String(), gin.H{
		"category": "gas", "amount": 1.0, "expenseDate": "2025-06-10",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestExpenseOrderLinking(t *testing.T) {
	db := setupExpenseDB(t)
	userID := uuid.New()
	chefID := seedExpenseChef(t, db, userID)

	orderID := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal)
		VALUES (?, 'HC042', ?, 'preparing', NULL, 500)
	`, orderID, chefID.String()).Error)

	// Linking my own order works and the list echoes the order number.
	w := expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
		"category": "ingredients", "amount": 250.0, "expenseDate": "2025-06-10",
		"orderId": orderID,
	})
	require.Equal(t, http.StatusCreated, w.Code, w.Body.String())

	w = expenseReq(t, userID, http.MethodGet, "/chef/expenses?orderId="+orderID, nil)
	require.Equal(t, http.StatusOK, w.Code)
	var list struct {
		Expenses []struct {
			OrderID     *uuid.UUID `json:"orderId"`
			OrderNumber string     `json:"orderNumber"`
		} `json:"expenses"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &list))
	require.Len(t, list.Expenses, 1)
	assert.Equal(t, "HC042", list.Expenses[0].OrderNumber)

	// Another chef's order cannot be linked.
	otherUser := uuid.New()
	otherChef := seedExpenseChef(t, db, otherUser)
	otherOrder := uuid.NewString()
	require.NoError(t, db.Exec(`
		INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal)
		VALUES (?, 'HC099', ?, 'preparing', NULL, 100)
	`, otherOrder, otherChef.String()).Error)
	w = expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
		"category": "gas", "amount": 50.0, "expenseDate": "2025-06-10",
		"orderId": otherOrder,
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestExpenseSummaryBucketsByFY(t *testing.T) {
	db := setupExpenseDB(t)
	userID := uuid.New()
	seedExpenseChef(t, db, userID)

	// Two in FY 2025-26 (Jun 2025 gas + Feb 2026 ingredients), one outside (Mar 2025).
	for _, e := range []struct {
		cat  string
		amt  float64
		date string
	}{
		{"gas", 800, "2025-06-10"},
		{"ingredients", 1500, "2026-02-05"},
		{"gas", 999, "2025-03-20"}, // FY 2024-25 — must be excluded
	} {
		w := expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
			"category": e.cat, "amount": e.amt, "expenseDate": e.date,
		})
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
	}

	w := expenseReq(t, userID, http.MethodGet, "/chef/expenses/summary?year=2025", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var summary services.ExpenseSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))

	assert.Equal(t, 2300.0, summary.Total)
	assert.Equal(t, 2, summary.Count)
	assert.Equal(t, "FY 2025-26", summary.FYLabel)
	require.Len(t, summary.ByCategory, 2)
	// Declared order: ingredients before gas.
	assert.Equal(t, models.ExpenseIngredients, summary.ByCategory[0].Category)
	assert.Equal(t, 1500.0, summary.ByCategory[0].Amount)
	assert.Equal(t, models.ExpenseGas, summary.ByCategory[1].Category)
	assert.Equal(t, 800.0, summary.ByCategory[1].Amount)
	require.Len(t, summary.ByMonth, 2)
	assert.Equal(t, "2025-06", summary.ByMonth[0].Month)
	assert.Equal(t, "2026-02", summary.ByMonth[1].Month)
}

func TestFYStatementMath(t *testing.T) {
	db := setupExpenseDB(t)
	userID := uuid.New()
	chefID := seedExpenseChef(t, db, userID)

	// One intra-state delivered order in Q1 FY2025 (Jun 2025):
	// subtotal 1000, food GST 50, tip 20 → gross 1070, commission 6% = 60,
	// TDS 1% = 10.70, net = 999.30 (mirrors chef_earnings_test.go).
	insertOrder := func(id, num string, deliveredAt time.Time, subtotal, tax, tip float64, state string, splitPaise int) {
		require.NoError(t, db.Exec(`
			INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax,
				chef_funded_discount, delivery_fee, chef_tip, delivery_address_state, commission_rate, gateway_split_paise)
			VALUES (?, ?, ?, 'delivered', ?, ?, ?, 0, 30, ?, ?, 0, ?)
		`, id, num, chefID.String(), deliveredAt, subtotal, tax, tip, state, splitPaise).Error)
	}
	insertOrder(uuid.NewString(), "HC001", time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC), 1000, 50, 20, "Karnataka", 0)
	// An Easy Split order in Q4 (Feb 2026) — MUST still count as FY income.
	insertOrder(uuid.NewString(), "HC002", time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC), 500, 25, 0, "Karnataka", 44000)
	// Outside the FY — excluded.
	insertOrder(uuid.NewString(), "HC003", time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC), 700, 35, 0, "Karnataka", 0)

	// Expenses: 800 gas in-FY, 999 outside FY.
	for _, e := range []struct {
		amt  float64
		date string
	}{{800, "2025-07-01"}, {999, "2025-03-01"}} {
		w := expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
			"category": "gas", "amount": e.amt, "expenseDate": e.date,
		})
		require.Equal(t, http.StatusCreated, w.Code)
	}

	w := expenseReq(t, userID, http.MethodGet, "/chef/tax/fy-statement?year=2025", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var stmt services.FYStatement
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &stmt))

	assert.Equal(t, 2, stmt.OrdersCount)
	assert.Equal(t, 1500.0, stmt.FoodRevenue)
	assert.Equal(t, 75.0, stmt.GSTCollected)
	assert.Equal(t, 20.0, stmt.Tips)
	// gross = (1000+50+20) + (500+25) = 1595
	assert.Equal(t, 1595.0, stmt.GrossReceipts)
	// commission = 6% of item revenue: 60 + 30 = 90
	assert.Equal(t, 90.0, stmt.PlatformCommission)
	// TDS = 1% of gross: 10.70 + 5.25 = 15.95
	assert.Equal(t, 15.95, stmt.TDSWithheld)
	// net = 1595 − 90 − 15.95 = 1489.05
	assert.Equal(t, 1489.05, stmt.NetEarnings)
	assert.Equal(t, 800.0, stmt.TotalExpenses)
	assert.Equal(t, 689.05, stmt.NetIncome)

	// Quarters: Q1 holds order 1, Q4 holds order 2.
	require.Len(t, stmt.Quarters, 4)
	assert.Equal(t, 1, stmt.Quarters[0].OrdersCount)
	assert.Equal(t, 1070.0, stmt.Quarters[0].Gross)
	assert.Equal(t, 1, stmt.Quarters[3].OrdersCount)
	assert.Equal(t, 525.0, stmt.Quarters[3].Gross)
	assert.Equal(t, 0, stmt.Quarters[1].OrdersCount)
}

func TestExpenseSummaryFollowsAUTaxYear(t *testing.T) {
	db := setupExpenseDB(t)
	userID := uuid.New()
	chefID := seedExpenseChef(t, db, userID)
	require.NoError(t, db.Exec(`UPDATE chef_profiles SET payout_country = 'AU' WHERE id = ?`, chefID.String()).Error)

	for _, e := range []struct {
		amt  float64
		date string
	}{{40, "2025-06-30"}, {25, "2025-07-01"}, {60, "2026-06-30"}} {
		w := expenseReq(t, userID, http.MethodPost, "/chef/expenses", gin.H{
			"category": "gas", "amount": e.amt, "expenseDate": e.date,
		})
		require.Equal(t, http.StatusCreated, w.Code, w.Body.String())
		var created struct {
			Expense models.ChefExpense `json:"expense"`
		}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &created))
		assert.Equal(t, "AUD", created.Expense.Currency)
	}

	w := expenseReq(t, userID, http.MethodGet, "/chef/expenses/summary?year=2025", nil)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var summary services.ExpenseSummary
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &summary))

	assert.Equal(t, 85.0, summary.Total, "1 Jul 2025 – 30 Jun 2026 only")
	assert.Equal(t, "AUD", summary.Currency)
	require.Len(t, summary.ByMonth, 2)
	assert.Equal(t, "2025-07", summary.ByMonth[0].Month)
	assert.Equal(t, "2026-06", summary.ByMonth[1].Month)
}
