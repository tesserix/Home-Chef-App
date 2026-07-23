package services

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/payouts"
)

// payout_release_cron.go — the 15-minute release sweep (#741).
//
// Route creates each order's payee transfers held at checkout. This finds
// delivered orders whose hold has matured, asks the governor whether they may
// be released, and clears the ones that may. Blocked orders keep their
// transfers held and surface in the admin queue with every reason.
//
// A sweep rather than a per-order timer: the worst-case lag is one interval,
// which is immaterial against Razorpay's fixed 2-working-day settlement, and a
// missed tick self-heals on the next one with no per-order state to reconcile.

const payoutReleaseInterval = 15 * time.Minute

// defaultMaturation is used when payout.maturation_minutes is unset or
// unparseable. Never zero — that would release the instant an order is
// delivered, removing the window a late refund needs.
const defaultMaturation = 2 * time.Hour

// maturationWindow reads the configured hold, falling back on anything invalid.
func maturationWindow(db *gorm.DB) time.Duration {
	mins, err := strconv.Atoi(settingValue(db, "payout.maturation_minutes"))
	if err != nil || mins <= 0 {
		return defaultMaturation
	}
	return time.Duration(mins) * time.Minute
}

// defaultReviewThresholdPaise / defaultRampOrders are the safe-ON fallbacks for
// payout.review_above_paise / payout.new_chef_ramp_orders — the plan's
// documented defaults (₹5,000 / 3 orders). Zero means "guardrail disabled", so
// falling back to zero on an unset or unparseable setting would silently strip
// the guardrail out; falling back to these instead keeps it on until an admin
// deliberately sets 0. Only an explicit, successfully-parsed 0 disables either
// check.
const (
	defaultReviewThresholdPaise = 500000
	defaultRampOrders           = 3
)

// reviewThreshold reads the value above which a human looks. Explicit 0
// disables the check; unset or unparseable fails safe to the default (never
// silently disabling a guardrail an admin never touched).
func reviewThreshold(db *gorm.DB) payouts.Money {
	raw := settingValue(db, "payout.review_above_paise")
	if raw == "" {
		return payouts.Money{Minor: defaultReviewThresholdPaise, Currency: payouts.CurrencyINR}
	}
	paise, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || paise < 0 {
		log.Printf("payout-sweep: payout.review_above_paise=%q is invalid, falling back to the %dp default", raw, defaultReviewThresholdPaise)
		return payouts.Money{Minor: defaultReviewThresholdPaise, Currency: payouts.CurrencyINR}
	}
	return payouts.Money{Minor: paise, Currency: payouts.CurrencyINR}
}

// rampOrders reads the new-chef review period (orders held regardless of other
// checks). Explicit 0 disables it; unset or unparseable fails safe to the
// default for the same reason as reviewThreshold.
func rampOrders(db *gorm.DB) int {
	raw := settingValue(db, "payout.new_chef_ramp_orders")
	if raw == "" {
		return defaultRampOrders
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		log.Printf("payout-sweep: payout.new_chef_ramp_orders=%q is invalid, falling back to the %d-order default", raw, defaultRampOrders)
		return defaultRampOrders
	}
	return n
}

// deliveredAt reads the order's delivery timestamp. A delivered order should
// always carry one (stamped at the delivered transition); a nil value here is
// data corruption rather than "delivered long ago", so it maps to "just now"
// rather than the zero time — the zero time is centuries in the past and would
// read as trivially matured, releasing money against a broken row. Mapping to
// "now" instead makes the row look freshly delivered, so the maturation window
// blocks it until the data is fixed.
//
// Takes now from the caller rather than reading time.Now() itself — the same
// deterministic clock BuildReleaseInput already threads through everything
// else — so the nil-fallback path is exercisable from a test.
func deliveredAt(order *models.Order, now time.Time) time.Time {
	if order.DeliveredAt == nil {
		return now
	}
	return *order.DeliveredAt
}

// BuildReleaseInput maps one order onto the governor's inputs.
func BuildReleaseInput(db *gorm.DB, order *models.Order, now time.Time) (payouts.ReleaseInput, error) {
	// Count the chef's PRIOR delivered orders only — the sweep only ever
	// evaluates rows already status=delivered, so counting unconditionally would
	// include this order in its own tally and let the Nth order in the ramp
	// (DeliveredOrderCount == RampOrders) auto-release a step early.
	var delivered int64
	if err := db.Model(&models.Order{}).
		Where("chef_id = ? AND status = ? AND id <> ?", order.ChefID, models.OrderStatusDelivered, order.ID).
		Count(&delivered).Error; err != nil {
		return payouts.ReleaseInput{}, err
	}

	// LANDMINE: this is one of TWO uncoordinated places that act on the same
	// non-discharging recovery debt (services/payout_recovery.go — nothing
	// anywhere writes a resolving ledger entry, so ApplyRecoveryDeduction
	// re-derives the SAME full outstanding balance every time it is called).
	// The other is handlers/payment.go's applyChefRecoveryDeduction, which
	// already reduced this same chef's gross transfer for a PRIOR order at
	// checkout. Neither site knows the other exists or discharges anything, so
	// today the debt can be independently deducted-from-transfer here-and-there
	// AND block-release here, against the identical outstanding figure. No
	// penalty/ledger writer may ship until exactly one of these two mechanisms
	// actually collects-and-discharges the debt and the other defers to it —
	// do not add a third site, and do not wire a discharging writer to only one
	// of the two without also fixing the other.
	gross := payouts.Money{Minor: int64(ToPaise(ChefNetPayoutFor(order))), Currency: payouts.CurrencyINR}
	_, deducted, err := ApplyRecoveryDeduction(db, order.ChefID, gross, now)
	if err != nil {
		return payouts.ReleaseInput{}, err
	}

	return payouts.ReleaseInput{
		Now:               now,
		DeliveredAt:       deliveredAt(order, now),
		Maturation:        maturationWindow(db),
		AutomationEnabled: PayoutAutomationEnabled(db, &order.Chef),
		// Both must hold: Route only ever settles by NEFT/IMPS to a bank
		// account, so "activated" alone is not enough — a chef who switches
		// to UPI keeps whatever razorpay_settlement_status was last written
		// (SavePayoutDetails now clears it on that switch, but this is the
		// belt-and-suspenders read for any row that predates that fix or any
		// gap in it). Trusting status alone would let the sweep auto-release
		// money toward a bank account the chef has already abandoned (review
		// finding 2).
		SettlementActivated: order.Chef.RazorpaySettlementStatus == "activated" && order.Chef.PayoutMethod == "bank_transfer",
		RefundOpen:          order.RefundedAt != nil,
		RecoveryBalance:     deducted,
		DeliveredOrderCount: int(delivered),
		RampOrders:          rampOrders(db),
		OrderTotal:          payouts.Money{Minor: int64(ToPaise(order.Total)), Currency: payouts.CurrencyINR},
		ReviewAbove:         reviewThreshold(db),
	}, nil
}

// runPayoutReleaseSweep releases every matured, unblocked delivered order.
func runPayoutReleaseSweep(ctx context.Context) {
	db := database.DB

	// payout.sweep_enabled is the master kill switch (services/payout_automation.go)
	// and ships OFF. Bail here, before the per-order query and loop, rather than
	// leaving the gate to PayoutAutomationEnabled inside the per-order decision:
	// with the switch off, EVERY delivered, release-eligible order would land
	// BlockAutomationOff as its sole reason, and recordPayoutBlock's noise guard
	// only silences a sole BlockNotMatured — so every one of those orders would
	// publish payments.payout_blocked, with a fresh msg_id (no JetStream dedup),
	// on every 15-minute tick, indefinitely, before any admin has opted in.
	if !settingBool(db, payoutSweepEnabledKey) {
		log.Println("payout-sweep: payout.sweep_enabled is off — skipping this sweep")
		return
	}

	// payout.sweep_enabled being on is not the whole gate: this loop only ever
	// releases aggTypeOrder holds (the query below is orders-only), and
	// releaseMoney's order branch (ReleaseOrderPayouts, services/order_payout.go)
	// is itself gated on payoutMovementEnabled() alone — a meal-plan-day or
	// group-order hold is never touched by this sweep, so MealPlanEscrowActive()
	// is not this guard's concern (contrast payout_reconcile_cron.go's
	// runPayoutReconcileScan, which re-drives every aggregate kind and so checks
	// both flags). Without this check, ReleaseHold still commits the
	// release_eligible -> released transition and settleRelease still stamps
	// payout_settled_at, because ReleaseOrderPayouts silently returns nil when
	// movement is off (a deliberate no-op for its OTHER callers, the admin queue
	// with the flag on) — leaving the row looking fully settled while the
	// Razorpay transfer stays on_hold, permanently excluded from the reconcile
	// cron's own `released AND payout_settled_at IS NULL` query. Bail here,
	// before that flip, not after.
	if !payoutMovementEnabled() {
		log.Println("payout-sweep: payout movement (ORDER_PAYOUT_AUTO_RELEASE_ENABLED) is off — skipping this sweep")
		return
	}

	now := time.Now()

	// payout_hold_status = release_eligible is the hold state machine's own gate
	// (services/payout_hold.go's KEY INVARIANT: a disputed or unconfirmed hold
	// must NEVER reach release_eligible) — status=delivered alone is not enough,
	// since a delivered order sits in awaiting_customer_confirmation (or
	// disputed) until the customer confirms. payout_settled_at IS NULL is a
	// belt-and-suspenders guard: once ReleaseHold settles a row it also flips
	// payout_hold_status away from release_eligible, so this clause should
	// already be redundant in practice.
	var orders []models.Order
	if err := db.Preload("Chef").
		Where("status = ? AND payout_hold_status = ? AND payout_settled_at IS NULL",
			models.OrderStatusDelivered, models.PayoutHoldReleaseEligible).
		Limit(sweepBatchLimit).Find(&orders).Error; err != nil {
		log.Printf("payout-sweep: load orders: %v", err)
		return
	}

	for i := range orders {
		order := &orders[i]
		in, err := BuildReleaseInput(db, order, now)
		if err != nil {
			log.Printf("payout-sweep: build input order=%s: %v", order.OrderNumber, err)
			continue
		}
		decision := payouts.DecideRelease(in)
		if !decision.Release {
			recordPayoutBlock(db, order, decision)
			continue
		}
		// Drive the transition through the hold state machine rather than
		// moving money directly: ReleaseHold re-checks the refund/dispute
		// cross-guards atomically with the release_eligible -> released flip
		// (closing the window this governor's own, narrower RefundOpen check
		// leaves open), and its settleRelease stamps payout_settled_at so this
		// row drops out of the query above and is never re-picked.
		if err := ReleaseHold(db, aggTypeOrder, order.ID); err != nil {
			if errors.Is(err, ErrHoldNotEligible) {
				// A guard refused (e.g. a dispute or refund landed on this order
				// after it was loaded above) — an ordinary block, not a sweep
				// failure. No release event; the next sweep re-evaluates.
				continue
			}
			// Any other error: if the release_eligible -> released flip itself
			// never committed, this row is untouched and this sweep's own query
			// picks it up again next tick. But if the flip DID commit and it was
			// the post-commit money seam (settleRelease) that failed, the row now
			// sits at payout_hold_status=released with payout_settled_at NULL —
			// excluded from THIS sweep's release_eligible query from here on. That
			// drift is re-driven by the separate payout-reconcile cron
			// (reconcileOrders(models.PayoutHoldReleased, settleRelease) in
			// services/payout_reconcile_cron.go), not by another run of this sweep.
			log.Printf("payout-sweep: release order=%s: %v", order.OrderNumber, err)
			continue
		}
		if err := EnqueueOutbox(db, SubjectPayoutReleased, "order", order.ID.String(), map[string]any{
			"orderId":     order.ID,
			"orderNumber": order.OrderNumber,
			"chefId":      order.ChefID,
		}); err != nil {
			log.Printf("payout-sweep: enqueue release event order=%s: %v", order.OrderNumber, err)
		}
	}
}

// StartPayoutReleaseCron is the legacy in-process fallback ticker.
func StartPayoutReleaseCron(ctx context.Context) {
	go func() {
		t := time.NewTicker(payoutReleaseInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runPayoutReleaseSweep(ctx)
			}
		}
	}()
}

// NATS subjects for payout movement. Product-scoped under payments.* to match
// the existing subject taxonomy (services/nats.go), and covered by the
// existing PAYMENTS stream's "payments.>" subject filter — no stream change
// needed.
const (
	SubjectPayoutReleased = "payments.payout_released"
	SubjectPayoutBlocked  = "payments.payout_blocked"
)

// PayoutBlockEvent is what the admin surface and analytics consume.
type PayoutBlockEvent struct {
	OrderID     uuid.UUID `json:"orderId"`
	OrderNumber string    `json:"orderNumber"`
	ChefID      uuid.UUID `json:"chefId"`
	Reasons     []string  `json:"reasons"`
	// Overridable is false when any reason is a hard block, so the admin UI can
	// disable the release action rather than rejecting it on click.
	Overridable bool `json:"overridable"`
}

// recordPayoutBlock publishes why an order was withheld.
//
// Through the transactional outbox rather than a direct NATS publish, so the
// event cannot be lost if the process dies between deciding and publishing —
// the same guarantee the rest of the money path relies on. OutboxRelay
// publishes to JetStream with PubAck and Nats-Msg-Id dedup, so a redelivery
// after a crash is at-least-once at the transport and effectively-once here.
// payoutBlockNoiseReasons are block reasons that are expected to hold for a
// long or indefinite time and carry nothing an admin can act on: a still-
// maturing order, or a chef who simply hasn't been opted into automation yet
// (the normal state for most chefs during the intended opt-in rollout —
// sweep_enabled on, auto_release_default unset/off, PayoutAutoRelease==""). A
// block is silenced only when EVERY reason it carries is in this set; any
// reason outside it is actionable (settlement not activated, an open refund,
// a recovery balance, the new-chef ramp, above the review threshold) and must
// always surface, even alongside a suppressed reason.
var payoutBlockNoiseReasons = map[payouts.BlockReason]bool{
	payouts.BlockNotMatured:     true,
	payouts.BlockAutomationOff: true,
}

// isPayoutBlockNoise reports whether every reason in the decision is one this
// package treats as expected/non-actionable noise, not worth publishing.
func isPayoutBlockNoise(reasons []payouts.BlockReason) bool {
	for _, r := range reasons {
		if !payoutBlockNoiseReasons[r] {
			return false
		}
	}
	return true
}

func recordPayoutBlock(db *gorm.DB, order *models.Order, decision payouts.ReleaseDecision) {
	// A still-maturing order, or a not-yet-enrolled chef, is the common case
	// during rollout, not a problem — publishing either every 15 minutes would
	// drown the queue in noise. Only silence a block whose reasons are ENTIRELY
	// within that persistent, non-actionable set; a genuinely actionable reason
	// alongside them must still surface.
	if isPayoutBlockNoise(decision.Reasons) {
		return
	}

	reasons := make([]string, 0, len(decision.Reasons))
	overridable := true
	for _, r := range decision.Reasons {
		reasons = append(reasons, string(r))
		if !r.Overridable() {
			overridable = false
		}
	}

	if err := EnqueueOutbox(db, SubjectPayoutBlocked, "order", order.ID.String(), PayoutBlockEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		ChefID:      order.ChefID,
		Reasons:     reasons,
		Overridable: overridable,
	}); err != nil {
		log.Printf("payout-sweep: enqueue block event order=%s: %v", order.OrderNumber, err)
	}
}
