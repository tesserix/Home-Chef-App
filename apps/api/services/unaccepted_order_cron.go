package services

// unaccepted_order_cron.go — refund a PAID order no chef ever accepted (#694).
//
// THE HOLE THIS CLOSES
//
// A customer pays, the money is captured, and no chef accepts. Nothing happened.
// No timeout, no escalation, no refund. The order sat `pending` forever and the
// customer was simply out of pocket until they contacted support.
//
// stale_order_cron.go looks like it covers this and does not: it filters
// payment_status = pending, i.e. abandoned checkouts where nothing was captured.
// The money case had no sweeper at all.
//
// WHY THIS EXISTS RATHER THAN JUST ENABLING THE SAGA
//
// temporal/workflows/order.go already implements this: a 30-minute
// chefAcceptTimeout that compensates. It is written, merged, tested — and gated
// behind ORDER_SAGA_ENABLED, which defaults false and is set NOWHERE (not in
// tesserix-k8s, not on the running pod). So it never runs.
//
// Turning it on is the better long-term answer (#127), but it is not a one-line
// swap today, because the saga's compensation (CompensateOrderRefund) credits
// WALLET STORE CREDIT — not the original payment method. Customer-facing policy
// is refund-to-original (#686), and the wallet is not spendable. Flipping the
// flag would trade "no refund" for "a refund they cannot use", which is the #691
// bug in a new place.
//
// So this sweep refunds through RefundOrderForCancellation → the #689
// coordinator → the provider the customer actually paid with. When the saga does
// come on, sagaActive() below hands the job straight back to it.

import (
	"context"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

const (
	unacceptedOrderInterval = 5 * time.Minute
	// Bound the batch so a backlog can't monopolise a tick.
	unacceptedOrderBatch = 200
)

// StartUnacceptedOrderCron launches the paid-but-unaccepted sweep. Returns
// immediately; lives for the life of ctx.
func StartUnacceptedOrderCron(ctx context.Context) {
	go func() {
		runUnacceptedOrderScan(ctx)
		ticker := time.NewTicker(unacceptedOrderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("unaccepted-order: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runUnacceptedOrderScan(ctx)
			}
		}
	}()
	log.Printf("unaccepted-order: cron started (interval=%s, deadline=chef kitchen close per slot)",
		unacceptedOrderInterval)
}

func runUnacceptedOrderScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("unaccepted-order: panic recovered: %v", r)
		}
	}()

	// The saga owns this when it is on. Two things auto-refunding the same order
	// is exactly the double-refund the whole payments epic exists to prevent —
	// the shared reservation would stop the money moving twice, but the customer
	// would still get two cancellation notifications and we would be racing our
	// own compensation for no reason.
	if sagaActive() {
		return
	}

	voided := refundUnacceptedOrders(ctx, database.DB, time.Now())
	if voided > 0 {
		log.Printf("unaccepted-order: voided + refunded %d paid order(s) the kitchen closed on", voided)
	}
}

// refundUnacceptedOrders is the scan body, taking db + now so it is testable
// without a ticker or a clock wait. Returns how many orders it voided.
func refundUnacceptedOrders(ctx context.Context, db *gorm.DB, now time.Time) int {
	// Every still-pending paid order is a CANDIDATE; the deadline decides. It
	// cannot be a SQL cutoff on created_at, because the deadline depends on the
	// chef's schedule and the slot the order is FOR — an order placed today for
	// Saturday's dinner is younger than one placed an hour ago for today's lunch,
	// yet has days longer to live. Filtering by age is exactly the bug.
	var stranded []models.Order
	if err := db.
		Preload("Items").
		Where("payment_status = ? AND status = ?",
			models.PaymentCompleted, models.OrderStatusPending).
		Order("created_at ASC").
		Limit(unacceptedOrderBatch).
		Find(&stranded).Error; err != nil {
		log.Printf("unaccepted-order: query failed: %v", err)
		return 0
	}

	// voidReason is customer-facing via the apology, so it says what happened to
	// THEM, not what our scheduler did.
	const voidReason = "the kitchen closed before this order was accepted"

	refunded := 0
	for i := range stranded {
		select {
		case <-ctx.Done():
			return refunded
		default:
		}
		order := stranded[i]

		// The chef's own closing time for the meal this order is FOR. Still open →
		// still theirs.
		deadline := ResolveAcceptDeadline(db, &order)
		if now.Before(deadline.At) {
			continue
		}

		// A typed escrow order (meal-plan day / group) is owned end-to-end by ITS
		// flow — refunds on a disjoint keyspace, and a held chef payout this generic
		// path cannot reverse. Skip it entirely.
		//
		// This has to be checked HERE and not left to RefundOrderForCancellation:
		// that function skips typed orders by returning nil, which is
		// indistinguishable from "refunded fine" — so the sweep would sail on and
		// CANCEL a meal-plan day it never refunded, stealing the lifecycle from the
		// flow that owns it. A nil that means "I did nothing" is not a nil that
		// means "done". Caught by TestUnacceptedSweep_SkipsTypedEscrowOrders.
		//
		// Fail safe on a type-check error: never generic-refund a possibly-typed order.
		switch kind, kErr := TypedRefundOrderKind(db, order.ID); {
		case kErr != nil:
			log.Printf("unaccepted-order: type check failed for %s (skipping): %v", order.ID, kErr)
			continue
		case kind != "":
			continue
		}

		// Refund FIRST, and only cancel if the money actually moved.
		//
		// The other order — cancel then refund — is tempting because it stops the
		// chef accepting a doomed order, but it is how you end up with a cancelled
		// order the customer was never refunded for: the worst possible state, and
		// invisible, because the order LOOKS resolved.
		//
		// A refund failure leaves the order pending and this sweep retries it in 5
		// minutes. RefundOrderForCancellation is idempotent (the shared claim) and
		// no-ops an order a sibling path already refunded, so retrying is safe.
		if err := RefundOrderForCancellation(&order, "system", voidReason); err != nil {
			log.Printf("unaccepted-order: refund failed for %s (will retry): %v", order.ID, err)
			CaptureBackgroundError(err)
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			// Guarded on status: a chef accepting in the gap between the refund and
			// this update would otherwise have their acceptance clobbered back to
			// cancelled. RowsAffected==0 means they beat us — the refund still
			// stands and the reconcile cron's refund_mismatch check surfaces it,
			// which is the right outcome: money out, order visibly odd, a human
			// looks. Silently cancelling under a chef mid-cook is worse.
			res := tx.Model(&models.Order{}).
				Where("id = ? AND status = ?", order.ID, models.OrderStatusPending).
				Updates(map[string]any{
					"status":        models.OrderStatusCancelled,
					"cancel_reason": voidReason,
					"cancelled_at":  now,
				})
			if res.Error != nil {
				return res.Error
			}
			if res.RowsAffected == 0 {
				log.Printf("unaccepted-order: %s left pending — a chef acted during the refund; refund stands", order.ID)
				return nil
			}

			// Release the reserved capacity + slot — these dishes will not be made,
			// and holding them means the chef cannot sell that cover to anyone else.
			capDay := CapacityDay(order.CreatedAt)
			for _, it := range order.Items {
				if err := ReleaseCapacity(tx, it.MenuItemID, it.Quantity, capDay); err != nil {
					return err
				}
			}
			if order.DeliverySlot != "" && order.ScheduledFor != nil {
				if err := ReleaseSlot(tx, order.ChefID, order.DeliverySlot, 1, CapacityDay(*order.ScheduledFor)); err != nil {
					return err
				}
			}

			// Staged in the same tx (outbox) so the customer is told even if this
			// process dies right after the commit. Being refunded without being told
			// reads as "they took my money and cancelled my dinner".
			return EnqueueEvent(tx, SubjectOrderVoided, "order.voided", order.CustomerID, map[string]any{
				"order_id":     order.ID.String(),
				"order_number": order.OrderNumber,
				"chef_id":      order.ChefID.String(),
				"reason":       voidReason,
				"refunded":     true,
				"initiated_by": "system",
				// Carried so the apology can be specific ("your Sunday lunch") and so
				// a void is explainable after the fact rather than a bare timestamp.
				"slot":              deadline.Slot,
				"deadline":          deadline.At,
				"deadline_source":   deadline.Source,
				"refunded_amount":   order.Total,
				"apology_warranted": true,
			})
		}); err != nil {
			// The refund already happened, so never retry the refund on this path —
			// surface it instead. The reconcile cron is the backstop.
			log.Printf("unaccepted-order: %s refunded but finalize failed: %v", order.ID, err)
			CaptureBackgroundError(err)
			continue
		}
		refunded++
	}
	return refunded
}
