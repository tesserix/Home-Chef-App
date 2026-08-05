// FSSAI filing-request SLA bridge. The request lifecycle publishes
// fssai.request.* to NATS; this consumer turns those into the
// FssaiRequestWorkflow loop: `submitted` SignalWithStarts the per-request
// workflow, every later status signals it, and a terminal status lets it
// complete.
//
// The chef pays up front and cannot get a refund, so a request going quiet is
// the one outcome that must not be possible. When Temporal is disabled this
// degrades to nothing — the admin queue is still the primary surface.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
	"go.temporal.io/api/serviceerror"
)

// RegisterFssaiSlaConsumers binds the durable consumer for filing-request
// events. Its own durable so an SLA backlog can never stall the notification
// consumers that share the stream.
func RegisterFssaiSlaConsumers(ctx context.Context, cm *ConsumerManager) error {
	return cm.RegisterAll(ctx, ConsumerSpec{
		Stream:   "FSSAI",
		Durable:  "fssai-request-sla",
		Subjects: []string{"fssai.request.>"},
		Handler:  handleFssaiRequestEvent,
	})
}

// fssaiEventEnvelope is the Event shape PublishEvent writes, narrowed to the
// fields the SLA loop needs.
type fssaiEventEnvelope struct {
	Data struct {
		RequestID    string `json:"requestId"`
		ChefID       string `json:"chefId"`
		Status       string `json:"status"`
		KitchenName  string `json:"kitchenName"`
		Mode         string `json:"mode"`
		AwaitingChef bool   `json:"awaitingChef"`
	} `json:"data"`
	UserID string `json:"user_id"`
}

func handleFssaiRequestEvent(ctx context.Context, subject string, data []byte) error {
	var ev fssaiEventEnvelope
	if err := json.Unmarshal(data, &ev); err != nil {
		log.Printf("[fssai-sla] drop unparseable event on %s: %v", subject, err)
		return nil // malformed — retrying will never fix it
	}
	if ev.Data.RequestID == "" {
		return nil
	}
	if temporalRT == nil {
		return nil // Temporal disabled — admin queue remains the surface
	}

	wfID := "homechef:fssai:" + ev.Data.RequestID
	sig := workflows.FssaiRequestSignal{
		Status:       ev.Data.Status,
		AwaitingChef: ev.Data.AwaitingChef,
	}

	if ev.Data.Status == models.FssaiSubmitted {
		in := workflows.FssaiRequestInput{
			RequestID:   ev.Data.RequestID,
			ChefID:      ev.Data.ChefID,
			UserID:      ev.UserID,
			KitchenName: ev.Data.KitchenName,
			Mode:        ev.Data.Mode,
		}
		return temporalRT.SignalWithStart(ctx, apitemporal.TaskQueueOnboarding, wfID,
			workflows.FssaiRequestSignalName, sig, workflows.FssaiRequestWorkflow, in)
	}

	err := temporalRT.Signal(ctx, wfID, workflows.FssaiRequestSignalName, sig)
	if _, notFound := err.(*serviceerror.NotFound); notFound {
		// Already completed, or a request that predates this workflow.
		return nil
	}
	return err
}

// SendFssaiSlaNotice delivers one nudge — to the chef when we are waiting on
// them, to onboarding when the request is sitting unworked.
//
// Wired to workflows.FssaiNotifyFunc by the worker.
func SendFssaiSlaNotice(_ context.Context, n workflows.FssaiNotice) error {
	waited := int(n.Waiting.Hours())
	if n.Kind == workflows.FssaiNoticeChefChase {
		userID, err := uuid.Parse(n.UserID)
		if err != nil {
			return nil // nothing to notify without a user
		}
		svc := GetNotificationService()
		if svc == nil {
			return nil
		}
		return svc.SaveUserNotification(&models.Notification{
			UserID: userID,
			Type:   "fssai_request_reminder",
			Title:  "Your FSSAI application is waiting on you",
			Message: "We still need what we asked for before we can file your " +
				"registration. Open the app to send it.",
		})
	}

	// Staff-facing: the request is paid and nobody has moved it.
	subject := fmt.Sprintf("FSSAI request unworked for %dh — %s", waited, n.KitchenName)
	if n.Kind == workflows.FssaiNoticeStalled {
		subject = fmt.Sprintf("FSSAI request STALLED %dh — %s", waited, n.KitchenName)
	}
	svc := GetEmailService()
	if svc == nil {
		log.Printf("[fssai-sla] %s: %s (no email service)", n.Kind, subject)
		return nil
	}
	body := fmt.Sprintf(
		`<p>A paid FSSAI filing request has been waiting <strong>%d hours</strong>.</p>
<table cellpadding="6">
<tr><td>Kitchen</td><td>%s</td></tr>
<tr><td>Request</td><td>%s</td></tr>
<tr><td>Mode</td><td>%s</td></tr>
</table>
<p>The chef has paid and cannot be refunded. Open it in admin and move it on.</p>`,
		waited, html.EscapeString(n.KitchenName),
		html.EscapeString(n.RequestID), html.EscapeString(n.Mode))
	return svc.Send(FssaiOnboardingRecipient, subject, body)
}

// FssaiRequestsAwaitingAction is the reconciliation view the SLA is a backstop
// for: paid requests that have been sitting in one state too long. Read-only,
// used by admin tooling and by the ops runbook.
func FssaiRequestsAwaitingAction(olderThan time.Duration) ([]models.FssaiRequest, error) {
	cutoff := time.Now().UTC().Add(-olderThan)
	var rows []models.FssaiRequest
	err := database.DB.Preload("Documents").
		Where("status IN ? AND updated_at < ?",
			[]string{models.FssaiSubmitted, models.FssaiInProgress, models.FssaiMoreInfoRequired},
			cutoff).
		Order("updated_at ASC").Find(&rows).Error
	return rows, err
}
