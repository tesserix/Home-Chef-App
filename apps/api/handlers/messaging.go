package handlers

// messaging.go — HTTP surface for admin-mediated in-app messaging (#53/#303).
// Customer and chef each message via an admin relay (no direct channel); admins
// run the mediation inbox (relay / block / direct send). Storage is MongoDB, so
// every endpoint 503s gracefully when Mongo is unavailable.

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

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

// ── Admin (audit read, #312) ────────────────────────────────────────────────

// conversationFilterFromQuery reads orderId/customerId/chefId/status/from/to.
func conversationFilterFromQuery(c *gin.Context) services.ConversationFilter {
	f := services.ConversationFilter{
		OrderID:    c.Query("orderId"),
		CustomerID: c.Query("customerId"),
		ChefID:     c.Query("chefId"),
		Status:     c.Query("status"),
	}
	if t, ok := parseAuditTime(c.Query("from")); ok {
		f.From = &t
	}
	if t, ok := parseAuditTime(c.Query("to")); ok {
		f.To = &t
	}
	return f
}

// parseAuditTime accepts RFC3339 or a plain YYYY-MM-DD date.
func parseAuditTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// AdminListConversations — GET /admin/conversations?orderId=&customerId=&chefId=&status=&from=&to=&limit=&offset=
// Lists every conversation for audit, newest-activity first.
func (h *MessagingHandler) AdminListConversations(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	convs, total, err := svc.AdminListConversations(c.Request.Context(), conversationFilterFromQuery(c), limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load conversations"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": convs, "total": total, "limit": limit, "offset": offset})
}

// AdminConversationTranscript — GET /admin/conversations/:id
// Returns the conversation plus its complete message history (all statuses).
func (h *MessagingHandler) AdminConversationTranscript(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	conv, msgs, err := svc.AdminTranscript(c.Request.Context(), c.Param("id"))
	if err != nil {
		messagingActionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"conversation": conv, "messages": msgs})
}

// AdminExportConversation — GET /admin/conversations/:id/export?format=json|csv
// Streams the full transcript as a downloadable audit record.
func (h *MessagingHandler) AdminExportConversation(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	conv, msgs, err := svc.AdminTranscript(c.Request.Context(), c.Param("id"))
	if err != nil {
		messagingActionError(c, err)
		return
	}
	if c.DefaultQuery("format", "json") == "csv" {
		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", "attachment; filename=\"conversation-"+conv.ID+".csv\"")
		w := csv.NewWriter(c.Writer)
		_ = w.Write([]string{"createdAt", "senderRole", "recipientRole", "relayStatus", "piiDetected", "content", "attachment"})
		for _, m := range msgs {
			pii := "false"
			if m.PIIDetected {
				pii = "true"
			}
			_ = w.Write([]string{
				m.CreatedAt.UTC().Format(time.RFC3339), m.SenderRole, m.RecipientRole,
				m.RelayStatus, pii, csvSafe(m.Content), csvSafe(m.Filename),
			})
		}
		w.Flush()
		return
	}
	c.Header("Content-Disposition", "attachment; filename=\"conversation-"+conv.ID+".json\"")
	c.JSON(http.StatusOK, gin.H{"conversation": conv, "messages": msgs})
}

// ── Attachments (#304) ──────────────────────────────────────────────────────

// uploadAttachment is the shared upload path for customer + chef. It validates,
// stores the file in GridFS, and records a pending attachment message.
func (h *MessagingHandler) uploadAttachment(c *gin.Context, role string) {
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
	custID, chefID, ok := orderParticipants(c, orderID, userID, role)
	if !ok {
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "A file is required"})
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	if !services.AllowedChatAttachmentType(contentType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unsupported file type (images or PDF only)"})
		return
	}
	if header.Size > services.MaxChatAttachmentBytes() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large (max 10 MB)"})
		return
	}
	attachmentID, err := services.UploadChatAttachment(c.Request.Context(), header.Filename, contentType, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload attachment"})
		return
	}
	caption := c.PostForm("caption")
	var m *services.MediatedMessage
	if role == services.MsgRoleChef {
		m, err = svc.ChefSendAttachment(c.Request.Context(), orderID.String(), custID, chefID, attachmentID, header.Filename, contentType, caption)
	} else {
		m, err = svc.CustomerSendAttachment(c.Request.Context(), orderID.String(), custID, chefID, attachmentID, header.Filename, contentType, caption)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to attach file"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": m})
}

// CustomerUploadAttachment — POST /customer/orders/:id/attachments (multipart: file, caption?)
func (h *MessagingHandler) CustomerUploadAttachment(c *gin.Context) {
	h.uploadAttachment(c, services.MsgRoleCustomer)
}

// ChefUploadAttachment — POST /chef/orders/:orderId/attachments
func (h *MessagingHandler) ChefUploadAttachment(c *gin.Context) {
	h.uploadAttachment(c, services.MsgRoleChef)
}

// DownloadAttachment — GET /chat/attachments/:id. Streams the file when the
// requester is authorized (admin always; a participant for their own or a
// relayed message).
func (h *MessagingHandler) DownloadAttachment(c *gin.Context) {
	svc := svcOr503(c)
	if svc == nil {
		return
	}
	userID, _ := middleware.GetUserID(c)
	var user models.User
	if err := database.DB.Select("id", "role").First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	attachmentID := c.Param("id")
	msg, ok, err := svc.AuthorizeAttachmentDownload(c.Request.Context(), attachmentID, userID.String(), string(user.Role))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	if !ok {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not authorized to view this attachment"})
		return
	}
	c.Header("Content-Type", msg.ContentType)
	// Stop the browser from MIME-sniffing the payload into something executable,
	// and force non-image types (e.g. PDF) to download rather than render inline.
	c.Header("X-Content-Type-Options", "nosniff")
	disposition := "attachment"
	if services.IsImageContentType(msg.ContentType) {
		disposition = "inline"
	}
	c.Header("Content-Disposition", disposition+"; filename=\""+msg.Filename+"\"")
	c.Header("Cache-Control", "private, no-store")
	if err := services.DownloadChatAttachment(c.Request.Context(), attachmentID, c.Writer); err != nil {
		// Headers may already be flushed; nothing more we can do but log upstream.
		return
	}
}

// csvSafe neutralises CSV formula injection: a cell that begins with a formula
// trigger (= + - @, tab or carriage return) is prefixed with a single quote so
// spreadsheet apps treat it as literal text instead of evaluating it.
func csvSafe(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		return "'" + s
	}
	return s
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
