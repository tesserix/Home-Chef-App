package services

import (
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// ErrInsufficientWalletBalance is returned when a debit exceeds the balance.
var ErrInsufficientWalletBalance = errors.New("insufficient wallet balance")

// GetOrCreateWallet returns the user's wallet, creating an empty INR one on
// first touch. Safe to call on a read path.
func GetOrCreateWallet(db *gorm.DB, userID uuid.UUID) (*models.Wallet, error) {
	var w models.Wallet
	// Set ID/Currency on the create path explicitly rather than relying on the
	// Postgres gen_random_uuid() column default — that keeps the ledger portable
	// (and lets the sqlite-backed unit tests create wallets).
	err := db.Where(models.Wallet{UserID: userID}).
		Attrs(models.Wallet{ID: uuid.New(), Currency: "INR"}).
		FirstOrCreate(&w).Error
	if err != nil {
		return nil, err
	}
	return &w, nil
}

// CreditWallet adds store credit (refund-to-wallet, referral, promo, cashback,
// admin top-up). Amount must be positive. Idempotent on idempotencyKey: a repeat
// call with the same key returns the original entry without double-crediting.
func CreditWallet(db *gorm.DB, userID uuid.UUID, amount float64, source models.WalletTxnSource, orderID *uuid.UUID, reason, idempotencyKey string, createdBy *uuid.UUID) (*models.WalletTxn, error) {
	return applyWalletTxn(db, userID, amount, models.WalletCredit, source, orderID, reason, idempotencyKey, createdBy)
}

// DebitWallet removes store credit (applied at checkout, admin clawback). Amount
// must be positive and must not exceed the balance (returns
// ErrInsufficientWalletBalance otherwise). Idempotent on idempotencyKey.
func DebitWallet(db *gorm.DB, userID uuid.UUID, amount float64, source models.WalletTxnSource, orderID *uuid.UUID, reason, idempotencyKey string, createdBy *uuid.UUID) (*models.WalletTxn, error) {
	return applyWalletTxn(db, userID, amount, models.WalletDebit, source, orderID, reason, idempotencyKey, createdBy)
}

// applyWalletTxn is the single mutate path for the ledger. It runs in a DB
// transaction, locks the wallet row (FOR UPDATE on Postgres; whole-DB lock on
// the sqlite test driver) to prevent concurrent double-spend, writes the
// immutable ledger entry, and updates the cached balance — all atomically. The
// idempotency check inside the transaction (plus the unique index on
// idempotency_key) guarantees a retried event is a no-op.
func applyWalletTxn(db *gorm.DB, userID uuid.UUID, amount float64, txnType models.WalletTxnType, source models.WalletTxnSource, orderID *uuid.UUID, reason, idempotencyKey string, createdBy *uuid.UUID) (*models.WalletTxn, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("wallet amount must be positive, got %v", amount)
	}
	// Every entry carries a non-empty unique key. Callers pass a semantic key
	// (e.g. "refund:<orderID>") for dedup; absent one we mint a UUID so the
	// unique index never collides on empty strings.
	if idempotencyKey == "" {
		idempotencyKey = "wtx:" + uuid.NewString()
	}

	var result *models.WalletTxn
	err := db.Transaction(func(tx *gorm.DB) error {
		// Idempotency: if this logical event already landed, return it as-is.
		var existing models.WalletTxn
		err := tx.Where("idempotency_key = ?", idempotencyKey).First(&existing).Error
		if err == nil {
			result = &existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		w, err := GetOrCreateWallet(tx, userID)
		if err != nil {
			return err
		}
		// Re-read under a row lock so a concurrent txn can't race the balance.
		// FOR UPDATE is Postgres (prod); the sqlite test driver serializes
		// writes via its global lock and rejects the clause, so we skip it there.
		reread := tx
		if tx.Dialector.Name() == "postgres" {
			reread = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := reread.First(w, "id = ?", w.ID).Error; err != nil {
			return err
		}

		newBalance := w.Balance
		if txnType == models.WalletCredit {
			newBalance += amount
		} else {
			if w.Balance < amount {
				return ErrInsufficientWalletBalance
			}
			newBalance -= amount
		}

		entry := &models.WalletTxn{
			ID:             uuid.New(),
			WalletID:       w.ID,
			UserID:         userID,
			Type:           txnType,
			Source:         source,
			Amount:         amount,
			BalanceAfter:   newBalance,
			Currency:       w.Currency,
			OrderID:        orderID,
			Reason:         reason,
			CreatedBy:      createdBy,
			IdempotencyKey: idempotencyKey,
		}
		if err := tx.Create(entry).Error; err != nil {
			return err
		}
		if err := tx.Model(w).Update("balance", newBalance).Error; err != nil {
			return err
		}
		result = entry
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// WalletBalance returns the current balance + currency for a user, creating an
// empty wallet if none exists.
func WalletBalance(db *gorm.DB, userID uuid.UUID) (*models.Wallet, error) {
	return GetOrCreateWallet(db, userID)
}

// ListWalletTxns returns a page of a user's ledger entries, newest first.
func ListWalletTxns(db *gorm.DB, userID uuid.UUID, limit, offset int) ([]models.WalletTxn, int64, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var txns []models.WalletTxn
	var total int64
	if err := db.Model(&models.WalletTxn{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := db.Where("user_id = ?", userID).
		Order("created_at DESC").
		Limit(limit).Offset(offset).
		Find(&txns).Error; err != nil {
		return nil, 0, err
	}
	return txns, total, nil
}
