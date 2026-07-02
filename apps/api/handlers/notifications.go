package handlers

import (
	"context"
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
// Reuses allowedWSOrigin (see handlers/orders.go) so the notification socket enforces
// the same cross-site WS hijacking protection as the order-track socket.
var notifWSUpgrader = websocket.Upgrader{
	CheckOrigin:     allowedWSOrigin,
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

// RegisterPushConsumers registers the durable JetStream consumers that fire FCM
// push notifications for order/delivery events. They run under "push-*" durables,
// independent of the notification service's "notify-*" durables, so each event is
// pushed exactly once per group with retry + dead-lettering (issue #134).
func RegisterPushConsumers(ctx context.Context, cm *services.ConsumerManager) error {
	return cm.RegisterAll(ctx,
		services.ConsumerSpec{Stream: "CHEF", Durable: "push-chef-new-order",
			Subjects: []string{services.SubjectChefNewOrder}, Handler: pushChefNewOrder},
		services.ConsumerSpec{Stream: "DELIVERY", Durable: "push-delivery-assigned",
			Subjects: []string{services.SubjectDeliveryAssigned}, Handler: pushDeliveryAssigned},
		services.ConsumerSpec{Stream: "ORDERS", Durable: "push-order-updated",
			Subjects: []string{services.SubjectOrderUpdated}, Handler: pushOrderUpdated},
	)
}

// pushChefNewOrder → actionable push (Accept/Reject) to the vendor. The API
// endpoint verifies chef_id = current user before accepting the action (T-04-13).
func pushChefNewOrder(_ context.Context, _ string, data []byte) error {
	var event services.OrderEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("parse chef new order: %w", err)
	}
	if event.ChefID == uuid.Nil {
		return nil
	}
	var chef models.ChefProfile
	if err := database.DB.Select("id, user_id").Where("id = ?", event.ChefID).First(&chef).Error; err != nil {
		log.Printf("[push] chef not found (chef_id=%s), dropping: %v", event.ChefID, err)
		return nil // unresolvable target is not retryable
	}
	return services.SendActionablePush(
		chef.UserID,
		"New Order",
		"You have a new order waiting for your confirmation",
		"new-orders", // Android channel ID (D-09)
		"new_order",  // iOS category — matches vendor _layout.tsx registration
		map[string]string{"type": "new_order", "orderId": event.OrderID.String(), "action": "vendor_new_order"},
	)
}

// pushDeliveryAssigned → plain push to the driver (new delivery alert).
func pushDeliveryAssigned(_ context.Context, _ string, data []byte) error {
	var event services.Event
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("parse delivery assigned: %w", err)
	}
	driverIDStr, _ := event.Data["driver_id"].(string)
	deliveryIDStr, _ := event.Data["delivery_id"].(string)
	if driverIDStr == "" {
		return nil
	}
	driverID, err := uuid.Parse(driverIDStr)
	if err != nil {
		log.Printf("[push] invalid driver_id, dropping: %v", err)
		return nil
	}
	var partner models.DeliveryPartner
	if err := database.DB.Select("id, user_id").Where("id = ?", driverID).First(&partner).Error; err != nil {
		log.Printf("[push] delivery partner not found (id=%s), dropping: %v", driverID, err)
		return nil
	}
	return services.SendPushNotification(
		partner.UserID,
		"New Delivery Available",
		"A delivery near you is ready for pickup",
		map[string]string{"type": "new_delivery", "deliveryId": deliveryIDStr},
	)
}

// pushOrderUpdated → plain push to the customer on order status changes, with
// structured data for deep linking into the order screen.
func pushOrderUpdated(_ context.Context, _ string, data []byte) error {
	var event services.OrderEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return fmt.Errorf("parse order updated: %w", err)
	}
	if event.CustomerID == uuid.Nil {
		return nil
	}
	return services.SendPushNotification(
		event.CustomerID,
		"Order Update",
		fmt.Sprintf("Your order is now %s", humanReadableOrderStatus(event.Status)),
		map[string]string{"type": "order_update", "orderId": event.OrderID.String(), "status": event.Status},
	)
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
