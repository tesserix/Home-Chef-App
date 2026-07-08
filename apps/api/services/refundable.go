package services

import (
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// refundable.go — #527. The single correct "how much is still owed to the customer"
// computation for every refund/cancel path.
//
// The naive `order.Total - order.RefundAmount` under-states after a PER-LINE cancel:
// recomputeOrderTotals reduces Order.Total by exactly the cancelled line's value AND
// that same value is added to Order.RefundAmount — so subtracting RefundAmount from the
// already-reduced Total double-counts the line and strands the remaining LIVE items'
// money (a full cancel then refunds ≈0). Goodwill / issue / full-cancel refunds bump
// RefundAmount WITHOUT touching Total or the item rows, so only per-line cancels create
// the skew — and per-line cancels are the only refunds recorded on OrderItem.RefundAmount.
// Adding that per-line total back recovers the true amount paid − already-refunded.

// PerLineRefundedTotal sums refunds already issued per-line (recorded on cancelled
// order_items). This is exactly the amount recomputeOrderTotals removed from Order.Total,
// so callers add it back to undo the Total-vs-RefundAmount double-subtraction (#527).
// Zero when no line was cancelled ⇒ RemainingRefundable is a strict no-op change for
// orders that never had a per-line cancel. Fails safe (returns 0, i.e. the old
// under-refunding value) so a query error can never OVER-refund.
func PerLineRefundedTotal(orderID uuid.UUID) float64 {
	return PerLineRefundedTotalTx(database.DB, orderID)
}

// PerLineRefundedTotalTx is PerLineRefundedTotal computed via the provided db/tx — use it
// INSIDE a transaction so the sum is read on the SAME connection as the enclosing work
// (a cross-connection read is a separate DB under sqlite :memory: and a separate snapshot
// under a Postgres tx).
func PerLineRefundedTotalTx(db *gorm.DB, orderID uuid.UUID) float64 {
	sum, err := PerLineRefundedTotalTxErr(db, orderID)
	if err != nil {
		log.Printf("refundable: sum per-line refunds for order %s failed (treating as 0): %v", orderID, err)
		return 0
	}
	return sum
}

// PerLineRefundedTotalTxErr is PerLineRefundedTotalTx but PROPAGATES the query error instead of
// failing open to 0. Use it wherever a 0-on-error flips the safety direction — notably when
// reconstructing the original captured amount (Total + per-line): understating per-line there
// UNDERSTATES captured, which could misclassify a still-live, partially-refunded order as fully
// refunded (#640). Callers that reconstruct captured MUST use this and skip/alert on error.
func PerLineRefundedTotalTxErr(db *gorm.DB, orderID uuid.UUID) (float64, error) {
	var sum float64
	if err := db.Model(&models.OrderItem{}).
		Where("order_id = ? AND is_cancelled = ?", orderID, true).
		Select("COALESCE(SUM(refund_amount), 0)").Scan(&sum).Error; err != nil {
		return 0, err
	}
	return sum, nil
}

// RemainingRefundable is the amount still owed to the customer on an order:
// what they PAID minus what's already been refunded. It corrects the naive
// Total - RefundAmount, which strands the remaining live items' money after a
// per-line cancel (#527). Never returns negative.
//
// It reads Total, RefundAmount AND the per-line sum in ONE consistent snapshot
// keyed by the order id — deliberately NOT trusting the caller's in-memory
// order.Total / order.RefundAmount, which can be STALE by the time a refund runs
// (a concurrent per-line cancel may have moved them). Mixing a stale Total with a
// fresh per-line sum would over-count and OVER-REFUND, so all three inputs must
// come from the same committed state. On a query error it falls back to the
// caller's snapshot minus refunded (the old under-refunding value) — never over-refunds.
func RemainingRefundable(order *models.Order) float64 {
	var row struct {
		Total        float64
		RefundAmount float64
		PerLine      float64
	}
	err := database.DB.Raw(`
		SELECT o.total AS total, o.refund_amount AS refund_amount,
		       COALESCE((SELECT SUM(oi.refund_amount) FROM order_items oi
		                 WHERE oi.order_id = o.id AND oi.is_cancelled = ?), 0) AS per_line
		FROM orders o WHERE o.id = ?`, true, order.ID).Scan(&row)
	if err.Error != nil || err.RowsAffected == 0 {
		if err.Error != nil {
			log.Printf("refundable: consistent read for order %s failed; using caller snapshot: %v", order.ID, err.Error)
		}
		return maxZero(order.Total - order.RefundAmount) // fail safe: never over-refund
	}
	return maxZero(row.Total - row.RefundAmount + row.PerLine)
}

func maxZero(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}
