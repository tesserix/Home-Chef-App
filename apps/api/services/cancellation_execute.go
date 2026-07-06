package services

// cancellation_execute.go — the durable money mover for a cancellation refund
// (#475/#477). Lives in services (not the handler) so the retry SWEEP can re-run
// it. The whole refund is ONE atomic transaction, so a crash/failure leaves
// nothing half-done: either the customer is credited + the order cancelled +
// refund_executed=true all commit, or none do and the sweep retries. The wallet
// credit is idempotent on "cancel:<id>", so a retry never double-refunds.

import (
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// ExecuteCancellationRefund issues the refund on cr's snapshot, cancels the
// order, and cross-guards the payout hold. Idempotent + crash-safe (see above).
func ExecuteCancellationRefund(order *models.Order, cr *models.CancellationRequest) error {
	refund := float64(cr.RefundTotalPaise) / 100.0

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		if refund > 0 {
			// #392: shared refund claim — the SAME two-column mutex every full-refund
			// path uses (payment_status completed→refunded AND refunded_at IS NULL).
			// Claiming payment_status (not just refunded_at) is what mutually excludes
			// InitiateRefund, which stamps refunded_at only AFTER its gateway call.
			claim := tx.Model(&models.Order{}).
				Where("id = ? AND payment_status = ? AND refunded_at IS NULL", order.ID, models.PaymentCompleted).
				Updates(map[string]any{"payment_status": models.PaymentRefunded, "refunded_at": now})
			if claim.Error != nil {
				return claim.Error
			}
			if claim.RowsAffected == 0 {
				// A sibling path holds/completed the claim. Only mark THIS request done
				// if the customer has actually been refunded at least what it owed
				// (refund_amount is stamped only after a sibling's gateway SUCCESS). If
				// the sibling is still mid-flight or later reverts, leave it for the
				// sweep to retry — never permanently strand an owed refund on a
				// transient claim loss.
				var fresh models.Order
				if err := tx.Select("refund_amount").First(&fresh, "id = ?", order.ID).Error; err != nil {
					return err
				}
				if fresh.RefundAmount+0.001 < refund {
					return nil // not yet refunded — sweep retries (refund_executed stays false)
				}
				cr.RefundExecuted = true
				cr.RefundRef = "already-refunded"
				return tx.Model(&models.CancellationRequest{}).Where("order_id = ?", order.ID).
					Updates(map[string]any{"refund_executed": true, "refund_ref": cr.RefundRef, "resolved_at": now}).Error
			}
			if cr.RefundDestination == "original" {
				if order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == "" {
					return fmt.Errorf("original-method refund needs a razorpay payment")
				}
				rzp := GetRazorpay()
				if rzp == nil {
					return fmt.Errorf("razorpay unavailable")
				}
				resp, rErr := rzp.CreateRefund(order.RazorpayPaymentID, &RefundRequest{
					Amount: cr.RefundTotalPaise, Speed: "normal",
					Notes:          map[string]string{"order_id": order.ID.String(), "scope": "cancellation", "reason": cr.VendorReason},
					IdempotencyKey: RefundFullIdempotencyKey(order.ID), // one cancellation refund per order; claim + sweep re-drive with the same key. #574
				})
				if rErr != nil {
					return rErr
				}
				cr.RefundRef = resp.ID
			} else {
				if _, wErr := CreditWallet(tx, order.CustomerID, refund, models.WalletSourceRefund,
					&order.ID, "Cancellation refund", "cancel:"+cr.ID.String(), nil); wErr != nil {
					return wErr
				}
				cr.RefundRef = "wallet:cancel:" + cr.ID.String()
			}
		}
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]any{
			"status":        models.OrderStatusCancelled,
			"refund_amount": order.RefundAmount + refund,
			"refunded_at":   now,
			"refund_reason": "customer cancellation",
		}).Error; err != nil {
			return err
		}
		cr.RefundExecuted = true
		// Key by order_id (one request per order, unique) so this is robust even
		// when the request id was assigned by the DB default.
		if err := tx.Model(&models.CancellationRequest{}).Where("order_id = ?", order.ID).
			Updates(map[string]any{"refund_executed": true, "refund_ref": cr.RefundRef, "resolved_at": now}).Error; err != nil {
			return err
		}
		// Tell the customer their cancellation is confirmed + refund issued.
		// Staged in THIS tx (transactional outbox) so the notification is never
		// lost even if the process dies right after the refund commits.
		return EnqueueEvent(tx, SubjectCancellationResolved, "cancellation.resolved", order.CustomerID, map[string]any{
			"order_id": order.ID.String(), "refund": refund, "reason": cr.VendorReason,
		})
	})
	if err != nil {
		return err
	}
	// Cross-guard the payout hold — the refunded slice must never reach the chef
	// (audit #10). Best-effort here; the payout-reconcile cron is the backstop.
	if hErr := WithholdOrReverseOrderHoldForRefund(database.DB, order.ID, "customer cancellation"); hErr != nil {
		log.Printf("cancellation payout cross-guard failed for order %s: %v", order.ID, hErr)
	}
	return nil
}

// SweepCancellationRefunds retries any owed-but-unexecuted refund — the
// never-lost guarantee. A refund that failed inline (gateway blip, crash between
// create and execute) is picked up here and re-issued idempotently until
// refund_executed flips true.
func SweepCancellationRefunds() {
	var owed []models.CancellationRequest
	if err := database.DB.
		Where("status IN ? AND refund_executed = ?",
			[]models.CancellationRequestStatus{models.CancelReqApproved, models.CancelReqAutoRefunded, models.CancelReqResolved},
			false).
		Limit(200).Find(&owed).Error; err != nil {
		log.Printf("cancellation-sweep: query owed refunds failed: %v", err)
		return
	}
	for i := range owed {
		var order models.Order
		if err := database.DB.First(&order, "id = ?", owed[i].OrderID).Error; err != nil {
			continue
		}
		if err := ExecuteCancellationRefund(&order, &owed[i]); err != nil {
			log.Printf("cancellation-sweep: retry refund for %s failed (will retry): %v", owed[i].ID, err)
		}
	}
}

// SweepCancellationTimeouts routes vendor requests that blew past their response
// window to admin review, so a customer is never stuck on an unresponsive vendor.
func SweepCancellationTimeouts(now time.Time) {
	res := database.DB.Model(&models.CancellationRequest{}).
		Where("status = ? AND vendor_respond_by IS NOT NULL AND vendor_respond_by <= ?",
			models.CancelReqPendingVendor, now).
		Update("status", models.CancelReqAdminReview)
	if res.Error != nil {
		log.Printf("cancellation-sweep: timeout sweep failed: %v", res.Error)
	} else if res.RowsAffected > 0 {
		log.Printf("cancellation-sweep: %d vendor request(s) timed out → admin review", res.RowsAffected)
	}
}
