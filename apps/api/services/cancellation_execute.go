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
		capped := false // #644: set when the money move was capped below the snapshot; drives the breakdown re-persist
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
			// #644: the snapshot cr.RefundTotalPaise was capped at RemainingRefundable at DECISION
			// time — an UNLOCKED read. A RefundIssueToWallet partial (which never flips
			// payment_status, so the claim above didn't exclude it) committing since then lowered the
			// true remaining. Re-derive it here UNDER the row lock the winning claim just took and cap
			// the money move, so cumulative refunds can never exceed the order total. refund_amount is
			// read fresh (the claim didn't touch it) and is BEFORE this cancellation's own increment
			// (added below), so remaining = Total − already-refunded + per-line (#527). The claim
			// serializes any concurrent refund AFTER this tx (its FOR UPDATE blocks on our lock).
			var locked models.Order
			if e := tx.Select("total", "refund_amount").First(&locked, "id = ?", order.ID).Error; e != nil {
				return e
			}
			remaining := models.RoundAmount(locked.Total - locked.RefundAmount + PerLineRefundedTotalTx(tx, order.ID))
			if remaining < 0 {
				remaining = 0
			}
			if refund > remaining {
				refund = remaining
				// Re-scale the WHOLE breakdown to the capped total (not just RefundTotalPaise) so the
				// persisted cancellation_request row stays internally conserved
				// (grand == RefundTotal + VendorKept + PlatformKept) AND the customer-facing refund
				// figure matches what actually moved — otherwise the row would show the stale,
				// larger snapshot the customer never received (#644 verify).
				scaled := CancellationRefund{
					FoodRefund: cr.FoodRefundPaise, DeliveryRefund: cr.DeliveryRefundPaise, TaxRefund: cr.TaxRefundPaise,
					Total: cr.RefundTotalPaise, VendorKept: cr.VendorKeptPaise, PlatformKept: cr.PlatformKeptPaise,
				}.CappedAt(ToPaise(remaining))
				cr.FoodRefundPaise = scaled.FoodRefund
				cr.DeliveryRefundPaise = scaled.DeliveryRefund
				cr.TaxRefundPaise = scaled.TaxRefund
				cr.RefundTotalPaise = scaled.Total
				cr.VendorKeptPaise = scaled.VendorKept
				cr.PlatformKeptPaise = scaled.PlatformKept
				capped = true
			}
			if refund <= 0 {
				// A concurrent refund already covered the whole order — nothing left to move. The
				// claim already flipped payment_status→refunded; finalize with a zero increment.
				cr.RefundRef = "already-refunded"
			} else if cr.RefundDestination == "original" {
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
		// #636: increment refund_amount ATOMICALLY in-SQL rather than persisting the caller's
		// stale in-memory `order.RefundAmount + refund`. The claim above flips payment_status,
		// which mutually excludes the ReserveRefund-family paths — but RefundIssueToWallet (the
		// customer-issue refund) does NOT flip payment_status, so a partial issue refund that
		// committed between the caller loading `order` and this write would be CLOBBERED by the
		// stale read-modify-write (refund_amount under-counted → a later refund over-refunds).
		// COALESCE so a NULL column increments from 0. Runs exactly once per cancellation (the
		// payment_status claim gates re-entry; the sweep's retry loses the claim).
		if err := tx.Model(&models.Order{}).Where("id = ?", order.ID).Updates(map[string]any{
			"status":        models.OrderStatusCancelled,
			"refund_amount": gorm.Expr("COALESCE(refund_amount, 0) + ?", refund),
			"refunded_at":   now,
			"refund_reason": "customer cancellation",
		}).Error; err != nil {
			return err
		}
		cr.RefundExecuted = true
		// Key by order_id (one request per order, unique) so this is robust even
		// when the request id was assigned by the DB default.
		finalize := map[string]any{"refund_executed": true, "refund_ref": cr.RefundRef, "resolved_at": now}
		if capped {
			// Persist the re-scaled breakdown so the stored request reflects the amount ACTUALLY
			// refunded, not the stale pre-cap snapshot (#644 verify). Only on cap — the un-capped
			// path leaves the handler's already-correct breakdown untouched (no behavior change).
			finalize["refund_total_paise"] = cr.RefundTotalPaise
			finalize["food_refund_paise"] = cr.FoodRefundPaise
			finalize["delivery_refund_paise"] = cr.DeliveryRefundPaise
			finalize["tax_refund_paise"] = cr.TaxRefundPaise
			finalize["vendor_kept_paise"] = cr.VendorKeptPaise
			finalize["platform_kept_paise"] = cr.PlatformKeptPaise
		}
		if err := tx.Model(&models.CancellationRequest{}).Where("order_id = ?", order.ID).
			Updates(finalize).Error; err != nil {
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
