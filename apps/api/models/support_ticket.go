package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TicketCategory string

const (
	TicketCategoryOrderIssue         TicketCategory = "order_issue"
	TicketCategoryPaymentIssue       TicketCategory = "payment_issue"
	TicketCategoryAccountIssue       TicketCategory = "account_issue"
	TicketCategoryChefComplaint      TicketCategory = "chef_complaint"
	TicketCategoryDeliveryComplaint  TicketCategory = "delivery_complaint"
	TicketCategoryTechnical          TicketCategory = "technical"
	TicketCategoryOther              TicketCategory = "other"
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

	// Relations
	Messages []SupportMessage `gorm:"foreignKey:TicketID" json:"messages,omitempty"`
	Reporter User             `gorm:"foreignKey:ReporterID" json:"reporter,omitempty"`
	Order    *Order           `gorm:"foreignKey:OrderID" json:"order,omitempty"`
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

	Sender User `gorm:"foreignKey:SenderID" json:"sender,omitempty"`
}
