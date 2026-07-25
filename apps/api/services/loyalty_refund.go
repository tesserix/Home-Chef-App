package services

import (
	"errors"

	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReverseOrderLoyalty claws back the points earned on an order when it is refunded — but
// only the portion still sitting unredeemed in THAT order's earn lot. It debits the account
// and zeroes the specific lot directly (never routing through the account-wide FIFO
// consumer), so a reversal can never drain an unrelated, un-refunded order's lot. Idempotent
// per order. Runs inside the caller's refund transaction.
func ReverseOrderLoyalty(tx *gorm.DB, orderID uuid.UUID) error {
	q := tx.Where("order_id = ? AND source = ?", orderID, models.LoyaltySourceOrder)
	if tx.Dialector.Name() == "postgres" {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var lot models.LoyaltyEarnBatch
	err := q.First(&lot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil // nothing earned on this order
	}
	if err != nil {
		return err
	}
	if lot.PointsRemaining <= 0 {
		return nil // already redeemed or expired — nothing left to claw back
	}
	toReverse := lot.PointsRemaining
	cfg := GetLoyaltyConfig(tx)
	oid := orderID
	_, created, err := applyLoyaltyLedgerTxn(tx, lot.UserID, toReverse, models.LoyaltyDebit,
		models.LoyaltyTxnSource("refund_reversal"), &oid, "Order refunded — earned points reversed",
		"loyalty:order-refund:"+orderID.String(), nil, cfg)
	if err != nil {
		return err
	}
	if !created {
		return nil // already reversed (idempotent)
	}
	// Zero exactly THIS order's lot (targeted, not FIFO): account balance and this lot both
	// drop by toReverse, and no other order's lot is touched.
	return tx.Model(&models.LoyaltyEarnBatch{}).Where("id = ?", lot.ID).
		Update("points_remaining", 0).Error
}
