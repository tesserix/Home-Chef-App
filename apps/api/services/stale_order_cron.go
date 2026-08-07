package services

// stale_order_cron.go — expire abandoned unpaid orders (#872 step 1).
//
// An order row is created BEFORE payment, reserving the chef's daily dish
// capacity (#48). Settlement is client-verify-only — there is no webhook
// configured — so the customer's browser calling back after checkout is the
// ONLY thing that ever confirms a payment. If that call never lands (dismissed
// checkout sheet, app killed, network drop after Cashfree/Razorpay already
// captured the money) the order sits payment_status=pending with no other gate
// behind it. This cron is that last gate.
//
// Before #872 it inferred "nothing was captured" from our own
// payment_status=pending row and cancelled on that alone — exactly the field
// that is wrong in the lost-callback case. Now, for any order that was actually
// stamped with a gateway order id, it asks the gateway directly, and an unknown
// answer (gateway error, gateway not configured for that mode, unrecognised
// provider) NEVER cancels — it skips and retries next tick. A captured-but-
// unconfirmed order is logged loudly and left pending for
// order_payment_reconcile_cron.go to settle.
//
// Only an order with no gateway order id ever stamped — the common abandoned-
// checkout case — is cancelled with zero gateway calls.
//
// ── The PENDING hole (4 Aug 2026) ────────────────────────────────────────────
//
// The gateway probe was a bool: "is there a CAPTURED payment?". That folds two
// opposite answers into one — "every attempt is dead" and "an attempt is still
// running" — and cancelled on both. Cashfree parks a card at PENDING while the
// bank's OTP page is open and a UPI collect while the payer decides; Razorpay's
// `authorized` is money already held on the card. All of them read as "" and got
// the order cancelled, after which the forward-only reconcile cron will not
// touch it (it excludes status=cancelled by design), so a charge that later
// succeeded landed on a cancelled order with nothing to recover it.
//
// The probe is now tri-state — captured / in-flight / dead — and only the last
// cancels. See gatewayPaymentState and CashfreePayment.IsInFlight.

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

const (
	staleOrderInterval = 10 * time.Minute
	// Payment normally completes within seconds; 30 min comfortably covers the
	// checkout sheet, retries, and the webhook fallback before we give up.
	staleOrderThreshold = 30 * time.Minute
	// Bound the batch so a backlog can't hold a transaction open too long.
	staleOrderBatch = 100
)

// StartStaleOrderCron launches the abandoned-order sweep. Returns immediately;
// lives for the life of ctx.
func StartStaleOrderCron(ctx context.Context) {
	go func() {
		runStaleOrderScan(ctx)
		ticker := time.NewTicker(staleOrderInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("stale-order: shutting down on ctx cancel")
				return
			case <-ticker.C:
				runStaleOrderScan(ctx)
			}
		}
	}()
	log.Println("stale-order: cron started (interval=10m, grace=30m)")
}

// runStaleOrderScan is the thin production wrapper: real database.DB, real
// clock, panic recovery, and a summary log line. The scan body lives in
// runStaleOrderScanWithDB so it is testable without a ticker or a live DB.
func runStaleOrderScan(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("stale-order: panic recovered: %v", r)
		}
	}()

	expired, skippedCaptured, skippedInFlight, skippedError := runStaleOrderScanWithDB(ctx, database.DB, time.Now())
	if expired > 0 || skippedCaptured > 0 || skippedInFlight > 0 || skippedError > 0 {
		log.Printf("stale-order: expired %d, skipped %d (captured, awaiting settlement), skipped %d (in flight at the gateway), skipped %d (gateway error, will retry)",
			expired, skippedCaptured, skippedInFlight, skippedError)
	}
}

// staleOrderAction is the outcome decideStaleOrderAction picks for one order.
type staleOrderAction int

const (
	// staleOrderCancel: the gateway confirmed (or there is nothing to ask —
	// no gateway order id was ever stamped) that no payment was captured.
	// Cancel exactly as before.
	staleOrderCancel staleOrderAction = iota
	// staleOrderSkipCaptured: the gateway found a captured payment. The order
	// stays pending; it must not be cancelled or mutated.
	staleOrderSkipCaptured
	// staleOrderSkipError: the gateway answer is unknown (fetch error,
	// unconfigured slot, unrecognised provider). Never cancel on an unknown
	// answer — skip and let the next tick retry.
	staleOrderSkipError
	// staleOrderSkipInFlight: the gateway holds an attempt that has not resolved
	// yet. Not captured, but not dead either — cancelling now is what strands a
	// charge on a cancelled order.
	staleOrderSkipInFlight
)

// gatewayPaymentState is what the gateway says about an order's attempts. The
// probe used to be a bool ("is there a captured payment?"), which folded two
// very different answers together: "every attempt is dead" and "an attempt is
// still running". Both read as "" and both cancelled.
type gatewayPaymentState int

const (
	// gatewayNoPayment: every attempt is terminally failed, or there are none.
	gatewayNoPayment gatewayPaymentState = iota
	// gatewayCaptured: money moved.
	gatewayCaptured
	// gatewayInFlight: an attempt exists that could still take the money —
	// Cashfree PENDING, Razorpay created/authorized.
	gatewayInFlight
)

// decideStaleOrderAction is the pure decision seam — no DB, no gateway I/O — so
// it can be exercised directly against every branch of the decision table.
//
//	hasGatewayOrderID  gatewayErr  state        → action
//	false              (unasked)   (unasked)    → cancel
//	true               non-nil     —            → skipError
//	true               nil         captured     → skipCaptured
//	true               nil         inFlight     → skipInFlight
//	true               nil         noPayment    → cancel
//
// Only ONE branch cancels an order that reached the gateway, and it is the one
// where the gateway itself said every attempt is dead.
func decideStaleOrderAction(hasGatewayOrderID bool, state gatewayPaymentState, gatewayErr error) staleOrderAction {
	if !hasGatewayOrderID {
		return staleOrderCancel
	}
	if gatewayErr != nil {
		return staleOrderSkipError
	}
	switch state {
	case gatewayCaptured:
		return staleOrderSkipCaptured
	case gatewayInFlight:
		return staleOrderSkipInFlight
	default:
		return staleOrderCancel
	}
}

// staleOrderPaymentState asks the gateway that took (or was supposed to take)
// this order's payment what state its attempts are in. Returns gatewayCaptured
// with the payment id when money moved, gatewayInFlight when an attempt is still
// live, gatewayNoPayment when the gateway confirms every attempt is dead, and a
// non-nil error for anything the caller must treat as unknown.
//
// One FetchOrderPayments call answers all three — the captured/in-flight split
// is made here rather than with a second round-trip.
func staleOrderPaymentState(order *models.Order) (state gatewayPaymentState, capturedPaymentID string, err error) {
	switch models.NormalizeProvider(order.PaymentProvider) {
	case models.PaymentProviderCashfree:
		cf := GetCashfreeFor(order.Mode)
		if cf == nil {
			return gatewayNoPayment, "", fmt.Errorf("no cashfree gateway configured for mode %q", order.Mode)
		}
		payments, err := cf.FetchOrderPayments(order.GatewayOrderID)
		if err != nil {
			return gatewayNoPayment, "", err
		}
		inFlight := false
		for i := range payments {
			if payments[i].IsCaptured() {
				return gatewayCaptured, payments[i].CFPaymentID.String(), nil
			}
			if payments[i].IsInFlight() {
				inFlight = true
			}
		}
		if inFlight {
			return gatewayInFlight, "", nil
		}
		return gatewayNoPayment, "", nil

	default:
		// Provider-mismatch backstop (see PLAN.md's decision table): a provider
		// this cron doesn't know how to ask is an unknown answer, never a
		// cancel. Legacy Razorpay orders land here since #1086 — the retired
		// gateway is not asked, and they are never cancelled on its silence.
		return gatewayNoPayment, "", fmt.Errorf("unrecognised payment provider %q", order.PaymentProvider)
	}
}

// runStaleOrderScanWithDB is the scan body: query + per-row loop, taking db +
// now explicitly so it is testable without a ticker or a live clock. Returns
// how many orders it cancelled, skipped as captured, and skipped on a gateway
// error.
func runStaleOrderScanWithDB(ctx context.Context, db *gorm.DB, now time.Time) (expired, skippedCaptured, skippedInFlight, skippedError int) {
	cutoff := now.Add(-staleOrderThreshold)

	var stale []models.Order
	if err := db.
		Preload("Items").
		Where("payment_status = ? AND status NOT IN ? AND created_at < ?",
			models.PaymentPending,
			[]models.OrderStatus{
				models.OrderStatusCancelled,
				models.OrderStatusRejected,
				models.OrderStatusRefunded,
				models.OrderStatusDelivered,
			},
			cutoff).
		Order("created_at ASC").
		Limit(staleOrderBatch).
		Find(&stale).Error; err != nil {
		log.Printf("stale-order: query failed: %v", err)
		return 0, 0, 0, 0
	}
	if len(stale) == 0 {
		return 0, 0, 0, 0
	}

	for i := range stale {
		select {
		case <-ctx.Done():
			return expired, skippedCaptured, skippedInFlight, skippedError
		default:
		}
		order := stale[i]

		hasGatewayOrderID := order.GatewayOrderID != ""
		var state gatewayPaymentState
		var capturedPaymentID string
		var gatewayErr error
		if hasGatewayOrderID {
			state, capturedPaymentID, gatewayErr = staleOrderPaymentState(&order)
		}

		switch decideStaleOrderAction(hasGatewayOrderID, state, gatewayErr) {
		case staleOrderSkipCaptured:
			log.Printf("stale-order: CAPTURED PAYMENT — order %s (gateway order %s, provider %s, mode %s) has a captured payment %s but is still payment_status=pending; leaving it pending, not settling",
				order.OrderNumber, order.GatewayOrderID, order.PaymentProvider, order.Mode, capturedPaymentID)
			skippedCaptured++
			continue

		case staleOrderSkipInFlight:
			log.Printf("stale-order: PAYMENT IN FLIGHT — order %s (gateway order %s, provider %s, mode %s) has an unresolved attempt at the gateway; leaving it pending rather than cancelling under a live charge",
				order.OrderNumber, order.GatewayOrderID, order.PaymentProvider, order.Mode)
			skippedInFlight++
			continue

		case staleOrderSkipError:
			log.Printf("stale-order: gateway answer unknown for order %s (gateway order %s, provider %s, mode %s): %v; leaving it pending for retry",
				order.OrderNumber, order.GatewayOrderID, order.PaymentProvider, order.Mode, gatewayErr)
			skippedError++
			continue

		case staleOrderCancel:
			if err := db.Transaction(func(tx *gorm.DB) error {
				// Mark cancelled — the gateway confirmed (or there was
				// nothing to ask) that no payment was captured, so nothing to
				// refund.
				cancelledAt := now
				if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
					"status":         models.OrderStatusCancelled,
					"payment_status": models.PaymentFailed,
					"cancel_reason":  "payment not completed",
					"cancelled_at":   cancelledAt,
				}).Error; err != nil {
					return err
				}
				// Release the reserved daily capacity — these dishes won't be made.
				capDay := CapacityDay(order.CreatedAt)
				for _, it := range order.Items {
					if err := ReleaseCapacity(tx, it.MenuItemID, it.Quantity, capDay); err != nil {
						return err
					}
				}
				// Release the scheduled delivery-slot booking too (#51), if any.
				if order.DeliverySlot != "" && order.ScheduledFor != nil {
					if err := ReleaseSlot(tx, order.ChefID, order.DeliverySlot, 1, CapacityDay(*order.ScheduledFor)); err != nil {
						return err
					}
				}
				return nil
			}); err != nil {
				log.Printf("stale-order: failed to expire order %s: %v", order.ID, err)
				continue
			}
			expired++
		}
	}
	return expired, skippedCaptured, skippedInFlight, skippedError
}
