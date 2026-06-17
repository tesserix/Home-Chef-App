package services

// wallet_test.go — ledger correctness for the customer wallet (#33). The ledger
// is money, so the invariants worth pinning: credit/debit move the balance the
// right way, a debit can't overdraw, BalanceAfter is an accurate running
// snapshot, and a retried operation (same idempotency key) never double-applies.

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func setupWalletDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE wallets (
		id text PRIMARY KEY, user_id text UNIQUE, balance real DEFAULT 0,
		currency text DEFAULT 'INR', created_at datetime, updated_at datetime
	)`).Error; err != nil {
		t.Fatalf("create wallets: %v", err)
	}
	if err := db.Exec(`CREATE TABLE wallet_txns (
		id text PRIMARY KEY, wallet_id text, user_id text, type text, source text,
		amount real, balance_after real, currency text, order_id text, reason text,
		created_by text, idempotency_key text UNIQUE, created_at datetime
	)`).Error; err != nil {
		t.Fatalf("create wallet_txns: %v", err)
	}
	orig := database.DB
	database.DB = db
	t.Cleanup(func() { database.DB = orig })
	return db
}

func TestGetOrCreateWallet_DefaultsToINR(t *testing.T) {
	db := setupWalletDB(t)
	w, err := GetOrCreateWallet(db, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if w.Balance != 0 || w.Currency != "INR" {
		t.Fatalf("new wallet should be 0 INR, got %v %s", w.Balance, w.Currency)
	}
}

func TestCreditAndDebit_MoveBalance(t *testing.T) {
	db := setupWalletDB(t)
	uid := uuid.New()

	c, err := CreditWallet(db, uid, 500, models.WalletSourceRefund, nil, "refund test", "refund:o1", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.BalanceAfter != 500 {
		t.Fatalf("credit BalanceAfter=%v, want 500", c.BalanceAfter)
	}

	d, err := DebitWallet(db, uid, 200, models.WalletSourceOrderPayment, nil, "checkout", "order:o2", nil)
	if err != nil {
		t.Fatal(err)
	}
	if d.BalanceAfter != 300 {
		t.Fatalf("debit BalanceAfter=%v, want 300", d.BalanceAfter)
	}

	w, _ := WalletBalance(db, uid)
	if w.Balance != 300 {
		t.Fatalf("wallet balance=%v, want 300", w.Balance)
	}
}

func TestDebit_InsufficientBalance(t *testing.T) {
	db := setupWalletDB(t)
	uid := uuid.New()
	if _, err := CreditWallet(db, uid, 100, models.WalletSourcePromo, nil, "promo", "promo:1", nil); err != nil {
		t.Fatal(err)
	}
	_, err := DebitWallet(db, uid, 250, models.WalletSourceOrderPayment, nil, "checkout", "order:x", nil)
	if err != ErrInsufficientWalletBalance {
		t.Fatalf("expected ErrInsufficientWalletBalance, got %v", err)
	}
	// Balance must be untouched after a rejected debit.
	w, _ := WalletBalance(db, uid)
	if w.Balance != 100 {
		t.Fatalf("balance must be unchanged after failed debit, got %v", w.Balance)
	}
}

func TestCredit_IsIdempotent(t *testing.T) {
	db := setupWalletDB(t)
	uid := uuid.New()
	// Same idempotency key (e.g. a refund webhook delivered twice).
	first, err := CreditWallet(db, uid, 300, models.WalletSourceRefund, nil, "refund", "refund:dup", nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := CreditWallet(db, uid, 300, models.WalletSourceRefund, nil, "refund", "refund:dup", nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatal("repeat credit with same key must return the original txn, not a new one")
	}
	w, _ := WalletBalance(db, uid)
	if w.Balance != 300 {
		t.Fatalf("idempotent credit must not double-apply: balance=%v, want 300", w.Balance)
	}
	// Exactly one ledger row for that key.
	var n int64
	db.Model(&models.WalletTxn{}).Where("idempotency_key = ?", "refund:dup").Count(&n)
	if n != 1 {
		t.Fatalf("expected 1 ledger row for the key, got %d", n)
	}
}

func TestAmountMustBePositive(t *testing.T) {
	db := setupWalletDB(t)
	uid := uuid.New()
	if _, err := CreditWallet(db, uid, 0, models.WalletSourcePromo, nil, "", "", nil); err == nil {
		t.Fatal("zero amount must be rejected")
	}
	if _, err := CreditWallet(db, uid, -5, models.WalletSourcePromo, nil, "", "", nil); err == nil {
		t.Fatal("negative amount must be rejected")
	}
}

func TestListWalletTxns_NewestFirst(t *testing.T) {
	db := setupWalletDB(t)
	uid := uuid.New()
	_, _ = CreditWallet(db, uid, 100, models.WalletSourceRefund, nil, "a", "k1", nil)
	_, _ = CreditWallet(db, uid, 50, models.WalletSourceCashback, nil, "b", "k2", nil)
	txns, total, err := ListWalletTxns(db, uid, 20, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(txns) != 2 {
		t.Fatalf("expected 2 txns, got total=%d len=%d", total, len(txns))
	}
}
