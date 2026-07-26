package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// countScoped is what a vendor-facing screen sees for this kitchen right now:
// rows in whichever world the kitchen currently inhabits.
func countScoped(t *testing.T, db *gorm.DB, table string, chefID uuid.UUID) int64 {
	t.Helper()
	var n int64
	require.NoError(t, db.Table(table).
		Where("chef_id = ?", chefID).
		Scopes(ChefOwnModeScope(chefID)).
		Count(&n).Error)
	return n
}

// TestSupportEngineerJourney walks the exact flow an admin support engineer
// follows to debug a production issue on a REAL kitchen, and asserts the
// property that makes it safe to do at all: no live data is lost or altered.
//
//  1. "Amma ka Kitchen" is live, with real orders and a real menu.
//  2. A customer reports a bug. The engineer flips the kitchen to Test.
//  3. The kitchen's setup and recent orders are cloned into a sandbox session.
//  4. The chef and engineer work in the sandbox: place fake orders, change the
//     menu, break things. None of it touches real data.
//  5. The fix ships. The engineer flips back to Live.
//  6. The real kitchen resumes EXACTLY as it was — same orders, same menu,
//     same totals — and the sandbox evidence is retained for reference.
func TestSupportEngineerJourney(t *testing.T) {
	db := setupSessionDB(t)
	engineer := uuid.New()

	// ── 1. A real kitchen, mid-life ──────────────────────────────────────────
	chefID := seedChefWithData(t, db) // 3 menu items; 1 recent order, 1 old order

	liveOrdersBefore := countScoped(t, db, "orders", chefID)
	liveMenuBefore := countScoped(t, db, "menu_items", chefID)
	require.EqualValues(t, 2, liveOrdersBefore, "fixture must have real orders to protect")
	require.EqualValues(t, 3, liveMenuBefore, "fixture must have a real menu to protect")
	snapshotBefore := snapshotLiveRows(t, db, chefID)

	// ── 2. Flip to Test ──────────────────────────────────────────────────────
	session, err := OpenTestSession(db, chefID, engineer,
		"customer reports double-charge on HC-4821", 30)
	require.NoError(t, err)
	require.Equal(t, 1, session.SessionNo)

	var chef models.ChefProfile
	require.NoError(t, db.First(&chef, "id = ?", chefID).Error)
	require.True(t, chef.IsTestMode(), "the kitchen is now a sandbox")
	require.False(t, chef.IsBornTest(),
		"an established kitchen must stay established, so customers see it as CLOSED rather than gone")

	// ── 3. The clone gave the sandbox a working kitchen ──────────────────────
	// The vendor app now shows sandbox rows only — a clean slate, but not an
	// empty one: the menu came across so the kitchen behaves like the real one.
	require.EqualValues(t, 3, countScoped(t, db, "menu_items", chefID),
		"the sandbox must start with the kitchen's real menu, cloned")
	require.EqualValues(t, 1, countScoped(t, db, "orders", chefID),
		"only orders inside the 30-day window are cloned; the 200-day-old one is not")

	// Customers, meanwhile, must not be offered any of it.
	require.Equal(t, VisibilityClosed, ChefVisibility(&chef, "regular-customer@example.com"),
		"a regular sees the kitchen listed but CLOSED, not vanished")
	require.Equal(t, VisibilityFull, ChefVisibility(&chef, "samyak.rout@gmail.com"),
		"the engineer on the allowlist sees it fully")

	// ── 4. Work in the sandbox: place fake orders, wreck the menu ────────────
	for i := 0; i < 4; i++ {
		require.NoError(t, db.Exec(
			`INSERT INTO orders (id, chef_id, order_number, status, total, mode, test_session_id, created_at)
			 VALUES (?,?,?,?,?,?,?,?)`,
			uuid.New().String(), chefID.String(), fmt.Sprintf("TEST-REPRO-%d", i), "delivered", 999.0,
			models.ChefModeTest, session.ID.String(), time.Now()).Error)
	}
	require.NoError(t, db.Exec(`UPDATE orders SET status='cancelled' WHERE mode='test'`).Error)
	require.NoError(t, db.Exec(`DELETE FROM menu_items WHERE mode='test'`).Error)

	require.EqualValues(t, 0, countScoped(t, db, "menu_items", chefID),
		"the engineer deleted the sandbox menu entirely")
	require.EqualValues(t, 5, countScoped(t, db, "orders", chefID),
		"1 cloned + 4 fake orders, all sandbox")

	// The real kitchen is untouched THROUGHOUT, not merely afterwards.
	require.Equal(t, snapshotBefore, snapshotLiveRows(t, db, chefID),
		"live data must be untouched while the sandbox is being wrecked")

	// ── 5. The fix ships. Flip back to Live ──────────────────────────────────
	require.NoError(t, CloseTestSession(db, chefID, engineer))

	// Read into a FRESH struct: GORM's First into an already-populated one does
	// not reset a field whose column is now NULL, which would quietly mask a
	// session that was never cleared.
	var afterClose models.ChefProfile
	require.NoError(t, db.First(&afterClose, "id = ?", chefID).Error)
	require.False(t, afterClose.IsTestMode(), "the kitchen is real again")
	require.Nil(t, afterClose.ActiveTestSessionID, "no session is open")
	chef = afterClose

	// ── 6. Nothing was lost ──────────────────────────────────────────────────
	require.Equal(t, snapshotBefore, snapshotLiveRows(t, db, chefID),
		"every live row must be identical to before the investigation")
	require.EqualValues(t, liveOrdersBefore, countScoped(t, db, "orders", chefID),
		"the chef's real order queue is back, whole")
	require.EqualValues(t, liveMenuBefore, countScoped(t, db, "menu_items", chefID),
		"the chef's real menu is back, whole — the sandbox deletion did not touch it")

	// The sandbox evidence is kept, so the engineer can look back at what happened.
	var closed models.ChefTestSession
	require.NoError(t, db.First(&closed, "id = ?", session.ID).Error)
	require.Equal(t, models.TestSessionClosed, closed.Status)
	require.NotNil(t, closed.ClosedAt)

	var sandboxRows int64
	require.NoError(t, db.Table("orders").
		Where("test_session_id = ?", session.ID).Count(&sandboxRows).Error)
	require.EqualValues(t, 5, sandboxRows, "sandbox rows are retained, not deleted, on close")

	// And customers can order again.
	require.Equal(t, VisibilityFull, ChefVisibility(&chef, "regular-customer@example.com"))
}

// A second investigation months later must not disturb the first one's record,
// and must clone from the kitchen's CURRENT live state rather than the stale
// contents of the previous sandbox.
func TestSecondInvestigationIsIndependent(t *testing.T) {
	db := setupSessionDB(t)
	engineer := uuid.New()
	chefID := seedChefWithData(t, db)

	s1, err := OpenTestSession(db, chefID, engineer, "first bug", 30)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`DELETE FROM menu_items WHERE mode='test'`).Error)
	require.NoError(t, CloseTestSession(db, chefID, engineer))

	// The kitchen adds a dish in the real world between investigations.
	require.NoError(t, db.Exec(
		`INSERT INTO menu_items (id, chef_id, name, price, mode) VALUES (?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "New Dish", 200.0, models.ChefModeLive).Error)

	s2, err := OpenTestSession(db, chefID, engineer, "second bug", 30)
	require.NoError(t, err)
	require.Equal(t, 2, s2.SessionNo)
	require.NotEqual(t, s1.ID, s2.ID)

	// Session 2 clones the kitchen as it is NOW — four dishes, not the three it
	// had last time and not the zero session 1 was left with.
	require.EqualValues(t, 4, countScoped(t, db, "menu_items", chefID),
		"a new session clones current live state, not the previous sandbox")

	var first models.ChefTestSession
	require.NoError(t, db.First(&first, "id = ?", s1.ID).Error)
	require.Equal(t, models.TestSessionClosed, first.Status,
		"the first investigation's record is untouched by the second")
}

// The clean-slate rule, stated on its own: the same query returns the kitchen's
// real rows or its sandbox rows purely according to which world it is in.
func TestChefOwnModeScopeSwitchesWorlds(t *testing.T) {
	db := setupSessionDB(t)
	chefID := seedChefWithData(t, db)

	require.EqualValues(t, 3, countScoped(t, db, "menu_items", chefID), "live: the real menu")

	_, err := OpenTestSession(db, chefID, uuid.New(), "switching", 30)
	require.NoError(t, err)
	require.NoError(t, db.Exec(`DELETE FROM menu_items WHERE mode = ? AND chef_id = ?`,
		models.ChefModeTest, chefID.String()).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO menu_items (id, chef_id, name, price, mode) VALUES (?,?,?,?,?)`,
		uuid.New().String(), chefID.String(), "Sandbox Dish", 1.0, models.ChefModeTest).Error)

	require.EqualValues(t, 1, countScoped(t, db, "menu_items", chefID),
		"test: only the sandbox menu, never the real one")

	require.NoError(t, CloseTestSession(db, chefID, uuid.New()))
	require.EqualValues(t, 3, countScoped(t, db, "menu_items", chefID),
		"live again: the real menu, whole")
}
