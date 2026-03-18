package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// NotificationService handles notification processing
type NotificationService struct {
	nats         *NATSClient
	subscriptions []*nats.Subscription
	consumers    []jetstream.Consumer
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	running      bool
	mu           sync.Mutex
}

var (
	notificationService *NotificationService
	notifOnce           sync.Once
)

// GetNotificationService returns the singleton notification service
func GetNotificationService() *NotificationService {
	notifOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		notificationService = &NotificationService{
			nats:   GetNATSClient(),
			ctx:    ctx,
			cancel: cancel,
		}
	})
	return notificationService
}

// Start starts the notification service and subscribes to events
func (s *NotificationService) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	log.Println("Starting notification service...")

	// Subscribe to order events
	if err := s.subscribeToOrders(); err != nil {
		log.Printf("Warning: Failed to subscribe to orders: %v", err)
	}

	// Subscribe to notification events
	if err := s.subscribeToNotifications(); err != nil {
		log.Printf("Warning: Failed to subscribe to notifications: %v", err)
	}

	// Subscribe to user events
	if err := s.subscribeToUserEvents(); err != nil {
		log.Printf("Warning: Failed to subscribe to user events: %v", err)
	}

	// Subscribe to chef events
	if err := s.subscribeToChefEvents(); err != nil {
		log.Printf("Warning: Failed to subscribe to chef events: %v", err)
	}

	// Subscribe to delivery events
	if err := s.subscribeToDeliveryEvents(); err != nil {
		log.Printf("Warning: Failed to subscribe to delivery events: %v", err)
	}

	// Subscribe to approval events
	if err := s.subscribeToApprovalEvents(); err != nil {
		log.Printf("Warning: Failed to subscribe to approval events: %v", err)
	}

	s.running = true
	log.Println("Notification service started successfully")
	return nil
}

// subscribeToOrders subscribes to order-related events
func (s *NotificationService) subscribeToOrders() error {
	// Order created - notify chef
	sub, err := s.nats.QueueSubscribe(SubjectOrderCreated, "notification-workers", func(msg *nats.Msg) {
		var event OrderEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal order created event: %v", err)
			return
		}
		s.handleOrderCreated(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Order updated - notify customer
	sub, err = s.nats.QueueSubscribe(SubjectOrderUpdated, "notification-workers", func(msg *nats.Msg) {
		var event OrderEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal order updated event: %v", err)
			return
		}
		s.handleOrderUpdated(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Order cancelled
	sub, err = s.nats.QueueSubscribe(SubjectOrderCancelled, "notification-workers", func(msg *nats.Msg) {
		var event OrderEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal order cancelled event: %v", err)
			return
		}
		s.handleOrderCancelled(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Order delivered
	sub, err = s.nats.QueueSubscribe(SubjectOrderDelivered, "notification-workers", func(msg *nats.Msg) {
		var event OrderEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal order delivered event: %v", err)
			return
		}
		s.handleOrderDelivered(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	return nil
}

// subscribeToNotifications subscribes to notification dispatch events
func (s *NotificationService) subscribeToNotifications() error {
	// Email notifications
	sub, err := s.nats.QueueSubscribe(SubjectNotificationEmail, "notification-workers", func(msg *nats.Msg) {
		var notif NotificationEvent
		if err := json.Unmarshal(msg.Data, &notif); err != nil {
			log.Printf("Failed to unmarshal email notification: %v", err)
			return
		}
		s.sendEmailNotification(notif)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Push notifications
	sub, err = s.nats.QueueSubscribe(SubjectNotificationPush, "notification-workers", func(msg *nats.Msg) {
		var notif NotificationEvent
		if err := json.Unmarshal(msg.Data, &notif); err != nil {
			log.Printf("Failed to unmarshal push notification: %v", err)
			return
		}
		s.sendPushNotification(notif)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// SMS notifications
	sub, err = s.nats.QueueSubscribe(SubjectNotificationSMS, "notification-workers", func(msg *nats.Msg) {
		var notif NotificationEvent
		if err := json.Unmarshal(msg.Data, &notif); err != nil {
			log.Printf("Failed to unmarshal SMS notification: %v", err)
			return
		}
		s.sendSMSNotification(notif)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	return nil
}

// subscribeToUserEvents subscribes to user-related events
func (s *NotificationService) subscribeToUserEvents() error {
	sub, err := s.nats.QueueSubscribe(SubjectUserRegistered, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal user registered event: %v", err)
			return
		}
		s.handleUserRegistered(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)
	return nil
}

// subscribeToChefEvents subscribes to chef-related events
func (s *NotificationService) subscribeToChefEvents() error {
	// New order for chef
	sub, err := s.nats.QueueSubscribe(SubjectChefNewOrder, "notification-workers", func(msg *nats.Msg) {
		var event OrderEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal chef new order event: %v", err)
			return
		}
		s.handleChefNewOrder(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Chef verified
	sub, err = s.nats.QueueSubscribe(SubjectChefVerified, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal chef verified event: %v", err)
			return
		}
		s.handleChefVerified(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	return nil
}

// subscribeToDeliveryEvents subscribes to delivery-related events
func (s *NotificationService) subscribeToDeliveryEvents() error {
	// Delivery assigned
	sub, err := s.nats.QueueSubscribe(SubjectDeliveryAssigned, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal delivery assigned event: %v", err)
			return
		}
		s.handleDeliveryAssigned(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Delivery picked up
	sub, err = s.nats.QueueSubscribe(SubjectDeliveryPickedUp, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal delivery picked up event: %v", err)
			return
		}
		s.handleDeliveryPickedUp(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Driver onboarding submitted - notify admins
	sub, err = s.nats.QueueSubscribe(SubjectDriverOnboardingSubmitted, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal driver onboarding submitted event: %v", err)
			return
		}
		s.handleDriverOnboardingSubmitted(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	return nil
}

// Event handlers

func (s *NotificationService) handleDriverOnboardingSubmitted(event Event) {
	log.Printf("Processing driver onboarding submitted event: %s", event.ID)

	city, _ := event.Data["city"].(string)

	// Notify all admin users about new driver application
	var admins []models.User
	database.DB.Where("role = ?", models.RoleAdmin).Find(&admins)

	data, _ := json.Marshal(event.Data)
	for _, admin := range admins {
		notification := &models.Notification{
			UserID:  admin.ID,
			Type:    "driver_onboarding_submitted",
			Title:   "New Driver Application",
			Message: fmt.Sprintf("A new driver from %s has submitted their onboarding application for review.", city),
			Data:    string(data),
		}
		if err := s.saveNotification(notification); err != nil {
			log.Printf("Failed to save driver onboarding notification for admin %s: %v", admin.ID, err)
		}
	}
}

func (s *NotificationService) handleOrderCreated(event OrderEvent) {
	log.Printf("Processing order created event: Order #%s", event.OrderID.String())

	// Create notification record in database
	data, _ := json.Marshal(map[string]interface{}{"order_id": event.OrderID.String(), "total": event.Total})
	notification := &models.Notification{
		UserID:  event.ChefID,
		Type:    "order_created",
		Title:   "New Order Received",
		Message: "You have received a new order!",
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save notification: %v", err)
	}

	// Send push notification to chef
	PublishNotification(NotificationEvent{
		UserID:  event.ChefID,
		Type:    "push",
		Title:   "New Order Received",
		Message: "You have a new order waiting to be prepared!",
		Data:    map[string]interface{}{"order_id": event.OrderID.String()},
	})
}

func (s *NotificationService) handleOrderUpdated(event OrderEvent) {
	log.Printf("Processing order updated event: Order #%s -> %s", event.OrderID.String(), event.Status)

	// Notify customer about order status change
	data, _ := json.Marshal(map[string]interface{}{"order_id": event.OrderID.String(), "status": event.Status})
	notification := &models.Notification{
		UserID:  event.CustomerID,
		Type:    "order_status",
		Title:   "Order Status Updated",
		Message: getOrderStatusMessage(event.Status),
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save notification: %v", err)
	}

	// Send push notification
	PublishNotification(NotificationEvent{
		UserID:  event.CustomerID,
		Type:    "push",
		Title:   "Order Update",
		Message: getOrderStatusMessage(event.Status),
		Data:    map[string]interface{}{"order_id": event.OrderID.String(), "status": event.Status},
	})
}

func (s *NotificationService) handleOrderCancelled(event OrderEvent) {
	log.Printf("Processing order cancelled event: Order #%s", event.OrderID.String())

	// Notify both customer and chef
	data, _ := json.Marshal(map[string]interface{}{"order_id": event.OrderID.String()})
	for _, userID := range []uuid.UUID{event.CustomerID, event.ChefID} {
		notification := &models.Notification{
			UserID:  userID,
			Type:    "order_cancelled",
			Title:   "Order Cancelled",
			Message: "Order has been cancelled",
			Data:    string(data),
		}
		if err := s.saveNotification(notification); err != nil {
			log.Printf("Failed to save notification: %v", err)
		}
	}
}

func (s *NotificationService) handleOrderDelivered(event OrderEvent) {
	log.Printf("Processing order delivered event: Order #%s", event.OrderID.String())

	// Notify customer
	data, _ := json.Marshal(map[string]interface{}{"order_id": event.OrderID.String()})
	notification := &models.Notification{
		UserID:  event.CustomerID,
		Type:    "order_delivered",
		Title:   "Order Delivered",
		Message: "Your order has been delivered! Enjoy your meal!",
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save notification: %v", err)
	}

	// Send push notification
	PublishNotification(NotificationEvent{
		UserID:  event.CustomerID,
		Type:    "push",
		Title:   "Order Delivered",
		Message: "Your order has been delivered! Enjoy your meal!",
		Data:    map[string]interface{}{"order_id": event.OrderID.String()},
	})
}

func (s *NotificationService) handleUserRegistered(event Event) {
	log.Printf("Processing user registered event: User #%s", event.UserID.String())

	// Send welcome email
	PublishNotification(NotificationEvent{
		UserID:  event.UserID,
		Type:    "email",
		Title:   "Welcome to HomeChef!",
		Message: "Thank you for joining HomeChef. Discover amazing home-cooked meals near you!",
		Data:    event.Data,
	})
}

func (s *NotificationService) handleChefNewOrder(event OrderEvent) {
	log.Printf("Processing chef new order event: Chef #%s, Order #%s", event.ChefID.String(), event.OrderID.String())

	data, _ := json.Marshal(map[string]interface{}{"order_id": event.OrderID.String(), "total": event.Total})
	notification := &models.Notification{
		UserID:  event.ChefID,
		Type:    "new_order",
		Title:   "New Order!",
		Message: "You have a new order to prepare",
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save notification: %v", err)
	}
}

func (s *NotificationService) handleChefVerified(event Event) {
	log.Printf("Processing chef verified event: User #%s", event.UserID.String())

	data, _ := json.Marshal(event.Data)
	notification := &models.Notification{
		UserID:  event.UserID,
		Type:    "chef_verified",
		Title:   "Congratulations!",
		Message: "Your chef profile has been verified. You can now start accepting orders!",
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save notification: %v", err)
	}

	// Send email
	PublishNotification(NotificationEvent{
		UserID:  event.UserID,
		Type:    "email",
		Title:   "Your Chef Profile is Verified!",
		Message: "Congratulations! Your chef profile has been verified. You can now start accepting orders!",
	})
}

func (s *NotificationService) handleDeliveryAssigned(event Event) {
	log.Printf("Processing delivery assigned event")

	if customerIDStr, ok := event.Data["customer_id"].(string); ok {
		customerID, err := uuid.Parse(customerIDStr)
		if err != nil {
			log.Printf("Failed to parse customer_id: %v", err)
			return
		}
		data, _ := json.Marshal(event.Data)
		notification := &models.Notification{
			UserID:  customerID,
			Type:    "delivery_assigned",
			Title:   "Delivery Partner Assigned",
			Message: "A delivery partner has been assigned to your order",
			Data:    string(data),
		}
		if err := s.saveNotification(notification); err != nil {
			log.Printf("Failed to save notification: %v", err)
		}
	}
}

func (s *NotificationService) handleDeliveryPickedUp(event Event) {
	log.Printf("Processing delivery picked up event")

	if customerIDStr, ok := event.Data["customer_id"].(string); ok {
		customerID, err := uuid.Parse(customerIDStr)
		if err != nil {
			log.Printf("Failed to parse customer_id: %v", err)
			return
		}
		data, _ := json.Marshal(event.Data)
		notification := &models.Notification{
			UserID:  customerID,
			Type:    "delivery_picked_up",
			Title:   "Order Picked Up",
			Message: "Your order has been picked up and is on its way!",
			Data:    string(data),
		}
		if err := s.saveNotification(notification); err != nil {
			log.Printf("Failed to save notification: %v", err)
		}

		// Send push notification
		PublishNotification(NotificationEvent{
			UserID:  customerID,
			Type:    "push",
			Title:   "Order On The Way!",
			Message: "Your order has been picked up and is on its way to you!",
			Data:    event.Data,
		})
	}
}

// subscribeToApprovalEvents subscribes to approval lifecycle events
func (s *NotificationService) subscribeToApprovalEvents() error {
	// Approval approved - notify the chef
	sub, err := s.nats.QueueSubscribe(SubjectApprovalApproved, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal approval approved event: %v", err)
			return
		}
		s.handleApprovalApproved(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Approval rejected - notify the chef
	sub, err = s.nats.QueueSubscribe(SubjectApprovalRejected, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal approval rejected event: %v", err)
			return
		}
		s.handleApprovalRejected(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Approval info requested - notify the chef
	sub, err = s.nats.QueueSubscribe(SubjectApprovalInfoRequested, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal approval info_requested event: %v", err)
			return
		}
		s.handleApprovalInfoRequested(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	// Approval created - notify all admins
	sub, err = s.nats.QueueSubscribe(SubjectApprovalCreated, "notification-workers", func(msg *nats.Msg) {
		var event Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("Failed to unmarshal approval created event: %v", err)
			return
		}
		s.handleApprovalCreated(event)
	})
	if err != nil {
		return err
	}
	s.subscriptions = append(s.subscriptions, sub)

	return nil
}

// resolveChefUserID resolves a chef's UserID from a ChefProfile.ID.
// Falls back to looking up the approval request's SubmittedByID if chef not found.
func (s *NotificationService) resolveChefUserID(chefIDStr string, eventData map[string]interface{}) (uuid.UUID, error) {
	chefID, err := uuid.Parse(chefIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid chef_id: %v", err)
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err == nil {
		return chef.UserID, nil
	}
	// Fallback: look up the approval request to find the submitter
	if approvalIDStr, ok := eventData["approval_id"].(string); ok {
		approvalID, _ := uuid.Parse(approvalIDStr)
		var approval models.ApprovalRequest
		if err := database.DB.First(&approval, "id = ?", approvalID).Error; err == nil {
			return approval.SubmittedByID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("could not resolve user for chef_id=%s", chefIDStr)
}

// resolveApprovalUserID resolves the user ID for an approval event.
// Handles both chef-based and driver/partner-based approvals.
func (s *NotificationService) resolveApprovalUserID(event Event) (uuid.UUID, error) {
	// Try partner_id first (driver approvals)
	if partnerIDStr, ok := event.Data["partner_id"].(string); ok && partnerIDStr != "" {
		partnerID, err := uuid.Parse(partnerIDStr)
		if err == nil {
			var partner models.DeliveryPartner
			if err := database.DB.First(&partner, "id = ?", partnerID).Error; err == nil {
				return partner.UserID, nil
			}
		}
	}

	// Try chef_id (chef approvals)
	if chefIDStr, ok := event.Data["chef_id"].(string); ok && chefIDStr != "" {
		userID, err := s.resolveChefUserID(chefIDStr, event.Data)
		if err == nil {
			return userID, nil
		}
	}

	// Fallback: look up the approval request to find the submitter
	if approvalIDStr, ok := event.Data["approval_id"].(string); ok {
		approvalID, _ := uuid.Parse(approvalIDStr)
		var approval models.ApprovalRequest
		if err := database.DB.First(&approval, "id = ?", approvalID).Error; err == nil {
			return approval.SubmittedByID, nil
		}
	}

	return uuid.Nil, fmt.Errorf("could not resolve user for approval event")
}

func (s *NotificationService) handleApprovalApproved(event Event) {
	log.Printf("Processing approval approved event: %s", event.ID)

	approvalType, _ := event.Data["type"].(string)
	title, _ := event.Data["title"].(string)

	userID, err := s.resolveApprovalUserID(event)
	if err != nil {
		log.Printf("Failed to resolve user for approval approved: %v", err)
		return
	}

	data, _ := json.Marshal(event.Data)
	notification := &models.Notification{
		UserID:  userID,
		Type:    "approval_approved",
		Title:   "Request Approved",
		Message: fmt.Sprintf("Your %s has been approved: %s", approvalType, title),
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save approval approved notification: %v", err)
	}

	// Send push notification
	PublishNotification(NotificationEvent{
		UserID:  userID,
		Type:    "push",
		Title:   "Request Approved!",
		Message: fmt.Sprintf("Your %s has been approved", approvalType),
		Data:    event.Data,
	})
}

func (s *NotificationService) handleApprovalRejected(event Event) {
	log.Printf("Processing approval rejected event: %s", event.ID)

	approvalType, _ := event.Data["type"].(string)
	notes, _ := event.Data["notes"].(string)

	userID, err := s.resolveApprovalUserID(event)
	if err != nil {
		log.Printf("Failed to resolve user for approval rejected: %v", err)
		return
	}

	message := fmt.Sprintf("Your %s has been rejected.", approvalType)
	if notes != "" {
		message += fmt.Sprintf(" Notes: %s", notes)
	}

	data, _ := json.Marshal(event.Data)
	notification := &models.Notification{
		UserID:  userID,
		Type:    "approval_rejected",
		Title:   "Request Rejected",
		Message: message,
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save approval rejected notification: %v", err)
	}

	PublishNotification(NotificationEvent{
		UserID:  userID,
		Type:    "push",
		Title:   "Request Rejected",
		Message: message,
		Data:    event.Data,
	})
}

func (s *NotificationService) handleApprovalInfoRequested(event Event) {
	log.Printf("Processing approval info_requested event: %s", event.ID)

	approvalType, _ := event.Data["type"].(string)
	notes, _ := event.Data["notes"].(string)

	userID, err := s.resolveApprovalUserID(event)
	if err != nil {
		log.Printf("Failed to resolve user for approval info_requested: %v", err)
		return
	}

	message := fmt.Sprintf("Admin needs more info about your %s.", approvalType)
	if notes != "" {
		message += fmt.Sprintf(" Notes: %s", notes)
	}

	data, _ := json.Marshal(event.Data)
	notification := &models.Notification{
		UserID:  userID,
		Type:    "approval_info_requested",
		Title:   "More Information Needed",
		Message: message,
		Data:    string(data),
	}
	if err := s.saveNotification(notification); err != nil {
		log.Printf("Failed to save approval info_requested notification: %v", err)
	}

	PublishNotification(NotificationEvent{
		UserID:  userID,
		Type:    "push",
		Title:   "More Information Needed",
		Message: message,
		Data:    event.Data,
	})
}

func (s *NotificationService) handleApprovalCreated(event Event) {
	log.Printf("Processing approval created event: %s", event.ID)

	title, _ := event.Data["title"].(string)

	// Notify all admin users
	var admins []models.User
	database.DB.Where("role = ?", models.RoleAdmin).Find(&admins)

	data, _ := json.Marshal(event.Data)
	for _, admin := range admins {
		notification := &models.Notification{
			UserID:  admin.ID,
			Type:    "approval_created",
			Title:   "New Approval Request",
			Message: fmt.Sprintf("New approval request pending: %s", title),
			Data:    string(data),
		}
		if err := s.saveNotification(notification); err != nil {
			log.Printf("Failed to save approval created notification for admin %s: %v", admin.ID, err)
		}
	}
}

// Notification dispatch methods

func (s *NotificationService) sendEmailNotification(notif NotificationEvent) {
	log.Printf("Sending email notification to user %s: %s", notif.UserID.String(), notif.Title)
	// TODO: Implement actual email sending via SendGrid
	// For now, just log it
}

func (s *NotificationService) sendPushNotification(notif NotificationEvent) {
	log.Printf("Sending push notification to user %s: %s", notif.UserID.String(), notif.Title)
	// TODO: Implement actual push notification via FCM/APNS
	// For now, just log it
}

func (s *NotificationService) sendSMSNotification(notif NotificationEvent) {
	log.Printf("Sending SMS notification to user #%s: %s", notif.UserID.String(), notif.Title)
	// TODO: Implement actual SMS sending via Twilio
	// For now, just log it
}

// saveNotification saves a notification to the database
func (s *NotificationService) saveNotification(notification *models.Notification) error {
	notification.CreatedAt = time.Now()
	return database.DB.Create(notification).Error
}

// Helper functions

func getOrderStatusMessage(status string) string {
	messages := map[string]string{
		"confirmed":   "Your order has been confirmed by the chef!",
		"preparing":   "Your order is being prepared",
		"ready":       "Your order is ready for pickup/delivery",
		"picked_up":   "Your order has been picked up by the delivery partner",
		"on_the_way":  "Your order is on its way!",
		"delivered":   "Your order has been delivered. Enjoy!",
		"cancelled":   "Your order has been cancelled",
	}
	if msg, ok := messages[status]; ok {
		return msg
	}
	return "Your order status has been updated to: " + status
}

// Stop stops the notification service
func (s *NotificationService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	log.Println("Stopping notification service...")
	s.cancel()

	// Unsubscribe from all subscriptions
	for _, sub := range s.subscriptions {
		sub.Unsubscribe()
	}
	s.subscriptions = nil

	s.wg.Wait()
	s.running = false
	log.Println("Notification service stopped")
}
