package services

import (
	"context"
	"log"

	"github.com/google/uuid"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

// EnqueueDeliveryDispatch dispatches an order to a 3PL provider durably via a
// Temporal workflow (retrying the flaky booking and surviving crashes). When
// Temporal is disabled it falls back to the legacy fire-and-forget goroutine,
// so behaviour is unchanged until the cluster (#119) exists. The workflow ID is
// keyed on the order, so a duplicate trigger never double-books. It is detached
// from the caller's request context (the dispatch outlives the HTTP request).
func EnqueueDeliveryDispatch(orderID uuid.UUID) {
	if temporalRT != nil {
		id := "homechef:delivery:" + orderID.String()
		if _, err := temporalRT.Start(context.Background(), apitemporal.TaskQueueDelivery, id, workflows.DeliveryWorkflow, orderID); err != nil {
			log.Printf("delivery dispatch: Temporal start failed (%v) — dispatching inline", err)
			go func() { _ = DispatchOrderDelivery(orderID) }()
		}
		return
	}
	go func() { _ = DispatchOrderDelivery(orderID) }()
}
