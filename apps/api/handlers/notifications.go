package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	natsclient "github.com/nats-io/nats.go"
)

type NotificationHandler struct{}

func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{}
}

// GetNotifications returns paginated notifications for the authenticated user
func (h *NotificationHandler) GetNotifications(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	unreadOnly := c.Query("unread") == "true"

	if page < 1 {
		page = 1
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	query := database.DB.Model(&models.Notification{}).Where("user_id = ?", userID)
	if unreadOnly {
		query = query.Where("is_read = ?", false)
	}

	var total int64
	query.Count(&total)

	var notifications []models.Notification
	query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&notifications)

	c.JSON(http.StatusOK, gin.H{
		"data": notifications,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// GetUnreadCount returns the count of unread notifications
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var count int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count)

	c.JSON(http.StatusOK, gin.H{"unreadCount": count})
}

// MarkAsRead marks a single notification as read
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	notifID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification ID"})
		return
	}

	now := time.Now()
	result := database.DB.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now})

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Marked as read"})
}

// MarkAllAsRead marks all notifications as read for the user
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	now := time.Now()

	database.DB.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Updates(map[string]interface{}{"is_read": true, "read_at": &now})

	c.JSON(http.StatusOK, gin.H{"message": "All notifications marked as read"})
}

// GetPreferences returns the caller's notification preferences as a
// category → channel-toggles grid. Missing categories come back with
// their built-in defaults so the UI can render the complete matrix.
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	prefs := services.GetUserPreferences(userID)
	categories := models.AllNotificationCategories()
	c.JSON(http.StatusOK, gin.H{
		"categories":  categories,
		"preferences": prefs,
	})
}

// UpdatePreference upserts one category row for the caller. Body:
// { "category": "order", "emailEnabled": true, "pushEnabled": false, "smsEnabled": false }
func (h *NotificationHandler) UpdatePreference(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Category     string `json:"category" binding:"required"`
		EmailEnabled bool   `json:"emailEnabled"`
		PushEnabled  bool   `json:"pushEnabled"`
		SMSEnabled   bool   `json:"smsEnabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Validate the category is one of the known values so we don't accept
	// arbitrary strings into the DB.
	valid := false
	for _, cat := range models.AllNotificationCategories() {
		if string(cat) == req.Category {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown category"})
		return
	}
	if err := services.UpsertUserPreference(userID, models.NotificationCategory(req.Category), req.EmailEnabled, req.PushEnabled, req.SMSEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Preference saved"})
}

// notifWSUpgrader upgrades HTTP connections to WebSocket for the notification stream.
var notifWSUpgrader = websocket.Upgrader{
	CheckOrigin:  func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// StreamNotificationsWS upgrades to WebSocket and streams real-time notifications to
// the authenticated user. Subscribes to the per-user NATS subject notifications.user.{userID}.
// On connect, sends the current unread count so the bell badge can be set immediately.
// GET /api/v1/notifications/ws
func (h *NotificationHandler) StreamNotificationsWS(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := notifWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Notification WS upgrade failed for user %s: %v", userID, err)
		return
	}
	defer conn.Close()

	// Send initial unread count on connect
	var unreadCount int64
	database.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&unreadCount)
	initial, _ := json.Marshal(map[string]interface{}{
		"type":        "unread_count",
		"unreadCount": unreadCount,
	})
	conn.WriteMessage(websocket.TextMessage, initial)

	// Buffered channel for serialised writes
	writeCh := make(chan []byte, 64)
	defer close(writeCh)

	// Write pump
	go func() {
		for msg := range writeCh {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				log.Printf("Notification WS write error for user %s: %v", userID, err)
				return
			}
		}
	}()

	// Subscribe to per-user NATS subject
	subject := fmt.Sprintf("%s.%s", services.SubjectNotificationUser, userID.String())
	sub, err := services.GetNATSClient().Subscribe(subject, func(msg *natsclient.Msg) {
		// Wrap the notification payload with a type field
		var notif map[string]interface{}
		if err := json.Unmarshal(msg.Data, &notif); err != nil {
			return
		}
		notif["type"] = "new_notification"

		// Also include updated unread count
		var count int64
		database.DB.Model(&models.Notification{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count)
		notif["unreadCount"] = count

		payload, _ := json.Marshal(notif)
		select {
		case writeCh <- payload:
		default:
			// Channel full — drop
		}
	})
	if err != nil {
		log.Printf("NATS subscribe failed for notification stream user %s: %v", userID, err)
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(1011, "Internal error"))
		return
	}
	defer sub.Unsubscribe()

	// Read pump — blocks until client disconnects
	conn.SetReadLimit(512)
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// RegisterPushConsumers starts NATS queue subscribers that fire FCM push notifications
// when order/delivery events are received. Uses the "push-workers" queue group so it
// runs alongside the existing "notification-workers" group without duplicate processing
// within each group.
//
// Call this once after NATS connects, inside the NATS-available block in main.go.
func RegisterPushConsumers() {
	nats := services.GetNATSClient()

	// SubjectChefNewOrder → actionable push (Accept/Reject lock-screen buttons) to vendor.
	// T-04-13: the API endpoint verifies chef_id = current user before accepting the action.
	_, err := nats.QueueSubscribe(services.SubjectChefNewOrder, "push-workers", func(msg *natsclient.Msg) {
		var event services.OrderEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("[push] Failed to parse SubjectChefNewOrder event: %v", err)
			return
		}
		if event.ChefID == uuid.Nil {
			return
		}

		// Resolve the chef's UserID so SendActionablePush can look up the FCM token.
		var chef models.ChefProfile
		if err := database.DB.Select("id, user_id").Where("id = ?", event.ChefID).First(&chef).Error; err != nil {
			log.Printf("[push] Chef not found for push (chef_id=%s): %v", event.ChefID, err)
			return
		}

		// Non-blocking: push failure must not crash the NATS consumer loop (T-04-15).
		go func() {
			if err := services.SendActionablePush(
				chef.UserID,
				"New Order",
				"You have a new order waiting for your confirmation",
				"new-orders", // Android channel ID (D-09)
				"new_order",  // iOS category — matches what vendor _layout.tsx registers
				map[string]string{
					"type":    "new_order",
					"orderId": event.OrderID.String(),
					"action":  "vendor_new_order",
				},
			); err != nil {
				log.Printf("[push] SendActionablePush failed for chef user %s: %v", chef.UserID, err)
			}
		}()
	})
	if err != nil {
		log.Printf("[push] Failed to subscribe to SubjectChefNewOrder for push: %v", err)
	}

	// SubjectDeliveryAssigned → plain push to driver (new delivery alert).
	_, err = nats.QueueSubscribe(services.SubjectDeliveryAssigned, "push-workers", func(msg *natsclient.Msg) {
		var event services.Event
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("[push] Failed to parse SubjectDeliveryAssigned event: %v", err)
			return
		}

		driverIDStr, _ := event.Data["driver_id"].(string)
		deliveryIDStr, _ := event.Data["delivery_id"].(string)
		if driverIDStr == "" {
			return
		}

		driverID, parseErr := uuid.Parse(driverIDStr)
		if parseErr != nil {
			log.Printf("[push] Invalid driver_id in SubjectDeliveryAssigned: %v", parseErr)
			return
		}

		// Resolve the driver's UserID so SendPushNotification can look up the FCM token.
		var partner models.DeliveryPartner
		if err := database.DB.Select("id, user_id").Where("id = ?", driverID).First(&partner).Error; err != nil {
			log.Printf("[push] DeliveryPartner not found for push (partner_id=%s): %v", driverID, err)
			return
		}

		go func() {
			if err := services.SendPushNotification(
				partner.UserID,
				"New Delivery Available",
				"A delivery near you is ready for pickup",
				map[string]string{
					"type":       "new_delivery",
					"deliveryId": deliveryIDStr,
				},
			); err != nil {
				log.Printf("[push] SendPushNotification failed for driver user %s: %v", partner.UserID, err)
			}
		}()
	})
	if err != nil {
		log.Printf("[push] Failed to subscribe to SubjectDeliveryAssigned for push: %v", err)
	}

	// SubjectOrderUpdated → plain push to customer on order status changes.
	// The existing "notification-workers" group stores a DB record; this consumer
	// fires the FCM push with structured data for deep linking into the order screen.
	_, err = nats.QueueSubscribe(services.SubjectOrderUpdated, "push-workers", func(msg *natsclient.Msg) {
		var event services.OrderEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("[push] Failed to parse SubjectOrderUpdated event: %v", err)
			return
		}
		if event.CustomerID == uuid.Nil {
			return
		}

		go func() {
			if err := services.SendPushNotification(
				event.CustomerID,
				"Order Update",
				fmt.Sprintf("Your order is now %s", humanReadableOrderStatus(event.Status)),
				map[string]string{
					"type":    "order_update",
					"orderId": event.OrderID.String(),
					"status":  event.Status,
				},
			); err != nil {
				log.Printf("[push] SendPushNotification failed for customer %s: %v", event.CustomerID, err)
			}
		}()
	})
	if err != nil {
		log.Printf("[push] Failed to subscribe to SubjectOrderUpdated for push: %v", err)
	}

	log.Println("[push] Push consumers registered (SubjectChefNewOrder, SubjectDeliveryAssigned, SubjectOrderUpdated)")
}

// humanReadableOrderStatus returns a customer-friendly label for an order status string.
func humanReadableOrderStatus(status string) string {
	labels := map[string]string{
		"accepted":   "accepted by the chef",
		"preparing":  "being prepared",
		"ready":      "ready for pickup",
		"picked_up":  "picked up by the driver",
		"delivering": "on the way to you",
		"delivered":  "delivered",
		"cancelled":  "cancelled",
	}
	if label, ok := labels[status]; ok {
		return label
	}
	return status
}
