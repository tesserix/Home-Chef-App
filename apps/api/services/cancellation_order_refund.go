package services

// cancellation_order_refund.go — the shared full-refund money mover for order
// CANCELLATIONS that don't go through the tiered arbitration flow (#392): the
// customer's direct cancel (handlers/orders.go CancelOrder) and the chef's
// status-endpoint reject (handlers/chefs.go UpdateOrderStatus → rejected). Both
// previously flipped the order state and refunded ₹0 on a paid order — a live money
// loss on every chef reject.
//
// DOUBLE-REFUND SAFETY (the whole point): every full-refund path in the codebase
// must claim the SAME mutex before touching the gateway. This service claims BOTH
//   payment_status = 'completed'  (shared with handlers/payment.go InitiateRefund)
//   AND refunded_at IS NULL       (shared with ExecuteCancellationRefund +
//                                  chef_order_cancel.go, which don't touch
//                                  payment_status)
// in one conditional UPDATE. RowsAffected==1 ⇒ this call owns the refund; 0 ⇒ a
// sibling path already refunded → no-op. On a gateway failure the claim is reverted
// so a retry can re-refund. Razorpay's CreateRefund carries no idempotency key, so
// the atomic claim is the ONLY thing preventing a concurrent double-refund.
//
// It does NOT write `status` — the caller owns that (cancelled / rejected), so the
// reject semantics aren't clobbered. It stamps only payment_status + the refund
// columns + drives the payout hold.

import (
	"fmt"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// RefundOrderForCancellation issues the customer's full refund (Total − already-
// refunded) for an order being cancelled/rejected, on whichever gateway it was paid.
// Idempotent + concurrency-safe via the shared refund claim. No-op for an unpaid
// order (nothing was ever charged) or one already refunded. Best-effort by contract
// — callers must not fail the cancel on a refund error (the reconcile cron's
// refund_mismatch check is the backstop), but the error is returned + logged.
func RefundOrderForCancellation(order *models.Order, initiatedBy, reason string) error {
	// Never charged ⇒ nothing to refund (e.g. a still-unpaid pending order rejected).
	if order.PaymentStatus != models.PaymentCompleted {
		return nil
	}
	// #527 race: refresh the money fields from the DB — a concurrent per-line cancel may
	// have moved Total/RefundAmount/WalletApplied since the caller loaded `order`, and the
	// refund math (RemainingRefundable + the wallet-capture cap below) must run on the
	// current committed state, not a stale snapshot, or it can over-refund. Best-effort:
	// on a read error we keep the caller's snapshot (RemainingRefundable itself re-reads
	// consistently and fails safe).
	var money struct{ Total, RefundAmount, WalletApplied float64 }
	if err := database.DB.Model(&models.Order{}).Select("total", "refund_amount", "wallet_applied").
		Where("id = ?", order.ID).Scan(&money).Error; err == nil {
		order.Total, order.RefundAmount, order.WalletApplied = money.Total, money.RefundAmount, money.WalletApplied
	}
	// #527: RemainingRefundable (consistent snapshot), NOT Total − RefundAmount — after a
	// per-line cancel the latter double-subtracts the refunded lines (Total was already
	// reduced) and strands the remaining live items' money on a full cancel/reject.
	refundAmount := RemainingRefundable(order)
	if refundAmount <= 0 {
		return nil // already fully refunded via prior per-line refunds — nothing left
	}
	now := time.Now()

	// Unified atomic claim — the single guard against a cross-path double refund.
	claim := database.DB.Model(&models.Order{}).
		Where("id = ? AND payment_status = ? AND refunded_at IS NULL", order.ID, models.PaymentCompleted).
		Updates(map[string]any{"payment_status": models.PaymentRefunded, "refunded_at": now})
	if claim.Error != nil {
		return fmt.Errorf("cancel-refund: claim order %s: %w", order.ID, claim.Error)
	}
	if claim.RowsAffected == 0 {
		return nil // a sibling refund path already claimed/refunded this order
	}
	revertClaim := func() {
		database.DB.Model(&models.Order{}).Where("id = ?", order.ID).
			Updates(map[string]any{"payment_status": models.PaymentCompleted, "refunded_at": gorm.Expr("NULL")})
	}

	provider := strings.ToLower(order.PaymentProvider)
	if provider == "" {
		provider = "razorpay"
	}

	// Wallet-at-checkout (#141): a wallet-funded order only captured (Total −
	// WalletApplied) at the gateway, so the gateway can't refund more than that.
	// Re-credit the wallet slice as store credit and cap the gateway refund to the
	// captured amount (parity with InitiateRefund).
	if order.WalletApplied > 0 && provider != "wallet" {
		capture := order.Total - order.WalletApplied
		if refundAmount > capture {
			walletPortion := refundAmount - capture
			if _, werr := CreditWallet(database.DB, order.CustomerID, walletPortion,
				models.WalletSourceRefund, &order.ID,
				fmt.Sprintf("Wallet-portion refund for order %s: %s", order.OrderNumber, reason),
				"refund-wallet:"+order.ID.String(), nil); werr != nil {
				log.Printf("cancel-refund: wallet-portion re-credit failed order=%s: %v", order.OrderNumber, werr)
				CaptureBackgroundError(werr)
			}
			refundAmount = capture
		}
	}

	refundID, gerr := runCancellationGatewayRefund(order, provider, refundAmount, initiatedBy, reason)
	if gerr != nil {
		revertClaim()
		return fmt.Errorf("cancel-refund: gateway refund order %s: %w", order.ID, gerr)
	}

	// Persist the refund columns. payment_status + refunded_at were set by the claim;
	// status is intentionally left to the caller (cancelled / rejected).
	if err := database.DB.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]any{
		"refund_id":           refundID,
		"refund_amount":       order.RefundAmount + refundAmount,
		"refund_reason":       reason,
		"refund_initiated_by": initiatedBy,
	}).Error; err != nil {
		// Money moved but the stamp failed — surface for reconciliation; do NOT revert
		// the claim (that would allow a re-refund of money already sent).
		log.Printf("cancel-refund: persist refund cols failed order=%s (money moved, refundId=%s): %v", order.ID, refundID, err)
		CaptureBackgroundError(err)
	}

	// Cross-guard the payout hold — the refunded slice must never reach the chef.
	if hErr := WithholdOrReverseOrderHoldForRefund(database.DB, order.ID, reason); hErr != nil {
		log.Printf("cancel-refund: payout cross-guard failed order=%s: %v", order.ID, hErr)
		CaptureBackgroundError(hErr)
	}
	return nil
}

// runCancellationGatewayRefund issues the actual gateway/wallet refund and returns
// the refund reference. Mirrors handlers/payment.go InitiateRefund's provider switch.
func runCancellationGatewayRefund(order *models.Order, provider string, refundAmount float64, initiatedBy, reason string) (string, error) {
	switch provider {
	case "wallet":
		txn, werr := CreditWallet(database.DB, order.CustomerID, refundAmount,
			models.WalletSourceRefund, &order.ID,
			fmt.Sprintf("Refund for order %s: %s", order.OrderNumber, reason),
			"refund:"+order.ID.String(), nil)
		if werr != nil {
			return "", werr
		}
		return "wallet:" + txn.ID.String(), nil
	case "stripe":
		if order.StripePaymentIntentID == "" {
			return "", fmt.Errorf("no stripe payment on order")
		}
		st := GetStripe()
		if st == nil {
			return "", fmt.Errorf("stripe gateway not configured")
		}
		currency := strings.ToLower(order.Currency)
		if currency == "" {
			currency = CurrencyForCountry(order.Chef.PayoutCountry)
		}
		r, err := st.CreateRefund(&StripeRefundRequest{
			PaymentIntent:        order.StripePaymentIntentID,
			Amount:               ToMinor(refundAmount, currency),
			Reason:               "requested_by_customer",
			ReverseTransfer:      true,
			RefundApplicationFee: true,
			Metadata: map[string]string{
				"order_id": order.ID.String(), "order_number": order.OrderNumber,
				"reason": reason, "initiated_by": initiatedBy,
			},
		})
		if err != nil {
			return "", err
		}
		return r.ID, nil
	default: // razorpay
		if order.RazorpayPaymentID == "" {
			return "", fmt.Errorf("no razorpay payment on order")
		}
		rz := GetRazorpay()
		if rz == nil {
			return "", fmt.Errorf("razorpay gateway not configured")
		}
		r, err := rz.CreateRefund(order.RazorpayPaymentID, &RefundRequest{
			Amount: ToPaise(refundAmount),
			Speed:  "normal",
			Notes: map[string]string{
				"order_id": order.ID.String(), "order_number": order.OrderNumber,
				"reason": reason, "initiated_by": initiatedBy,
			},
			Receipt: fmt.Sprintf("refund-%s", order.OrderNumber),
			// Full cancellation refund, issued once per order. #574.
			IdempotencyKey: RefundFullIdempotencyKey(order.ID),
		})
		if err != nil {
			return "", err
		}
		return r.ID, nil
	}
}
