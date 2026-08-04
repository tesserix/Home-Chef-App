package workflows

import (
	"context"
	"time"

	"github.com/google/uuid"
	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// payment_resolution.go — the durable "did this payment actually happen?" flow.
//
// ── Why this exists ──────────────────────────────────────────────────────────
//
// An order row is created BEFORE payment. Whether it ever becomes paid was
// decided by three things that can all miss:
//
//  1. the client calling /payments/order/:id/verify after checkout closes —
//     lost to a killed app, a dismissed sheet, or a dropped network;
//  2. order_payment_reconcile_cron, a 5-minute sweep that only settles
//     payments already CAPTURED;
//  3. stale_order_cron, a 10-minute sweep that cancels at the 30-minute mark.
//
// The gap between (2) and (3) is where money goes wrong. A gateway payment sits
// in a non-terminal state — Cashfree PENDING while the bank's OTP page is open,
// Razorpay `authorized` with a hold already on the card — for longer than the
// stale threshold. (2) will not settle it because it is not captured. (3) used
// to cancel it, and the reconcile cron is forward-only, so a payment that then
// succeeded landed on a cancelled order that nothing would ever recover.
//
// stale_order_cron.go now refuses to cancel a live attempt, which closes the
// money hole. This workflow closes the LIVENESS one: instead of an order's fate
// depending on which sweep happens to look at it and when, each payment gets its
// own durable timer that polls the gateway on a backoff until the answer is
// terminal, then settles or expires it exactly once.
//
// ── Why Temporal rather than a tighter cron ──────────────────────────────────
//
// The poll must survive a deploy, a pod eviction, and a spot preemption — all
// routine here — without either forgetting an order or double-settling one. A
// goroutine loses its state; a cron re-derives it by scanning every pending row
// every tick. Workflow state is durable and the workflow ID is keyed on the
// order, so it can never run twice for one payment.
//
// The two crons stay exactly as they are. This workflow is a faster, per-order
// path to the same decisions, gated OFF by default; when it is off, or Temporal
// is down, the crons are unchanged and still authoritative. Nothing here is a
// second money path: every terminal edge calls the SAME service op the crons
// call, each of which is idempotent.

// SignalPaymentResolved lets the HTTP verify leg and the gateway webhook end the
// poll the moment they settle an order themselves, instead of leaving the
// workflow to discover it on its next tick. Purely an optimisation — the poll
// reaches the same answer on its own.
const SignalPaymentResolved = "payment.resolved"

// PaymentResolutionInput starts the flow for one unpaid order.
type PaymentResolutionInput struct {
	OrderID uuid.UUID `json:"orderId"`
}

// PaymentOutcome is what one poll learned, mirroring services' tri-state
// gateway probe. The names are the workflow's vocabulary; the mapping from each
// gateway's own statuses lives in services, next to the client that speaks them.
type PaymentOutcome string

const (
	// PaymentOutcomeSettled — the order is paid (this poll settled it, or
	// something else already had). Terminal.
	PaymentOutcomeSettled PaymentOutcome = "settled"
	// PaymentOutcomeInFlight — an attempt exists that could still take the
	// money. Keep waiting; never cancel.
	PaymentOutcomeInFlight PaymentOutcome = "in_flight"
	// PaymentOutcomeDead — the gateway says every attempt is terminally failed,
	// or none was ever made. Safe to expire the order.
	PaymentOutcomeDead PaymentOutcome = "dead"
	// PaymentOutcomeUnknown — the gateway could not be reached or answered
	// something we refuse to interpret. Treated exactly like in-flight: an
	// unknown answer must never cancel an order.
	PaymentOutcomeUnknown PaymentOutcome = "unknown"
	// PaymentOutcomeGone — the order is already cancelled, refunded or
	// otherwise no longer ours to resolve. Terminal, and NOT a failure.
	PaymentOutcomeGone PaymentOutcome = "gone"
)

// Polling shape. The first minute is where almost every payment resolves, so it
// is polled hard; after that the interval widens to keep the gateway call
// volume proportional to the (small) number of genuinely slow payments.
//
// paymentResolutionHorizon is deliberately LONGER than stale_order_cron's
// 30-minute threshold. The workflow must not be the thing that gives up first:
// if it is still waiting at the horizon it hands the order back to the crons
// rather than cancelling on a timer of its own.
const (
	paymentPollFast     = 10 * time.Second
	paymentPollSlow     = 60 * time.Second
	paymentFastWindow   = 2 * time.Minute
	paymentStalledAfter = 10 * time.Minute
	paymentHorizon      = 45 * time.Minute
)

// Pluggable activity transports, wired to services.* by the worker at startup
// and replaced in tests. Same pattern as DispatchFunc in delivery.go.
var (
	// ResolvePaymentFunc asks the gateway about one order and, if it finds a
	// captured payment, settles it through the shared settle core. Idempotent:
	// an already-paid order returns Settled without touching anything.
	ResolvePaymentFunc func(ctx context.Context, orderID uuid.UUID) (PaymentOutcome, error)
	// ExpireUnpaidOrderFunc cancels an order the gateway has confirmed dead,
	// releasing its capacity and slot. Idempotent on an already-cancelled order.
	ExpireUnpaidOrderFunc func(ctx context.Context, orderID uuid.UUID) error
	// PaymentStalledFunc raises the ops signal for a payment still unresolved
	// after paymentStalledAfter. Publishes, never blocks the flow.
	PaymentStalledFunc func(ctx context.Context, orderID uuid.UUID, waited time.Duration) error
)

// ResolvePaymentActivity polls the gateway and settles a captured payment.
func ResolvePaymentActivity(ctx context.Context, orderID uuid.UUID) (PaymentOutcome, error) {
	if ResolvePaymentFunc == nil {
		activity.GetLogger(ctx).Warn("payment resolution: no resolver wired")
		return PaymentOutcomeUnknown, nil
	}
	return ResolvePaymentFunc(ctx, orderID)
}

// ExpireUnpaidOrderActivity cancels an order whose payment is confirmed dead.
func ExpireUnpaidOrderActivity(ctx context.Context, orderID uuid.UUID) error {
	if ExpireUnpaidOrderFunc == nil {
		return nil
	}
	return ExpireUnpaidOrderFunc(ctx, orderID)
}

// PaymentStalledActivity raises the ops signal for a slow payment.
func PaymentStalledActivity(ctx context.Context, in PaymentStalledInput) error {
	if PaymentStalledFunc == nil {
		return nil
	}
	return PaymentStalledFunc(ctx, in.OrderID, in.Waited)
}

// PaymentStalledInput carries how long the payment has been unresolved.
type PaymentStalledInput struct {
	OrderID uuid.UUID     `json:"orderId"`
	Waited  time.Duration `json:"waited"`
}

// PaymentResolutionResult is the workflow's terminal answer, returned so it is
// visible in the Temporal UI without reading logs.
type PaymentResolutionResult struct {
	Outcome PaymentOutcome `json:"outcome"`
	Polls   int            `json:"polls"`
	Waited  time.Duration  `json:"waited"`
}

// PaymentResolutionWorkflow polls one order's gateway payment to a terminal
// answer.
//
// The loop only ever ENDS on an answer the gateway gave: settled, dead, or the
// order no longer being ours. Running out of horizon is not an answer, so it
// exits as in-flight and leaves the order pending for the crons — the whole
// point being that a timer must never be what cancels an order.
func PaymentResolutionWorkflow(ctx workflow.Context, in PaymentResolutionInput) (PaymentResolutionResult, error) {
	log := workflow.GetLogger(ctx)

	pollCtx := workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		// The gateway is a third party: retry the transport, but let the
		// workflow's own loop own the pacing rather than burying a long retry
		// inside one activity.
		RetryPolicy: apitemporal.DefaultRetryPolicy(),
	})

	resolved := workflow.GetSignalChannel(ctx, SignalPaymentResolved)

	started := workflow.Now(ctx)
	stalledRaised := false
	polls := 0

	for {
		elapsed := workflow.Now(ctx).Sub(started)
		if elapsed >= paymentHorizon {
			log.Info("payment resolution: horizon reached, handing back to the sweeps",
				"orderId", in.OrderID, "polls", polls)
			return PaymentResolutionResult{Outcome: PaymentOutcomeInFlight, Polls: polls, Waited: elapsed}, nil
		}

		var outcome PaymentOutcome
		polls++
		if err := workflow.ExecuteActivity(pollCtx, ResolvePaymentActivity, in.OrderID).Get(ctx, &outcome); err != nil {
			// Even an exhausted retry is only an UNKNOWN answer. Keep waiting.
			log.Warn("payment resolution: poll failed, treating as unknown",
				"orderId", in.OrderID, "err", err)
			outcome = PaymentOutcomeUnknown
		}

		switch outcome {
		case PaymentOutcomeSettled, PaymentOutcomeGone:
			log.Info("payment resolution: terminal", "orderId", in.OrderID,
				"outcome", outcome, "polls", polls)
			return PaymentResolutionResult{Outcome: outcome, Polls: polls, Waited: elapsed}, nil

		case PaymentOutcomeDead:
			// The gateway itself confirmed nothing can still take the money.
			// This is the ONLY branch that cancels, and it cancels on the
			// gateway's answer, never on elapsed time.
			if err := workflow.ExecuteActivity(pollCtx, ExpireUnpaidOrderActivity, in.OrderID).Get(ctx, nil); err != nil {
				// Failing to expire is not a reason to cancel by other means —
				// leave it to the stale cron, which reaches the same decision.
				log.Warn("payment resolution: expire failed, leaving it to the sweep",
					"orderId", in.OrderID, "err", err)
			}
			return PaymentResolutionResult{Outcome: PaymentOutcomeDead, Polls: polls, Waited: elapsed}, nil
		}

		// In-flight or unknown: raise the ops signal once, then wait.
		if !stalledRaised && elapsed >= paymentStalledAfter {
			stalledRaised = true
			_ = workflow.ExecuteActivity(pollCtx, PaymentStalledActivity,
				PaymentStalledInput{OrderID: in.OrderID, Waited: elapsed}).Get(ctx, nil)
		}

		wait := paymentPollSlow
		if elapsed < paymentFastWindow {
			wait = paymentPollFast
		}

		// Race the timer against the settled signal so a client verify or a
		// gateway webhook ends the poll immediately.
		sel := workflow.NewSelector(ctx)
		woken := false
		sel.AddFuture(workflow.NewTimer(ctx, wait), func(workflow.Future) {})
		sel.AddReceive(resolved, func(c workflow.ReceiveChannel, _ bool) {
			c.Receive(ctx, nil)
			woken = true
		})
		sel.Select(ctx)
		if woken {
			log.Info("payment resolution: woken by signal", "orderId", in.OrderID)
		}
	}
}
