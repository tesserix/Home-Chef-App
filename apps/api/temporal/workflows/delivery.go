package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// DispatchFunc is the pluggable booking call; the worker wires it to
// services.DispatchOrderDelivery. Default is a safe no-op so the workflow is
// runnable before integration (epic #116, sub-issue #124).
var DispatchFunc = func(_ context.Context, _ uuid.UUID) error { return nil }

// DispatchDeliveryActivity books the order with a 3PL provider. It is retried by
// the workflow's retry policy — booking is the flaky external call that today
// silently loses work when the fire-and-forget goroutine's process dies.
func DispatchDeliveryActivity(ctx context.Context, orderID uuid.UUID) error {
	activity.GetLogger(ctx).Info("dispatch delivery", "order", orderID.String())
	return DispatchFunc(ctx, orderID)
}

// DeliveryWorkflow durably dispatches an order to a delivery provider. The
// workflow ID is keyed on the order (homechef:delivery:<orderID>), so retries
// and duplicate triggers never double-book. DispatchOrderDelivery is itself
// idempotent (skips if a delivery already exists), making retries safe.
func DeliveryWorkflow(ctx workflow.Context, orderID uuid.UUID) error {
	ctx = apitemporal.Activities(ctx, 2*time.Minute)
	return workflow.ExecuteActivity(ctx, DispatchDeliveryActivity, orderID).Get(ctx, nil)
}
