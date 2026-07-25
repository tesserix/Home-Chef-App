package services

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// payout_recovery.go — collecting a chef's debt from their next payout (#741),
// and — as of the chef-cancel penalty (the first real penalty writer) —
// actually discharging it once collected.
//
// Route moves money as per-payment transfers, so a penalty cannot be netted
// across orders after the fact the way a daily batch would. Recovery therefore
// happens before the next transfer is created: the outstanding balance comes
// off the gross, floored at zero.
//
// ApplyRecoveryDeduction itself is a non-discharging read: it never writes a
// resolving ledger entry, so DeriveBalance would re-derive the SAME full debt
// on every call forever if nothing else acted on its result. The discharge is
// therefore a SEPARATE, deliberate step (DischargeChefRecovery) fired by the
// caller once the reduced amount is actually collected — see
// handlers/payment.go's dischargeChefRecoveryForOrder, which is the ONLY site
// that calls it, and services/payout_release_cron.go's BuildReleaseInput,
// which now defers to it instead of independently blocking on the same
// balance (formerly a LANDMINE: two uncoordinated sites acting on one debt).
//
// Chef statements do not yet show a recovery line item (services/statement.go
// builds NetPayout from ComputeOrderEarnings alone), so once a deduction fires
// here, the amount actually transferred will disagree with the chef's own
// statement until that gap is closed.

// recoveryLedgerEntryExists reports whether a ledger row already exists for
// the given (payee, kind, source) tuple — the shared dedupe check both
// RaiseChefRecoveryPenalty and DischargeChefRecovery use so a retried caller
// (a re-cancel, a redelivered webhook, a racing concurrent request) never
// writes the same entry twice. Check-then-insert rather than a DB-level
// unique constraint (none exists on this table today) — acceptable here the
// same way the rest of this money-adjacent code accepts best-effort
// idempotency over a hard constraint (see chef_order_cancel.go's own
// dedupe comments).
func recoveryLedgerEntryExists(db *gorm.DB, payeeType payouts.PayeeType, payeeID uuid.UUID, kind payouts.EntryKind, sourceType, sourceID string) (bool, error) {
	var count int64
	err := db.Model(&payouts.LedgerEntry{}).
		Where("payee_type = ? AND payee_id = ? AND kind = ? AND source_type = ? AND source_id = ?",
			payeeType, payeeID, kind, sourceType, sourceID).
		Count(&count).Error
	return count > 0, err
}

// RaiseChefRecoveryPenalty writes a debit.penalty entry against a chef for
// amountPaise, sourced from the order that caused it (e.g. a chef-fault
// cancel — see handlers/chef_order_cancel.go's CancelOrder). A no-op when
// amountPaise <= 0.
//
// Idempotent on (payee, debit.penalty, "order", order.ID): a retried caller —
// notably a concurrent duplicate CancelOrder request that loses
// ReserveFullRefund's claim but still runs the rest of the handler — finds
// the existing row and writes nothing a second time, so the SAME order can
// never penalise its chef twice.
func RaiseChefRecoveryPenalty(db *gorm.DB, chefID uuid.UUID, order *models.Order, amountPaise int64, reason string) error {
	if amountPaise <= 0 {
		return nil
	}
	sourceID := order.ID.String()
	exists, err := recoveryLedgerEntryExists(db, payouts.PayeeChef, chefID, payouts.EntryDebitPenalty, "order", sourceID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	entry := payouts.LedgerEntry{
		ID:          uuid.New(),
		PayeeType:   payouts.PayeeChef,
		PayeeID:     chefID,
		Kind:        payouts.EntryDebitPenalty,
		AmountMinor: amountPaise,
		Currency:    payouts.CurrencyINR,
		SourceType:  "order",
		SourceID:    sourceID,
		Reason:      reason,
	}
	return db.Create(&entry).Error
}

// DischargeChefRecovery writes the resolving credit.recovery_collected entry
// that nets a chef's recovery debt down by deductedPaise — the amount
// ApplyRecoveryDeduction actually took off a settling order's Route transfer
// at creation. Sourced from the SETTLING order (not the order that raised the
// debt), so a chef's debt collected in slices across several orders leaves
// one resolving entry per collecting order, and DeriveBalance's fold nets
// them all against the original debit: fully to zero once the whole debt has
// been collected, or partially — leaving the remainder to be picked up by the
// chef's next settling order — when it has not. A no-op when
// deductedPaise <= 0.
//
// Idempotent on (payee, credit.recovery_collected, "order", settlingOrderID):
// the client-verify path and the payment.captured webhook both call this for
// the same order (whichever confirms gateway capture first), and a
// redelivered webhook retries it again later — only the first write lands.
func DischargeChefRecovery(db *gorm.DB, chefID, settlingOrderID uuid.UUID, settlingOrderNumber string, deductedPaise int64) error {
	if deductedPaise <= 0 {
		return nil
	}
	sourceID := settlingOrderID.String()
	exists, err := recoveryLedgerEntryExists(db, payouts.PayeeChef, chefID, payouts.EntryCreditRecoveryCollected, "order", sourceID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	entry := payouts.LedgerEntry{
		ID:          uuid.New(),
		PayeeType:   payouts.PayeeChef,
		PayeeID:     chefID,
		Kind:        payouts.EntryCreditRecoveryCollected,
		AmountMinor: deductedPaise,
		Currency:    payouts.CurrencyINR,
		SourceType:  "order",
		SourceID:    sourceID,
		Reason:      "recovery collected via order " + settlingOrderNumber,
	}
	return db.Create(&entry).Error
}

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
