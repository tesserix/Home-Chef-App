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

	// Transient states stay billable ON PURPOSE. The query is windowed on
	// delivered_at and each (chef, week) statement is frozen after one generation,
	// so an order skipped for its own week is never billed on any later one.
	// Excluding a state the order can still LEAVE would silently lose the chef that
	// money — the failure mode the naive "only bill release_eligible/released" fix
	// would introduce.
	assert.True(t, billed["PAYABLE-awaiting"],
		"excluding a transient state would strand this order forever — the week is frozen")
	assert.True(t, billed["PAYABLE-disputed"],
		"a dispute resolved in the chef's favour must not cost them the order")

	assert.Len(t, rows, 5)
}
