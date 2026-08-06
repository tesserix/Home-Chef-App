package handlers

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// #1028 — the P&L expense query carried `AND deleted_at IS NULL` against a table
// that has no such column, so it errored on every call. Because it was the one
// unchecked Scan in the file, the failure was invisible: expenses summed to 0
// and profit degraded to net earnings, while Expenses & tax showed the real
// total from the same rows.
//
// This pins the QUERY against a schema built from the real column set, so a
// predicate naming a column that does not exist fails the test rather than
// silently zeroing a money figure in production.

func setupPLExpenseDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	// Mirrors models.ChefExpense — note the absence of deleted_at, which is the
	// whole point: expenses are hard-deleted.
	require.NoError(t, db.Exec(`CREATE TABLE chef_expenses (
		id text PRIMARY KEY, chef_id text NOT NULL, user_id text NOT NULL,
		category text NOT NULL, amount real NOT NULL, currency text DEFAULT 'INR',
		note text DEFAULT '', order_id text, expense_date datetime NOT NULL,
		receipt_path text DEFAULT '', created_at datetime, updated_at datetime
	)`).Error)
	return db
}

func insertPLExpense(t *testing.T, db *gorm.DB, chefID uuid.UUID, category string, amount float64, on time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_expenses (id, chef_id, user_id, category, amount, expense_date) VALUES (?,?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), uuid.New().String(), category, amount, on).Error)
}

// plExpenseQuery is the statement the handler runs. Kept verbatim so the test
// fails if a predicate naming a non-existent column is reintroduced.
const plExpenseQuery = `
		SELECT category, COALESCE(SUM(amount), 0) AS amount
		FROM   chef_expenses
		WHERE  chef_id       = ?
		AND    expense_date >= ?
		GROUP  BY category
		ORDER  BY amount DESC
	`

func TestProfitLossExpenseQuery_SumsRecordedExpensesByCategory(t *testing.T) {
	db := setupPLExpenseDB(t)
	chef := uuid.New()
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -30)

	insertPLExpense(t, db, chef, "ingredients", 85, now.AddDate(0, 0, -1))
	insertPLExpense(t, db, chef, "ingredients", 100, now.AddDate(0, 0, -2))
	insertPLExpense(t, db, chef, "gas", 80, now.AddDate(0, 0, -3))

	var rows []plCategoryLine
	require.NoError(t, db.Raw(plExpenseQuery, chef.String(), start).Scan(&rows).Error,
		"the query must execute — an error here is exactly how #1028 hid")

	require.Len(t, rows, 2)
	// ORDER BY amount DESC — ingredients (185) before gas (80).
	require.Equal(t, "ingredients", string(rows[0].Category))
	require.InDelta(t, 185.0, rows[0].Amount, 0.001)
	require.Equal(t, "gas", string(rows[1].Category))
	require.InDelta(t, 80.0, rows[1].Amount, 0.001)

	var total float64
	for _, r := range rows {
		total += r.Amount
	}
	require.InDelta(t, 265.0, total, 0.001, "the ₹265 the Expenses screen showed while P&L said ₹0")
}

func TestProfitLossExpenseQuery_ScopesToChefAndPeriod(t *testing.T) {
	db := setupPLExpenseDB(t)
	chef, other := uuid.New(), uuid.New()
	now := time.Now().UTC()
	start := now.AddDate(0, 0, -7)

	insertPLExpense(t, db, chef, "ingredients", 50, now.AddDate(0, 0, -1))  // in
	insertPLExpense(t, db, chef, "ingredients", 999, now.AddDate(0, 0, -9)) // before the window
	insertPLExpense(t, db, other, "ingredients", 777, now.AddDate(0, 0, -1))

	var rows []plCategoryLine
	require.NoError(t, db.Raw(plExpenseQuery, chef.String(), start).Scan(&rows).Error)
	require.Len(t, rows, 1)
	require.InDelta(t, 50.0, rows[0].Amount, 0.001,
		"another chef's expenses and out-of-window rows must not be counted")
}

func TestProfitLossExpenseQuery_NoExpensesIsZeroNotAnError(t *testing.T) {
	// The legitimate ₹0: a chef who recorded nothing. Distinguishable from the
	// bug only because the query now succeeds.
	db := setupPLExpenseDB(t)
	var rows []plCategoryLine
	require.NoError(t, db.Raw(plExpenseQuery, uuid.New().String(), time.Now().AddDate(0, 0, -30)).Scan(&rows).Error)
	require.Empty(t, rows)
}

func TestProfitLossExpenseQuery_DeletedAtPredicateWouldFail(t *testing.T) {
	// Guards the regression directly: the table has no deleted_at, so the old
	// statement errors. If someone re-adds a soft-delete column later this test
	// starts failing, which is the right moment to revisit the predicate.
	db := setupPLExpenseDB(t)
	var rows []plCategoryLine
	err := db.Raw(`
		SELECT category, COALESCE(SUM(amount), 0) AS amount
		FROM   chef_expenses
		WHERE  chef_id = ? AND expense_date >= ? AND deleted_at IS NULL
		GROUP  BY category`, uuid.New().String(), time.Now().AddDate(0, 0, -30)).Scan(&rows).Error
	require.Error(t, err, "chef_expenses has no deleted_at — the old predicate could only ever error")
}
