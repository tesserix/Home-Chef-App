package handlers

// messaging.go — HTTP surface for admin-mediated in-app messaging (#53/#303).
// Customer and chef each message via an admin relay (no direct channel); admins
// run the mediation inbox (relay / block / direct send). Storage is MongoDB, so
// every endpoint 503s gracefully when Mongo is unavailable.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// MessagingHandler builds a fresh service per request from the live Mongo store
// (nil when Mongo is down).
type MessagingHandler struct{}

func NewMessagingHandler() *MessagingHandler { return &MessagingHandler{} }

// orderIDFromParam reads the order id from either :id (customer/admin routes) or
// :orderId (chef routes use that name elsewhere in the same group).
func orderIDFromParam(c *gin.Context) (uuid.UUID, error) {
	v := c.Param("id")
	if v == "" {
		v = c.Param("orderId")
	}
	return uuid.Parse(v)
}

func svcOr503(c *gin.Context) *services.MessagingService {
	svc := services.MessagingFromMongo()
	if svc == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Messaging is temporarily unavailable"})
		return nil
	}
	return svc
}

type sendMessageRequest struct {
	Content string `json:"content" binding:"required"`
}

// orderParticipants loads the order and returns (customerUserID, chefUserID),
// verifying the requester (by role) is a participant.
func orderParticipants(c *gin.Context, orderID, requesterID uuid.UUID, asRole string) (string, string, bool) {
	var order models.Order
	if err := database.DB.Select("id", "customer_id", "chef_id").First(&order, "id = ?", orderID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return "", "", false
	}
	// Map the order's chef profile to its user id (push + identity use the user id).
	var chef models.ChefProfile
	if err := database.DB.Select("id", "user_id").First(&chef, "id = ?", order.ChefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return "", "", false
	}
	switch asRole {
	case services.MsgRoleCustomer:
		if order.CustomerID != requesterID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not your order"})
			return "", "", false
		}
	case services.MsgRoleChef:
		if chef.UserID != requesterID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not your order"})
			return "", "", false
		}
	}
	return order.CustomerID.String(), chef.UserID.String(), true
}

func messagesJSON(c *gin.Context, msgs []services.MediatedMessage) {
	c.JSON(http.StatusOK, gin.H{"data": msgs})
}

// ── Customer ────────────────────────────────────────────────────────────────

// CustomerSendMessage — POST /customer/orders/:id/messages
func (h *MessagingHandler) CustomerSendMessage(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	userID, _ := middleware.GetUserID(c)
	orderID, err := orderIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message content is required"})
		return
	}
	custID, chefID, ok := orderParticipants(c, orderID, userID, services.MsgRoleCustomer)
	if !ok {
		return
	}
	m, err := svc.CustomerSend(c.Request.Context(), orderID.String(), custID, chefID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}
	resp := gin.H{"data": m}
	if m.PIIDetected {
		resp["warning"] = "Your message contained personal information that was redacted for safety."
	}
	c.JSON(http.StatusCreated, resp)
}

// CustomerListMessages — GET /customer/orders/:id/messages
func (h *MessagingHandler) CustomerListMessages(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	userID, _ := middleware.GetUserID(c)
	orderID, err := orderIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	custID, chefID, ok := orderParticipants(c, orderID, userID, services.MsgRoleCustomer)
	if !ok {
		return
	}
	msgs, _, err := svc.OrderThread(c.Request.Context(), orderID.String(), custID, chefID, services.MsgRoleCustomer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages"})
		return
	}
	messagesJSON(c, msgs)
}

// ── Chef ────────────────────────────────────────────────────────────────────

// ChefSendMessage — POST /chef/orders/:id/messages
func (h *MessagingHandler) ChefSendMessage(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	userID, _ := middleware.GetUserID(c)
	orderID, err := orderIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message content is required"})
		return
	}
	custID, chefID, ok := orderParticipants(c, orderID, userID, services.MsgRoleChef)
	if !ok {
		return
	}
	m, err := svc.ChefSend(c.Request.Context(), orderID.String(), custID, chefID, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send message"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": m})
}

// ChefListMessages — GET /chef/orders/:id/messages
func (h *MessagingHandler) ChefListMessages(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	userID, _ := middleware.GetUserID(c)
	orderID, err := orderIDFromParam(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order id"})
		return
	}
	custID, chefID, ok := orderParticipants(c, orderID, userID, services.MsgRoleChef)
	if !ok {
		return
	}
	msgs, _, err := svc.OrderThread(c.Request.Context(), orderID.String(), custID, chefID, services.MsgRoleChef)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load messages"})
		return
	}
	messagesJSON(c, msgs)
}

// ── Admin (mediation) ───────────────────────────────────────────────────────

// AdminInbox — GET /admin/messages/inbox
func (h *MessagingHandler) AdminInbox(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	msgs, err := svc.AdminInbox(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load inbox"})
		return
	}
	messagesJSON(c, msgs)
}

// AdminRelayMessage — POST /admin/messages/:id/relay
func (h *MessagingHandler) AdminRelayMessage(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	adminID, _ := middleware.GetUserID(c)
	m, err := svc.AdminRelay(c.Request.Context(), c.Param("id"), adminID.String())
	if err != nil {
		messagingActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": m})
}

// AdminBlockMessage — POST /admin/messages/:id/block
func (h *MessagingHandler) AdminBlockMessage(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	adminID, _ := middleware.GetUserID(c)
	if err := svc.AdminBlock(c.Request.Context(), c.Param("id"), adminID.String()); err != nil {
		messagingActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

type adminSendRequest struct {
	RecipientRole string `json:"recipientRole" binding:"required"`
	Content       string `json:"content" binding:"required"`
}

// AdminSendMessage — POST /admin/conversations/:id/send
func (h *MessagingHandler) AdminSendMessage(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	adminID, _ := middleware.GetUserID(c)
	var req adminSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipientRole and content are required"})
		return
	}
	if req.RecipientRole != services.MsgRoleCustomer && req.RecipientRole != services.MsgRoleChef {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recipientRole must be customer or chef"})
		return
	}
	m, err := svc.AdminSend(c.Request.Context(), c.Param("id"), adminID.String(), req.RecipientRole, req.Content)
	if err != nil {
		messagingActionError(c, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": m})
}

func messagingActionError(c *gin.Context, err error) {
	switch err {
	case services.ErrMessageNotFound:
		c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
	case services.ErrMessageNotPending:
		c.JSON(http.StatusConflict, gin.H{"error": "Message is no longer pending"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Something went wrong"})
	}
}
