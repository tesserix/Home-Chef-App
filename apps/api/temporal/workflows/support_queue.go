// Support-chat staff queue SLA. One workflow instance per otto conversation
// (workflow ID "homechef:support-queue:<conversation_id>"), started from the
// otto.support.* NATS events when a chat enters the staff queue. It notifies
// the support mailbox immediately, nudges again while the chat sits
// unaccepted, and completes when staff accept or the thread closes.
package workflows

import (
	"context"
	"time"

	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/workflow"
)

// SupportQueueSignalName carries queue transitions into the running workflow.
const SupportQueueSignalName = "support-queue-event"

// Queue-notice kinds, in escalation order.
const (
	SupportNoticeWaiting    = "waiting"    // new chat waiting for a human
	SupportNoticeEscalated  = "escalated"  // AI handed the chat off
	SupportNoticeReminder   = "reminder"   // still unaccepted after the first SLA
	SupportNoticeUnattended = "unattended" // breached the second SLA
)

// SupportQueueInput describes the conversation entering the queue.
type SupportQueueInput struct {
	ConversationID string `json:"conversationId"`
	TenantID       string `json:"tenantId"`
	CaseID         string `json:"caseId,omitempty"`
	Subject        string `json:"subject,omitempty"`
	IntakeReason   string `json:"intakeReason,omitempty"`
	CustomerName   string `json:"customerName,omitempty"`
	Escalated      bool   `json:"escalated"`
}

// SupportQueueSignal is the payload delivered on SupportQueueSignalName.
// Event mirrors the otto queue events: "escalated" | "accepted" | "closed".
type SupportQueueSignal struct {
	Event        string `json:"event"`
	AssigneeName string `json:"assigneeName,omitempty"`
}

// SupportQueueNotice is what the notify activity delivers to staff.
type SupportQueueNotice struct {
	Kind           string        `json:"kind"`
	ConversationID string        `json:"conversationId"`
	TenantID       string        `json:"tenantId"`
	CaseID         string        `json:"caseId,omitempty"`
	Subject        string        `json:"subject,omitempty"`
	IntakeReason   string        `json:"intakeReason,omitempty"`
	CustomerName   string        `json:"customerName,omitempty"`
	Waiting        time.Duration `json:"waiting"`
}

// SupportQueueNotifyFunc is the pluggable transport, wired by the worker to
// services.SendSupportQueueNotice. Safe no-op default (issue #126 pattern).
var SupportQueueNotifyFunc = func(_ context.Context, _ SupportQueueNotice) error { return nil }

// SupportQueueNotifyActivity delivers one staff notice.
func SupportQueueNotifyActivity(ctx context.Context, n SupportQueueNotice) error {
	return SupportQueueNotifyFunc(ctx, n)
}

// SLA thresholds. The first nudge lands fast — "notified within a few
// seconds" is the immediate notice; these cover nobody picking it up.
const (
	supportQueueFirstReminder = 3 * time.Minute
	supportQueueUnattended    = 15 * time.Minute
	supportQueueGiveUp        = 24 * time.Hour
)

// SupportQueueWorkflow notifies staff about a waiting chat and escalates
// until the thread is accepted or closed.
func SupportQueueWorkflow(ctx workflow.Context, in SupportQueueInput) error {
	ctx = apitemporal.Activities(ctx, 30*time.Second)

	notice := func(kind string, waiting time.Duration) {
		n := SupportQueueNotice{
			Kind:           kind,
			ConversationID: in.ConversationID,
			TenantID:       in.TenantID,
			CaseID:         in.CaseID,
			Subject:        in.Subject,
			IntakeReason:   in.IntakeReason,
			CustomerName:   in.CustomerName,
			Waiting:        waiting,
		}
		// Best-effort by design: a failed notice must not fail the queue
		// workflow (the admin inbox WebSocket remains the primary surface).
		if err := workflow.ExecuteActivity(ctx, SupportQueueNotifyActivity, n).Get(ctx, nil); err != nil {
			workflow.GetLogger(ctx).Warn("support queue notice failed", "kind", kind, "err", err)
		}
	}

	first := SupportNoticeWaiting
	if in.Escalated {
		first = SupportNoticeEscalated
	}
	notice(first, 0)

	sig := workflow.GetSignalChannel(ctx, SupportQueueSignalName)
	start := workflow.Now(ctx)
	reminders := []struct {
		after time.Duration
		kind  string
	}{
		{supportQueueFirstReminder, SupportNoticeReminder},
		{supportQueueUnattended, SupportNoticeUnattended},
	}
	next := 0

	for {
		elapsed := workflow.Now(ctx).Sub(start)
		var timer workflow.Future
		var timerCtx workflow.Context
		var cancelTimer workflow.CancelFunc
		if next < len(reminders) {
			timerCtx, cancelTimer = workflow.WithCancel(ctx)
			timer = workflow.NewTimer(timerCtx, reminders[next].after-elapsed)
		} else {
			timerCtx, cancelTimer = workflow.WithCancel(ctx)
			timer = workflow.NewTimer(timerCtx, supportQueueGiveUp-elapsed)
		}

		done := false
		sel := workflow.NewSelector(ctx)
		sel.AddReceive(sig, func(ch workflow.ReceiveChannel, _ bool) {
			var s SupportQueueSignal
			ch.Receive(ctx, &s)
			switch s.Event {
			case "accepted", "closed":
				done = true
			case "escalated":
				// AI handed off mid-thread: surface it, keep the SLA clock.
				notice(SupportNoticeEscalated, workflow.Now(ctx).Sub(start))
			}
		})
		sel.AddFuture(timer, func(workflow.Future) {
			if next < len(reminders) {
				notice(reminders[next].kind, workflow.Now(ctx).Sub(start))
				next++
			} else {
				done = true // 24h without accept/close — stop nagging
			}
		})
		sel.Select(ctx)
		cancelTimer()
		if done {
			return nil
		}
	}
}
