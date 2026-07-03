package services

// payout_release.go — the admin payout release actuator (#388). Turns
// release_eligible payout holds (#387) into real vendor payouts: list the pending
// queue, and release / withhold / reverse a hold (single + bulk from the handler).
//
// RACE / MONEY SAFETY (the whole point of this file):
//   1. Every transition is a single conditional UPDATE
//      `... WHERE payout_hold_status IN (from...)` guarded by RowsAffected. Zero
//      rows ⇒ the hold wasn't in an eligible source state ⇒ ErrHoldNotEligible
//      (the handler maps this to 409). Two concurrent releases: exactly one wins.
//   2. The status flip runs and COMMITS *before* any Razorpay call, so a
//      non-eligible / already-actioned hold can never reach ReleaseTransfer /
//      ReverseTransfer. Release only from release_eligible; withheld/reversed are
//      terminal and excluded from the queue and from re-release.
//   3. The money seams are all flag-gated (ReleaseOrderPayouts / ReverseOrderPayouts
//      on OrderPayoutAutoReleaseEnabled; ReleaseDayPayout / ReverseTransfer on
//      MealPlanEscrowActive) — OFF ⇒ pure DB state advance, zero money moved. That
//      is the launch config.
//
// PARTIAL-FAILURE DRIFT (must-fix before flags ON): the money seam is dispatched
// AFTER the tx commits. If that post-commit Razorpay call fails, the row is left
// `released` with the money UNMOVED, and there is NO in-slice re-drive — nothing
// here (nor the out-of-scope auto-approve sweep, which only ever operates on
// release_eligible rows) will retry that stranded row. This is safe ONLY because
// both escrow flags are OFF at launch (the seam is a no-op, so it cannot fail).
// Before either flag is turned ON, a reconcile/re-drive path for released-but-
// unpaid holds MUST be built (#388 follow-up).

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// ErrHoldNotEligible is returned when a payout action's conditional UPDATE matches
// no row — the hold wasn't in an allowed source state (racing/duplicate action).
var ErrHoldNotEligible = errors.New("payout hold not in an eligible state")

const (
	aggTypeOrder       = "order"
	aggTypeMealPlanDay = "meal-plan-day"
)

// PendingPayout is one release-eligible hold row for the admin queue. It is a
// review DTO (amount = gross the customer paid), not a reconciliation record — the
// actual Razorpay transfer split is gateway-side.
type PendingPayout struct {
	AggType             string                  `json:"aggType"` // "order" | "meal-plan-day"
	ID                  uuid.UUID               `json:"id"`
	ChefID              uuid.UUID               `json:"chefId"`
	Amount              float64                 `json:"amount"` // Order.Total / MealPlanDay.Price
	HoldStatus          models.PayoutHoldStatus `json:"holdStatus"`
	DeliveredAt         *time.Time              `json:"deliveredAt,omitempty"`
	AgeHours            float64                 `json:"ageHours"` // now - DeliveredAt (SLA clock vs 24h)
	CustomerConfirmedAt *time.Time              `json:"customerConfirmedAt,omitempty"`
	Context             string                  `json:"context"` // order / meal-plan number for the queue row
}

// PendingFilter narrows the pending queue.
type PendingFilter struct {
	IncludeAwaiting bool      // also surface awaiting_customer_confirmation
	ChefID          uuid.UUID // optional chef scope
	Before          *time.Time
}

// pendingStatuses is the source set the queue lists: release_eligible always, plus
// awaiting_customer_confirmation on opt-in. Never disputed/withheld/reversed/released.
func (f PendingFilter) pendingStatuses() []models.PayoutHoldStatus {
	s := []models.PayoutHoldStatus{models.PayoutHoldReleaseEligible}
	if f.IncludeAwaiting {
		s = append(s, models.PayoutHoldAwaitingConfirmation)
	}
	return s
}

// ListPendingPayouts merges pending order and meal-plan-day holds into one queue,
// oldest-first (age desc), so the admin works the longest-waiting payouts first.
func ListPendingPayouts(db *gorm.DB, f PendingFilter) ([]PendingPayout, error) {
	orders, err := listPendingOrders(db, f)
	if err != nil {
		return nil, fmt.Errorf("payout-release: list pending orders: %w", err)
	}
	days, err := listPendingDays(db, f)
	if err != nil {
		return nil, fmt.Errorf("payout-release: list pending days: %w", err)
	}
	out := append(orders, days...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].AgeHours > out[j].AgeHours })
	return out, nil
}

// pendingRow is the flat scan shape for both aggregate queries.
type pendingRow struct {
	ID                  string
	ChefID              string
	Amount              float64
	PayoutHoldStatus    string
	DeliveredAt         *time.Time
	CustomerConfirmedAt *time.Time
	Context             string
}

func listPendingOrders(db *gorm.DB, f PendingFilter) ([]PendingPayout, error) {
	q := db.Table("orders").
		Select("id, chef_id, total AS amount, payout_hold_status, delivered_at, customer_confirmed_at, order_number AS context").
		Where("payout_hold_status IN ?", f.pendingStatuses())
	if f.ChefID != uuid.Nil {
		q = q.Where("chef_id = ?", f.ChefID)
	}
	if f.Before != nil {
		q = q.Where("delivered_at < ?", *f.Before)
	}
	var rows []pendingRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return toPending(aggTypeOrder, rows), nil
}

func listPendingDays(db *gorm.DB, f PendingFilter) ([]PendingPayout, error) {
	q := db.Table("meal_plan_days").
		Select("meal_plan_days.id AS id, meal_plans.chef_id AS chef_id, meal_plan_days.price AS amount, "+
			"meal_plan_days.payout_hold_status AS payout_hold_status, meal_plan_days.delivered_at AS delivered_at, "+
			"meal_plan_days.customer_confirmed_at AS customer_confirmed_at, meal_plans.meal_plan_number AS context").
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Where("meal_plan_days.payout_hold_status IN ?", f.pendingStatuses())
	if f.ChefID != uuid.Nil {
		q = q.Where("meal_plans.chef_id = ?", f.ChefID)
	}
	if f.Before != nil {
		q = q.Where("meal_plan_days.delivered_at < ?", *f.Before)
	}
	var rows []pendingRow
	if err := q.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return toPending(aggTypeMealPlanDay, rows), nil
}

// toPending maps flat rows to DTOs, computing AgeHours from DeliveredAt.
func toPending(aggType string, rows []pendingRow) []PendingPayout {
	now := time.Now()
	out := make([]PendingPayout, 0, len(rows))
	for _, r := range rows {
		id, _ := uuid.Parse(r.ID)
		chef, _ := uuid.Parse(r.ChefID)
		age := 0.0
		if r.DeliveredAt != nil {
			age = now.Sub(*r.DeliveredAt).Hours()
		}
		out = append(out, PendingPayout{
			AggType: aggType, ID: id, ChefID: chef, Amount: r.Amount,
			HoldStatus: models.PayoutHoldStatus(r.PayoutHoldStatus), DeliveredAt: r.DeliveredAt,
			AgeHours: age, CustomerConfirmedAt: r.CustomerConfirmedAt, Context: r.Context,
		})
	}
	return out
}

// holdModel returns an empty model row to scope the conditional UPDATE by aggType.
func holdModel(aggType string) (any, error) {
	switch aggType {
	case aggTypeOrder:
		return &models.Order{}, nil
	case aggTypeMealPlanDay:
		return &models.MealPlanDay{}, nil
	default:
		return nil, fmt.Errorf("payout-release: unknown aggregate type %q", aggType)
	}
}

// transitionHold applies the race-safe conditional status flip in a short tx:
// UPDATE ... WHERE id = ? AND payout_hold_status IN (from). RowsAffected == 0 ⇒ no
// genuine transition ⇒ (false, nil). When emit is set, the hold_released event is
// staged on the outbox inside the SAME tx (so a no-op never double-emits). The
// money seam is intentionally NOT here — it runs post-commit in the caller.
func transitionHold(db *gorm.DB, aggType string, id uuid.UUID,
	from []models.PayoutHoldStatus, to models.PayoutHoldStatus, emit bool) (bool, error) {
	model, err := holdModel(aggType)
	if err != nil {
		return false, err
	}
	changed := false
	err = db.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(model).
			Where("id = ? AND payout_hold_status IN ?", id, from).
			Update("payout_hold_status", to)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		changed = true
		if emit {
			return EnqueueEvent(tx, SubjectHoldReleased, "payout.hold_released", id, map[string]any{
				"aggregate_type": aggType, "aggregate_id": id.String(), "payout_hold_status": string(to),
			})
		}
		return nil
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

// ReleaseHold advances a hold release_eligible → released (race-safe conditional
// UPDATE, committed) then dispatches the flag-gated money seam. Re-release of an
// already-released row is a no-op → ErrHoldNotEligible. See the file header on
// post-commit partial-failure drift (safe only while flags are OFF).
func ReleaseHold(db *gorm.DB, aggType string, id uuid.UUID) error {
	ok, err := transitionHold(db, aggType, id,
		[]models.PayoutHoldStatus{models.PayoutHoldReleaseEligible}, models.PayoutHoldReleased, true)
	if err != nil {
		return fmt.Errorf("payout-release: release %s %s: %w", aggType, id, err)
	}
	if !ok {
		return ErrHoldNotEligible
	}
	return releaseMoney(db, aggType, id)
}

// releaseMoney runs the post-commit, flag-gated release seam. Order → the Route
// transfer release; meal-plan day → the held-transfer release. Both no-op when
// their escrow flag is OFF or the gateway is unconfigured.
func releaseMoney(db *gorm.DB, aggType string, id uuid.UUID) error {
	switch aggType {
	case aggTypeOrder:
		if err := ReleaseOrderPayouts(id); err != nil {
			return fmt.Errorf("payout-release: release order payout %s: %w", id, err)
		}
	case aggTypeMealPlanDay:
		var day models.MealPlanDay
		if err := db.First(&day, "id = ?", id).Error; err != nil {
			return fmt.Errorf("payout-release: load day %s: %w", id, err)
		}
		if err := ReleaseDayPayout(database.DB, &day); err != nil {
			return fmt.Errorf("payout-release: release day payout %s: %w", id, err)
		}
	}
	return nil
}

// WithholdHold blocks an eligible payout: release_eligible/awaiting → withheld. No
// event, no money. The reason is recorded in audit at the handler (guarded here
// too so a direct call can't skip it). Terminal — the row leaves the queue.
func WithholdHold(db *gorm.DB, aggType string, id uuid.UUID, reason string) error {
	if reason == "" {
		return fmt.Errorf("payout-release: withhold reason required")
	}
	ok, err := transitionHold(db, aggType, id,
		[]models.PayoutHoldStatus{models.PayoutHoldReleaseEligible, models.PayoutHoldAwaitingConfirmation},
		models.PayoutHoldWithheld, false)
	if err != nil {
		return fmt.Errorf("payout-release: withhold %s %s: %w", aggType, id, err)
	}
	if !ok {
		return ErrHoldNotEligible
	}
	return nil
}

// ReverseHold claws a released/eligible payout back to the platform: → reversed,
// then the flag-gated reverse seam. Reason mandatory (audit at the handler).
func ReverseHold(db *gorm.DB, aggType string, id uuid.UUID, reason string) error {
	if reason == "" {
		return fmt.Errorf("payout-release: reverse reason required")
	}
	ok, err := transitionHold(db, aggType, id,
		[]models.PayoutHoldStatus{models.PayoutHoldReleased, models.PayoutHoldReleaseEligible},
		models.PayoutHoldReversed, false)
	if err != nil {
		return fmt.Errorf("payout-release: reverse %s %s: %w", aggType, id, err)
	}
	if !ok {
		return ErrHoldNotEligible
	}
	return reverseMoney(db, aggType, id)
}

// reverseMoney runs the post-commit, flag-gated reverse seam. Order → the Route
// transfer claw-back; meal-plan day → a full ReverseTransfer of the held transfer
// (guarded on the escrow flag + a present transfer id + a configured gateway).
func reverseMoney(db *gorm.DB, aggType string, id uuid.UUID) error {
	switch aggType {
	case aggTypeOrder:
		if err := ReverseOrderPayouts(id); err != nil {
			return fmt.Errorf("payout-release: reverse order payout %s: %w", id, err)
		}
	case aggTypeMealPlanDay:
		var day models.MealPlanDay
		if err := db.First(&day, "id = ?", id).Error; err != nil {
			return fmt.Errorf("payout-release: load day %s: %w", id, err)
		}
		if MealPlanEscrowActive() && day.PayoutTransferID != "" && GetRazorpay() != nil {
			if _, err := GetRazorpay().ReverseTransfer(day.PayoutTransferID, 0); err != nil {
				return fmt.Errorf("payout-release: reverse day transfer %s: %w", day.PayoutTransferID, err)
			}
		}
	}
	return nil
}

// GetPayoutAutoApproveHours reads the auto-approve window (hours) from
// PlatformSettings `payout.*`, DEFAULT 0 (disabled = manual-first). Consumed by a
// follow-up auto-approve sweep (not wired in this slice); the knob exists now so
// ops can pre-tune it. Mirrors GetCustomerConfirmWindowHours' fold pattern.
func GetPayoutAutoApproveHours(db *gorm.DB) int {
	hours := 0
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "payout.%").Find(&settings)
	for _, s := range settings {
		if s.Key == "payout.auto_approve_after_hours" {
			if v, err := strconv.Atoi(s.Value); err == nil {
				hours = v
			}
		}
	}
	return hours
}
