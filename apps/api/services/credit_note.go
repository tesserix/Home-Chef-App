package services

// credit_note.go — GST credit-note issuance (#834 item 2, docs/refund-policy-v3-spec.md).
//
// Refund policy v3 refunds the GST the platform collected. Output tax that has been returned
// to the customer must be backed out of the filing, and the instrument for that is a credit
// note against the original supply. Nothing of the sort existed anywhere in the codebase
// before v3 — refunds simply never touched tax — so this ships WITH the base change rather
// than after it: a refund that returns GST without issuing the note creates a filing
// discrepancy on day one.
//
// The note is written INSIDE the refund transaction, so a note that cannot be recorded rolls
// the refund back instead of quietly returning tax off the books. Idempotency is a UNIQUE
// source key, so a re-driven or concurrent refund can never mint a second note.

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/homechef/api/models"
)

// CreditNoteSourceKeyMealPlanDay is the natural key of a meal-plan day refund. It intentionally
// mirrors dayRefundKey's granularity — one day, one refund, one note.
func CreditNoteSourceKeyMealPlanDay(dayID uuid.UUID) string { return "mealplanday:" + dayID.String() }

// IssueMealPlanDayCreditNote records the credit note reversing the GST slice of a refund that
// has just landed for one meal-plan day. refundAmount is what the customer got back; gstAmount
// is the tax portion of it. Runs in the caller's tx.
//
// Idempotent: a second call for the same day is a no-op, so the executor's re-drive path and a
// concurrent writer can never mint two notes for the same money.
func IssueMealPlanDayCreditNote(tx *gorm.DB, plan *models.MealPlan, day *models.MealPlanDay, refundAmount, gstAmount float64) error {
	if plan == nil || day == nil || gstAmount <= 0 {
		return nil
	}
	planID, dayID := plan.ID, day.ID
	note := models.CreditNote{
		CreditNoteNumber: generateCreditNoteNumber(),
		SourceKey:        CreditNoteSourceKeyMealPlanDay(dayID),
		CustomerID:       plan.CustomerID,
		MealPlanID:       &planID,
		MealPlanDayID:    &dayID,
		Reference:        plan.MealPlanNumber,
		Currency:         EarningsCurrency,
		TaxAmount:        Round2(gstAmount),
		TaxableValue:     Round2(refundAmount - gstAmount),
		TotalAmount:      Round2(refundAmount),
		RefundPercent:    day.RefundPercentOf(),
		Reason:           fmt.Sprintf("Tiffin %s — cancelled day refunded, GST reversed", plan.MealPlanNumber),
		IssuedAt:         time.Now().UTC(),
	}
	if plan.ChefID != uuid.Nil {
		chefID := plan.ChefID
		note.ChefID = &chefID
	}
	return issueCreditNote(tx, &note)
}

// issueCreditNote persists a note, treating a unique-key collision on SourceKey as success —
// the note already exists, which is all any caller needs.
func issueCreditNote(tx *gorm.DB, note *models.CreditNote) error {
	err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "source_key"}}, DoNothing: true}).
		Create(note).Error
	if err != nil && isDuplicateKeyErr(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("issue credit note %s: %w", note.SourceKey, err)
	}
	return nil
}

// isDuplicateKeyErr reports a unique-violation. The ON CONFLICT clause covers Postgres; this
// is the fallback for dialects (the sqlite test harness) that surface the collision as a plain
// constraint error instead.
func isDuplicateKeyErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	msg := strings.ToLower(err.Error())
	for _, m := range []string{"unique constraint", "duplicate key", "unique_violation", "constraint failed"} {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}

// generateCreditNoteNumber mints a credit-note reference in its OWN series — never the invoice
// series, which must stay a clean consecutive run of supplies. Format: CN-YYYYMMDD-XXXXXX,
// mirroring generateOrderInvoiceNumber. A random suffix (rather than a counter) keeps issuance
// lock-free across pods; the DB unique index is the collision backstop.
func generateCreditNoteNumber() string {
	now := time.Now().UTC()
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("CN-%s-%06d", now.Format("20060102"), now.UnixNano()%1000000)
	}
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	out := make([]byte, len(b))
	for i, v := range b {
		out[i] = chars[int(v)%len(chars)]
	}
	return fmt.Sprintf("CN-%s-%s", now.Format("20060102"), string(out))
}
