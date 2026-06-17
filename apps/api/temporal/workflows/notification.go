// Package workflows holds HomeChef's Temporal workflow + activity definitions.
// NotificationWorkflow is the first durable flow (epic #116, sub-issue #126):
// it guarantees a notification is delivered, retrying transient failures, so a
// worker crash never silently drops a message the way a fire-and-forget
// goroutine does today.
package workflows

import (
	"context"
	"time"

	apitemporal "github.com/homechef/api/temporal"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// NotificationInput is the message to deliver. For "email" the Recipient is an
// address; for "push" it is the target user ID and Data carries the FCM payload.
type NotificationInput struct {
	Channel   string            `json:"channel"` // "email" | "push"
	Recipient string            `json:"recipient"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	Data      map[string]string `json:"data,omitempty"`
}

// SendFunc is the pluggable transport. The worker wires it to the existing
// services.* senders (mailer fallback chain / push). The default is a safe
// no-op so the workflow is runnable before integration (issue #126).
var SendFunc = func(_ context.Context, _ NotificationInput) error { return nil }

// SendNotificationActivity performs the actual send. It is automatically retried
// by the workflow's retry policy on error.
func SendNotificationActivity(ctx context.Context, in NotificationInput) error {
	activity.GetLogger(ctx).Info("send notification", "channel", in.Channel, "to", in.Recipient)
	return SendFunc(ctx, in)
}

// NotificationWorkflow durably delivers a single notification.
func NotificationWorkflow(ctx workflow.Context, in NotificationInput) error {
	ctx = apitemporal.Activities(ctx, 30*time.Second)
	return workflow.ExecuteActivity(ctx, SendNotificationActivity, in).Get(ctx, nil)
}
