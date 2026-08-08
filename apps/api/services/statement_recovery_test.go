package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// statement_recovery_test.go — #1092. A chef whose transfer was reduced to pay
// down a debt got a statement that did not mention it, so the money that
// arrived did not match the money the statement promised. Recovery gets its own
// line, exactly as penalties and bonuses already do.
//
// The property under test is that a debt is discharged ONCE. The per-order site
// (CollectRecoveryDeduction from order settlement) and this one share a ledger,
// so whichever collects first leaves nothing for the other.

func newStatementRecoveryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := newRecoveryTestDB(t)
	// Hand-DDL'd for the same reason payout_ledger_entries is: the model carries
	// a Postgres-only gen_random_uuid() default that sqlite's AutoMigrate rejects.
	require.NoError(t, db.Exec(`CREATE TABLE weekly_statements (
		id TEXT PRIMARY KEY, chef_id TEXT, user_id TEXT, week_start DATETIME, week_end DATETIME,
		currency TEXT, orders_count INTEGER, gross_revenue REAL, platform_commission REAL,
		cgst REAL, sgst REAL, igst REAL, tds REAL, penalty_deductions REAL, bonus_additions REAL DEFAULT 0,
		recovery_deductions REAL DEFAULT 0, net_payout REAL, status TEXT, paid_at DATETIME,
		payout_ref TEXT, created_at DATETIME)`).Error)
	return db
}

func seedStatement(t *testing.T, db *gorm.DB, chefID uuid.UUID, netPayout float64) *models.WeeklyStatement {
	t.Helper()
	stmt := &models.WeeklyStatement{
		ID: uuid.New(), ChefID: chefID, UserID: uuid.New(),
		WeekStart: time.Now().UTC().AddDate(0, 0, -7), WeekEnd: time.Now().UTC(),
		Currency: "INR", NetPayout: netPayout,
	}
	require.NoError(t, db.Create(stmt).Error)
	return stmt
}

func TestApplyChefRecoveryToStatement_NoDebtChangesNothing(t *testing.T) {
	db := newStatementRecoveryDB(t)
	stmt := seedStatement(t, db, uuid.New(), 4193.62)

	collected, err := ApplyChefRecoveryToStatement(db, stmt)

	require.NoError(t, err)
	require.Zero(t, collected)
	require.Equal(t, 4193.62, stmt.NetPayout)
	require.Zero(t, stmt.RecoveryDeductions)
}

func TestApplyChefRecoveryToStatement_CollectsOntoItsOwnLine(t *testing.T) {
	db := newStatementRecoveryDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 15_000) // ₹150
	stmt := seedStatement(t, db, chefID, 1000)

	collected, err := ApplyChefRecoveryToStatement(db, stmt)

	require.NoError(t, err)
	require.Equal(t, 150.0, collected)
	require.Equal(t, 150.0, stmt.RecoveryDeductions)
	require.Equal(t, 850.0, stmt.NetPayout, "net payout is what the chef will actually be paid")

	var stored models.WeeklyStatement
	require.NoError(t, db.First(&stored, "id = ?", stmt.ID).Error)
	require.Equal(t, 150.0, stored.RecoveryDeductions)
	require.Equal(t, 850.0, stored.NetPayout)
}

func TestApplyChefRecoveryToStatement_DebtLargerThanPayoutFloorsAtZero(t *testing.T) {
	db := newStatementRecoveryDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 150_000) // ₹1500 owed against a ₹1000 week
	stmt := seedStatement(t, db, chefID, 1000)

	collected, err := ApplyChefRecoveryToStatement(db, stmt)

	require.NoError(t, err)
	require.Equal(t, 1000.0, collected)
	require.Equal(t, 0.0, stmt.NetPayout, "never promise a negative payout")

	// The remainder stays owed and is collected from the next settlement.
	owed := remainingRecovery(t, db, chefID)
	require.Equal(t, int64(50_000), owed.Minor)
}

func TestApplyChefRecoveryToStatement_DischargesTheDebtOnlyOnce(t *testing.T) {
	db := newStatementRecoveryDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 15_000)
	stmt := seedStatement(t, db, chefID, 1000)

	first, err := ApplyChefRecoveryToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 150.0, first)

	// A re-run of statement generation reports the same line — it is this
	// statement's recovery, not a fresh collection — and must not move the money.
	second, err := ApplyChefRecoveryToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 150.0, second)
	require.Equal(t, 850.0, stmt.NetPayout, "a re-run must not deduct the debt a second time")
	require.Equal(t, 150.0, stmt.RecoveryDeductions)

	var stored models.WeeklyStatement
	require.NoError(t, db.First(&stored, "id = ?", stmt.ID).Error)
	require.Equal(t, 850.0, stored.NetPayout)
	require.Equal(t, 150.0, stored.RecoveryDeductions)
}

func TestApplyChefRecoveryToStatement_LeavesNothingForTheOrderSite(t *testing.T) {
	// One debt, two collection sites. Whichever runs first takes it; the other
	// must find nothing — otherwise the chef pays the same debt twice.
	db := newStatementRecoveryDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 15_000)
	stmt := seedStatement(t, db, chefID, 1000)

	collected, err := ApplyChefRecoveryToStatement(db, stmt)
	require.NoError(t, err)
	require.Equal(t, 150.0, collected)

	net, deducted, err := CollectRecoveryDeduction(db, chefID, inr(50_000), "order", uuid.NewString(), time.Now())
	require.NoError(t, err)
	require.True(t, deducted.IsZero(), "the statement already discharged it")
	require.Equal(t, int64(50_000), net.Minor)
}

func remainingRecovery(t *testing.T, db *gorm.DB, chefID uuid.UUID) payouts.Money {
	t.Helper()
	var entries []payouts.LedgerEntry
	require.NoError(t, db.Where("payee_type = ? AND payee_id = ?", payouts.PayeeChef, chefID).Find(&entries).Error)
	balance, err := payouts.DeriveBalance(entries, time.Now())
	require.NoError(t, err)
	return balance.Recovery()
}
