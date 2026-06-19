package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// order.go — the durable Order lifecycle Saga (epic #116, sub-issue #122).
//
// It orchestrates the post-payment lifecycle of a paid order:
//
//	notify chef → await chef accept (signal / timeout) → await ready (signal)
//	→ dispatch 3PL → await delivered (signal) → settle
//
// with a single compensation — refund — taken on chef rejection, accept timeout,
// or a cancellation signal at any stage. Today this orchestration is spread
// across HTTP handlers + fire-and-forget goroutines, so a crash mid-flow loses
// work (no dispatch, no settle) with no retry. As a workflow it's durable and
// resumes exactly where it left off after a worker crash/deploy.
//
// Every activity wraps an EXISTING idempotent service op (dispatch is a no-op if
// a delivery exists; refund is guarded on the order's refund id), so the saga is
// safe to retry and safe to run alongside the legacy handlers during migration.
// The workflow ID is keyed on the order (homechef:order:<id>) so it can never
// run twice for one order.

// Signal names the saga listens for. The HTTP handlers (chef accept/reject,
// mark-ready, delivered, cancel) forward these when ORDER_SAGA_ENABLED is on.
const (
	SignalChefDecision   = "order.chef_decision"
	SignalOrderReady     = "order.ready"
	SignalOrderDelivered = "order.delivered"
	SignalOrderCancelled = "order.cancelled"
)

// chefAcceptTimeout is how long the saga waits for the chef to accept before it
// auto-refunds the customer.
const chefAcceptTimeout = 30 * time.Minute

// OrderSagaInput starts the saga for one paid order.
type OrderSagaInput struct {
	OrderID uuid.UUID `json:"orderId"`
}

// ChefDecisionSignal carries the chef's accept/reject.
type ChefDecisionSignal struct {
	Accepted bool   `json:"accepted"`
	Reason   string `json:"reason,omitempty"`
}

// OrderCancelSignal carries a cancellation reason.
type OrderCancelSignal struct {
	Reason string `json:"reason,omitempty"`
}

// orderRefundInput is the compensation activity's argument.
type orderRefundInput struct {
	OrderID uuid.UUID
	Reason  string
}

// Pluggable activity transports — the worker wires these to services.* at
// startup; tests replace them. Mirrors the DispatchFunc pattern in delivery.go.
// DispatchFunc (the 3PL booking) is reused from delivery.go.
var (
	// NotifyChefFunc publishes the new-order notification to the chef.
	NotifyChefFunc = func(_ context.Context, _ uuid.UUID) error { return nil }
	// OrderSettleFunc finalizes payouts on delivery (release held Route
	// transfers). Implemented in the payments slice (#123); a safe no-op until.
	OrderSettleFunc = func(_ context.Context, _ uuid.UUID) error { return nil }
	// OrderRefundFunc is the compensation — refund the customer. Must be
	// idempotent (guarded on the order's refund id).
	OrderRefundFunc = func(_ context.Context, _ uuid.UUID, _ string) error { return nil }
)

// NotifyChefActivity tells the chef a paid order is waiting.
func NotifyChefActivity(ctx context.Context, orderID uuid.UUID) error {
	activity.GetLogger(ctx).Info("notify chef of new order", "order", orderID.String())
	return NotifyChefFunc(ctx, orderID)
}

// OrderSettleActivity releases payouts for a delivered order.
func OrderSettleActivity(ctx context.Context, orderID uuid.UUID) error {
	activity.GetLogger(ctx).Info("settle order", "order", orderID.String())
	return OrderSettleFunc(ctx, orderID)
}

// OrderRefundActivity compensates a failed/cancelled order by refunding it.
func OrderRefundActivity(ctx context.Context, in orderRefundInput) error {
	activity.GetLogger(ctx).Info("refund order (compensation)", "order", in.OrderID.String(), "reason", in.Reason)
	return OrderRefundFunc(ctx, in.OrderID, in.Reason)
}

// OrderSagaWorkflow runs the durable post-payment order lifecycle.
func OrderSagaWorkflow(ctx workflow.Context, in OrderSagaInput) error {
	actx := apitemporal.Activities(ctx, 2*time.Minute)
	orderID := in.OrderID

	chefCh := workflow.GetSignalChannel(ctx, SignalChefDecision)
	readyCh := workflow.GetSignalChannel(ctx, SignalOrderReady)
	deliveredCh := workflow.GetSignalChannel(ctx, SignalOrderDelivered)
	cancelCh := workflow.GetSignalChannel(ctx, SignalOrderCancelled)

	compensate := func(reason string) error {
		return workflow.ExecuteActivity(actx, OrderRefundActivity, orderRefundInput{OrderID: orderID, Reason: reason}).Get(ctx, nil)
	}

	// 1. Notify the chef.
	if err := workflow.ExecuteActivity(actx, NotifyChefActivity, orderID).Get(ctx, nil); err != nil {
		return err
	}

	// 2. Await the chef's accept/reject — or a cancel, or the accept timeout.
	var dec ChefDecisionSignal
	var cancel OrderCancelSignal
	cancelled, rejected, timedOut := false, false, false
	sel := workflow.NewSelector(ctx)
	sel.AddReceive(chefCh, func(ch workflow.ReceiveChannel, _ bool) { ch.Receive(ctx, &dec); rejected = !dec.Accepted })
	sel.AddReceive(cancelCh, func(ch workflow.ReceiveChannel, _ bool) { ch.Receive(ctx, &cancel); cancelled = true })
	sel.AddFuture(workflow.NewTimer(ctx, chefAcceptTimeout), func(workflow.Future) { timedOut = true })
	sel.Select(ctx)
	switch {
	case cancelled:
		return compensate(cancelReason(cancel, "cancelled before chef accepted"))
	case timedOut:
		return compensate("chef did not accept within the allowed time")
	case rejected:
		return compensate(decisionReason(dec, "chef rejected the order"))
	}

	// 3. Await the chef marking the order ready — or a cancel.
	if done, err := awaitOrCancel(ctx, readyCh, cancelCh, compensate); done {
		return err
	}

	// 4. Dispatch to a 3PL provider (idempotent — no-op if already dispatched).
	if err := workflow.ExecuteActivity(actx, DispatchDeliveryActivity, orderID).Get(ctx, nil); err != nil {
		return err
	}

	// 5. Await delivery — or a cancel.
	if done, err := awaitOrCancel(ctx, deliveredCh, cancelCh, compensate); done {
		return err
	}

	// 6. Settle payouts. This is terminal: once delivered, a late cancel signal is
	// intentionally NOT auto-compensated here — a post-delivery dispute is handled
	// by the manual refund path (handlers.InitiateRefund), not the saga.
	return workflow.ExecuteActivity(actx, OrderSettleActivity, orderID).Get(ctx, nil)
}

// awaitOrCancel blocks until `proceedCh` fires (returns false, nil) or a cancel
// arrives (returns true and the compensation result).
func awaitOrCancel(ctx workflow.Context, proceedCh, cancelCh workflow.ReceiveChannel, compensate func(string) error) (bool, error) {
	var cancel OrderCancelSignal
	cancelled := false
	sel := workflow.NewSelector(ctx)
	sel.AddReceive(proceedCh, func(ch workflow.ReceiveChannel, _ bool) { ch.Receive(ctx, nil) })
	sel.AddReceive(cancelCh, func(ch workflow.ReceiveChannel, _ bool) { ch.Receive(ctx, &cancel); cancelled = true })
	sel.Select(ctx)
	if cancelled {
		return true, compensate(cancelReason(cancel, "cancelled"))
	}
	return false, nil
}

func cancelReason(c OrderCancelSignal, fallback string) string {
	if c.Reason != "" {
		return c.Reason
	}
	return fallback
}

func decisionReason(d ChefDecisionSignal, fallback string) string {
	if d.Reason != "" {
		return d.Reason
	}
	return fallback
}
