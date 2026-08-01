package services

// order_payment_reconcile_cron.go — order-level payment reconcile (#872 step 2).
//
// Step 1 (stale_order_cron.go, `fix/872-stale-order-gateway-gate`) stopped the
// stale-order cron from cancelling a payment_status=pending order the gateway
// actually captured — it now skips and logs "CAPTURED PAYMENT" loudly, leaving
// the row pending forever. Nothing settled it: the settle core lived in
// `handlers`, which this package cannot import. Settlement is client-verify-only
// (no webhooks configured — 0 hits/24h), so a lost callback (app killed,
// network drop, dismissed checkout sheet after Cashfree/Razorpay already
// captured the money) had no path to ever complete.
//
// This cron is that path: it finds captured-but-unconfirmed orders and settles
// them through order_payment_settle.go's SettleCashfreeOrder /
// SettleRazorpayOrderFromPayment — the SAME transactional core the HTTP verify
// legs use (#872 step 2, Task 3 extracted it out of `handlers` for exactly this
// reason).
//
// Hard boundary — forward reconcile only. The query's `payment_status =
// 'pending'` clause structurally excludes every `status = 'cancelled'` row: a
// cancelled order is never selected, let alone settled. The stale cron, before
// step 1, wrongly cancelled captured orders (status=cancelled,
// payment_status=failed, cancel_reason='payment not completed', a non-empty
// gateway order id, refund_amount=0) — those need a REFUND, not a settle, and
// backfilling that is a separate, owner-gated task blocked on a production
// audit query nobody has run yet. This cron must never touch that set.
//
// Grace/interval, justified against stale_order_cron.go's two constants by
// name: staleOrderThreshold (30m) is when step 1 gives up waiting for the
// client verify and starts asking the gateway before cancelling;
// staleOrderInterval (10m) is how often it ticks. This cron uses a 5-minute
// grace and a 5-minute interval — worst case, a lost callback is discovered and
// settled within grace+interval = 10 minutes of the order being created, a
// third of staleOrderThreshold and comfortably before the stale cron's OWN
// first look at the row. That makes step 1's gateway-gate genuinely
// belt-and-braces (a captured order it would otherwise leave pending forever is
// already settled by the time it gets there) rather than the primary defence.

import (
	"context"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

const (
	orderPaymentReconcileGrace    = 5 * time.Minute
	orderPaymentReconcileInterval = 5 * time.Minute
	// Bound the batch so a backlog can't hold a transaction open too long —
	// mirrors staleOrderBatch.
	orderPaymentReconcileBatch = 100
)

// StartOrderPaymentReconcileCron launches the order-payment reconcile sweep.
// Returns immediately; lives for the life of ctx. Ticker-only (no immediate
// run at startup), mirroring StartMealPlanAdvanceReconcileCron's shape.
func StartOrderPaymentReconcileCron(ctx context.Context) {
	go func() {
		t := time.NewTicker(orderPaymentReconcileInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				runOrderPaymentReconcileScan(ctx)
			}
		}
	}()
}

// runOrderPaymentReconcileScan is the thin production wrapper: real
// database.DB, real clock, panic recovery, and a summary log line. The scan
// body lives in reconcileOrderPayments so it is testable without a ticker or a
// live DB.
func runOrderPaymentReconcileScan(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("order-payment-reconcile: panic recovered: %v", r)
		}
	}()
	if n := reconcileOrderPayments(database.DB, time.Now()); n > 0 {
		log.Printf("order-payment-reconcile: settled %d captured-but-unconfirmed order(s)", n)
	}
}

// reconcileOrderPayments settles captured-but-unconfirmed order payments and
// returns how many it settled. Belt-and-suspenders behind the HTTP client
// verify path AND step 1's stale-order gateway gate.
func reconcileOrderPayments(db *gorm.DB, now time.Time) int {
	// No top-level client: each order is settled against the gateway that took
	// its payment, so a live order and a test order in the same sweep talk to
	// different accounts. Bail only when NO slot at all is configured — a live
	// gateway or a test gateway, either provider, is enough to proceed (a
	// per-order unconfigured slot is handled per-row below).
	if GetCashfreeFor(models.ChefModeLive) == nil && GetCashfreeFor(models.ChefModeTest) == nil &&
		GetRazorpayFor(models.ChefModeLive) == nil && GetRazorpayFor(models.ChefModeTest) == nil {
		return 0
	}

	cutoff := now.Add(-orderPaymentReconcileGrace)
	var orders []models.Order
	if err := db.
		Preload("Chef").Preload("Delivery.DeliveryPartner").
		Where("payment_status = ? AND "+models.GatewayOrderIDColumn+" <> '' AND status NOT IN ? AND created_at < ?",
			models.PaymentPending,
			[]models.OrderStatus{
				models.OrderStatusCancelled,
				models.OrderStatusRejected,
				models.OrderStatusRefunded,
				models.OrderStatusDelivered,
			},
			cutoff).
		Order("created_at ASC").
		Limit(orderPaymentReconcileBatch).
		Find(&orders).Error; err != nil {
		log.Printf("order-payment-reconcile: query failed: %v", err)
		return 0
	}

	settled := 0
	for i := range orders {
		order := &orders[i]

		var ok bool
		var reason string
		switch models.NormalizeProvider(order.PaymentProvider) {
		case models.PaymentProviderCashfree:
			ok, reason, _ = SettleCashfreeOrder(order)
		case models.PaymentProviderRazorpay:
			ok, reason = settleRazorpayFromDiscovery(order)
		default:
			// Stripe/unrecognised — known gap, not this cron's job (Stripe orders
			// key off stripe_payment_intent_id, never razorpay_order_id, so this
			// predicate structurally excludes them already; this default only
			// guards an unrecognised value in the column). Not a transient
			// condition, so no log.
			continue
		}

		if ok {
			settled++
			log.Printf("order-payment-reconcile: settled order %s", order.OrderNumber)
			continue
		}
		// "Payment not completed" (Cashfree) and "" (Razorpay discovery finding
		// nothing captured) are the plain "still unpaid, waiting" case — must not
		// spam the log every 5 minutes. Everything else (gateway error,
		// unconfigured slot, underpayment, binding mismatch) must be loud.
		if reason != "" && reason != "Payment not completed" {
			log.Printf("order-payment-reconcile: order %s not settled: %s", order.OrderNumber, reason)
		}
	}
	return settled
}

// settleRazorpayFromDiscovery is the Razorpay counterpart to
// SettleCashfreeOrder's direct gateway-order lookup: the cron has no
// client-supplied payment id (unlike the HTTP verify leg), so it discovers the
// captured payment via FetchOrderPayments — the same discovery mechanism
// meal_plan_advance_reconcile_cron.go and stale_order_cron.go already use —
// then hands it to SettleRazorpayOrderFromPayment for the binding gate,
// completion transaction, and wallet settlement.
func settleRazorpayFromDiscovery(order *models.Order) (bool, string) {
	rz := GetRazorpayFor(order.Mode)
	if rz == nil {
		log.Printf("order-payment-reconcile: no razorpay gateway configured for mode %q (order %s)",
			order.Mode, order.OrderNumber)
		return false, "gateway not configured"
	}
	pays, err := rz.FetchOrderPayments(order.RazorpayOrderID)
	if err != nil {
		log.Printf("order-payment-reconcile: fetch payments failed for order %s (gateway order %s): %v",
			order.OrderNumber, order.RazorpayOrderID, err)
		return false, err.Error()
	}
	var found *PaymentResponse
	for i := range pays {
		if pays[i].Status == "captured" && pays[i].OrderID == order.RazorpayOrderID {
			found = &pays[i]
			break
		}
	}
	if found == nil {
		return false, "" // not captured yet — leave for the stale cron / next tick, no log
	}
	return SettleRazorpayOrderFromPayment(order, found)
}
