package services

import (
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// order_payout.go — release a delivered REGULAR order's held chef/rider Route
// transfers (#217). Regular-order payouts are payment-linked transfers created
// OnHold at checkout (orderSettlements); this releases them once the food is
// delivered, closing the long-standing gap where they were never auto-released.
//
// Gated by ORDER_PAYOUT_AUTO_RELEASE_ENABLED (default OFF) because it moves live
// settlement — verify capture→hold→release in the Razorpay sandbox (#218) first.
// No-op for orders without a Razorpay order id (meal-plan/group consolidated
// orders settle through their own release paths), so it never double-releases.

// ReleaseOrderPayouts releases any held Route transfers on a delivered order.
func ReleaseOrderPayouts(orderID uuid.UUID) {
	if config.AppConfig == nil || !config.AppConfig.OrderPayoutAutoReleaseEnabled {
		return
	}
	var order models.Order
	if err := database.DB.Select("id", "razorpay_order_id").
		First(&order, "id = ?", orderID).Error; err != nil {
		return
	}
	if order.RazorpayOrderID == "" {
		return // not a gateway-charged regular order (e.g. meal-plan/group)
	}
	rz := GetRazorpay()
	if rz == nil {
		return
	}
	transfers, err := rz.FetchOrderTransfers(order.RazorpayOrderID)
	if err != nil {
		log.Printf("order-payout: fetch transfers for order %s failed: %v", orderID, err)
		return
	}
	for _, t := range transfers {
		if t.OnHold && t.ID != "" {
			if _, err := rz.ReleaseTransfer(t.ID); err != nil {
				log.Printf("order-payout: release transfer %s (order %s) failed: %v", t.ID, orderID, err)
			}
		}
	}
}
