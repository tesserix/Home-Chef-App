package services

// reconciliation.go — settlement reconciliation against payment gateways.
//
// Compares the platform's recorded payment/refund state for an order against
// what the gateway (Cashfree / Stripe) reports, and flags drift. This catches
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
	// DriftFullRefundUnstamped: the gateway's cumulative amount_refunded reaches the
	// captured total, but the order's refunded_at is still NULL — so the payout release
	// guard never blocks it (#640). Arises when a refund is full only in AGGREGATE across
	// channels (e.g. an in-app partial + an out-of-band gateway-dashboard refund), which
	// no single app event stamps. The reconcile cron self-heals this via
	// FinalizeGatewayFullRefund; the drift is still alerted so ops can reconcile books.
	DriftFullRefundUnstamped DriftKind = "full_refund_unstamped"
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

// Overridden in tests to assert which credential slot an order is reconciled
// against without reaching Secret Manager.
var cashfreeClientFor = GetCashfreeFor

// reconcileOne reconciles a single order. The bool is false when the order
// can't be reconciled (no gateway reference or provider not configured) — that
// is a skip, not a drift.
// Dispatch is on PROVIDER first, then on the presence of a reference. Testing the
// reference alone (the original shape) breaks the moment two gateways share the
// gateway_payment_id column: a Cashfree order is non-empty there, so it would be
// reconciled against the wrong gateway's API with a Cashfree payment id and report
// DriftGatewayUnreachable on every single sweep — drowning the real findings.
func reconcileOne(o *models.Order) ([]Drift, bool) {
	switch models.NormalizeProvider(o.PaymentProvider) {
	case models.PaymentProviderCashfree:
		if o.GatewayOrderID == "" {
			return nil, false
		}
		return reconcileCashfree(o), true
	case models.PaymentProviderStripe:
		if o.StripePaymentIntentID == "" {
			return nil, false
		}
		return reconcileStripe(o), true
	default:
		// Wallet is a ledger credit with no gateway; a retired-gateway row has no
		// client left to ask since #1086. Both are skips, not drift.
		return nil, false
	}
}

// reconcileCashfree cross-checks one Cashfree order against the gateway.
//
// Cashfree has no single "payment" object carrying a cumulative amount_refunded,
// so the refunded total is summed from the order's refunds. The drift kinds, the
// captured-amount basis and the error-propagating per-line read are shared with
// reconcileStripe, so a drift means the same thing whichever gateway produced it.
func reconcileCashfree(o *models.Order) []Drift {
	// The slot the order was actually paid on. ReconcileSettlements sweeps every
	// partition, so asking the live account about a sandbox order would report
	// DriftPaymentNotCaptured on money that was captured — on the test slot.
	client := cashfreeClientFor(o.Mode)
	if client == nil {
		return nil // not configured — skip silently (logged once at startup)
	}

	payment, err := client.SuccessfulPayment(o.GatewayOrderID)
	if err != nil {
		return []Drift{driftFor(o, models.PaymentProviderCashfree, DriftGatewayUnreachable,
			fmt.Sprintf("fetch payments for order %s: %v", o.GatewayOrderID, err), 0, 0)}
	}

	var drifts []Drift
	if payment == nil {
		return append(drifts, driftFor(o, models.PaymentProviderCashfree, DriftPaymentNotCaptured,
			"no SUCCESS payment on the gateway order", o.Total, 0))
	}

	refunds, err := client.OrderRefundedPaise(o.GatewayOrderID)
	if err != nil {
		return append(drifts, driftFor(o, models.PaymentProviderCashfree, DriftGatewayUnreachable,
			fmt.Sprintf("fetch refunds for order %s: %v", o.GatewayOrderID, err), 0, 0))
	}

	// An error-PROPAGATING per-line read, so a DB blip skips this order rather than
	// understating captured and spuriously flagging DriftFullRefundUnstamped.
	perLine, plErr := PerLineRefundedTotalTxErr(database.DB, o.ID)
	if plErr != nil {
		log.Printf("reconciliation: skip cashfree order %s — per-line refund read failed: %v", o.ID, plErr)
		return drifts
	}
	localRefunded := o.RefundAmount + perLine
	gatewayRefunded := FromPaise(refunds)
	// Original captured amount: Total is mutated down by per-line cancels so add them
	// back; the credit rails were never charged to the gateway so subtract them.
	capturedPaise := ToPaise(o.Total+perLine-o.WalletApplied) - ToPaise(o.LoyaltyApplied)

	switch {
	case capturedPaise > 0 && refunds >= capturedPaise && o.RefundedAt == nil:
		drifts = append(drifts, driftFor(o, models.PaymentProviderCashfree, DriftFullRefundUnstamped,
			"gateway refunds >= captured but refunded_at is NULL (cumulative/out-of-band full refund)",
			localRefunded, gatewayRefunded))
	case math.Abs(gatewayRefunded-localRefunded) > reconAmountTolerance:
		drifts = append(drifts, driftFor(o, models.PaymentProviderCashfree, DriftRefundMismatch,
			"gateway refunds != platform cumulative refunded (RefundAmount + per-line)",
			localRefunded, gatewayRefunded))
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
