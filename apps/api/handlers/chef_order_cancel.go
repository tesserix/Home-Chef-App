package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"gorm.io/gorm"
)

// ChefOrderCancelHandler holds the chef-side cancellation routes.
// Whole-order and per-line cancel both refund through Razorpay; per-
// line additionally recomputes the order totals so subsequent
// statements + invoices reflect the smaller scope.
//
// Razorpay-only for now — orders paid via Stripe Connect return 422
// with a hint to use the Stripe-specific flow (TODO once that ships).
type ChefOrderCancelHandler struct{}

func NewChefOrderCancelHandler() *ChefOrderCancelHandler {
	return &ChefOrderCancelHandler{}
}

// cancellableStatuses lists the order states where a chef can still
// pull the plug. Once the order is out the door (picked_up onward)
// cancellation becomes a customer-support problem, not a chef
// problem — we don't expose it to mobile then.
var cancellableStatuses = map[models.OrderStatus]bool{
	models.OrderStatusAccepted:  true,
	models.OrderStatusPreparing: true,
	models.OrderStatusReady:     true,
}

type cancelRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// CancelOrder cancels the whole order, issues a full Razorpay refund,
// and notifies the customer. Idempotent on a re-call: if the order
// is already cancelled with a refund ID we return 200 + the same
// payload so retries from the mobile client don't double-refund.
// POST /chef/orders/:orderId/cancel
func (h *ChefOrderCancelHandler) CancelOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderID, chef.ID).
		Preload("Items").First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req cancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := models.CancelReason(req.Reason)
	if !reason.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reason; expected out_of_ingredient|equipment_failure|customer_request|other"})
		return
	}

	// Idempotency — same order cancelled already with a refund?
	// Return the existing state instead of re-refunding.
	if order.Status == models.OrderStatusCancelled && order.RefundID != "" {
		c.JSON(http.StatusOK, order.ToChefResponse())
		return
	}

	if !cancellableStatuses[order.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order cannot be cancelled at this stage"})
		return
	}

	if order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "only Razorpay-paid orders can be cancelled from the chef app today; reach out to support for Stripe orders",
		})
		return
	}

	// Already-refunded amount (e.g. per-line cancels) reduces what we
	// can refund now. amountToRefund is in paise — Razorpay's native
	// unit. Floor + min 1 paise; if there's nothing left to refund we
	// flip the status and skip the gateway call.
	// #527: RemainingRefundable, NOT Total − RefundAmount — a prior per-line cancel
	// reduced Total AND bumped RefundAmount, so the naive difference double-subtracts
	// the refunded lines and strands the remaining live items' money on this full cancel.
	refundable := services.RemainingRefundable(&order)
	amountPaise := int(roundPaise(refundable))
	var refundID string
	var refundedAt time.Time
	if amountPaise > 0 {
		// #392: shared refund claim. The mutex is the SAME two-column predicate every
		// full-refund path uses (payment.go InitiateRefund, RefundOrderForCancellation,
		// ExecuteCancellationRefund): claim payment_status completed→refunded AND stamp
		// refunded_at in one conditional UPDATE. Claiming payment_status (not just
		// refunded_at) is what mutually excludes InitiateRefund, which sets refunded_at
		// only AFTER its gateway call. RowsAffected==0 ⇒ a sibling already refunded.
		claim := database.DB.Model(&models.Order{}).
			Where("id = ? AND payment_status = ? AND refunded_at IS NULL", order.ID, models.PaymentCompleted).
			Updates(map[string]any{"payment_status": models.PaymentRefunded, "refunded_at": time.Now().UTC()})
		if claim.Error != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process cancellation"})
			return
		}
		if claim.RowsAffected == 0 {
			amountPaise = 0 // already refunded elsewhere
		}
	}
	revertRefundClaim := func() {
		database.DB.Model(&models.Order{}).Where("id = ?", order.ID).
			Updates(map[string]any{"payment_status": models.PaymentCompleted, "refunded_at": gorm.Expr("NULL")})
	}
	if amountPaise > 0 {
		rzp := services.GetRazorpay()
		if rzp == nil {
			revertRefundClaim() // let a retry refund once the gateway is back
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "razorpay client unavailable; refund deferred"})
			return
		}
		refundResp, err := rzp.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
			Amount: amountPaise,
			Speed:  "normal",
			Notes: map[string]string{
				"order_id":  order.ID.String(),
				"order_no":  order.OrderNumber,
				"chef_id":   chef.ID.String(),
				"reason":    string(reason),
				"initiator": "chef",
			},
			// Full-order cancel refund is issued once (order goes terminal-cancelled);
			// the claim above serializes concurrent attempts. #574.
			IdempotencyKey: services.RefundFullIdempotencyKey(order.ID),
		})
		if err != nil {
			revertRefundClaim()
			services.CaptureSentryError(c, err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "refund failed at gateway; please retry"})
			return
		}
		refundID = refundResp.ID
		refundedAt = time.Now().UTC()
	}

	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":              models.OrderStatusCancelled,
		"cancelled_at":        now,
		"cancel_reason":       string(reason),
		"refund_reason":       string(reason),
		"refund_initiated_by": "chef",
	}
	// #392: stamp refund_amount only when a refund was actually issued, and stamp the
	// cumulative (RefundAmount + refundable), not an unconditional Total — otherwise a
	// skipped/partial refund over-states the ledger and trips reconciliation.
	if refundID != "" {
		updates["refund_id"] = refundID
		updates["refunded_at"] = refundedAt
		updates["refund_amount"] = order.RefundAmount + refundable
	}
	if err := database.DB.Model(&order).Updates(updates).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}
	// Cross-guard the payout hold (#457) — the customer was fully refunded, so the
	// chef must not be paid. Best-effort; never fail the cancel on a hold-drive error.
	if hErr := services.WithholdOrReverseOrderHoldForRefund(database.DB, order.ID, "chef cancel: "+string(reason)); hErr != nil {
		log.Printf("payout cross-guard failed for cancelled order %s: %v", order.ID, hErr)
	}
	// Release the reserved daily capacity (#48) — these dishes won't be made.
	// Runs once per cancel (the already-cancelled guard above prevents re-entry).
	capDay := services.CapacityDay(order.CreatedAt)
	for _, it := range order.Items {
		_ = services.ReleaseCapacity(database.DB, it.MenuItemID, it.Quantity, capDay)
	}
	// Refresh the in-memory copy so the response reflects the saved state.
	_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error

	services.LogAudit(c, "chef.order.cancel", "order", order.ID.String(),
		nil, gin.H{"reason": string(reason), "refundAmount": order.RefundAmount, "refundId": refundID})

	publishOrderCancelled(order)

	// Cancel any booked 3PL delivery (no-op if none / already terminal). Off
	// the response path; failure must not fail the order cancellation/refund.
	go func() {
		if err := services.CancelOrderDelivery(order.ID, string(reason)); err != nil {
			log.Printf("Failed to cancel 3PL delivery for order %s: %v", order.ID, err)
			services.CaptureBackgroundError(err)
		}
	}()

	c.JSON(http.StatusOK, order.ToChefResponse())
}

// TODO(#457-followup): deferred cross-guard edges not wired in this slice —
//   - CancelOrderItem (below): per-line refund is SAFE without a hold drive because
//     it only runs pre-delivery (cancellableStatuses), where payout_hold_status is
//     still 'none' (the hold is parked at delivery). Wire it only if per-line refunds
//     ever become reachable post-delivery.
//   - handleRefundProcessed webhook (payment.go): an out-of-band gateway refund that
//     leaves orders.status='delivered' should also drive the hold.
//   - RefundGroupParticipant → hold-reverse parity for the group-order aggregate.
//   - day/group admin queue-exclusion (the ReleaseHold pre-check already backstops
//     all three aggregates; the queue filter itself is order-only for now).

// CancelOrderItem marks a single line as unfulfillable, refunds only
// that line (subtotal + proportional tax share), and recomputes the
// order totals atomically. The remaining items continue prep.
// POST /chef/orders/:orderId/items/:itemId/cancel
func (h *ChefOrderCancelHandler) CancelOrderItem(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")
	itemID := c.Param("itemId")

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderID, chef.ID).
		Preload("Items").First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req cancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	reason := models.CancelReason(req.Reason)
	if !reason.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reason"})
		return
	}

	if !cancellableStatuses[order.Status] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order is no longer mid-prep; cannot cancel a single line"})
		return
	}

	// Locate the target line in the preloaded slice. Looping is fine —
	// orders cap at a handful of lines in practice.
	var target *models.OrderItem
	for i := range order.Items {
		if order.Items[i].ID.String() == itemID {
			target = &order.Items[i]
			break
		}
	}
	if target == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Item not found on this order"})
		return
	}
	if target.IsCancelled {
		// Idempotent — return the current order state without re-refunding.
		c.JSON(http.StatusOK, order.ToChefResponse())
		return
	}

	if order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "per-line refunds only supported on Razorpay orders today",
		})
		return
	}

	// Tax share for this line = order.Tax * (line.Subtotal / order.Subtotal),
	// computed against the ORIGINAL subtotal/tax so concurrent partial cancels
	// can't drift the proportional split. (see lineRefundAmount)
	lineRefund := lineRefundAmount(target.Subtotal, order.Subtotal, order.Tax)
	amountPaise := int(roundPaise(lineRefund))
	if amountPaise <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "computed refund amount is zero; nothing to do"})
		return
	}

	rzp := services.GetRazorpay()
	if rzp == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "razorpay client unavailable; refund deferred"})
		return
	}

	now := time.Now().UTC()
	// #576: CLAIM the line BEFORE the gateway call — the same order the other refund paths
	// use (CancelOrder, InitiateRefund). A guarded is_cancelled=false→true CAS: two concurrent
	// cancels of the SAME line both passed the stale preloaded IsCancelled check above, but
	// only the winner flips the row and only the winner reaches CreateRefund. The loser never
	// issues a second real refund — closing the double-refund at the source, not relying on
	// gateway-side dedup of concurrent identical-key requests.
	var won bool
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		w, err := claimOrderItemForCancel(tx, target.ID, string(reason), now)
		won = w
		return err
	}); err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not cancel item"})
		return
	}
	if !won {
		// A concurrent cancel already claimed this line — respond idempotently, no refund.
		_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error
		c.JSON(http.StatusOK, order.ToChefResponse())
		return
	}

	refundResp, err := rzp.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
		Amount: amountPaise,
		Speed:  "normal",
		Notes: map[string]string{
			"order_id":      order.ID.String(),
			"order_item_id": target.ID.String(),
			"chef_id":       chef.ID.String(),
			"reason":        string(reason),
			"initiator":     "chef",
			"scope":         "line",
		},
		// #574: keyed by the immutable line id — a lost-response retry dedups to one refund.
		IdempotencyKey: services.RefundLineIdempotencyKey(order.ID, target.ID),
	})
	if err != nil {
		// The gateway refused — revert the claim so a retry can cancel this line (no money moved).
		if rErr := database.DB.Model(&models.OrderItem{}).Where("id = ?", target.ID).
			Updates(map[string]interface{}{"is_cancelled": false, "cancelled_reason": "", "cancelled_at": nil}).Error; rErr != nil {
			log.Printf("failed to revert per-line cancel claim for item %s after gateway error: %v", target.ID, rErr)
		}
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "refund failed at gateway; please retry"})
		return
	}

	// Persist the line refund + recompute the order totals atomically. On failure the line
	// stays cancelled (a retry answers idempotently) but the order ledger isn't updated — the
	// standard gateway-succeeded-persist-failed edge; the line key makes a retry's refund dedup.
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OrderItem{}).Where("id = ?", target.ID).Updates(map[string]interface{}{
			"refund_id":     refundResp.ID,
			"refund_amount": lineRefund,
		}).Error; err != nil {
			return err
		}

		// Release this line's reserved daily capacity (#48). Runs once — the CAS above
		// gates it to the winning cancel.
		if err := services.ReleaseCapacity(tx, target.MenuItemID, target.Quantity, services.CapacityDay(order.CreatedAt)); err != nil {
			return err
		}

		// Re-read fresh order + items inside the txn so we don't race
		// a concurrent per-line cancel.
		var fresh models.Order
		if err := tx.Where("id = ?", order.ID).Preload("Items").First(&fresh).Error; err != nil {
			return err
		}

		newSubtotal, newTax, newTotal := recomputeOrderTotals(
			fresh.Items, fresh.Subtotal, fresh.Tax,
			fresh.DeliveryFee, fresh.ServiceFee, fresh.Tip, fresh.Discount,
		)

		return tx.Model(&fresh).Updates(map[string]interface{}{
			"subtotal":      newSubtotal,
			"tax":           newTax,
			"total":         newTotal,
			"refund_amount": fresh.RefundAmount + lineRefund,
		}).Error
	})
	if err != nil {
		// The ledger persist failed after a successful gateway refund. REVERT the claim
		// (is_cancelled=false) — mirroring the gateway-error revert above — so this line is
		// retriable and doesn't drift PERMANENTLY: if it stayed is_cancelled=true with
		// refund_amount=0 and the order totals un-recomputed, the early IsCancelled
		// short-circuit would 200 every retry (never re-running this persist), leaving the
		// line's money still counted as refundable by RemainingRefundable — a later full
		// CancelOrder would then issue a SECOND real refund (its RefundFullIdempotencyKey
		// differs from the line key, so the gateway wouldn't dedup it). With the revert a
		// retry re-claims and re-refunds under the SAME line key (deduped, no double refund)
		// and re-records the ledger, healing the row.
		if rErr := database.DB.Model(&models.OrderItem{}).Where("id = ?", target.ID).
			Updates(map[string]interface{}{"is_cancelled": false, "cancelled_reason": "", "cancelled_at": nil}).Error; rErr != nil {
			log.Printf("failed to revert per-line cancel claim for item %s after persist error: %v", target.ID, rErr)
		}
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}

	// Refresh + emit. Skip status flip even if every line is now
	// cancelled — chef may intend to re-add via the customer-support
	// path; an explicit CancelOrder call is the right way to flip
	// status. (Could revisit later if zero-item-orders prove confusing.)
	_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error

	services.LogAudit(c, "chef.order.item_cancel", "order_item", target.ID.String(),
		nil, gin.H{"orderId": order.ID.String(), "reason": string(reason), "refundAmount": lineRefund})

	publishOrderUpdated(order)

	c.JSON(http.StatusOK, order.ToChefResponse())
}

// RefundOrder handles post-delivery refunds. Unlike CancelOrder
// (which is for in-flight orders + auto-refunds the full amount),
// this is for a delivered order where the customer complained later
// and the chef wants to refund all or part of it as a goodwill
// gesture or partner-mandated remedy.
//
// Idempotent on a re-call with the same amount: if RefundAmount
// already covers the requested total, we 200 with the existing state
// instead of double-refunding.
//
// POST /chef/orders/:orderId/refund   { amount: number, reason: string }
func (h *ChefOrderCancelHandler) RefundOrder(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderID, chef.ID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	var req struct {
		Amount float64 `json:"amount" binding:"required,gt=0"`
		Reason string  `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Refund is only meaningful after the customer received the order —
	// for in-flight cancellations the CancelOrder handler is the right
	// path (atomic state flip + full refund).
	if order.Status != models.OrderStatusDelivered {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "refunds via this endpoint are only available for delivered orders; for in-flight orders, use cancel",
		})
		return
	}

	if order.PaymentProvider != "razorpay" || order.RazorpayPaymentID == "" {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "only Razorpay-paid orders support refunds from the chef app today",
		})
		return
	}

	// Cap by what's left on the order. A goodwill refund covering the
	// full amount that's already been partially refunded just no-ops
	// the gateway call and returns the current state. Use RemainingRefundable
	// (#527/#560), NOT Total − RefundAmount: after a per-line cancel the naive
	// difference under-states the remaining, so a genuinely-partial refund could be
	// misread as FULL and forfeit the chef's entire hold (the exact #549 bug).
	remaining := services.RemainingRefundable(&order)
	if remaining <= 0 {
		c.JSON(http.StatusOK, order.ToChefResponse())
		return
	}
	if req.Amount > remaining {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("amount exceeds refundable balance (%.2f remaining)", remaining),
		})
		return
	}

	amountPaise := int(roundPaise(req.Amount))
	if amountPaise <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount rounds to zero paise"})
		return
	}

	rzp := services.GetRazorpay()
	if rzp == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "razorpay client unavailable; refund deferred"})
		return
	}

	// #549: only a FULL goodwill refund (exhausting the remaining balance) is terminal.
	fullRefund := req.Amount >= remaining

	// #576: atomically claim the order for refund (completed→refunded) BEFORE the gateway
	// call — the same mutex InitiateRefund uses (claimRefundForProcessing). This serializes
	// a concurrent goodwill + customer refund on order.RefundAmount; the loser gets 409
	// instead of issuing a second real refund and double-counting the ledger. It also makes
	// the gateway idempotency key below safe (a stable key with no local mutex would let the
	// gateway dedup one real refund while both ledgers still added their amount).
	won, cErr := claimRefundForProcessing(database.DB, order.ID)
	if cErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process refund"})
		return
	}
	if !won {
		c.JSON(http.StatusConflict, gin.H{"error": "a refund for this order is already in progress or has completed"})
		return
	}
	revertClaim := func() {
		database.DB.Model(&models.Order{}).Where("id = ?", order.ID).
			Update("payment_status", models.PaymentCompleted)
	}

	refundResp, err := rzp.CreateRefund(order.RazorpayPaymentID, &services.RefundRequest{
		Amount: amountPaise,
		Speed:  "normal",
		Notes: map[string]string{
			"order_id":  order.ID.String(),
			"order_no":  order.OrderNumber,
			"chef_id":   chef.ID.String(),
			"reason":    req.Reason,
			"initiator": "chef",
			"scope":     "post_delivery_goodwill",
		},
		// #574/#576: keyed by prior cumulative refunded paise (stable on retry, distinct
		// across sequential partials) — safe now that the claim above serializes a
		// same-order double-submit so the ledger can't double-count.
		IdempotencyKey: services.RefundPartialIdempotencyKey(order.ID, services.ToPaise(order.RefundAmount)),
	})
	if err != nil {
		revertClaim()
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "refund failed at gateway; please retry"})
		return
	}

	// Persist the refund. Target the row by id (not the preloaded &order) so a refund
	// doesn't spuriously upsert the belongs-to Chef association. #549: a PARTIAL refund
	// must NOT stamp refunded_at (the release-side guards block the WHOLE chef hold on
	// refunded_at IS NOT NULL) and reverts payment_status→completed so the hold stays
	// releasable + sequential partials re-claim; a FULL refund keeps refunded + stamps
	// refunded_at (terminal).
	refundUpdates := map[string]interface{}{
		"refund_id":           refundResp.ID,
		"refund_amount":       order.RefundAmount + req.Amount,
		"refund_reason":       req.Reason,
		"refund_initiated_by": "chef",
	}
	if fullRefund {
		refundUpdates["refunded_at"] = time.Now().UTC()
	} else {
		refundUpdates["payment_status"] = models.PaymentCompleted
	}
	persistErr := database.DB.Model(&models.Order{}).Where("id = ?", order.ID).Updates(refundUpdates).Error
	if persistErr != nil {
		// The gateway refund already succeeded but the ledger write failed. Revert the claim
		// (payment_status→completed) UNCONDITIONALLY — for BOTH partial AND full — so the
		// order isn't stuck at `refunded`, which would block every future refund path
		// (claimRefundForProcessing gates on completed) and strand this order needing a
		// manual DB fix. Reverting is safe here because the gateway idempotency key above
		// makes a client retry dedup to the SAME refund rather than issue a second one
		// (this is why RefundOrder can revert full where InitiateRefund's un-keyed gateway
		// branch cannot). The cross-guard still runs below so the payout stays blocked.
		log.Printf("goodwill refund persist failed for order %s: %v", order.ID, persistErr)
		services.CaptureBackgroundError(persistErr)
		revertClaim()
	}
	// Cross-guard the payout hold (#457/#549/#568) — a FULL refund drives the whole hold to
	// withheld/reversed (unconditional safety net, runs even on persist failure so the chef
	// is never paid for a refunded order); a PARTIAL claws back only the refunded portion
	// and only when the refund persisted (a retry can't double-claw). Best-effort.
	if hErr := crossGuardRefundHold(order.ID, req.Amount, req.Reason, fullRefund, persistErr == nil); hErr != nil {
		log.Printf("payout cross-guard failed for goodwill-refunded order %s: %v", order.ID, hErr)
	}
	if persistErr != nil {
		// Honest retry signal: the gateway refunded but state didn't save. The claim is
		// reverted + the key makes a retry dedup, so a re-submit heals the ledger.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}
	_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error

	services.LogAudit(c, "chef.order.refund", "order", order.ID.String(),
		nil, gin.H{"amount": req.Amount, "reason": req.Reason, "refundId": refundResp.ID})

	publishOrderUpdated(order)

	c.JSON(http.StatusOK, order.ToChefResponse())
}

// GetOrderInvoicePDF streams the tax-compliant PDF invoice for the
// chef's own order (separate auth path from the customer-side endpoint).
// GET /chef/orders/:orderId/invoice.pdf
func (h *ChefOrderCancelHandler) GetOrderInvoicePDF(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	orderID := c.Param("orderId")

	chef, err := loadChefForUser(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef profile not found"})
		return
	}

	orderUUID, parseErr := uuid.Parse(orderID)
	if parseErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var order models.Order
	if err := database.DB.Where("id = ? AND chef_id = ?", orderUUID, chef.ID).
		First(&order).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}
	if order.Status != models.OrderStatusDelivered {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invoice available after delivery"})
		return
	}

	pdfBytes, filename, genErr := services.GenerateOrderInvoicePDF(orderUUID)
	if genErr != nil {
		services.CaptureSentryError(c, genErr)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invoice"})
		return
	}
	services.LogAudit(c, "chef.invoice.download", "order", order.ID.String(),
		nil, gin.H{"orderNumber": order.OrderNumber})
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/pdf", pdfBytes)
}

// loadChefForUser returns the ChefProfile owned by the given user.
// Wrapped so both cancel handlers can share the same not-found path.
// claimOrderItemForCancel atomically flips one order line to cancelled, guarded on
// is_cancelled=false, so exactly one of two concurrent per-line cancels wins (#576). It
// claims the line BEFORE the gateway refund (like claimRefundForProcessing / the CancelOrder
// CAS): the loser (won=false, RowsAffected==0) never reaches CreateRefund, so no second real
// refund is issued. The refund_id / refund_amount are written afterward, once the gateway
// call for the winning claim succeeds.
func claimOrderItemForCancel(tx *gorm.DB, itemID uuid.UUID, reason string, at time.Time) (bool, error) {
	res := tx.Model(&models.OrderItem{}).
		Where("id = ? AND is_cancelled = ?", itemID, false).
		Updates(map[string]interface{}{
			"is_cancelled":     true,
			"cancelled_reason": reason,
			"cancelled_at":     at,
		})
	return res.RowsAffected == 1, res.Error
}

func loadChefForUser(userID uuid.UUID) (models.ChefProfile, error) {
	var chef models.ChefProfile
	err := database.DB.Where("user_id = ?", userID).First(&chef).Error
	return chef, err
}

// roundPaise converts a rupee amount to paise (integer Razorpay unit)
// using round-half-away-from-zero so refunds match what the customer
// sees on their statement. Returns float64 only so callers can pick
// their own int width.
func roundPaise(rupees float64) float64 {
	paise := rupees * 100
	if paise >= 0 {
		return float64(int64(paise + 0.5))
	}
	return float64(int64(paise - 0.5))
}

// publishOrderCancelled stages the order.cancelled event the customer + driver
// services consume. Routed through the durable outbox so it is delivered
// reliably with retries (#131).
func publishOrderCancelled(order models.Order) {
	if err := services.EnqueueOrderEvent(database.DB, services.SubjectOrderCancelled, services.OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	}); err != nil {
		log.Printf("failed to enqueue order.cancelled: %v", err)
	}
}

// publishOrderUpdated stages the generic update event — used after per-line
// cancels since the order itself stays alive but totals have changed.
func publishOrderUpdated(order models.Order) {
	if err := services.EnqueueOrderEvent(database.DB, services.SubjectOrderUpdated, services.OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	}); err != nil {
		log.Printf("failed to enqueue order.updated: %v", err)
	}
}
