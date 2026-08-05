// FSSAI filing-request SLA. One workflow instance per request (workflow ID
// "homechef:fssai:<request_id>"), started from the fssai.request.submitted NATS
// event. It chases whichever side is holding the request up — staff while it
// sits unworked, the chef while we are waiting on information — and completes
// when the request is issued, rejected or refunded.
//
// The chef has paid up front and cannot get a refund, so silence is the one
// outcome that must not be possible.
package workflows

import (
	"context"
	"time"

	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/workflow"
)

// FssaiRequestSignalName carries status changes into the running workflow.
const FssaiRequestSignalName = "fssai-request-event"

// Nudge kinds, addressed at whoever is holding the request up.
const (
	FssaiNoticeUnworked  = "unworked"   // paid, submitted, nobody has picked it up
	FssaiNoticeStalled   = "stalled"    // still not filed well past the promise
	FssaiNoticeChefChase = "chef_chase" // we asked the chef for something; no answer
)

// FssaiRequestInput describes the request entering the queue.
type FssaiRequestInput struct {
	RequestID   string `json:"requestId"`
	ChefID      string `json:"chefId"`
	UserID      string `json:"userId"`
	KitchenName string `json:"kitchenName"`
	Mode        string `json:"mode"`
}

// FssaiRequestSignal mirrors the fssai.request.* events. Status is the request
// status just reached; AwaitingChef says which side now holds it.
type FssaiRequestSignal struct {
	Status       string `json:"status"`
	AwaitingChef bool   `json:"awaitingChef"`
}

// FssaiNotice is what the notify activity delivers.
type FssaiNotice struct {
	Kind        string        `json:"kind"`
	RequestID   string        `json:"requestId"`
	ChefID      string        `json:"chefId"`
	UserID      string        `json:"userId"`
	KitchenName string        `json:"kitchenName"`
	Mode        string        `json:"mode"`
	Waiting     time.Duration `json:"waiting"`
}

// FssaiNotifyFunc is the pluggable transport, wired by the worker to
// services.SendFssaiSlaNotice. Safe no-op default.
var FssaiNotifyFunc = func(_ context.Context, _ FssaiNotice) error { return nil }

// FssaiNotifyActivity delivers one nudge.
func FssaiNotifyActivity(ctx context.Context, n FssaiNotice) error {
	return FssaiNotifyFunc(ctx, n)
}

// SLA thresholds. Deliberately generous: filing on FoSCoS is manual work with a
// government portal at the other end, so these mark "nobody is looking at this"
// rather than "this is late".
const (
	fssaiUnworkedAfter  = 24 * time.Hour
	fssaiStalledAfter   = 72 * time.Hour
	fssaiChefChaseEvery = 48 * time.Hour
	// Give up chasing rather than nag forever. Three unanswered asks is the
	// point at which this stops being a reminder and becomes a person's problem.
	fssaiMaxChefChases = 3
)

// FssaiRequestWorkflow chases a paid filing request until it is finished.
func FssaiRequestWorkflow(ctx workflow.Context, in FssaiRequestInput) error {
	ctx = apitemporal.Activities(ctx, 30*time.Second)
	log := workflow.GetLogger(ctx)

	notice := func(kind string, waiting time.Duration) {
		n := FssaiNotice{
			Kind: kind, RequestID: in.RequestID, ChefID: in.ChefID,
			UserID: in.UserID, KitchenName: in.KitchenName, Mode: in.Mode,
			Waiting: waiting,
		}
		// Best-effort: a failed nudge must not fail the workflow, or one flaky
		// send would stop every later stage from running.
		if err := workflow.ExecuteActivity(ctx, FssaiNotifyActivity, n).Get(ctx, nil); err != nil {
			log.Warn("fssai sla notice failed", "kind", kind, "err", err)
		}
	}

	sig := workflow.GetSignalChannel(ctx, FssaiRequestSignalName)
	awaitingChef := false
	chefChases := 0
	// Reset on every transition: each side gets the full window from the moment
	// the request landed with them, not from when it was submitted.
	since := workflow.Now(ctx)
	staffStage := 0
	staffStages := []struct {
		after time.Duration
		kind  string
	}{
		{fssaiUnworkedAfter, FssaiNoticeUnworked},
		{fssaiStalledAfter, FssaiNoticeStalled},
	}

	for {
		// Whichever side holds the request decides what we are waiting for.
		var wait time.Duration
		var kind string
		switch {
		case awaitingChef:
			if chefChases >= fssaiMaxChefChases {
				// Nothing left to chase automatically; the request stays open and
				// visible in the admin queue for a human to close.
				wait = 0
			} else {
				wait = fssaiChefChaseEvery
				kind = FssaiNoticeChefChase
			}
		case staffStage < len(staffStages):
			wait = staffStages[staffStage].after
			kind = staffStages[staffStage].kind
		default:
			wait = 0 // staff stages exhausted; only a terminal signal ends this
		}

		elapsed := workflow.Now(ctx).Sub(since)
		var timer workflow.Future
		timerCtx, cancelTimer := workflow.WithCancel(ctx)
		if wait > 0 {
			remaining := wait - elapsed
			if remaining < 0 {
				remaining = 0
			}
			timer = workflow.NewTimer(timerCtx, remaining)
		}

		sel := workflow.NewSelector(ctx)
		var s FssaiRequestSignal
		fired := false
		signalled := false
		sel.AddReceive(sig, func(ch workflow.ReceiveChannel, _ bool) {
			ch.Receive(ctx, &s)
			signalled = true
		})
		if timer != nil {
			sel.AddFuture(timer, func(workflow.Future) { fired = true })
		}
		sel.Select(ctx)
		cancelTimer()

		if signalled {
			// Terminal: the request is finished one way or another.
			switch s.Status {
			case "issued", "rejected", "refunded":
				return nil
			}
			if s.AwaitingChef != awaitingChef {
				// The ball changed hands — restart that side's clock.
				awaitingChef = s.AwaitingChef
				since = workflow.Now(ctx)
				if awaitingChef {
					chefChases = 0
				} else {
					staffStage = 0
				}
			}
			continue
		}

		if !fired {
			continue
		}
		notice(kind, workflow.Now(ctx).Sub(since))
		if awaitingChef {
			chefChases++
			since = workflow.Now(ctx)
		} else {
			staffStage++
		}
	}
}
