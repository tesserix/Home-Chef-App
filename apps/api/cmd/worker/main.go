// Command worker runs the HomeChef Temporal workers. Thanks to the reusable
// temporal module, a service's entire worker entrypoint is just dependency init
// plus a list of queues with their workflows/activities — add more as the
// migration grows (orders, payments, delivery, …). See epic #116.
package main

import (
	"log"

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

	// Wire the notification activity's transport to the real senders.
	workflows.SendFunc = services.DispatchNotification

	if err := temporal.RunWorkers(
		temporal.Queue(temporal.TaskQueueNotifications).
			Workflows(workflows.NotificationWorkflow).
			Activities(workflows.SendNotificationActivity),
	); err != nil {
		log.Fatalf("temporal worker: %v", err)
	}
}
