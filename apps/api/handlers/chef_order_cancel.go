package handlers

import (
	"errors"
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
	"gorm.io/gorm/clause"
)

// errOrderRefundInProgress signals that a per-line cancel lost to a concurrent order-level
// refund (InitiateRefund / CancelOrder claimed the order's payment_status). #620.
var errOrderRefundInProgress = errors.New("order-level refund in progress")

// errLineAlreadyRefunded signals that a per-line cancel would double-refund money already
// returned via the customer-issue path — the whole order is refunded (refunded_at set) or a
// resolved issue already refunded this specific line. #622.
var errLineAlreadyRefunded = errors.New("line or order already refunded")

// ChefOrderCancelHandler holds the chef-side cancellation routes.
// Whole-order and per-line cancel both refund through the order's own gateway
// (services.IssueOrderGatewayRefund); per-line additionally recomputes the order
// totals so subsequent statements + invoices reflect the smaller scope.
//
// Gateway-agnostic: every route gates on order.GatewayRefundable() rather than
// naming a provider, so an order with nothing refundable at a gateway returns 422.
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

// CancelOrder cancels the whole order and refunds the customer the FULL total across
// every rail that funded it — wallet + loyalty credits back to the wallet instantly, the
// remainder (if any) via a gateway refund — then notifies the customer. Idempotent on a
// re-call: if the order is already cancelled with a refund ID we return 200 + the same
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

	// Any order with a refundable gateway payment can be cancelled here; naming a
	// provider would 422 an order that is perfectly refundable.
	if !order.GatewayRefundable() {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "this order has no refundable payment to cancel against; reach out to support",
		})
		return
	}

	// #609: claim + RESERVE the remaining refundable under a row lock via the shared helper —
	// the SAME atomic discipline the partial path uses. This replaces the old unlocked
	// read-then-claim (a concurrent partial could over-refund) AND the stale
	// `order.RefundAmount + refundable` write below. The reservation stamps
	// payment_status/refunded_at/refund_amount together; won=false ⇒ a sibling already refunded
	// or nothing is left, so we skip the gateway and just cancel. models.RoundAmount guarantees
	// a winning reservation is ≥ 1 paise, so amountPaise>0 whenever won.
	reserved, won, rErr := services.ReserveFullRefund(database.DB, order.ID)
	if rErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process cancellation"})
		return
	}
	amountPaise := 0
	if won {
		amountPaise = int(roundPaise(reserved))
	}

	// Mixed-payment split (checkout-credits bugfix): the reservation above records the
	// FULL refund obligation, but orders can now be part-funded by wallet + loyalty
	// credits at checkout, so the GATEWAY only ever captured (Total − WalletApplied −
	// LoyaltyApplied). Refunding the full reserved amount to the gateway fails with
	// "refund amount greater than amount captured", the cancel defers it, and the retry
	// cron re-sends the SAME wrong amount forever — the customer is never refunded. Credit
	// the wallet + loyalty slices back instantly and keep only the CARD slice for the
	// gateway; cardPaise replaces amountPaise for every gateway concern from here on.
	cardPaise := splitCancelRefundAcrossRails(&order, amountPaise)

	// #766-followup: persist the CANCELLED state — including a deferred-refund sentinel
	// when there's money owed — BEFORE the (up to 30s) synchronous gateway call. This is
	// now the recoverable checkpoint: a crash between here and the gateway call below
	// leaves an order at status=cancelled with refund_id LIKE the sentinel prefix, which
	// BOTH RetryDeferredCancelRefunds (services/deferred_cancel_refund.go) and
	// reconcileStuckRefunds can pick up and complete. The old ordering — gateway call
	// first, this persist last — left a crash-window orphan (refunded_at set,
	// status=preparing, refund_id='') that NO actor could recover: the cron needs
	// status=cancelled, and the stuck-refund reconciler needs refunded_at IS NULL. The
	// residual crash window is now just the two DB writes between ReserveFullRefund's
	// commit and this persist — no external call in between.
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":              models.OrderStatusCancelled,
		"cancelled_at":        now,
		"cancel_reason":       string(reason),
		"refund_reason":       string(reason),
		"refund_initiated_by": "chef",
	}
	sentinel := ""
	if cardPaise > 0 {
		sentinel = fmt.Sprintf("%s%d", services.DeferredCancelRefundPrefix, cardPaise)
		updates["refund_id"] = sentinel
	}
	// #609: refund_amount + refunded_at were stamped atomically by the reservation; this
	// persist only records status/cancellation bookkeeping + the deferred sentinel (never
	// re-write refund_amount — that was the stale read-modify-write that clobbered a
	// concurrent partial's increment).
	if err := database.DB.Model(&order).Updates(updates).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}

	// A chef must be able to cancel a mid-prep order even when the synchronous gateway
	// refund can't complete right now — the reservation + the sentinel just persisted
	// above already commit the full refund obligation, so nothing here can under-refund
	// the customer. When the gateway is unreachable or refuses the call, DEFER it instead of
	// blocking the cancel: the sentinel already encodes the owed paise, and
	// RetryDeferredCancelRefunds (services/deferred_cancel_refund.go) re-issues the SAME
	// idempotency-keyed refund on a cron until it lands. Do NOT release the reservation on
	// a deferral — the refund is still owed, just not yet issued.
	refundID := sentinel
	deferred := false
	if cardPaise > 0 {
		if !services.GatewayRefundAvailable(&order) {
			deferred = true
			log.Printf("chef cancel: %s client unavailable for order %s; deferring refund of %d paise to the retry cron",
				models.NormalizeProvider(order.PaymentProvider), order.ID, cardPaise)
		} else {
			refundResp, err := services.IssueOrderGatewayRefund(&order, cardPaise, map[string]string{
				"order_id":  order.ID.String(),
				"order_no":  order.OrderNumber,
				"chef_id":   chef.ID.String(),
				"reason":    string(reason),
				"initiator": "chef",
				// Full-order cancel refund is issued once (order goes terminal-cancelled);
				// the reservation above serializes concurrent attempts, and the retry cron
				// re-sends this SAME key on a deferral — so a lost-response success dedups
				// at the gateway instead of double-refunding. #574.
			}, services.RefundFullIdempotencyKey(order.ID))
			if err != nil {
				deferred = true
				log.Printf("chef cancel: gateway refund failed for order %s (%d paise); deferring to the retry cron: %v", order.ID, cardPaise, err)
			} else {
				refundID = refundResp.RefundID
				// Guarded replace: swap the sentinel for the real gateway id ONLY if it's
				// still there — a concurrent cron/Temporal completion that already replaced
				// it first is a harmless no-op (mirrors retryOneDeferredCancelRefund's
				// guarded UPDATE in deferred_cancel_refund.go).
				if uErr := database.DB.Model(&models.Order{}).
					Where("id = ? AND refund_id LIKE ?", order.ID, services.DeferredCancelRefundPrefix+"%").
					Update("refund_id", refundID).Error; uErr != nil {
					log.Printf("chef cancel: failed to replace deferred sentinel with the real refund id for order %s: %v", order.ID, uErr)
				}
			}
		}
	}
	if deferred {
		// Fire the durable Temporal retry immediately instead of waiting for the next
		// RetryDeferredCancelRefunds cron tick (up to ~12 minutes) — that cron still
		// backstops if Temporal is down or this workflow never completes. Both paths
		// call the gateway with the SAME idempotency key, so they can never double-refund.
		services.StartDeferredRefundFlow(order.ID, order.GatewayRefundReference(), cardPaise)
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

	// order.RefundID post-refresh reflects the FINAL state: the real gateway id when the
	// guarded replace above landed, or the sentinel when the refund is still deferred.
	// #834 item 6: a chef who cancels close to service is charged a percentage of the order,
	// netted off their next weekly settlement. Guarded by a grace allowance and waivable by an
	// admin (services/chef_penalty.go). Best-effort BY CONTRACT — the customer's refund is the
	// money that matters and is already committed above; a levy that fails to record is
	// recoverable, a cancel that 500s because of it is not.
	levied := false
	if p, pErr := services.LevyChefCancelPenalty(database.DB, chef.ID, userID, order.ID,
		order.OrderNumber, order.Total, chefCancelLeadHours(&order, now)); pErr != nil {
		log.Printf("chef cancel: penalty levy failed for order %s: %v", order.OrderNumber, pErr)
		services.CaptureBackgroundError(pErr)
	} else if p != nil {
		levied = true
		services.LogAudit(c, "chef.penalty.levy", "chef_penalty", p.ID.String(), nil,
			gin.H{"orderId": order.ID.String(), "amount": p.Amount, "leadHours": p.LeadHours})
	}
	// #885: the payment gateway's transaction-fee loss on THIS Cashfree refund, recovered from
	// the chef through the same best-effort mechanism as the cancel_late levy above. Cashfree
	// only; the stacking/grace guards inside LevyGatewayFeePenalty already prevent double-
	// charging an order that just raised a cancel_late levy.
	if fp, fErr := services.LevyGatewayFeePenalty(database.DB, chef.ID, userID, order.ID,
		order.OrderNumber, models.NormalizeProvider(order.PaymentProvider), services.FromPaise(cardPaise),
		services.RefundFullIdempotencyKey(order.ID)); fErr != nil {
		log.Printf("chef cancel: gateway-fee levy failed for order %s: %v", order.OrderNumber, fErr)
		services.CaptureBackgroundError(fErr)
	} else if fp != nil {
		services.LogAudit(c, "chef.penalty.levy", "chef_penalty", fp.ID.String(), nil,
			gin.H{"orderId": order.ID.String(), "amount": fp.Amount, "kind": "gateway_fee"})
	}
	// The customer is told what happened and that they are made whole — the notification the
	// policy calls for. order.RefundAmount is post-refresh, so it is the full reserved amount.
	services.NotifyCustomerOfChefCancelPenalty(order.CustomerID, order.OrderNumber, order.RefundAmount, levied)

	services.LogAudit(c, "chef.order.cancel", "order", order.ID.String(),
		nil, gin.H{"reason": string(reason), "refundAmount": order.RefundAmount, "refundId": order.RefundID})

	publishOrderCancelled(order)

	// Cancel any booked 3PL delivery (no-op if none / already terminal). Off
	// the response path; failure must not fail the order cancellation/refund.
	//
	// The handle is captured HERE, on the request goroutine, and passed in — the
	// detached goroutine below outlives this request, so reading database.DB
	// inside it would read whatever the global is by then.
	db := database.DB
	go func() {
		if err := services.CancelOrderDelivery(db, order.ID, string(reason)); err != nil {
			log.Printf("Failed to cancel 3PL delivery for order %s: %v", order.ID, err)
			services.CaptureBackgroundError(err)
		}
	}()

	c.JSON(http.StatusOK, order.ToChefResponse())
}

// chefCancelLeadHours is how much notice the customer got when the chef cancelled: the hours
// between now and the order's scheduled service time.
//
// An order with NO scheduled time is on-demand — the chef accepted it and the kitchen is
// cooking it right now — so its lead is 0 and it falls inside any penalty window. Treating an
// unscheduled order as "infinite notice" would exempt the single most disruptive cancellation
// there is: a hungry customer waiting on an order already in prep.
func chefCancelLeadHours(order *models.Order, now time.Time) float64 {
	if order.ScheduledFor == nil {
		return 0
	}
	return order.ScheduledFor.Sub(now).Hours()
}

// splitCancelRefundAcrossRails credits the wallet + loyalty slices of a chef-cancel
// refund to the customer's wallet INSTANTLY and returns the remaining CARD slice — the
// only piece that still needs to go through the payment gateway. It mirrors the
// funding-rail split in services/orderrefund_gateway.go: a mixed-payment order (checkout
// credits) only ever CAPTURED (Total − WalletApplied − LoyaltyApplied) at the gateway, so
// handing the gateway the full reserved amount fails ("refund amount greater than amount
// captured") and — pre-fix — left the customer permanently unrefunded while the retry
// cron kept re-sending the same wrong amount. Both credit legs are best-effort: a failure
// here must never fail the cancel, since the reservation already recorded the FULL amount
// owed and reconciliation backstops a missed credit (mirroring the gateway adapter).
func splitCancelRefundAcrossRails(order *models.Order, amountPaise int) (cardPaise int) {
	if amountPaise <= 0 {
		return 0
	}
	split := services.SplitRefundByFunding(order, amountPaise)

	credit := func(paise int, source models.WalletTxnSource, label string) float64 {
		if paise <= 0 {
			return 0
		}
		amount := services.FromPaise(paise)
		if _, err := services.CreditWallet(database.DB, order.CustomerID, amount,
			source, &order.ID,
			"chef cancel refund for order "+order.OrderNumber,
			label+":cancel:"+order.ID.String(), nil); err != nil {
			log.Printf("chef cancel: %s slice re-credit failed order=%s: %v", label, order.OrderNumber, err)
			services.CaptureBackgroundError(err)
			return 0
		}
		return amount
	}
	walletBack := credit(split.WalletPaise, models.WalletSourceRefund, "refund-wallet")
	loyaltyBack := credit(split.LoyaltyPaise, models.WalletSourceLoyalty, "refund-loyalty")

	// Record what each rail has now been returned — mirrors orderrefund_gateway.go so a
	// later partial (e.g. a goodwill top-up) computes its share against the REMAINING
	// funded amount rather than double-counting the original.
	if walletBack > 0 || loyaltyBack > 0 {
		if err := database.DB.Model(&models.Order{}).Where("id = ?", order.ID).
			Updates(map[string]any{
				"wallet_refunded":  order.WalletRefunded + walletBack,
				"loyalty_refunded": order.LoyaltyRefunded + loyaltyBack,
			}).Error; err != nil {
			log.Printf("chef cancel: stamp refunded rails failed order=%s: %v", order.OrderNumber, err)
			services.CaptureBackgroundError(err)
		}
	}

	// The gateway can never refund more than it captured. Pro-rata already keeps the
	// card slice within the captured amount, but the credit caps above (a rail already
	// partly refunded elsewhere) can push their remainder onto the card; clamp and
	// return any excess as credit so the customer is still made whole.
	cardPaise = split.CardPaise
	capturePaise := services.ToPaise(order.Total) - services.ToPaise(order.WalletApplied) - services.ToPaise(order.LoyaltyApplied)
	if cardPaise > capturePaise {
		if over := cardPaise - capturePaise; over > 0 {
			credit(over, models.WalletSourceRefund, "refund-overflow")
		}
		cardPaise = capturePaise
	}
	return cardPaise
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

	if !order.GatewayRefundable() {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "this order has no refundable payment to issue a per-line refund against",
		})
		return
	}

	// Tax share for this line = foodTax * (line.Subtotal / order.Subtotal), computed
	// against the ORIGINAL subtotal/food tax so concurrent partial cancels can't drift
	// the proportional split. (see lineRefundAmount)
	lineRefund := lineRefundAmount(target.Subtotal, order.Subtotal, services.ChefTaxOf(order.Tax, order.TaxFood, order.TaxService))
	amountPaise := int(roundPaise(lineRefund))
	if amountPaise <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "computed refund amount is zero; nothing to do"})
		return
	}

	if !services.GatewayRefundAvailable(&order) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment gateway unavailable; refund deferred"})
		return
	}

	now := time.Now().UTC()
	// #628/#576: RESERVE the line BEFORE the gateway call — the same atomic discipline the
	// order-level paths use (ReserveRefund / the CancelOrder CAS). reserveOrderItemForCancel
	// flips is_cancelled=false→true AND records the whole ledger (line.refund_amount + reduced
	// order subtotal/tax/total + order.refund_amount increment) in one order-locked txn, so a
	// concurrent RefundIssueToWallet immediately sees the reservation via its
	// RemainingRefundable / cancelledAffected caps (no gateway-window where the line reads
	// cancelled-but-unrefunded). Two concurrent cancels of the SAME line: only the winner flips
	// the row and reaches CreateRefund; the loser (won=false) never issues a second refund.
	var won bool
	var reservedRefund float64
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		r, w, e := reserveOrderItemForCancel(tx, order.ID, target.ID, string(reason), now)
		reservedRefund, won = r, w
		return e
	}); err != nil {
		if errors.Is(err, errOrderRefundInProgress) {
			// #620: an order-level refund holds the order — don't per-line-refund on top of it.
			c.JSON(http.StatusConflict, gin.H{"error": "an order-level refund is in progress; retry the line cancel shortly"})
			return
		}
		if errors.Is(err, errLineAlreadyRefunded) {
			// #622: this line / the whole order was already refunded via a customer issue —
			// refunding the line again would double-refund. Route to admin for any adjustment.
			c.JSON(http.StatusConflict, gin.H{"error": "this item or order has already been refunded via a customer issue; resolve any adjustment through admin"})
			return
		}
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
	// The reservation recomputed the refund under the lock; use it for the gateway so the amount
	// matches exactly what the ledger recorded (they agree with the pre-check by the invariant
	// effective rate; the reserved value is authoritative).
	amountPaise = int(roundPaise(reservedRefund))

	refundResp, err := services.IssueOrderGatewayRefund(&order, amountPaise, map[string]string{
		"order_id":      order.ID.String(),
		"order_item_id": target.ID.String(),
		"chef_id":       chef.ID.String(),
		"reason":        string(reason),
		"initiator":     "chef",
		"scope":         "line",
		// #574: keyed by the immutable line id — a lost-response retry dedups to one refund.
	}, services.RefundLineIdempotencyKey(order.ID, target.ID))
	if err != nil {
		// The gateway refused — no money moved, so RELEASE the reservation (un-cancel the line +
		// restore the order totals) so a retry can cancel this line cleanly.
		releaseOrderItemCancelReservation(database.DB, order.ID, target.ID, reservedRefund)
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "refund failed at gateway; please retry"})
		return
	}

	// Record the gateway refund id + release the line's reserved daily capacity (#48). The money
	// ledger was already committed atomically by the reservation, so this is a non-money-critical
	// follow-up: on failure the row is left STUCK (do NOT revert — the gateway refund succeeded
	// and the reservation's refund_amount is already correct, so reverting would erase a real
	// refund, the #602/#615 corruption trap). A retry hits the early is_cancelled idempotent 200;
	// only the refund_id reference + capacity release are lost, which is money-safe.
	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.OrderItem{}).Where("id = ?", target.ID).
			Update("refund_id", refundResp.RefundID).Error; err != nil {
			return err
		}
		return services.ReleaseCapacity(tx, target.MenuItemID, target.Quantity, services.CapacityDay(order.CreatedAt))
	})
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}
	lineRefund = reservedRefund // audit log uses the reserved (authoritative) amount

	// #885: best-effort gateway-fee levy for this Cashfree line refund, keyed the same as the
	// gateway call above so a retry can't double-charge the chef.
	if fp, fErr := services.LevyGatewayFeePenalty(database.DB, chef.ID, userID, order.ID,
		order.OrderNumber, models.NormalizeProvider(order.PaymentProvider), reservedRefund,
		services.RefundLineIdempotencyKey(order.ID, target.ID)); fErr != nil {
		log.Printf("chef cancel item: gateway-fee levy failed for order %s: %v", order.OrderNumber, fErr)
		services.CaptureBackgroundError(fErr)
	} else if fp != nil {
		services.LogAudit(c, "chef.penalty.levy", "chef_penalty", fp.ID.String(), nil,
			gin.H{"orderId": order.ID.String(), "amount": fp.Amount, "kind": "gateway_fee"})
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

	if !order.GatewayRefundable() {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "this order has no refundable payment to refund against",
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

	if !services.GatewayRefundAvailable(&order) {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "payment gateway unavailable; refund deferred"})
		return
	}

	// #600: dedup a client retry-after-success. The gateway idempotency key only covers the
	// server↔gateway timeout window; once this goodwill refund fully succeeds, RefundAmount
	// advances, so the chef app re-submitting the identical "refund ₹X" (its HTTP response was
	// dropped) would compute a new key + re-reserve → a second real refund. Claim the submission
	// BEFORE the reserve (keyed by the chef's Idempotency-Key header when present, else the
	// order+amount+reason within a short window). A duplicate returns the current state without
	// refunding. Released on any pre-commit failure so a legit retry re-attempts; KEPT once the
	// money moves so a retry dedups.
	clientKey := services.NormalizeRefundClientKey(c.GetHeader("Idempotency-Key"))
	proceed, dedupKey, dErr := services.ClaimRefundRequest(database.DB, order.ID, clientKey, int64(roundPaise(req.Amount)), req.Reason)
	if dErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process refund"})
		return
	}
	if !proceed {
		_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error
		c.JSON(http.StatusOK, order.ToChefResponse())
		return
	}
	releaseClaim := func() { services.ReleaseRefundRequestClaim(database.DB, dedupKey) }

	// #611: reserve the refund UNDER A ROW LOCK (read remaining + claim payment_status +
	// increment refund_amount atomically) instead of the old unlocked RemainingRefundable
	// read + separate claimRefundForProcessing. A concurrent goodwill + customer refund can
	// no longer read the same stale `remaining` and collectively over-refund. requested =
	// the goodwill amount (binding gt=0); fullRefund is true only when it exhausts the
	// remaining. priorRefunded is the locked prior-cumulative basis for the gateway key.
	reserved, priorRefunded, fullRefund, won, rErr := services.ReserveRefund(database.DB, order.ID, req.Amount)
	if rErr != nil {
		releaseClaim()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not process refund"})
		return
	}
	if !won {
		releaseClaim()
		c.JSON(http.StatusConflict, gin.H{"error": "a refund for this order is already in progress or has completed"})
		return
	}
	revertReservation := func() {
		services.ReleaseRefundReservation(database.DB, order.ID, reserved)
		releaseClaim()
	}
	amountPaise := int(roundPaise(reserved))
	if amountPaise <= 0 {
		revertReservation()
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount rounds to zero paise"})
		return
	}

	refundResp, err := services.IssueOrderGatewayRefund(&order, amountPaise, map[string]string{
		"order_id":  order.ID.String(),
		"order_no":  order.OrderNumber,
		"chef_id":   chef.ID.String(),
		"reason":    req.Reason,
		"initiator": "chef",
		"scope":     "post_delivery_goodwill",
		// #574/#576/#611: keyed by prior cumulative refunded paise (stable on retry, distinct
		// across sequential partials) — safe because the reservation above serializes a
		// same-order double-submit under a row lock so the ledger can't double-count.
	}, services.RefundPartialIdempotencyKey(order.ID, services.ToPaise(priorRefunded)))
	if err != nil {
		revertReservation()
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "refund failed at gateway; please retry"})
		return
	}

	// Persist the refund. Target the row by id (not the preloaded &order) so a refund
	// doesn't spuriously upsert the belongs-to Chef association. #611: refund_amount was
	// already incremented atomically by ReserveRefund — never re-write it here (the stale
	// read-modify-write that clobbered a concurrent partial's increment). #549: a PARTIAL
	// refund must NOT stamp refunded_at (the release-side guards block the WHOLE chef hold on
	// refunded_at IS NOT NULL) and reverts payment_status→completed so the hold stays
	// releasable + sequential partials re-claim; a FULL refund keeps refunded + stamps
	// refunded_at (terminal).
	refundUpdates := map[string]interface{}{
		"refund_id":           refundResp.RefundID,
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
		// #602: the gateway refund already succeeded and services.ReserveRefund already
		// committed `refund_amount += reserved` in its OWN tx, so refund_amount is CORRECT
		// here. Do NOT release the reservation: decrementing it back would ERASE a refund that
		// actually happened → the next distinct refund over-refunds and collides the
		// amount-based idempotency key (razorpay rejects → stuck; wallet silently
		// under-credits). Leave the order STUCK at refunded with the ledger correct;
		// reconcileStuckRefunds finalizes it (payment_status=refunded AND refunded_at IS NULL).
		// The cross-guard below still blocks the payout meanwhile. (The GATEWAY-error revert
		// above is different — there the money did NOT move, so undoing the reserve is correct.)
		log.Printf("goodwill refund persist failed for order %s: %v", order.ID, persistErr)
		services.CaptureBackgroundError(persistErr)
	}
	// Cross-guard the payout hold (#457/#549/#568) — a FULL refund drives the whole hold to
	// withheld/reversed (unconditional safety net, runs even on persist failure so the chef
	// is never paid for a refunded order); a PARTIAL claws back only the refunded portion
	// and only when the refund persisted (a retry can't double-claw). Best-effort. #611: claw
	// the RESERVED (capped) amount, not the raw request — they differ only if a concurrent
	// refund shrank the remaining under the lock, in which case reserved is what was refunded.
	if hErr := crossGuardRefundHold(order.ID, reserved, req.Reason, fullRefund, persistErr == nil); hErr != nil {
		log.Printf("payout cross-guard failed for goodwill-refunded order %s: %v", order.ID, hErr)
	}
	// #885: best-effort gateway-fee levy for this Cashfree goodwill refund. Fires regardless of
	// whether OUR persist above succeeded, since the GATEWAY refund already happened by this
	// point — the gateway's fee was incurred either way.
	if fp, fErr := services.LevyGatewayFeePenalty(database.DB, chef.ID, userID, order.ID,
		order.OrderNumber, models.NormalizeProvider(order.PaymentProvider), reserved,
		services.RefundPartialIdempotencyKey(order.ID, services.ToPaise(priorRefunded))); fErr != nil {
		log.Printf("goodwill refund: gateway-fee levy failed for order %s: %v", order.ID, fErr)
		services.CaptureBackgroundError(fErr)
	} else if fp != nil {
		services.LogAudit(c, "chef.penalty.levy", "chef_penalty", fp.ID.String(), nil,
			gin.H{"orderId": order.ID.String(), "amount": fp.Amount, "kind": "gateway_fee"})
	}
	if persistErr != nil {
		// Honest signal: the gateway refunded but the terminal write didn't save. The reserve
		// durably recorded refund_amount, so the order is left STUCK at refunded (ledger
		// correct) for reconcileStuckRefunds to finalize — not reverted (#602).
		c.JSON(http.StatusInternalServerError, gin.H{"error": "refund completed but state save failed; see ops"})
		return
	}
	_ = database.DB.Preload("Items").First(&order, "id = ?", order.ID).Error

	services.LogAudit(c, "chef.order.refund", "order", order.ID.String(),
		nil, gin.H{"amount": reserved, "reason": req.Reason, "refundId": refundResp.RefundID})

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
// reserveOrderItemForCancel atomically claims one order line for cancellation AND records its
// refund on the ledger — the line's refund_amount, the reduced order subtotal/tax/total, and the
// order-level refund_amount increment — all in ONE transaction under the order lock. It returns
// the line refund amount (for the gateway call) and won=false when a sibling already claimed the
// line (idempotent, no error).
//
// #628 — why the whole ledger write lives in the claim: the pre-#628 flow flipped is_cancelled
// here but wrote refund_amount + reduced orders.total only AFTER the (unlocked) gateway call. In
// that window a concurrent services.RefundIssueToWallet locking the order read
// PerLineRefundedTotalTx=0 and an un-reduced Total, so its RemainingRefundable / cancelledAffected
// caps saw the cancelled line's value as STILL refundable and could refund it a second time.
// Writing the whole reservation inside the claim (both paths lock the same orders row FOR UPDATE)
// makes it visible to those caps the instant this commits — no window remains.
//
// #576: guarded is_cancelled=false→true CAS, so exactly one of two concurrent per-line cancels of
// the SAME line wins; the loser never reaches CreateRefund. The refund_id is written afterward,
// once the winning claim's gateway call succeeds.
func reserveOrderItemForCancel(tx *gorm.DB, orderID, itemID uuid.UUID, reason string, at time.Time) (lineRefund float64, won bool, err error) {
	// #620: serialize with the order-level refund mutex. Lock the order row FIRST (order-first
	// ordering per #585) and require it still payment_status=completed — InitiateRefund /
	// CancelOrder flip it to refunded while reserving, so a per-line cancel racing an
	// order-level refund on the SAME order loses here (errOrderRefundInProgress) instead of
	// issuing a second real gateway refund and double-counting refund_amount (customer
	// over-refund). A partial order-level refund reverts payment_status→completed, so the
	// line cancel just retries once it settles.
	lockTx := tx
	if tx.Dialector.Name() == "postgres" {
		lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var o models.Order
	if e := lockTx.Select("id", "subtotal", "tax", "tax_food", "tax_service", "total", "refund_amount", "payment_status", "refunded_at").
		First(&o, "id = ?", orderID).Error; e != nil {
		return 0, false, e
	}
	if o.PaymentStatus != models.PaymentCompleted {
		return 0, false, errOrderRefundInProgress
	}
	// #622: block a per-line refund that would double-refund money already returned through the
	// customer-issue path (services.RefundIssueToWallet), which — unlike the ReserveRefund
	// family — does NOT flip payment_status, so the #620 mutex above can't see it.
	//   (a) A FULL issue refund stamps refunded_at (but leaves payment_status=completed): the
	//       whole order was already refunded, so no line may be refunded again.
	//   (b) A PARTIAL issue refund naming this line (OrderIssue.AffectedItemIDs) already
	//       returned that line's money to the customer's wallet: refunding the line again is a
	//       straight double-refund. Both are checked under the same order lock so a concurrent
	//       issue refund (which also locks the order) can't slip between check and claim.
	if o.RefundedAt != nil {
		return 0, false, errLineAlreadyRefunded
	}
	if refunded, rErr := lineRefundedViaIssue(tx, orderID, itemID); rErr != nil {
		return 0, false, rErr
	} else if refunded {
		return 0, false, errLineAlreadyRefunded
	}

	// Read the target line's subtotal under the lock (targeted Select, not a full Preload, so
	// the reserve stays column-light for the sqlite harness). A line already cancelled is an
	// idempotent no-op (won=false, no error) — the caller responds with the current state.
	var item models.OrderItem
	if e := tx.Select("id", "subtotal", "is_cancelled").First(&item, "id = ? AND order_id = ?", itemID, orderID).Error; e != nil {
		return 0, false, e
	}
	if item.IsCancelled {
		return 0, false, nil
	}
	lineRefund = lineRefundAmount(item.Subtotal, o.Subtotal, services.ChefTaxOf(o.Tax, o.TaxFood, o.TaxService))

	// Guarded CAS: flip the line AND record its refund together. RowsAffected!=1 ⇒ a concurrent
	// duplicate won → won=false, no ledger change.
	res := tx.Model(&models.OrderItem{}).
		Where("id = ? AND is_cancelled = ?", itemID, false).
		Updates(map[string]interface{}{
			"is_cancelled":     true,
			"cancelled_reason": reason,
			"cancelled_at":     at,
			"refund_amount":    lineRefund,
		})
	if res.Error != nil {
		return 0, false, res.Error
	}
	if res.RowsAffected != 1 {
		return 0, false, nil
	}

	// Reduce the order totals via deltas. This is exactly recomputeOrderTotals with only THIS
	// line newly cancelled (under the lock no other item changed): Total drops by lineRefund,
	// subtotal by the line subtotal, tax by the line's proportional share — so the fees/tip/
	// discount need not be read, and the effective tax rate is preserved. refund_amount is the
	// same increment the old txn3 applied. All three RemainingRefundable inputs move together.
	//
	// D-19: the share is of the FOOD GST only, so tax_food carries the whole reduction and the
	// GST on the surviving fee/delivery supplies stays on the order. The delta must mirror the
	// refund exactly or Total stops equalling the sum of its parts. Pre-split orders have no
	// tax_food to move (their basis is the whole Tax, which is how they were charged).
	taxShare := lineRefund - item.Subtotal // = foodTax * item.Subtotal / o.Subtotal
	orderUpdates := map[string]interface{}{
		"subtotal":      o.Subtotal - item.Subtotal,
		"tax":           o.Tax - taxShare,
		"total":         o.Total - lineRefund,
		"refund_amount": o.RefundAmount + lineRefund,
	}
	if o.TaxFood > 0 {
		orderUpdates["tax_food"] = o.TaxFood - taxShare
	}
	if e := tx.Model(&models.Order{}).Where("id = ?", orderID).Updates(orderUpdates).Error; e != nil {
		return 0, false, e
	}
	return lineRefund, true, nil
}

// releaseOrderItemCancelReservation undoes a winning reserveOrderItemForCancel when the
// downstream gateway refund FAILS (the money did NOT move, so undoing the reservation is correct
// — mirroring the gateway-error revert the order-level paths use). It is the exact inverse of the
// reserve's ledger write: un-cancels the line + zeroes its refund_amount, adds the line back into
// the order subtotal/tax/total, and decrements the order-level refund_amount. Best-effort; the
// caller is already returning the gateway failure.
//
// It is NOT called on a post-gateway persist failure: there the gateway refund succeeded and the
// reserve already committed the correct refund_amount, so reverting would ERASE a real refund
// (#602/#615 corruption trap) — that row is left stuck-but-ledger-correct instead.
//
// SAFE-REVERT GUARD (#628 verify): the reserve commits its ledger BEFORE the (unlocked) gateway
// call, so while that call is in flight a concurrent ORDER-LEVEL full refund can lock the order,
// compute its remaining OFF this reservation (Total already reduced, refund_amount already bumped),
// refund it, and mark the order terminal. If our gateway then fails, blindly reverting would
// restore Total/refund_amount that the concurrent refund already consumed → a terminal order with
// overstated RemainingRefundable and money silently stranded (reconciliation sees no gateway-vs-
// refund_amount drift). So we revert ONLY while the order is still payment_status=completed AND
// refunded_at IS NULL — the state the reserve left it. Every full-refund path (ReserveFullRefund /
// ReserveRefund-fullRefund / RefundIssueToWallet-full / ExecuteCancellationRefund) stamps
// refunded_at (or flips payment_status→refunded) as its claim, so this catches every terminal
// transition; a concurrent PARTIAL refund keeps completed/NULL and its independent refund_amount
// delta survives our arithmetic inverse cleanly. If the order has moved on we leave the reservation
// STUCK (line cancelled + refund_amount recorded, Total reduced) and log loudly — reconciliation
// then flags the gateway-vs-refund_amount drift for manual repair (never silently lost). Held under
// the order lock so the check + revert can't itself race a concurrent refund.
func releaseOrderItemCancelReservation(db *gorm.DB, orderID, itemID uuid.UUID, lineRefund float64) {
	if err := db.Transaction(func(tx *gorm.DB) error {
		lockTx := tx
		if tx.Dialector.Name() == "postgres" {
			lockTx = tx.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		var o models.Order
		if e := lockTx.Select("id", "subtotal", "tax", "tax_food", "tax_service", "tax_delivery", "total", "refund_amount", "payment_status", "refunded_at").
			First(&o, "id = ?", orderID).Error; e != nil {
			return e
		}
		if o.PaymentStatus != models.PaymentCompleted || o.RefundedAt != nil {
			// A concurrent order-level refund consumed this reservation and moved the order
			// terminal — reverting now would corrupt the ledger + strand money. Leave the
			// reservation stuck for reconciliation to flag.
			log.Printf("per-line cancel reservation for item %s NOT released: order %s moved terminal "+
				"(payment_status=%s refunded_at_set=%t) during the gateway window — left stuck for reconciliation",
				itemID, orderID, o.PaymentStatus, o.RefundedAt != nil)
			return nil
		}
		var item models.OrderItem
		if e := tx.Select("id", "subtotal", "is_cancelled").First(&item, "id = ? AND order_id = ?", itemID, orderID).Error; e != nil {
			return e
		}
		if !item.IsCancelled {
			return nil // already released — idempotent
		}
		if e := tx.Model(&models.OrderItem{}).Where("id = ? AND is_cancelled = ?", itemID, true).
			Updates(map[string]interface{}{
				"is_cancelled":     false,
				"cancelled_reason": "",
				"cancelled_at":     nil,
				"refund_amount":    0,
			}).Error; e != nil {
			return e
		}
		// Exact inverse of the reserve's delta, tax_food included — the share is derived
		// from the reserved refund, so it stays symmetric without re-deriving the basis.
		taxShare := lineRefund - item.Subtotal
		restore := map[string]interface{}{
			"subtotal":      o.Subtotal + item.Subtotal,
			"tax":           o.Tax + taxShare,
			"total":         o.Total + lineRefund,
			"refund_amount": o.RefundAmount - lineRefund,
		}
		// The reserve moved tax_food iff the order carries a per-supply snapshot; a
		// snapshot whose every component is now zero has no split left to restore.
		if taxShare > 0 && (o.TaxFood > 0 || o.TaxService > 0 || o.TaxDelivery > 0) {
			restore["tax_food"] = o.TaxFood + taxShare
		}
		return tx.Model(&models.Order{}).Where("id = ?", orderID).Updates(restore).Error
	}); err != nil {
		log.Printf("failed to release per-line cancel reservation for item %s: %v", itemID, err)
	}
}

// lineRefundedViaIssue reports whether a resolved/auto-refunded customer OrderIssue that
// actually credited money (refund_amount > 0) named this order line in AffectedItemIDs — i.e.
// the line's money was already returned to the customer via the issue path, so a per-line
// cancel-refund on top of it would double-refund. Read on the passed tx (under the order lock)
// so it sees a concurrent issue refund that committed first. #622.
func lineRefundedViaIssue(tx *gorm.DB, orderID, itemID uuid.UUID) (bool, error) {
	var issues []models.OrderIssue
	if err := tx.Select("affected_item_ids").
		Where("order_id = ? AND refund_amount > 0 AND status IN ?", orderID,
			[]models.IssueStatus{models.IssueAutoRefunded, models.IssueResolved}).
		Find(&issues).Error; err != nil {
		return false, err
	}
	want := itemID.String()
	for i := range issues {
		for _, aid := range issues[i].AffectedItemIDs {
			if aid == want {
				return true, nil
			}
		}
	}
	return false, nil
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
