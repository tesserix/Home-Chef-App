package services

import (
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// statement_recovery.go — the weekly statement's recovery line (#1092).
//
// Collection itself is CollectRecoveryDeduction's job and stays there: this is a
// second call site for it, not a second mechanism. The ledger is what keeps the
// two honest — whichever site collects first writes the resolving entry, and the
// other finds nothing owed.

// ApplyChefRecoveryToStatement collects the chef's outstanding debt off a
// freshly-issued statement and records it as that statement's recovery line.
//
// Returns the rupees collected. Idempotent through the derived entry id: a
// re-run of statement generation for the same statement re-derives it, writes
// nothing, and reports what it collected the first time — so the payout is
// reduced once and only once.
func ApplyChefRecoveryToStatement(db *gorm.DB, stmt *models.WeeklyStatement) (float64, error) {
	// Gross is the payout BEFORE recovery, reconstructed from the line already on
	// the statement. Passing the reduced net instead would let a re-run deduct
	// the same debt off an already-reduced figure — the collector reports what it
	// withheld the first time, so net would fall by the debt again on every run.
	gross := payouts.FromMinor(rupeesToMinor(stmt.NetPayout+stmt.RecoveryDeductions), payouts.CurrencyINR)
	if !gross.IsPositive() {
		return 0, nil
	}

	net, deducted, err := CollectRecoveryDeduction(
		db, stmt.ChefID, gross, "statement", stmt.ID.String(), time.Now().UTC())
	if err != nil {
		return 0, err
	}
	if deducted.IsZero() {
		return 0, nil
	}

	collected := minorToRupees(deducted.Minor)
	// Written as a total, not an increment: a re-run reports the same collection
	// and must land on the same number rather than doubling the line.
	if err := db.Model(&models.WeeklyStatement{}).Where("id = ?", stmt.ID).
		Updates(map[string]any{
			"recovery_deductions": collected,
			"net_payout":          minorToRupees(net.Minor),
		}).Error; err != nil {
		return 0, err
	}
	stmt.RecoveryDeductions = collected
	stmt.NetPayout = minorToRupees(net.Minor)
	return collected, nil
}

func rupeesToMinor(rupees float64) int64 { return int64(math.Round(rupees * 100)) }

func minorToRupees(minor int64) float64 { return float64(minor) / 100 }
