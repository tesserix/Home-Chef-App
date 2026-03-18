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
	SubjectDeliveryAssigned  = "delivery.assigned"
	SubjectDeliveryPickedUp  = "delivery.picked_up"
	SubjectPaymentSuccess    = "payments.success"
	SubjectPaymentFailed     = "payments.failed"
	SubjectUserRegistered    = "users.registered"
	SubjectChefVerified      = "chef.verified"
	SubjectReviewPosted      = "reviews.posted"
	SubjectCateringRequest   = "catering.request"
	SubjectCateringQuote     = "catering.quote"
	SubjectNotificationEmail = "notifications.email"
	SubjectNotificationPush  = "notifications.push"
	SubjectNotificationSMS   = "notifications.sms"

	SubjectApprovalCreated       = "approvals.created"
	SubjectApprovalApproved      = "approvals.approved"
	SubjectApprovalRejected      = "approvals.rejected"
	SubjectApprovalInfoRequested = "approvals.info_requested"

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
	UserID  uuid.UUID `json:"user_id"`
	Type    string    `json:"type"` // email, push, sms
	Title   string    `json:"title"`
	Message string    `json:"message"`
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

// setupStreams creates the necessary JetStream streams
func (n *NATSClient) setupStreams() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Orders stream
	_, err := n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "ORDERS",
		Description: "Order events stream",
		Subjects:    []string{"orders.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      7 * 24 * time.Hour, // 7 days
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create ORDERS stream: %v", err)
	}

	// Notifications stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "NOTIFICATIONS",
		Description: "Notification events stream",
		Subjects:    []string{"notifications.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      24 * time.Hour, // 1 day
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create NOTIFICATIONS stream: %v", err)
	}

	// Chef events stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "CHEF",
		Description: "Chef events stream",
		Subjects:    []string{"chef.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      7 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create CHEF stream: %v", err)
	}

	// Delivery stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "DELIVERY",
		Description: "Delivery events stream",
		Subjects:    []string{"delivery.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      7 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create DELIVERY stream: %v", err)
	}

	// Payments stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "PAYMENTS",
		Description: "Payment events stream",
		Subjects:    []string{"payments.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      30 * 24 * time.Hour, // 30 days
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create PAYMENTS stream: %v", err)
	}

	// Users stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "USERS",
		Description: "User events stream",
		Subjects:    []string{"users.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      7 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create USERS stream: %v", err)
	}

	// Reviews stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "REVIEWS",
		Description: "Review events stream",
		Subjects:    []string{"reviews.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      7 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create REVIEWS stream: %v", err)
	}

	// Catering stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "CATERING",
		Description: "Catering events stream",
		Subjects:    []string{"catering.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      30 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create CATERING stream: %v", err)
	}

	// Approvals stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "APPROVALS",
		Description: "Approval request lifecycle events",
		Subjects:    []string{"approvals.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      30 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create APPROVALS stream: %v", err)
	}

	// Subscriptions stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "SUBSCRIPTIONS",
		Description: "Subscription billing events stream",
		Subjects:    []string{"subscription.*", "subscription.*.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      30 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create SUBSCRIPTIONS stream: %v", err)
	}

	// Provider delivery stream
	_, err = n.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:        "PROVIDER",
		Description: "Third-party delivery provider events stream",
		Subjects:    []string{"provider.*", "provider.*.*"},
		Retention:   jetstream.WorkQueuePolicy,
		MaxAge:      30 * 24 * time.Hour,
		Storage:     jetstream.FileStorage,
		Replicas:    1,
	})
	if err != nil {
		log.Printf("Failed to create PROVIDER stream: %v", err)
	}

	log.Println("NATS JetStream streams configured")
	return nil
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

// PublishNotification publishes a notification event
func PublishNotification(notif NotificationEvent) error {
	var subject string
	switch notif.Type {
	case "email":
		subject = SubjectNotificationEmail
	case "push":
		subject = SubjectNotificationPush
	case "sms":
		subject = SubjectNotificationSMS
	default:
		subject = SubjectNotificationPush
	}
	return GetNATSClient().Publish(subject, notif)
}

// generateEventID generates a unique event ID
func generateEventID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

// randomString generates a random string of given length
func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
