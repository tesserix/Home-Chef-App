package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/payouts"
)

// payout_recovery.go — collecting a chef's debt from their next payout (#741).
//
// Route moves money as per-payment transfers, so a penalty cannot be netted
// across orders after the fact the way a daily batch would. Recovery therefore
// happens before the next transfer is created: the outstanding balance comes
// off the gross, floored at zero.
//
// There are exactly two entry points, and the split between them is the whole
// design (#1079):
//
//   - ApplyRecoveryDeduction READS. It answers "what would this payout collect"
//     and changes nothing. The release governor uses it to decide whether to
//     block, and must not discharge a debt merely by looking at it.
//   - CollectRecoveryDeduction COLLECTS. It is called only where money is
//     actually withheld, and writes the resolving entry that pays the debt down.
//
// Adding a third entry point, or making the reader collect, reintroduces the bug
// this file used to document: the same debt charged to a chef on every payout.
//
// Order settlement and the weekly statement (statement_recovery.go, #1092) both
// collect, and both do it through CollectRecoveryDeduction — the shared ledger is
// what stops the second one finding a debt the first already discharged.

// ApplyRecoveryDeduction reduces a gross payout by the chef's outstanding
// recovery balance, without collecting it.
//
// Returns the net to transfer and the amount recovered; the two always sum to
// the gross, so no money is created or destroyed by the deduction.
//
// This is deliberately a pure read — see CollectRecoveryDeduction to collect.
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

// CollectRecoveryDeduction withholds the chef's outstanding recovery balance
// from a gross payout AND discharges what it withheld.
//
// sourceType/sourceID identify the payout the money is withheld from. They make
// collection idempotent: a retried disbursement re-derives the same resolving
// entry id and writes nothing the second time, so a retry can never forgive a
// debt the chef has not paid.
//
// Idempotency rests on the primary key rather than a unique index, because the
// entry id is derived from (payee, source) rather than random. A duplicate is
// then a PK conflict on a constraint that cannot be absent, which matters for a
// table AutoMigrate owns.
func CollectRecoveryDeduction(db *gorm.DB, chefID uuid.UUID, gross payouts.Money, sourceType, sourceID string, now time.Time) (payouts.Money, payouts.Money, error) {
	zero := payouts.Zero(gross.Currency)
	entryID := recoveryCollectionID(chefID, sourceType, sourceID)

	var entries []payouts.LedgerEntry
	if err := db.Where("payee_type = ? AND payee_id = ?", payouts.PayeeChef, chefID).
		Find(&entries).Error; err != nil {
		return zero, zero, err
	}

	// A retry answers from what it already collected, not from the balance it
	// has since discharged — otherwise the second attempt reports withholding
	// money that was never withheld again, and pays the gross on top.
	for _, e := range entries {
		if e.ID == entryID {
			collected := payouts.Money{Minor: e.AmountMinor, Currency: e.Currency}
			net, err := gross.Sub(collected)
			if err != nil {
				return zero, zero, err
			}
			return net, collected, nil
		}
	}

	balance, err := payouts.DeriveBalance(entries, now)
	if err != nil {
		return zero, zero, err
	}
	owed := balance.Recovery()
	if owed.IsZero() {
		return gross, zero, nil
	}

	deducted := owed
	if cmp, err := owed.Cmp(gross); err == nil && cmp > 0 {
		deducted = gross
	}
	net, err := gross.Sub(deducted)
	if err != nil {
		return zero, zero, err
	}

	entry := payouts.LedgerEntry{
		ID:          entryID,
		TenantID:    recoveryTenant(entries),
		PayeeType:   payouts.PayeeChef,
		PayeeID:     chefID,
		Kind:        payouts.EntryCreditRecoveryCollected,
		AmountMinor: deducted.Minor,
		Currency:    deducted.Currency,
		SourceType:  sourceType,
		SourceID:    sourceID,
		CreatedAt:   now,
	}
	if err := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&entry).Error; err != nil {
		return zero, zero, err
	}
	return net, deducted, nil
}

// recoveryCollectionNamespace scopes the derived entry ids so they cannot
// collide with any other deterministic id the platform mints.
var recoveryCollectionNamespace = uuid.MustParse("6f1a4c2e-0d3b-4f5a-9c7e-1b2d3e4f5a60")

func recoveryCollectionID(chefID uuid.UUID, sourceType, sourceID string) uuid.UUID {
	return uuid.NewSHA1(recoveryCollectionNamespace, []byte(chefID.String()+"|"+sourceType+"|"+sourceID))
}

// recoveryTenant is the tenant of the debt being discharged, so a tenant-scoped
// read cannot see a penalty without its settlement.
func recoveryTenant(entries []payouts.LedgerEntry) string {
	for _, e := range entries {
		if e.TenantID != "" {
			return e.TenantID
		}
	}
	return ""
}
