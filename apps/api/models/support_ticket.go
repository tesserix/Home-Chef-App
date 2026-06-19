package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketCategory string

const (
	TicketCategoryOrderIssue        TicketCategory = "order_issue"
	TicketCategoryPaymentIssue      TicketCategory = "payment_issue"
	TicketCategoryAccountIssue      TicketCategory = "account_issue"
	TicketCategoryChefComplaint     TicketCategory = "chef_complaint"
	TicketCategoryDeliveryComplaint TicketCategory = "delivery_complaint"
	TicketCategoryTechnical         TicketCategory = "technical"
	TicketCategoryOther             TicketCategory = "other"
)

type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "low"
	TicketPriorityMedium TicketPriority = "medium"
	TicketPriorityHigh   TicketPriority = "high"
	TicketPriorityUrgent TicketPriority = "urgent"
)

type TicketStatus string

const (
	TicketStatusOpen              TicketStatus = "open"
	TicketStatusInProgress        TicketStatus = "in_progress"
	TicketStatusWaitingOnCustomer TicketStatus = "waiting_on_customer"
	TicketStatusWaitingOnChef     TicketStatus = "waiting_on_chef"
	TicketStatusResolved          TicketStatus = "resolved"
	TicketStatusClosed            TicketStatus = "closed"
)

type SupportTicket struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TicketNumber string         `gorm:"uniqueIndex;not null" json:"ticketNumber"`
	ReporterID   uuid.UUID      `gorm:"type:uuid;not null;index" json:"reporterId"`
	ReporterRole string         `gorm:"type:varchar(20);not null" json:"reporterRole"`
	AssignedToID *uuid.UUID     `gorm:"type:uuid;index" json:"assignedToId,omitempty"`
	OrderID      *uuid.UUID     `gorm:"type:uuid;index" json:"orderId,omitempty"`
	Category     TicketCategory `gorm:"type:varchar(30);not null" json:"category"`
	Priority     TicketPriority `gorm:"type:varchar(10);default:'medium'" json:"priority"`
	Status       TicketStatus   `gorm:"type:varchar(30);default:'open'" json:"status"`
	Subject      string         `gorm:"not null" json:"subject"`
	Description  string         `gorm:"type:text;not null" json:"description"`
	Resolution   string         `gorm:"type:text" json:"resolution,omitempty"`
	ResolvedAt   *time.Time     `gorm:"" json:"resolvedAt,omitempty"`
	ClosedAt     *time.Time     `gorm:"" json:"closedAt,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relations. Reporter and Messages.Sender are NEVER serialized as raw
	// User structs — that leaks the reporter's email/phone (when staff read
	// it) or the staff's email/phone (when the customer reads it). Handlers
	// must serialize via ToResponse(includeInternal).
	Messages []SupportMessage `gorm:"foreignKey:TicketID" json:"-"`
	Reporter User             `gorm:"foreignKey:ReporterID" json:"-"`
	Order    *Order           `gorm:"foreignKey:OrderID" json:"-"`
}

type SupportMessage struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TicketID    uuid.UUID `gorm:"type:uuid;not null;index" json:"ticketId"`
	SenderID    uuid.UUID `gorm:"type:uuid;not null" json:"senderId"`
	SenderRole  string    `gorm:"type:varchar(20);not null" json:"senderRole"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	PIIDetected bool      `gorm:"default:false" json:"piiDetected"`
	IsInternal  bool      `gorm:"default:false" json:"isInternal"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`

	// Sender is loadable for handlers that need first/last name to render
	// the message attribution, but it's NEVER serialized as-is — that
	// would leak the sender's email, phone, lastLoginAt, totpEnabled, etc.
	// to the ticket reporter (a customer reading admin replies). Use
	// SupportMessageResponse to expose only the fields a UI needs.
	Sender User `gorm:"foreignKey:SenderID" json:"-"`
}

// SupportMessageResponse is the safe DTO for a message in a ticket. Senders
// are identified by role + name only — no email, no phone, no auth metadata.
type SupportMessageResponse struct {
	ID         uuid.UUID `json:"id"`
	TicketID   uuid.UUID `json:"ticketId"`
	SenderID   uuid.UUID `json:"senderId"`
	SenderRole string    `json:"senderRole"`
	SenderName string    `json:"senderName,omitempty"`
	Content    string    `json:"content"`
	IsInternal bool      `json:"isInternal"`
	CreatedAt  time.Time `json:"createdAt"`
}

// ToResponse strips PII from the embedded Sender User. SenderName is the
// display name only — first + last, no email or other identifiers.
func (m *SupportMessage) ToResponse() SupportMessageResponse {
	name := strings.TrimSpace(m.Sender.FirstName + " " + m.Sender.LastName)
	return SupportMessageResponse{
		ID:         m.ID,
		TicketID:   m.TicketID,
		SenderID:   m.SenderID,
		SenderRole: m.SenderRole,
		SenderName: name,
		Content:    m.Content,
		IsInternal: m.IsInternal,
		CreatedAt:  m.CreatedAt,
	}
}

// SupportTicketResponse is the safe DTO for a whole ticket including
// messages. Reporter is similarly stripped to a name + role.
type SupportTicketResponse struct {
	ID           uuid.UUID                `json:"id"`
	TicketNumber string                   `json:"ticketNumber,omitempty"`
	Subject      string                   `json:"subject"`
	Description  string                   `json:"description,omitempty"`
	Category     TicketCategory           `json:"category,omitempty"`
	Priority     TicketPriority           `json:"priority,omitempty"`
	Status       TicketStatus             `json:"status,omitempty"`
	ReporterID   uuid.UUID                `json:"reporterId"`
	ReporterName string                   `json:"reporterName,omitempty"`
	ReporterRole string                   `json:"reporterRole,omitempty"`
	OrderID      *uuid.UUID               `json:"orderId,omitempty"`
	AssignedToID *uuid.UUID               `json:"assignedToId,omitempty"`
	CreatedAt    time.Time                `json:"createdAt"`
	UpdatedAt    time.Time                `json:"updatedAt"`
	ResolvedAt   *time.Time               `json:"resolvedAt,omitempty"`
	ClosedAt     *time.Time               `json:"closedAt,omitempty"`
	Messages     []SupportMessageResponse `json:"messages,omitempty"`
}

// ToResponse for SupportTicket. Includes messages and reporter as DTO.
// includeInternal=true keeps admin notes; false strips them so the
// reporter doesn't see internal-only commentary.
func (t *SupportTicket) ToResponse(includeInternal bool) SupportTicketResponse {
	out := SupportTicketResponse{
		ID:           t.ID,
		TicketNumber: t.TicketNumber,
		Subject:      t.Subject,
		Description:  t.Description,
		Category:     t.Category,
		Priority:     t.Priority,
		Status:       t.Status,
		ReporterID:   t.ReporterID,
		ReporterName: strings.TrimSpace(t.Reporter.FirstName + " " + t.Reporter.LastName),
		ReporterRole: string(t.Reporter.Role),
		OrderID:      t.OrderID,
		AssignedToID: t.AssignedToID,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
		ResolvedAt:   t.ResolvedAt,
		ClosedAt:     t.ClosedAt,
	}
	for _, m := range t.Messages {
		if !includeInternal && m.IsInternal {
			continue
		}
		out.Messages = append(out.Messages, m.ToResponse())
	}
	return out
}
