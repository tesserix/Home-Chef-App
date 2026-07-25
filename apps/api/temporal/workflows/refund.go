package workflows

// refund.go — the durable deferred chef-cancel gateway-refund retry flow.
// Started IMMEDIATELY when ChefOrderCancelHandler.CancelOrder can't reach
// Razorpay synchronously (handlers/chef_order_cancel.go stamps a
// "pending:gateway-retry:<paise>" sentinel into orders.refund_id and returns
// 200 rather than blocking the cancel — see that file's header). This
// workflow retries the SAME idempotency-keyed gateway refund with backoff
// until it lands, so the customer's refund typically lands within
// seconds/minutes instead of waiting for the next
// services.RetryDeferredCancelRefunds cron tick (up to ~12 minutes).
//
// The cron remains the backstop: if Temporal is down at cancel time, or this
// workflow itself never completes (exhausts its 24h retry window), the cron
// keeps sweeping the sentinel until the gateway refund lands. Both paths call
// the gateway with the identical stable key
// (services.RefundFullIdempotencyKey(order.ID)), so a double-fire from either
// side dedups at Razorpay — never a double refund.

import (
	"context"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"

	apitemporal "github.com/homechef/api/temporal"
)

// DeferredRefundInput starts the flow for one deferred chef-cancel refund.
type DeferredRefundInput struct {
	OrderID     uuid.UUID
	PaymentID   string
	AmountPaise int
}

// PersistRefundIDInput is the persist activity's argument.
type PersistRefundIDInput struct {
	OrderID  uuid.UUID
	RefundID string
}

// Transport seams — the worker wires these to services.* (cmd/worker/main.go).
// Nil in unit tests unless the test overrides them.
var (
	// GatewayRefundFunc issues the deferred gateway refund and returns its id.
	GatewayRefundFunc func(ctx context.Context, orderID uuid.UUID, paymentID string, amountPaise int) (refundID string, err error)
	// PersistRefundIDFunc replaces the deferred sentinel with the real refund id.
	PersistRefundIDFunc func(ctx context.Context, orderID uuid.UUID, refundID string) error
)

// GatewayRefundActivity issues the deferred refund at the gateway. Retried by
// the workflow's backoff policy until it succeeds or the schedule-to-close
// window is exhausted.
func GatewayRefundActivity(ctx context.Context, in DeferredRefundInput) (string, error) {
	if GatewayRefundFunc == nil {
		return "", nil
	}
	return GatewayRefundFunc(ctx, in.OrderID, in.PaymentID, in.AmountPaise)
}

// PersistRefundIDActivity replaces the deferred sentinel in orders.refund_id
// with the real gateway refund id.
func PersistRefundIDActivity(ctx context.Context, in PersistRefundIDInput) error {
	if PersistRefundIDFunc == nil {
		return nil
	}
	return PersistRefundIDFunc(ctx, in.OrderID, in.RefundID)
}

// DeferredRefundWorkflow retries the deferred gateway refund with exponential
// backoff for up to 24h until it lands (the gateway call itself is quick; the
// long schedule-to-close is what makes "retry until it lands" durable), then
// persists the real refund id over the sentinel.
func DeferredRefundWorkflow(ctx workflow.Context, in DeferredRefundInput) error {
	actx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout:    30 * time.Second,
		ScheduleToCloseTimeout: 24 * time.Hour,
		RetryPolicy:            apitemporal.DefaultRetryPolicy(),
	})

	var refundID string
	if err := workflow.ExecuteActivity(actx, GatewayRefundActivity, in).Get(ctx, &refundID); err != nil {
		// The 24h retry window was exhausted — Temporal marks this workflow
		// failed. The cron backstop (RetryDeferredCancelRefunds) keeps sweeping
		// the sentinel independently, so the refund is never lost.
		return err
	}
	if refundID == "" {
		return nil // nothing to refund (e.g. the transport seam isn't wired)
	}

	return workflow.ExecuteActivity(actx, PersistRefundIDActivity, PersistRefundIDInput{
		OrderID: in.OrderID, RefundID: refundID,
	}).Get(ctx, nil)
}
