package services

import (
	"testing"

	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// The clone must reproduce the kitchen's setup so the sandbox behaves like the
// real thing, and reach back far enough to include the order being debugged —
// but no further, or a busy kitchen's clone would never finish.
func TestCloneCopiesConfigAndWindowedOrders(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db) // 3 menu items; 1 order 5d old, 1 order 200d old

	session := &models.ChefTestSession{ID: uuid.New(), ChefID: chefID, SessionNo: 1}
	summary, err := CloneChefIntoSession(db, chefID, session, 30)
	require.NoError(t, err)
	require.Contains(t, summary, `"menu_items":3`, "all menu items must be cloned: %s", summary)
	require.Contains(t, summary, `"orders":1`, "only orders inside the window may be cloned: %s", summary)

	var items int64
	require.NoError(t, db.Table("menu_items").
		Where("mode = ? AND test_session_id = ?", models.ChefModeTest, session.ID).
		Count(&items).Error)
	require.EqualValues(t, 3, items)

	// The originals must be untouched — the clone reads, it never moves.
	var liveItems int64
	require.NoError(t, db.Table("menu_items").
		Where("mode = ?", models.ChefModeLive).Count(&liveItems).Error)
	require.EqualValues(t, 3, liveItems, "cloning must not consume the live rows")
}

// seedChefMenus gives a kitchen the tiffin configuration that carries unique
// constraints: a weekly template and a dated daily menu, both with dishes.
func seedChefMenus(t *testing.T, db *gorm.DB, chefID uuid.UUID) {
	t.Helper()
	require.NoError(t, db.Exec(
		`INSERT INTO weekly_menus (id, chef_id, is_published, mode) VALUES (?,?,?,?)`,
		uuid.New().String(), chefID.String(), true, "live").Error)
	for _, day := range []int{1, 2} {
		require.NoError(t, db.Exec(
			`INSERT INTO weekly_menu_items (id, chef_id, day_of_week, slot, variant, name, price, mode)
			 VALUES (?,?,?,?,?,?,?,?)`,
			uuid.New().String(), chefID.String(), day, "lunch", "veg", "Dal Chawal", 140.0, "live").Error)
	}

	dailyID := uuid.New()
	date := time.Now().Format("2006-01-02")
	require.NoError(t, db.Exec(
		`INSERT INTO daily_menus (id, chef_id, date, is_published, mode) VALUES (?,?,?,?,?)`,
		dailyID.String(), chefID.String(), date, true, "live").Error)
	for _, dish := range []string{"Rajma", "Jeera Rice"} {
		require.NoError(t, db.Exec(
			`INSERT INTO daily_menu_items (id, daily_menu_id, chef_id, date, slot, variant, name, price, mode)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			uuid.New().String(), dailyID.String(), chefID.String(), date,
			"lunch", "veg", dish, 120.0, "live").Error)
	}
}

// The regression this whole change exists for. The clone writes a second row
// for the same natural key differing only by mode, so any unique index that
// omits mode rejects it — in prod that was idx_weekly_menus_chef_id, and
// opening a session for a chef with a weekly menu failed outright with 23505.
//
// TWO consecutive sessions, because a closed session's test rows survive until
// an explicit purge: a mode-only constraint would pass the first open and fail
// the second against session 1's leftovers.
func TestCloneSurvivesUniqueConstraintsAcrossSessions(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)
	seedChefMenus(t, db, chefID)
	adminID := uuid.New()

	s1, err := OpenTestSession(db, chefID, adminID, "first", 30)
	require.NoError(t, err, "a kitchen with a weekly menu must be able to enter test mode")
	require.NoError(t, CloseTestSession(db, chefID, adminID))

	// Deliberately NOT purged — session 1's test rows are still present.
	s2, err := OpenTestSession(db, chefID, adminID, "second", 30)
	require.NoError(t, err, "a second session must not collide with the first session's rows")
	require.NoError(t, CloseTestSession(db, chefID, adminID))

	// The live originals are untouched and still singular.
	var liveWeekly int64
	require.NoError(t, db.Table("weekly_menus").
		Where("chef_id = ? AND mode = ?", chefID.String(), models.ChefModeLive).
		Count(&liveWeekly).Error)
	require.EqualValues(t, 1, liveWeekly, "the live weekly menu must survive untouched")

	// Each session got its own copy.
	for _, s := range []*models.ChefTestSession{s1, s2} {
		var n int64
		require.NoError(t, db.Table("weekly_menus").
			Where("test_session_id = ?", s.ID).Count(&n).Error)
		require.EqualValues(t, 1, n, "session %s must have its own weekly menu", s.ID)
	}
}

// A cloned menu with no dishes in it is useless for reproducing a menu bug.
// weekly_menu_items and daily_menu_items are in partitionedTables — so purge
// removes them — but were never added to the clone.
func TestCloneCopiesMenuItems(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)
	seedChefMenus(t, db, chefID)

	session, err := OpenTestSession(db, chefID, uuid.New(), "menu bug", 30)
	require.NoError(t, err)

	var weeklyItems, dailyItems int64
	require.NoError(t, db.Table("weekly_menu_items").
		Where("test_session_id = ?", session.ID).Count(&weeklyItems).Error)
	require.EqualValues(t, 2, weeklyItems, "the weekly template's dishes must be cloned")

	require.NoError(t, db.Table("daily_menu_items").
		Where("test_session_id = ?", session.ID).Count(&dailyItems).Error)
	require.EqualValues(t, 2, dailyItems, "the daily menu's dishes must be cloned")

	// The child rows must hang off the CLONED parent, not the live one, or the
	// sandbox menu reads through to live data.
	var orphans int64
	require.NoError(t, db.Raw(
		`SELECT count(*) FROM daily_menu_items i
		 WHERE i.test_session_id = ?
		   AND i.daily_menu_id NOT IN (SELECT id FROM daily_menus WHERE test_session_id = ?)`,
		session.ID, session.ID).Scan(&orphans).Error)
	require.Zero(t, orphans, "cloned daily items must point at the cloned daily menu")
}

// Cloned rows are historical replicas. If they carried live gateway ids they
// could be charged or refunded against a real payment.
func TestClonedOrdersCarryNoGatewayIdentifiers(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)
	session := &models.ChefTestSession{ID: uuid.New(), ChefID: chefID, SessionNo: 1}
	_, err := CloneChefIntoSession(db, chefID, session, 30)
	require.NoError(t, err)

	type row struct {
		ID                string
		RazorpayOrderID   string
		RazorpayPaymentID string
		ClonedFromID      *string
		TestSessionID     *string
	}
	var rows []row
	require.NoError(t, db.Raw(
		`SELECT id, razorpay_order_id, razorpay_payment_id, cloned_from_id, test_session_id
		 FROM orders WHERE mode = ? AND cloned_from_id IS NOT NULL`, models.ChefModeTest).
		Scan(&rows).Error)
	require.NotEmpty(t, rows, "the clone must have produced at least one order")

	for _, r := range rows {
		require.Empty(t, r.RazorpayOrderID, "cloned order %s kept a gateway order id", r.ID)
		require.Empty(t, r.RazorpayPaymentID, "cloned order %s kept a gateway payment id", r.ID)
		require.NotNil(t, r.ClonedFromID, "a clone must record its origin")
		require.NotNil(t, r.TestSessionID, "a clone must be tied to its session")
		require.Equal(t, session.ID.String(), *r.TestSessionID)
		require.NotEqual(t, r.ID, *r.ClonedFromID, "a clone must get a fresh identity")
	}
}

// The single biggest correctness risk in this feature. Cloning 118 orders must
// not fire 118 order-created pushes at a real customer, enqueue 118 NATS
// events, or start 118 Temporal workflows.
func TestCloneEmitsNoSideEffects(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)
	session := &models.ChefTestSession{ID: uuid.New(), ChefID: chefID, SessionNo: 1}

	_, err := CloneChefIntoSession(db, chefID, session, 30)
	require.NoError(t, err)

	var outbox, notifications int64
	require.NoError(t, db.Table("outbox_events").Count(&outbox).Error)
	require.Zero(t, outbox, "the clone must enqueue no events")
	require.NoError(t, db.Table("notifications").Count(&notifications).Error)
	require.Zero(t, notifications, "the clone must create no notifications")
}

// A partial clone would leave a kitchen half-copied and in test mode with no
// way to tell what is missing.
func TestOpenSessionRollsBackWhollyOnCloneFailure(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)
	require.NoError(t, db.Exec(`DROP TABLE orders`).Error) // force a mid-clone failure

	_, err := OpenTestSession(db, chefID, uuid.New(), "will fail", 30)
	require.Error(t, err)

	var cloned int64
	require.NoError(t, db.Table("menu_items").
		Where("mode = ?", models.ChefModeTest).Count(&cloned).Error)
	require.Zero(t, cloned, "a failed clone must leave no rows behind")

	var sessions int64
	require.NoError(t, db.Table("chef_test_sessions").Count(&sessions).Error)
	require.Zero(t, sessions, "a failed flip must leave no session behind")

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.False(t, chef.IsTestMode(), "a failed flip must leave the kitchen live")
}

// snapshotLiveRows captures every live-partition row for a kitchen, ordered
// deterministically, so a round trip can be compared byte for byte.
func snapshotLiveRows(t *testing.T, db *gorm.DB, chefID uuid.UUID) map[string][]map[string]any {
	t.Helper()
	out := map[string][]map[string]any{}
	for _, table := range []string{"orders", "menu_items"} {
		var rows []map[string]any
		require.NoError(t, db.Raw(
			"SELECT * FROM "+table+" WHERE chef_id = ? AND mode = ? ORDER BY id", // #nosec: fixed table list
			chefID.String(), models.ChefModeLive).Scan(&rows).Error)
		out[table] = rows
	}
	return out
}

// The behaviour the whole feature is judged on: flip a live kitchen to test,
// wreck it thoroughly in the sandbox, flip back — and every live row is exactly
// as it was. This works because live rows are never written while in test mode,
// so there is no restore step to get wrong.
func TestLiveDataSurvivesARoundTrip(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)
	adminID := uuid.New()

	before := snapshotLiveRows(t, db, chefID)
	require.NotEmpty(t, before["orders"], "the fixture must have live orders to protect")

	_, err := OpenTestSession(db, chefID, adminID, "reproducing a prod issue", 30)
	require.NoError(t, err)

	// Wreck the sandbox thoroughly.
	require.NoError(t, db.Exec(`UPDATE orders SET status='cancelled' WHERE mode='test'`).Error)
	require.NoError(t, db.Exec(`DELETE FROM menu_items WHERE mode='test'`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO orders (id, chef_id, status, mode, created_at) VALUES (?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "delivered", "test", time.Now()).Error)

	require.NoError(t, CloseTestSession(db, chefID, adminID))

	require.Equal(t, before, snapshotLiveRows(t, db, chefID),
		"live data must be identical after a full test round trip")

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.False(t, chef.IsTestMode(), "closing must return the kitchen to live")
	require.Nil(t, chef.ActiveTestSessionID, "closing must clear the active session")
}

// Purge must remove exactly the session's rows and nothing else — above all,
// not a single live row.
func TestPurgeRemovesOnlyItsOwnSession(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)
	adminID := uuid.New()

	s1, err := OpenTestSession(db, chefID, adminID, "first", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))
	s2, err := OpenTestSession(db, chefID, adminID, "second", 30)
	require.NoError(t, err)
	require.NoError(t, CloseTestSession(db, chefID, adminID))

	liveBefore := snapshotLiveRows(t, db, chefID)

	_, err = PurgeTestSession(db, s1.ID)
	require.NoError(t, err)

	var remaining int64
	require.NoError(t, db.Table("orders").Where("test_session_id = ?", s1.ID).Count(&remaining).Error)
	require.Zero(t, remaining, "session 1 rows must be gone")

	require.NoError(t, db.Table("orders").Where("test_session_id = ?", s2.ID).Count(&remaining).Error)
	require.NotZero(t, remaining, "session 2 must be untouched")

	require.Equal(t, liveBefore, snapshotLiveRows(t, db, chefID),
		"purge must never touch live data")

	var s models.ChefTestSession
	require.NoError(t, db.First(&s, "id = ?", s1.ID).Error)
	require.Equal(t, models.TestSessionPurged, s.Status)
}

// Purging a session the kitchen is still operating in would delete rows out
// from under live traffic.
func TestPurgeRefusesAnOpenSession(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)

	s, err := OpenTestSession(db, chefID, uuid.New(), "open", 30)
	require.NoError(t, err)

	_, err = PurgeTestSession(db, s.ID)
	require.ErrorIs(t, err, ErrSessionOpen)
}
