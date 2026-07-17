package services

// order_accept_deadline_test.go — #694.
//
// The deadline is the whole safety of the auto-void: get it wrong and we refund
// orders a chef was about to cook, or never refund orders nobody will. These pin
// the rule against the chef's OWN configuration, and above all that an ADVANCE
// order keeps its full lifetime — the bug that a "30 minutes after payment"
// timeout would have caused, silently, to every future booking.

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func seedCapacity(t *testing.T, db *gorm.DB, chefID uuid.UUID, lunchEnd, dinnerEnd string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO chef_capacity_settings
		(id, chef_id, cutoff_enabled, lunch_cutoff, dinner_cutoff, slots_enabled,
		 lunch_slot_start, lunch_slot_end, dinner_slot_start, dinner_slot_end)
		VALUES (?,?,?,?,?,?,?,?,?,?)`,
		uuid.NewString(), chefID.String(), true, "11:00", "18:00", true,
		"12:00", lunchEnd, "19:00", dinnerEnd).Error)
}

func seedSchedule(t *testing.T, db *gorm.DB, chefID uuid.UUID, dow int, closeTime string) {
	t.Helper()
	require.NoError(t, db.Exec(`INSERT INTO chef_schedules (id, chef_id, day_of_week, open_time, close_time, is_closed)
		VALUES (?,?,?,?,?,0)`, uuid.NewString(), chefID.String(), dow, "09:00", closeTime).Error)
}

func setupDeadlineDB(t *testing.T) (*gorm.DB, uuid.UUID) {
	t.Helper()
	db := setupCancelRefundDB(t)
	for _, s := range []string{
		`CREATE TABLE chef_capacity_settings (id text PRIMARY KEY, chef_id text, cutoff_enabled boolean,
			lunch_cutoff text, dinner_cutoff text, slots_enabled boolean, lunch_slot_start text,
			lunch_slot_end text, dinner_slot_start text, dinner_slot_end text,
			lunch_slot_capacity int, dinner_slot_capacity int, auto_sold_out boolean,
			created_at datetime, updated_at datetime)`,
		`CREATE TABLE chef_schedules (id text PRIMARY KEY, chef_id text, day_of_week int,
			open_time text, close_time text, is_closed boolean, created_at datetime, updated_at datetime)`,
		// ReleaseSlot decrements this when a voided order's slot booking is freed.
		`CREATE TABLE chef_slot_daily_bookings (id text PRIMARY KEY, chef_id text, slot text,
			booking_date datetime, booked_qty int NOT NULL DEFAULT 0,
			created_at datetime, updated_at datetime)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db, uuid.New()
}

// ist builds an IST wall-clock instant.
func ist(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, capacityIST)
}

func orderFor(chefID uuid.UUID, slot string, scheduledFor time.Time, created time.Time) *models.Order {
	return &models.Order{
		ID: uuid.New(), ChefID: chefID, DeliverySlot: slot,
		ScheduledFor: &scheduledFor, CreatedAt: created,
	}
}

// The chef said their lunch runs to 14:00. That is the deadline.
func TestDeadline_UsesTheChefsStatedSlotEnd(t *testing.T) {
	db, chefID := setupDeadlineDB(t)
	seedCapacity(t, db, chefID, "14:00", "22:00")

	day := ist(2026, 7, 20, 12, 0)
	d := ResolveAcceptDeadline(db, orderFor(chefID, "lunch", day, ist(2026, 7, 20, 9, 0)))

	require.Equal(t, "slot_end", d.Source)
	require.Equal(t, "lunch", d.Slot)
	require.Equal(t, ist(2026, 7, 20, 14, 0), d.At)
}

func TestDeadline_DinnerUsesTheDinnerEnd(t *testing.T) {
	db, chefID := setupDeadlineDB(t)
	seedCapacity(t, db, chefID, "14:00", "22:00")

	day := ist(2026, 7, 20, 20, 0)
	d := ResolveAcceptDeadline(db, orderFor(chefID, "dinner", day, ist(2026, 7, 20, 9, 0)))

	require.Equal(t, ist(2026, 7, 20, 22, 0), d.At)
	require.Equal(t, "dinner", d.Slot)
}

// THE bug the old model caused. An order placed today for TOMORROW's dinner must
// keep its full life — a stopwatch from checkout would have voided it in 30
// minutes, before the chef ever plausibly looked.
func TestDeadline_AdvanceOrder_KeepsItsFullLifetime(t *testing.T) {
	db, chefID := setupDeadlineDB(t)
	seedCapacity(t, db, chefID, "14:00", "22:00")

	placed := ist(2026, 7, 20, 10, 0)
	tomorrowDinner := ist(2026, 7, 21, 20, 0)
	d := ResolveAcceptDeadline(db, orderFor(chefID, "dinner", tomorrowDinner, placed))

	require.Equal(t, ist(2026, 7, 21, 22, 0), d.At,
		"the deadline is TOMORROW's dinner close — not 30 minutes after checkout")
	require.True(t, d.At.Sub(placed) > 24*time.Hour,
		"an advance order legitimately has more than a day of life")
}

// No slot windows configured → fall back to when the chef says the kitchen shuts.
func TestDeadline_FallsBackToTheKitchenCloseForThatWeekday(t *testing.T) {
	db, chefID := setupDeadlineDB(t)
	// 2026-07-20 is a Monday (weekday 1).
	seedSchedule(t, db, chefID, 1, "21:30")

	day := ist(2026, 7, 20, 19, 0)
	d := ResolveAcceptDeadline(db, orderFor(chefID, "dinner", day, ist(2026, 7, 20, 9, 0)))

	require.Equal(t, "kitchen_close", d.Source)
	require.Equal(t, ist(2026, 7, 20, 21, 30), d.At)
}

// Nothing configured at all → a deadline still exists. "No deadline" means the
// customer's money is captured forever, which is the hole this closes.
func TestDeadline_UnconfiguredChefStillGetsADeadline(t *testing.T) {
	db, chefID := setupDeadlineDB(t)

	day := ist(2026, 7, 20, 12, 0)
	d := ResolveAcceptDeadline(db, orderFor(chefID, "lunch", day, ist(2026, 7, 20, 9, 0)))

	require.Equal(t, "platform_default", d.Source)
	require.Equal(t, ist(2026, 7, 20, 15, 0), d.At)
}

// An ASAP order has no slot; infer it from the hour it is for.
func TestDeadline_AsapOrder_InfersTheSlot(t *testing.T) {
	db, chefID := setupDeadlineDB(t)
	seedCapacity(t, db, chefID, "14:00", "22:00")

	noon := ist(2026, 7, 20, 12, 30)
	d := ResolveAcceptDeadline(db, &models.Order{ID: uuid.New(), ChefID: chefID, CreatedAt: noon})
	require.Equal(t, "lunch", d.Slot)
	require.Equal(t, ist(2026, 7, 20, 14, 0), d.At)

	evening := ist(2026, 7, 20, 19, 30)
	d = ResolveAcceptDeadline(db, &models.Order{ID: uuid.New(), ChefID: chefID, CreatedAt: evening})
	require.Equal(t, "dinner", d.Slot)
	require.Equal(t, ist(2026, 7, 20, 22, 0), d.At)
}

// A malformed configured time must not produce a garbage deadline — fall through
// to the next source rather than voiding at 00:00.
func TestDeadline_MalformedTimeFallsThrough(t *testing.T) {
	db, chefID := setupDeadlineDB(t)
	seedCapacity(t, db, chefID, "not-a-time", "99:99")
	seedSchedule(t, db, chefID, 1, "20:00")

	day := ist(2026, 7, 20, 12, 0)
	d := ResolveAcceptDeadline(db, orderFor(chefID, "lunch", day, ist(2026, 7, 20, 9, 0)))

	require.Equal(t, "kitchen_close", d.Source, "garbage in the slot end must not become the deadline")
	require.Equal(t, ist(2026, 7, 20, 20, 0), d.At)
}

// ── Reminder schedule ───────────────────────────────────────────────────────

func TestAcceptReminders_EveryThirtyMinutesForTheFinalTwoHours(t *testing.T) {
	deadline := ist(2026, 7, 20, 14, 0)
	now := ist(2026, 7, 20, 9, 0)

	require.Equal(t, []time.Time{
		ist(2026, 7, 20, 12, 0),
		ist(2026, 7, 20, 12, 30),
		ist(2026, 7, 20, 13, 0),
		ist(2026, 7, 20, 13, 30),
	}, AcceptReminderTimes(deadline, now),
		"from 2h out, every 30 minutes — and never AT the deadline, where the void lands")
}

// An order placed inside the reminder window gets the nudges still ahead of it,
// not an instant burst of the ones it missed.
func TestAcceptReminders_SkipsTimesAlreadyPast(t *testing.T) {
	deadline := ist(2026, 7, 20, 14, 0)
	now := ist(2026, 7, 20, 13, 10)

	require.Equal(t, []time.Time{ist(2026, 7, 20, 13, 30)}, AcceptReminderTimes(deadline, now))
}

func TestAcceptReminders_NoneWhenTheDeadlineHasPassed(t *testing.T) {
	deadline := ist(2026, 7, 20, 14, 0)
	require.Empty(t, AcceptReminderTimes(deadline, ist(2026, 7, 20, 15, 0)),
		"nothing to remind about — the order is being voided")
}
