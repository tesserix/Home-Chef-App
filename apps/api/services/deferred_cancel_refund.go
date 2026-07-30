package services

// deferred_cancel_refund.go — the retry side of the chef-cancel gateway-refund deferral.
//
// ChefOrderCancelHandler.CancelOrder must never hard-block a chef's cancel on the
// synchronous Razorpay refund: the full-refund obligation is reserved (payment_status /
// refunded_at / refund_amount) unconditionally, but when GetRazorpay() is nil or
// CreateRefund errors, the handler defers the gateway call instead of failing the
// request — it stamps a durable sentinel ("pending:gateway-retry:<paise>") into
// orders.refund_id and returns 200. RetryDeferredCancelRefunds is the sweep that finds
// those deferred orders and re-issues the gateway refund until it lands.
//
// SAFETY: the retry re-sends the SAME stable idempotency key
// (RefundFullIdempotencyKey(order.ID)) the original CancelOrder call used. So if that
// original gateway call actually succeeded but its HTTP response was lost (timeout,
// network drop) — the exact reason we deferred instead of erroring — the retry dedups
// to the SAME Razorpay refund instead of issuing a second one. No double refund is
// possible on this path.
//
// The guarded UPDATE (`WHERE refund_id LIKE 'pending:gateway-retry:%'`) is the
// serialization point: if a concurrent actor already healed the row between our read
// and write, RowsAffected is 0 and we simply don't count it — no clobber.
//
// DeferredCancelRefundPrefix (below) is exported so handlers.ChefOrderCancelHandler.CancelOrder
// builds the SAME sentinel from this one constant instead of keeping its own duplicate literal.

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// DeferredCancelRefundPrefix is the single source of truth for the deferred-gateway-refund
// sentinel written into orders.refund_id. handlers.ChefOrderCancelHandler.CancelOrder builds
// the sentinel from this exported constant (handlers can import services) instead of keeping
// its own duplicate literal — two independently-maintained copies risk drifting, which would
// silently strand a refund the cron/Temporal retry can never match.
const DeferredCancelRefundPrefix = "pending:gateway-retry:"

// deferredRefundGrace is how long a deferred sentinel must sit before the retry sweep
// acts on it — guards against racing the handler's own write (read-then-write inside
// CancelOrder is effectively instantaneous, but the grace keeps this sweep from ever
// contending with an in-flight request for the same order).
const deferredRefundGrace = 2 * time.Minute

// RetryDeferredCancelRefunds finds cancelled orders whose gateway refund was deferred
// by a chef cancel and re-issues it, idempotently. Returns the count healed.
func RetryDeferredCancelRefunds() int {
	cutoff := time.Now().Add(-deferredRefundGrace)
	var orders []models.Order
	if err := database.DB.
		Where("status = ? AND refund_id LIKE ? AND razorpay_payment_id <> '' AND updated_at < ?",
			models.OrderStatusCancelled, DeferredCancelRefundPrefix+"%", cutoff).
		Limit(sweepBatchLimit).
		Find(&orders).Error; err != nil {
		log.Printf("deferred-cancel-refund: load failed: %v", err)
		return 0
	}
	healed := 0
	for i := range orders {
		if retryOneDeferredCancelRefund(orders[i].ID) {
			healed++
		}
	}
	if healed > 0 {
		log.Printf("deferred-cancel-refund: healed %d deferred cancel refund(s)", healed)
	}
	return healed
}

// retryOneDeferredCancelRefund re-reads and re-issues ONE deferred cancel refund under a
// row lock, mirroring finalizeStuckRefund's dialect-guarded lock + re-check-under-lock
// discipline (stuck_refund_reconcile.go). Returns whether it healed the order.
func retryOneDeferredCancelRefund(orderID uuid.UUID) bool {
	healed := false
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		lockTx := tx
		if tx.Dialector.Name() == "postgres" {
			lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var o models.Order
		if e := lockTx.Select("id", "refund_id", "razorpay_payment_id").
			First(&o, "id = ?", orderID).Error; e != nil {
			return e
		}
		// A concurrent actor (another sweep tick, or an unlikely second CancelOrder retry)
		// may already have completed this refund — re-check under the lock.
		if !strings.HasPrefix(o.RefundID, DeferredCancelRefundPrefix) {
			return nil
		}
		paise, pErr := strconv.Atoi(strings.TrimPrefix(o.RefundID, DeferredCancelRefundPrefix))
		if pErr != nil || paise <= 0 {
			log.Printf("deferred-cancel-refund: order %s has an unparseable sentinel %q; skipping", orderID, o.RefundID)
			return nil
		}

		// Provider-agnostic: the deferral that created this sentinel could have come
		// from any gateway, so the retry must route the same way the original call
		// did (gateway_refund.go) rather than assuming Razorpay — a Cashfree order
		// deferred here would otherwise be retried against a Razorpay payment id it
		// does not have, and would never heal.
		if !GatewayRefundAvailable(&o) {
			return nil // gateway still unavailable — leave the sentinel, retry next sweep
		}

		refundResp, cErr := IssueOrderGatewayRefund(&o, paise, map[string]string{
			"order_id":  o.ID.String(),
			"reason":    "deferred chef cancel",
			"initiator": "reconcile",
			// SAME key CancelOrder used — a lost-response success dedups here instead of
			// double-refunding. See the file header.
		}, RefundFullIdempotencyKey(o.ID))
		if cErr != nil {
			log.Printf("deferred-cancel-refund: gateway refund still failing for order %s: %v", orderID, cErr)
			return nil // leave the sentinel; the next sweep retries
		}

		res := tx.Model(&models.Order{}).
			Where("id = ? AND refund_id LIKE ?", orderID, DeferredCancelRefundPrefix+"%").
			Update("refund_id", refundResp.RefundID)
		if res.Error != nil {
			return res.Error
		}
		healed = res.RowsAffected == 1
		return nil
	})
	if err != nil {
		log.Printf("deferred-cancel-refund: retry for order %s failed: %v", orderID, err)
		return false
	}
	return healed
}
