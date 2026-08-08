package services

// statement_split_sunset_test.go — #1087. Once a chef's orders settle through
// Easy Split, the weekly statement stops being a payment instrument for them and
// becomes a record. The exclusion that does this already lives in the statement
// query; what was never pinned is the edge it creates.
//
// The trap being guarded is a ZERO-VALUE statement: a fully-split chef whose
// query returns no rows must produce no statement row at all, because a ₹0 row
// lands in the admin payout queue as something to prepare, approve and disburse.

import (
	"context"
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

func newSplitSunsetDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	createTablesFor(t, db, &models.Order{}, &models.ChefProfile{}, &models.WeeklyStatement{})
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

// splitSunsetWeek is a settled week in the past, so nothing depends on wall clock.
func splitSunsetWeek() (time.Time, time.Time) {
	start := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	return start, start.AddDate(0, 0, 7)
}

func seedSunsetChef(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, state) VALUES (?,?,?,?)`,
		chefID.String(), uuid.New().String(), "Saffron Home Kitchen", "KA").Error)
	return chefID
}

// seedSunsetOrder adds one delivered order; splitPaise > 0 marks it as already
// settled to the chef's vendor account at the gateway.
func seedSunsetOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, number string, subtotal float64, splitPaise int) uuid.UUID {
	t.Helper()
	weekStart, _ := splitSunsetWeek()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax, tax_food,
		   total, commission_rate, gateway_split_paise)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), number, chefID.String(), "delivered", weekStart.Add(36*time.Hour),
		subtotal, 0.0, 0.0, subtotal, 0.06, splitPaise).Error)
	return id
}

func TestGenerateWeeklyStatements_FullySplitChefGetsNoStatement(t *testing.T) {
	db := newSplitSunsetDB(t)
	chefID := seedSunsetChef(t, db)
	seedSunsetOrder(t, db, chefID, "SPLIT-1", 500, 42_000)
	seedSunsetOrder(t, db, chefID, "SPLIT-2", 300, 25_000)
	weekStart, weekEnd := splitSunsetWeek()

	issued, err := GenerateWeeklyStatements(context.Background(), weekStart, weekEnd)
	require.NoError(t, err)
	assert.Zero(t, issued)

	var count int64
	require.NoError(t, db.Model(&models.WeeklyStatement{}).
		Where("chef_id = ?", chefID).Count(&count).Error)
	assert.Zero(t, count,
		"a fully-split chef must get NO statement — a ₹0 row is a payout the admin queue asks someone to disburse")
}

func TestGenerateWeeklyStatements_MixedChefBillsOnlyTheUnsplitOrders(t *testing.T) {
	db := newSplitSunsetDB(t)
	chefID := seedSunsetChef(t, db)
	splitID := seedSunsetOrder(t, db, chefID, "SPLIT-1", 500, 42_000)
	unsplitID := seedSunsetOrder(t, db, chefID, "RAIL-B-1", 800, 0)
	weekStart, weekEnd := splitSunsetWeek()

	issued, err := GenerateWeeklyStatements(context.Background(), weekStart, weekEnd)
	require.NoError(t, err)
	require.Equal(t, 1, issued)

	var stmt models.WeeklyStatement
	require.NoError(t, db.Where("chef_id = ?", chefID).First(&stmt).Error)
	assert.Equal(t, 1, stmt.OrdersCount, "only the unsplit order is a statement line")

	// The statement's gross must equal what the one unsplit order earns, to the
	// paisa — a mixed chef paid for a split order here is paid for it twice.
	var subtotal, tax, taxFood, tip, discount, rate float64
	require.NoError(t, db.Raw(
		`SELECT subtotal, tax, tax_food, COALESCE(chef_tip,0), COALESCE(chef_funded_discount,0), commission_rate
		 FROM orders WHERE id = ?`, unsplitID.String()).
		Row().Scan(&subtotal, &tax, &taxFood, &tip, &discount, &rate))
	assert.InDelta(t, subtotal, stmt.GrossRevenue, 0.01)

	// The split order must be left entirely alone by the statement path.
	var billed *string
	require.NoError(t, db.Raw(`SELECT billed_statement_id FROM orders WHERE id = ?`, splitID.String()).
		Row().Scan(&billed))
	assert.Nil(t, billed, "a split order was already settled at the gateway — the statement must not claim it")

	var unsplitBilled *string
	require.NoError(t, db.Raw(`SELECT billed_statement_id FROM orders WHERE id = ?`, unsplitID.String()).
		Row().Scan(&unsplitBilled))
	require.NotNil(t, unsplitBilled)
	assert.Equal(t, stmt.ID.String(), *unsplitBilled)
}

func TestGenerateWeeklyStatements_UnsplitChefIsUnaffected(t *testing.T) {
	db := newSplitSunsetDB(t)
	chefID := seedSunsetChef(t, db)
	seedSunsetOrder(t, db, chefID, "RAIL-B-1", 500, 0)
	seedSunsetOrder(t, db, chefID, "RAIL-B-2", 300, 0)
	weekStart, weekEnd := splitSunsetWeek()

	issued, err := GenerateWeeklyStatements(context.Background(), weekStart, weekEnd)
	require.NoError(t, err)
	require.Equal(t, 1, issued)

	var stmt models.WeeklyStatement
	require.NoError(t, db.Where("chef_id = ?", chefID).First(&stmt).Error)
	assert.Equal(t, 2, stmt.OrdersCount)
	assert.Positive(t, stmt.NetPayout, "Rail B still pays a chef who does not split")
}

func TestGenerateWeeklyStatements_SplitOrderIsNeverPaidOnBothRails(t *testing.T) {
	db := newSplitSunsetDB(t)
	chefID := seedSunsetChef(t, db)
	splitID := seedSunsetOrder(t, db, chefID, "SPLIT-1", 500, 42_000)
	seedSunsetOrder(t, db, chefID, "RAIL-B-1", 800, 0)
	weekStart, weekEnd := splitSunsetWeek()

	// Two runs: the second is the re-generation the cron performs on retry.
	_, err := GenerateWeeklyStatements(context.Background(), weekStart, weekEnd)
	require.NoError(t, err)
	_, err = GenerateWeeklyStatements(context.Background(), weekStart, weekEnd)
	require.NoError(t, err)

	var stmts int64
	require.NoError(t, db.Model(&models.WeeklyStatement{}).Where("chef_id = ?", chefID).Count(&stmts).Error)
	assert.EqualValues(t, 1, stmts, "a re-run must not issue a second statement for the same week")

	var billed *string
	require.NoError(t, db.Raw(`SELECT billed_statement_id FROM orders WHERE id = ?`, splitID.String()).
		Row().Scan(&billed))
	assert.Nil(t, billed, "no number of statement runs may claim an order the gateway already settled")
}
