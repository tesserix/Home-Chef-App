package services

// meal_plan_reminder_cron_test.go — the chef cook-reminder sweep (night-before +
// morning-of, IST). Pure DB logic on the shared in-memory sqlite harness
// (setupHoldDB). The notifier + dedup seams are swapped so tests never touch the
// outbox or Redis; the sweep core takes an injected `now`.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// reminderIST is the fixed +5:30 zone used to construct deterministic test clocks.
var reminderIST = time.FixedZone("IST", 5*3600+30*60)

// capturedReminder records one chefCookReminderNotifier call.
type capturedReminder struct {
	chefUserID uuid.UUID
	targetDate string
	window     string
	lunch      int
	dinner     int
}

// installReminderSeams swaps the notifier (into a recorder) and the dedup gate
// (to the given "already sent?" answer), restoring both on cleanup. Returns a
// pointer to the growing slice of captured calls.
func installReminderSeams(t *testing.T, alreadySent bool) *[]capturedReminder {
	t.Helper()
	calls := &[]capturedReminder{}
	prevNotifier := chefCookReminderNotifier
	prevDedup := reminderAlreadySent
	chefCookReminderNotifier = func(_ *gorm.DB, chefUserID uuid.UUID, targetDate, window string, lunch, dinner int) error {
		*calls = append(*calls, capturedReminder{chefUserID, targetDate, window, lunch, dinner})
		return nil
	}
	reminderAlreadySent = func(uuid.UUID, string, string) bool { return alreadySent }
	t.Cleanup(func() {
		chefCookReminderNotifier = prevNotifier
		reminderAlreadySent = prevDedup
	})
	return calls
}

// setupReminderDB extends setupHoldDB with the columns/tables the reminder query
// joins on but the base harness lacks: a `slot` column on meal_plan_days and a
// chef_profiles table.
func setupReminderDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := setupHoldDB(t)
	require.NoError(t, db.Exec(`ALTER TABLE meal_plan_days ADD COLUMN slot TEXT DEFAULT ''`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE chef_profiles (id TEXT PRIMARY KEY, user_id TEXT)`).Error)
	return db
}

// seedReminderChef inserts a chef_profile + a meal_plan owned by it, returning the
// chef's users.id and the plan id.
func seedReminderChef(t *testing.T, db *gorm.DB) (chefUserID, planID uuid.UUID) {
	t.Helper()
	chefUserID = uuid.New()
	chefProfileID := uuid.New()
	planID = uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO chef_profiles (id, user_id) VALUES (?,?)`,
		chefProfileID.String(), chefUserID.String()).Error)
	require.NoError(t, db.Exec(`INSERT INTO meal_plans (id, customer_id, chef_id, status) VALUES (?,?,?,?)`,
		planID.String(), uuid.NewString(), chefProfileID.String(), "active").Error)
	return chefUserID, planID
}

// seedDay inserts one meal_plan_day. `date` is stored in UTC so the sqlite string
// comparison lines up with the sweep's UTC window bounds.
func seedDay(t *testing.T, db *gorm.DB, planID uuid.UUID, slot, status string, date time.Time) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, slot, date) VALUES (?,?,?,?,?)`,
		uuid.NewString(), planID.String(), status, slot, date.UTC()).Error)
}

// istMidnight builds the IST-midnight instant for a Y/M/D — the "date" a
// meal-plan day carries.
func istMidnight(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, reminderIST)
}

// findCall returns the (single) captured call for a chef, asserting exactly one.
func findCall(t *testing.T, calls []capturedReminder, chef uuid.UUID) capturedReminder {
	t.Helper()
	var found []capturedReminder
	for _, c := range calls {
		if c.chefUserID == chef {
			found = append(found, c)
		}
	}
	require.Len(t, found, 1, "expected exactly one reminder for chef %s", chef)
	return found[0]
}

// (a) 20:00 IST → each chef notified once for window="tomorrow" with correct
// lunch/dinner counts targeting tomorrow's date; (d) delivered/cancelled days are
// not counted.
func TestChefCookReminder_NightBefore_20IST(t *testing.T) {
	db := setupReminderDB(t)
	calls := installReminderSeams(t, false)

	now := time.Date(2026, 7, 25, 20, 30, 0, 0, reminderIST) // 20:xx IST → tomorrow window
	tomorrow := istMidnight(2026, 7, 26)

	chefA, planA := seedReminderChef(t, db)
	// chefA: 2 lunch + 1 dinner confirmed for tomorrow ...
	seedDay(t, db, planA, "lunch", "confirmed", tomorrow)
	seedDay(t, db, planA, "lunch", "confirmed", tomorrow)
	seedDay(t, db, planA, "dinner", "confirmed", tomorrow)
	// ... plus a delivered + a cancelled day that MUST NOT be counted (d).
	seedDay(t, db, planA, "lunch", "delivered", tomorrow)
	seedDay(t, db, planA, "dinner", "cancelled", tomorrow)

	chefB, planB := seedReminderChef(t, db)
	seedDay(t, db, planB, "lunch", "confirmed", tomorrow) // 1 lunch, 0 dinner

	n := scanChefCookReminders(db, now)
	require.Equal(t, 2, n, "both chefs notified")
	require.Len(t, *calls, 2)

	a := findCall(t, *calls, chefA)
	require.Equal(t, "tomorrow", a.window)
	require.Equal(t, "2026-07-26", a.targetDate)
	require.Equal(t, 2, a.lunch, "delivered/cancelled days must be excluded")
	require.Equal(t, 1, a.dinner)

	b := findCall(t, *calls, chefB)
	require.Equal(t, "tomorrow", b.window)
	require.Equal(t, "2026-07-26", b.targetDate)
	require.Equal(t, 1, b.lunch)
	require.Equal(t, 0, b.dinner)
}

// (b) 07:00 IST → window="today" targeting today's date.
func TestChefCookReminder_MorningOf_07IST(t *testing.T) {
	db := setupReminderDB(t)
	calls := installReminderSeams(t, false)

	now := time.Date(2026, 7, 25, 7, 15, 0, 0, reminderIST) // 07:xx IST → today window
	today := istMidnight(2026, 7, 25)

	chefA, planA := seedReminderChef(t, db)
	seedDay(t, db, planA, "lunch", "confirmed", today)
	seedDay(t, db, planA, "dinner", "confirmed", today)
	// A confirmed day for TOMORROW must not leak into the today window.
	seedDay(t, db, planA, "lunch", "confirmed", istMidnight(2026, 7, 26))

	n := scanChefCookReminders(db, now)
	require.Equal(t, 1, n)

	a := findCall(t, *calls, chefA)
	require.Equal(t, "today", a.window)
	require.Equal(t, "2026-07-25", a.targetDate)
	require.Equal(t, 1, a.lunch)
	require.Equal(t, 1, a.dinner)
}

// (c) 13:00 IST (no window) → zero notifications.
func TestChefCookReminder_NoWindow_13IST(t *testing.T) {
	db := setupReminderDB(t)
	calls := installReminderSeams(t, false)

	now := time.Date(2026, 7, 25, 13, 0, 0, 0, reminderIST)
	_, planA := seedReminderChef(t, db)
	seedDay(t, db, planA, "lunch", "confirmed", istMidnight(2026, 7, 25))
	seedDay(t, db, planA, "dinner", "confirmed", istMidnight(2026, 7, 26))

	n := scanChefCookReminders(db, now)
	require.Equal(t, 0, n, "no reminder window is active at 13:00 IST")
	require.Empty(t, *calls)
}

// (e) when reminderAlreadySent returns true → chef skipped (dedup on a re-tick).
func TestChefCookReminder_DedupSkips(t *testing.T) {
	db := setupReminderDB(t)
	calls := installReminderSeams(t, true) // pretend every reminder already sent

	now := time.Date(2026, 7, 25, 20, 30, 0, 0, reminderIST)
	_, planA := seedReminderChef(t, db)
	seedDay(t, db, planA, "lunch", "confirmed", istMidnight(2026, 7, 26))
	seedDay(t, db, planA, "dinner", "confirmed", istMidnight(2026, 7, 26))

	n := scanChefCookReminders(db, now)
	require.Equal(t, 0, n, "an already-sent reminder must be skipped")
	require.Empty(t, *calls)
}
