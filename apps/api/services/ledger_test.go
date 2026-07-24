package services

// ledger_test.go — Phase 1 financial-core invariants (docs/wallet-ledger-plan.md).
// The ledger is the source of financial truth, so its guarantees are tested, not assumed:
// double-entry balance, positive amounts, idempotency (exactly-once under retry/concurrency),
// paise-integer money, and the projected user balance = Σcredits − Σdebits.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/models"
)

func setupLedgerCoreDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE ledger_transactions (id TEXT PRIMARY KEY, tenant_id TEXT,
		idempotency_key TEXT UNIQUE NOT NULL, reason TEXT, ref_type TEXT, ref_id TEXT, created_at DATETIME)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE ledger_entries (id TEXT PRIMARY KEY, transaction_id TEXT NOT NULL,
		account_kind TEXT NOT NULL, user_id TEXT, direction TEXT NOT NULL, amount_minor BIGINT NOT NULL,
		currency TEXT NOT NULL DEFAULT 'INR', created_at DATETIME)`).Error)
	return db
}

// creditUserLeg / debitUserLeg build a wallet credit/debit posting: the user_wallet leg +
// the balancing system counterparty leg.
func creditUser(userID uuid.UUID, amount models.Money, counter models.LedgerAccountKind) []LedgerLeg {
	return []LedgerLeg{
		{AccountKind: models.LedgerAcctUserWallet, UserID: &userID, Direction: models.LedgerCredit, Amount: amount},
		{AccountKind: counter, Direction: models.LedgerDebit, Amount: amount},
	}
}
func debitUser(userID uuid.UUID, amount models.Money, counter models.LedgerAccountKind) []LedgerLeg {
	return []LedgerLeg{
		{AccountKind: models.LedgerAcctUserWallet, UserID: &userID, Direction: models.LedgerDebit, Amount: amount},
		{AccountKind: counter, Direction: models.LedgerCredit, Amount: amount},
	}
}

// ₹250.75 must be exactly 25075 paise — never a float.
func TestMoney_PaiseInteger(t *testing.T) {
	require.Equal(t, models.Money(25075), models.RupeesToMoney(250.75))
	require.Equal(t, models.Money(1099), models.RupeesToMoney(10.99))
	require.Equal(t, 250.75, models.Money(25075).Rupees())
	// The classic float trap: 0.1+0.2 in rupees still lands on exact paise.
	require.Equal(t, models.Money(30), models.RupeesToMoney(0.1)+models.RupeesToMoney(0.2))
}

// A balanced credit posts and projects into the user's balance.
func TestPostLedger_CreditProjectsBalance(t *testing.T) {
	db := setupLedgerCoreDB(t)
	u := uuid.New()
	_, err := PostLedgerTransaction(db, "refund:order_1", "refund", "refund", "order_1",
		creditUser(u, 55000, models.LedgerAcctSystemRefund))
	require.NoError(t, err)

	bal, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(55000), bal, "₹550 credited")

	// Whole-ledger invariant: debit == credit.
	d, c, err := LedgerTotals(db)
	require.NoError(t, err)
	require.Equal(t, d, c, "double-entry: Σdebit == Σcredit")
}

// Credits then a debit project the net balance.
func TestPostLedger_CreditThenDebit(t *testing.T) {
	db := setupLedgerCoreDB(t)
	u := uuid.New()
	_, err := PostLedgerTransaction(db, "refund:a", "refund", "refund", "a", creditUser(u, 55000, models.LedgerAcctSystemRefund))
	require.NoError(t, err)
	_, err = PostLedgerTransaction(db, "referral:b", "referral", "referral", "b", creditUser(u, 10000, models.LedgerAcctSystemReferral))
	require.NoError(t, err)
	_, err = PostLedgerTransaction(db, "spend:order_9", "wallet payment", "order", "order_9", debitUser(u, 32000, models.LedgerAcctSystemSpend))
	require.NoError(t, err)

	bal, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(33000), bal, "550 + 100 − 320 = ₹330")
	d, c, _ := LedgerTotals(db)
	require.Equal(t, d, c)
}

// An unbalanced posting is rejected — nothing is written.
func TestPostLedger_UnbalancedRejected(t *testing.T) {
	db := setupLedgerCoreDB(t)
	u := uuid.New()
	legs := []LedgerLeg{
		{AccountKind: models.LedgerAcctUserWallet, UserID: &u, Direction: models.LedgerCredit, Amount: 55000},
		{AccountKind: models.LedgerAcctSystemRefund, Direction: models.LedgerDebit, Amount: 50000}, // 500 != 550
	}
	_, err := PostLedgerTransaction(db, "bad:1", "x", "x", "1", legs)
	require.ErrorContains(t, err, "unbalanced")
	bal, _ := LedgerUserBalance(db, u)
	require.Equal(t, models.Money(0), bal, "nothing posted on an unbalanced transaction")
}

// A non-positive leg amount is rejected.
func TestPostLedger_NonPositiveRejected(t *testing.T) {
	db := setupLedgerCoreDB(t)
	u := uuid.New()
	_, err := PostLedgerTransaction(db, "z", "x", "x", "1", creditUser(u, 0, models.LedgerAcctSystemRefund))
	require.ErrorContains(t, err, "positive")
}

// Idempotency: re-posting the same key credits exactly once (Temporal-retry safety).
func TestPostLedger_Idempotent(t *testing.T) {
	db := setupLedgerCoreDB(t)
	u := uuid.New()
	for i := 0; i < 5; i++ {
		_, err := PostLedgerTransaction(db, "refund:order_1:refund_7", "refund", "refund", "order_1",
			creditUser(u, 50000, models.LedgerAcctSystemRefund))
		require.NoError(t, err)
	}
	bal, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(50000), bal, "5 retries → credited exactly ₹500, not ₹2,500")
}

// A true race (two Creates for the same key) is enforced by the UNIQUE index on
// idempotency_key in Postgres, with PostLedgerTransaction re-reading the winner's row on
// the conflict. sqlite :memory: can't exercise real cross-connection concurrency, so the
// re-read-existing path is covered above by the sequential 5× retry test.
