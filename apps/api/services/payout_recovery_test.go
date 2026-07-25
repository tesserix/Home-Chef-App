package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/payouts"
)

// payout_recovery_test.go — #741 — recovering a chef's debt from their next
// payout.
//
// Conservation is the property that matters: what is paid plus what is
// recovered plus what is carried forward must equal what was owed. Anything
// else quietly creates or destroys money.
//
// Hand-DDL'd payout_ledger_entries (own helper, distinct from
// setupPlatformSettingsDB in premium_pricing_test.go) — payouts.LedgerEntry
// carries a Postgres-only gen_random_uuid() default that sqlite's AutoMigrate
// rejects. A missing/wrong-shaped table would make DeriveBalance silently
// return a zero balance and these tests would pass against broken code, so
// TestApplyRecoveryDeduction_PartialDebtReducesThePayout is the canary: it
// only goes green if the seeded penalty round-trips through the real table.

func newRecoveryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`
		CREATE TABLE payout_ledger_entries (
			id           TEXT PRIMARY KEY,
			tenant_id    TEXT NOT NULL,
			payee_type   TEXT NOT NULL,
			payee_id     TEXT NOT NULL,
			kind         TEXT NOT NULL,
			amount_minor INTEGER NOT NULL,
			currency     TEXT NOT NULL,
			source_type  TEXT NOT NULL,
			source_id    TEXT NOT NULL,
			matures_at   DATETIME,
			batch_id     TEXT,
			actor_id     TEXT,
			reason       TEXT,
			created_at   DATETIME
		)
	`).Error)
	return db
}

func inr(minor int64) payouts.Money {
	return payouts.Money{Minor: minor, Currency: payouts.CurrencyINR}
}

func seedPenalty(t *testing.T, db *gorm.DB, chefID uuid.UUID, minor int64) {
	t.Helper()
	entry := payouts.LedgerEntry{
		ID: uuid.New(), TenantID: "t1",
		PayeeType: payouts.PayeeChef, PayeeID: chefID,
		Kind: payouts.EntryDebitPenalty, AmountMinor: minor,
		Currency:   payouts.CurrencyINR,
		SourceType: "order_issue", SourceID: uuid.NewString(),
	}
	if err := db.Create(&entry).Error; err != nil {
		t.Fatalf("seed penalty: %v", err)
	}
}

func TestApplyRecoveryDeduction_NoDebtPaysInFull(t *testing.T) {
	db := newRecoveryTestDB(t)
	net, deducted, err := ApplyRecoveryDeduction(db, uuid.New(), inr(50_000), time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor != 50_000 {
		t.Fatalf("net = %d, want 50000", net.Minor)
	}
	if !deducted.IsZero() {
		t.Fatalf("deducted = %v, want zero", deducted)
	}
}

func TestApplyRecoveryDeduction_PartialDebtReducesThePayout(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 15_000)

	net, deducted, err := ApplyRecoveryDeduction(db, chefID, inr(50_000), time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor != 35_000 {
		t.Fatalf("net = %d, want 35000", net.Minor)
	}
	if deducted.Minor != 15_000 {
		t.Fatalf("deducted = %d, want 15000", deducted.Minor)
	}
}

func TestApplyRecoveryDeduction_DebtLargerThanPayoutFloorsAtZero(t *testing.T) {
	// Never emit a negative transfer. The remainder stays owed.
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 80_000)

	net, deducted, err := ApplyRecoveryDeduction(db, chefID, inr(50_000), time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor != 0 {
		t.Fatalf("net = %d, want 0 — never a negative transfer", net.Minor)
	}
	if deducted.Minor != 50_000 {
		t.Fatalf("deducted = %d, want the whole payout", deducted.Minor)
	}
}

func TestApplyRecoveryDeduction_Conserves(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 20_000)

	gross := inr(50_000)
	net, deducted, err := ApplyRecoveryDeduction(db, chefID, gross, time.Now())
	if err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}
	if net.Minor+deducted.Minor != gross.Minor {
		t.Fatalf("conservation broken: net %d + deducted %d != gross %d",
			net.Minor, deducted.Minor, gross.Minor)
	}
}
