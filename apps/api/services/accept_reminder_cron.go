package services

// accept_reminder_cron.go — nudge a chef about an order they have not accepted,
// in the last two hours before their kitchen closes on it (#694).
//
// This is the other half of the auto-void: the void (unaccepted_order_cron.go) is
// the stick, this is the tap on the shoulder that should make it rare. From two
// hours before the deadline, every 30 minutes, an unaccepted paid order earns a
// reminder — a chef busy in the kitchen gets a real chance to act before their
// customer is auto-refunded.
//
// It is a sweep, not a per-order scheduler, on purpose: the deadline moves with
// the chef's schedule and there is no durable per-order timer to keep in sync.
// The cadence is held by AcceptRemindersOwed — the schedule is deterministic, so
// "how many nudges should have fired by now" minus "how many did" is what is due,
// and a 5-minute tick can drive a 30-minute cadence without ever double-sending.

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

const acceptReminderInterval = 5 * time.Minute

// StartAcceptReminderCron launches the pre-close nudge sweep.
func StartAcceptReminderCron(ctx context.Context) {
	go func() {
		runAcceptReminderScan(ctx)
		ticker := time.NewTicker(acceptReminderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("accept-reminder: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runAcceptReminderScan(ctx)
			}
		}
	}()
	log.Printf("accept-reminder: cron started (interval=%s)", acceptReminderInterval)
}

func runAcceptReminderScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("accept-reminder: panic recovered: %v", r)
		}
	}()

	// The saga owns the whole unaccepted-order lifecycle when it is on, reminders
	// included — don't double up on the nudge any more than on the refund.
	if sagaActive() {
		return
	}

	sent := scanAcceptReminders(ctx, database.DB, time.Now())
	if sent > 0 {
		log.Printf("accept-reminder: nudged %d unaccepted order(s) nearing close", sent)
	}
}

// scanAcceptReminders is the sweep body (db + now injected for tests). Returns
// how many reminders it dispatched.
func scanAcceptReminders(ctx context.Context, db *gorm.DB, now time.Time) int {
	var pending []models.Order
	if err := db.
		Where("payment_status = ? AND status = ?",
			models.PaymentCompleted, models.OrderStatusPending).
		Order("created_at ASC").
		Limit(unacceptedOrderBatch).
		Find(&pending).Error; err != nil {
		log.Printf("accept-reminder: query failed: %v", err)
		return 0
	}

	sent := 0
	for i := range pending {
		select {
		case <-ctx.Done():
			return sent
		default:
		}
		order := pending[i]

		deadline := ResolveAcceptDeadline(db, &order)
		// Past the deadline the void sweep owns it — a nudge now would race the
		// cancellation. Before the 2h window there is nothing to nudge about.
		if !now.Before(deadline.At) {
			continue
		}
		owed := AcceptRemindersOwed(deadline.At, order.CreatedAt, now, order.AcceptReminderCount)
		if owed <= 0 {
			continue
		}

		// Fire ONE per tick even when several are owed (a long outage). Bursting a
		// chef with four "please accept" pushes at once is worse than catching up
		// one per 5-minute tick, and the customer's real protection is the void, not
		// the nudge count.
		newCount := order.AcceptReminderCount + 1

		// Stamp FIRST, guarded on the count we read, and only notify if the stamp
		// won. Two sweep instances (or a Temporal Schedule plus a stray ticker)
		// would otherwise both read the same count and both push. The guarded
		// UPDATE makes exactly one of them win; the loser sends nothing.
		res := db.Model(&models.Order{}).
			Where("id = ? AND accept_reminder_count = ?", order.ID, order.AcceptReminderCount).
			Updates(map[string]any{
				"accept_reminder_count":   newCount,
				"last_accept_reminder_at": now,
			})
		if res.Error != nil {
			log.Printf("accept-reminder: claim failed for %s: %v", order.ID, res.Error)
			continue
		}
		if res.RowsAffected == 0 {
			continue // another sweep beat us to this nudge
		}

		if err := dispatchAcceptReminder(db, &order, deadline, newCount); err != nil {
			// The stamp is already committed, so this nudge is spent — do not roll it
			// back and re-fire, which would risk a double push. A missed single
			// reminder is a far smaller harm than a duplicate, and the next one is 30
			// minutes out. Surface it and move on.
			log.Printf("accept-reminder: dispatch failed for %s (skipping this nudge): %v", order.ID, err)
			CaptureBackgroundError(err)
			continue
		}
		sent++
	}
	return sent
}

// dispatchAcceptReminder stages the chef nudge. Staged in the outbox (not pushed
// inline) so it rides the same durable, retried path as every other chef
// notification and cannot be lost if this process dies mid-sweep.
func dispatchAcceptReminder(db *gorm.DB, order *models.Order, deadline AcceptDeadline, n int) error {
	minsLeft := int(deadline.At.Sub(time.Now()).Minutes())
	if minsLeft < 0 {
		minsLeft = 0
	}
	return EnqueueEvent(db, SubjectOrderAcceptReminder, "order.accept_reminder", order.CustomerID, map[string]any{
		"order_id":     order.ID.String(),
		"order_number": order.OrderNumber,
		"chef_id":      order.ChefID.String(),
		"slot":         deadline.Slot,
		"deadline":     deadline.At,
		"reminder_n":   n,
		"minutes_left": minsLeft,
		// So the consumer can escalate the tone as the deadline nears without
		// re-deriving it.
		"final_call": minsLeft <= int(acceptReminderEvery.Minutes()),
	})
}
