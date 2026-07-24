package services

// ledger_holds.go — Phase 3 of the wallet→ledger evolution (docs/wallet-ledger-plan.md).
// A generic wallet HOLD: reserve part of a customer's available balance for an in-flight order
// or payment, then either CAPTURE it (the money is spent) or RELEASE it (return it, unspent, to
// the exact buckets it came from). This is the primitive checkout and the Phase 5 payment
// workflow build on so a dropped/failed payment never strands the customer's money.
//
// Expressed entirely in the double-entry ledger — no separate holds table:
//   - PLACE   : debit the spendable buckets (spend priority), credit user_wallet_held. Total
//               unchanged; value just moves available → held.
//   - CAPTURE : debit user_wallet_held, credit system_spend. The reserved value leaves the user.
//   - RELEASE : debit user_wallet_held, credit back the exact buckets the placement drained.
//
// A hold is identified by (refType, refID) — e.g. ("order", "<uuid>") — reusing the caller's
// per-order idempotency. Place/capture/release each post at most once (idempotency key), and a
// hold can be captured XOR released, never both. Fund checks are against the AVAILABLE balance
// under a per-user row lock, so two concurrent holds can't both reserve the same rupees.

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

var (
	// ErrHoldNotFound is returned when capturing/releasing a hold that was never placed.
	ErrHoldNotFound = errors.New("wallet hold not found")
	// ErrHoldAlreadyCaptured / ErrHoldAlreadyReleased enforce the capture-XOR-release machine.
	ErrHoldAlreadyCaptured = errors.New("wallet hold already captured")
	ErrHoldAlreadyReleased = errors.New("wallet hold already released")
)

// Ledger ref-types (LedgerTransaction.RefType) marking the three hold operations.
const (
	holdRefTypePlace   = "wallet_hold"
	holdRefTypeCapture = "wallet_hold_capture"
	holdRefTypeRelease = "wallet_hold_release"
)

// holdRef is the stable identity of a hold within the ledger: "<refType>:<refID>".
func holdRef(refType, refID string) string { return refType + ":" + refID }

// lockUserWallet serializes concurrent hold/spend operations for one user by taking a row lock
// on their wallet row (FOR UPDATE on Postgres; the sqlite test driver serializes writes itself).
// The wallet row is the per-user mutex shared with the legacy debit path, so a legacy debit and
// a ledger hold on the same user cannot interleave during the migration.
func lockUserWallet(tx *gorm.DB, userID uuid.UUID) error {
	w, err := GetOrCreateWallet(tx, userID)
	if err != nil {
		return err
	}
	if tx.Dialector.Name() != "postgres" {
		return nil
	}
	var locked models.Wallet
	return tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, "id = ?", w.ID).Error
}

// ledgerTxnExists reports whether a ledger transaction with the given ref-type + ref-id exists.
func ledgerTxnExists(tx *gorm.DB, refType, refID string) (bool, error) {
	var n int64
	if err := tx.Model(&models.LedgerTransaction{}).
		Where("ref_type = ? AND ref_id = ?", refType, refID).Count(&n).Error; err != nil {
		return false, err
	}
	return n > 0, nil
}

// loadHoldPlacement loads the PLACE transaction (with its legs) for a hold, or ErrHoldNotFound.
func loadHoldPlacement(tx *gorm.DB, ref string) (*models.LedgerTransaction, error) {
	var place models.LedgerTransaction
	err := tx.Preload("Entries").
		Where("ref_type = ? AND ref_id = ?", holdRefTypePlace, ref).First(&place).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrHoldNotFound
	}
	if err != nil {
		return nil, err
	}
	return &place, nil
}

// heldLeg returns the reserved amount and user id from a placement's held-credit leg.
func heldLeg(place *models.LedgerTransaction) (models.Money, uuid.UUID) {
	for _, e := range place.Entries {
		if e.AccountKind == models.LedgerAcctUserHeld && e.Direction == models.LedgerCredit {
			if e.UserID != nil {
				return e.AmountMinor, *e.UserID
			}
			return e.AmountMinor, uuid.Nil
		}
	}
	return 0, uuid.Nil
}

// PlaceWalletHold reserves `amount` of the user's AVAILABLE balance for (refType, refID), moving
// it from the spendable buckets (in spend priority) into the held account. Idempotent: re-placing
// the same (refType, refID) returns the existing hold without reserving again. Returns
// ErrInsufficientWalletBalance when available < amount.
func PlaceWalletHold(db *gorm.DB, userID uuid.UUID, amount models.Money, refType, refID string) (*models.LedgerTransaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("wallet hold amount must be positive, got %d", amount)
	}
	ref := holdRef(refType, refID)
	var result *models.LedgerTransaction
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := lockUserWallet(tx, userID); err != nil {
			return err
		}
		// Idempotent: an existing placement wins (no double-reserve).
		if existing, err := loadHoldPlacement(tx, ref); err == nil {
			result = existing
			return nil
		} else if !errors.Is(err, ErrHoldNotFound) {
			return err
		}
		// Fund check against AVAILABLE (never total): can't reserve already-held money twice.
		avail, err := LedgerUserAvailableBalance(tx, userID)
		if err != nil {
			return err
		}
		if avail < amount {
			return ErrInsufficientWalletBalance
		}
		// Drain the spendable buckets in priority; guaranteed to fully cover (avail >= amount).
		debitLegs, err := userWalletDebitLegs(tx, userID, amount)
		if err != nil {
			return err
		}
		uid := userID
		legs := append(debitLegs, LedgerLeg{
			AccountKind: models.LedgerAcctUserHeld, UserID: &uid,
			Direction: models.LedgerCredit, Amount: amount,
		})
		txn, err := PostLedgerTransaction(tx, "wallet-hold:"+ref, "wallet hold", holdRefTypePlace, ref, legs)
		if err != nil {
			return err
		}
		result = txn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CaptureWalletHold spends a placed hold: the reserved value leaves the user (held → system_spend).
// Idempotent. ErrHoldNotFound if never placed; ErrHoldAlreadyReleased if it was released.
func CaptureWalletHold(db *gorm.DB, refType, refID string) (*models.LedgerTransaction, error) {
	ref := holdRef(refType, refID)
	var result *models.LedgerTransaction
	err := db.Transaction(func(tx *gorm.DB) error {
		place, err := loadHoldPlacement(tx, ref)
		if err != nil {
			return err
		}
		amount, uid := heldLeg(place)
		if err := lockUserWallet(tx, uid); err != nil {
			return err
		}
		released, err := ledgerTxnExists(tx, holdRefTypeRelease, ref)
		if err != nil {
			return err
		}
		if released {
			return ErrHoldAlreadyReleased
		}
		legs := []LedgerLeg{
			{AccountKind: models.LedgerAcctUserHeld, UserID: &uid, Direction: models.LedgerDebit, Amount: amount},
			{AccountKind: models.LedgerAcctSystemSpend, Direction: models.LedgerCredit, Amount: amount},
		}
		txn, err := PostLedgerTransaction(tx, "wallet-hold-capture:"+ref, "wallet hold capture", holdRefTypeCapture, ref, legs)
		if err != nil {
			return err
		}
		result = txn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ReleaseWalletHold returns a placed hold to the customer, unspent, restoring value to the exact
// buckets it was drawn from (held → original buckets). Idempotent. ErrHoldAlreadyCaptured if spent.
func ReleaseWalletHold(db *gorm.DB, refType, refID string) (*models.LedgerTransaction, error) {
	ref := holdRef(refType, refID)
	var result *models.LedgerTransaction
	err := db.Transaction(func(tx *gorm.DB) error {
		place, err := loadHoldPlacement(tx, ref)
		if err != nil {
			return err
		}
		amount, uid := heldLeg(place)
		if err := lockUserWallet(tx, uid); err != nil {
			return err
		}
		captured, err := ledgerTxnExists(tx, holdRefTypeCapture, ref)
		if err != nil {
			return err
		}
		if captured {
			return ErrHoldAlreadyCaptured
		}
		// Reverse the placement: one held debit, plus restore each bucket the placement drained
		// (source-preserving — a released promo hold returns to promo, not to a generic bucket).
		legs := []LedgerLeg{
			{AccountKind: models.LedgerAcctUserHeld, UserID: &uid, Direction: models.LedgerDebit, Amount: amount},
		}
		for _, e := range place.Entries {
			if e.AccountKind == models.LedgerAcctUserHeld {
				continue // the held credit is reversed by the single held-debit leg above
			}
			euid := uid
			legs = append(legs, LedgerLeg{
				AccountKind: e.AccountKind, UserID: &euid,
				Direction: models.LedgerCredit, Amount: e.AmountMinor,
			})
		}
		txn, err := PostLedgerTransaction(tx, "wallet-hold-release:"+ref, "wallet hold release", holdRefTypeRelease, ref, legs)
		if err != nil {
			return err
		}
		result = txn
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
