package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// loyalty_order_redeem.go — spending points DIRECTLY on an order at checkout.
//
// Distinct from RedeemLoyalty (points → wallet credit), which keeps its 500-point
// MinRedeem floor to stop dust conversions. At checkout any balance is spendable:
// the floor would leave a customer holding 180 unusable points while the whole
// point of the feature is to encourage them to burn the balance.
//
// Both routes still share one monthly cap — see MonthlyRedeemedPaise.

// RedeemLoyaltyToOrder debits `points` from the customer's FIFO earn lots to fund
// an order, writing one ledger entry keyed to that order.
//
// Idempotent per order: a retried payment settle debits nothing further. Returns
// ErrInsufficientLoyaltyPoints if the lots cannot cover the request.
//
// The ledger write and the lot consumption run in ONE transaction, and the ledger
// write takes the account row lock first — so two concurrent settles for the same
// customer cannot both read the same lot snapshot and then clobber it.
func RedeemLoyaltyToOrder(db *gorm.DB, userID, orderID uuid.UUID, points float64) error {
	if points <= 0 {
		return nil
	}
	cfg := GetLoyaltyConfig(db)
	oid := orderID
	return db.Transaction(func(tx *gorm.DB) error {
		_, created, err := applyLoyaltyLedgerTxn(tx, userID, points, models.LoyaltyDebit,
			models.LoyaltySourceOrderRedemption, &oid, "Points applied at checkout",
			"loyalty:order-redeem:"+orderID.String(), nil, cfg)
		if err != nil {
			return err
		}
		if !created {
			return nil // already redeemed for this order
		}
		return consumeBatchesFIFO(tx, userID, points)
	})
}
