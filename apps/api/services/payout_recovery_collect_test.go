package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// payout_recovery_collect_test.go — #1079 — collecting a recovery debt must
// discharge it.
//
// The property under test is that a debt is collected exactly once. Before
// this, ApplyRecoveryDeduction only read the ledger: the same full debt was
// re-derived on every call, so a chef was charged it again on every subsequent
// payout, forever.

func recoveryOwed(t *testing.T, db *gorm.DB, chefID uuid.UUID) int64 {
	t.Helper()
	var entries []payouts.LedgerEntry
	if err := db.Where("payee_type = ? AND payee_id = ?", payouts.PayeeChef, chefID).
		Find(&entries).Error; err != nil {
		t.Fatalf("read ledger: %v", err)
	}
	balance, err := payouts.DeriveBalance(entries, time.Now())
	if err != nil {
		t.Fatalf("derive balance: %v", err)
	}
	return balance.Recovery().Minor
}

func TestCollectRecoveryDeductionDischargesWhatItCollects(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 50_000)

	net, deducted, err := CollectRecoveryDeduction(db, chefID, inr(20_000), "order", uuid.NewString(), time.Now())
	if err != nil {
		t.Fatalf("CollectRecoveryDeduction: %v", err)
	}

	if net.Minor != 0 || deducted.Minor != 20_000 {
		t.Fatalf("net = %d, deducted = %d; want 0 and 20000", net.Minor, deducted.Minor)
	}
	if owed := recoveryOwed(t, db, chefID); owed != 30_000 {
		t.Errorf("outstanding = %d, want 30000 — the collected 20000 must be discharged", owed)
	}
}

// The AC from #1079: ₹500 debt against a ₹200 payout then a ₹300 one settles
// the debt exactly, and a third payout is untouched.
func TestCollectRecoveryDeductionAcrossSuccessivePayoutsSettlesExactly(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 50_000)

	collected := int64(0)
	for _, gross := range []int64{20_000, 30_000, 40_000} {
		_, deducted, err := CollectRecoveryDeduction(db, chefID, inr(gross), "order", uuid.NewString(), time.Now())
		if err != nil {
			t.Fatalf("CollectRecoveryDeduction: %v", err)
		}
		collected += deducted.Minor
	}

	if collected != 50_000 {
		t.Errorf("collected %d across three payouts, want exactly the 50000 debt", collected)
	}
	if owed := recoveryOwed(t, db, chefID); owed != 0 {
		t.Errorf("outstanding = %d, want 0 once the debt is settled", owed)
	}
}

// A retried disbursement re-runs collection for the SAME payout. It must not
// discharge twice — that would forgive a debt the chef never paid.
func TestCollectRecoveryDeductionIsIdempotentPerSource(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 50_000)
	sourceID := uuid.NewString()

	first, firstDeducted, err := CollectRecoveryDeduction(db, chefID, inr(20_000), "order", sourceID, time.Now())
	if err != nil {
		t.Fatalf("first collect: %v", err)
	}
	second, secondDeducted, err := CollectRecoveryDeduction(db, chefID, inr(20_000), "order", sourceID, time.Now())
	if err != nil {
		t.Fatalf("retried collect: %v", err)
	}

	if first.Minor != second.Minor || firstDeducted.Minor != secondDeducted.Minor {
		t.Errorf("retry returned (net %d, deducted %d), want the first answer (net %d, deducted %d)",
			second.Minor, secondDeducted.Minor, first.Minor, firstDeducted.Minor)
	}
	if owed := recoveryOwed(t, db, chefID); owed != 30_000 {
		t.Errorf("outstanding = %d, want 30000 — a retry must not discharge twice", owed)
	}

	var rows int64
	if err := db.Model(&payouts.LedgerEntry{}).
		Where("payee_id = ? AND kind = ?", chefID, payouts.EntryCreditRecoveryCollected).
		Count(&rows).Error; err != nil {
		t.Fatalf("count entries: %v", err)
	}
	if rows != 1 {
		t.Errorf("wrote %d resolving entries for one payout, want 1", rows)
	}
	// What the retry REPORTS collecting must be what the ledger records, or the
	// caller withholds money nothing discharged.
	var discharged int64
	if err := db.Model(&payouts.LedgerEntry{}).
		Where("payee_id = ? AND kind = ?", chefID, payouts.EntryCreditRecoveryCollected).
		Select("COALESCE(SUM(amount_minor), 0)").Scan(&discharged).Error; err != nil {
		t.Fatalf("sum entries: %v", err)
	}
	if discharged != secondDeducted.Minor {
		t.Errorf("retry reported collecting %d but the ledger discharged %d", secondDeducted.Minor, discharged)
	}
}

func TestCollectRecoveryDeductionWritesNothingWithoutADebt(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()

	net, deducted, err := CollectRecoveryDeduction(db, chefID, inr(50_000), "order", uuid.NewString(), time.Now())
	if err != nil {
		t.Fatalf("CollectRecoveryDeduction: %v", err)
	}
	if net.Minor != 50_000 || !deducted.IsZero() {
		t.Fatalf("net = %d, deducted = %v; want the gross paid in full", net.Minor, deducted)
	}

	var rows int64
	if err := db.Model(&payouts.LedgerEntry{}).Where("payee_id = ?", chefID).Count(&rows).Error; err != nil {
		t.Fatalf("count entries: %v", err)
	}
	if rows != 0 {
		t.Errorf("wrote %d ledger rows for a chef with no debt, want none", rows)
	}
}

// The resolving entry belongs to the same tenant as the debt it discharges —
// otherwise a tenant-scoped read sees the penalty but not its settlement.
func TestCollectRecoveryDeductionInheritsTheDebtsTenant(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 50_000)

	if _, _, err := CollectRecoveryDeduction(db, chefID, inr(20_000), "order", uuid.NewString(), time.Now()); err != nil {
		t.Fatalf("CollectRecoveryDeduction: %v", err)
	}

	var entry payouts.LedgerEntry
	if err := db.Where("kind = ?", payouts.EntryCreditRecoveryCollected).First(&entry).Error; err != nil {
		t.Fatalf("read resolving entry: %v", err)
	}
	if entry.TenantID != "t1" {
		t.Errorf("tenant = %q, want the debt's tenant t1", entry.TenantID)
	}
	if entry.SourceType != "order" {
		t.Errorf("source type = %q, want order", entry.SourceType)
	}
}

// ApplyRecoveryDeduction stays a pure read: the release governor calls it to
// decide whether to block, and must never collect as a side effect.
func TestApplyRecoveryDeductionStillWritesNothing(t *testing.T) {
	db := newRecoveryTestDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 50_000)

	if _, _, err := ApplyRecoveryDeduction(db, chefID, inr(20_000), time.Now()); err != nil {
		t.Fatalf("ApplyRecoveryDeduction: %v", err)
	}

	if owed := recoveryOwed(t, db, chefID); owed != 50_000 {
		t.Errorf("outstanding = %d after a read-only call, want the full 50000", owed)
	}
}

// ApplyChefRecoveryDeduction is the one site that actually withholds money, so
// it is the one site that must discharge. Two orders against a debt smaller
// than either of them must collect it once, not once each.
func TestApplyChefRecoveryDeductionCollectsADebtOnlyOnce(t *testing.T) {
	db := setupOrderSettlementsDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 20_000)

	first := &models.Order{OrderNumber: "HC-R1", ID: uuid.New(), ChefID: chefID}
	second := &models.Order{OrderNumber: "HC-R2", ID: uuid.New(), ChefID: chefID}

	firstNet := ApplyChefRecoveryDeduction(db, first, 50_000)
	secondNet := ApplyChefRecoveryDeduction(db, second, 50_000)

	if firstNet != 30_000 {
		t.Errorf("first payout = %d paise, want 30000 after collecting the debt", firstNet)
	}
	if secondNet != 50_000 {
		t.Errorf("second payout = %d paise, want the full 50000 — the debt was already collected", secondNet)
	}
	if owed := recoveryOwed(t, db, chefID); owed != 0 {
		t.Errorf("outstanding = %d, want 0", owed)
	}
}

// A retry of the same order re-derives the same answer without collecting again.
func TestApplyChefRecoveryDeductionIsIdempotentPerOrder(t *testing.T) {
	db := setupOrderSettlementsDB(t)
	chefID := uuid.New()
	seedPenalty(t, db, chefID, 20_000)
	order := &models.Order{OrderNumber: "HC-R3", ID: uuid.New(), ChefID: chefID}

	first := ApplyChefRecoveryDeduction(db, order, 50_000)
	second := ApplyChefRecoveryDeduction(db, order, 50_000)

	if first != 30_000 || second != 30_000 {
		t.Errorf("net = %d then %d, want 30000 both times", first, second)
	}
	if owed := recoveryOwed(t, db, chefID); owed != 0 {
		t.Errorf("outstanding = %d, want 0 — a retry must not over-collect", owed)
	}
}
