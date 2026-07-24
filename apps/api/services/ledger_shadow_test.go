package services

// ledger_shadow_test.go — Phase 1 shadow-layer proof (docs/wallet-ledger-plan.md).
// Dual-write must make the ledger balance track the legacy float balance exactly; the
// backfill must seed openings so a pre-existing wallet reconciles; and reconcile must
// surface any gap without touching it.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/homechef/api/config"
	"github.com/homechef/api/models"
)

func setupShadowDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: gormlogger.Default.LogMode(gormlogger.Silent)})
	require.NoError(t, err)
	for _, s := range []string{
		`CREATE TABLE wallets (id TEXT PRIMARY KEY, user_id TEXT UNIQUE, balance REAL DEFAULT 0,
			currency TEXT DEFAULT 'INR', created_at DATETIME, updated_at DATETIME)`,
		`CREATE TABLE wallet_txns (id TEXT PRIMARY KEY, wallet_id TEXT, user_id TEXT, type TEXT, source TEXT,
			amount REAL, balance_after REAL, currency TEXT, order_id TEXT, reason TEXT, created_by TEXT,
			idempotency_key TEXT UNIQUE, created_at DATETIME)`,
		`CREATE TABLE ledger_transactions (id TEXT PRIMARY KEY, tenant_id TEXT, idempotency_key TEXT UNIQUE NOT NULL,
			reason TEXT, ref_type TEXT, ref_id TEXT, created_at DATETIME)`,
		`CREATE TABLE ledger_entries (id TEXT PRIMARY KEY, transaction_id TEXT NOT NULL, account_kind TEXT NOT NULL,
			user_id TEXT, direction TEXT NOT NULL, amount_minor BIGINT NOT NULL, currency TEXT NOT NULL DEFAULT 'INR',
			created_at DATETIME)`,
	} {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func ledgerShadow(t *testing.T, on bool) {
	t.Helper()
	prev := config.AppConfig
	config.AppConfig = &config.Config{LedgerShadowEnabled: on}
	t.Cleanup(func() { config.AppConfig = prev })
}

// With shadow ON, credits/debits mirror into the ledger and the projected ledger balance
// equals the legacy float balance, to the paise.
func TestLedgerShadow_DualWriteTracksBalance(t *testing.T) {
	ledgerShadow(t, true)
	db := setupShadowDB(t)
	u := uuid.New()

	_, err := CreditWallet(db, u, 550, models.WalletSourceRefund, nil, "refund", "refund:o1", nil)
	require.NoError(t, err)
	_, err = CreditWallet(db, u, 100, models.WalletSourceReferral, nil, "referral", "referral:1", nil)
	require.NoError(t, err)
	_, err = DebitWallet(db, u, 320, models.WalletSourceOrderPayment, nil, "order", "spend:o9", nil)
	require.NoError(t, err)

	var w models.Wallet
	require.NoError(t, db.First(&w, "user_id = ?", u).Error)
	require.Equal(t, 330.0, w.Balance, "legacy: 550 + 100 − 320")

	lb, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.RupeesToMoney(w.Balance), lb, "ledger tracks legacy to the paise")
	require.Equal(t, models.Money(33000), lb)

	d, c, err := LedgerTotals(db)
	require.NoError(t, err)
	require.Equal(t, d, c, "double-entry invariant holds")

	// Reconcile clean.
	drift, err := ReconcileLedgerVsWallet(db)
	require.NoError(t, err)
	require.Empty(t, drift, "no drift while dual-writing")
}

// With shadow OFF, no ledger rows are written.
func TestLedgerShadow_Off_NoLedgerWrite(t *testing.T) {
	ledgerShadow(t, false)
	db := setupShadowDB(t)
	u := uuid.New()
	_, err := CreditWallet(db, u, 550, models.WalletSourceRefund, nil, "refund", "refund:o1", nil)
	require.NoError(t, err)

	lb, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(0), lb, "shadow off → nothing mirrored")
}

// A pre-existing wallet (balance but no ledger) drifts, backfill seeds an opening that
// reconciles it, and backfill is idempotent.
func TestLedgerBackfillAndReconcile(t *testing.T) {
	db := setupShadowDB(t)
	u := uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO wallets (id, user_id, balance, currency) VALUES (?,?,?,?)`,
		uuid.NewString(), u.String(), 725.50, "INR").Error)

	// Before backfill: ledger is empty → drift.
	drift, err := ReconcileLedgerVsWallet(db)
	require.NoError(t, err)
	require.Len(t, drift, 1)
	require.Equal(t, models.Money(72550), drift[0].LegacyMinor)
	require.Equal(t, models.Money(0), drift[0].LedgerMinor)

	// Backfill seeds the opening.
	n, err := BackfillLedgerOpeningBalances(db)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	lb, err := LedgerUserBalance(db, u)
	require.NoError(t, err)
	require.Equal(t, models.Money(72550), lb, "ledger now equals legacy")

	drift, err = ReconcileLedgerVsWallet(db)
	require.NoError(t, err)
	require.Empty(t, drift, "reconciled after backfill")

	// Idempotent: a second backfill posts nothing.
	n, err = BackfillLedgerOpeningBalances(db)
	require.NoError(t, err)
	require.Equal(t, 0, n)
}
