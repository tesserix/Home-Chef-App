package services

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// order_source.go — classify an order's origin (#435) so the vendor feed can
// group à-la-carte, meal-plan-day, subscription-day, and group orders. The
// Order row has no type column; the origin is a REVERSE link — the order id
// appears as order_id in meal_plan_days / meal_subscription_fulfillments /
// group_orders. We resolve a whole page of orders in three batch queries (no
// N+1) rather than a per-order lookup.

// ClassifyOrderSources returns the source for each order id. Every id defaults
// to alacarte; a reverse-link match promotes it. An order belongs to at most one
// origin, so the assignment order doesn't matter. Returns an all-alacarte map on
// empty input.
func ClassifyOrderSources(db *gorm.DB, orderIDs []uuid.UUID) map[uuid.UUID]models.OrderSource {
	out := make(map[uuid.UUID]models.OrderSource, len(orderIDs))
	for _, id := range orderIDs {
		out[id] = models.OrderSourceAlacarte
	}
	if len(orderIDs) == 0 {
		return out
	}

	assign := func(model any, src models.OrderSource) {
		var ids []uuid.UUID
		// order_id IN (…) already excludes NULLs, so only real links match.
		db.Model(model).Where("order_id IN ?", orderIDs).Distinct().Pluck("order_id", &ids)
		for _, id := range ids {
			out[id] = src
		}
	}
	assign(&models.MealPlanDay{}, models.OrderSourceMealPlan)
	assign(&models.MealSubscriptionFulfillment{}, models.OrderSourceSubscription)
	assign(&models.GroupOrder{}, models.OrderSourceGroup)
	return out
}
