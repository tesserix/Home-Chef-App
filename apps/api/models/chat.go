package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ChatRoomType constants
const (
	ChatRoomCustomerChef     = "customer_chef"
	ChatRoomCustomerDelivery = "customer_delivery"
)

// ChatRoomStatus constants
const (
	ChatRoomActive = "active"
	ChatRoomClosed = "closed"
)

// ChatRoom represents a one-to-one chat linked to a specific order.
// Types: "customer_chef" (customer <-> chef), "customer_delivery" (customer <-> delivery agent)
type ChatRoom struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"orderId"`
	Type           string     `gorm:"type:varchar(30);not null" json:"type"`
	CustomerID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"customerId"`
	CounterpartyID uuid.UUID  `gorm:"type:uuid;not null;index" json:"counterpartyId"`
	Status         string     `gorm:"type:varchar(20);default:'active'" json:"status"`
	LastMessageAt  *time.Time `gorm:"" json:"lastMessageAt,omitempty"`
	CreatedAt      time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt      time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`

	// Relationships
	Order        Order         `gorm:"foreignKey:OrderID" json:"-"`
	Customer     User          `gorm:"foreignKey:CustomerID" json:"-"`
	Counterparty User          `gorm:"foreignKey:CounterpartyID" json:"-"`
	Messages     []ChatMessage `gorm:"foreignKey:ChatRoomID" json:"-"`
}

// ChatMessage represents a single message within a chat room.
type ChatMessage struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ChatRoomID     uuid.UUID      `gorm:"type:uuid;not null;index" json:"chatRoomId"`
	SenderID       uuid.UUID      `gorm:"type:uuid;not null;index" json:"senderId"`
	SenderRole     string         `gorm:"type:varchar(20);not null" json:"senderRole"`
	Content        string         `gorm:"type:text;not null" json:"content"`
	OriginalLength int            `gorm:"not null" json:"-"`
	PIIDetected    bool           `gorm:"default:false" json:"piiDetected"`
	PIIViolations  pq.StringArray `gorm:"type:text[]" json:"-"`
	MessageType    string         `gorm:"type:varchar(20);default:'text'" json:"messageType"`
	IsRead         bool           `gorm:"default:false" json:"isRead"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"createdAt"`

	// Relationships
	ChatRoom ChatRoom `gorm:"foreignKey:ChatRoomID" json:"-"`
	Sender   User     `gorm:"foreignKey:SenderID" json:"-"`
}

// ---------- Response DTOs ----------

// ChatRoomResponse is the safe JSON representation of a chat room.
type ChatRoomResponse struct {
	ID               uuid.UUID  `json:"id"`
	OrderID          uuid.UUID  `json:"orderId"`
	Type             string     `json:"type"`
	Status           string     `json:"status"`
	LastMessageAt    *time.Time `json:"lastMessageAt,omitempty"`
	CounterpartyName string     `json:"counterpartyName"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// ChatMessageResponse is the safe JSON representation of a chat message.
type ChatMessageResponse struct {
	ID          uuid.UUID `json:"id"`
	ChatRoomID  uuid.UUID `json:"chatRoomId"`
	SenderID    uuid.UUID `json:"senderId"`
	SenderRole  string    `json:"senderRole"`
	Content     string    `json:"content"`
	PIIDetected bool      `json:"piiDetected"`
	MessageType string    `json:"messageType"`
	IsRead      bool      `json:"isRead"`
	CreatedAt   time.Time `json:"createdAt"`
}

// ToResponse converts a ChatRoom to its safe API response.
func (r *ChatRoom) ToResponse() ChatRoomResponse {
	counterpartyName := ""
	if r.Counterparty.ID != uuid.Nil {
		counterpartyName = r.Counterparty.FirstName
		if r.Counterparty.LastName != "" {
			counterpartyName += " " + r.Counterparty.LastName
		}
	}

	return ChatRoomResponse{
		ID:               r.ID,
		OrderID:          r.OrderID,
		Type:             r.Type,
		Status:           r.Status,
		LastMessageAt:    r.LastMessageAt,
		CounterpartyName: counterpartyName,
		CreatedAt:        r.CreatedAt,
	}
}

// ToResponse converts a ChatMessage to its safe API response.
// The content field always holds the sanitized version; original content is never exposed.
func (m *ChatMessage) ToResponse() ChatMessageResponse {
	return ChatMessageResponse{
		ID:          m.ID,
		ChatRoomID:  m.ChatRoomID,
		SenderID:    m.SenderID,
		SenderRole:  m.SenderRole,
		Content:     m.Content,
		PIIDetected: m.PIIDetected,
		MessageType: m.MessageType,
		IsRead:      m.IsRead,
		CreatedAt:   m.CreatedAt,
	}
}
