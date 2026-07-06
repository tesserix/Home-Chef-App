package services

// delivery_failure_reconcile.go — #594. The durable safety net behind the delivery-failure
// freeze for every path that leaves a failed/returned `deliveries` row (own-fleet courier
// + 3PL webhooks; see COVERAGE below for the chef-self-delivery caveat). The freeze —
// open a `delivery_failed` OrderIssue + dispute the order's payout hold — runs
// synchronously when a delivery is reported failed/returned. If that synchronous path
// errored (a 3PL webhook that exhausted its retry budget, an own-fleet retry whose freeze
// crashed), the order strands: status=failed/returned, hold never disputed, no dispute
// ticket. It is MONEY-SAFE (the hold stays `none`, so the payout queue never pays the
// chef for a non-delivered order) but the customer is never refunded and no admin is
// alerted. This sweep re-drives TerminalizeDeliveryFailure for such orders.
//
// Runs REGARDLESS of the escrow flags — disputing a hold is plain DB state (no money
// moves), so it must self-heal even pre-launch. Scoped to GATEWAY orders (a
// razorpay_order_id, so TerminalizeDeliveryFailure can actually freeze them) that are NOT
// already frozen (hold still none/awaiting AND no pending delivery_failed issue) nor
// resolved. Group/meal-plan-day shell orders (no razorpay_order_id) are excluded so the
// sweep never re-drives a froze=false no-op forever (their freeze handlers are tracked
// separately).
//
// COVERAGE: keyed on a failed/returned `deliveries` row, so it catches the 3PL webhook
// paths and the own-fleet courier path (whose handler now persists the failed status
// BEFORE the freeze). Chef self-delivery creates NO deliveries row, so a stranded chef
// self-delivery freeze is out of this sweep's reach — mitigated because that path is a
// synchronous chef HTTP action the chef re-tries on error (unlike the async 3PL webhook,
// which has no human in the loop and is the primary reason this net exists). An
// order-level chef-self-delivery reconcile is a tracked follow-up.
//
// GRACE: a strand must have a `deliveries.updated_at` older than the grace window. That
// timestamp is bumped by each provider webhook re-fire, so while a provider is still
// retrying (and might yet freeze the order) the row stays inside the window and is left
// alone — the sweep only acts once the retries have stopped and the strand is genuine.

import (
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// deliveryFailureReconcileGrace is how long a failed/returned delivery must have been
// stale before the reconcile freezes it — long enough that the synchronous freeze (and
// any provider webhook retries) has certainly finished, so the sweep only catches genuine
// strands and doesn't race an in-flight freeze into a duplicate issue.
const deliveryFailureReconcileGrace = 15 * time.Minute

// reconcileStrandedDeliveryFailures freezes any gateway order left stranded by a
// delivery-failure freeze that never completed. Idempotent (TerminalizeDeliveryFailure
// no-ops on an already-frozen order); once frozen the order carries a pending
// delivery_failed issue and is excluded next sweep, so the sweep self-terminates. Returns
// the number of orders freshly frozen.
func reconcileStrandedDeliveryFailures() int {
	type strandRow struct {
		DeliveryID string
		OrderID    string
	}
	var rows []strandRow
	cutoff := time.Now().Add(-deliveryFailureReconcileGrace)
	err := database.DB.Table("deliveries").
		Select("deliveries.id AS delivery_id, deliveries.order_id AS order_id").
		Joins("JOIN orders ON orders.id = deliveries.order_id AND orders.deleted_at IS NULL").
		Where("deliveries.status IN ?", []models.DeliveryStatus{models.DeliveryFailed, models.DeliveryReturned}).
		Where("deliveries.updated_at < ?", cutoff).
		Where("orders.razorpay_order_id <> ''").
		Where("orders.payout_hold_status IN ?", []models.PayoutHoldStatus{models.PayoutHoldNone, models.PayoutHoldAwaitingConfirmation}).
		Where("orders.status NOT IN ?", []models.OrderStatus{models.OrderStatusRefunded, models.OrderStatusCancelled}).
		Where("NOT EXISTS (SELECT 1 FROM order_issues oi WHERE oi.order_id = orders.id AND oi.reason = ? AND oi.status = ?)",
							models.IssueDeliveryFailed, models.IssuePending).
		Order("deliveries.updated_at ASC"). // oldest strands first so a backlog can't starve them past the batch limit
		Limit(sweepBatchLimit).Scan(&rows).Error
	if err != nil {
		log.Printf("delivery-failure-reconcile: query stranded orders failed: %v", err)
		return 0
	}

	driven := 0
	for _, r := range rows {
		orderID, err := uuid.Parse(r.OrderID)
		if err != nil {
			log.Printf("delivery-failure-reconcile: bad order id %q: %v", r.OrderID, err)
			continue
		}
		var order models.Order
		if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
			log.Printf("delivery-failure-reconcile: load order %s: %v", orderID, err)
			continue
		}
		froze, err := TerminalizeDeliveryFailure(database.DB, &order, models.FailureOther, "reconcile",
			map[string]any{"delivery_id": r.DeliveryID, "source": "reconcile"})
		if err != nil {
			log.Printf("delivery-failure-reconcile: terminalize order %s: %v", orderID, err)
			continue
		}
		if froze {
			driven++
			log.Printf("delivery-failure-reconcile: froze stranded order %s (delivery %s)", orderID, r.DeliveryID)
		}
	}
	if driven > 0 {
		log.Printf("delivery-failure-reconcile: froze %d stranded delivery-failure order(s)", driven)
	}
	return driven
}
