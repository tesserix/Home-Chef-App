package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
)

// NotificationService consumes domain events from durable JetStream consumers
// and turns them into in-app notifications, emails, pushes and SMS. It binds one
// durable per source stream; the subject decides which handler runs. Handlers
// return an error to trigger retry/dead-letter (see ConsumerManager).
type NotificationService struct {
	nats    *NATSClient
	cm      *ConsumerManager
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
	mu      sync.Mutex
}

var (
	notificationService *NotificationService
	notifOnce           sync.Once
)

// GetNotificationService returns the singleton notification service.
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

// Start registers the notification service's durable consumers on the shared
// ConsumerManager. Streams must already exist (NATSClient.Connect sets them up).
func (s *NotificationService) Start(cm *ConsumerManager) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return nil
	}
	s.cm = cm
	if err := cm.RegisterAll(s.ctx, s.consumerSpecs()...); err != nil {
		return err
	}
	s.running = true
	log.Println("Notification service started (durable consumers)")
	return nil
}

// consumerSpecs declares one durable consumer per source stream. The "notify-*"
// durables are independent of the push-workers (issue #134), so each event is
// processed once per group.
func (s *NotificationService) consumerSpecs() []ConsumerSpec {
	h := s.handleBySubject
	return []ConsumerSpec{
		{Stream: "ORDERS", Durable: "notify-orders", Handler: h,
			Subjects: []string{SubjectOrderCreated, SubjectOrderUpdated, SubjectOrderCancelled, SubjectOrderDelivered}},
		{Stream: "NOTIFICATIONS", Durable: "notify-dispatch", Handler: h,
			Subjects: []string{SubjectNotificationEmail, SubjectNotificationPush, SubjectNotificationSMS}},
		{Stream: "USERS", Durable: "notify-users", Handler: h,
			Subjects: []string{SubjectUserRegistered}},
		{Stream: "CHEF", Durable: "notify-chef", Handler: h,
			Subjects: []string{SubjectChefNewOrder, SubjectChefVerified}},
		{Stream: "DELIVERY", Durable: "notify-delivery", Handler: h,
			Subjects: []string{SubjectDeliveryAssigned, SubjectDeliveryPickedUp, SubjectDriverOnboardingSubmitted}},
		{Stream: "APPROVALS", Durable: "notify-approvals", Handler: h,
			Subjects: []string{SubjectApprovalApproved, SubjectApprovalRejected, SubjectApprovalInfoRequested, SubjectApprovalCreated}},
	}
}

// handleBySubject decodes the event and routes it to the right handler. A decode
// error is surfaced (poison message → retry → dead-letter). The earlier giant
// set of per-subject QueueSubscribe blocks collapses into this single table.
func (s *NotificationService) handleBySubject(_ context.Context, subject string, data []byte) error {
	switch subject {
	case SubjectOrderCreated:
		return decodeThen(data, s.handleOrderCreated)
	case SubjectOrderUpdated:
		return decodeThen(data, s.handleOrderUpdated)
	case SubjectOrderCancelled:
		return decodeThen(data, s.handleOrderCancelled)
	case SubjectOrderDelivered:
		return decodeThen(data, s.handleOrderDelivered)
	case SubjectChefNewOrder:
		return decodeThen(data, s.handleChefNewOrder)
	case SubjectNotificationEmail:
		return decodeThen(data, s.sendEmailNotification)
	case SubjectNotificationPush:
		return decodeThen(data, s.sendPushNotification)
	case SubjectNotificationSMS:
		return decodeThen(data, s.sendSMSNotification)
	case SubjectUserRegistered:
		return decodeThen(data, s.handleUserRegistered)
	case SubjectChefVerified:
		return decodeThen(data, s.handleChefVerified)
	case SubjectDeliveryAssigned:
		return decodeThen(data, s.handleDeliveryAssigned)
	case SubjectDeliveryPickedUp:
		return decodeThen(data, s.handleDeliveryPickedUp)
	case SubjectDriverOnboardingSubmitted:
		return decodeThen(data, s.handleDriverOnboardingSubmitted)
	case SubjectApprovalApproved:
		return decodeThen(data, s.handleApprovalApproved)
	case SubjectApprovalRejected:
		return decodeThen(data, s.handleApprovalRejected)
	case SubjectApprovalInfoRequested:
		return decodeThen(data, s.handleApprovalInfoRequested)
	case SubjectApprovalCreated:
		return decodeThen(data, s.handleApprovalCreated)
	default:
		log.Printf("notification: no handler for subject %q", subject)
		return nil
	}
}

// decodeThen unmarshals data into T then runs fn — the one place JSON decoding
// for event handlers lives.
func decodeThen[T any](data []byte, fn func(T) error) error {
	v, err := decodeEvent[T](data)
	if err != nil {
		return fmt.Errorf("decode event: %w", err)
	}
	return fn(v)
}

// Stop stops the notification service. Consume loops are owned by the shared
// ConsumerManager (stopped by main); this just cancels the service context.
func (s *NotificationService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.cancel()
	s.running = false
	log.Println("Notification service stopped")
}

// ── Event handlers ───────────────────────────────────────────────────────────

func (s *NotificationService) handleDriverOnboardingSubmitted(event Event) error {
	city, _ := event.Data["city"].(string)

	var admins []models.User
	database.DB.Where("role = ?", models.RoleAdmin).Find(&admins)

	data, _ := json.Marshal(event.Data)
	for _, admin := range admins {
		if err := s.saveNotification(&models.Notification{
			UserID:  admin.ID,
			Type:    "driver_onboarding_submitted",
			Title:   "New Driver Application",
			Message: fmt.Sprintf("A new driver from %s has submitted their onboarding application for review.", city),
			Data:    string(data),
		}); err != nil {
			return fmt.Errorf("save driver onboarding notification for admin %s: %w", admin.ID, err)
		}
	}
	return nil
}

func (s *NotificationService) handleOrderCreated(event OrderEvent) error {
	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String(), "total": event.Total})
	if err := s.saveNotification(&models.Notification{
		UserID:  event.ChefID,
		Type:    "order_created",
		Title:   "New Order Received",
		Message: "You have received a new order!",
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save order_created notification: %w", err)
	}

	// Push to chef + emails to customer and chef (best-effort: they are durable
	// events on the NOTIFICATIONS stream, retried independently by notify-dispatch).
	PublishNotification(NotificationEvent{
		UserID: event.ChefID, Type: "push",
		Title: "New Order Received", Message: "You have a new order waiting to be prepared!",
		Data: map[string]any{"order_id": event.OrderID.String()},
	})
	PublishNotification(NotificationEvent{
		UserID: event.CustomerID, Type: "email",
		Title: "Order Confirmed", Message: "Your order has been placed successfully!",
		Data: map[string]any{"type": "order_confirmation", "order_number": event.OrderNumber, "total": event.Total},
	})
	PublishNotification(NotificationEvent{
		UserID: event.ChefID, Type: "email",
		Title: "New Order Received", Message: "You have a new order to prepare!",
		Data: map[string]any{"type": "chef_new_order", "order_number": event.OrderNumber, "total": event.Total},
	})
	return nil
}

func (s *NotificationService) handleOrderUpdated(event OrderEvent) error {
	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String(), "status": event.Status})
	if err := s.saveNotification(&models.Notification{
		UserID:  event.CustomerID,
		Type:    "order_status",
		Title:   "Order Status Updated",
		Message: getOrderStatusMessage(event.Status),
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save order_status notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: event.CustomerID, Type: "push",
		Title: "Order Update", Message: getOrderStatusMessage(event.Status),
		Data: map[string]any{"order_id": event.OrderID.String(), "status": event.Status},
	})
	PublishNotification(NotificationEvent{
		UserID: event.CustomerID, Type: "email",
		Title: "Order Update", Message: getOrderStatusMessage(event.Status),
		Data: map[string]any{"type": "order_status", "order_number": event.OrderNumber, "status": event.Status},
	})
	return nil
}

func (s *NotificationService) handleOrderCancelled(event OrderEvent) error {
	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String()})
	for _, userID := range []uuid.UUID{event.CustomerID, event.ChefID} {
		if err := s.saveNotification(&models.Notification{
			UserID:  userID,
			Type:    "order_cancelled",
			Title:   "Order Cancelled",
			Message: "Order has been cancelled",
			Data:    string(data),
		}); err != nil {
			return fmt.Errorf("save order_cancelled notification for %s: %w", userID, err)
		}
		PublishNotification(NotificationEvent{
			UserID: userID, Type: "push",
			Title: "Order Cancelled", Message: "Order has been cancelled",
			Data: map[string]any{"order_id": event.OrderID.String()},
		})
	}

	PublishNotification(NotificationEvent{
		UserID: event.CustomerID, Type: "email",
		Title: "Order Cancelled", Message: "Your order has been cancelled",
		Data: map[string]any{"type": "order_status", "order_number": event.OrderNumber, "status": "cancelled"},
	})
	return nil
}

func (s *NotificationService) handleOrderDelivered(event OrderEvent) error {
	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String()})
	if err := s.saveNotification(&models.Notification{
		UserID:  event.CustomerID,
		Type:    "order_delivered",
		Title:   "Order Delivered",
		Message: "Your order has been delivered! Enjoy your meal!",
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save order_delivered notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: event.CustomerID, Type: "push",
		Title: "Order Delivered", Message: "Your order has been delivered! Enjoy your meal!",
		Data: map[string]any{"order_id": event.OrderID.String()},
	})

	// Wave 3: auto-email the GST tax invoice PDF to the customer shortly after
	// delivery. Kept off the worker hot path (PDF render is ~hundreds of ms).
	go emailDeliveredInvoice(event)
	return nil
}

// emailDeliveredInvoice resolves the customer's email and dispatches the PDF
// invoice. Best-effort: the in-app + push confirmations above are authoritative.
func emailDeliveredInvoice(event OrderEvent) {
	var customer models.User
	if err := database.DB.First(&customer, "id = ?", event.CustomerID).Error; err != nil {
		log.Printf("invoice-email skipped: customer %s lookup failed: %v", event.CustomerID, err)
		return
	}
	if customer.Email == "" {
		log.Printf("invoice-email skipped: customer %s has no email", event.CustomerID)
		return
	}
	pdfBytes, filename, err := GenerateOrderInvoicePDF(event.OrderID)
	if err != nil {
		log.Printf("invoice-email skipped: PDF generation failed for %s: %v", event.OrderNumber, err)
		return
	}
	if err := GetEmailService().SendOrderInvoice(customer.Email, customer.FirstName, event.OrderNumber, pdfBytes, filename); err != nil {
		log.Printf("invoice-email failed for %s: %v", event.OrderNumber, err)
	}
}

func (s *NotificationService) handleUserRegistered(event Event) error {
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "email",
		Title:   "Welcome to HomeChef!",
		Message: "Thank you for joining HomeChef. Discover amazing home-cooked meals near you!",
		Data:    event.Data,
	})
	return nil
}

func (s *NotificationService) handleChefNewOrder(event OrderEvent) error {
	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String(), "total": event.Total})
	if err := s.saveNotification(&models.Notification{
		UserID:  event.ChefID,
		Type:    "new_order",
		Title:   "New Order!",
		Message: "You have a new order to prepare",
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save new_order notification: %w", err)
	}
	return nil
}

func (s *NotificationService) handleChefVerified(event Event) error {
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "chef_verified",
		Title:   "Congratulations!",
		Message: "Your chef profile has been verified. You can now start accepting orders!",
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save chef_verified notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "email",
		Title:   "Your Chef Profile is Verified!",
		Message: "Congratulations! Your chef profile has been verified. You can now start accepting orders!",
	})
	return nil
}

func (s *NotificationService) handleDeliveryAssigned(event Event) error {
	customerIDStr, ok := event.Data["customer_id"].(string)
	if !ok {
		return nil // no customer to notify
	}
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return fmt.Errorf("parse customer_id: %w", err)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  customerID,
		Type:    "delivery_assigned",
		Title:   "Delivery Partner Assigned",
		Message: "A delivery partner has been assigned to your order",
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save delivery_assigned notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: customerID, Type: "push",
		Title:   "Delivery Partner Assigned",
		Message: "A delivery partner has been assigned to your order and will pick it up soon!",
		Data:    event.Data,
	})
	PublishNotification(NotificationEvent{
		UserID: customerID, Type: "email",
		Title:   "Delivery Partner Assigned",
		Message: "A delivery partner has been assigned to your order",
		Data: map[string]any{
			"type":        "delivery_assigned",
			"driver_name": event.Data["driver_name"],
			"order_id":    event.Data["order_id"],
		},
	})
	return nil
}

func (s *NotificationService) handleDeliveryPickedUp(event Event) error {
	customerIDStr, ok := event.Data["customer_id"].(string)
	if !ok {
		return nil
	}
	customerID, err := uuid.Parse(customerIDStr)
	if err != nil {
		return fmt.Errorf("parse customer_id: %w", err)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  customerID,
		Type:    "delivery_picked_up",
		Title:   "Order Picked Up",
		Message: "Your order has been picked up and is on its way!",
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save delivery_picked_up notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: customerID, Type: "push",
		Title:   "Order On The Way!",
		Message: "Your order has been picked up and is on its way to you!",
		Data:    event.Data,
	})
	return nil
}

// resolveChefUserID resolves a chef's UserID from a ChefProfile.ID, falling back
// to the approval request's submitter when the chef row is gone.
func (s *NotificationService) resolveChefUserID(chefIDStr string, eventData map[string]any) (uuid.UUID, error) {
	chefID, err := uuid.Parse(chefIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid chef_id: %v", err)
	}
	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err == nil {
		return chef.UserID, nil
	}
	if approvalIDStr, ok := eventData["approval_id"].(string); ok {
		approvalID, _ := uuid.Parse(approvalIDStr)
		var approval models.ApprovalRequest
		if err := database.DB.First(&approval, "id = ?", approvalID).Error; err == nil {
			return approval.SubmittedByID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("could not resolve user for chef_id=%s", chefIDStr)
}

// resolveApprovalUserID resolves the target user for an approval event, handling
// both chef and driver/partner approvals.
func (s *NotificationService) resolveApprovalUserID(event Event) (uuid.UUID, error) {
	if partnerIDStr, ok := event.Data["partner_id"].(string); ok && partnerIDStr != "" {
		if partnerID, err := uuid.Parse(partnerIDStr); err == nil {
			var partner models.DeliveryPartner
			if err := database.DB.First(&partner, "id = ?", partnerID).Error; err == nil {
				return partner.UserID, nil
			}
		}
	}
	if chefIDStr, ok := event.Data["chef_id"].(string); ok && chefIDStr != "" {
		if userID, err := s.resolveChefUserID(chefIDStr, event.Data); err == nil {
			return userID, nil
		}
	}
	if approvalIDStr, ok := event.Data["approval_id"].(string); ok {
		approvalID, _ := uuid.Parse(approvalIDStr)
		var approval models.ApprovalRequest
		if err := database.DB.First(&approval, "id = ?", approvalID).Error; err == nil {
			return approval.SubmittedByID, nil
		}
	}
	return uuid.Nil, fmt.Errorf("could not resolve user for approval event")
}

func (s *NotificationService) handleApprovalApproved(event Event) error {
	approvalType, _ := event.Data["type"].(string)
	title, _ := event.Data["title"].(string)

	userID, err := s.resolveApprovalUserID(event)
	if err != nil {
		log.Printf("approval approved: unresolved user (dropping): %v", err)
		return nil // unresolvable target is not retryable
	}

	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  userID,
		Type:    "approval_approved",
		Title:   "Request Approved",
		Message: fmt.Sprintf("Your %s has been approved: %s", approvalType, title),
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save approval_approved notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: userID, Type: "push",
		Title: "Request Approved!", Message: fmt.Sprintf("Your %s has been approved", approvalType),
		Data: event.Data,
	})
	return nil
}

func (s *NotificationService) handleApprovalRejected(event Event) error {
	approvalType, _ := event.Data["type"].(string)
	notes, _ := event.Data["notes"].(string)

	userID, err := s.resolveApprovalUserID(event)
	if err != nil {
		log.Printf("approval rejected: unresolved user (dropping): %v", err)
		return nil
	}

	message := fmt.Sprintf("Your %s has been rejected.", approvalType)
	if notes != "" {
		message += fmt.Sprintf(" Notes: %s", notes)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  userID,
		Type:    "approval_rejected",
		Title:   "Request Rejected",
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save approval_rejected notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: userID, Type: "push",
		Title: "Request Rejected", Message: message, Data: event.Data,
	})
	return nil
}

func (s *NotificationService) handleApprovalInfoRequested(event Event) error {
	approvalType, _ := event.Data["type"].(string)
	notes, _ := event.Data["notes"].(string)

	userID, err := s.resolveApprovalUserID(event)
	if err != nil {
		log.Printf("approval info_requested: unresolved user (dropping): %v", err)
		return nil
	}

	message := fmt.Sprintf("Admin needs more info about your %s.", approvalType)
	if notes != "" {
		message += fmt.Sprintf(" Notes: %s", notes)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  userID,
		Type:    "approval_info_requested",
		Title:   "More Information Needed",
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save approval_info_requested notification: %w", err)
	}

	PublishNotification(NotificationEvent{
		UserID: userID, Type: "push",
		Title: "More Information Needed", Message: message, Data: event.Data,
	})
	return nil
}

func (s *NotificationService) handleApprovalCreated(event Event) error {
	title, _ := event.Data["title"].(string)

	var admins []models.User
	database.DB.Where("role = ?", models.RoleAdmin).Find(&admins)

	data, _ := json.Marshal(event.Data)
	for _, admin := range admins {
		if err := s.saveNotification(&models.Notification{
			UserID:  admin.ID,
			Type:    "approval_created",
			Title:   "New Approval Request",
			Message: fmt.Sprintf("New approval request pending: %s", title),
			Data:    string(data),
		}); err != nil {
			return fmt.Errorf("save approval_created notification for admin %s: %w", admin.ID, err)
		}
	}
	return nil
}

// ── Notification dispatch (NOTIFICATIONS stream) ─────────────────────────────

func (s *NotificationService) sendEmailNotification(notif NotificationEvent) error {
	notifType, _ := notif.Data["type"].(string)
	if !ShouldSendForType(notif.UserID, notifType, ChannelEmail) {
		log.Printf("Email dispatch skipped for user %s (category=%s opted-out)", notif.UserID, notificationTypeCategory(notifType))
		return nil
	}

	var user models.User
	if err := database.DB.Select("id, email, first_name").First(&user, "id = ?", notif.UserID).Error; err != nil {
		log.Printf("Email dispatch: user %s not found (dropping): %v", notif.UserID, err)
		return nil // missing user is not retryable
	}

	emailSvc := GetEmailService()
	switch notifType {
	case "order_confirmation":
		orderNumber, _ := notif.Data["order_number"].(string)
		total, _ := notif.Data["total"].(float64)
		// TODO(CW-01e-backend): populate OrderInvoiceDetails (GST breakup, HSN/SAC,
		// chef name + FSSAI, address, ETA) so the confirmation email satisfies
		// CGST Act 2017 §31 + Rule 46. nil keeps the legacy minimal layout.
		return emailSvc.SendOrderConfirmation(user.Email, orderNumber, nil, total, nil)
	case "order_status":
		orderNumber, _ := notif.Data["order_number"].(string)
		status, _ := notif.Data["status"].(string)
		return emailSvc.SendOrderStatusUpdate(user.Email, orderNumber, status)
	case "chef_new_order":
		orderNumber, _ := notif.Data["order_number"].(string)
		total, _ := notif.Data["total"].(float64)
		return emailSvc.SendChefNewOrder(user.Email, orderNumber, total)
	case "delivery_assigned":
		orderID, _ := notif.Data["order_id"].(string)
		return emailSvc.SendDeliveryAssigned(user.Email, orderID, "")
	case "chef_verified":
		return emailSvc.SendChefVerificationApproved(user.Email, user.FirstName)
	case "welcome":
		return emailSvc.SendWelcomeEmail(user.Email, user.FirstName)
	default:
		html := fmt.Sprintf("<h2>%s</h2><p>%s</p>", notif.Title, notif.Message)
		return emailSvc.send(user.Email, notif.Title, html)
	}
}

func (s *NotificationService) sendPushNotification(notif NotificationEvent) error {
	notifType, _ := notif.Data["type"].(string)
	if !ShouldSendForType(notif.UserID, notifType, ChannelPush) {
		log.Printf("Push dispatch skipped for user %s (category=%s opted-out)", notif.UserID, notificationTypeCategory(notifType))
		return nil
	}

	data := make(map[string]string, len(notif.Data))
	for k, v := range notif.Data {
		data[k] = fmt.Sprintf("%v", v)
	}
	return SendPushNotification(notif.UserID, notif.Title, notif.Message, data)
}

func (s *NotificationService) sendSMSNotification(notif NotificationEvent) error {
	cfg := config.AppConfig
	if cfg.TwilioAccountSID == "" || cfg.TwilioAuthToken == "" || cfg.TwilioPhoneNumber == "" {
		log.Printf("SMS skipped (Twilio not configured): user=%s", notif.UserID)
		return nil
	}

	var user models.User
	if err := database.DB.Select("id, phone").First(&user, "id = ?", notif.UserID).Error; err != nil {
		log.Printf("SMS dispatch: user %s not found (dropping): %v", notif.UserID, err)
		return nil
	}
	if user.Phone == "" {
		log.Printf("SMS skipped: user %s has no phone number", notif.UserID)
		return nil
	}

	twilioURL := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", cfg.TwilioAccountSID)
	formData := fmt.Sprintf("From=%s&To=%s&Body=%s",
		url.QueryEscape(cfg.TwilioPhoneNumber),
		url.QueryEscape(user.Phone),
		url.QueryEscape(fmt.Sprintf("%s: %s", notif.Title, notif.Message)),
	)

	req, err := http.NewRequest(http.MethodPost, twilioURL, bytes.NewBufferString(formData))
	if err != nil {
		return fmt.Errorf("sms build request: %w", err)
	}
	req.SetBasicAuth(cfg.TwilioAccountSID, cfg.TwilioAuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("sms request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("sms: Twilio returned status %d for user %s", resp.StatusCode, notif.UserID)
	}
	log.Printf("SMS sent to user %s", notif.UserID)
	return nil
}

// saveNotification persists a notification and publishes it to the per-user NATS
// subject so WebSocket clients update the bell in real time (fire-and-forget —
// the durable record is the row, the WS push is a best-effort nicety).
func (s *NotificationService) saveNotification(notification *models.Notification) error {
	notification.CreatedAt = time.Now()
	if err := database.DB.Create(notification).Error; err != nil {
		return err
	}

	subject := fmt.Sprintf("%s.%s", SubjectNotificationUser, notification.UserID.String())
	payload := map[string]any{
		"id":        notification.ID.String(),
		"type":      notification.Type,
		"title":     notification.Title,
		"message":   notification.Message,
		"data":      notification.Data,
		"isRead":    notification.IsRead,
		"createdAt": notification.CreatedAt.Format(time.RFC3339),
	}
	if err := s.nats.Publish(subject, payload); err != nil {
		log.Printf("Failed to publish real-time notification for user %s: %v", notification.UserID, err)
	}
	return nil
}

// getOrderStatusMessage returns a customer-friendly status message.
func getOrderStatusMessage(status string) string {
	messages := map[string]string{
		"confirmed":  "Your order has been confirmed by the chef!",
		"preparing":  "Your order is being prepared",
		"ready":      "Your order is ready for pickup/delivery",
		"picked_up":  "Your order has been picked up by the delivery partner",
		"on_the_way": "Your order is on its way!",
		"delivered":  "Your order has been delivered. Enjoy!",
		"cancelled":  "Your order has been cancelled",
	}
	if msg, ok := messages[status]; ok {
		return msg
	}
	return "Your order status has been updated to: " + status
}
