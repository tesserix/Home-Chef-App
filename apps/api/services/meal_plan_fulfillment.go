package services

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// meal_plan_fulfillment.go — turns confirmed tiffin plans into the normal order
// pipeline (#197). A cron generates one scheduled Order per confirmed day as its
// date approaches (orderGenLeadTime); the existing prep/deliver flow then drives
// it, and the delivery hook (MarkMealPlanDayDelivered) marks the day delivered +
// releases its escrow payout. A plan flips active on first generation and
// completed once every day reaches a terminal state.
const (
	mealPlanFulfillmentInterval = 15 * time.Minute
	// Generate a day's order this far ahead of its date so the chef sees it in time.
	orderGenLeadTime = 24 * time.Hour
)

// StartMealPlanFulfillmentCron is the legacy in-process fallback (Temporal off).
func StartMealPlanFulfillmentCron(ctx context.Context) {
	go func() {
		runMealPlanFulfillment(ctx)
		t := time.NewTicker(mealPlanFulfillmentInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				log.Println("meal-plan-fulfillment: shutting down on ctx cancel")
				return
			case <-t.C:
				runMealPlanFulfillment(ctx)
			}
		}
	}()
	log.Println("meal-plan-fulfillment: cron started (interval=15m)")
}

func runMealPlanFulfillment(_ context.Context) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("meal-plan-fulfillment: panic recovered: %v", r)
		}
	}()
	generateDueDayOrders()
	completeFinishedPlans()
}

// generateDueDayOrders creates an Order for each confirmed day coming due that
// doesn't yet have one, and flips the plan active.
func generateDueDayOrders() {
	due := time.Now().Add(orderGenLeadTime)
	var plans []models.MealPlan
	if err := database.DB.
		Where("status IN ?", []models.MealPlanStatus{models.MealPlanConfirmed, models.MealPlanActive}).
		Preload("Days").Find(&plans).Error; err != nil {
		log.Printf("meal-plan-fulfillment: load plans failed: %v", err)
		return
	}
	for i := range plans {
		p := &plans[i]
		addr, hasAddr := defaultAddress(p.CustomerID)
		generated := 0
		for j := range p.Days {
			d := &p.Days[j]
			if d.OrderID != nil || d.Status != models.MealPlanDayConfirmed || d.Date.After(due) {
				continue
			}
			if !hasAddr {
				log.Printf("meal-plan-fulfillment: plan %s has no delivery address; skipping day %s", p.ID, d.ID)
				continue
			}
			if err := generateDayOrder(p, d, addr); err != nil {
				log.Printf("meal-plan-fulfillment: generate order for day %s failed: %v", d.ID, err)
				continue
			}
			generated++
		}
		if p.Status == models.MealPlanConfirmed && generated > 0 {
			database.DB.Model(&models.MealPlan{}).
				Where("id = ? AND status = ?", p.ID, models.MealPlanConfirmed).
				Update("status", models.MealPlanActive)
		}
	}
}

// generateDayOrder creates a pending Order + item for one confirmed day, links
// it back to the day, and stages the order-created + chef-new-order events.
func generateDayOrder(p *models.MealPlan, d *models.MealPlanDay, addr models.Address) error {
	// FK constraints are disabled (DisableForeignKeyConstraintWhenMigrating), so
	// linking the weekly-menu item id is safe when there's no backing MenuItem.
	menuItemID := uuid.Nil
	if d.WeeklyMenuItemID != nil {
		menuItemID = *d.WeeklyMenuItemID
		var wmi models.WeeklyMenuItem
		if err := database.DB.First(&wmi, "id = ?", *d.WeeklyMenuItemID).Error; err == nil && wmi.MenuItemID != nil {
			menuItemID = *wmi.MenuItemID
		}
	}
	// The advance was captured at booking (escrow), so generated orders are paid.
	paymentStatus := models.PaymentPending
	if MealPlanEscrowActive() && p.EscrowPaymentID != "" {
		paymentStatus = models.PaymentCompleted
	}
	scheduled := d.Date

	// Per-day breakdown mirrors the advance: food + GST + delivery (chef is paid
	// food only; platform keeps GST + delivery). Informational on the order record
	// — escrow money conservation is handled by the plan total + per-day refunds.
	dayTax := 0.0
	dayDelivery := 0.0
	dayTotal := d.Price
	if MealPlanEscrowActive() {
		policy := GetPlatformPolicy()
		dayTax = Round2(d.Price * (policy.TaxPercent / 100.0))
		dayDelivery = Round2(policy.BaseDeliveryFee)
		dayTotal = Round2(d.Price + dayTax + dayDelivery)
	}

	return database.DB.Transaction(func(tx *gorm.DB) error {
		order := models.Order{
			OrderNumber:               mealPlanOrderNumber(),
			CustomerID:                p.CustomerID,
			ChefID:                    p.ChefID,
			Status:                    models.OrderStatusPending,
			PaymentStatus:             paymentStatus,
			Currency:                  p.Currency,
			Subtotal:                  d.Price,
			Tax:                       dayTax,
			DeliveryFee:               dayDelivery,
			Total:                     dayTotal,
			DeliveryAddressLine1:      addr.Line1,
			DeliveryAddressLine2:      addr.Line2,
			DeliveryAddressCity:       addr.City,
			DeliveryAddressState:      addr.State,
			DeliveryAddressPostalCode: addr.PostalCode,
			DeliveryAddressCountry:    addr.Country,
			DeliveryLatitude:          addr.Latitude,
			DeliveryLongitude:         addr.Longitude,
			EstimatedPrepTime:         30,
			ScheduledFor:              &scheduled,
		}
		if err := tx.Create(&order).Error; err != nil {
			return err
		}
		item := models.OrderItem{
			OrderID:    order.ID,
			MenuItemID: menuItemID,
			Name:       d.DishName,
			Price:      d.Price,
			Quantity:   1,
			Subtotal:   d.Price,
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.MealPlanDay{}).Where("id = ?", d.ID).
			Update("order_id", order.ID).Error; err != nil {
			return err
		}
		ev := OrderEvent{
			OrderID: order.ID, OrderNumber: order.OrderNumber,
			CustomerID: order.CustomerID, ChefID: order.ChefID,
			Status: string(order.Status), Total: order.Total,
		}
		if err := EnqueueOrderEvent(tx, SubjectOrderCreated, ev); err != nil {
			return err
		}
		return EnqueueOrderEvent(tx, SubjectChefNewOrder, ev)
	})
}

// completeFinishedPlans marks an active plan completed once every day is terminal.
func completeFinishedPlans() {
	var plans []models.MealPlan
	if err := database.DB.Where("status = ?", models.MealPlanActive).
		Preload("Days").Find(&plans).Error; err != nil {
		return
	}
	for i := range plans {
		p := &plans[i]
		if allDaysTerminal(p) {
			database.DB.Model(&models.MealPlan{}).
				Where("id = ? AND status = ?", p.ID, models.MealPlanActive).
				Update("status", models.MealPlanCompleted)
		}
	}
}

// MarkMealPlanDayDelivered is the delivery-pipeline hook: when an order linked to
// a meal-plan day is delivered, mark the day delivered, park its payout in a
// customer-confirmation hold (#387; no release), and emit the day-delivered
// event. Safe + idempotent on any order (no-op if the order isn't a meal-plan
// order or the day is already delivered).
func MarkMealPlanDayDelivered(orderID uuid.UUID) {
	var day models.MealPlanDay
	if err := database.DB.Where("order_id = ?", orderID).First(&day).Error; err != nil {
		return // not a meal-plan order
	}
	if day.Status == models.MealPlanDayDelivered {
		return
	}
	now := time.Now()
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&models.MealPlanDay{}).
			Where("id = ? AND status <> ?", day.ID, models.MealPlanDayDelivered).
			Updates(map[string]any{"status": models.MealPlanDayDelivered, "delivered_at": now})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		if err := SetMealPlanDayHoldAwaitingConfirmation(tx, day.ID); err != nil {
			return err
		}
		return EnqueueEvent(tx, SubjectMealPlanDayDelivered, "meal_plan_day.delivered", day.MealPlanID, map[string]any{
			"meal_plan_id": day.MealPlanID.String(), "day_id": day.ID.String(),
		})
	})
	if err != nil {
		log.Printf("meal-plan: mark day delivered for order %s failed: %v", orderID, err)
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

func defaultAddress(userID uuid.UUID) (models.Address, bool) {
	var a models.Address
	if err := database.DB.Where("user_id = ?", userID).
		Order("is_default DESC, created_at ASC").First(&a).Error; err != nil {
		return models.Address{}, false
	}
	return a, true
}

func mealPlanOrderNumber() string {
	return "HCMP" + uuid.NewString()[:10]
}

// allDaysTerminal reports whether every day has reached a terminal state.
func allDaysTerminal(p *models.MealPlan) bool {
	if len(p.Days) == 0 {
		return false
	}
	for i := range p.Days {
		switch p.Days[i].Status {
		case models.MealPlanDayDelivered, models.MealPlanDaySkipped,
			models.MealPlanDayCancelled, models.MealPlanDayRefunded, models.MealPlanDayDeclined:
			// terminal
		default:
			return false
		}
	}
	return true
}
