package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/homechef/api/models"
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

// ── RaiseChefRecoveryPenalty / DischargeChefRecovery — the chef-cancel penalty
// (first real writer of this ledger) and its resolving discharge ──

// TestRaiseChefRecoveryPenalty_WritesOneDebitAndIsIdempotent pins the
// chef-cancel penalty writer: exactly one debit.penalty entry per order, and
// a retried call (the loser of a concurrent duplicate CancelOrder request
// still runs the penalty tail, or a genuine re-cancel) never writes a second
// one. After it, ApplyRecoveryDeduction must show the chef owes exactly the
// raised amount.
func TestRaiseChefRecoveryPenalty_WritesOneDebitAndIsIdempotent(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	order := &models.Order{ID: uuid.New(), OrderNumber: "HC-100", ChefID: chefID, ServiceFee: 18.96}
	amountPaise := int64(1_896) // ₹18.96
	reason := "chef cancel — platform fee penalty for order HC-100"

	require.NoError(t, RaiseChefRecoveryPenalty(db, chefID, order, amountPaise, reason))
	require.NoError(t, RaiseChefRecoveryPenalty(db, chefID, order, amountPaise, reason), "a re-cancel/retry must not error")

	var count int64
	require.NoError(t, db.Model(&payouts.LedgerEntry{}).
		Where("payee_id = ? AND kind = ?", chefID, payouts.EntryDebitPenalty).Count(&count).Error)
	require.EqualValues(t, 1, count, "a retried raise must never write a second penalty row")

	_, deducted, err := ApplyRecoveryDeduction(db, chefID, inr(50_000), time.Now())
	require.NoError(t, err)
	require.EqualValues(t, amountPaise, deducted.Minor, "the chef must owe exactly the raised penalty, not double it")
}

// TestRaiseChefRecoveryPenalty_ZeroAmountIsNoOp guards the service layer the
// same way CancelOrder's `order.ServiceFee > 0` gate does at the caller — a
// zero (or negative) amount must never write a row.
func TestRaiseChefRecoveryPenalty_ZeroAmountIsNoOp(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	order := &models.Order{ID: uuid.New(), OrderNumber: "HC-101", ChefID: chefID}

	require.NoError(t, RaiseChefRecoveryPenalty(db, chefID, order, 0, "no fee"))

	var count int64
	require.NoError(t, db.Model(&payouts.LedgerEntry{}).Count(&count).Error)
	require.Zero(t, count)
}

// TestDischargeChefRecovery_WritesOneCreditAndIsIdempotent proves the
// resolving half: after a debt is raised, discharging it nets the recovery
// balance to zero, and a retried discharge call (client-verify + webhook
// racing, or a redelivered webhook) never double-credits.
func TestDischargeChefRecovery_WritesOneCreditAndIsIdempotent(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 1_896)
	settlingOrderID := uuid.New()

	require.NoError(t, DischargeChefRecovery(db, chefID, settlingOrderID, "HC-200", 1_896))
	require.NoError(t, DischargeChefRecovery(db, chefID, settlingOrderID, "HC-200", 1_896))

	var count int64
	require.NoError(t, db.Model(&payouts.LedgerEntry{}).
		Where("payee_id = ? AND kind = ?", chefID, payouts.EntryCreditRecoveryCollected).Count(&count).Error)
	require.EqualValues(t, 1, count, "a retried discharge must never write a second resolving entry")

	var entries []payouts.LedgerEntry
	require.NoError(t, db.Where("payee_id = ?", chefID).Find(&entries).Error)
	balance, err := payouts.DeriveBalance(entries, time.Now())
	require.NoError(t, err)
	require.True(t, balance.Recovery().IsZero(), "the debt must be fully discharged — Recovery() must read zero")
}

// TestDischargeChefRecovery_PartialDischargeLeavesRemainder is the
// partial-collection case: a settling order's gross was smaller than the full
// debt, so only that slice discharges — the rest stays owed for the chef's
// next settling order to collect.
func TestDischargeChefRecovery_PartialDischargeLeavesRemainder(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 5_000) // ₹50.00 owed

	require.NoError(t, DischargeChefRecovery(db, chefID, uuid.New(), "HC-300", 3_000)) // only ₹30.00 collected

	var entries []payouts.LedgerEntry
	require.NoError(t, db.Where("payee_id = ?", chefID).Find(&entries).Error)
	balance, err := payouts.DeriveBalance(entries, time.Now())
	require.NoError(t, err)
	require.EqualValues(t, 2_000, balance.Recovery().Minor, "₹20 must remain owed after only ₹30 of the ₹50 debt was collected")
}

// TestDischargeChefRecovery_ZeroAmountIsNoOp mirrors the raise-side guard.
func TestDischargeChefRecovery_ZeroAmountIsNoOp(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()

	require.NoError(t, DischargeChefRecovery(db, chefID, uuid.New(), "HC-301", 0))

	var count int64
	require.NoError(t, db.Model(&payouts.LedgerEntry{}).Count(&count).Error)
	require.Zero(t, count)
}
