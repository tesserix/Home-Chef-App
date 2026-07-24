package services

// ledger.go — Phase 1 of the wallet→ledger evolution (docs/wallet-ledger-plan.md).
// A double-entry, append-only, idempotent ledger in paise. PostgreSQL is the source of
// financial truth. This is the CORE only: posting + balance projection + the balance
// invariant. It is not yet wired to the live wallet paths — that dual-write comes next,
// gated, so the ledger runs in shadow and is reconciled against the legacy float balance
// before any read flips over. No behaviour change until the shadow flag is enabled.

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// LedgerLeg is one side of a posting. Amount is always positive (paise); Direction signs it.
type LedgerLeg struct {
	AccountKind models.LedgerAccountKind
	UserID      *uuid.UUID // required for user_wallet, nil for system accounts
	Direction   models.LedgerDirection
	Amount      models.Money
}

// PostLedgerTransaction records one immutable, balanced double-entry transaction:
// Σdebit == Σcredit, every amount > 0, ≥ 2 legs. Idempotent on idempotencyKey — the same
// business event (e.g. "mealplan-refund:<dayID>") posts exactly once even under retries or
// concurrent callers. Runs inside the caller's tx so the posting commits atomically with
// the state change that triggered it (the transactional-outbox discipline, applied to money).
func PostLedgerTransaction(tx *gorm.DB, idempotencyKey, reason, refType, refID string, legs []LedgerLeg) (*models.LedgerTransaction, error) {
	if idempotencyKey == "" {
		return nil, errors.New("ledger: idempotency key required")
	}
	if len(legs) < 2 {
		return nil, errors.New("ledger: a transaction needs at least two legs")
	}
	var debit, credit models.Money
	for _, l := range legs {
		if l.Amount <= 0 {
			return nil, fmt.Errorf("ledger: leg amount must be positive, got %d", l.Amount)
		}
		if l.AccountKind == models.LedgerAcctUserWallet && l.UserID == nil {
			return nil, errors.New("ledger: user_wallet leg requires a user id")
		}
		switch l.Direction {
		case models.LedgerDebit:
			debit += l.Amount
		case models.LedgerCredit:
			credit += l.Amount
		default:
			return nil, fmt.Errorf("ledger: invalid direction %q", l.Direction)
		}
	}
	if debit != credit {
		return nil, fmt.Errorf("ledger: unbalanced transaction — debit %d != credit %d", debit, credit)
	}

	// Idempotency: a prior post with this key wins; return it without double-posting.
	var existing models.LedgerTransaction
	err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	// Generate ids in code (not via the DB gen_random_uuid() default) so posting is
	// deterministic across Postgres and the sqlite test harness.
	txn := &models.LedgerTransaction{
		ID:             uuid.New(),
		IdempotencyKey: idempotencyKey,
		Reason:         reason,
		RefType:        refType,
		RefID:          refID,
	}
	for _, l := range legs {
		txn.Entries = append(txn.Entries, models.LedgerEntry{
			ID:            uuid.New(),
			TransactionID: txn.ID,
			AccountKind:   l.AccountKind,
			UserID:        l.UserID,
			Direction:     l.Direction,
			AmountMinor:   l.Amount,
			Currency:      "INR",
		})
	}
	if err := tx.Create(txn).Error; err != nil {
		// A concurrent poster raced us to the unique idempotency_key — re-read theirs.
		var raced models.LedgerTransaction
		if err2 := tx.Where("idempotency_key = ?", idempotencyKey).First(&raced).Error; err2 == nil {
			return &raced, nil
		}
		return nil, err
	}
	return txn, nil
}

// LedgerUserBalance projects a user's spendable wallet balance from the ledger:
// Σcredits − Σdebits over their user_wallet entries. In paise. This is the authoritative
// balance once reads flip; during the shadow phase it is reconciled against the legacy
// float balance.
func LedgerUserBalance(db *gorm.DB, userID uuid.UUID) (models.Money, error) {
	var credit, debit int64
	q := func(dir models.LedgerDirection) *gorm.DB {
		return db.Model(&models.LedgerEntry{}).
			Where("user_id = ? AND account_kind = ? AND direction = ?",
				userID, models.LedgerAcctUserWallet, dir)
	}
	if err := q(models.LedgerCredit).Select("COALESCE(SUM(amount_minor),0)").Scan(&credit).Error; err != nil {
		return 0, err
	}
	if err := q(models.LedgerDebit).Select("COALESCE(SUM(amount_minor),0)").Scan(&debit).Error; err != nil {
		return 0, err
	}
	return models.Money(credit - debit), nil
}

// LedgerTotals returns the whole ledger's debit and credit sums (paise) — must always be
// equal (the core financial-integrity invariant). Used by the reconciliation sweep; a
// mismatch is a SEV-1.
func LedgerTotals(db *gorm.DB) (debit, credit models.Money, err error) {
	var d, c int64
	if err = db.Model(&models.LedgerEntry{}).Where("direction = ?", models.LedgerDebit).
		Select("COALESCE(SUM(amount_minor),0)").Scan(&d).Error; err != nil {
		return 0, 0, err
	}
	if err = db.Model(&models.LedgerEntry{}).Where("direction = ?", models.LedgerCredit).
		Select("COALESCE(SUM(amount_minor),0)").Scan(&c).Error; err != nil {
		return 0, 0, err
	}
	return models.Money(d), models.Money(c), nil
}
