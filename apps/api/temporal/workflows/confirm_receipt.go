package workflows

// confirm_receipt.go — the durable confirm-receipt reminder + auto-confirm
// flow. Started per-order when a delivery transitions to delivered/awaiting
// confirmation. It reminds the customer up to MaxReminders times (one every
// ReminderIntervalSeconds); an order.confirmed or order.disputed signal ends
// it early. If the customer never acts, it auto-confirms on their behalf.
//
// This workflow only advances the hold to release_eligible/disputed via the
// existing ConfirmOrderHold transition (wired through AutoConfirmFunc) — it
// never moves money. The existing payout-release sweep does that on its own
// schedule, and the 24h payout-auto-confirm cron remains the fallback if this
// flow is disabled or Temporal is unavailable.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"

	apitemporal "github.com/homechef/api/temporal"
)

// Signal names the flow listens for. HTTP handlers forward these when the
// confirm-receipt flow is enabled (manual confirm / dispute-opened sites).
const (
	SignalOrderConfirmed = "order.confirmed"
	SignalOrderDisputed  = "order.disputed"
)

// ConfirmReceiptInput starts the flow for one delivered order. Interval/count
// are read once from PlatformSettings at start time and passed in so the
// running workflow stays deterministic.
type ConfirmReceiptInput struct {
	OrderID                 uuid.UUID
	ReminderIntervalSeconds int
	MaxReminders            int
}

// ReminderActivityInput is the reminder activity's argument.
type ReminderActivityInput struct {
	OrderID uuid.UUID
	Attempt int
}

// Transport seams — the worker wires these to services.* (Task 5). Nil in
// unit tests unless the test overrides them.
var (
	// ConfirmReminderFunc sends one confirm-receipt reminder to the customer.
	ConfirmReminderFunc func(ctx context.Context, orderID uuid.UUID, attempt int) error
	// AutoConfirmFunc confirms receipt on the customer's behalf.
	AutoConfirmFunc func(ctx context.Context, orderID uuid.UUID) error
)

// ReminderActivity nudges the customer to confirm they received their order.
func ReminderActivity(ctx context.Context, in ReminderActivityInput) error {
	if ConfirmReminderFunc == nil {
		return nil
	}
	return ConfirmReminderFunc(ctx, in.OrderID, in.Attempt)
}

// AutoConfirmActivity confirms receipt on the customer's behalf once the
// reminder window is exhausted with no response.
func AutoConfirmActivity(ctx context.Context, orderID uuid.UUID) error {
	if AutoConfirmFunc == nil {
		return nil
	}
	return AutoConfirmFunc(ctx, orderID)
}

// ConfirmReceiptWorkflow reminds the customer to confirm receipt up to
// MaxReminders times (one every ReminderIntervalSeconds), then auto-confirms
// if they never act. A confirmed/disputed signal ends it early.
func ConfirmReceiptWorkflow(ctx workflow.Context, in ConfirmReceiptInput) error {
	interval := time.Duration(in.ReminderIntervalSeconds) * time.Second
	confirmedCh := workflow.GetSignalChannel(ctx, SignalOrderConfirmed)
	disputedCh := workflow.GetSignalChannel(ctx, SignalOrderDisputed)

	actx := apitemporal.Activities(ctx, 30*time.Second)

	for attempt := 1; attempt <= in.MaxReminders; attempt++ {
		done := false
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(confirmedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
		sel.AddReceive(disputedCh, func(c workflow.ReceiveChannel, _ bool) { c.Receive(ctx, nil); done = true })
		sel.AddFuture(workflow.NewTimer(ctx, interval), func(workflow.Future) {})
		sel.Select(ctx)
		if done {
			return nil // customer confirmed or a dispute opened
		}
		// Timer fired → send this attempt's reminder (best-effort).
		_ = workflow.ExecuteActivity(actx, ReminderActivity, ReminderActivityInput{
			OrderID: in.OrderID, Attempt: attempt,
		}).Get(ctx, nil)
	}

	// Reminders exhausted, no confirmation → auto-confirm.
	return workflow.ExecuteActivity(actx, AutoConfirmActivity, in.OrderID).Get(ctx, nil)
}
