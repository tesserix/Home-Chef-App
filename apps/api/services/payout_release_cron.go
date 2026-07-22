package services

import (
	"context"
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

// reviewThreshold reads the value above which a human looks; zero disables it.
func reviewThreshold(db *gorm.DB) payouts.Money {
	paise, err := strconv.ParseInt(settingValue(db, "payout.review_above_paise"), 10, 64)
	if err != nil || paise < 0 {
		return payouts.Zero(payouts.CurrencyINR)
	}
	return payouts.Money{Minor: paise, Currency: payouts.CurrencyINR}
}

// rampOrders reads the new-chef review period; zero disables it.
func rampOrders(db *gorm.DB) int {
	n, err := strconv.Atoi(settingValue(db, "payout.new_chef_ramp_orders"))
	if err != nil || n < 0 {
		return 0
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
func deliveredAt(order *models.Order) time.Time {
	if order.DeliveredAt == nil {
		return time.Now()
	}
	return *order.DeliveredAt
}

// BuildReleaseInput maps one order onto the governor's inputs.
func BuildReleaseInput(db *gorm.DB, order *models.Order, now time.Time) (payouts.ReleaseInput, error) {
	var delivered int64
	if err := db.Model(&models.Order{}).
		Where("chef_id = ? AND status = ?", order.ChefID, models.OrderStatusDelivered).
		Count(&delivered).Error; err != nil {
		return payouts.ReleaseInput{}, err
	}

	gross := payouts.Money{Minor: int64(ToPaise(ChefNetPayoutFor(order))), Currency: payouts.CurrencyINR}
	_, deducted, err := ApplyRecoveryDeduction(db, order.ChefID, gross, now)
	if err != nil {
		return payouts.ReleaseInput{}, err
	}

	return payouts.ReleaseInput{
		Now:                 now,
		DeliveredAt:         deliveredAt(order),
		Maturation:          maturationWindow(db),
		AutomationEnabled:   PayoutAutomationEnabled(db, &order.Chef),
		SettlementActivated: order.Chef.RazorpaySettlementStatus == "activated",
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
	now := time.Now()
	db := database.DB

	var orders []models.Order
	if err := db.Preload("Chef").
		Where("status = ? AND payout_settled_at IS NULL", models.OrderStatusDelivered).
		Limit(500).Find(&orders).Error; err != nil {
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
		if err := ReleaseOrderPayouts(order.ID); err != nil {
			// Returned so the Temporal activity retries; the next sweep
			// re-drives it either way.
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
func recordPayoutBlock(db *gorm.DB, order *models.Order, decision payouts.ReleaseDecision) {
	// A still-maturing order is the common case, not a problem — publishing it
	// every 15 minutes would drown the queue in noise. Only silence the SOLE
	// not_matured reason; a real block alongside it must still surface.
	if len(decision.Reasons) == 1 && decision.Reasons[0] == payouts.BlockNotMatured {
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
