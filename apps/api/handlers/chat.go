package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
	"github.com/lib/pq"
)

type ChatHandler struct {
	// Simple per-user rate limiter: tracks last message timestamp per user ID.
	rateMu    sync.Mutex
	rateLimit map[uuid.UUID]time.Time
}

func NewChatHandler() *ChatHandler {
	return &ChatHandler{
		rateLimit: make(map[uuid.UUID]time.Time),
	}
}

// ---------- Endpoints ----------

// GetOrCreateChatRoom returns an existing chat room for the order+type, or creates one.
// GET /orders/:orderId/chat/:type
// :type = "chef" or "delivery"
func (h *ChatHandler) GetOrCreateChatRoom(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	orderID, err := uuid.Parse(c.Param("orderId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	chatType := c.Param("type")
	var roomType string
	switch chatType {
	case "chef":
		roomType = models.ChatRoomCustomerChef
	case "delivery":
		roomType = models.ChatRoomCustomerDelivery
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chat type. Must be 'chef' or 'delivery'"})
		return
	}

	// Load the order to verify the user is a participant
	var order models.Order
	if err := database.DB.First(&order, "id = ?", orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Determine if the requesting user is a valid participant
	if !isOrderParticipant(userID, &order, roomType) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You are not a participant in this order"})
		return
	}

	// Check if the order is in a terminal state — block new room creation
	if isTerminalOrderStatus(order.Status) {
		// Still allow fetching existing rooms, but don't create new ones
		var room models.ChatRoom
		if err := database.DB.Preload("Counterparty").
			Where("order_id = ? AND type = ?", orderID, roomType).
			First(&room).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Chat is not available for completed/cancelled orders"})
			return
		}
		// Auto-close if still active
		if room.Status == models.ChatRoomActive {
			database.DB.Model(&room).Update("status", models.ChatRoomClosed)
			room.Status = models.ChatRoomClosed
		}
		c.JSON(http.StatusOK, room.ToResponse())
		return
	}

	// Determine counterparty ID
	counterpartyID, err := resolveCounterpartyID(&order, roomType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find or create the chat room
	var room models.ChatRoom
	result := database.DB.Where("order_id = ? AND type = ?", orderID, roomType).First(&room)
	if result.Error != nil {
		// Create new room
		room = models.ChatRoom{
			OrderID:        orderID,
			Type:           roomType,
			CustomerID:     order.CustomerID,
			CounterpartyID: counterpartyID,
			Status:         models.ChatRoomActive,
		}
		if err := database.DB.Create(&room).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chat room"})
			return
		}
	}

	// Load counterparty for response
	database.DB.First(&room.Counterparty, "id = ?", room.CounterpartyID)

	c.JSON(http.StatusOK, room.ToResponse())
}

// GetMessages returns paginated messages for a chat room (newest first).
// GET /chat/:roomId/messages?page=1&limit=50
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	roomID, err := uuid.Parse(c.Param("roomId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	room, err := h.loadAndAuthorize(roomID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}
	_ = room

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}
	offset := (page - 1) * limit

	var total int64
	database.DB.Model(&models.ChatMessage{}).Where("chat_room_id = ?", roomID).Count(&total)

	var messages []models.ChatMessage
	if err := database.DB.Where("chat_room_id = ?", roomID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&messages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch messages"})
		return
	}

	responses := make([]models.ChatMessageResponse, len(messages))
	for i, msg := range messages {
		responses[i] = msg.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
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

// SendMessage sends a text message to a chat room. The content is PII-filtered before saving.
// POST /chat/:roomId/messages
// Body: { "content": "..." }
func (h *ChatHandler) SendMessage(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Reject multipart/form-data (no file uploads)
	ct := c.ContentType()
	if ct == "multipart/form-data" || ct == "multipart/mixed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File uploads are not allowed in chat"})
		return
	}

	roomID, err := uuid.Parse(c.Param("roomId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	room, err := h.loadAndAuthorize(roomID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}

	// Check room is active
	if room.Status != models.ChatRoomActive {
		c.JSON(http.StatusBadRequest, gin.H{"error": "This chat room is closed"})
		return
	}

	// Auto-close if order reached terminal status
	var order models.Order
	if err := database.DB.First(&order, "id = ?", room.OrderID).Error; err == nil {
		if isTerminalOrderStatus(order.Status) {
			database.DB.Model(&room).Update("status", models.ChatRoomClosed)
			c.JSON(http.StatusBadRequest, gin.H{"error": "This chat room is closed because the order is no longer active"})
			return
		}
	}

	// Rate limit: max 1 message per second per user
	if !h.checkRateLimit(userID) {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Please wait a moment before sending another message"})
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message content is required"})
		return
	}

	// Max 500 characters
	if len(req.Content) > 500 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message must be 500 characters or fewer"})
		return
	}

	if len(req.Content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message content cannot be empty"})
		return
	}

	// Determine sender role
	senderRole := determineSenderRole(userID, room)

	// Run PII filter
	sanitized, hasPII, violations := services.FilterChatMessage(req.Content)

	msg := models.ChatMessage{
		ChatRoomID:     roomID,
		SenderID:       userID,
		SenderRole:     senderRole,
		Content:        sanitized,
		OriginalLength: len(req.Content),
		PIIDetected:    hasPII,
		PIIViolations:  pq.StringArray(violations),
		MessageType:    "text",
		IsRead:         false,
	}

	if err := database.DB.Create(&msg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}

	// Update last message timestamp on the room
	now := time.Now()
	database.DB.Model(&room).Update("last_message_at", now)

	response := msg.ToResponse()

	// If PII was detected, include a warning in the response
	if hasPII {
		c.JSON(http.StatusCreated, gin.H{
			"data":    response,
			"warning": "Your message contained personal information that was automatically redacted for safety.",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": response})
}

// MarkAsRead marks a message as read.
// PUT /chat/:roomId/messages/:messageId/read
func (h *ChatHandler) MarkAsRead(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	roomID, err := uuid.Parse(c.Param("roomId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid room ID"})
		return
	}

	messageID, err := uuid.Parse(c.Param("messageId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// Verify user is a participant in this room
	_, err = h.loadAndAuthorize(roomID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chat room not found"})
		return
	}

	// Mark the message as read (only if the reader is NOT the sender)
	result := database.DB.Model(&models.ChatMessage{}).
		Where("id = ? AND chat_room_id = ? AND sender_id != ?", messageID, roomID, userID).
		Update("is_read", true)

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found or you are the sender"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Message marked as read"})
}

// ListChatRooms returns all active chat rooms for the authenticated user.
// GET /chat/rooms
func (h *ChatHandler) ListChatRooms(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var rooms []models.ChatRoom
	if err := database.DB.Preload("Counterparty").
		Where("(customer_id = ? OR counterparty_id = ?) AND status = ?", userID, userID, models.ChatRoomActive).
		Order("last_message_at DESC NULLS LAST, created_at DESC").
		Find(&rooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch chat rooms"})
		return
	}

	// For counterparties: if the requesting user IS the counterparty, swap to show the customer name
	responses := make([]models.ChatRoomResponse, len(rooms))
	for i, room := range rooms {
		if room.CounterpartyID == userID {
			// The user is the counterparty, so show the customer's name
			var customer models.User
			database.DB.First(&customer, "id = ?", room.CustomerID)
			room.Counterparty = customer
		}
		responses[i] = room.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"data": responses})
}

// ---------- Helpers ----------

// loadAndAuthorize loads a chat room and verifies the user is a participant.
func (h *ChatHandler) loadAndAuthorize(roomID, userID uuid.UUID) (*models.ChatRoom, error) {
	var room models.ChatRoom
	if err := database.DB.Where(
		"id = ? AND (customer_id = ? OR counterparty_id = ?)", roomID, userID, userID,
	).First(&room).Error; err != nil {
		return nil, err
	}
	return &room, nil
}

// isOrderParticipant checks whether a user is a valid participant for the given room type.
func isOrderParticipant(userID uuid.UUID, order *models.Order, roomType string) bool {
	// Customer is always a valid participant
	if userID == order.CustomerID {
		return true
	}

	switch roomType {
	case models.ChatRoomCustomerChef:
		// The chef's user ID — look up chef profile
		var chef models.ChefProfile
		if err := database.DB.First(&chef, "id = ?", order.ChefID).Error; err != nil {
			return false
		}
		return userID == chef.UserID

	case models.ChatRoomCustomerDelivery:
		// The delivery partner's user ID
		if order.DeliveryID == nil {
			return false
		}
		var partner models.DeliveryPartner
		if err := database.DB.First(&partner, "user_id = ?", *order.DeliveryID).Error; err != nil {
			// DeliveryID on order might be the partner's user ID directly, or the partner's ID
			// Try by partner ID
			if err := database.DB.First(&partner, "id = ?", *order.DeliveryID).Error; err != nil {
				return false
			}
		}
		return userID == partner.UserID
	}

	return false
}

// resolveCounterpartyID determines who the counterparty (non-customer) user is for a room type.
func resolveCounterpartyID(order *models.Order, roomType string) (uuid.UUID, error) {
	switch roomType {
	case models.ChatRoomCustomerChef:
		var chef models.ChefProfile
		if err := database.DB.First(&chef, "id = ?", order.ChefID).Error; err != nil {
			return uuid.Nil, fmt.Errorf("chef not found for this order")
		}
		return chef.UserID, nil

	case models.ChatRoomCustomerDelivery:
		if order.DeliveryID == nil {
			return uuid.Nil, fmt.Errorf("no delivery partner assigned to this order yet")
		}
		// Try as partner table ID first
		var partner models.DeliveryPartner
		if err := database.DB.First(&partner, "id = ?", *order.DeliveryID).Error; err != nil {
			// Try as user ID
			if err := database.DB.First(&partner, "user_id = ?", *order.DeliveryID).Error; err != nil {
				return uuid.Nil, fmt.Errorf("delivery partner not found for this order")
			}
		}
		return partner.UserID, nil
	}

	return uuid.Nil, fmt.Errorf("unknown room type")
}

// isTerminalOrderStatus returns true for order statuses that should close the chat.
func isTerminalOrderStatus(status models.OrderStatus) bool {
	return status == models.OrderStatusDelivered ||
		status == models.OrderStatusCancelled ||
		status == models.OrderStatusRefunded
}

// determineSenderRole returns the role string for the sender in a chat room.
func determineSenderRole(userID uuid.UUID, room *models.ChatRoom) string {
	if userID == room.CustomerID {
		return "customer"
	}
	// Counterparty role depends on room type
	if room.Type == models.ChatRoomCustomerChef {
		return "chef"
	}
	return "delivery"
}

// checkRateLimit enforces a 1-message-per-second limit per user. Returns true if allowed.
func (h *ChatHandler) checkRateLimit(userID uuid.UUID) bool {
	h.rateMu.Lock()
	defer h.rateMu.Unlock()

	last, exists := h.rateLimit[userID]
	now := time.Now()

	if exists && now.Sub(last) < time.Second {
		return false
	}

	h.rateLimit[userID] = now
	return true
}
