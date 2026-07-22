package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/payouts"
)

// payout_recovery.go — collecting a chef's debt from their next payout (#741).
//
// Route moves money as per-payment transfers, so a penalty cannot be netted
// across orders after the fact the way a daily batch would. Recovery therefore
// happens before the next transfer is created: the outstanding balance comes
// off the gross, floored at zero, and whatever is left stays owed and is
// collected from the order after that.

// ApplyRecoveryDeduction reduces a gross payout by the chef's outstanding
// recovery balance.
//
// Returns the net to transfer and the amount recovered; the two always sum to
// the gross, so no money is created or destroyed by the deduction.
func ApplyRecoveryDeduction(db *gorm.DB, chefID uuid.UUID, gross payouts.Money, now time.Time) (payouts.Money, payouts.Money, error) {
	zero := payouts.Zero(gross.Currency)

	var entries []payouts.LedgerEntry
	if err := db.Where("payee_type = ? AND payee_id = ?", payouts.PayeeChef, chefID).
		Find(&entries).Error; err != nil {
		return zero, zero, err
	}

	balance, err := payouts.DeriveBalance(entries, now)
	if err != nil {
		return zero, zero, err
	}
	owed := balance.Recovery()
	if owed.IsZero() {
		return gross, zero, nil
	}

	// Deduct at most the whole payout — never emit a negative transfer.
	deducted := owed
	if cmp, err := owed.Cmp(gross); err == nil && cmp > 0 {
		deducted = gross
	}
	net, err := gross.Sub(deducted)
	if err != nil {
		return zero, zero, err
	}
	return net, deducted, nil
}
