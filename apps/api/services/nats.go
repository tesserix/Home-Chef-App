package services

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NATS subjects for different event types
const (
	SubjectOrderCreated   = "orders.created"
	SubjectOrderUpdated   = "orders.updated"
	SubjectOrderCancelled = "orders.cancelled"
	// SubjectOrderVoided — a paid order the chef never accepted before their
	// kitchen closed (#694). Distinct from orders.cancelled on purpose: nobody
	// chose this, the customer is owed an apology as well as their money, and the
	// two need different copy and different reporting.
	SubjectOrderVoided = "orders.voided"
	// SubjectOrderAcceptReminder — an unaccepted order is inside the final two
	// hours before the kitchen closes (#694).
	SubjectOrderAcceptReminder = "orders.accept_reminder"
	// SubjectOrderStale — an order the chef ACCEPTED and then never finished. Separate
	// from accept_reminder, which chases an order nobody took: this one has an owner who
	// stopped, so it goes to BOTH sides and ends in a refund rather than a void.
	SubjectOrderStale = "orders.stale_reminder"
	SubjectOrderDelivered      = "orders.delivered"
	// SubjectOrderReadyForPickup — a PICKUP order is cooked and waiting to be
	// collected. Distinct from orders.updated on purpose: for a delivery order
	// `ready` is a passive milestone the customer does nothing about, but for a
	// pickup order it is the moment they have to act on, and the message needs the
	// kitchen's name in it. Lumping the two together is why every pickup customer
	// got the limp "ready for pickup/delivery" catch-all.
	SubjectOrderReadyForPickup = "orders.ready_for_pickup"
	// SubjectOrderPickupReminder — the customer still hasn't collected. Driven by
	// the durable pickup flow, not a ticker.
	SubjectOrderPickupReminder = "orders.pickup_reminder"
	// SubjectOrderPickupUncollected — the reminder window is exhausted and the food
	// is still on the chef's counter. Goes to the CHEF: they are the one holding
	// cooked food and the only one who can decide what to do with it.
	SubjectOrderPickupUncollected = "orders.pickup_uncollected"
	SubjectOrderIssueReported     = "orders.issue_reported"   // → chef: a customer reported an order issue (#37)
	SubjectOrderConfirmReminder   = "orders.confirm_reminder" // → customer: reminder to confirm receipt (auto-confirm flow)
	// Cancellation with vendor arbitration (#475).
	SubjectCancellationRequested = "orders.cancellation_requested" // → chef: confirm the cancellation
	SubjectCancellationResolved  = "orders.cancellation_resolved"  // → customer: refund issued
	SubjectChefNewOrder          = "chef.new_order"
	SubjectChefTipReceived       = "chef.tip_received"   // → chef: post-delivery tip
	SubjectDriverTipReceived     = "driver.tip_received" // → rider: post-delivery tip

	// Group / office orders (#46)
	SubjectGroupOrderInvited   = "group_orders.invited"   // → guest: invited/joined
	SubjectGroupOrderLocked    = "group_orders.locked"    // → participants: pay your share
	SubjectGroupOrderPlaced    = "group_orders.placed"    // → host: order placed
	SubjectGroupOrderCancelled = "group_orders.cancelled" // → participants: cancelled/refunded
	SubjectGroupOrderFailed    = "group_orders.failed"    // → host: delivery failed, resolution pending (#393/#594)
	SubjectDeliveryAssigned    = "delivery.assigned"
	SubjectDeliveryPickedUp    = "delivery.picked_up"
	SubjectDeliveryFailed      = "delivery.failed"   // → customer/chef: delivery failed, resolution pending (#393)
	SubjectDeliveryLocation    = "delivery.location" // Base subject; full subject: delivery.location.{deliveryID}
	SubjectPaymentSuccess = "payments.success"
	SubjectPaymentFailed  = "payments.failed"
	// SubjectPaymentStalled — a gateway payment has been live but unresolved for
	// longer than any customer would wait (bank OTP page left open, a UPI collect
	// nobody approved). Deliberately an OPS signal, not a customer notification:
	// there is nothing to tell the customer that the order screen does not
	// already say, and the value is in seeing a run of these when a gateway or a
	// bank rail degrades. No notification handler subscribes to it.
	SubjectPaymentStalled = "payments.stalled"
	// There is deliberately NO payments.refunded subject: every refund path
	// already reaches the customer through orders.cancelled, orders.voided or
	// orders.cancellation_resolved (and the chef's fee reduction through its own
	// push). A fourth event would notify twice for the same money.
	// Payout hold state machine (#387). Both route to the PAYMENTS stream
	// (payments.>) and drive the admin release queue (#388) downstream.
	SubjectHoldReleaseEligible = "payments.hold_release_eligible" // → hold advanced awaiting → release_eligible
	SubjectHoldDisputed        = "payments.hold_disputed"         // → hold advanced awaiting → disputed
	SubjectHoldReleased        = "payments.hold_released"         // → admin payout queue released the hold (#388)
	SubjectUserRegistered      = "users.registered"
	// Account lifecycle (DPDP). deleted/restored notify the user — the deleted
	// handler resolves the address with an Unscoped lookup, because the row is
	// soft-deleted by the time the consumer runs. purged is downstream-only:
	// the account no longer exists to notify, but analytics/cleanup consumers
	// need to know the erasure completed.
	SubjectAccountDeleted  = "users.account_deleted"
	SubjectAccountRestored = "users.account_restored"
	SubjectAccountPurged   = "users.account_purged"
	SubjectChefVerified    = "chef.verified"
	// Docs-deadline guardrail: warning fires 5 days before the 30-day document
	// window closes; expired fires when the window lapses and the pending
	// application is withdrawn (chef must re-apply).
	SubjectChefDocsDeadlineWarning = "chef.docs_deadline.warning"
	SubjectChefDocsDeadlineExpired = "chef.docs_deadline.expired"
	// Day-25 nudge for a chef who onboarded without payout details: earnings
	// hold until a bank account is added (no removal — money just waits).
	SubjectChefPayoutReminder = "chef.payout_details.reminder"
	// SubjectChefAvailabilityChanged — the chef opened, closed or paused their
	// kitchen. Customers browsing a closed kitchen would otherwise keep seeing
	// it as open until their chef list happens to refetch.
	SubjectChefAvailabilityChanged = "chef.availability_changed"
	SubjectWeeklyMenuPublished     = "chef.weekly_menu.published" // → followers: a favorited chef dropped a new menu (#239)
	SubjectDailyMenuPublished      = "chef.daily_menu.published"  // → followers: a favorited chef published a per-date menu (#405)
	SubjectReferralRewarded        = "referral.reward.granted"    // → referrer: a referee placed their first paid order (#38)
	SubjectLoyaltyEarned           = "loyalty.points_earned"      // → customer: earned points on a delivered order / streak (#40)
	SubjectLoyaltyRedeemed         = "loyalty.redeemed"           // → customer: points converted to wallet credit (#40)
	SubjectCampaignDispatch        = "campaigns.dispatch"         // → fan out a marketing campaign to its segment (#56)
	SubjectReviewPosted            = "reviews.posted"
	SubjectCateringRequest         = "catering.request"
	SubjectCateringQuote           = "catering.quote"
	SubjectNotificationEmail       = "notifications.email"
	SubjectNotificationPush        = "notifications.push"
	SubjectNotificationSMS         = "notifications.sms"

	// SubjectRiskCustomerFlagged — a customer's refund-abuse score reached severe (#937).
	// Fans out to admins; the money paths never consume it.
	SubjectRiskCustomerFlagged = "risk.customer_flagged"

	SubjectApprovalCreated       = "approvals.created"
	SubjectApprovalApproved      = "approvals.approved"
	SubjectApprovalRejected      = "approvals.rejected"
	SubjectApprovalInfoRequested = "approvals.info_requested"
	// SubjectApprovalReminded — the submitter bumped an unattended request (#697).
	// Carries reminderCount + escalated so a consumer can tell a nudge from a
	// request that has been ignored for days.
	SubjectApprovalReminded = "approvals.reminded"

	// Tiffin meal plans (#193) — the request→accept→approve handshake + per-day lifecycle.
	SubjectMealPlanCreated        = "meal_plans.created"       // → chef: new request
	SubjectMealPlanAcceptedFull   = "meal_plans.accepted_full" // → customer: notify only
	SubjectMealPlanModified       = "meal_plans.modified"      // → customer: approve the trim
	SubjectMealPlanConfirmed      = "meal_plans.confirmed"
	SubjectMealPlanCancelled      = "meal_plans.cancelled"
	SubjectMealPlanDayPrepared    = "meal_plans.day_prepared" // → customer: dish is being cooked (#50)
	SubjectMealPlanDayDelivered   = "meal_plans.day_delivered"
	SubjectMealPlanDayFailed      = "meal_plans.day_failed" // → customer/chef: day delivery failed, resolution pending (#393)
	SubjectMealPlanDayRefunded    = "meal_plans.day_refunded"
	SubjectMealPlanDaySkippedChef = "meal_plans.day_skipped_chef" // → chef: customer skipped a day (#422)
	// #422 policy change: a skip is now an admin-reviewed REQUEST, not an auto-credit.
	SubjectMealPlanDaySkipRequested = "meal_plans.day_skip_requested" // → chef: customer requested a skip, pending admin review
	SubjectMealPlanDaySkipDeclined  = "meal_plans.day_skip_declined"  // → customer: admin declined the skip; the day stands
	SubjectMealPlanCompleted        = "meal_plans.completed"          // → customer: every day served, plan done
	SubjectMealPlanChefReminder     = "meal_plans.chef_cook_reminder" // → chef: tiffin meals to cook today/tomorrow
	SubjectMealPlanPayoutReleased   = "meal_plans.payout_released"    // → chef: a tiffin day's payment was released

	SubjectDriverOnboardingSubmitted = "driver.onboarding.submitted"

	SubjectSubscriptionCreated        = "subscription.created"
	SubjectSubscriptionActivated      = "subscription.activated"
	SubjectSubscriptionPastDue        = "subscription.past_due"
	SubjectSubscriptionSuspended      = "subscription.suspended"
	SubjectSubscriptionCancelled      = "subscription.cancelled"
	SubjectSubscriptionWinbackOffered = "subscription.winback_offered"
	SubjectSubscriptionInvoiceCreated = "subscription.invoice.created"
	// Customer meal subscription (tiffin, #2/#3) — on the SUBSCRIPTIONS stream.
	SubjectMealSubscriptionCreated   = "subscription.meal.created"
	SubjectMealSubscriptionPaused    = "subscription.meal.paused"
	SubjectMealSubscriptionResumed   = "subscription.meal.resumed"
	SubjectMealSubscriptionCancelled = "subscription.meal.cancelled"
	SubjectEarningsThresholdMet      = "subscription.earnings.threshold_met"

	SubjectProviderDeliveryCreated = "provider.delivery.created"
	SubjectProviderDeliveryUpdated = "provider.delivery.updated"
	SubjectProviderDeliveryFailed  = "provider.delivery.failed"

	// Per-user notification subject for real-time bell updates.
	// Full subject: notifications.user.{userID}
	SubjectNotificationUser = "notifications.user"

	// DLQSubjectPrefix is the root subject for dead-lettered events. A durable
	// consumer that exhausts its retries republishes the poison message to
	// dlq.<stream>.<durable>, captured by the DLQ stream for inspection/replay.
	DLQSubjectPrefix = "dlq"
)

// Event represents a generic event message
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	UserID    uuid.UUID              `json:"user_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
}

// OrderEvent represents an order-related event
type OrderEvent struct {
	OrderID     uuid.UUID `json:"order_id"`
	OrderNumber string    `json:"order_number,omitempty"`
	CustomerID  uuid.UUID `json:"customer_id"`
	ChefID      uuid.UUID `json:"chef_id"`
	Status      string    `json:"status"`
	Total       float64   `json:"total"`
	// FulfillmentType lets a consumer word the message correctly without
	// re-reading the order. `ready` and `delivered` mean different things for a
	// pickup order than a delivered one, and the notification handler is a NATS
	// consumer with only this payload to go on. Omitempty keeps older queued
	// events decoding cleanly — an absent value reads as delivery, the server's
	// own default.
	FulfillmentType string `json:"fulfillment_type,omitempty"`
	// Reason is free text for the diagnostic subjects (payments.stalled) that
	// carry a "why" no status enum captures. Omitempty — every existing producer
	// and consumer is unaffected.
	Reason string `json:"reason,omitempty"`
}

// NotificationEvent represents a notification to be sent
type NotificationEvent struct {
	UserID  uuid.UUID              `json:"user_id"`
	Type    string                 `json:"type"` // email, push, sms
	Title   string                 `json:"title"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// NATSClient wraps the NATS connection and JetStream context
type NATSClient struct {
	conn      *nats.Conn
	js        jetstream.JetStream
	mu        sync.RWMutex
	connected bool
}

var (
	natsClient *NATSClient
	natsOnce   sync.Once
)

// GetNATSClient returns the singleton NATS client
func GetNATSClient() *NATSClient {
	natsOnce.Do(func() {
		natsClient = &NATSClient{}
	})
	return natsClient
}

// Connect establishes connection to NATS server
func (n *NATSClient) Connect() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	opts := []nats.Option{
		nats.Name("homechef-api"),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // Unlimited reconnects
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("NATS reconnected to %s", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Printf("NATS disconnected: %v", err)
			}
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Printf("NATS error: %v", err)
		}),
	}

	// Authenticate with account credentials when the cluster enforces
	// operator/JWT auth (Home-Chef-App#758). Empty until then → anonymous.
	if creds := config.AppConfig.NATSCreds; creds != "" {
		opts = append(opts, nats.UserCredentials(creds))
		log.Printf("NATS using credentials file %s", creds)
	}

	conn, err := nats.Connect(config.AppConfig.NATSURL, opts...)
	if err != nil {
		return err
	}
	n.conn = conn

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return err
	}
	n.js = js

	// Setup streams
	if err := n.setupStreams(); err != nil {
		log.Printf("Warning: Failed to setup NATS streams: %v", err)
	}

	n.connected = true
	log.Printf("Connected to NATS at %s", config.AppConfig.NATSURL)
	return nil
}

// streamDef declares a JetStream stream in one place. See setupStreams.
type streamDef struct {
	name     string
	desc     string
	subjects []string
	maxAge   time.Duration
	maxBytes int64
}

// setupStreams creates/updates the JetStream streams.
//
// Design (issue #135): streams use LimitsPolicy (NOT WorkQueuePolicy) so that
//   - multiple independent durable consumers can read the same subjects
//     (notification-workers AND push-workers both consume orders.updated), and
//   - the stream is bounded by both MaxAge and MaxBytes with DiscardOld, so it
//     can never fill the JetStream PVC and block publishes.
//
// High-frequency, real-time-only subjects are deliberately NOT captured by any
// stream and remain core-NATS pub/sub:
//   - notifications.user.*  (per-user notification-bell fan-out)
//   - delivery.location.*   (live driver GPS)
func (n *NATSClient) setupStreams() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	const gib = int64(1024 * 1024 * 1024)
	defs := []streamDef{
		{"ORDERS", "Order lifecycle events", []string{"orders.>"}, 7 * 24 * time.Hour, gib},
		{"PAYMENTS", "Payment events", []string{"payments.>"}, 30 * 24 * time.Hour, gib},
		{"CHEF", "Chef events", []string{"chef.>"}, 7 * 24 * time.Hour, gib / 2},
		// delivery.location.* (high-frequency GPS) is intentionally excluded.
		{"DELIVERY", "Delivery + driver onboarding events", []string{"delivery.assigned", "delivery.picked_up", "driver.>"}, 7 * 24 * time.Hour, gib / 2},
		// notifications.user.* (real-time bell) is intentionally excluded.
		{"NOTIFICATIONS", "Notification dispatch events", []string{"notifications.email", "notifications.push", "notifications.sms"}, 3 * 24 * time.Hour, gib},
		{"USERS", "User events", []string{"users.>"}, 7 * 24 * time.Hour, gib / 4},
		{"REVIEWS", "Review events", []string{"reviews.>"}, 7 * 24 * time.Hour, gib / 4},
		{"CATERING", "Catering events", []string{"catering.>"}, 30 * 24 * time.Hour, gib / 4},
		{"APPROVALS", "Approval lifecycle events", []string{"approvals.>"}, 30 * 24 * time.Hour, gib / 2},
		{"SUBSCRIPTIONS", "Subscription billing events", []string{"subscription.>"}, 30 * 24 * time.Hour, gib / 2},
		{"REFERRAL", "Referral program events", []string{"referral.>"}, 30 * 24 * time.Hour, gib / 4},
		{"LOYALTY", "Loyalty points & streak events", []string{"loyalty.>"}, 30 * 24 * time.Hour, gib / 4},
		{"CAMPAIGNS", "Marketing campaign dispatch events", []string{"campaigns.>"}, 30 * 24 * time.Hour, gib / 4},
		{"MEAL_PLANS", "Tiffin meal-plan lifecycle events", []string{"meal_plans.>"}, 30 * 24 * time.Hour, gib / 2},
		{"GROUP_ORDERS", "Group / office order lifecycle events", []string{"group_orders.>"}, 30 * 24 * time.Hour, gib / 2},
		{"PROVIDER", "Third-party delivery provider events", []string{"provider.>"}, 30 * 24 * time.Hour, gib / 2},
		// Customer refund-abuse flags (#937). Long retention: these drive an
		// investigation an admin may not open for days.
		{"RISK", "Customer refund-abuse risk events", []string{"risk.>"}, 30 * 24 * time.Hour, gib / 4},
		// otto (support-platform) publishes these with a Nats-Msg-Id per
		// transition, so the 2-minute dedup window absorbs the duplicate
		// publishes from otto's replicated change-stream watchers.
		{"SUPPORT", "Otto support-chat staff queue events", []string{"otto.support.>"}, 7 * 24 * time.Hour, gib / 4},
		{"DLQ", "Dead-letter: events that exhausted consumer retries", []string{DLQSubjectPrefix + ".>"}, 30 * 24 * time.Hour, gib},
	}

	replicas := config.AppConfig.NATSStreamReplicas
	if replicas < 1 {
		replicas = 1
	}

	var firstErr error
	for _, d := range defs {
		_, err := n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
			Name:        d.name,
			Description: d.desc,
			Subjects:    d.subjects,
			Retention:   jetstream.LimitsPolicy,
			Discard:     jetstream.DiscardOld,
			MaxAge:      d.maxAge,
			MaxBytes:    d.maxBytes,
			Storage:     jetstream.FileStorage,
			Replicas:    replicas,
		})
		if err != nil {
			// A stream originally created with WorkQueuePolicy cannot be switched
			// to LimitsPolicy in place — NATS rejects a retention-policy change.
			// Surface an actionable warning; the one-time migration (drain + delete
			// the old stream so it is recreated) is in the tesserix-k8s NATS runbook.
			log.Printf("NATS stream %s: setup failed (if this is a retention change on an existing WorkQueue stream, delete the stream once so it is recreated): %v", d.name, err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	log.Printf("NATS JetStream streams configured (%d streams, replicas=%d)", len(defs), replicas)
	return firstErr
}

// Publish publishes a message to a subject
func (n *NATSClient) Publish(subject string, data interface{}) error {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.connected || n.conn == nil {
		return nats.ErrConnectionClosed
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return n.conn.Publish(subject, payload)
}

// PublishAsync publishes a message asynchronously with JetStream
func (n *NATSClient) PublishAsync(ctx context.Context, subject string, data interface{}) (jetstream.PubAckFuture, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.connected || n.js == nil {
		return nil, nats.ErrConnectionClosed
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return n.js.PublishAsync(subject, payload)
}

// PublishJS publishes durably to JetStream and BLOCKS until the broker confirms
// the message was persisted (PubAck). Unlike core Publish, a nil error here
// means the event is safely stored in the stream and will be delivered to
// durable consumers even across a publisher crash. msgID is set as the
// Nats-Msg-Id header so JetStream dedups duplicate publishes within the stream's
// dedup window (the outbox relay relies on this for at-least-once → effectively
// once). The subject MUST be covered by a configured stream, else JetStream
// returns "no stream matches subject".
func (n *NATSClient) PublishJS(ctx context.Context, subject, msgID string, data interface{}) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return n.PublishJSRaw(ctx, subject, msgID, payload)
}

// PublishJSRaw is PublishJS for an already-encoded payload (used by the outbox
// relay, which stores the marshalled event).
func (n *NATSClient) PublishJSRaw(ctx context.Context, subject, msgID string, payload []byte) error {
	n.mu.RLock()
	js := n.js
	connected := n.connected
	n.mu.RUnlock()

	if !connected || js == nil {
		return nats.ErrConnectionClosed
	}

	opts := []jetstream.PublishOpt{}
	if msgID != "" {
		opts = append(opts, jetstream.WithMsgID(msgID))
	}
	_, err := js.Publish(ctx, subject, payload, opts...)
	return err
}

// Subscribe subscribes to a subject with a handler
func (n *NATSClient) Subscribe(subject string, handler nats.MsgHandler) (*nats.Subscription, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.connected || n.conn == nil {
		return nil, nats.ErrConnectionClosed
	}

	return n.conn.Subscribe(subject, handler)
}

// QueueSubscribe subscribes to a subject with a queue group
func (n *NATSClient) QueueSubscribe(subject, queue string, handler nats.MsgHandler) (*nats.Subscription, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.connected || n.conn == nil {
		return nil, nats.ErrConnectionClosed
	}

	return n.conn.QueueSubscribe(subject, queue, handler)
}

// CreateConsumer creates a JetStream consumer
func (n *NATSClient) CreateConsumer(ctx context.Context, stream string, cfg jetstream.ConsumerConfig) (jetstream.Consumer, error) {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if !n.connected || n.js == nil {
		return nil, nats.ErrConnectionClosed
	}

	return n.js.CreateOrUpdateConsumer(ctx, stream, cfg)
}

// GetJetStream returns the JetStream context
func (n *NATSClient) GetJetStream() jetstream.JetStream {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.js
}

// IsConnected returns the connection status
func (n *NATSClient) IsConnected() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.connected && n.conn != nil && n.conn.IsConnected()
}

// Close closes the NATS connection
func (n *NATSClient) Close() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.conn != nil {
		n.conn.Drain()
		n.conn.Close()
		n.connected = false
		log.Println("NATS connection closed")
	}
}

// PublishEvent publishes a generic event
func PublishEvent(subject string, eventType string, userID uuid.UUID, data map[string]interface{}) error {
	event := Event{
		ID:        generateEventID(),
		Type:      eventType,
		Timestamp: time.Now().UTC(),
		UserID:    userID,
		Data:      data,
	}
	return GetNATSClient().Publish(subject, event)
}

// PublishOrderEvent publishes an order event
func PublishOrderEvent(subject string, order OrderEvent) error {
	return GetNATSClient().Publish(subject, order)
}

// notificationSubject maps a notification channel to its NATS subject. Shared by
// the core-publish and outbox-enqueue paths so the routing lives in one place.
func notificationSubject(channel string) string {
	switch channel {
	case "email":
		return SubjectNotificationEmail
	case "sms":
		return SubjectNotificationSMS
	default:
		return SubjectNotificationPush
	}
}

// PublishNotification publishes a notification event
func PublishNotification(notif NotificationEvent) error {
	return GetNATSClient().Publish(notificationSubject(notif.Type), notif)
}

// generateEventID returns a collision-free unique event ID. It is also used as
// the JetStream Nats-Msg-Id for broker-side dedup and consumer idempotency, so
// it MUST be unique — the previous time-based generator emitted repeated
// characters under concurrency (issue #138).
func generateEventID() string {
	return uuid.NewString()
}
