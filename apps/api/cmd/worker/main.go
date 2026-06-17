// Command worker runs the HomeChef Temporal workers. Thanks to the reusable
// temporal module, a service's entire worker entrypoint is just a list of
// queues with their workflows/activities — add more as the migration grows
// (orders, payments, delivery, …). See epic #116.
package main

import (
	"log"

	"github.com/homechef/api/config"
	"github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func main() {
	config.Load()

	if err := temporal.RunWorkers(
		temporal.Queue(temporal.TaskQueueNotifications).
			Workflows(workflows.NotificationWorkflow).
			Activities(workflows.SendNotificationActivity),
	); err != nil {
		log.Fatalf("temporal worker: %v", err)
	}
}
