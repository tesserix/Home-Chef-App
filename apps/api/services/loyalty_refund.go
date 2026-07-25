package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// ReverseOrderLoyalty claws back the points earned on an order when it is refunded — but only
// the portion still sitting unredeemed in that order's earn lot (if the customer already
// redeemed them to their wallet, the wallet refund handles that side, so we don't over-debit).
// Idempotent per order. Runs inside the caller's refund transaction.
func ReverseOrderLoyalty(tx *gorm.DB, orderID uuid.UUID) error {
	var lot models.LoyaltyEarnBatch
	err := tx.Where("order_id = ? AND source = ?", orderID, models.LoyaltySourceOrder).First(&lot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil // nothing earned on this order
	}
	if err != nil {
		return err
	}
	if lot.PointsRemaining <= 0 {
		return nil // already spent or expired
	}
	cfg := GetLoyaltyConfig(tx)
	oid := orderID
	_, _, err = applyLoyaltyTxnInTx(tx, lot.UserID, lot.PointsRemaining, models.LoyaltyDebit,
		models.LoyaltyTxnSource("refund_reversal"), &oid, "Order refunded — earned points reversed",
		"loyalty:order-refund:"+orderID.String(), nil, cfg)
	return err
}
