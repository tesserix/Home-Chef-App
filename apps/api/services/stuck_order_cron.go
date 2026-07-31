package services

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// stuck_order_cron.go — orders the chef ACCEPTED and then never finished.
//
// The unaccepted-order sweep covers an order nobody took. This covers the other hole: one
// somebody took, charged for, and abandoned mid-fulfilment. Nothing moved it, so it sat in
// `preparing` indefinitely — the customer out of pocket with no way to close it, the chef's
// list clogged, and neither side prompted to act.
//
// Nudge both sides on a cadence, then refund the customer in full at the deadline. Full,
// because nothing was delivered: there is no partial service to price, so there is nothing
// for the chef to decide. That is why this does NOT route through the chef the way a
// meal-plan cancellation does.
const stuckOrderSweepInterval = 1 * time.Hour

// stuckOrderBatch bounds one pass so a backlog cannot hold a transaction open for minutes.
const stuckOrderBatch = 200

// inFlightOrderStatuses are the states an ACCEPTED order passes through before `delivered`.
// A stuck order is one that stopped somewhere in here.
func inFlightOrderStatuses() []models.OrderStatus {
	return []models.OrderStatus{
		models.OrderStatusAccepted, models.OrderStatusPreparing, models.OrderStatusReady,
		models.OrderStatusPickedUp, models.OrderStatusDelivering,
	}
}

// StartStuckOrderCron is the legacy in-process fallback (used when Temporal is off).
func StartStuckOrderCron(ctx context.Context) {
	go func() {
		runStuckOrderSweep(ctx)
		ticker := time.NewTicker(stuckOrderSweepInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("stuck-order: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runStuckOrderSweep(ctx)
			}
		}
	}()
	log.Println("stuck-order: cron started (interval=1h)")
}

func runStuckOrderSweep(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("stuck-order: panic recovered: %v", r)
		}
	}()
	if database.DB == nil {
		return
	}
	nudged, refunded := sweepStuckOrders(ctx, database.DB, time.Now())
	if nudged > 0 || refunded > 0 {
		log.Printf("stuck-order: nudged %d, refunded %d abandoned order(s)", nudged, refunded)
	}
}

// stuckOrderPolicy reads the tunables, falling back to the defaults. refundDays <= 0 means
// the auto-refund is disabled and only the nudges run.
func stuckOrderPolicy() (reminderDays, refundDays, maxReminders int) {
	p := GetPlatformPolicy()
	d := DefaultPlatformPolicy()
	reminderDays, refundDays, maxReminders = p.StuckOrderReminderDays, p.StuckOrderRefundDays, p.StuckOrderMaxReminders
	if reminderDays <= 0 {
		reminderDays = d.StuckOrderReminderDays
	}
	if refundDays == 0 {
		refundDays = d.StuckOrderRefundDays
	}
	if maxReminders <= 0 {
		maxReminders = d.StuckOrderMaxReminders
	}
	return
}

func sweepStuckOrders(ctx context.Context, db *gorm.DB, now time.Time) (nudged, refunded int) {
	reminderDays, refundDays, maxReminders := stuckOrderPolicy()
	staleBefore := now.AddDate(0, 0, -reminderDays)

	var stuck []models.Order
	if err := db.
		Preload("Items").
		Where("payment_status = ? AND status IN ? AND created_at <= ?",
			models.PaymentCompleted, inFlightOrderStatuses(), staleBefore).
		Order("created_at ASC").
		Limit(stuckOrderBatch).
		Find(&stuck).Error; err != nil {
		log.Printf("stuck-order: query failed: %v", err)
		return 0, 0
	}

	for i := range stuck {
		select {
		case <-ctx.Done():
			return nudged, refunded
		default:
		}
		order := stuck[i]

		// A typed escrow order (meal-plan day, group share) belongs end-to-end to its own
		// flow: different refund keyspace, and a held chef payout this generic path cannot
		// reverse. Checked here rather than left to the refund helper, which returns nil
		// for typed orders — indistinguishable from "refunded fine", which would let us
		// cancel an order we never refunded.
		switch kind, kErr := TypedRefundOrderKind(db, order.ID); {
		case kErr != nil:
			log.Printf("stuck-order: type check failed for %s (skipping): %v", order.ID, kErr)
			continue
		case kind != "":
			continue
		}

		if refundDays > 0 && !order.CreatedAt.After(now.AddDate(0, 0, -refundDays)) {
			if refundStuckOrder(db, &order, now) {
				refunded++
			}
			continue
		}
		if nudgeStuckOrder(db, &order, now, reminderDays, maxReminders) {
			nudged++
		}
	}
	return nudged, refunded
}

// refundStuckOrder makes the customer whole and closes the order. Refund FIRST: a cancelled
// order the customer was never refunded for is the worst state, and an invisible one.
func refundStuckOrder(db *gorm.DB, order *models.Order, now time.Time) bool {
	const reason = "this order was never completed, so it has been cancelled and refunded"

	if err := RefundOrderForCancellation(order, "system", reason); err != nil {
		log.Printf("stuck-order: refund failed for %s (will retry): %v", order.ID, err)
		CaptureBackgroundError(err)
		return false
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// Guarded on the status we saw: a chef or driver finishing the order in the gap
		// must not be clobbered back to cancelled. RowsAffected==0 means they beat us —
		// the refund stands and the reconcile surfaces the mismatch for a human.
		res := tx.Model(&models.Order{}).
			Where("id = ? AND status = ?", order.ID, order.Status).
			Updates(map[string]any{
				"status":        models.OrderStatusCancelled,
				"cancel_reason": reason,
				"cancelled_at":  now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			log.Printf("stuck-order: %s moved during the refund; refund stands", order.ID)
			return nil
		}
		return EnqueueEvent(tx, SubjectOrderVoided, "order.stale_refunded", order.CustomerID, map[string]any{
			"order_id": order.ID.String(), "order_number": order.OrderNumber,
			"chef_id": order.ChefID.String(), "reason": reason,
			"refunded": true, "initiated_by": "system",
		})
	})
	if err != nil {
		log.Printf("stuck-order: cancel failed for %s: %v", order.ID, err)
		CaptureBackgroundError(err)
		return false
	}
	return true
}

// nudgeStuckOrder reminds both sides that an order is sitting unfinished, at most once per
// reminder window and no more than maxReminders times.
func nudgeStuckOrder(db *gorm.DB, order *models.Order, now time.Time, reminderDays, maxReminders int) bool {
	if order.StaleReminderCount >= maxReminders {
		return false
	}
	if order.LastStaleReminderAt != nil && order.LastStaleReminderAt.After(now.AddDate(0, 0, -reminderDays)) {
		return false
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		// Compare-and-swap on the count so two schedulers racing the same order send one
		// reminder, not two.
		res := tx.Model(&models.Order{}).
			Where("id = ? AND stale_reminder_count = ?", order.ID, order.StaleReminderCount).
			Updates(map[string]any{
				"stale_reminder_count":   order.StaleReminderCount + 1,
				"last_stale_reminder_at": now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil // another pass got there first
		}

		payload := map[string]any{
			"order_id": order.ID.String(), "order_number": order.OrderNumber,
			"chef_id": order.ChefID.String(), "status": string(order.Status),
			"stuck_since": order.CreatedAt, "reminder": order.StaleReminderCount + 1,
		}
		if err := EnqueueEvent(tx, SubjectOrderStale, "order.stale_reminder", order.CustomerID, payload); err != nil {
			return err
		}
		var ids []uuid.UUID
		tx.Model(&models.ChefProfile{}).Where("id = ?", order.ChefID).Pluck("user_id", &ids)
		if len(ids) == 0 {
			return nil
		}
		return EnqueueEvent(tx, SubjectOrderStale, "order.stale_reminder", ids[0], payload)
	})
	if err != nil {
		log.Printf("stuck-order: nudge failed for %s: %v", order.ID, err)
		return false
	}
	return true
}
