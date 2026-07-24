package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
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
		// Auto-void (#694): the customer's apology + refund confirmation, and the
		// chef's pre-close nudge. Own durable so this flow is independent of the
		// order-lifecycle notifications above — a backlog on one must not stall the
		// other, and the apology in particular must always get through.
		{Stream: "ORDERS", Durable: "notify-order-void", Handler: h,
			Subjects: []string{SubjectOrderVoided, SubjectOrderAcceptReminder}},
		// Order issue reported → notify the chef (#37). Own durable so the
		// issue fan-out is independent of the order-lifecycle notifications.
		{Stream: "ORDERS", Durable: "notify-order-issue", Handler: h,
			Subjects: []string{SubjectOrderIssueReported}},
		// Cancellation arbitration (#475): notify the chef of a request, the
		// customer of the refund. Own durable, independent of order lifecycle.
		{Stream: "ORDERS", Durable: "notify-cancellation", Handler: h,
			Subjects: []string{SubjectCancellationRequested, SubjectCancellationResolved}},
		{Stream: "NOTIFICATIONS", Durable: "notify-dispatch", Handler: h,
			Subjects: []string{SubjectNotificationEmail, SubjectNotificationPush, SubjectNotificationSMS}},
		{Stream: "USERS", Durable: "notify-users", Handler: h,
			Subjects: []string{SubjectUserRegistered}},
		{Stream: "CHEF", Durable: "notify-chef", Handler: h,
			Subjects: []string{SubjectChefNewOrder, SubjectChefVerified, SubjectChefTipReceived}},
		// Follower fan-out when a favorited chef publishes a weekly menu (#239).
		// Its own durable so the (potentially large) fan-out is processed
		// independently of the chef-facing notifications above.
		{Stream: "CHEF", Durable: "notify-weekly-menu", Handler: h,
			Subjects: []string{SubjectWeeklyMenuPublished}},
		// Follower fan-out on a per-DATE menu publish (#405/#419) — its own
		// durable so the fan-out is independent of the chef-facing notifications.
		{Stream: "CHEF", Durable: "notify-daily-menu", Handler: h,
			Subjects: []string{SubjectDailyMenuPublished}},
		// New review on a chef's order → notify the chef (#422).
		{Stream: "REVIEWS", Durable: "notify-reviews", Handler: h,
			Subjects: []string{SubjectReviewPosted}},
		// Referral reward granted → notify the referrer (#38).
		{Stream: "REFERRAL", Durable: "notify-referral", Handler: h,
			Subjects: []string{SubjectReferralRewarded}},
		// Loyalty points earned / redeemed → notify the customer (#40).
		{Stream: "LOYALTY", Durable: "notify-loyalty", Handler: h,
			Subjects: []string{SubjectLoyaltyEarned, SubjectLoyaltyRedeemed}},
		// Marketing campaign dispatch → fan out to the segment (#56). Long
		// AckWait (the fan-out can take a while) and few retries (the delivery
		// ledger makes a retry a cheap resume, not a re-send).
		{Stream: "CAMPAIGNS", Durable: "campaign-dispatch", Handler: h,
			Subjects: []string{SubjectCampaignDispatch}, AckWait: 10 * time.Minute, MaxDeliver: 3},
		// Win-back offer issued → nudge the lapsed/cancelled user (#42).
		{Stream: "SUBSCRIPTIONS", Durable: "notify-winback", Handler: h,
			Subjects: []string{SubjectSubscriptionWinbackOffered}},
		// Customer meal-subscription lifecycle → confirmations (#2/#3).
		{Stream: "SUBSCRIPTIONS", Durable: "notify-meal-subscription", Handler: h,
			Subjects: []string{SubjectMealSubscriptionCreated, SubjectMealSubscriptionCancelled}},
		{Stream: "DELIVERY", Durable: "notify-delivery", Handler: h,
			Subjects: []string{SubjectDeliveryAssigned, SubjectDeliveryPickedUp, SubjectDriverOnboardingSubmitted, SubjectDriverTipReceived}},
		{Stream: "APPROVALS", Durable: "notify-approvals", Handler: h,
			Subjects: []string{SubjectApprovalApproved, SubjectApprovalRejected, SubjectApprovalInfoRequested, SubjectApprovalCreated}},
		{Stream: "MEAL_PLANS", Durable: "notify-meal-plans", Handler: h,
			Subjects: []string{SubjectMealPlanCreated, SubjectMealPlanAcceptedFull, SubjectMealPlanModified, SubjectMealPlanConfirmed, SubjectMealPlanCancelled, SubjectMealPlanDayPrepared, SubjectMealPlanDayDelivered, SubjectMealPlanDayRefunded, SubjectMealPlanDaySkippedChef, SubjectMealPlanDaySkipRequested, SubjectMealPlanDaySkipDeclined, SubjectMealPlanDayFailed, SubjectMealPlanCompleted, SubjectMealPlanChefReminder, SubjectMealPlanPayoutReleased}},
		{Stream: "GROUP_ORDERS", Durable: "notify-group-orders", Handler: h,
			Subjects: []string{SubjectGroupOrderLocked, SubjectGroupOrderPlaced, SubjectGroupOrderCancelled}},
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
	case SubjectOrderVoided:
		return decodeThen(data, s.handleOrderVoided)
	case SubjectOrderAcceptReminder:
		return decodeThen(data, s.handleOrderAcceptReminder)
	case SubjectCancellationRequested:
		return decodeThen(data, s.handleCancellationRequested)
	case SubjectCancellationResolved:
		return decodeThen(data, s.handleCancellationResolved)
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
	case SubjectWeeklyMenuPublished:
		return decodeThen(data, s.handleWeeklyMenuPublished)
	case SubjectDailyMenuPublished:
		return decodeThen(data, s.handleDailyMenuPublished)
	case SubjectReviewPosted:
		return decodeThen(data, s.handleReviewPosted)
	case SubjectMealPlanDaySkippedChef:
		return decodeThen(data, s.handleMealPlanDaySkippedChef)
	case SubjectMealPlanDaySkipRequested:
		return decodeThen(data, s.handleMealPlanDaySkipRequested)
	case SubjectMealPlanDaySkipDeclined:
		return decodeThen(data, s.handleMealPlanDaySkipDeclined)
	case SubjectReferralRewarded:
		return decodeThen(data, s.handleReferralRewarded)
	case SubjectLoyaltyEarned:
		return decodeThen(data, s.handleLoyaltyEarned)
	case SubjectLoyaltyRedeemed:
		return decodeThen(data, s.handleLoyaltyRedeemed)
	case SubjectCampaignDispatch:
		return decodeThen(data, s.handleCampaignDispatch)
	case SubjectSubscriptionWinbackOffered:
		return decodeThen(data, s.handleSubscriptionWinbackOffered)
	case SubjectMealSubscriptionCreated:
		return decodeThen(data, s.handleMealSubscriptionLifecycle("Your meal subscription is active", "Your daily tiffin from your chef starts now. Manage it anytime in the app."))
	case SubjectMealSubscriptionCancelled:
		return decodeThen(data, s.handleMealSubscriptionLifecycle("Subscription cancelled", "Your meal subscription has been cancelled. We hope to cook for you again soon."))
	case SubjectOrderIssueReported:
		return decodeThen(data, s.handleOrderIssueReported)
	case SubjectChefTipReceived, SubjectDriverTipReceived:
		return decodeThen(data, s.handleTipReceived)
	case SubjectGroupOrderLocked:
		return decodeThen(data, s.handleGroupOrderLocked)
	case SubjectGroupOrderPlaced:
		return decodeThen(data, s.handleGroupOrderPlaced)
	case SubjectGroupOrderCancelled:
		return decodeThen(data, s.handleGroupOrderCancelled)
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
	case SubjectMealPlanCreated:
		return decodeThen(data, s.handleMealPlanCreated)
	case SubjectMealPlanAcceptedFull:
		return decodeThen(data, s.handleMealPlanAcceptedFull)
	case SubjectMealPlanModified:
		return decodeThen(data, s.handleMealPlanModified)
	case SubjectMealPlanConfirmed:
		return decodeThen(data, s.handleMealPlanConfirmed)
	case SubjectMealPlanCancelled:
		return decodeThen(data, s.handleMealPlanCancelled)
	case SubjectMealPlanDayPrepared:
		return decodeThen(data, s.handleMealPlanDayPrepared)
	case SubjectMealPlanDayDelivered:
		return decodeThen(data, s.handleMealPlanDayDelivered)
	case SubjectMealPlanDayRefunded:
		return decodeThen(data, s.handleMealPlanDayRefunded)
	case SubjectMealPlanDayFailed:
		return decodeThen(data, s.handleMealPlanDayFailed)
	case SubjectMealPlanCompleted:
		return decodeThen(data, s.handleMealPlanCompleted)
	case SubjectMealPlanChefReminder:
		return decodeThen(data, s.handleMealPlanChefReminder)
	case SubjectMealPlanPayoutReleased:
		return decodeThen(data, s.handleMealPlanPayoutReleased)
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

// ── Identity resolution ──────────────────────────────────────────────────────

// chefUserID maps OrderEvent.ChefID — which is a **chef_profiles.id**, NOT a
// users.id — onto the chef's user id.
//
// This asymmetry is the trap: Order.CustomerID references users directly
// (models.Customer is an alias of models.User), so passing it to a user lookup
// works. Order.ChefID references chef_profiles (models.Order.Chef has
// `foreignKey:ChefID`), so passing THAT to a user lookup always misses —
// `push: user <id> not found: record not found` — and every chef-facing
// notification silently dies, retries 5x and dead-letters. Notifications are
// best-effort, so nothing surfaces the failure to the caller: chefs simply
// never learned an order arrived.
//
// Resolve here rather than adding ChefUserID to OrderEvent: the NOTIFICATIONS
// stream is durable, so events published before this change (and any replayed
// from a DLQ) still carry only the profile id and must keep working.
func chefUserID(chefProfileID uuid.UUID) (uuid.UUID, error) {
	if chefProfileID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("resolve chef user: empty chef profile id")
	}
	var profile models.ChefProfile
	if err := database.DB.Select("id, user_id").First(&profile, "id = ?", chefProfileID).Error; err != nil {
		return uuid.Nil, fmt.Errorf("resolve chef user for profile %s: %w", chefProfileID, err)
	}
	if profile.UserID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("resolve chef user for profile %s: profile has no user_id", chefProfileID)
	}
	return profile.UserID, nil
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

// handleOrderCreated is intentionally a no-op. orders.created fires at order
// CREATION — before payment — and used to also alert the chef ("New Order
// Received"), duplicating the chef.new_order alert ("New Order!" + actionable
// Accept/Reject push) that fires on payment success for regular orders, or
// alongside generated/group/subscription orders. That left the chef with two
// notifications per order, and (worse) one for an order that was never paid.
// The chef is now alerted exactly once, via chef.new_order → handleChefNewOrder.
// Kept as a registered handler (rather than removed) so the notify-orders
// consumer's subscription to SubjectOrderCreated stays intact for future use.
func (s *NotificationService) handleOrderCreated(_ OrderEvent) error {
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

	// Status changes are push + in-app only — no email per transition (owner
	// decision 2026-07-20). This removes ~5 redundant emails per order lifecycle.
	PublishNotification(NotificationEvent{
		UserID: event.CustomerID, Type: "push",
		Title: "Order Update", Message: getOrderStatusMessage(event.Status),
		Data: map[string]any{"order_id": event.OrderID.String(), "status": event.Status},
	})
	return nil
}

func (s *NotificationService) handleOrderCancelled(event OrderEvent) error {
	// CustomerID is already a users.id; ChefID is a chef_profiles.id and must be
	// mapped before it can key a notification (see chefUserID).
	chefUser, err := chefUserID(event.ChefID)
	if err != nil {
		return fmt.Errorf("order_cancelled: %w", err)
	}

	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String(), "status": string(models.OrderStatusCancelled)})
	for _, userID := range []uuid.UUID{event.CustomerID, chefUser} {
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
	// No cancellation email — push + in-app only (owner decision 2026-07-20).
	return nil
}

// voidEvent is the orders.voided payload (#694). A distinct struct rather than
// OrderEvent because a void carries WHY — the slot and deadline that explain it —
// and the apology is only decent if it can be specific.
type voidEvent struct {
	OrderID        uuid.UUID `json:"order_id"`
	OrderNumber    string    `json:"order_number"`
	ChefID         uuid.UUID `json:"chef_id"`
	CustomerID     uuid.UUID `json:"customer_id"`
	Reason         string    `json:"reason"`
	Slot           string    `json:"slot"`
	RefundedAmount float64   `json:"refunded_amount"`
}

// handleOrderVoided apologises to the customer and confirms the refund for an
// order the kitchen closed on. The refund itself already happened (the void cron
// moves the money before publishing); this is the "we're sorry, here's your
// money back" the customer is owed. NO chef notification — a void is not the
// chef's fault to be told off for, and they already had their reminders.
func (s *NotificationService) handleOrderVoided(ev voidEvent) error {
	// The event carries customer_id, but fall back to the order if an older
	// producer omitted it, so an apology is never silently skipped.
	customerID := ev.CustomerID
	if customerID == uuid.Nil {
		var o models.Order
		if err := database.DB.Select("customer_id").First(&o, "id = ?", ev.OrderID).Error; err != nil {
			return fmt.Errorf("order_voided: resolve customer for %s: %w", ev.OrderID, err)
		}
		customerID = o.CustomerID
	}

	meal := ev.Slot
	if meal == "" {
		meal = "order"
	}
	title := "We're sorry — your order was cancelled"
	body := fmt.Sprintf(
		"No chef was able to accept your %s in time, so we've cancelled it and refunded ₹%.0f in full. We're sorry for the inconvenience.",
		meal, ev.RefundedAmount)

	data, _ := json.Marshal(map[string]any{
		"order_id": ev.OrderID.String(), "type": "order_voided", "refunded": ev.RefundedAmount,
	})
	if err := s.saveNotification(&models.Notification{
		UserID: customerID, Type: "order_voided", Title: title, Message: body, Data: string(data),
	}); err != nil {
		return fmt.Errorf("save order_voided notification for %s: %w", customerID, err)
	}
	// Push so the apology reaches them even with the app closed — being silently
	// refunded reads as "they took my money and cancelled my dinner". Email
	// intentionally dropped: order emails are push + in-app only, invoice excepted
	// (owner decision 2026-07-20).
	PublishNotification(NotificationEvent{
		UserID: customerID, Type: "push", Title: title, Message: body,
		// Both key casings: the customer app's deep-link handler reads camelCase
		// `orderId`, while every other consumer + our own tests read snake_case
		// `order_id`. Carrying both is cheaper than a coordinated rename and lets
		// a tap on the apology open the order (#694 UI).
		Data: map[string]any{"order_id": ev.OrderID.String(), "orderId": ev.OrderID.String(), "type": "order_voided"},
	})
	return nil
}

// acceptReminderEvent is the orders.accept_reminder payload (#694).
type acceptReminderEvent struct {
	OrderID     uuid.UUID `json:"order_id"`
	OrderNumber string    `json:"order_number"`
	ChefID      uuid.UUID `json:"chef_id"`
	Slot        string    `json:"slot"`
	MinutesLeft int       `json:"minutes_left"`
	FinalCall   bool      `json:"final_call"`
}

// handleOrderAcceptReminder nudges the CHEF about an order they have not accepted
// as their kitchen close approaches. An actionable push (not just a bell) so it
// can act as a real prompt to open the app and accept — the whole point is to
// prevent the void.
func (s *NotificationService) handleOrderAcceptReminder(ev acceptReminderEvent) error {
	chefUser, err := chefUserID(ev.ChefID)
	if err != nil {
		return fmt.Errorf("order_accept_reminder: %w", err)
	}

	title := "Order still needs accepting"
	if ev.FinalCall {
		title = "Last chance — accept before your kitchen closes"
	}
	body := fmt.Sprintf(
		"Order %s hasn't been accepted yet. Accept it in the next %d min or it will be auto-cancelled and the customer refunded.",
		ev.OrderNumber, ev.MinutesLeft)

	// Reuse the vendor app's EXISTING new-order push contract rather than inventing
	// a parallel one it doesn't handle: the shipped app registers the `new_order`
	// iOS category (Accept/Reject actions) and the high-priority `new-orders`
	// Android channel, and its tap handler + Accept action both read camelCase
	// `orderId`. Matching them makes tap-to-accept and the actionable buttons work
	// with NO mobile build — the app is already out there. (#694 UI)
	data := map[string]string{
		"orderId": ev.OrderID.String(), "order_id": ev.OrderID.String(),
		"type": "accept_reminder", "action": "accept_order",
	}
	// Actionable push so the chef can accept straight from the notification. Best-
	// effort by contract; the in-app row below is the durable record.
	if err := SendActionablePush(chefUser, title, body, "new-orders", "new_order", data); err != nil {
		log.Printf("accept-reminder push failed for chef %s: %v", chefUser, err)
	}

	dataJSON, _ := json.Marshal(map[string]any{
		"order_id": ev.OrderID.String(), "type": "accept_reminder",
	})
	return s.saveNotification(&models.Notification{
		UserID: chefUser, Type: "accept_reminder", Title: title, Message: body, Data: string(dataJSON),
	})
}

func (s *NotificationService) handleOrderDelivered(event OrderEvent) error {
	// Award loyalty points for the delivered order (#40). Idempotent on the
	// order id, so a redelivered event never double-earns; best-effort so a
	// transient points failure never blocks or duplicates the delivery
	// confirmation below.
	if _, err := AwardOrderLoyalty(database.DB, event.CustomerID, event.OrderID, event.Total); err != nil {
		log.Printf("loyalty award failed for order %s: %v", event.OrderID, err)
	}
	// If this delivered order is a meal-subscription fulfillment, flip its
	// fulfillment row to delivered and advance the adherence streak (#40).
	// No-op for normal orders; idempotent on redelivery.
	if err := MarkMealFulfillmentDelivered(database.DB, event.OrderID); err != nil {
		log.Printf("meal fulfillment delivered transition failed for order %s: %v", event.OrderID, err)
	}

	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String(), "status": string(models.OrderStatusDelivered)})
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
	// Capture the DB handle at spawn time rather than reading the global inside
	// the detached goroutine: the handler returns immediately, and the global can
	// be swapped/torn down out from under the goroutine (nil-deref panic in tests
	// that reset database.DB between cases).
	go emailDeliveredInvoice(database.DB, event)
	return nil
}

// emailDeliveredInvoice resolves the customer's email and dispatches the PDF
// invoice. Best-effort: the in-app + push confirmations above are authoritative.
func emailDeliveredInvoice(db *gorm.DB, event OrderEvent) {
	if db == nil {
		return
	}
	var customer models.User
	if err := db.First(&customer, "id = ?", event.CustomerID).Error; err != nil {
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
	// event.ChefID is a chef_profiles.id, not a users.id (see chefUserID).
	chefUser, err := chefUserID(event.ChefID)
	if err != nil {
		return fmt.Errorf("chef_new_order: %w", err)
	}

	data, _ := json.Marshal(map[string]any{"order_id": event.OrderID.String(), "total": event.Total})
	if err := s.saveNotification(&models.Notification{
		UserID:  chefUser,
		Type:    "new_order",
		Title:   "New Order!",
		Message: "You have a new order to prepare",
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save new_order notification: %w", err)
	}
	return nil
}

// handleTipReceived → chef or rider (event.UserID): a post-delivery tip (#45).
// In-app + push; the amount is the beneficiary's share.
func (s *NotificationService) handleTipReceived(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	amount, _ := event.Data["amount"].(float64)
	title := "You received a tip! 🎉"
	message := fmt.Sprintf("A customer tipped you ₹%.0f — it's on its way to your payout.", amount)
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "tip_received",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save tip_received notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleReviewPosted notifies the chef when a customer leaves a review on one of
// their delivered orders (#422). The event targets the chef's user id.
func (s *NotificationService) handleReviewPosted(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	rating, _ := event.Data["rating"].(float64)
	title := "New review ⭐"
	message := "A customer reviewed one of your orders."
	if rating > 0 {
		message = fmt.Sprintf("A customer left you a %.0f-star review.", rating)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "review_posted",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save review_posted notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleCancellationRequested notifies the chef that a customer wants to cancel
// an order — confirm it + pick a refund tier (#475). Targets the chef's user id.
func (s *NotificationService) handleCancellationRequested(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	title := "Cancellation request"
	message := "A customer asked to cancel an order — confirm it and choose the refund in the app."
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID: event.UserID, Type: "cancellation_requested", Title: title, Message: message, Data: string(data),
	}); err != nil {
		return fmt.Errorf("save cancellation_requested notification: %w", err)
	}
	PublishNotification(NotificationEvent{UserID: event.UserID, Type: "push", Title: title, Message: message, Data: event.Data})
	return nil
}

// handleCancellationResolved notifies the customer that their cancellation was
// confirmed and a refund issued (#475). Targets the customer's user id.
func (s *NotificationService) handleCancellationResolved(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	refund, _ := event.Data["refund"].(float64)
	title := "Order cancelled"
	message := "Your cancellation is confirmed."
	if refund > 0 {
		message = fmt.Sprintf("Your cancellation is confirmed — ₹%.0f refunded.", refund)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID: event.UserID, Type: "cancellation_resolved", Title: title, Message: message, Data: string(data),
	}); err != nil {
		return fmt.Errorf("save cancellation_resolved notification: %w", err)
	}
	PublishNotification(NotificationEvent{UserID: event.UserID, Type: "push", Title: title, Message: message, Data: event.Data})
	return nil
}

// ── Group / office orders (#46) ──────────────────────────────────────────────

func (s *NotificationService) notifyGroup(event Event, notifType, title, message string) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    notifType,
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save %s notification: %w", notifType, err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleGroupOrderLocked → participant: pay your share.
func (s *NotificationService) handleGroupOrderLocked(event Event) error {
	share, _ := event.Data["share"].(float64)
	return s.notifyGroup(event, "group_order_locked",
		"Time to pay your share",
		fmt.Sprintf("Your group order is locked. Pay your share of ₹%.0f to confirm it.", share))
}

// handleGroupOrderPlaced → host: the consolidated order is placed.
func (s *NotificationService) handleGroupOrderPlaced(event Event) error {
	return s.notifyGroup(event, "group_order_placed",
		"Group order placed 🎉",
		"Everyone has paid — your group order is on its way to the chef.")
}

// handleGroupOrderCancelled → participant: cancelled + refunded.
func (s *NotificationService) handleGroupOrderCancelled(event Event) error {
	return s.notifyGroup(event, "group_order_cancelled",
		"Group order cancelled",
		"A group order you were part of was cancelled. Any payment was refunded to your wallet.")
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

// handleWeeklyMenuPublished fans a "menu drop" out to every customer who has
// favorited the chef (#239): an in-app feed entry (+ real-time bell) and a push,
// each gated by the user's "favorites" notification preference. The heavy FCM
// call is offloaded — we publish to notifications.push and let notify-dispatch
// deliver, so this handler stays fast even for a chef with many followers.
func (s *NotificationService) handleWeeklyMenuPublished(event Event) error {
	chefID, chefName, err := followerEventChef(event, "weekly_menu_published")
	if err != nil {
		return err
	}
	return s.fanOutToFollowers(chefID, "weekly_menu_published",
		"New menu just dropped",
		fmt.Sprintf("%s published a new weekly menu — take a look!", chefName),
		map[string]any{"type": "weekly_menu_published", "chefId": chefID.String()})
}

// handleDailyMenuPublished fans a per-DATE "menu drop" (#405/#419) out to the
// chef's followers — the daily-menu counterpart of the weekly handler. Emitted
// by PutDailyMenu on the unpublished→published transition.
func (s *NotificationService) handleDailyMenuPublished(event Event) error {
	chefID, chefName, err := followerEventChef(event, "daily_menu_published")
	if err != nil {
		return err
	}
	date, _ := event.Data["date"].(string)
	msg := fmt.Sprintf("%s just published a fresh menu — book your tiffin!", chefName)
	if date != "" {
		msg = fmt.Sprintf("%s published the menu for %s — book your tiffin!", chefName, date)
	}
	return s.fanOutToFollowers(chefID, "daily_menu_published",
		"A new menu is up",
		msg,
		map[string]any{"type": "daily_menu_published", "chefId": chefID.String(), "date": date})
}

// followerEventChef extracts + validates the chef id/name from a menu-drop event.
func followerEventChef(event Event, kind string) (uuid.UUID, string, error) {
	chefIDStr, _ := event.Data["chef_id"].(string)
	chefName, _ := event.Data["chef_name"].(string)
	chefID, err := uuid.Parse(chefIDStr)
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("%s: bad chef_id %q: %w", kind, chefIDStr, err)
	}
	if chefName == "" {
		chefName = "A chef you follow"
	}
	return chefID, chefName, nil
}

// fanOutToFollowers sends an in-app feed entry (+ real-time bell) and a push to
// every customer who has favorited the chef, gated by their "favorites"
// preference (checked downstream in the push path). One follower's failure is
// logged and skipped so the whole batch isn't redelivered. Shared by the weekly
// and daily menu-drop handlers (#239/#419).
func (s *NotificationService) fanOutToFollowers(chefID uuid.UUID, notifType, title, message string, data map[string]any) error {
	var favorites []models.FavoriteChef
	if err := database.DB.Where("chef_id = ?", chefID).Find(&favorites).Error; err != nil {
		return fmt.Errorf("%s: load followers: %w", notifType, err)
	}
	inAppData, _ := json.Marshal(data)
	for _, fav := range favorites {
		if err := s.saveNotification(&models.Notification{
			UserID:  fav.UserID,
			Type:    notifType,
			Title:   title,
			Message: message,
			Data:    string(inAppData),
		}); err != nil {
			log.Printf("%s: save notification for %s: %v", notifType, fav.UserID, err)
			continue
		}
		PublishNotification(NotificationEvent{
			UserID:  fav.UserID,
			Type:    "push",
			Title:   title,
			Message: message,
			Data:    data,
		})
	}
	return nil
}

// handleReferralRewarded notifies the referrer that their reward landed (#38):
// in-app feed + push. event.UserID is the referrer; referrer_reward is the credit.
func (s *NotificationService) handleReferralRewarded(event Event) error {
	reward, _ := event.Data["referrer_reward"].(float64)
	if reward <= 0 {
		return nil
	}
	title := "You earned referral credit!"
	message := fmt.Sprintf("A friend placed their first order — ₹%.0f is now in your wallet.", reward)
	data, _ := json.Marshal(map[string]any{"type": "referral_rewarded"})

	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "referral_rewarded",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save referral_rewarded notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID:  event.UserID,
		Type:    "push",
		Title:   title,
		Message: message,
		Data:    map[string]any{"type": "referral_rewarded"},
	})
	return nil
}

// handleLoyaltyEarned → customer: points landed from a delivered order or a
// meal-subscription streak (#40). In-app + push; the body adapts to whether it
// was an order earn or a streak bonus.
func (s *NotificationService) handleLoyaltyEarned(event Event) error {
	points, _ := event.Data["points"].(float64)
	if points <= 0 {
		return nil
	}
	source, _ := event.Data["source"].(string)
	title := "You earned loyalty points!"
	message := fmt.Sprintf("You just earned %.0f points. Redeem them for wallet credit anytime.", points)
	if source == string(models.LoyaltySourceStreak) {
		streak, _ := event.Data["streak"].(float64)
		title = "Streak bonus unlocked! 🔥"
		message = fmt.Sprintf("%.0f days in a row — here's %.0f bonus points. Keep the streak going!", streak, points)
	}
	data, _ := json.Marshal(map[string]any{"type": "loyalty_earned", "points": points})
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "loyalty_earned",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save loyalty_earned notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message,
		Data: map[string]any{"type": "loyalty_earned", "points": points},
	})
	return nil
}

// handleLoyaltyRedeemed → customer: points converted to wallet store credit (#40).
func (s *NotificationService) handleLoyaltyRedeemed(event Event) error {
	points, _ := event.Data["points"].(float64)
	amount, _ := event.Data["amount"].(float64)
	if points <= 0 || amount <= 0 {
		return nil
	}
	title := "Points redeemed"
	message := fmt.Sprintf("You redeemed %.0f points for ₹%.0f wallet credit. Use it on your next order!", points, amount)
	data, _ := json.Marshal(map[string]any{"type": "loyalty_redeemed", "points": points, "amount": amount})
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "loyalty_redeemed",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save loyalty_redeemed notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message,
		Data: map[string]any{"type": "loyalty_redeemed"},
	})
	return nil
}

// handleCampaignDispatch fans a marketing campaign out to its segment (#56).
// Idempotent + resumable — DispatchCampaign skips recipients already sent, so a
// redelivery is a cheap resume.
func (s *NotificationService) handleCampaignDispatch(event Event) error {
	idStr, _ := event.Data["campaign_id"].(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		log.Printf("campaign dispatch: bad campaign_id %q — dropping", idStr)
		return nil
	}
	return DispatchCampaign(s.ctx, database.DB, id)
}

// handleSubscriptionWinbackOffered nudges a lapsed/cancelled user about their
// targeted win-back offer (#42) — in-app + push + email carrying the code,
// discount and expiry so they can come back in one tap. Mirrors the referral
// reward fan-out; the actual promo/offer was already minted by OfferWinback.
func (s *NotificationService) handleSubscriptionWinbackOffered(event Event) error {
	code, _ := event.Data["code"].(string)
	discount, _ := event.Data["discount_percent"].(float64)
	if code == "" || discount <= 0 {
		return nil
	}
	title := "We miss you — here's a treat"
	message := fmt.Sprintf("Come back and save %.0f%%. Use code %s before it expires.", discount, code)
	dataMap := map[string]any{"type": "winback_offer", "code": code, "discount_percent": discount}
	if exp, ok := event.Data["expires_at"].(string); ok {
		dataMap["expires_at"] = exp
	}
	data, _ := json.Marshal(dataMap)

	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "winback_offer",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save winback_offer notification: %w", err)
	}
	PublishNotification(NotificationEvent{UserID: event.UserID, Type: "push", Title: title, Message: message, Data: dataMap})
	PublishNotification(NotificationEvent{UserID: event.UserID, Type: "email", Title: title, Message: message, Data: dataMap})
	return nil
}

// handleMealSubscriptionLifecycle returns a notification handler that saves an
// in-app confirmation + push for a meal-subscription lifecycle event (#2/#3).
func (s *NotificationService) handleMealSubscriptionLifecycle(title, message string) func(Event) error {
	return func(event Event) error {
		data, _ := json.Marshal(map[string]any{"type": "meal_subscription"})
		if err := s.saveNotification(&models.Notification{
			UserID:  event.UserID,
			Type:    "meal_subscription",
			Title:   title,
			Message: message,
			Data:    string(data),
		}); err != nil {
			return fmt.Errorf("save meal_subscription notification: %w", err)
		}
		PublishNotification(NotificationEvent{UserID: event.UserID, Type: "push", Title: title, Message: message, Data: map[string]any{"type": "meal_subscription"}})
		return nil
	}
}

// handleOrderIssueReported notifies the chef that a customer reported an issue on
// one of their orders (#37): in-app feed + push. event.UserID is the chef's user.
func (s *NotificationService) handleOrderIssueReported(event Event) error {
	orderNumber, _ := event.Data["order_number"].(string)
	reason, _ := event.Data["reason"].(string)
	orderID, _ := event.Data["order_id"].(string)
	issueID, _ := event.Data["issue_id"].(string)

	title := "A customer reported an issue"
	message := fmt.Sprintf("Order %s — %s. Our team is reviewing it.", orderNumber, strings.ReplaceAll(reason, "_", " "))
	data, _ := json.Marshal(map[string]any{"type": "order_issue_reported", "orderId": orderID, "issueId": issueID})

	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "order_issue_reported",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save order_issue_reported notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID:  event.UserID,
		Type:    "push",
		Title:   title,
		Message: message,
		Data:    map[string]any{"type": "order_issue_reported", "orderId": orderID},
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

	// Delivery-assignment is a lifecycle update: push + in-app only, no email.
	// The delivered GST invoice is the sole order email (owner decision 2026-07-20).
	PublishNotification(NotificationEvent{
		UserID: customerID, Type: "push",
		Title:   "Delivery Partner Assigned",
		Message: "A delivery partner has been assigned to your order and will pick it up soon!",
		Data:    event.Data,
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

// ── Meal-plan (tiffin) lifecycle (MEAL_PLANS stream, #198) ───────────────────
//
// The producer sets event.UserID to the recipient's User.ID already — the chef's
// User.ID for chef-facing events, the customer's User.ID for customer-facing ones
// (see handlers/meal_plan.go + meal_plan_cron.go). So these handlers notify
// event.UserID directly; no chef-profile→user resolution is needed here.

// notifyMealPlan persists an in-app notification and emits a push to the event's
// target user. A nil target is dropped (unresolvable → not retryable). NOTE: this
// is NOT idempotent across consumer redelivery — Nats-Msg-Id dedup only collapses
// duplicate PUBLISHES of the same outbox row, not a redelivery after an ack loss /
// AckWait expiry / consumer restart, which would re-run this and create a second
// in-app row + push. Duplicates are tolerated (redelivery is rare; the happy path
// acks fast) rather than deduped on an event key.
func (s *NotificationService) notifyMealPlan(event Event, notifType, title, message string) error {
	if event.UserID == uuid.Nil {
		log.Printf("meal-plan notification %q: nil target user (dropping)", notifType)
		return nil
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    notifType,
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save %s notification: %w", notifType, err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleMealPlanCreated → chef: a customer pre-booked a tiffin plan to review.
func (s *NotificationService) handleMealPlanCreated(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_request",
		"New tiffin request",
		"A customer pre-booked a meal plan. Review the days you can cook and respond.")
}

// handleMealPlanAcceptedFull → customer: chef accepted every requested day.
func (s *NotificationService) handleMealPlanAcceptedFull(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_accepted",
		"Your meal plan is confirmed",
		"Your chef accepted every day of your tiffin plan. You're all set!")
}

// handleMealPlanModified → customer: chef cherry-picked a subset; approval needed.
func (s *NotificationService) handleMealPlanModified(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_modified",
		"Your chef revised the plan",
		"Your chef can cook some of the days you picked. Review and approve the updated plan.")
}

// handleMealPlanConfirmed → chef: customer approved the revised plan.
func (s *NotificationService) handleMealPlanConfirmed(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_confirmed",
		"Meal plan confirmed",
		"The customer approved your revised plan. It's confirmed.")
}

// handleMealPlanDayPrepared → customer: their chef has started cooking today's
// tiffin from a confirmed meal-plan day (#50). In-app + push — unlike
// day-delivered below, no other pipeline notification covers "prepared", so
// this is the customer's only signal that the dish is on the stove.
func (s *NotificationService) handleMealPlanDayPrepared(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	dishName, _ := event.Data["dishName"].(string)
	title := "Your tiffin is being prepared"
	message := "Your chef has started preparing today's tiffin."
	if dishName != "" {
		message = fmt.Sprintf("Your chef has started preparing today's tiffin: %s.", dishName)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "meal_plan_day_prepared",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save meal_plan_day_prepared notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleMealPlanDayDelivered → customer: a tiffin day was delivered. In-app +
// push. Previously in-app only on the assumption the order pipeline's own
// delivery push covers it, but a meal-plan day isn't always paired 1:1 with an
// order-level delivered event, so the tiffin-specific confirmation pushes too.
func (s *NotificationService) handleMealPlanDayDelivered(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	title := "Tiffin delivered"
	message := "Today's tiffin from your meal plan was delivered. Enjoy!"
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "meal_plan_day_delivered",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save meal_plan_day_delivered notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleMealPlanDayRefunded → customer: a skipped/undelivered day was refunded to
// the wallet. In-app + push (money movement is worth a push).
func (s *NotificationService) handleMealPlanDayRefunded(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_day_refunded",
		"Day refunded to your wallet",
		"A meal-plan day was refunded to your HomeChef wallet.")
}

// handleMealPlanDaySkippedChef notifies the CHEF that a customer skipped a plan
// day, so they don't cook it (#422). The event targets the chef's user id. Fired only
// once an ADMIN approves the skip (the day is now terminally `skipped`).
func (s *NotificationService) handleMealPlanDaySkippedChef(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_day_skipped_chef",
		"A plan day was skipped",
		"A customer skipped a day in their tiffin plan — no need to cook that meal.")
}

// handleMealPlanDaySkipRequested notifies the CHEF that a customer REQUESTED to skip a
// plan day, pending admin review (#422 policy change). Until it's approved the meal is
// on hold — the chef shouldn't start cooking it. The event targets the chef's user id.
func (s *NotificationService) handleMealPlanDaySkipRequested(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_day_skip_requested",
		"A skip was requested",
		"A customer asked to skip a day in their tiffin plan. Our team is reviewing it — hold off on cooking that meal for now.")
}

// handleMealPlanDaySkipDeclined notifies the CUSTOMER that their skip request was
// declined by the platform, so the day still stands and will be cooked/delivered as
// planned. In-app + push (they were expecting a refund). Targets the customer's user id.
func (s *NotificationService) handleMealPlanDaySkipDeclined(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_day_skip_declined",
		"Skip request declined",
		"Your request to skip a tiffin day couldn't be approved, so that day stands and will be delivered as planned.")
}

// handleMealPlanCancelled → chef or customer (event.UserID), tailored by cause:
// expiry sweep (no response in time) vs the customer declining the revised plan.
func (s *NotificationService) handleMealPlanCancelled(event Event) error {
	title, message := "Meal plan cancelled", "This tiffin meal plan was cancelled."
	switch {
	case event.Type == "meal_plan.expired":
		title = "Meal plan expired"
		message = "A tiffin meal plan expired because it wasn't responded to in time."
	default:
		if approved, ok := event.Data["approved"].(bool); ok && !approved {
			title = "Customer declined the revised plan"
			message = "The customer declined your revised meal plan, so it was cancelled."
		}
	}
	return s.notifyMealPlan(event, "meal_plan_cancelled", title, message)
}

// handleMealPlanDayFailed → customer: a tiffin day couldn't be delivered and is
// frozen pending admin resolution (#393). In-app + push (the customer paid and is
// waiting for food). The event targets the customer's user id.
func (s *NotificationService) handleMealPlanDayFailed(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	dishName, _ := event.Data["dishName"].(string)
	title := "Tiffin delivery issue"
	message := "Today's tiffin from your meal plan couldn't be delivered. Our team is sorting it out — you'll be updated soon."
	if dishName != "" {
		message = fmt.Sprintf("Today's tiffin (%s) couldn't be delivered. Our team is sorting it out — you'll be updated soon.", dishName)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "meal_plan_day_failed",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save meal_plan_day_failed notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleMealPlanCompleted → customer: every day of the plan has been served.
func (s *NotificationService) handleMealPlanCompleted(event Event) error {
	return s.notifyMealPlan(event, "meal_plan_completed",
		"Your weekly plan is complete",
		"Every day of your tiffin plan has been served. Book another week whenever you're ready!")
}

// handleMealPlanChefReminder → chef: the day's tiffin cook count (#tiffin-reminder).
// A night-before (tomorrow) or morning-of (today) push of how many meals to cook,
// with the lunch/dinner split. The event targets the chef's user id.
func (s *NotificationService) handleMealPlanChefReminder(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	total := numFromEventData(event.Data["total"])
	if total <= 0 {
		return nil
	}
	lunch := numFromEventData(event.Data["lunch"])
	dinner := numFromEventData(event.Data["dinner"])
	when := "today"
	if w, _ := event.Data["window"].(string); w == "tomorrow" {
		when = "tomorrow"
	}
	mealWord := "meals"
	if total == 1 {
		mealWord = "meal"
	}
	title := fmt.Sprintf("%d tiffin %s to cook %s", total, mealWord, when)
	parts := make([]string, 0, 2)
	if lunch > 0 {
		parts = append(parts, fmt.Sprintf("%d lunch", lunch))
	}
	if dinner > 0 {
		parts = append(parts, fmt.Sprintf("%d dinner", dinner))
	}
	message := fmt.Sprintf("You have %d tiffin %s to cook %s.", total, mealWord, when)
	if len(parts) > 0 {
		message = fmt.Sprintf("You have %d tiffin %s to cook %s (%s). Tap to see your prep list.", total, mealWord, when, strings.Join(parts, ", "))
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "meal_plan_chef_reminder",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save meal_plan_chef_reminder notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// handleMealPlanPayoutReleased → chef: a tiffin day's payment was released to their
// account (auto-release ~2 days after delivery, or an admin release).
func (s *NotificationService) handleMealPlanPayoutReleased(event Event) error {
	if event.UserID == uuid.Nil {
		return nil
	}
	dishName, _ := event.Data["dishName"].(string)
	title := "Tiffin payment released"
	message := "A tiffin day's payment has been released to your account."
	if dishName != "" {
		message = fmt.Sprintf("Your payment for %s has been released to your account.", dishName)
	}
	data, _ := json.Marshal(event.Data)
	if err := s.saveNotification(&models.Notification{
		UserID:  event.UserID,
		Type:    "meal_plan_payout_released",
		Title:   title,
		Message: message,
		Data:    string(data),
	}); err != nil {
		return fmt.Errorf("save meal_plan_payout_released notification: %w", err)
	}
	PublishNotification(NotificationEvent{
		UserID: event.UserID, Type: "push",
		Title: title, Message: message, Data: event.Data,
	})
	return nil
}

// numFromEventData coerces a JSON-decoded numeric event field (float64 over the
// wire, but tolerant of int/int64) to an int; anything else is 0.
func numFromEventData(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
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
	// Order-lifecycle status emails (order_confirmation, order_status, chef_new_order,
	// delivery_assigned) are intentionally NOT dispatched — lifecycle updates go out as
	// push + in-app only. The delivered GST invoice is the sole order email
	// (owner decision 2026-07-20). Do not re-add these cases.
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
	// Keys MUST be models.OrderStatus values. "confirmed" and "on_the_way" used
	// to sit here and matched nothing — the real statuses are "accepted" and
	// "delivering" — so those two stages, plus "rejected" which was missing
	// entirely, fell through to the raw-status fallback and pushed
	// "Your order status has been updated to: accepted" at the customer (#716).
	messages := map[string]string{
		string(models.OrderStatusAccepted):   "Your order has been confirmed by the chef!",
		string(models.OrderStatusPreparing):  "Your order is being prepared",
		string(models.OrderStatusReady):      "Your order is ready for pickup/delivery",
		string(models.OrderStatusPickedUp):   "Your order has been picked up by the delivery partner",
		string(models.OrderStatusDelivering): "Your order is on its way!",
		string(models.OrderStatusDelivered):  "Your order has been delivered. Enjoy!",
		string(models.OrderStatusCancelled):  "Your order has been cancelled",
		string(models.OrderStatusRejected):   "The chef couldn't take this order. Any payment is being refunded.",
		string(models.OrderStatusRefunded):   "Your order has been refunded",
	}
	if msg, ok := messages[status]; ok {
		return msg
	}
	return "Your order status has been updated to: " + status
}
