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
	CustomerEmail  string `json:"customerEmail,omitempty"`
	Escalated      bool   `json:"escalated"`
}

// SupportQueueTicketInput asks the activity to materialise a ticket for a
// chat nobody answered in time.
type SupportQueueTicketInput struct {
	ConversationID string `json:"conversationId"`
	TenantID       string `json:"tenantId"`
	CaseID         string `json:"caseId,omitempty"`
	Subject        string `json:"subject,omitempty"`
	IntakeReason   string `json:"intakeReason,omitempty"`
	CustomerName   string `json:"customerName,omitempty"`
	CustomerEmail  string `json:"customerEmail,omitempty"`
	WaitedSeconds  int    `json:"waitedSeconds"`
}

// SupportQueueTicketResult reports what the activity did.
type SupportQueueTicketResult struct {
	TicketNumber string `json:"ticketNumber"`
	Created      bool   `json:"created"`
}

// SupportQueueTicketFunc is the pluggable ticket writer, wired by the worker
// to services.RaiseSupportQueueTicket.
var SupportQueueTicketFunc = func(_ context.Context, _ SupportQueueTicketInput) (SupportQueueTicketResult, error) {
	return SupportQueueTicketResult{}, nil
}

// SupportQueueTicketActivity creates the ticket and notifies the admin.
func SupportQueueTicketActivity(ctx context.Context, in SupportQueueTicketInput) (SupportQueueTicketResult, error) {
	return SupportQueueTicketFunc(ctx, in)
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

// SLA thresholds. The immediate notice goes out on entry; these cover nobody
// picking the chat up. At supportQueueTicketTimeout the customer has waited
// long enough that live chat has failed them, so the thread is converted into
// a durable ticket and the admin is told — nobody is left waiting silently.
const (
	supportQueueFirstReminder  = 3 * time.Minute
	supportQueueTicketTimeout  = 10 * time.Minute
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
	// Staged waits: nudge staff first, then give up on live chat and raise a
	// ticket so the customer gets a tracked answer instead of silence.
	stages := []struct {
		after time.Duration
		kind  string
	}{
		{supportQueueFirstReminder, SupportNoticeReminder},
		{supportQueueTicketTimeout, SupportNoticeUnattended},
	}
	next := 0

	for next < len(stages) {
		elapsed := workflow.Now(ctx).Sub(start)
		timerCtx, cancelTimer := workflow.WithCancel(ctx)
		timer := workflow.NewTimer(timerCtx, stages[next].after-elapsed)

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
			waited := workflow.Now(ctx).Sub(start)
			notice(stages[next].kind, waited)
			if stages[next].kind == SupportNoticeUnattended {
				// Nobody picked it up inside the window: convert to a ticket
				// so it is tracked and the customer hears back.
				raiseTicket(ctx, in, waited)
				done = true
			}
			next++
		})
		sel.Select(ctx)
		cancelTimer()
		if done {
			return nil
		}
	}
	return nil
}

// raiseTicket converts an unanswered queued chat into a durable ticket and
// tells the admin. Best-effort: a failure here must not fail the workflow,
// and the activity is idempotent on conversation_id.
func raiseTicket(ctx workflow.Context, in SupportQueueInput, waited time.Duration) {
	t := SupportQueueTicketInput{
		ConversationID: in.ConversationID,
		TenantID:       in.TenantID,
		CaseID:         in.CaseID,
		Subject:        in.Subject,
		IntakeReason:   in.IntakeReason,
		CustomerName:   in.CustomerName,
		CustomerEmail:  in.CustomerEmail,
		WaitedSeconds:  int(waited.Seconds()),
	}
	var out SupportQueueTicketResult
	if err := workflow.ExecuteActivity(ctx, SupportQueueTicketActivity, t).Get(ctx, &out); err != nil {
		workflow.GetLogger(ctx).Warn("support queue ticket creation failed",
			"conversation_id", in.ConversationID, "err", err)
		return
	}
	workflow.GetLogger(ctx).Info("support queue ticket raised",
		"conversation_id", in.ConversationID, "ticket", out.TicketNumber, "created", out.Created)
}
