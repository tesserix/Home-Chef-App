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
	SubjectOrderCreated      = "orders.created"
	SubjectOrderUpdated      = "orders.updated"
	SubjectOrderCancelled    = "orders.cancelled"
	SubjectOrderDelivered    = "orders.delivered"
	SubjectChefNewOrder      = "chef.new_order"
	SubjectChefTipReceived   = "chef.tip_received"   // → chef: post-delivery tip
	SubjectDriverTipReceived = "driver.tip_received" // → rider: post-delivery tip

	// Group / office orders (#46)
	SubjectGroupOrderInvited   = "group_orders.invited"   // → guest: invited/joined
	SubjectGroupOrderLocked    = "group_orders.locked"    // → participants: pay your share
	SubjectGroupOrderPlaced    = "group_orders.placed"    // → host: order placed
	SubjectGroupOrderCancelled = "group_orders.cancelled" // → participants: cancelled/refunded
	SubjectDeliveryAssigned    = "delivery.assigned"
	SubjectDeliveryPickedUp    = "delivery.picked_up"
	SubjectDeliveryLocation    = "delivery.location" // Base subject; full subject: delivery.location.{deliveryID}
	SubjectPaymentSuccess      = "payments.success"
	SubjectPaymentFailed       = "payments.failed"
	SubjectUserRegistered      = "users.registered"
	SubjectChefVerified        = "chef.verified"
	SubjectReviewPosted        = "reviews.posted"
	SubjectCateringRequest     = "catering.request"
	SubjectCateringQuote       = "catering.quote"
	SubjectNotificationEmail   = "notifications.email"
	SubjectNotificationPush    = "notifications.push"
	SubjectNotificationSMS     = "notifications.sms"

	SubjectApprovalCreated       = "approvals.created"
	SubjectApprovalApproved      = "approvals.approved"
	SubjectApprovalRejected      = "approvals.rejected"
	SubjectApprovalInfoRequested = "approvals.info_requested"

	// Tiffin meal plans (#193) — the request→accept→approve handshake + per-day lifecycle.
	SubjectMealPlanCreated      = "meal_plans.created"       // → chef: new request
	SubjectMealPlanAcceptedFull = "meal_plans.accepted_full" // → customer: notify only
	SubjectMealPlanModified     = "meal_plans.modified"      // → customer: approve the trim
	SubjectMealPlanConfirmed    = "meal_plans.confirmed"
	SubjectMealPlanCancelled    = "meal_plans.cancelled"
	SubjectMealPlanDayDelivered = "meal_plans.day_delivered"
	SubjectMealPlanDayRefunded  = "meal_plans.day_refunded"

	SubjectDriverOnboardingSubmitted = "driver.onboarding.submitted"

	SubjectSubscriptionCreated        = "subscription.created"
	SubjectSubscriptionActivated      = "subscription.activated"
	SubjectSubscriptionPastDue        = "subscription.past_due"
	SubjectSubscriptionSuspended      = "subscription.suspended"
	SubjectSubscriptionCancelled      = "subscription.cancelled"
	SubjectSubscriptionInvoiceCreated = "subscription.invoice.created"
	SubjectEarningsThresholdMet       = "subscription.earnings.threshold_met"

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
		{"MEAL_PLANS", "Tiffin meal-plan lifecycle events", []string{"meal_plans.>"}, 30 * 24 * time.Hour, gib / 2},
		{"GROUP_ORDERS", "Group / office order lifecycle events", []string{"group_orders.>"}, 30 * 24 * time.Hour, gib / 2},
		{"PROVIDER", "Third-party delivery provider events", []string{"provider.>"}, 30 * 24 * time.Hour, gib / 2},
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
