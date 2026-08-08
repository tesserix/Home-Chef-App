package services

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

// setupSessionDB builds the slice of schema the session + clone paths touch.
// Column sets mirror production closely enough that the INSERT…SELECT clone
// exercises the real code path rather than a simplified one.
func setupSessionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	require.NoError(t, err)

	stmts := []string{
		`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT, business_name TEXT DEFAULT '',
			mode TEXT DEFAULT 'live', first_live_at DATETIME, active_test_session_id TEXT,
			is_verified BOOLEAN DEFAULT 0, is_active BOOLEAN DEFAULT 1,
			rating REAL DEFAULT 0, total_reviews INTEGER DEFAULT 0,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE chef_test_sessions (id TEXT PRIMARY KEY, chef_id TEXT, session_no INTEGER,
			status TEXT DEFAULT 'open', reason TEXT DEFAULT '', order_window_days INTEGER DEFAULT 30,
			cloned_at DATETIME, clone_summary TEXT DEFAULT '{}', forced_blockers TEXT DEFAULT '',
			opened_by_id TEXT, opened_at DATETIME, closed_by_id TEXT, closed_at DATETIME,
			purged_at DATETIME, created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE orders (id TEXT PRIMARY KEY, chef_id TEXT, customer_id TEXT,
			order_number TEXT DEFAULT '', status TEXT DEFAULT 'pending',
			total REAL DEFAULT 0, payout_settled_at DATETIME, payout_hold_status TEXT DEFAULT '',
			gateway_order_id TEXT DEFAULT '', gateway_payment_id TEXT DEFAULT '',
			payout_transfer_id TEXT DEFAULT '', refund_id TEXT DEFAULT '',
			mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE menu_items (id TEXT PRIMARY KEY, chef_id TEXT, name TEXT DEFAULT '',
			price REAL DEFAULT 0, is_approved BOOLEAN DEFAULT 1,
			mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
			created_at DATETIME, updated_at DATETIME, deleted_at DATETIME)`,
		`CREATE TABLE chef_schedules (id TEXT PRIMARY KEY, chef_id TEXT, day_of_week INTEGER DEFAULT 0,
			mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT)`,
		`CREATE TABLE meal_plans (id TEXT PRIMARY KEY, chef_id TEXT, status TEXT DEFAULT 'completed',
			mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE weekly_menus (id TEXT PRIMARY KEY, chef_id TEXT, is_published BOOLEAN DEFAULT 0,
			published_at DATETIME, mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE weekly_menu_items (id TEXT PRIMARY KEY, chef_id TEXT, day_of_week INTEGER DEFAULT 0,
			slot TEXT DEFAULT 'lunch', variant TEXT DEFAULT 'veg', name TEXT DEFAULT '',
			price REAL DEFAULT 0, mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE daily_menus (id TEXT PRIMARY KEY, chef_id TEXT, date DATE,
			is_published BOOLEAN DEFAULT 0, published_at DATETIME,
			mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE daily_menu_items (id TEXT PRIMARY KEY, daily_menu_id TEXT, chef_id TEXT,
			date DATE, slot TEXT DEFAULT 'lunch', variant TEXT DEFAULT 'veg', name TEXT DEFAULT '',
			price REAL DEFAULT 0, sort_order INTEGER DEFAULT 0,
			mode TEXT DEFAULT 'live', test_session_id TEXT, cloned_from_id TEXT,
			created_at DATETIME, updated_at DATETIME)`,
		// The clone enqueues nothing; these exist so the test can PROVE it.
		`CREATE TABLE outbox_events (id TEXT PRIMARY KEY, subject TEXT)`,
		`CREATE TABLE notifications (id TEXT PRIMARY KEY, user_id TEXT)`,
	}
	for _, s := range stmts {
		require.NoError(t, db.Exec(s).Error)
	}

	// The production unique indexes, which the fixtures previously omitted
	// entirely. Without them the INSERT…SELECT clone never meets the constraint
	// it actually violates in prod, which is how the mode-blind
	// idx_weekly_menus_chef_id shipped. These mirror the postMigrate block in
	// database.go — if that block changes, change these with it.
	for _, s := range prodUniqueIndexes {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

// prodUniqueIndexes are the partitioned-table unique indexes as they exist in
// Postgres after database.go's postMigrate block runs. A clone writes a second
// row for the same natural key differing only by mode, so every one of these
// must be either mode-scoped or derived per session.
var prodUniqueIndexes = []string{
	`CREATE UNIQUE INDEX idx_orders_order_number ON orders (order_number)`,
	`CREATE UNIQUE INDEX idx_weekly_menus_chef_live ON weekly_menus (chef_id) WHERE mode = 'live'`,
	`CREATE UNIQUE INDEX idx_weekly_menus_chef_test ON weekly_menus (chef_id, test_session_id) WHERE mode = 'test'`,
	`CREATE UNIQUE INDEX idx_weekly_cell_live ON weekly_menu_items (chef_id, day_of_week, slot, variant) WHERE mode = 'live'`,
	`CREATE UNIQUE INDEX idx_weekly_cell_test ON weekly_menu_items (chef_id, day_of_week, slot, variant, test_session_id) WHERE mode = 'test'`,
	`CREATE UNIQUE INDEX idx_daily_menu_chef_date_live ON daily_menus (chef_id, date) WHERE mode = 'live'`,
	`CREATE UNIQUE INDEX idx_daily_menu_chef_date_test ON daily_menus (chef_id, date, test_session_id) WHERE mode = 'test'`,
}

func seedLiveChef(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	past := time.Now().AddDate(0, -6, 0)
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, business_name, mode, first_live_at) VALUES (?,?,?,?)`,
		id.String(), "Amma ka Kitchen", "live", past).Error)
	return id
}

func seedChefWithData(t *testing.T, db *gorm.DB) uuid.UUID {
	t.Helper()
	chefID := seedLiveChef(t, db)
	for i := 0; i < 3; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO menu_items (id, chef_id, name, price, mode) VALUES (?,?,?,?,?)`,
			uuid.New().String(), chefID.String(), "Dish", 120.0, "live").Error)
	}
	// One order inside the 30-day window, one well outside it.
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, order_number, status, total, gateway_order_id, gateway_payment_id, mode, created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "HC-1", "delivered", 500.0,
		"order_live_1", "pay_live_1", "live", time.Now().AddDate(0, 0, -5)).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, order_number, status, total, mode, created_at)
		 VALUES (?,?,?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "HC-OLD", "delivered", 300.0,
		"live", time.Now().AddDate(0, 0, -200)).Error)
	return chefID
}

// Flipping a kitchen mid-service strands real customers holding an order they
// can no longer see. The blocker list is what the admin UI shows, so it must
// NAME each obstruction rather than return a bare boolean.
func TestFlipBlockedWhileOrdersInFlight(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, status, mode, created_at) VALUES (?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "preparing", "live", time.Now()).Error)

	blockers := TestFlipBlockers(db, chefID)
	require.NotEmpty(t, blockers, "an active order must block a live→test flip")
	require.Contains(t, blockers[0], "active order")

	_, err := OpenTestSession(db, chefID, uuid.New(), "debugging", 30)
	require.True(t, errors.Is(err, ErrFlipBlocked), "open must refuse with ErrFlipBlocked, got %v", err)

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.False(t, chef.IsTestMode(), "a refused flip must leave the kitchen live")
}

// A kitchen that trades every day always has something in flight, so a blanket
// refusal means an established kitchen can never be sandboxed at all. A forced
// flip parks the live work rather than refusing, and records what it parked:
// a flip whose consequences were never written down cannot be audited later.
func TestForcedFlipParksInFlightWorkAndRecordsIt(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, status, mode, created_at) VALUES (?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "preparing", "live", time.Now()).Error)

	s, err := ForceOpenTestSession(db, chefID, uuid.New(), "sandbox the payout rails", 30)
	require.NoError(t, err, "a forced flip must not be refused by blockers")
	require.Contains(t, s.ForcedBlockers, "active order",
		"the session must record what was in flight when it was forced past")

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.True(t, chef.IsTestMode(), "a forced flip must put the kitchen in test mode")

	// The live order is left exactly as it was. Parking is derived from the
	// chef's mode, so returning to live restores it with nothing to restore.
	var status string
	require.NoError(t, db.Raw(
		`SELECT status FROM orders WHERE chef_id = ? AND mode = 'live'`,
		chefID.String()).Scan(&status).Error)
	require.Equal(t, "preparing", status, "a forced flip must not rewrite live work")
}

// An UNforced flip records nothing, so a non-empty forced_blockers is proof the
// flip was forced rather than clean.
func TestCleanFlipRecordsNoForcedBlockers(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)

	s, err := OpenTestSession(db, chefID, uuid.New(), "nothing in flight", 30)
	require.NoError(t, err)
	require.Empty(t, s.ForcedBlockers)
}

func TestFlipAllowedOnceSettled(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)

	s, err := OpenTestSession(db, chefID, uuid.New(), "reproducing #123", 30)
	require.NoError(t, err)
	require.Equal(t, 1, s.SessionNo)
	require.Equal(t, models.TestSessionOpen, s.Status)

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.True(t, chef.IsTestMode(), "opening a session must put the kitchen in test mode")
	require.NotNil(t, chef.ActiveTestSessionID)
	require.Equal(t, s.ID, *chef.ActiveTestSessionID)
}

// Each flip is its own investigation. Overwriting the previous session would
// destroy the evidence from the last one.
func TestSecondFlipOpensSessionTwoAndRetainsTheFirst(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedLiveChef(t, db)
	adminID := uuid.New()

	s1, err := OpenTestSession(db, chefID, adminID, "first", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))
	s2, err := OpenTestSession(db, chefID, adminID, "second", 30)
	require.NoError(t, err)

	require.Equal(t, 2, s2.SessionNo)

	var first models.ChefTestSession
	require.NoError(t, db.First(&first, "id = ?", s1.ID).Error)
	require.Equal(t, models.TestSessionClosed, first.Status)
	require.NotNil(t, first.ClosedAt, "the first session must be retained and stamped closed")
}

// Going live for the first time stamps FirstLiveAt, which is what turns a
// born-test kitchen (hidden) into an established one (shown as closed).
func TestCloseStampsFirstLiveAtOnce(t *testing.T) {
	db := setupSessionDB(t)
	adminID := uuid.New()
	chefID := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO chef_profiles (id, business_name, mode) VALUES (?,?,?)`,
		chefID.String(), "New Kitchen", "test").Error)

	require.NoError(t, CloseTestSession(db, chefID, adminID))
	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.NotNil(t, chef.FirstLiveAt)
	stamped := *chef.FirstLiveAt

	_, err := OpenTestSession(db, chefID, adminID, "again", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.True(t, chef.FirstLiveAt.Equal(stamped), "FirstLiveAt must be stamped once and never moved")
}
