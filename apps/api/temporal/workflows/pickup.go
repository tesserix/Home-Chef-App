package workflows

// pickup.go — the durable "your food is ready to collect" flow.
//
// Started per-order the moment a PICKUP order is marked ready. It sends the
// ready notice, then reminds the customer up to MaxReminders times (one every
// ReminderIntervalSeconds), and if nobody ever collects it tells the chef the
// food is still sitting there.
//
// Why this is durable rather than a goroutine or a cron:
//
//   - A goroutine dies with the pod. The gap between "ready" and "collected" is
//     open-ended, so an in-process timer is exactly the thing least likely to
//     survive to its own deadline.
//   - A cron sweep would have to re-scan every ready pickup order on every tick
//     and keep its own "have I already reminded this one, how many times" state
//     in the database. The workflow IS that state, and it is crash-proof.
//
// This flow only ever SENDS MESSAGES. It moves no money and completes no order
// — see EscalateUncollectedPickup for why auto-completing an uncollected pickup
// would be a lie rather than a convenience.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"

	apitemporal "github.com/homechef/api/temporal"
)

// Signal names the flow listens for. Both end the reminder loop early — there is
// nothing left to chase once the food is in the customer's hands or the order is
// gone.
const (
	// SignalOrderCollected — the chef marked a pickup order delivered, i.e. the
	// customer turned up and took it.
	SignalOrderCollected = "order.collected"
	// SignalPickupCancelled — the order was cancelled or refunded while waiting.
	SignalPickupCancelled = "order.pickup_cancelled"
)

// PickupReadyInput starts the flow for one ready-to-collect order. Interval and
// count are read once from PlatformSettings at start time and passed in, so a
// running workflow stays deterministic even if an admin retunes the settings
// mid-flight.
type PickupReadyInput struct {
	OrderID                 uuid.UUID
	ReminderIntervalSeconds int
	MaxReminders            int
}

// PickupReminderActivityInput is the reminder activity's argument.
type PickupReminderActivityInput struct {
	OrderID uuid.UUID
	Attempt int
}

// Transport seams — cmd/worker wires these to services.*. Nil in unit tests
// unless the test overrides them, so the workflow logic is testable without a
// database.
var (
	// PickupReadyNoticeFunc tells the customer the order is ready to collect.
	PickupReadyNoticeFunc func(ctx context.Context, orderID uuid.UUID) error
	// PickupReminderFunc nudges a customer who still hasn't collected.
	PickupReminderFunc func(ctx context.Context, orderID uuid.UUID, attempt int) error
	// PickupUncollectedFunc tells the chef the food was never collected.
	PickupUncollectedFunc func(ctx context.Context, orderID uuid.UUID) error
)

// PickupReadyNoticeActivity announces that the order is ready for collection.
func PickupReadyNoticeActivity(ctx context.Context, orderID uuid.UUID) error {
	if PickupReadyNoticeFunc == nil {
		return nil
	}
	return PickupReadyNoticeFunc(ctx, orderID)
}

// PickupReminderActivity nudges the customer to come and collect.
func PickupReminderActivity(ctx context.Context, in PickupReminderActivityInput) error {
	if PickupReminderFunc == nil {
		return nil
	}
	return PickupReminderFunc(ctx, in.OrderID, in.Attempt)
}

// PickupUncollectedActivity escalates an uncollected order to the chef.
func PickupUncollectedActivity(ctx context.Context, orderID uuid.UUID) error {
	if PickupUncollectedFunc == nil {
		return nil
	}
	return PickupUncollectedFunc(ctx, orderID)
}

// PickupReadyWorkflow announces a ready pickup order, reminds the customer up to
// MaxReminders times, then escalates to the chef if it was never collected. A
// collected/cancelled signal ends it early at any point.
func PickupReadyWorkflow(ctx workflow.Context, in PickupReadyInput) error {
	interval := time.Duration(in.ReminderIntervalSeconds) * time.Second
	collectedCh := workflow.GetSignalChannel(ctx, SignalOrderCollected)
	cancelledCh := workflow.GetSignalChannel(ctx, SignalPickupCancelled)

	actx := apitemporal.Activities(ctx, 30*time.Second)

	// The ready notice goes out immediately — this is the whole point of the
	// flow, and it must not wait out the first reminder interval.
	_ = workflow.ExecuteActivity(actx, PickupReadyNoticeActivity, in.OrderID).Get(ctx, nil)

	for attempt := 1; attempt <= in.MaxReminders; attempt++ {
		done := false
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(collectedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
		sel.AddReceive(cancelledCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
		sel.AddFuture(workflow.NewTimer(ctx, interval), func(workflow.Future) {})
		sel.Select(ctx)
		if done {
			return nil // collected, or the order is gone
		}
		// Timer fired → send this attempt's reminder (best-effort: a failed push
		// must not abandon the remaining reminders or the escalation).
		_ = workflow.ExecuteActivity(actx, PickupReminderActivity, PickupReminderActivityInput{
			OrderID: in.OrderID, Attempt: attempt,
		}).Get(ctx, nil)
	}

	// Wait out one final interval before escalating, so the chef isn't told the
	// order is abandoned the same instant the last reminder went out — the
	// customer deserves the chance to act on it.
	done := false
	sel := workflow.NewSelector(ctx)
	sel.AddReceive(collectedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
	sel.AddReceive(cancelledCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
	sel.AddFuture(workflow.NewTimer(ctx, interval), func(workflow.Future) {})
	sel.Select(ctx)
	if done {
		return nil
	}

	return workflow.ExecuteActivity(actx, PickupUncollectedActivity, in.OrderID).Get(ctx, nil)
}
