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
	aggTypeGroupOrder  = "group-order"
)

// PendingPayout is one release-eligible hold row for the admin queue. It is a
// review DTO, not a reconciliation record — the actual Razorpay transfer split is
// gateway-side. Amount is per-aggregate: Order.Total (gross the customer paid),
// MealPlanDay.Price (the day's price), or the group order's chef slice
// (subtotal + tax) — NOT always the gross customer total.
type PendingPayout struct {
	AggType string    `json:"aggType"` // "order" | "meal-plan-day" | "group-order"
	ID      uuid.UUID `json:"id"`
	ChefID  uuid.UUID `json:"chefId"`
	Amount  float64   `json:"amount"` // Order.Total / MealPlanDay.Price / group chef slice (subtotal+tax)
	// NetPayout is the chef's actual net transfer for this aggregate — what the
	// held Route transfer carries and what the admin is really releasing (#462/#546):
	// order = ComputeOrderEarnings.NetPayout, day = perDayNetPayout, group =
	// groupNetPayout — all net of commission + TDS. For orders and groups it is below
	// Amount (the gross customer/chef-slice value). For meal-plan days Amount is the
	// food-only Price, so Net (which adds the day's food GST) can sit slightly above
	// it — the two just measure different things, and Net is the figure that moves money.
	NetPayout           float64                 `json:"netPayout"`
	HoldStatus          models.PayoutHoldStatus `json:"holdStatus"`
	DeliveredAt         *time.Time              `json:"deliveredAt,omitempty"`
	AgeHours            float64                 `json:"ageHours"` // now - DeliveredAt (SLA clock vs 24h)
	CustomerConfirmedAt *time.Time              `json:"customerConfirmedAt,omitempty"`
	Context             string                  `json:"context"`      // order / meal-plan number for the queue row
	HasOpenIssue        bool                    `json:"hasOpenIssue"` // order aggregate has a pending OrderIssue (#457)
}

// PendingFilter narrows the pending queue.
type PendingFilter struct {
	IncludeAwaiting bool      // also surface awaiting_customer_confirmation
	IncludeDisputed bool      // also surface disputed (visibility only — non-releasable) (#458)
	ChefID          uuid.UUID // optional chef scope
	Before          *time.Time
}

// pendingStatuses is the source set the queue lists: release_eligible always, plus
// awaiting_customer_confirmation and/or disputed on opt-in. disputed rows are
// surfaced for admin visibility only — they stay non-releasable (the release path's
// pending-issue guard and the release_eligible-only source guard both block them).
// Never withheld/reversed/released.
func (f PendingFilter) pendingStatuses() []models.PayoutHoldStatus {
	s := []models.PayoutHoldStatus{models.PayoutHoldReleaseEligible}
	if f.IncludeAwaiting {
		s = append(s, models.PayoutHoldAwaitingConfirmation)
	}
	if f.IncludeDisputed {
		s = append(s, models.PayoutHoldDisputed)
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
	groups, err := listPendingGroupOrders(db, f)
	if err != nil {
		return nil, fmt.Errorf("payout-release: list pending group orders: %w", err)
	}
	out := append(orders, days...)
	out = append(out, groups...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].AgeHours > out[j].AgeHours })
	return out, nil
}

// pendingRow is the flat scan shape for the aggregate queries. NetPayout is filled
// per aggregate (in Go for orders/days, in SQL for groups) — see the list funcs.
type pendingRow struct {
	ID                  string
	ChefID              string
	Amount              float64
	NetPayout           float64
	PayoutHoldStatus    string
	DeliveredAt         *time.Time
	CustomerConfirmedAt *time.Time
	Context             string
	HasOpenIssue        bool
}

func listPendingOrders(db *gorm.DB, f PendingFilter) ([]PendingPayout, error) {
	// NET is computed in Go via ComputeOrderEarnings — the SAME function that sizes
	// the held Route transfer — so the queue can't drift from what actually moves.
	// NetPayout ignores the CGST/SGST vs IGST split, so chefState is irrelevant here
	// (passed ""); the extra money columns feed the pure computation.
	type orderNetRow struct {
		ID                  string
		ChefID              string
		Amount              float64
		PayoutHoldStatus    string
		DeliveredAt         *time.Time
		CustomerConfirmedAt *time.Time
		Context             string
		HasOpenIssue        bool
		Subtotal            float64
		Tax                 float64
		ChefTip             float64
		ChefFundedDiscount  float64
		CommissionRate      float64
	}
	q := db.Table("orders").
		Select("id, chef_id, total AS amount, payout_hold_status, delivered_at, customer_confirmed_at, "+
			"order_number AS context, subtotal, tax, chef_tip, chef_funded_discount, commission_rate, "+
			"EXISTS(SELECT 1 FROM order_issues oi WHERE oi.order_id = orders.id AND oi.status = 'pending') AS has_open_issue").
		Where("payout_hold_status IN ?", f.pendingStatuses()).
		// Cross-guard (#457): never surface a refunded/cancelled order NOR one with
		// refunded_at set (the issue path leaves status='delivered' but stamps
		// refunded_at). The admin must not be able to release a refunded order.
		Where("status NOT IN ?", []string{string(models.OrderStatusRefunded), string(models.OrderStatusCancelled)}).
		Where("refunded_at IS NULL")
	if f.ChefID != uuid.Nil {
		q = q.Where("chef_id = ?", f.ChefID)
	}
	if f.Before != nil {
		q = q.Where("delivered_at < ?", *f.Before)
	}
	var raw []orderNetRow
	if err := q.Scan(&raw).Error; err != nil {
		return nil, err
	}
	rows := make([]pendingRow, 0, len(raw))
	for _, r := range raw {
		net := ComputeOrderEarnings(EarningsInput{
			ItemRevenue:        r.Subtotal,
			Tax:                r.Tax,
			ChefTip:            r.ChefTip,
			ChefFundedDiscount: r.ChefFundedDiscount,
			CommissionRate:     r.CommissionRate,
		}, "").NetPayout
		rows = append(rows, pendingRow{
			ID: r.ID, ChefID: r.ChefID, Amount: r.Amount, NetPayout: net,
			PayoutHoldStatus: r.PayoutHoldStatus, DeliveredAt: r.DeliveredAt,
			CustomerConfirmedAt: r.CustomerConfirmedAt, Context: r.Context, HasOpenIssue: r.HasOpenIssue,
		})
	}
	return toPending(aggTypeOrder, rows), nil
}

func listPendingDays(db *gorm.DB, f PendingFilter) ([]PendingPayout, error) {
	// NET per day via perDayNetPayout — the SAME basis HoldChefPayouts uses to size
	// the held transfer (#518). Unlike orders (which freeze Order.CommissionRate),
	// meal-plan days don't persist the rate the transfer was held at, so we resolve
	// the CURRENT flat rate once. If the platform rate changes while pre-change day
	// holds are still pending, this display can drift a few paise from the amount
	// actually held — a review DTO, not the money seam (follow-up: persist the rate).
	type dayNetRow struct {
		ID                  string
		ChefID              string
		Amount              float64
		PayoutHoldStatus    string
		DeliveredAt         *time.Time
		CustomerConfirmedAt *time.Time
		Context             string
		Price               float64
		PlanSubtotal        float64
		PlanTax             float64
	}
	q := db.Table("meal_plan_days").
		Select("meal_plan_days.id AS id, meal_plans.chef_id AS chef_id, meal_plan_days.price AS amount, "+
			"meal_plan_days.payout_hold_status AS payout_hold_status, meal_plan_days.delivered_at AS delivered_at, "+
			"meal_plan_days.customer_confirmed_at AS customer_confirmed_at, meal_plans.meal_plan_number AS context, "+
			"meal_plan_days.price AS price, meal_plans.subtotal AS plan_subtotal, meal_plans.tax AS plan_tax").
		Joins("JOIN meal_plans ON meal_plans.id = meal_plan_days.meal_plan_id").
		Where("meal_plan_days.payout_hold_status IN ?", f.pendingStatuses())
	if f.ChefID != uuid.Nil {
		q = q.Where("meal_plans.chef_id = ?", f.ChefID)
	}
	if f.Before != nil {
		q = q.Where("meal_plan_days.delivered_at < ?", *f.Before)
	}
	var raw []dayNetRow
	if err := q.Scan(&raw).Error; err != nil {
		return nil, err
	}
	rate := GetCommissionRate(db)
	rows := make([]pendingRow, 0, len(raw))
	for _, r := range raw {
		net := perDayNetPayout(
			&models.MealPlan{Subtotal: r.PlanSubtotal, Tax: r.PlanTax},
			&models.MealPlanDay{Price: r.Price}, rate)
		rows = append(rows, pendingRow{
			ID: r.ID, ChefID: r.ChefID, Amount: r.Amount, NetPayout: net,
			PayoutHoldStatus: r.PayoutHoldStatus, DeliveredAt: r.DeliveredAt,
			CustomerConfirmedAt: r.CustomerConfirmedAt, Context: r.Context,
		})
	}
	return toPending(aggTypeMealPlanDay, rows), nil
}

// listPendingGroupOrders lists group/office order holds in the pending states. Amount
// is the gross chef slice (subtotal + tax); NetPayout is the chef's actual transfer,
// net of commission + TDS (#546) — computed in Go via groupNetPayout, the SAME basis
// HoldGroupChefPayout uses to size the held transfer. Context is a short
// human-scannable id (GRP-<8 hex>), never a raw UUID.
func listPendingGroupOrders(db *gorm.DB, f PendingFilter) ([]PendingPayout, error) {
	type groupNetRow struct {
		ID                  string
		ChefID              string
		Amount              float64
		PayoutHoldStatus    string
		DeliveredAt         *time.Time
		CustomerConfirmedAt *time.Time
		Context             string
		Subtotal            float64
		Tax                 float64
	}
	q := db.Table("group_orders").
		Select("id, chef_id, subtotal + tax AS amount, payout_hold_status, delivered_at, "+
			"customer_confirmed_at, 'GRP-' || substr(id, 1, 8) AS context, subtotal, tax").
		Where("payout_hold_status IN ?", f.pendingStatuses())
	if f.ChefID != uuid.Nil {
		q = q.Where("chef_id = ?", f.ChefID)
	}
	if f.Before != nil {
		q = q.Where("delivered_at < ?", *f.Before)
	}
	var raw []groupNetRow
	if err := q.Scan(&raw).Error; err != nil {
		return nil, err
	}
	rate := GetCommissionRate(db)
	rows := make([]pendingRow, 0, len(raw))
	for _, r := range raw {
		net := groupNetPayout(&models.GroupOrder{Subtotal: r.Subtotal, Tax: r.Tax}, rate)
		rows = append(rows, pendingRow{
			ID: r.ID, ChefID: r.ChefID, Amount: r.Amount, NetPayout: net,
			PayoutHoldStatus: r.PayoutHoldStatus, DeliveredAt: r.DeliveredAt,
			CustomerConfirmedAt: r.CustomerConfirmedAt, Context: r.Context,
		})
	}
	return toPending(aggTypeGroupOrder, rows), nil
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
			AggType: aggType, ID: id, ChefID: chef, Amount: r.Amount, NetPayout: r.NetPayout,
			HoldStatus: models.PayoutHoldStatus(r.PayoutHoldStatus), DeliveredAt: r.DeliveredAt,
			AgeHours: age, CustomerConfirmedAt: r.CustomerConfirmedAt, Context: r.Context,
			HasOpenIssue: r.HasOpenIssue, // populated only for order rows; day/group default false
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
	case aggTypeGroupOrder:
		return &models.GroupOrder{}, nil
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
// orderRefundBlocks reports whether an underlying order forbids a payout release
// (#457): it is refunded/cancelled, has refunded_at stamped (the issue path leaves
// status='delivered' but sets refunded_at), or still has a pending OrderIssue.
func orderRefundBlocks(db *gorm.DB, orderID uuid.UUID) (bool, error) {
	var order models.Order
	if err := db.Select("status", "refunded_at").First(&order, "id = ?", orderID).Error; err != nil {
		// A missing / soft-deleted linked order has no refund to guard against — do
		// not block a legitimate release on an unloadable row (this is a backstop;
		// the five refund wiring sites already drove the hold at refund time).
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("payout-release: load order %s for release guard: %w", orderID, err)
	}
	if order.Status == models.OrderStatusRefunded || order.Status == models.OrderStatusCancelled {
		return true, nil
	}
	if order.RefundedAt != nil {
		return true, nil
	}
	return HasOpenOrderIssue(db, orderID), nil
}

// releaseBlockedForAgg resolves the underlying order for any aggregate and asks
// orderRefundBlocks (#457). A meal-plan-day / group-order whose OrderID is nil has
// no dispute source and is never blocked here.
func releaseBlockedForAgg(db *gorm.DB, aggType string, id uuid.UUID) (bool, error) {
	switch aggType {
	case aggTypeOrder:
		return orderRefundBlocks(db, id)
	case aggTypeMealPlanDay:
		var day models.MealPlanDay
		if err := db.Select("order_id").First(&day, "id = ?", id).Error; err != nil {
			return false, fmt.Errorf("payout-release: load day %s for release guard: %w", id, err)
		}
		if day.OrderID == nil {
			return false, nil
		}
		return orderRefundBlocks(db, *day.OrderID)
	case aggTypeGroupOrder:
		var g models.GroupOrder
		if err := db.Select("order_id").First(&g, "id = ?", id).Error; err != nil {
			return false, fmt.Errorf("payout-release: load group order %s for release guard: %w", id, err)
		}
		if g.OrderID == nil {
			return false, nil
		}
		return orderRefundBlocks(db, *g.OrderID)
	}
	return false, nil
}

// refundedOrderSubquery builds `SELECT 1 FROM orders WHERE id = ? AND (status IN
// ('refunded','cancelled') OR refunded_at IS NOT NULL)` for a NOT EXISTS predicate.
// Built via a NewDB Model session so (a) GORM applies the soft-delete scope
// (`deleted_at IS NULL`) — matching orderRefundBlocks' First()-based
// not-found→not-blocked parity, so a soft-deleted order never wrongly blocks — and
// (b) it can't bleed clauses from the outer UPDATE when that UPDATE is also on
// orders (the order aggregate). Mirrors openOrderIssueSubquery (#496).
func refundedOrderSubquery(tx *gorm.DB, orderID uuid.UUID) *gorm.DB {
	return tx.Session(&gorm.Session{NewDB: true}).
		Model(&models.Order{}).
		Select("1").
		Where("id = ? AND (status IN ? OR refunded_at IS NOT NULL)", orderID,
			[]models.OrderStatus{models.OrderStatusRefunded, models.OrderStatusCancelled})
}

// resolveReleaseOrderID returns the order whose refund/dispute state gates a release
// of the aggregate: the row itself for an order, the linked order_id for a
// meal-plan-day / group-order (nil when unlinked — nothing to guard against, and a
// missing aggregate row makes the release UPDATE a no-op anyway).
func resolveReleaseOrderID(tx *gorm.DB, aggType string, id uuid.UUID) (*uuid.UUID, error) {
	switch aggType {
	case aggTypeOrder:
		return &id, nil
	case aggTypeMealPlanDay:
		var day models.MealPlanDay
		if err := tx.Select("order_id").First(&day, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return day.OrderID, nil
	case aggTypeGroupOrder:
		var g models.GroupOrder
		if err := tx.Select("order_id").First(&g, "id = ?", id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		return g.OrderID, nil
	}
	return nil, nil
}

// releaseTransition performs release_eligible → released with the refund/dispute
// block folded INTO the guarded UPDATE (#496): a refund (refunded_at / cancelled /
// refunded status) or pending OrderIssue filed AFTER ReleaseHold's pre-check but
// BEFORE this flip is observed by the UPDATE's own NOT EXISTS predicates, closing
// the check-then-act window that could otherwise commit `released` on a
// just-refunded/disputed order (double-pay). Same fold-the-predicate approach as
// applyHoldConfirm (#460). The predicates key on the concrete linked order id
// (resolved in-tx; a stable FK) so the block is evaluated atomically with the flip.
func releaseTransition(db *gorm.DB, aggType string, id uuid.UUID) (bool, error) {
	model, err := holdModel(aggType)
	if err != nil {
		return false, err
	}
	changed := false
	err = db.Transaction(func(tx *gorm.DB) error {
		orderID, err := resolveReleaseOrderID(tx, aggType, id)
		if err != nil {
			return err
		}
		q := tx.Model(model).
			Where("id = ? AND payout_hold_status = ?", id, models.PayoutHoldReleaseEligible)
		if orderID != nil {
			q = q.Where("NOT EXISTS (?)", refundedOrderSubquery(tx, *orderID)).
				Where("NOT EXISTS (?)", openOrderIssueSubquery(tx, *orderID))
		}
		res := q.Update("payout_hold_status", models.PayoutHoldReleased)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		changed = true
		return EnqueueEvent(tx, SubjectHoldReleased, "payout.hold_released", id, map[string]any{
			"aggregate_type": aggType, "aggregate_id": id.String(),
			"payout_hold_status": string(models.PayoutHoldReleased),
		})
	})
	if err != nil {
		return false, err
	}
	return changed, nil
}

func ReleaseHold(db *gorm.DB, aggType string, id uuid.UUID) error {
	// Cross-guard (#457): refuse to release any aggregate whose underlying order is
	// refunded/cancelled/refunded_at-set/pending-issue — the fast-fail pre-check.
	blocked, err := releaseBlockedForAgg(db, aggType, id)
	if err != nil {
		return err
	}
	if blocked {
		return ErrHoldNotEligible
	}
	// The block is ALSO folded into the guarded UPDATE (#496), so a refund/issue that
	// lands after the pre-check above but before the flip cannot slip a `released`
	// commit through the check-then-act window.
	ok, err := releaseTransition(db, aggType, id)
	if err != nil {
		return fmt.Errorf("payout-release: release %s %s: %w", aggType, id, err)
	}
	if !ok {
		return ErrHoldNotEligible
	}
	return settleRelease(db, aggType, id)
}

// stampPayoutSettled marks that the money seam completed for a hold (#459): a
// conditional UPDATE `SET payout_settled_at = now WHERE id = ? AND
// payout_settled_at IS NULL`. Idempotent — a row already settled is left
// untouched, so a re-drive never moves the original stamp. This is the single
// source of truth for "seam succeeded → mark settled", reused by both the primary
// ReleaseHold/ReverseHold path and the payout-reconcile cron (no duplicated logic).
func stampPayoutSettled(db *gorm.DB, aggType string, id uuid.UUID) error {
	model, err := holdModel(aggType)
	if err != nil {
		return err
	}
	res := db.Model(model).
		Where("id = ? AND payout_settled_at IS NULL", id).
		Update("payout_settled_at", time.Now())
	if res.Error != nil {
		return fmt.Errorf("payout-release: stamp settled %s %s: %w", aggType, id, res.Error)
	}
	return nil
}

// settlePayout is the single money-seam dispatcher (#508). It RE-READS the hold's
// CURRENT terminal status and runs the seam that matches THAT status — not the
// transition the calling goroutine happened to win. This closes the check-then-act
// window where ReleaseHold committed `released` and dispatched releaseMoney while a
// concurrent refund had since flipped the row to `reversed` (chef paid after the
// claw-back). By re-reading, a goroutine that lost the race observes `reversed` and
// runs reverseMoney instead of releaseMoney — no chef payment survives a claw-back.
//
// Idempotent: a row already settled (payout_settled_at set) is skipped, and the
// underlying seams tolerate an already-applied gateway op, so a residual concurrent
// double-dispatch is a safe no-op. On seam failure settled_at stays NULL and the
// error propagates — the drift the payout-reconcile cron re-drives (which now also
// dispatches through settlePayout, so it too acts on the fresh status).
func settlePayout(db *gorm.DB, aggType string, id uuid.UUID) error {
	model, err := holdModel(aggType)
	if err != nil {
		return err
	}
	var rows []struct {
		PayoutHoldStatus models.PayoutHoldStatus
		PayoutSettledAt  *time.Time
	}
	if err := db.Model(model).Select("payout_hold_status", "payout_settled_at").
		Where("id = ?", id).Scan(&rows).Error; err != nil {
		return fmt.Errorf("payout-release: settle re-read %s %s: %w", aggType, id, err)
	}
	if len(rows) == 0 || rows[0].PayoutSettledAt != nil {
		return nil // gone, or already settled — nothing to move
	}
	switch rows[0].PayoutHoldStatus {
	case models.PayoutHoldReleased:
		if err := releaseMoney(db, aggType, id); err != nil {
			return err
		}
		return stampPayoutSettled(db, aggType, id)
	case models.PayoutHoldReversed:
		if err := reverseMoney(db, aggType, id); err != nil {
			return err
		}
		return stampPayoutSettled(db, aggType, id)
	}
	return nil // not a terminal money state (a concurrent path moved it) — nothing to settle
}

// settleRelease and settleReverse are thin, backward-compatible wrappers over
// settlePayout so the ReleaseHold / ReverseHold / crossguard / reconcile call sites
// stay unchanged. Both dispatch by the FRESH status, so which wrapper a caller uses
// no longer decides which seam runs — the row's current state does (#508).
func settleRelease(db *gorm.DB, aggType string, id uuid.UUID) error {
	return settlePayout(db, aggType, id)
}

func settleReverse(db *gorm.DB, aggType string, id uuid.UUID) error {
	return settlePayout(db, aggType, id)
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
	case aggTypeGroupOrder:
		var g models.GroupOrder
		if err := db.First(&g, "id = ?", id).Error; err != nil {
			return fmt.Errorf("payout-release: load group order %s: %w", id, err)
		}
		if err := ReleaseGroupChefPayout(&g); err != nil {
			return fmt.Errorf("payout-release: release group payout %s: %w", id, err)
		}
	}
	return nil
}

// WithholdOrReverseOrderHoldForRefund cross-guards an order refund against the
// payout hold (#457): whatever refund path fired, the chef must not keep money the
// customer got back. It drives ONLY the order aggregate:
//   - release_eligible | awaiting_customer_confirmation | disputed → withheld
//     (block the payout). disputed is included so an approved assisted-refund on a
//     disputed order leaves the hold blocked, not stuck in the dead-end (#458).
//   - released → reversed, then settleReverse claws the transfer back (state-only
//     while the escrow flag is OFF; stampPayoutSettled keeps the reconcile safe).
//   - none | withheld | reversed → no-op.
//
// Idempotent + race-safe: transitionHold's conditional `WHERE payout_hold_status
// IN (from)` means a double refund never double-reverses. Best-effort audit on a
// real transition; callers invoke this best-effort so a hold-drive failure never
// fails the refund (the release-side guard is the backstop).
func WithholdOrReverseOrderHoldForRefund(db *gorm.DB, orderID uuid.UUID, reason string) error {
	// Drive the order aggregate itself, then fan out (#498) to any meal-plan-day /
	// group-order carrying its hold on the linked order: each per-day fulfillment
	// order maps to exactly one day and each consolidated order to one group, and a
	// standalone food order has zero linked rows (no-op). All three key on the SAME
	// order refund, so the refund must withhold/reverse them together.
	if err := withholdOrReverseHoldForRefund(db, aggTypeOrder, orderID, reason); err != nil {
		return err
	}
	var dayIDs []uuid.UUID
	if err := db.Model(&models.MealPlanDay{}).Where("order_id = ?", orderID).Pluck("id", &dayIDs).Error; err != nil {
		return fmt.Errorf("payout-crossguard: list days for order %s: %w", orderID, err)
	}
	for _, id := range dayIDs {
		if err := withholdOrReverseHoldForRefund(db, aggTypeMealPlanDay, id, reason); err != nil {
			return err
		}
	}
	var groupIDs []uuid.UUID
	if err := db.Model(&models.GroupOrder{}).Where("order_id = ?", orderID).Pluck("id", &groupIDs).Error; err != nil {
		return fmt.Errorf("payout-crossguard: list groups for order %s: %w", orderID, err)
	}
	for _, id := range groupIDs {
		if err := withholdOrReverseHoldForRefund(db, aggTypeGroupOrder, id, reason); err != nil {
			return err
		}
	}
	return nil
}

// withholdOrReverseHoldForRefund drives ONE aggregate's payout hold on refund and is
// the aggType-generic core shared by the order entry point and its day/group fan-out:
// release_eligible | awaiting | disputed → withheld; released → reversed (then
// settleReverse claws the transfer back — state-only while the escrow flag is OFF).
// none | withheld | reversed → no-op. Idempotent + race-safe via transitionHold's
// conditional WHERE. Best-effort audit on a genuine transition.
func withholdOrReverseHoldForRefund(db *gorm.DB, aggType string, id uuid.UUID, reason string) error {
	model, err := holdModel(aggType)
	if err != nil {
		return err
	}
	var statuses []models.PayoutHoldStatus
	if err := db.Model(model).Where("id = ?", id).Pluck("payout_hold_status", &statuses).Error; err != nil {
		return fmt.Errorf("payout-crossguard: load %s %s: %w", aggType, id, err)
	}
	if len(statuses) == 0 {
		return nil // aggregate gone — nothing to cross-guard
	}
	old := statuses[0]
	var to models.PayoutHoldStatus
	var ok bool
	switch old {
	case models.PayoutHoldReleaseEligible, models.PayoutHoldAwaitingConfirmation, models.PayoutHoldDisputed:
		to = models.PayoutHoldWithheld
		ok, err = transitionHold(db, aggType, id,
			[]models.PayoutHoldStatus{models.PayoutHoldReleaseEligible, models.PayoutHoldAwaitingConfirmation, models.PayoutHoldDisputed}, to, false)
	case models.PayoutHoldReleased:
		to = models.PayoutHoldReversed
		if ok, err = transitionHold(db, aggType, id,
			[]models.PayoutHoldStatus{models.PayoutHoldReleased}, to, false); ok && err == nil {
			err = settleReverse(db, aggType, id)
		}
	default:
		return nil // none / withheld / reversed — nothing to cross-guard
	}
	if err != nil {
		return fmt.Errorf("payout-crossguard: drive %s %s refund hold: %w", aggType, id, err)
	}
	if ok {
		LogSystemAudit(nil, "payout.hold.refund_crossguard", aggType, id.String(),
			string(old), map[string]any{"to": string(to), "reason": reason})
	}
	return nil
}

// parkedDayHolds is the set of not-yet-released day holds RefundDay may act on: the
// on-hold transfer exists but the chef hasn't been paid out yet.
var parkedDayHolds = []models.PayoutHoldStatus{
	models.PayoutHoldReleaseEligible, models.PayoutHoldAwaitingConfirmation, models.PayoutHoldDisputed,
}

// reverseRefundedDayHold drives a meal-plan-day's payout hold out of the releasable
// set after RefundDay has refunded the customer AND attempted the gateway claw-back
// of any held transfer (#498 Part C; #398). STATE-ONLY: it never runs reverseMoney
// itself — RefundDay already issued the ReverseTransfer — it only records the terminal
// state that matches whether that claw-back LANDED (reverseOK).
//
//   - reverseOK (claw-back succeeded, or there was no transfer): released → reversed
//     + payout_settled_at stamped (terminal — the reconcile cron won't re-reverse an
//     already-clawed-back transfer); eligible/awaiting/disputed → withheld (the
//     on-hold transfer was freed).
//   - !reverseOK (claw-back FAILED — the transfer is still live at the gateway): the
//     day is left as RE-DRIVABLE DRIFT — released AND parked both → reversed with
//     settled_at LEFT NULL — so reconcileMealPlanDays(reversed, settleReverse) retries
//     the ReverseTransfer (attempt-capped + ALERT) instead of silently stranding the
//     chef's held payout for a refunded day (#398 core defect: the old code stamped
//     settled regardless of gateway success and hid the strand from every sweep).
//
// none/withheld/reversed → no-op.
func reverseRefundedDayHold(tx *gorm.DB, dayID uuid.UUID, reverseOK bool) error {
	res := tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status = ?", dayID, models.PayoutHoldReleased).
		Update("payout_hold_status", models.PayoutHoldReversed)
	if res.Error != nil {
		return fmt.Errorf("payout-release: mark refunded day %s reversed: %w", dayID, res.Error)
	}
	if res.RowsAffected > 0 {
		if reverseOK {
			return stampPayoutSettled(tx, aggTypeMealPlanDay, dayID)
		}
		return nil // failed claw-back → reversed + unsettled drift; reconcile re-drives
	}
	// Parked (not yet released): a successful reverse freed the on-hold transfer →
	// withheld; a failed reverse leaves the transfer live → reversed drift so the
	// reconcile retries the claw-back (never withheld, which would hide it).
	to := models.PayoutHoldWithheld
	if !reverseOK {
		to = models.PayoutHoldReversed
	}
	res = tx.Model(&models.MealPlanDay{}).
		Where("id = ? AND payout_hold_status IN ?", dayID, parkedDayHolds).
		Update("payout_hold_status", to)
	if res.Error != nil {
		return fmt.Errorf("payout-release: mark refunded day %s %s: %w", dayID, to, res.Error)
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
	return settleReverse(db, aggType, id)
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
			if _, err := GetRazorpay().ReverseTransfer(day.PayoutTransferID, 0); err != nil && !isAlreadyReversedErr(err) {
				return fmt.Errorf("payout-release: reverse day transfer %s: %w", day.PayoutTransferID, err)
			}
		}
	case aggTypeGroupOrder:
		var g models.GroupOrder
		if err := db.First(&g, "id = ?", id).Error; err != nil {
			return fmt.Errorf("payout-release: load group order %s: %w", id, err)
		}
		if err := ReverseGroupChefPayout(&g); err != nil { // flag-gated claw-back
			return fmt.Errorf("payout-release: reverse group payout %s: %w", id, err)
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
