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
// off the gross, floored at zero.
//
// This is a non-discharging read: nothing here (or anywhere in this package)
// writes a resolving ledger entry, so DeriveBalance re-derives the SAME FULL
// original debt on every call, forever — it is not progressively paid down.
// The same debt is re-collected from every future order until a resolving
// entry exists. No penalty writer may ship without an accompanying
// resolving-entry writer.
//
// TODO: recovery is non-discharging until a resolving ledger entry exists.
//
// Chef statements do not yet show a recovery line item (services/statement.go
// builds NetPayout from ComputeOrderEarnings alone), so once a deduction fires
// here, the amount actually transferred will disagree with the chef's own
// statement until that gap is closed.

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
