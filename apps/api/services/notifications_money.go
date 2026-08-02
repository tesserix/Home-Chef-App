package services

// notifications_money.go — handlers for money moving (#notify-money).
//
// These subjects were all published with no handler registered, so the events
// reached JetStream and were dropped on the floor: a customer was never told a
// payment succeeded or failed, and a chef was never told a payout cleared, was
// released, or was frozen by a dispute.
//
// Refunds are deliberately absent. Every refund path already notifies through
// orders.cancelled / orders.voided / orders.cancellation_resolved (or, for the
// chef's fee reduction, its own push), so a refund handler here would be a
// second notification for the same money. See nats.go.
//
// All of these carry a `type` that notificationTypeCategory maps to
// NotifCategoryPayment, so a user who has muted Payments in settings gets none
// of them — the gate lives in sendPushNotification, one layer down.

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

func (s *NotificationService) handlePaymentSuccess(event Event) error {
	amount, _ := event.Data["amount"].(float64)
	number, _ := event.Data["orderNumber"].(string)
	message := "Your payment went through — your chef is on it."
	if amount > 0 {
		message = fmt.Sprintf("₹%.0f paid for order %s. Your chef is on it.", amount, number)
	}
	return s.notifyMoney(event, "payment_success", "Payment confirmed", message)
}

func (s *NotificationService) handlePaymentFailed(event Event) error {
	number, _ := event.Data["orderNumber"].(string)
	message := "We couldn't take payment for your order. Nothing was charged — tap to try again."
	if number != "" {
		message = fmt.Sprintf(
			"We couldn't take payment for order %s. Nothing was charged — tap to try again.", number)
	}
	return s.notifyMoney(event, "payment_failed", "Payment didn't go through", message)
}

// The three payout-hold events carry an AGGREGATE id in Event.UserID, not a user
// id — emitHoldEvent and the release path both pass the order/group/meal-plan-day
// id. Notifying event.UserID directly would address a user that does not exist,
// so each of these resolves the owning chef first.

func (s *NotificationService) handleHoldReleaseEligible(event Event) error {
	return s.notifyChefHold(event, "payout_hold_release_eligible",
		"Earnings cleared",
		"This order's earnings have cleared the hold and are queued for your next payout.")
}

func (s *NotificationService) handleHoldReleased(event Event) error {
	return s.notifyChefHold(event, "payout_hold_released",
		"Payout released",
		"Your earnings for this order have been released and are on their way to your bank.")
}

func (s *NotificationService) handleHoldDisputed(event Event) error {
	return s.notifyChefHold(event, "payout_hold_disputed",
		"Payout on hold",
		"An open issue on this order has paused its payout. We'll tell you as soon as it's resolved.")
}

func (s *NotificationService) handleEarningsThresholdMet(event Event) error {
	return s.notifyMoney(event, "earnings_threshold_met",
		"Earnings milestone reached",
		"You've crossed your earnings threshold for this cycle — see Earnings for the breakdown.")
}

// notifyChefHold resolves the chef behind a payout-hold aggregate and notifies
// their user. A missing aggregate is not an error worth redelivering: the event
// is about a row that no longer resolves, and retrying cannot change that.
func (s *NotificationService) notifyChefHold(event Event, notifType, title, message string) error {
	aggType, _ := event.Data["aggregate_type"].(string)
	chefUser, ok := chefUserForAggregate(aggType, event.UserID)
	if !ok {
		return nil
	}
	scoped := event
	scoped.UserID = chefUser
	return s.notifyMoney(scoped, notifType, title, message)
}

// chefUserForAggregate maps a payout-hold aggregate to the chef's user id.
func chefUserForAggregate(aggType string, id uuid.UUID) (uuid.UUID, bool) {
	if id == uuid.Nil {
		return uuid.Nil, false
	}
	var chefID uuid.UUID
	switch aggType {
	case aggTypeOrder, "":
		var order models.Order
		if err := database.DB.Select("chef_id").First(&order, "id = ?", id).Error; err != nil {
			log.Printf("notify-money: order %s not found for hold event: %v", id, err)
			return uuid.Nil, false
		}
		chefID = order.ChefID
	case aggTypeGroupOrder:
		var group models.GroupOrder
		if err := database.DB.Select("chef_id").First(&group, "id = ?", id).Error; err != nil {
			log.Printf("notify-money: group order %s not found for hold event: %v", id, err)
			return uuid.Nil, false
		}
		chefID = group.ChefID
	case aggTypeMealPlanDay:
		var day models.MealPlanDay
		if err := database.DB.Select("meal_plan_id").First(&day, "id = ?", id).Error; err != nil {
			log.Printf("notify-money: meal-plan day %s not found for hold event: %v", id, err)
			return uuid.Nil, false
		}
		var plan models.MealPlan
		if err := database.DB.Select("chef_id").First(&plan, "id = ?", day.MealPlanID).Error; err != nil {
			log.Printf("notify-money: meal plan %s not found for hold event: %v", day.MealPlanID, err)
			return uuid.Nil, false
		}
		chefID = plan.ChefID
	default:
		log.Printf("notify-money: unknown payout-hold aggregate type %q", aggType)
		return uuid.Nil, false
	}

	var chef models.ChefProfile
	if err := database.DB.Select("user_id").First(&chef, "id = ?", chefID).Error; err != nil {
		log.Printf("notify-money: chef %s not found for hold event: %v", chefID, err)
		return uuid.Nil, false
	}
	return chef.UserID, true
}

// notifyMoney is the shared tail: persist the in-app row, then publish the push.
// The push carries `type` so the Payments preference gate can read it.
func (s *NotificationService) notifyMoney(event Event, notifType, title, message string) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	payload := make(map[string]any, len(event.Data)+1)
	for k, v := range event.Data {
		payload[k] = v
	}
	payload["type"] = notifType

	raw, _ := json.Marshal(payload)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    notifType,
		Title:   title,
		Message: message,
		Data:    string(raw),
	}); err != nil {
		return fmt.Errorf("save %s notification: %w", notifType, err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: payload,
	})
	return nil
}
