package services

// accept_reminder_cron_test.go — #694. The pre-close nudge sweep.
//
// The logic (how many nudges are owed when) is pinned in
// order_accept_deadline_test.go. These pin the SWEEP around it: it stamps and
// stages exactly one nudge per tick, never double-fires within a slot, and never
// nudges outside the window.

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func acceptReminderRow(t *testing.T, db *gorm.DB, id uuid.UUID) (count int, staged int64) {
	t.Helper()
	require.NoError(t, db.Raw(`SELECT COALESCE(accept_reminder_count,0) FROM orders WHERE id = ?`, id.String()).Scan(&count).Error)
	require.NoError(t, db.Raw(`SELECT count(*) FROM outbox_events WHERE subject = ?`, SubjectOrderAcceptReminder).Scan(&staged).Error)
	return count, staged
}

// Inside the final two hours, an unaccepted order earns a nudge — and the count
// is stamped so the next tick doesn't re-send.
func TestAcceptReminderSweep_NudgesInsideTheWindow(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	// lunch closes 14:00; placed 09:00; order for today's lunch.
	o := seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))

	// 12:05 — first slot (12:00) has passed.
	require.Equal(t, 1, scanAcceptReminders(context.Background(), db, ist(2026, 7, 20, 12, 5)))

	count, staged := acceptReminderRow(t, db, o.ID)
	require.Equal(t, 1, count, "the nudge count is stamped")
	require.Equal(t, int64(1), staged, "exactly one reminder event staged")
}

// The anti-spam guarantee: a second tick inside the same 30-minute slot sends
// nothing.
func TestAcceptReminderSweep_DoesNotDoubleFireWithinASlot(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	o := seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))

	require.Equal(t, 1, scanAcceptReminders(context.Background(), db, ist(2026, 7, 20, 12, 5)))
	// 12:20 — still the 12:00 slot; the 12:30 one hasn't arrived.
	require.Equal(t, 0, scanAcceptReminders(context.Background(), db, ist(2026, 7, 20, 12, 20)))
	// 12:35 — the 12:30 slot has now passed.
	require.Equal(t, 1, scanAcceptReminders(context.Background(), db, ist(2026, 7, 20, 12, 35)))

	count, staged := acceptReminderRow(t, db, o.ID)
	require.Equal(t, 2, count)
	require.Equal(t, int64(2), staged, "two ticks that crossed a slot boundary; two nudges — not three")
}

// Before the window, nothing fires.
func TestAcceptReminderSweep_SilentBeforeTheWindow(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))

	// 11:30 — the 12:00 window hasn't opened.
	require.Equal(t, 0, scanAcceptReminders(context.Background(), db, ist(2026, 7, 20, 11, 30)))
}

// Past the deadline the void sweep owns it — reminding now would race the
// cancellation.
func TestAcceptReminderSweep_StopsAtTheDeadline(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))

	require.Equal(t, 0, scanAcceptReminders(context.Background(), db, ist(2026, 7, 20, 14, 5)),
		"at/after the deadline the void takes over; no nudge should race it")
}

// An accepted order is not nudged.
func TestAcceptReminderSweep_IgnoresAcceptedOrders(t *testing.T) {
	db, chefID := setupUnacceptedDB(t)
	o := seedPaidPendingOrder(t, db, chefID, "lunch", ist(2026, 7, 20, 12, 0), ist(2026, 7, 20, 9, 0))
	require.NoError(t, db.Exec(`UPDATE orders SET status = ? WHERE id = ?`,
		string(models.OrderStatusAccepted), o.ID.String()).Error)

	require.Equal(t, 0, scanAcceptReminders(context.Background(), db, ist(2026, 7, 20, 13, 30)))
}
