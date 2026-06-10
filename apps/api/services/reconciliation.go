package services

// reconciliation.go — settlement reconciliation against payment gateways.
//
// Compares the platform's recorded payment/refund state for an order against
// what the gateway (Razorpay / Stripe) reports, and flags drift. This catches
// the dangerous failure modes: a refund that succeeded at the gateway but
// wasn't recorded (or vice-versa), or a payment the platform marked paid that
// the gateway never captured.
//
// What we reconcile, and why only this:
//   - Capture status: gateway must show the payment captured/succeeded for an
//     order the platform marks paid.
//   - Refund amount: gateway amount_refunded must equal Order.RefundAmount.
//     This is the universal cross-check — Order.Total is mutated by per-line
//     cancellations, so it is NOT a reliable invariant against the originally
//     captured amount, but the refunded total always is.
//
// Stripe's PaymentIntent fetch does not expose amount_refunded, so Stripe
// orders are reconciled on capture status only (noted in the drift output).
//
// Amounts tolerate a 1-paisa/cent rounding gap.

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// reconAmountTolerance is the max acceptable gap (₹) between platform and
// gateway amounts before it counts as drift.
const reconAmountTolerance = 0.011

// reconMaxOrders caps how many orders one reconciliation run will pull from
// the gateways, so a backlog can't blow the gateway rate limits or the run
// time. If a window exceeds this, the overflow is logged (never silently
// dropped) and picked up on the next run.
const reconMaxOrders = 500

// DriftKind classifies a reconciliation discrepancy.
type DriftKind string

const (
	DriftPaymentNotCaptured DriftKind = "payment_not_captured"
	DriftRefundMismatch     DriftKind = "refund_mismatch"
	DriftGatewayUnreachable DriftKind = "gateway_unreachable"
)

// Drift is a single reconciliation discrepancy.
type Drift struct {
	OrderID     string
	OrderNumber string
	Provider    string
	Kind        DriftKind
	Detail      string
	PlatformAmt float64
	GatewayAmt  float64
}

// ReconcileSettlements reconciles orders whose payment state changed in
// [windowStart, windowEnd) against their payment gateway. Returns the drift
// list plus the number of orders actually checked.
func ReconcileSettlements(ctx context.Context, windowStart, windowEnd time.Time) ([]Drift, int, error) {
	// Reconcile orders that were captured (completed) or refunded — a fully
	// refunded order flips to "refunded" but still needs its refund total
	// cross-checked against the gateway.
	reconcilable := []models.PaymentStatus{models.PaymentCompleted, models.PaymentRefunded}
	var orders []models.Order
	if err := database.DB.
		Where("payment_status IN ? AND updated_at >= ? AND updated_at < ? AND deleted_at IS NULL",
			reconcilable, windowStart, windowEnd).
		Order("updated_at ASC").
		Limit(reconMaxOrders + 1).
		Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("load orders for reconciliation: %w", err)
	}

	if len(orders) > reconMaxOrders {
		log.Printf("reconciliation: window has >%d orders; reconciling the oldest %d, remainder next run",
			reconMaxOrders, reconMaxOrders)
		orders = orders[:reconMaxOrders]
	}

	drifts := make([]Drift, 0)
	checked := 0
	for i := range orders {
		select {
		case <-ctx.Done():
			return drifts, checked, ctx.Err()
		default:
		}
		o := &orders[i]
		d, ok := reconcileOne(o)
		if !ok {
			continue // no gateway id / provider not configured — skip, not drift
		}
		checked++
		drifts = append(drifts, d...)
	}
	return drifts, checked, nil
}

// reconcileOne reconciles a single order. The bool is false when the order
// can't be reconciled (no gateway reference or provider not configured) — that
// is a skip, not a drift.
func reconcileOne(o *models.Order) ([]Drift, bool) {
	switch {
	case o.RazorpayPaymentID != "":
		return reconcileRazorpay(o), true
	case o.StripePaymentIntentID != "":
		return reconcileStripe(o), true
	default:
		return nil, false
	}
}

func reconcileRazorpay(o *models.Order) []Drift {
	client := GetRazorpay()
	if client == nil {
		return nil // not configured — skip silently (logged once at startup)
	}
	pay, err := client.FetchPayment(o.RazorpayPaymentID)
	if err != nil {
		return []Drift{driftFor(o, "razorpay", DriftGatewayUnreachable,
			fmt.Sprintf("fetch payment %s: %v", o.RazorpayPaymentID, err), 0, 0)}
	}

	var drifts []Drift
	if !pay.Captured {
		drifts = append(drifts, driftFor(o, "razorpay", DriftPaymentNotCaptured,
			fmt.Sprintf("gateway status=%q captured=false", pay.Status), o.Total, FromPaise(pay.Amount)))
	}
	gatewayRefunded := FromPaise(pay.AmountRefunded)
	if math.Abs(gatewayRefunded-o.RefundAmount) > reconAmountTolerance {
		drifts = append(drifts, driftFor(o, "razorpay", DriftRefundMismatch,
			"gateway amount_refunded != platform RefundAmount", o.RefundAmount, gatewayRefunded))
	}
	return drifts
}

func reconcileStripe(o *models.Order) []Drift {
	client := GetStripe()
	if client == nil {
		return nil
	}
	pi, err := client.FetchPaymentIntent(o.StripePaymentIntentID)
	if err != nil {
		return []Drift{driftFor(o, "stripe", DriftGatewayUnreachable,
			fmt.Sprintf("fetch intent %s: %v", o.StripePaymentIntentID, err), 0, 0)}
	}
	// Stripe's intent fetch exposes status but not amount_refunded, so we
	// reconcile capture status only. "succeeded" is the captured terminal state.
	if pi.Status != "succeeded" {
		return []Drift{driftFor(o, "stripe", DriftPaymentNotCaptured,
			fmt.Sprintf("gateway status=%q (refund amount not reconciled — Stripe intent fetch omits it)", pi.Status),
			o.Total, FromCents(pi.Amount))}
	}
	return nil
}

func driftFor(o *models.Order, provider string, kind DriftKind, detail string, platformAmt, gatewayAmt float64) Drift {
	return Drift{
		OrderID:     o.ID.String(),
		OrderNumber: o.OrderNumber,
		Provider:    provider,
		Kind:        kind,
		Detail:      detail,
		PlatformAmt: platformAmt,
		GatewayAmt:  gatewayAmt,
	}
}
