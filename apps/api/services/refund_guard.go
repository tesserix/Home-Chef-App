package services

import (
	"github.com/google/uuid"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// Typed-refund order kinds. An order spawned by a typed escrow flow must be
// refunded through that flow, never the generic order-refund endpoint.
const (
	TypedRefundMealPlanDay = "meal_plan_day"
	TypedRefundGroupOrder  = "group_order"
)

// TypedRefundOrderKind reports whether an order is refund-managed by a typed
// escrow flow — a meal-plan day (MealPlanDay.OrderID back-ref → RefundDay, key
// mealplan-refund:<dayID>) or a group order (GroupOrder.OrderID back-ref →
// participant refunds, key grouporder-refund:<id>). It returns "" for a plain
// order.
//
// #394: the generic refund endpoint (key refund:<orderID>) knows nothing about
// escrow. It keys on a disjoint idempotency space and bypasses Order.RefundAmount,
// so it would credit the customer a SECOND time for a day/group already refunded
// via its typed flow. Worse, the held chef payout for these flows is a DIRECT
// transfer the generic path does not auto-reverse — so the customer is refunded
// while the chef keeps the money. The generic endpoint must refuse these orders
// and route the caller to the correct flow.
//
// The issue:<id> credit flow is intentionally NOT covered here: it already caps
// its credit at (and increments) Order.RefundAmount, so it can never stack past
// the order total with a generic refund.
func TypedRefundOrderKind(db *gorm.DB, orderID uuid.UUID) (string, error) {
	var n int64
	if err := db.Model(&models.MealPlanDay{}).Where("order_id = ?", orderID).Count(&n).Error; err != nil {
		return "", err
	}
	if n > 0 {
		return TypedRefundMealPlanDay, nil
	}
	if err := db.Model(&models.GroupOrder{}).Where("order_id = ?", orderID).Count(&n).Error; err != nil {
		return "", err
	}
	if n > 0 {
		return TypedRefundGroupOrder, nil
	}
	return "", nil
}
