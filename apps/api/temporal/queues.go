package temporal

// Task-queue naming convention (per ADR #118): "<product>-<domain>".
// Workflow IDs follow "<product>:<domain>:<entityID>" for idempotency.
const (
	TaskQueueOrders        = "homechef-orders"
	TaskQueuePayments      = "homechef-payments"
	TaskQueueDelivery      = "homechef-delivery"
	TaskQueuePayouts       = "homechef-payouts"
	TaskQueueOnboarding    = "homechef-onboarding"
	TaskQueueNotifications = "homechef-notifications"
	TaskQueueSubscriptions = "homechef-subscriptions"
)
