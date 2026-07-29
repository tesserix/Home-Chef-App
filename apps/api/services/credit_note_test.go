package services

// credit_note_test.go — GST credit notes (#834 item 2).
//
// v3 refunds collected tax, so a refund without a matching note leaves the filing overstating
// output tax. These tests pin that a note is issued for every refund that returns GST, that its
// arithmetic reconciles against the refund, and that it is issued exactly ONCE per day however
// many times the refund path is re-driven.

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

func creditNotesFor(t *testing.T, db *gorm.DB, dayID uuid.UUID) []models.CreditNote {
	t.Helper()
	var out []models.CreditNote
	require.NoError(t, db.Where("source_key = ?", CreditNoteSourceKeyMealPlanDay(dayID)).Find(&out).Error)
	return out
}

// A refund that returns GST issues a note whose amounts reconcile against it.
func TestRefund_IssuesGSTCreditNote(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 100, models.RefundDestinationWallet))

	notes := creditNotesFor(t, db, day.ID)
	require.Len(t, notes, 1)
	n := notes[0]
	require.Equal(t, 162.0, n.TotalAmount, "the note covers the whole refund")
	require.Equal(t, 16.0, n.TaxAmount, "the GST slice is what the filing must reverse")
	require.Equal(t, 146.0, n.TaxableValue, "total − tax")
	require.Equal(t, 100, n.RefundPercent)
	require.Equal(t, plan.MealPlanNumber, n.Reference)
	require.NotEmpty(t, n.CreditNoteNumber)
	require.Contains(t, n.CreditNoteNumber, "CN-", "credit notes use their own series, never the invoice series")
}

// A partial refund reverses only the GST it actually returned.
func TestRefund_PartialCreditNoteIsProportional(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 75, models.RefundDestinationWallet))

	notes := creditNotesFor(t, db, day.ID)
	require.Len(t, notes, 1)
	require.Equal(t, 121.5, notes[0].TotalAmount, "75% of 162")
	require.Equal(t, 12.0, notes[0].TaxAmount, "75% of the day's 16 GST")
	require.Equal(t, 75, notes[0].RefundPercent)
}

// Re-driving the refund must not mint a second note — a duplicated credit note under-reports
// output tax exactly as badly as a missing one over-reports it.
func TestRefund_CreditNoteIsIdempotent(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 100, models.RefundDestinationWallet))
	day2 := &models.MealPlanDay{ID: day.ID, MealPlanID: plan.ID, Price: 160, CommissionRate: 0.15}
	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day2, 100, models.RefundDestinationWallet))

	require.Len(t, creditNotesFor(t, db, day.ID), 1, "one refund, one note")
}

// A 0% decision moves no money and returns no tax, so there is nothing to credit.
func TestRefund_NoRefundNoCreditNote(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	plan, day := seedV2Day(t, db, u)

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 0, models.RefundDestinationWallet))
	require.Empty(t, creditNotesFor(t, db, day.ID))
}

// A legacy plan with no snapshotted tax collected no GST through this path, so no note is due.
func TestRefund_LegacyPlanNoTaxNoCreditNote(t *testing.T) {
	v2EscrowOn(t)
	db := setupV2RefundDB(t)
	u := uuid.New()
	planID, dayID := uuid.New(), uuid.New()
	require.NoError(t, db.Exec(`INSERT INTO meal_plan_days (id, meal_plan_id, status, price, commission_rate)
		VALUES (?,?,?,?,?)`, dayID.String(), planID.String(), string(models.MealPlanDaySkipRequested), 160.0, 0.15).Error)
	plan := &models.MealPlan{ID: planID, CustomerID: u, MealPlanNumber: "MP-LEGACY", EscrowPaymentID: "pay_test"}
	day := &models.MealPlanDay{ID: dayID, MealPlanID: planID, Price: 160, CommissionRate: 0.15}

	require.NoError(t, ExecuteMealPlanV2Refund(db, plan, day, 100, models.RefundDestinationWallet))
	require.Equal(t, 136.0, v2WalletBalance(t, db, u), "food net of commission — there was no GST to return")
	require.Empty(t, creditNotesFor(t, db, dayID))
}

func TestGenerateCreditNoteNumber_Unique(t *testing.T) {
	seen := map[string]bool{}
	for range 200 {
		n := generateCreditNoteNumber()
		require.False(t, seen[n], "credit-note numbers must not repeat: %s", n)
		seen[n] = true
	}
}
