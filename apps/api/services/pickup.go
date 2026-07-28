package services

// pickup.go — the collection side of a pickup order.
//
// A delivery order has a carrier: once the chef marks it ready, a rider is
// dispatched and the platform keeps driving it to a conclusion. A PICKUP order
// has nobody. The food is cooked, it sits on the chef's counter, and the only
// actor left is a customer who has to remember to turn up. Before this, the
// entire platform-side handling of that moment was one generic "Your order is
// ready for pickup/delivery" push riding on orders.updated — after which
// nothing else ever happened, no matter how long the food sat there.
//
// This file supplies the three messages that moment actually needs:
//
//	NotifyOrderReadyForPickup   → customer: it's ready, here's the kitchen
//	SendPickupReminder          → customer: you still haven't collected it
//	EscalateUncollectedPickup   → chef: nobody came; it's your call now
//
// The SCHEDULE for the last two is not here — a ticker would lose its place on
// every restart. temporal/workflows/pickup.go owns the timing durably, and
// services/temporal_pickup.go starts it. Every function here is safe to call
// more than once: each re-reads the order and no-ops once it is no longer
// awaiting collection, which is what makes an at-least-once activity retry
// harmless.

import (
	"fmt"
	"strconv"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// awaitingCollection reports whether a pickup order is still sitting uncollected
// — the precondition every message in this file shares.
//
// `ready` is the only status that means "cooked, not yet handed over": pickup has
// no picked_up/delivering leg (nobody carries it), and `delivered` is what the
// chef sets when the customer walks out with it. So a pickup order that has left
// `ready` has been collected, cancelled or refunded, and there is nothing left to
// chase.
func awaitingCollection(order models.Order) bool {
	return order.FulfillmentType == models.FulfillmentPickup &&
		order.Status == models.OrderStatusReady
}

// pickupChefName resolves a human name for the kitchen, falling back to a
// generic phrase. A notification that names the place is the difference between
// "go collect your order" and knowing where to go.
func pickupChefName(db *gorm.DB, chefID uuid.UUID) string {
	var chef models.ChefProfile
	if err := db.First(&chef, "id = ?", chefID).Error; err == nil && chef.BusinessName != "" {
		return chef.BusinessName
	}
	return "the kitchen"
}

// NotifyOrderReadyForPickup tells the customer their pickup order is cooked and
// waiting, naming the kitchen. Fired when a pickup order reaches `ready` —
// exactly where a delivery order would instead dispatch a rider.
//
// Best-effort push plus a staged outbox event, matching SendConfirmReceiptReminder:
// the push is a courtesy that may fail on a stale token, the outbox row is the
// durable record that survives to be relayed.
func NotifyOrderReadyForPickup(db *gorm.DB, orderID uuid.UUID) error {
	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return err
	}
	if !awaitingCollection(order) {
		return nil
	}

	chefName := pickupChefName(db, order.ChefID)
	title := "Ready to collect"
	body := fmt.Sprintf("Your order from %s is ready. Head over when you can.", chefName)
	_ = SendPushNotification(order.CustomerID, title, body, map[string]string{
		"order_id": order.ID.String(),
		"type":     "ready_for_pickup",
	})

	return EnqueueOrderEvent(db, SubjectOrderReadyForPickup, OrderEvent{
		OrderID:     order.ID,
		OrderNumber: order.OrderNumber,
		CustomerID:  order.CustomerID,
		ChefID:      order.ChefID,
		Status:      string(order.Status),
		Total:       order.Total,
	})
}

// SendPickupReminder nudges a customer who still hasn't collected. Returns
// (sent, err); sent=false with a nil error means the order moved on and there was
// nothing to remind about — the ordinary outcome once someone collects, and the
// reason a duplicate activity retry is harmless.
func SendPickupReminder(db *gorm.DB, orderID uuid.UUID, attempt int) (bool, error) {
	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return false, err
	}
	if !awaitingCollection(order) {
		return false, nil
	}

	chefName := pickupChefName(db, order.ChefID)
	title := "Your order is waiting"
	body := fmt.Sprintf("%s still has your order ready to collect.", chefName)
	_ = SendPushNotification(order.CustomerID, title, body, map[string]string{
		"order_id": order.ID.String(),
		"type":     "pickup_reminder",
	})

	if err := EnqueueEvent(db, SubjectOrderPickupReminder, "order.pickup_reminder", order.CustomerID, map[string]any{
		"order_id":     order.ID.String(),
		"order_number": order.OrderNumber,
		"attempt":      attempt,
	}); err != nil {
		return true, err
	}
	return true, nil
}

// EscalateUncollectedPickup tells the CHEF that the reminder window elapsed with
// the food still uncollected.
//
// It deliberately does NOT auto-complete the order the way the confirm-receipt
// flow auto-confirms. Auto-confirming a delivery is a safe inference — a courier
// recorded a handover, so the food demonstrably reached someone. Nobody recorded
// anything here: silently marking the order `delivered` would assert a handover
// that never happened, release the chef's payout on a fiction, and leave a
// customer who was charged with no order and no evidence. The honest terminal
// state is "a person has to decide", so this hands it to the person holding the
// food, who can complete it if the customer did turn up or cancel it (with the
// existing full refund) if they didn't.
func EscalateUncollectedPickup(db *gorm.DB, orderID uuid.UUID) (bool, error) {
	var order models.Order
	if err := db.First(&order, "id = ?", orderID).Error; err != nil {
		return false, err
	}
	if !awaitingCollection(order) {
		return false, nil
	}

	// ChefID is a chef_profiles.id; a notification needs the users.id behind it.
	chefUser, err := chefUserID(order.ChefID)
	if err != nil {
		return false, fmt.Errorf("pickup_uncollected: %w", err)
	}

	title := "Order not collected"
	body := fmt.Sprintf("Order #%s hasn't been picked up. Complete it if it was collected, or cancel for a refund.", order.OrderNumber)
	_ = SendPushNotification(chefUser, title, body, map[string]string{
		"order_id": order.ID.String(),
		"type":     "pickup_uncollected",
	})

	if err := EnqueueEvent(db, SubjectOrderPickupUncollected, "order.pickup_uncollected", chefUser, map[string]any{
		"order_id":     order.ID.String(),
		"order_number": order.OrderNumber,
	}); err != nil {
		return true, err
	}
	return true, nil
}

// pickupReminderSetting reads one `pickup.*` PlatformSettings key, falling back
// to def when absent or unparsable. Mirrors confirmReminderSetting.
func pickupReminderSetting(db *gorm.DB, key string, def int) int {
	var settings []models.PlatformSettings
	db.Where("key LIKE ?", "pickup.%").Find(&settings)
	for _, s := range settings {
		if s.Key == key {
			if v, err := strconv.Atoi(s.Value); err == nil && v > 0 {
				return v
			}
		}
	}
	return def
}

// PickupReminderIntervalMinutes is the gap between collection reminders, from
// PlatformSettings key `pickup.reminder_interval_minutes` (default 20).
//
// Deliberately longer than the 10-minute confirm-receipt gap: that flow chases a
// tap someone can do from the sofa, this one chases a journey across town.
func PickupReminderIntervalMinutes(db *gorm.DB) int {
	return pickupReminderSetting(db, "pickup.reminder_interval_minutes", 20)
}

// PickupReminderMaxCount is how many reminders fire before the chef is told the
// food went uncollected, from `pickup.reminder_max_count` (default 3) — so the
// default window is ~1 hour after the food is ready.
func PickupReminderMaxCount(db *gorm.DB) int {
	return pickupReminderSetting(db, "pickup.reminder_max_count", 3)
}
