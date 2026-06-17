package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	apitemporal "github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

// temporalRT is the process-wide Temporal runtime, set by main() when Temporal
// is configured (TEMPORAL_HOSTPORT). When nil, notification delivery runs inline
// — so the API behaves exactly as before until the cluster (#119) is up.
var temporalRT *apitemporal.Runtime

// SetTemporalRuntime registers the runtime used by EnqueueNotification. Pass nil
// to disable (inline fallback). Called once from main().
func SetTemporalRuntime(rt *apitemporal.Runtime) { temporalRT = rt }

// TemporalEnabled reports whether durable execution is active.
func TemporalEnabled() bool { return temporalRT != nil }

// EnqueueNotification delivers a notification durably via Temporal when enabled,
// guaranteeing exactly-once delivery with retries even across a crash. When
// Temporal is not configured it falls back to sending inline (current behaviour),
// so callers can adopt this safely anywhere a fire-and-forget notify exists today.
func EnqueueNotification(ctx context.Context, in workflows.NotificationInput) error {
	if temporalRT == nil {
		return DispatchNotification(ctx, in)
	}
	id := fmt.Sprintf("homechef:notification:%s", uuid.NewString())
	_, err := temporalRT.Start(ctx, apitemporal.TaskQueueNotifications, id, workflows.NotificationWorkflow, in)
	return err
}

// DispatchNotification performs the actual send. It is the work the Temporal
// activity executes (wired via workflows.SendFunc in the worker) and also the
// inline fallback used when Temporal is disabled.
func DispatchNotification(ctx context.Context, in workflows.NotificationInput) error {
	switch in.Channel {
	case "email":
		return GetEmailService().Send(in.Recipient, in.Subject, in.Body)
	case "push":
		uid, err := uuid.Parse(in.Recipient)
		if err != nil {
			return fmt.Errorf("push notification: invalid recipient user id %q: %w", in.Recipient, err)
		}
		return SendPushNotification(uid, in.Subject, in.Body, in.Data)
	default:
		return fmt.Errorf("notification: unsupported channel %q", in.Channel)
	}
}
