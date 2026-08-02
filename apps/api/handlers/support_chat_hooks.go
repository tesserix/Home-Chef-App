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
	"github.com/google/uuid"
	"gorm.io/gorm"

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

	// Idempotency: one ticket per conversation, ever.
	var existing models.SupportTicket
	err := database.DB.Where("conversation_id = ?", req.ConversationID).First(&existing).Error
	if err == nil {
		c.JSON(http.StatusOK, existing)
		return
	}
	if err != gorm.ErrRecordNotFound {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup failed"})
		return
	}

	// Attribute to the platform user when the chat identity matches one.
	reporterID := uuid.Nil
	reporterRole := "customer"
	if req.TenantID == services.OttoTenantVendor {
		reporterRole = "chef"
	}
	if email := strings.ToLower(strings.TrimSpace(req.CustomerEmail)); email != "" {
		var user models.User
		if err := database.DB.Where("LOWER(email) = ?", email).First(&user).Error; err == nil {
			reporterID = user.ID
			switch user.Role {
			case models.RoleChef:
				reporterRole = "chef"
			case models.RoleDelivery:
				reporterRole = "delivery"
			}
		}
	}

	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = "Support chat escalation"
	}
	if len(subject) > 200 {
		subject = subject[:200]
	}
	description := strings.TrimSpace(req.Description)
	if description == "" {
		description = "(no transcript provided)"
	}
	if len(description) > 5000 {
		description = description[:5000]
	}
	filteredDesc, _, _ := services.FilterChatMessage(description)

	convID := req.ConversationID
	ticket := models.SupportTicket{
		TicketNumber:   generateTicketNumber(),
		ReporterID:     reporterID,
		ReporterRole:   reporterRole,
		Category:       models.TicketCategoryOther,
		Priority:       models.TicketPriorityHigh,
		Status:         models.TicketStatusOpen,
		Subject:        subject,
		Description:    filteredDesc,
		ConversationID: &convID,
	}
	if req.EscalationReason == "" {
		// chat_started hook (not an escalation) — a plain record, not urgent.
		ticket.Priority = models.TicketPriorityMedium
	}

	if err := database.DB.Create(&ticket).Error; err != nil {
		// Unique-index race with a concurrent hook retry: return the winner.
		if dbErr := database.DB.Where("conversation_id = ?", req.ConversationID).
			First(&existing).Error; dbErr == nil {
			c.JSON(http.StatusOK, existing)
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ticket"})
		return
	}
	c.JSON(http.StatusCreated, ticket)
}
