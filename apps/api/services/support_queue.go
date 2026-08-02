// Support-chat staff queue bridge. Otto (support-platform) publishes queue
// transitions to NATS (otto.support.<tenant>.<event>, see the SUPPORT
// stream); this consumer turns them into the SupportQueueWorkflow SLA loop:
// created/escalated SignalWithStart the per-conversation workflow, and
// accepted/closed signal it to stop. When Temporal is disabled the first
// notice still goes out inline so staff never miss a waiting chat.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	"github.com/homechef/api/config"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
	"go.temporal.io/api/serviceerror"
)

// Tenants otto scopes HomeChef conversations to. The vendor tenant keeps
// chef threads in their own queue lane (see otto TenantReasons).
const (
	OttoTenantCustomer = "homechef"
	OttoTenantVendor   = "homechef-vendor"
)

// ottoQueueEvent mirrors otto's event.QueueEvent wire payload.
type ottoQueueEvent struct {
	Event          string    `json:"event"`
	TenantID       string    `json:"tenant_id"`
	StoreID        string    `json:"store_id"`
	ConversationID string    `json:"conversation_id"`
	CaseID         string    `json:"case_id"`
	Status         string    `json:"status"`
	NeedsHuman     bool      `json:"needs_human"`
	Subject        string    `json:"subject"`
	IntakeReason   string    `json:"intake_reason"`
	CustomerName   string    `json:"customer_name"`
	CustomerEmail  string    `json:"customer_email"`
	AssigneeName   string    `json:"assignee_name"`
	OccurredAt     time.Time `json:"occurred_at"`
}

// RegisterSupportQueueConsumers binds the durable consumer for otto queue
// events. Call after the ConsumerManager is up; no-op error when the SUPPORT
// stream is missing (NATS down ⇒ caller already warned).
func RegisterSupportQueueConsumers(ctx context.Context, cm *ConsumerManager) error {
	return cm.RegisterAll(ctx, ConsumerSpec{
		Stream:  "SUPPORT",
		Durable: "support-queue-workers",
		Subjects: []string{
			"otto.support." + OttoTenantCustomer + ".>",
			"otto.support." + OttoTenantVendor + ".>",
		},
		Handler: handleOttoQueueEvent,
	})
}

func handleOttoQueueEvent(ctx context.Context, subject string, data []byte) error {
	var ev ottoQueueEvent
	if err := json.Unmarshal(data, &ev); err != nil {
		log.Printf("[support-queue] drop unparseable event on %s: %v", subject, err)
		return nil // malformed — retrying will never fix it
	}
	if ev.ConversationID == "" {
		return nil
	}

	wfID := "homechef:support-queue:" + ev.ConversationID
	sig := workflows.SupportQueueSignal{Event: ev.Event, AssigneeName: ev.AssigneeName}

	switch ev.Event {
	case "created", "escalated":
		in := workflows.SupportQueueInput{
			ConversationID: ev.ConversationID,
			TenantID:       ev.TenantID,
			CaseID:         ev.CaseID,
			Subject:        ev.Subject,
			IntakeReason:   ev.IntakeReason,
			CustomerName:   ev.CustomerName,
			Escalated:      ev.Event == "escalated",
		}
		if temporalRT == nil {
			// Inline fallback — at least the immediate notice goes out.
			kind := workflows.SupportNoticeWaiting
			if in.Escalated {
				kind = workflows.SupportNoticeEscalated
			}
			return SendSupportQueueNotice(ctx, workflows.SupportQueueNotice{
				Kind:           kind,
				ConversationID: in.ConversationID,
				TenantID:       in.TenantID,
				CaseID:         in.CaseID,
				Subject:        in.Subject,
				IntakeReason:   in.IntakeReason,
				CustomerName:   in.CustomerName,
			})
		}
		return temporalRT.SignalWithStart(ctx, apitemporal.TaskQueueNotifications, wfID,
			workflows.SupportQueueSignalName, sig, workflows.SupportQueueWorkflow, in)

	case "accepted", "closed":
		if temporalRT == nil {
			return nil
		}
		err := temporalRT.Signal(ctx, wfID, workflows.SupportQueueSignalName, sig)
		if _, notFound := err.(*serviceerror.NotFound); notFound {
			return nil // workflow already completed (or predates this feature)
		}
		return err
	default:
		return nil
	}
}

// SendSupportQueueNotice emails the staff mailbox about a queue transition.
// Wired as workflows.SupportQueueNotifyFunc in the worker and used inline
// when Temporal is off.
func SendSupportQueueNotice(_ context.Context, n workflows.SupportQueueNotice) error {
	to := config.AppConfig.SupportStaffEmail
	if to == "" {
		return nil
	}

	lane := "Customer"
	if n.TenantID == OttoTenantVendor {
		lane = "Chef / vendor"
	}
	who := n.CustomerName
	if who == "" {
		who = "A user"
	}

	var subject, lead string
	switch n.Kind {
	case workflows.SupportNoticeEscalated:
		subject = fmt.Sprintf("[Support chat] Otto handed off %s — %s waiting", n.CaseID, who)
		lead = "Otto escalated this chat to a human. The customer is waiting in the live queue."
	case workflows.SupportNoticeReminder:
		subject = fmt.Sprintf("[Support chat] Still waiting %s — %s (%s)", n.CaseID, who, waitingLabel(n.Waiting))
		lead = "This chat has not been accepted yet. Please pick it up from the live-chat inbox."
	case workflows.SupportNoticeUnattended:
		subject = fmt.Sprintf("[Support chat] UNATTENDED %s — waiting %s", n.CaseID, waitingLabel(n.Waiting))
		lead = "SLA breach: nobody has accepted this chat. It needs attention now."
	default: // waiting
		subject = fmt.Sprintf("[Support chat] New chat in queue %s — %s", n.CaseID, who)
		lead = "A new support chat is waiting for a human in the live queue."
	}

	inboxURL := config.AppConfig.AdminLiveChatURL
	body := fmt.Sprintf(`
		<p>%s</p>
		<table cellpadding="4" style="border-collapse:collapse">
			<tr><td><b>Queue</b></td><td>%s</td></tr>
			<tr><td><b>Case</b></td><td>%s</td></tr>
			<tr><td><b>From</b></td><td>%s</td></tr>
			<tr><td><b>Reason</b></td><td>%s</td></tr>
			<tr><td><b>Subject</b></td><td>%s</td></tr>
		</table>
		<p><a href="%s">Open the live-chat inbox</a></p>`,
		html.EscapeString(lead),
		html.EscapeString(lane),
		html.EscapeString(n.CaseID),
		html.EscapeString(who),
		html.EscapeString(n.IntakeReason),
		html.EscapeString(n.Subject),
		inboxURL,
	)
	return GetEmailService().Send(to, subject, body)
}

func waitingLabel(d time.Duration) string {
	if d < time.Minute {
		return "under a minute"
	}
	return strings.TrimSuffix(d.Round(time.Minute).String(), "0s")
}
