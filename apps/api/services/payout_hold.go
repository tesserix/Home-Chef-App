package services

// payout_hold.go — the payout hold state machine seam (#387). Delivery parks a
// hold instead of releasing funds; an explicit customer confirmation advances it
// to release_eligible (consumed later by the admin payout queue #388). Every
// transition is a conditional UPDATE guarded on the source status, so replayed
// webhooks / retried saga activities can never double-advance or re-open a hold.
//
// KEY INVARIANT: a disputed or unconfirmed hold must NEVER reach
// release_eligible. The only UPDATE that produces release_eligible is gated on
// payout_hold_status = 'awaiting_customer_confirmation'; disputed/released rows
// fail the WHERE and are untouched.
//
// Setting the hold is plain DB state and runs regardless of the escrow flags;
// the flags only gate real MONEY movement (the release/reverse seams in
// order_payout.go / meal_plan_escrow.go), which #388 will drive off release_eligible.

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// SetOrderHoldAwaitingConfirmation parks a delivered regular order's payout in a
// customer-confirmation hold. No-op for orders without a Razorpay order id
// (meal-plan/group consolidated orders settle through their own paths — mirrors
// the order_payout.go scoping). Idempotent: the conditional update only fires
// from the empty (pre-delivery) state, so a replayed delivered event is a no-op.
func SetOrderHoldAwaitingConfirmation(db *gorm.DB, orderID uuid.UUID) error {
	var order models.Order
	if err := db.Select("id", "razorpay_order_id", "payout_hold_status").
		First(&order, "id = ?", orderID).Error; err != nil {
		return fmt.Errorf("payout-hold: load order %s: %w", orderID, err)
	}
	if order.RazorpayOrderID == "" {
		return nil // not a gateway-charged regular order
	}
	res := db.Model(&models.Order{}).
		Where("id = ? AND payout_hold_status = ?", orderID, models.PayoutHoldNone).
		Update("payout_hold_status", models.PayoutHoldAwaitingConfirmation)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: park order %s: %w", orderID, res.Error)
	}
	return nil
}

// SetMealPlanDayHoldAwaitingConfirmation parks a delivered meal-plan day's payout
// in a customer-confirmation hold. Called inside the delivery transaction.
// Idempotent (only advances from the empty state).
func SetMealPlanDayHoldAwaitingConfirmation(tx *gorm.DB, dayID uuid.UUID) error {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status = ?", dayID, models.PayoutHoldNone).
		Update("payout_hold_status", models.PayoutHoldAwaitingConfirmation)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: park meal-plan day %s: %w", dayID, res.Error)
	}
	return nil
}

// SetGroupOrderHoldAwaitingConfirmation parks a delivered group order's payout in a
// customer-confirmation hold. Called inside the delivery transaction. Idempotent
// (the conditional update only advances from the empty pre-delivery state, so a
// replayed delivered event is a no-op).
func SetGroupOrderHoldAwaitingConfirmation(tx *gorm.DB, groupOrderID uuid.UUID) error {
	res := tx.Model(&models.GroupOrder{}).
		Where("id = ? AND payout_hold_status = ?", groupOrderID, models.PayoutHoldNone).
		Update("payout_hold_status", models.PayoutHoldAwaitingConfirmation)
	if res.Error != nil {
		return fmt.Errorf("payout-hold: park group order %s: %w", groupOrderID, res.Error)
	}
	return nil
}

// HasOpenOrderIssue reports whether the order has a pending (still-under-review)
// OrderIssue — the dispute signal that blocks a hold from reaching release_eligible.
func HasOpenOrderIssue(db *gorm.DB, orderID uuid.UUID) bool {
	var count int64
	db.Model(&models.OrderIssue{}).
		Where("order_id = ? AND status = ?", orderID, models.IssuePending).
		Count(&count)
	return count > 0
}

// applyHoldConfirm performs the guarded confirm transition on a model row: to
// disputed when an open issue exists (only from awaiting/disputed), otherwise to
// release_eligible (only from awaiting). The WHERE guard IS the safety invariant —
// a released row can never flip to release_eligible. When (and only when) the
// conditional UPDATE actually changes a row it emits the matching NATS event onto
// the transactional outbox within the same tx, so a replayed/no-op confirm never
// double-emits. aggType is the aggregate identity ("order" / "meal_plan_day").
func applyHoldConfirm(tx *gorm.DB, model any, aggType string, id uuid.UUID, disputed bool, now time.Time) error {
	target := models.PayoutHoldReleaseEligible
	where := tx.Model(model).Where("id = ? AND payout_hold_status = ?", id, models.PayoutHoldAwaitingConfirmation)
	if disputed {
		target = models.PayoutHoldDisputed
		where = tx.Model(model).Where("id = ? AND payout_hold_status IN ?", id,
			[]models.PayoutHoldStatus{models.PayoutHoldAwaitingConfirmation, models.PayoutHoldDisputed})
	}
	res := where.Updates(map[string]any{"payout_hold_status": target, "customer_confirmed_at": now})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return nil // no genuine transition — nothing to emit
	}
	return emitHoldEvent(tx, target, aggType, id)
}

// emitHoldEvent stages the hold-transition event onto the outbox within tx.
func emitHoldEvent(tx *gorm.DB, target models.PayoutHoldStatus, aggType string, id uuid.UUID) error {
	subject, eventType := SubjectHoldReleaseEligible, "payout.hold_release_eligible"
	if target == models.PayoutHoldDisputed {
		subject, eventType = SubjectHoldDisputed, "payout.hold_disputed"
	}
	if err := EnqueueEvent(tx, subject, eventType, id, map[string]any{
		"aggregate_type": aggType, "aggregate_id": id.String(), "payout_hold_status": string(target),
	}); err != nil {
		return fmt.Errorf("payout-hold: emit %s for %s %s: %w", subject, aggType, id, err)
	}
	return nil
}

// ConfirmOrderHold advances a regular order's hold awaiting -> release_eligible on
// customer confirmation, or -> disputed when an open OrderIssue exists. Idempotent:
// an already-confirmed order returns its terminal status without re-stamping. The
// transition is a guarded conditional update, so disputed/released holds can never
// reach release_eligible. Returns the resulting status.
func ConfirmOrderHold(db *gorm.DB, order *models.Order) (models.PayoutHoldStatus, error) {
	if order.CustomerConfirmedAt != nil {
		return order.PayoutHoldStatus, nil
	}
	disputed := HasOpenOrderIssue(db, order.ID)
	now := time.Now()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return applyHoldConfirm(tx, &models.Order{}, "order", order.ID, disputed, now)
	}); err != nil {
		return "", fmt.Errorf("payout-hold: confirm order %s: %w", order.ID, err)
	}
	var updated models.Order
	if err := db.Select("payout_hold_status", "customer_confirmed_at").
		First(&updated, "id = ?", order.ID).Error; err != nil {
		return "", fmt.Errorf("payout-hold: reload order %s: %w", order.ID, err)
	}
	order.PayoutHoldStatus = updated.PayoutHoldStatus
	order.CustomerConfirmedAt = updated.CustomerConfirmedAt
	return updated.PayoutHoldStatus, nil
}

// ConfirmMealPlanDayHold is ConfirmOrderHold for a meal-plan day. The dispute
// check keys on the day's OrderID (a day with no order has no dispute source and
// proceeds to release_eligible).
func ConfirmMealPlanDayHold(db *gorm.DB, day *models.MealPlanDay) (models.PayoutHoldStatus, error) {
	if day.CustomerConfirmedAt != nil {
		return day.PayoutHoldStatus, nil
	}
	disputed := day.OrderID != nil && HasOpenOrderIssue(db, *day.OrderID)
	now := time.Now()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return applyHoldConfirm(tx, &models.MealPlanDay{}, "meal_plan_day", day.ID, disputed, now)
	}); err != nil {
		return "", fmt.Errorf("payout-hold: confirm meal-plan day %s: %w", day.ID, err)
	}
	var updated models.MealPlanDay
	if err := db.Select("payout_hold_status", "customer_confirmed_at").
		First(&updated, "id = ?", day.ID).Error; err != nil {
		return "", fmt.Errorf("payout-hold: reload meal-plan day %s: %w", day.ID, err)
	}
	day.PayoutHoldStatus = updated.PayoutHoldStatus
	day.CustomerConfirmedAt = updated.CustomerConfirmedAt
	return updated.PayoutHoldStatus, nil
}

// ConfirmGroupOrderHold is ConfirmOrderHold for a group/office order (#456). The
// host confirming receipt advances the hold awaiting -> release_eligible, or ->
// disputed when the consolidated order has an open OrderIssue (keyed on OrderID; a
// group with no consolidated order has no dispute source and proceeds). Idempotent
// on an already-confirmed group. Returns the resulting status.
func ConfirmGroupOrderHold(db *gorm.DB, g *models.GroupOrder) (models.PayoutHoldStatus, error) {
	if g.CustomerConfirmedAt != nil {
		return g.PayoutHoldStatus, nil
	}
	disputed := g.OrderID != nil && HasOpenOrderIssue(db, *g.OrderID)
	now := time.Now()
	if err := db.Transaction(func(tx *gorm.DB) error {
		return applyHoldConfirm(tx, &models.GroupOrder{}, "group-order", g.ID, disputed, now)
	}); err != nil {
		return "", fmt.Errorf("payout-hold: confirm group order %s: %w", g.ID, err)
	}
	var updated models.GroupOrder
	if err := db.Select("payout_hold_status", "customer_confirmed_at").
		First(&updated, "id = ?", g.ID).Error; err != nil {
		return "", fmt.Errorf("payout-hold: reload group order %s: %w", g.ID, err)
	}
	g.PayoutHoldStatus = updated.PayoutHoldStatus
	g.CustomerConfirmedAt = updated.CustomerConfirmedAt
	return updated.PayoutHoldStatus, nil
}

// ConfirmTodaysTiffinForCustomer bulk-confirms every one of the customer's own
// delivered-today, still-awaiting meal-plan days (bounded to the caller's plans).
// Returns the number confirmed.
func ConfirmTodaysTiffinForCustomer(db *gorm.DB, customerID uuid.UUID) (int, error) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		ist = time.FixedZone("IST", 5*3600+1800)
	}
	now := time.Now().In(ist)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, ist)
	end := start.AddDate(0, 0, 1)

	var days []models.MealPlanDay
	if err := db.
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Where("meal_plans.customer_id = ?", customerID).
		Where("meal_plan_days.status = ?", models.MealPlanDayDelivered).
		Where("meal_plan_days.payout_hold_status = ?", models.PayoutHoldAwaitingConfirmation).
		Where("meal_plan_days.customer_confirmed_at IS NULL").
		Where("meal_plan_days.date >= ? AND meal_plan_days.date < ?", start, end).
		Find(&days).Error; err != nil {
		return 0, fmt.Errorf("payout-hold: load today's tiffin for %s: %w", customerID, err)
	}

	confirmed := 0
	for i := range days {
		if _, err := ConfirmMealPlanDayHold(db, &days[i]); err != nil {
			return confirmed, err
		}
		confirmed++
	}
	return confirmed, nil
}

// GetCustomerConfirmWindowHours reads the auto-confirm window (hours) from
// PlatformSettings `payout.*` keys, defaulting to 24. Consumed by the follow-up
// auto-confirm sweep that advances stale awaiting_customer_confirmation holds; the
// setting exists now so ops can pre-tune it. Mirrors GetIssueConfig's fold pattern
// (parse errors are ignored, keeping the default). Not called by any transition
// in this slice.
func GetCustomerConfirmWindowHours(db *gorm.DB) int {
	hours := 24
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "payout.%").Find(&settings)
	for _, s := range settings {
		if s.Key == "payout.customer_confirm_window_hours" {
			if v, err := strconv.Atoi(s.Value); err == nil {
				hours = v
			}
		}
	}
	return hours
}
