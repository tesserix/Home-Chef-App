// Ticket materialisation for support chats. Shared by the slm-router hook
// endpoint and the queue-SLA workflow so both produce the same row and
// converge on one ticket per conversation.
package services

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/models"
)

// GenerateTicketNumber mints a human-quotable ticket reference (TKT-YYYYMM-NNNN).
func GenerateTicketNumber() string {
	return fmt.Sprintf("TKT-%s-%04d", time.Now().Format("200601"), rand.Intn(9999))
}

// ChatTicketInput describes the conversation a ticket is being raised for.
type ChatTicketInput struct {
	ConversationID string
	TenantID       string
	CaseID         string
	CustomerName   string
	CustomerEmail  string
	Subject        string
	Description    string
	Priority       models.TicketPriority
}

// CreateTicketFromConversation returns the existing ticket for the
// conversation, or creates one. The second return reports whether a row was
// created. Idempotent: support_tickets.conversation_id is uniquely indexed,
// so a concurrent caller loses the insert race and gets the winner back.
func CreateTicketFromConversation(db *gorm.DB, in ChatTicketInput) (*models.SupportTicket, bool, error) {
	convID := strings.TrimSpace(in.ConversationID)
	if convID == "" {
		return nil, false, gorm.ErrInvalidValue
	}

	var existing models.SupportTicket
	err := db.Where("conversation_id = ?", convID).First(&existing).Error
	if err == nil {
		return &existing, false, nil
	}
	if err != gorm.ErrRecordNotFound {
		return nil, false, err
	}

	// Attribute to the platform account when the chat identity matches one.
	reporterID := uuid.Nil
	reporterRole := "customer"
	if in.TenantID == OttoTenantVendor {
		reporterRole = "chef"
	}
	if email := strings.ToLower(strings.TrimSpace(in.CustomerEmail)); email != "" {
		var user models.User
		if db.Where("LOWER(email) = ?", email).First(&user).Error == nil {
			reporterID = user.ID
			switch user.Role {
			case models.RoleChef:
				reporterRole = "chef"
			case models.RoleDelivery:
				reporterRole = "delivery"
			}
		}
	}

	subject := strings.TrimSpace(in.Subject)
	if subject == "" {
		subject = "Support chat follow-up"
	}
	if len(subject) > 200 {
		subject = subject[:200]
	}
	description := strings.TrimSpace(in.Description)
	if description == "" {
		description = "(no transcript provided)"
	}
	if len(description) > 5000 {
		description = description[:5000]
	}
	filtered, _, _ := FilterChatMessage(description)

	priority := in.Priority
	if priority == "" {
		priority = models.TicketPriorityMedium
	}

	ticket := models.SupportTicket{
		TicketNumber:   GenerateTicketNumber(),
		ReporterID:     reporterID,
		ReporterRole:   reporterRole,
		Category:       models.TicketCategoryOther,
		Priority:       priority,
		Status:         models.TicketStatusOpen,
		Subject:        subject,
		Description:    filtered,
		ConversationID: &convID,
	}
	if err := db.Create(&ticket).Error; err != nil {
		// Unique-index race: return whoever won.
		if dbErr := db.Where("conversation_id = ?", convID).First(&existing).Error; dbErr == nil {
			return &existing, false, nil
		}
		return nil, false, err
	}
	return &ticket, true, nil
}
