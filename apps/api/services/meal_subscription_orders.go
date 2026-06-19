package services

// meal_subscription_orders.go — daily auto-order generation for customer meal
// subscriptions (#282). At the chef's cutoff the system places that day's order(s)
// from the chef's published weekly menu (dayOfWeek × slot × variant) on the
// customer's behalf — honoring skips, halting on past_due. A missed day (chef
// no-show) credits the next cycle (reuses the #281 credit path).

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// IsMealDeliveryDay reports whether `date`'s weekday (0=Sun..6=Sat) is one of the
// subscription's selected delivery days.
func IsMealDeliveryDay(sub *models.MealSubscription, date time.Time) bool {
	wd := int64(date.Weekday())
	for _, d := range sub.Days {
		if d == wd {
			return true
		}
	}
	return false
}

// isMealDateSkipped reports whether the customer skipped this delivery date.
func isMealDateSkipped(db *gorm.DB, subID uuid.UUID, date time.Time) bool {
	var n int64
	day := date.Format("2006-01-02")
	db.Model(&models.MealSubscriptionSkip{}).
		Where("meal_subscription_id = ? AND date(date) = ?", subID, day).Count(&n)
	return n > 0
}

// weeklyDishFor resolves the chef's weekly-menu dish for a (day, slot, variant), or nil.
func weeklyDishFor(db *gorm.DB, chefID uuid.UUID, weekday int, slot models.MealSlot, variant models.MealVariant) *models.WeeklyMenuItem {
	var item models.WeeklyMenuItem
	if err := db.Where("chef_id = ? AND day_of_week = ? AND slot = ? AND variant = ?", chefID, weekday, slot, variant).
		First(&item).Error; err != nil {
		return nil
	}
	return &item
}

// GenerateMealSubscriptionDay places the order(s) for one subscription on `date`:
// one per selected slot, resolving the weekly-menu dish, creating an Order +
// fulfillment row. A skipped date records a 'skipped' fulfillment (no order, no
// charge — it was already credited). Idempotent per (sub, date, slot). Returns the
// number of orders actually placed.
func GenerateMealSubscriptionDay(db *gorm.DB, sub *models.MealSubscription, date time.Time, addr models.Address) (int, error) {
	if !MealSubGeneratesOrders(sub.Status) || !IsMealDeliveryDay(sub, date) {
		return 0, nil
	}
	skipped := isMealDateSkipped(db, sub.ID, date)
	weekday := int(date.Weekday())
	placed := 0

	for _, slotStr := range sub.Slots {
		slot := models.MealSlot(slotStr)
		// Idempotency: one fulfillment per (sub, date, slot).
		var exists int64
		db.Model(&models.MealSubscriptionFulfillment{}).
			Where("meal_subscription_id = ? AND date(date) = ? AND slot = ?", sub.ID, date.Format("2006-01-02"), slot).
			Count(&exists)
		if exists > 0 {
			continue
		}

		dish := weeklyDishFor(db, sub.ChefID, weekday, slot, sub.Variant)
		if dish == nil {
			continue // chef has no dish for this cell today — nothing to deliver
		}

		if skipped {
			db.Create(&models.MealSubscriptionFulfillment{
				MealSubscriptionID: sub.ID, CustomerID: sub.CustomerID, ChefID: sub.ChefID,
				Date: date, Slot: slot, DishName: dish.Name, Price: dish.Price, Status: models.MealFulfillSkipped,
			})
			continue
		}

		if err := db.Transaction(func(tx *gorm.DB) error {
			menuItemID := uuid.Nil
			if dish.MenuItemID != nil {
				menuItemID = *dish.MenuItemID
			}
			order := models.Order{
				OrderNumber:               mealSubOrderNumber(),
				CustomerID:                sub.CustomerID,
				ChefID:                    sub.ChefID,
				Status:                    models.OrderStatusPending,
				PaymentStatus:             models.PaymentCompleted, // covered by the cycle charge (#281)
				Currency:                  sub.Currency,
				Subtotal:                  dish.Price,
				Total:                     dish.Price,
				DeliveryAddressLine1:      addr.Line1,
				DeliveryAddressLine2:      addr.Line2,
				DeliveryAddressCity:       addr.City,
				DeliveryAddressState:      addr.State,
				DeliveryAddressPostalCode: addr.PostalCode,
				DeliveryAddressCountry:    addr.Country,
				DeliveryLatitude:          addr.Latitude,
				DeliveryLongitude:         addr.Longitude,
				EstimatedPrepTime:         30,
				ScheduledFor:              &date,
			}
			if err := tx.Create(&order).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.OrderItem{
				OrderID: order.ID, MenuItemID: menuItemID, Name: dish.Name, Price: dish.Price, Quantity: 1, Subtotal: dish.Price,
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&models.MealSubscriptionFulfillment{
				MealSubscriptionID: sub.ID, CustomerID: sub.CustomerID, ChefID: sub.ChefID,
				Date: date, Slot: slot, DishName: dish.Name, Price: dish.Price,
				Status: models.MealFulfillPlaced, OrderID: &order.ID,
			}).Error; err != nil {
				return err
			}
			ev := OrderEvent{OrderID: order.ID, OrderNumber: order.OrderNumber, CustomerID: order.CustomerID, ChefID: order.ChefID, Status: string(order.Status), Total: order.Total}
			if err := EnqueueOrderEvent(tx, SubjectOrderCreated, ev); err != nil {
				return err
			}
			return EnqueueOrderEvent(tx, SubjectChefNewOrder, ev)
		}); err != nil {
			return placed, err
		}
		placed++
	}
	return placed, nil
}

// MarkMealFulfillmentMissed flags a placed fulfillment as missed (chef no-show) and
// credits its price to the subscription's next cycle (#281). Idempotent.
func MarkMealFulfillmentMissed(db *gorm.DB, fulfillmentID uuid.UUID) error {
	var f models.MealSubscriptionFulfillment
	if err := db.First(&f, "id = ?", fulfillmentID).Error; err != nil {
		return err
	}
	if f.Status == models.MealFulfillMissed {
		return nil
	}
	res := db.Model(&models.MealSubscriptionFulfillment{}).
		Where("id = ? AND status <> ?", fulfillmentID, models.MealFulfillMissed).
		Update("status", models.MealFulfillMissed)
	if res.RowsAffected == 1 {
		return CreditMealSubscription(db, f.MealSubscriptionID, f.Price)
	}
	return nil
}

// MealAdherence is delivered vs scheduled for a subscription.
type MealAdherence struct {
	Scheduled int64 `json:"scheduled"`
	Delivered int64 `json:"delivered"`
	Missed    int64 `json:"missed"`
	Skipped   int64 `json:"skipped"`
}

// GetMealAdherence summarises fulfillment outcomes for a subscription.
func GetMealAdherence(db *gorm.DB, subID uuid.UUID) MealAdherence {
	var a MealAdherence
	q := db.Model(&models.MealSubscriptionFulfillment{}).Where("meal_subscription_id = ?", subID)
	q.Count(&a.Scheduled)
	db.Model(&models.MealSubscriptionFulfillment{}).Where("meal_subscription_id = ? AND status = ?", subID, models.MealFulfillDelivered).Count(&a.Delivered)
	db.Model(&models.MealSubscriptionFulfillment{}).Where("meal_subscription_id = ? AND status = ?", subID, models.MealFulfillMissed).Count(&a.Missed)
	db.Model(&models.MealSubscriptionFulfillment{}).Where("meal_subscription_id = ? AND status = ?", subID, models.MealFulfillSkipped).Count(&a.Skipped)
	return a
}

func mealSubOrderNumber() string {
	return fmt.Sprintf("MSO-%d-%s", time.Now().Unix(), uuid.NewString()[:6])
}
