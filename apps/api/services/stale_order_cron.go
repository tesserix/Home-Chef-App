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
// stamped with a gateway order id, it asks the gateway directly whether a
// payment was captured, and an unknown answer (gateway error, gateway not
// configured for that mode, unrecognised provider) NEVER cancels — it skips and
// retries next tick. A captured-but-unconfirmed order is logged loudly and left
// pending; settling it is the next PR (an order-level reconcile cron), because
// the settle seam lives in `handlers`, which this package cannot import.
//
// Only an order with no gateway order id ever stamped — the common abandoned-
// checkout case — is cancelled with zero gateway calls.

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

	expired, skippedCaptured, skippedError := runStaleOrderScanWithDB(ctx, database.DB, time.Now())
	if expired > 0 || skippedCaptured > 0 || skippedError > 0 {
		log.Printf("stale-order: expired %d, skipped %d (captured, awaiting settlement), skipped %d (gateway error, will retry)",
			expired, skippedCaptured, skippedError)
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
)

// decideStaleOrderAction is the pure decision seam — no DB, no gateway I/O — so
// it can be exercised directly against every branch of the decision table.
//
//	hasGatewayOrderID  gatewayErr  capturedPaymentID  → action
//	false              (unasked)   (unasked)          → cancel
//	true               non-nil     —                  → skipError
//	true               nil         non-empty           → skipCaptured
//	true               nil         ""                  → cancel
func decideStaleOrderAction(hasGatewayOrderID bool, capturedPaymentID string, gatewayErr error) staleOrderAction {
	if !hasGatewayOrderID {
		return staleOrderCancel
	}
	if gatewayErr != nil {
		return staleOrderSkipError
	}
	if capturedPaymentID != "" {
		return staleOrderSkipCaptured
	}
	return staleOrderCancel
}

// staleOrderCapturedPayment asks the gateway that took (or was supposed to
// take) this order's payment whether a captured payment exists for its stamped
// gateway order id. Returns ("", nil) when the gateway confirms nothing was
// captured, a non-empty id when one was, and a non-nil error for anything the
// caller must treat as "unknown" — never captured, never not-captured.
func staleOrderCapturedPayment(order *models.Order) (capturedPaymentID string, err error) {
	switch models.NormalizeProvider(order.PaymentProvider) {
	case models.PaymentProviderCashfree:
		cf := GetCashfreeFor(order.Mode)
		if cf == nil {
			return "", fmt.Errorf("no cashfree gateway configured for mode %q", order.Mode)
		}
		payment, err := cf.SuccessfulPayment(order.RazorpayOrderID)
		if err != nil {
			return "", err
		}
		if payment == nil {
			return "", nil
		}
		return payment.CFPaymentID.String(), nil

	case models.PaymentProviderRazorpay:
		rz := GetRazorpayFor(order.Mode)
		if rz == nil {
			return "", fmt.Errorf("no razorpay gateway configured for mode %q", order.Mode)
		}
		pays, err := rz.FetchOrderPayments(order.RazorpayOrderID)
		if err != nil {
			return "", err
		}
		return capturedPaymentFor(pays, order.RazorpayOrderID), nil

	default:
		// Provider-mismatch backstop (see PLAN.md's decision table): a
		// provider this cron doesn't know how to ask is an unknown answer,
		// never a cancel.
		return "", fmt.Errorf("unrecognised payment provider %q", order.PaymentProvider)
	}
}

// runStaleOrderScanWithDB is the scan body: query + per-row loop, taking db +
// now explicitly so it is testable without a ticker or a live clock. Returns
// how many orders it cancelled, skipped as captured, and skipped on a gateway
// error.
func runStaleOrderScanWithDB(ctx context.Context, db *gorm.DB, now time.Time) (expired, skippedCaptured, skippedError int) {
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
		return 0, 0, 0
	}
	if len(stale) == 0 {
		return 0, 0, 0
	}

	for i := range stale {
		select {
		case <-ctx.Done():
			return expired, skippedCaptured, skippedError
		default:
		}
		order := stale[i]

		hasGatewayOrderID := order.RazorpayOrderID != ""
		var capturedPaymentID string
		var gatewayErr error
		if hasGatewayOrderID {
			capturedPaymentID, gatewayErr = staleOrderCapturedPayment(&order)
		}

		switch decideStaleOrderAction(hasGatewayOrderID, capturedPaymentID, gatewayErr) {
		case staleOrderSkipCaptured:
			log.Printf("stale-order: CAPTURED PAYMENT — order %s (gateway order %s, provider %s, mode %s) has a captured payment %s but is still payment_status=pending; leaving it pending, not settling",
				order.OrderNumber, order.RazorpayOrderID, order.PaymentProvider, order.Mode, capturedPaymentID)
			skippedCaptured++
			continue

		case staleOrderSkipError:
			log.Printf("stale-order: gateway answer unknown for order %s (gateway order %s, provider %s, mode %s): %v; leaving it pending for retry",
				order.OrderNumber, order.RazorpayOrderID, order.PaymentProvider, order.Mode, gatewayErr)
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
	return expired, skippedCaptured, skippedError
}
