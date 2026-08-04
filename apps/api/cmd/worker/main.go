// Command worker runs the HomeChef Temporal workers. Thanks to the reusable
// temporal module, a service's entire worker entrypoint is just dependency init
// plus a list of queues with their workflows/activities — add more as the
// migration grows (orders, payments, delivery, …). See epic #116.
package main

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/piicrypto"
	"github.com/homechef/api/services"
	"github.com/homechef/api/temporal"
	"github.com/homechef/api/temporal/workflows"
)

func main() {
	config.Load()

	// Workers are full app processes (minus the HTTP server): they need the same
	// dependencies the activities touch.
	if err := database.Connect(); err != nil {
		log.Fatalf("worker: database connect: %v", err)
	}
	// PII column encryption (#710). The worker is a full app process: its
	// activities GORM-scan orders and users, so it needs the DEK exactly as much
	// as the API server does. Without this every activity that loads a row with
	// an encrypted column fails permanently ("encrypted value but crypto not
	// initialized") and Temporal retries it forever — silent to the API's own
	// health checks. Fatal when enabled-but-failing, matching main.go.
	if err := piicrypto.InitIfEnabled(
		context.Background(),
		config.AppConfig.PIIEncryptionEnabled,
		config.AppConfig.GCSProjectID,
	); err != nil {
		log.Fatalf("worker: PII encryption enabled but failed to initialize: %v", err)
	}
	services.InitEmailService()
	if err := services.InitPushService(); err != nil {
		log.Printf("worker: push service init failed (push activities will error): %v", err)
	}
	// GCS. Same reasoning as the PII DEK above: the worker is a full app process
	// and its activities write objects, so it needs the storage client exactly as
	// much as the API does.
	//
	// Without this the account-purge cron panicked on every run: it archives an
	// account's financial records to the private bucket before erasing the row, and
	// UploadFile dereferenced a nil storageClient. runAccountPurgeScan recovers at
	// the top of the scan, so the panic aborted the WHOLE batch and no account was
	// ever erased — a silent DPDP retention failure that the API's own logs never
	// showed, because the cron runs here and not there.
	//
	// Non-fatal, matching main.go: an activity that cannot upload should fail and be
	// retried by Temporal, not stop the worker from serving every other task queue.
	if err := services.InitStorage(); err != nil {
		log.Printf("worker: GCS storage init failed (upload activities will error): %v", err)
	} else {
		defer services.CloseStorage()
	}
	// Secret Manager, for the same reason. The account purge deletes a chef's
	// payout bank secrets; without this client every attempt logged "secret
	// manager not initialized" and the most sensitive thing a vendor gives us
	// outlived their erasure. Non-fatal, matching main.go.
	if err := services.InitSecretManager(); err != nil {
		log.Printf("worker: secret manager init failed (secret deletion will be skipped): %v", err)
	}

	// Wire activity transports to the real services.* implementations.
	workflows.SendFunc = services.DispatchNotification
	workflows.DispatchFunc = func(_ context.Context, orderID uuid.UUID) error {
		return services.DispatchOrderDelivery(orderID)
	}
	// Order lifecycle saga activities (#122).
	workflows.NotifyChefFunc = services.NotifyChefNewOrder
	workflows.OrderSettleFunc = services.SettleOrderPayouts
	// Durable payment resolution — poll the gateway to a terminal answer.
	workflows.ResolvePaymentFunc = services.ResolveOrderPayment
	workflows.ExpireUnpaidOrderFunc = services.ExpireUnpaidOrder
	workflows.PaymentStalledFunc = services.PublishPaymentStalled
	workflows.OrderRefundFunc = services.CompensateOrderRefund
	// Onboarding activation (#126).
	workflows.ActivateChefFunc = services.ActivateChefOnboardingFromActivity
	// Confirm-receipt reminder + auto-confirm flow (#auto-confirm-delivery).
	workflows.ConfirmReminderFunc = func(_ context.Context, orderID uuid.UUID, attempt int) error {
		_, err := services.SendConfirmReceiptReminder(database.DB, orderID, attempt)
		return err
	}
	workflows.AutoConfirmFunc = func(_ context.Context, orderID uuid.UUID) error {
		_, _, err := services.AutoConfirmOrderReceipt(database.DB, orderID)
		return err
	}
	// Ready-to-collect flow for pickup orders: the ready notice, collection
	// reminders, then a chef escalation if nobody ever came. Every one of these
	// re-reads the order and no-ops once it has left `ready`, so an at-least-once
	// activity retry can't double-notify.
	workflows.PickupReadyNoticeFunc = func(_ context.Context, orderID uuid.UUID) error {
		return services.NotifyOrderReadyForPickup(database.DB, orderID)
	}
	workflows.PickupReminderFunc = func(_ context.Context, orderID uuid.UUID, attempt int) error {
		_, err := services.SendPickupReminder(database.DB, orderID, attempt)
		return err
	}
	workflows.PickupUncollectedFunc = func(_ context.Context, orderID uuid.UUID) error {
		_, err := services.EscalateUncollectedPickup(database.DB, orderID)
		return err
	}
	// Admin-initiated two-factor reset, held for 24h so a compromised admin
	// account cannot silently disarm someone's second factor (#login-otp-2fa).
	workflows.MFAResetNoticeFunc = func(_ context.Context, userID uuid.UUID, _ int) error {
		services.NotifySecurityEvent(services.NotifTypeMFAResetPending, userID)
		return nil
	}
	workflows.MFAResetCancelledFunc = func(_ context.Context, userID uuid.UUID) error {
		services.NotifySecurityEvent(services.NotifTypeMFAResetCancelled, userID)
		return nil
	}
	workflows.MFAResetApplyFunc = func(_ context.Context, userID uuid.UUID) error {
		if err := services.DisableMFA(database.DB, userID); err != nil {
			return err
		}
		services.NotifySecurityEvent(services.NotifTypeMFAResetApplied, userID)
		return nil
	}
	// Durable deferred chef-cancel gateway-refund retry — fires immediately on a
	// deferred cancel refund instead of waiting for the cron backstop.
	workflows.GatewayRefundFunc = services.GatewayRefundForWorkflow
	workflows.PersistRefundIDFunc = services.PersistDeferredRefundID
	// Durable mixed wallet + external payment flow (wallet-ledger Phase 5) — reserve/
	// capture/release ledger holds. Inert until WALLET_PAYMENT_FLOW_ENABLED + the ledger
	// are live; the activities are idempotent so a retry never double-moves money.
	workflows.PlaceWalletHoldFunc = func(_ context.Context, in workflows.WalletHoldActivityInput) error {
		_, err := services.PlaceWalletHold(database.DB, in.UserID, models.Money(in.AmountMinor), in.RefType, in.RefID)
		return err
	}
	workflows.CaptureWalletHoldFunc = func(_ context.Context, in workflows.WalletHoldRefInput) error {
		_, err := services.CaptureWalletHold(database.DB, in.RefType, in.RefID)
		return err
	}
	workflows.ReleaseWalletHoldFunc = func(_ context.Context, in workflows.WalletHoldRefInput) error {
		_, err := services.ReleaseWalletHold(database.DB, in.RefType, in.RefID)
		return err
	}

	// Support-chat staff queue SLA (otto.support.* NATS events).
	workflows.SupportQueueNotifyFunc = services.SendSupportQueueNotice
	workflows.SupportQueueTicketFunc = services.RaiseSupportQueueTicket

	if err := temporal.RunWorkers(
		temporal.Queue(temporal.TaskQueueNotifications).
			Workflows(workflows.NotificationWorkflow, workflows.SupportQueueWorkflow).
			Activities(workflows.SendNotificationActivity, workflows.SupportQueueNotifyActivity,
				workflows.SupportQueueTicketActivity),
		temporal.Queue(temporal.TaskQueueDelivery).
			Workflows(workflows.DeliveryWorkflow).
			Activities(workflows.DispatchDeliveryActivity),
		// Order lifecycle saga (#122) — notify → accept → ready → dispatch →
		// delivered → settle, with refund compensation.
		temporal.Queue(temporal.TaskQueueOrders).
			Workflows(workflows.OrderSagaWorkflow, workflows.ConfirmReceiptWorkflow,
				workflows.PickupReadyWorkflow).
			Activities(workflows.NotifyChefActivity, workflows.DispatchDeliveryActivity,
				workflows.OrderSettleActivity, workflows.OrderRefundActivity,
				workflows.ReminderActivity, workflows.AutoConfirmActivity,
				workflows.PickupReadyNoticeActivity, workflows.PickupReminderActivity,
				workflows.PickupUncollectedActivity),
		// Durable chef-onboarding activation (#126).
		temporal.Queue(temporal.TaskQueueOnboarding).
			Workflows(workflows.OnboardingActivationWorkflow).
			Activities(workflows.ActivateChefOnboardingActivity),
		// Admin-initiated two-factor reset, on the notifications queue since
		// every step of it is a message to the user plus one state change.
		temporal.Queue(temporal.TaskQueueNotifications).
			Workflows(workflows.AdminMFAResetWorkflow).
			Activities(
				workflows.MFAResetNoticeActivity,
				workflows.MFAResetApplyActivity,
				workflows.MFAResetCancelledActivity,
			),
		// Durable mixed wallet + external payment flow (wallet-ledger Phase 5) —
		// hold → await gateway → capture/release compensation. Also carries the
		// deferred chef-cancel gateway-refund retry (fires immediately on a
		// deferred cancel; RetryDeferredCancelRefunds cron remains the backstop).
		temporal.Queue(temporal.TaskQueuePayments).
			Workflows(workflows.WalletPaymentWorkflow, workflows.DeferredRefundWorkflow,
				workflows.PaymentResolutionWorkflow).
			Activities(workflows.PlaceWalletHoldActivity, workflows.CaptureWalletHoldActivity,
				workflows.ReleaseWalletHoldActivity, workflows.GatewayRefundActivity,
				workflows.PersistRefundIDActivity,
				workflows.ResolvePaymentActivity, workflows.ExpireUnpaidOrderActivity,
				workflows.PaymentStalledActivity),
		// Scheduled jobs (statements, reconciliation, FSSAI, availability, audit).
		services.RegisterCronWorker(),
	); err != nil {
		log.Fatalf("temporal worker: %v", err)
	}
}
