package services

// statement_payout_guard_test.go — #927. The weekly statement and the escrow hold
// machine are two independent ways to pay a chef for the same order, and nothing
// reconciles them. The statement builder selected on delivery alone, so an order
// whose payout had been refunded, withheld or clawed back on the hold path was
// still, unconditionally, a billable line on that chef's statement.
//
// Both paths are dark today (escrow flags off, ExecuteBatch has no cron), which
// is the only reason this has not paid anyone twice — 8 delivered orders sit at
// release_eligible AND are billed on the 2 existing statements.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupStatementRowsDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	createTablesFor(t, db, &models.Order{}, &models.ChefProfile{})
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func TestLoadStatementOrderRows_ExcludesUnpayableOrders(t *testing.T) {
	db := setupStatementRowsDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, state) VALUES (?,?,?,?)`,
		chefID.String(), uuid.New().String(), "Saffron Home Kitchen", "KA").Error)

	weekStart := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	weekEnd := weekStart.AddDate(0, 0, 7)
	delivered := weekStart.Add(36 * time.Hour)

	add := func(num, hold string, refundedAt *time.Time) {
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax, total,
			   payout_hold_status, refunded_at, commission_rate)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.NewString(), num, chefID.String(), "delivered", delivered,
			500.0, 25.0, 630.0, hold, refundedAt, 0.06).Error)
	}
	now := time.Now()

	add("PAYABLE-none", "", nil)                     // non-gateway order: hold never set
	add("PAYABLE-eligible", "release_eligible", nil) // confirmed, awaiting admin release
	add("PAYABLE-released", "released", nil)
	add("PAYABLE-awaiting", "awaiting_customer_confirmation", nil) // transient — still billed
	add("PAYABLE-disputed", "disputed", nil)                       // transient — still billed
	add("SKIP-withheld", "withheld", nil)                          // terminal: payout blocked
	add("SKIP-reversed", "reversed", nil)                          // terminal: clawed back
	add("SKIP-refunded", "release_eligible", &now)                 // issue-path refund keeps status 'delivered'

	rows, err := loadStatementOrderRows(weekStart, weekEnd)
	require.NoError(t, err)

	billed := map[string]bool{}
	for _, r := range rows {
		billed[r.OrderNumber] = true
	}

	assert.True(t, billed["PAYABLE-none"], "a non-gateway order has no hold and must still pay")
	assert.True(t, billed["PAYABLE-eligible"])
	assert.True(t, billed["PAYABLE-released"])

	assert.False(t, billed["SKIP-withheld"], "a withheld payout must never bill")
	assert.False(t, billed["SKIP-reversed"], "a clawed-back payout must never bill")
	assert.False(t, billed["SKIP-refunded"],
		"a refunded order must never bill — the issue path leaves status='delivered'")

	// Transient states are now EXCLUDED too — the customer has not confirmed, or is
	// contesting, so the money is still held and can still go back to them. Billing
	// them was the overpay risk this issue is about.
	//
	// This is only safe because reconcileStatementCatchup exists. The query is
	// windowed on delivered_at and each (chef, week) statement is frozen after one
	// generation, so an order skipped here is never billed by any later statement;
	// the catch-up credit settles it once it clears. Excluding a transient state
	// WITHOUT that credit — the shape suggested on #927 — would silently lose the
	// chef the whole order. Pinned end-to-end in
	// TestStatementCatchup_ClosesTheGapTheStatementOpens.
	assert.False(t, billed["PAYABLE-awaiting"],
		"the customer has not confirmed — the money is still held")
	assert.False(t, billed["PAYABLE-disputed"],
		"a contested order can still refund the customer; the catch-up pays it if it clears")

	assert.Len(t, rows, 3)
}

// An order already settled — by an earlier statement or by the catch-up credit —
// must never be billed again. This is the half that makes the statement path and
// the hold path reconcilable rather than merely independent.
func TestLoadStatementOrderRows_ExcludesAlreadySettledOrders(t *testing.T) {
	db := setupStatementRowsDB(t)
	chefID := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id, business_name, state) VALUES (?,?,?,?)`,
		chefID.String(), uuid.New().String(), "Saffron Home Kitchen", "KA").Error)

	weekStart := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	delivered := weekStart.Add(36 * time.Hour)
	priorStmt := uuid.New()

	for _, tc := range []struct {
		num    string
		billed any
	}{
		{"UNSETTLED", nil},
		{"ALREADY-SETTLED", priorStmt.String()},
	} {
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax, total,
			   payout_hold_status, commission_rate, billed_statement_id)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			uuid.NewString(), tc.num, chefID.String(), "delivered", delivered,
			500.0, 25.0, 630.0, "release_eligible", 0.06, tc.billed).Error)
	}

	rows, err := loadStatementOrderRows(weekStart, weekStart.AddDate(0, 0, 7))
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "UNSETTLED", rows[0].OrderNumber)
}
