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
// so a retry can re-refund. A gateway CreateRefund may carry no idempotency key, so
// the atomic claim is the ONLY thing preventing a concurrent double-refund.
//
// It does NOT write `status` — the caller owns that (cancelled / rejected), so the
// reject semantics aren't clobbered. It stamps only payment_status + the refund
// columns + drives the payout hold.

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services/orderrefund"
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
	// #544/#394: an order spawned by a typed escrow flow (meal-plan day / group order) is
	// refund-managed by THAT flow (RefundDay / participant refunds) on a DISJOINT idempotency
	// keyspace, and its held chef payout is a DIRECT transfer the generic path can't reverse. The
	// generic cancellation refund must never touch it: it keys refund:<orderID> (which the typed
	// keys don't see), so the moment a gateway payment id lands on such a row it would refund the
	// customer a SECOND time while the chef keeps the transfer. Skip here — the meal-plan / group
	// cancellation flow is the right place. Mirrors the InitiateRefund guard (handlers/payment.go).
	// On a type-check error, refuse the refund (fail safe: never generic-refund a possibly-typed
	// order); the callers are best-effort + the reconcile cron's refund_mismatch check backstops.
	switch kind, kErr := TypedRefundOrderKind(database.DB, order.ID); {
	case kErr != nil:
		return fmt.Errorf("cancel-refund: check order type %s: %w", order.ID, kErr)
	case kind != "":
		log.Printf("cancel-refund: skipping generic refund for %s order %s — refund-managed by its escrow flow", kind, order.ID)
		return nil
	}
	// #690: the reservation, the wallet-capture split, the gateway routing and the
	// release-on-failure all used to be written out here. They now live in the coordinator
	// and its gateway adapter — one implementation for every refund path, instead of this
	// one and four near-copies that each had to remember the same rules.
	//
	// Behaviour is deliberately unchanged: ScopeFull makes the gateway key byte-identical
	// to RefundFullIdempotencyKey (so this still dedups against the un-migrated cancellation
	// paths), and the coordinator takes the same payment_status claim this used to take
	// directly.
	res, rErr := NewOrderRefundCoordinator().Refund(context.Background(), orderrefund.RefundCommand{
		OrderID: order.ID,
		Reason:  reason,
		Actor:   initiatedBy,
		ScopeID: orderrefund.ScopeFull, // nil Amount ⇒ the full remaining
	})
	switch {
	// The legacy !won branch returned nil for all three of these: a sibling path already
	// refunded, nothing is left, or the order isn't payable. This path is best-effort by
	// contract and must not fail a cancel because the money is already handled.
	case errors.Is(rErr, orderrefund.ErrNoCapturedPayment),
		errors.Is(rErr, orderrefund.ErrExceedsRemaining),
		errors.Is(rErr, orderrefund.ErrRefundInFlight):
		return nil
	case rErr != nil:
		return fmt.Errorf("cancel-refund: refund order %s: %w", order.ID, rErr)
	}
	refundID := res.ProviderRefundID

	// Persist the refund columns. payment_status + refunded_at + refund_amount were set
	// atomically by the reservation; status is intentionally left to the caller (cancelled /
	// rejected). Do NOT re-write refund_amount here (the old `order.RefundAmount + refundAmount`
	// was a stale read-modify-write that clobbered a concurrent partial's increment — #609).
	if err := database.DB.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]any{
		"refund_id":           refundID,
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
//
// idempotencyKey is the LOGICAL operation id for this refund. It is a parameter rather
// than derived here (#690): it used to hardcode RefundFullIdempotencyKey, which is right
// for RefundOrderForCancellation — one full cancellation refund per order — and silently
// WRONG for the coordinator, which drives many distinct refunds per order and keys each by
// scope. Hardcoding it there would hand two different refunds the same key, the gateway
// would dedup the second as a retry of the first, and the customer would never receive it:
// no error, no trace, just missing money. The wallet branch keys its own credit off the
// same value for the same reason.
func runCancellationGatewayRefund(order *models.Order, provider string, refundAmount float64, initiatedBy, reason, idempotencyKey string) (string, error) {
	switch models.NormalizeProvider(provider) {
	case models.PaymentProviderWallet:
		txn, werr := CreditWallet(database.DB, order.CustomerID, refundAmount,
			models.WalletSourceRefund, &order.ID,
			fmt.Sprintf("Refund for order %s: %s", order.OrderNumber, reason),
			idempotencyKey, nil)
		if werr != nil {
			return "", werr
		}
		return "wallet:" + txn.ID.String(), nil
	default:
		// Every gateway provider routes through the ONE switch in
		// gateway_refund.go. What used to be three near-identical branches here was
		// the second copy of that switch, and the copy is what let a new provider be
		// added in one place and silently missed in the other.
		//
		// The caller's logical key (#574/#690) is passed straight through —
		// RefundFullIdempotencyKey for the cancellation path, a per-scope key when
		// the coordinator drives this.
		res, err := IssueOrderGatewayRefund(order, ToPaise(refundAmount), map[string]string{
			"order_id": order.ID.String(), "order_number": order.OrderNumber,
			"reason": reason, "initiated_by": initiatedBy,
		}, idempotencyKey)
		if err != nil {
			return "", err
		}
		return res.RefundID, nil
	}
}
