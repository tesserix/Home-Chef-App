// Package handlers — support_chat_hooks.go: slm-router's per-tenant
// escalation hook lands here (see slm-support-platform docs, EscalationHook).
// When Otto hands a chat off to a human, slm-router POSTs the conversation
// context and this endpoint materialises a durable SupportTicket, so the
// escalation is visible in the admin Support section even if the customer
// drops off the chat. Idempotent on conversation_id — retries and the
// user-initiated "create ticket from chat" path converge on one ticket.
package handlers

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// SupportChatHookHandler receives slm-router hook callbacks.
type SupportChatHookHandler struct{}

func NewSupportChatHookHandler() *SupportChatHookHandler {
	return &SupportChatHookHandler{}
}

// hookPayload is slm-router's fixed hook shape (escalation_reason empty for
// chat_started hooks — both kinds share this endpoint's parsing either way).
type hookPayload struct {
	ConversationID   string `json:"conversation_id" binding:"required"`
	TenantID         string `json:"tenant_id"`
	StoreID          string `json:"store_id"`
	CustomerName     string `json:"customer_name"`
	CustomerEmail    string `json:"customer_email"`
	Subject          string `json:"subject"`
	Description      string `json:"description"`
	EscalationReason string `json:"escalation_reason"`
}

// FromConversation creates (or returns) the ticket for an escalated chat.
// POST /internal/support/tickets/from-conversation
func (h *SupportChatHookHandler) FromConversation(c *gin.Context) {
	secret := config.AppConfig.SupportHookSecret
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "hook_disabled"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(c.GetHeader("X-Internal-Auth")), []byte(secret)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req hookPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	priority := models.TicketPriorityHigh
	if req.EscalationReason == "" {
		// chat_started hook (not an escalation) — a plain record, not urgent.
		priority = models.TicketPriorityMedium
	}
	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = "Support chat escalation"
	}

	ticket, created, err := services.CreateTicketFromConversation(database.DB, services.ChatTicketInput{
		ConversationID: req.ConversationID,
		TenantID:       req.TenantID,
		CustomerName:   req.CustomerName,
		CustomerEmail:  req.CustomerEmail,
		Subject:        subject,
		Description:    req.Description,
		Priority:       priority,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket"})
		return
	}
	if !created {
		c.JSON(http.StatusOK, ticket)
		return
	}
	c.JSON(http.StatusCreated, ticket)
}
