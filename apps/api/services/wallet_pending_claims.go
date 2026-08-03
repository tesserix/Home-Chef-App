package services

// wallet_pending_claims.go — credit that an unpaid order has already spoken for
// (#936).
//
// Wallet credit is stamped on the order at checkout (`wallet_applied`) but is not
// deducted from the wallet balance until the payment settles. Between those two
// moments the balance still reads full, so a customer who abandons one payment
// and starts another is offered the same rupees a second time.
//
// Observed live: two pending orders each claiming ₹254.55 of a ₹255.30 balance,
// and a third checkout offering the whole ₹255.30 again.
//
// This is not a double-spend — the ledger locks the row and the settle path
// refuses to pay a chef out of credit it never collected. The damage is that the
// second order cannot complete: it settles into reconcile instead, silently, and
// the customer is left with an order that looks placed and never progresses.
//
// The honest balance to offer is therefore the wallet less whatever unpaid orders
// have already claimed. A claim disappears on its own when the order reaches a
// terminal state — the stale-order cron cancels abandoned checkouts within ~30
// minutes — so this frees itself without any reaper of its own.

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// WalletClaimedByUnpaidOrdersPaise is the wallet credit stamped on this
// customer's orders that are still awaiting payment.
//
// excludeOrderID skips one order — the order being quoted, when that order
// already exists. A checkout preview passes uuid.Nil (nothing to exclude); a
// re-quote of a persisted order must not count its own claim against itself, or
// the credit it already holds would appear to be someone else's.
//
// Terminal orders are excluded: their claim is released, whether they were
// cancelled by the customer, rejected by the chef, or swept up by the
// stale-order cron.
func WalletClaimedByUnpaidOrdersPaise(db *gorm.DB, userID, excludeOrderID uuid.UUID) (int, error) {
	if userID == uuid.Nil {
		return 0, nil
	}
	q := db.Model(&models.Order{}).
		Where("customer_id = ?", userID).
		Where("payment_status = ?", models.PaymentPending).
		Where("status NOT IN ?", []models.OrderStatus{
			models.OrderStatusCancelled,
			models.OrderStatusRejected,
			models.OrderStatusRefunded,
			models.OrderStatusDelivered,
		}).
		Where("COALESCE(wallet_applied, 0) > 0")
	if excludeOrderID != uuid.Nil {
		q = q.Where("id <> ?", excludeOrderID)
	}

	var claimed float64
	if err := q.Select("COALESCE(SUM(wallet_applied), 0)").Scan(&claimed).Error; err != nil {
		return 0, err
	}
	if claimed <= 0 {
		return 0, nil
	}
	return ToPaise(claimed), nil
}

// spendableWalletPaise is the balance a checkout may actually offer: what is in
// the wallet, less what unpaid orders have already claimed, never below zero.
func spendableWalletPaise(balancePaise, claimedPaise int) int {
	if claimedPaise <= 0 {
		return balancePaise
	}
	if claimedPaise >= balancePaise {
		return 0
	}
	return balancePaise - claimedPaise
}
