// Command worker runs the HomeChef Temporal workers. Thanks to the reusable
// temporal module, a service's entire worker entrypoint is just dependency init
// plus a list of queues with their workflows/activities — add more as the
// migration grows (orders, payments, delivery, …). See epic #116.
package main

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/services"
	"github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func main() {
	config.Load()

	// Workers are full app processes (minus the HTTP server): they need the same
	// dependencies the activities touch.
	if err := database.Connect(); err != nil {
		log.Fatalf("worker: database connect: %v", err)
	}
	services.InitEmailService()
	if err := services.InitPushService(); err != nil {
		log.Printf("worker: push service init failed (push activities will error): %v", err)
	}

	// Wire activity transports to the real services.* implementations.
	workflows.SendFunc = services.DispatchNotification
	workflows.DispatchFunc = func(_ context.Context, orderID uuid.UUID) error {
		return services.DispatchOrderDelivery(orderID)
	}
	// Order lifecycle saga activities (#122).
	workflows.NotifyChefFunc = services.NotifyChefNewOrder
	workflows.OrderSettleFunc = services.SettleOrderPayouts
	workflows.OrderRefundFunc = services.CompensateOrderRefund
	// Onboarding activation (#126).
	workflows.ActivateChefFunc = services.ActivateChefOnboardingFromActivity
	// Confirm-receipt reminder + auto-confirm flow (#auto-confirm-delivery).
	workflows.ConfirmReminderFunc = func(_ context.Context, orderID uuid.UUID, attempt int) error {
		_, err := services.SendConfirmReceiptReminder(database.DB, orderID, attempt)
		return err
	}
	workflows.AutoConfirmFunc = func(_ context.Context, orderID uuid.UUID) error {
		_, _, err := services.AutoConfirmOrderReceipt(database.DB, orderID)
		return err
	}

	if err := temporal.RunWorkers(
		temporal.Queue(temporal.TaskQueueNotifications).
			Workflows(workflows.NotificationWorkflow).
			Activities(workflows.SendNotificationActivity),
		temporal.Queue(temporal.TaskQueueDelivery).
			Workflows(workflows.DeliveryWorkflow).
			Activities(workflows.DispatchDeliveryActivity),
		// Order lifecycle saga (#122) — notify → accept → ready → dispatch →
		// delivered → settle, with refund compensation.
		temporal.Queue(temporal.TaskQueueOrders).
			Workflows(workflows.OrderSagaWorkflow, workflows.ConfirmReceiptWorkflow).
			Activities(workflows.NotifyChefActivity, workflows.DispatchDeliveryActivity,
				workflows.OrderSettleActivity, workflows.OrderRefundActivity,
				workflows.ReminderActivity, workflows.AutoConfirmActivity),
		// Durable chef-onboarding activation (#126).
		temporal.Queue(temporal.TaskQueueOnboarding).
			Workflows(workflows.OnboardingActivationWorkflow).
			Activities(workflows.ActivateChefOnboardingActivity),
		// Scheduled jobs (statements, reconciliation, FSSAI, availability, audit).
		services.RegisterCronWorker(),
	); err != nil {
		log.Fatalf("temporal worker: %v", err)
	}
}
