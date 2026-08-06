package services

// statement_catchup_test.go — #927. The weekly statement and the escrow hold
// machine are two ways to pay a chef for the same order. These pin the two
// failure modes that make connecting them dangerous:
//
//   OVERPAY  — an order billed on a statement is billed (or credited) again.
//   UNDERPAY — an order skipped by its own week's frozen statement is never
//              paid at all, because no later statement is windowed to see it.
//
// The catch-up is what makes excluding a transient hold state safe. Without it,
// narrowing the statement predicate silently loses chefs money.

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

func setupCatchupDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)
	createTablesFor(t, db,
		&models.Order{}, &models.ChefProfile{}, &models.ChefBonus{}, &models.WeeklyStatement{})
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })

	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, user_id, business_name, state) VALUES (?,?,?,?)`,
		chefID.String(), uuid.New().String(), "Saffron Home Kitchen", "KA").Error)
	return db, chefID
}

var catchupWeek = time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)

func addCatchupOrder(t *testing.T, db *gorm.DB, chefID uuid.UUID, num, hold string, billed *uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	var billedStr any
	if billed != nil {
		billedStr = billed.String()
	}
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax,
		   total, commission_rate, payout_hold_status, billed_statement_id, delivery_address_state)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), num, chefID.String(), "delivered", catchupWeek.Add(36*time.Hour),
		500.0, 25.0, 630.0, 0.06, hold, billedStr, "KA").Error)
	return id
}

func addCatchupStatement(t *testing.T, db *gorm.DB, chefID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO weekly_statements (id, chef_id, user_id, week_start, week_end, status, net_payout)
		 VALUES (?,?,?,?,?,?,?)`,
		id.String(), chefID.String(), uuid.New().String(),
		catchupWeek, catchupWeek.AddDate(0, 0, 7), "pending", 1000.0).Error)
	return id
}

// An order held past its statement week gets credited once it clears.
func TestStatementCatchup_CreditsOrderHeldPastItsWeek(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)

	// Already billed by that statement — must NOT be credited again.
	addCatchupOrder(t, db, chefID, "BILLED", "release_eligible", &stmt)
	// Was awaiting confirmation when the statement closed; has since cleared.
	cleared := addCatchupOrder(t, db, chefID, "CLEARED-LATE", "release_eligible", nil)
	// Still disputed — not owed yet, must not be credited.
	addCatchupOrder(t, db, chefID, "STILL-DISPUTED", "disputed", nil)
	// Withheld — terminal, never owed.
	addCatchupOrder(t, db, chefID, "WITHHELD", "withheld", nil)

	reconcileStatementCatchup()

	var bonuses []models.ChefBonus
	require.NoError(t, db.Find(&bonuses).Error)
	require.Len(t, bonuses, 1, "only the order that became payable after its week closed")
	assert.Equal(t, models.ChefBonusStatementCatchup, bonuses[0].Kind)
	assert.Equal(t, StatementCatchupSourceKey(cleared), bonuses[0].SourceKey)
	assert.Contains(t, bonuses[0].Reason, "CLEARED-LATE")

	// Credited at the chef's real net payout — gross 525 less 6% commission on
	// food and 1% TDS on gross.
	assert.InDelta(t, 489.75, bonuses[0].Amount, 0.01)

	// And the order is now stamped, so it leaves the candidate set.
	var got models.Order
	require.NoError(t, db.First(&got, "id = ?", cleared.String()).Error)
	require.NotNil(t, got.BilledStatementID)
}

// The cron runs hourly forever — it must never re-credit.
//
// The statement carries a stamped order because that is what a post-change
// statement always looks like: upsertWeeklyStatement only creates one when it
// bills at least one order, and it stamps them in the same transaction. A
// statement with NO stamps is by definition a historical one, and the backfill
// claims its orders rather than crediting them (pinned separately below).
func TestStatementCatchup_Idempotent(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	addCatchupOrder(t, db, chefID, "BILLED", "release_eligible", &stmt)
	addCatchupOrder(t, db, chefID, "CLEARED-LATE", "release_eligible", nil)

	reconcileStatementCatchup()
	reconcileStatementCatchup()
	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 1, count, "one order, one credit, however often the cron runs")
}

// An order whose week has NO statement yet must not be credited — the next
// statement will bill it normally, and crediting it too would pay it twice.
func TestStatementCatchup_SkipsOrderWithNoStatementYet(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	addCatchupOrder(t, db, chefID, "NOT-YET-BILLED", "release_eligible", nil)

	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count,
		"no statement has closed on this order — its next one bills it")
}

// THE dangerous case. Statements generated before stamping existed have no
// stamps, so every order they billed reads as unsettled. Without the backfill
// the catch-up would credit all of them a second time.
func TestStatementCatchup_BackfillPreventsDoublePayOnHistoricalStatements(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	addCatchupStatement(t, db, chefID) // pre-change: no order carries its id
	addCatchupOrder(t, db, chefID, "HISTORICAL-1", "release_eligible", nil)
	addCatchupOrder(t, db, chefID, "HISTORICAL-2", "", nil)

	reconcileStatementCatchup()

	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 0, count,
		"these were billed by the historical statement — crediting them would pay twice")

	// They are now stamped, so the question never arises again.
	var unstamped int64
	db.Model(&models.Order{}).Where("billed_statement_id IS NULL").Count(&unstamped)
	assert.EqualValues(t, 0, unstamped)
}

// The backfill reproduces the OLD broad predicate, so it must never run against a
// statement that already has stamps — it would mark orders that statement
// deliberately SKIPPED as settled, suppressing their catch-up forever.
func TestStatementCatchup_BackfillLeavesStampedStatementsAlone(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	addCatchupOrder(t, db, chefID, "BILLED", "release_eligible", &stmt) // stamp exists
	skipped := addCatchupOrder(t, db, chefID, "SKIPPED-AWAITING", "awaiting_customer_confirmation", nil)

	n, err := BackfillBilledStatementIDs(db)
	require.NoError(t, err)
	assert.EqualValues(t, 0, n, "a statement that already has stamps is not a historical one")

	var got models.Order
	require.NoError(t, db.First(&got, "id = ?", skipped.String()).Error)
	assert.Nil(t, got.BilledStatementID,
		"an order the statement skipped must stay unsettled so its catch-up can fire")
}

// End to end: the order the statement skips is exactly the order the catch-up
// pays — so narrowing the predicate costs the chef nothing.
func TestStatementCatchup_ClosesTheGapTheStatementOpens(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	addCatchupOrder(t, db, chefID, "BILLED", "release_eligible", &stmt)
	held := addCatchupOrder(t, db, chefID, "HELD", "awaiting_customer_confirmation", nil)

	// While still awaiting, nothing is owed.
	reconcileStatementCatchup()
	var count int64
	db.Model(&models.ChefBonus{}).Count(&count)
	require.EqualValues(t, 0, count)

	// The customer confirms after the week closed.
	require.NoError(t, db.Model(&models.Order{}).Where("id = ?", held.String()).
		Update("payout_hold_status", string(models.PayoutHoldReleaseEligible)).Error)

	reconcileStatementCatchup()
	db.Model(&models.ChefBonus{}).Count(&count)
	assert.EqualValues(t, 1, count, "the money the frozen statement could not pay reaches the chef")
}

// A leg the chef drove themselves: the fee is priced from the chef's own rates
// and is their income, so the catch-up credit must carry it.
func TestStatementCatchup_CreditsTheFeeOnAChefCarriedLeg(t *testing.T) {
	db, chefID := setupCatchupDB(t)
	stmt := addCatchupStatement(t, db, chefID)
	// A stamped order keeps the historical backfill off this statement.
	addCatchupOrder(t, db, chefID, "BILLED", "release_eligible", &stmt)

	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, order_number, chef_id, status, delivered_at, subtotal, tax,
		   total, commission_rate, payout_hold_status, delivery_address_state,
		   delivery_fee, fulfillment_type)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		id.String(), "SELF-DELIVERED", chefID.String(), "delivered", catchupWeek.Add(36*time.Hour),
		500.0, 25.0, 630.0, 0.06, "release_eligible", "KA", 70.0, "chef_delivery").Error)

	reconcileStatementCatchup()

	var bonuses []models.ChefBonus
	require.NoError(t, db.Find(&bonuses).Error)
	require.Len(t, bonuses, 1)
	// gross 500 + 25 + 70 = 595; commission 6% of FOOD only = 30; TDS 1% = 5.95.
	assert.InDelta(t, 559.05, bonuses[0].Amount, 0.01)
}
