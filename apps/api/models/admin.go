package models

import (
	"time"

	"github.com/google/uuid"
)

type PlatformSettings struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Key       string    `gorm:"uniqueIndex;not null" json:"key"`
	Value     string    `gorm:"type:text" json:"value"`
	Type      string    `gorm:"default:'string'" json:"type"` // string, number, boolean, json
	UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updatedBy,omitempty"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

type AuditLog struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     *uuid.UUID `gorm:"type:uuid;index" json:"userId,omitempty"`
	Action     string    `gorm:"not null" json:"action"`
	EntityType string    `gorm:"" json:"entityType"`
	EntityID   string    `gorm:"" json:"entityId"`
	OldValue   string    `gorm:"type:text" json:"oldValue,omitempty"`
	NewValue   string    `gorm:"type:text" json:"newValue,omitempty"`
	IPAddress  string    `gorm:"" json:"ipAddress"`
	UserAgent  string    `gorm:"" json:"userAgent"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`

	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Notification struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"userId"`
	Type      string     `gorm:"not null" json:"type"` // order_update, new_order, review, promo, system
	Title     string     `gorm:"not null" json:"title"`
	Message   string     `gorm:"type:text" json:"message"`
	Data      string     `gorm:"type:jsonb" json:"data,omitempty"` // JSON data for deep linking
	IsRead    bool       `gorm:"default:false" json:"isRead"`
	ReadAt    *time.Time `gorm:"" json:"readAt,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"createdAt"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}

type Transaction struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	OrderID         *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`
	Type            string    `gorm:"not null" json:"type"` // payment, refund, payout, fee
	Amount          float64   `gorm:"not null" json:"amount"`
	Currency        string    `gorm:"default:'USD'" json:"currency"`
	Status          string    `gorm:"default:'pending'" json:"status"` // pending, completed, failed
	StripeID        string    `gorm:"" json:"-"`
	Description     string    `gorm:"" json:"description"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"createdAt"`

	User  User   `gorm:"foreignKey:UserID" json:"-"`
	Order *Order `gorm:"foreignKey:OrderID" json:"-"`
}

// Admin Dashboard DTOs
type AdminDashboardStats struct {
	TotalUsers           int     `json:"totalUsers"`
	NewUsersToday        int     `json:"newUsersToday"`
	TotalChefs           int     `json:"totalChefs"`
	PendingVerifications int     `json:"pendingVerifications"`
	TotalOrders          int     `json:"totalOrders"`
	OrdersToday          int     `json:"ordersToday"`
	Revenue              float64 `json:"revenue"`
	RevenueToday         float64 `json:"revenueToday"`
	RevenueChange        float64 `json:"revenueChange"`
	OrdersChange         float64 `json:"ordersChange"`
}

type AdminAnalytics struct {
	Overview struct {
		TotalRevenue  float64 `json:"totalRevenue"`
		RevenueChange float64 `json:"revenueChange"`
		TotalOrders   int     `json:"totalOrders"`
		OrdersChange  float64 `json:"ordersChange"`
		AvgOrderValue float64 `json:"avgOrderValue"`
		AOVChange     float64 `json:"aovChange"`
		ActiveUsers   int     `json:"activeUsers"`
		UsersChange   float64 `json:"usersChange"`
	} `json:"overview"`
	TopChefs []struct {
		ID      uuid.UUID `json:"id"`
		Name    string    `json:"name"`
		Orders  int       `json:"orders"`
		Revenue float64   `json:"revenue"`
	} `json:"topChefs"`
	TopCuisines []struct {
		Name       string  `json:"name"`
		Orders     int     `json:"orders"`
		Percentage float64 `json:"percentage"`
	} `json:"topCuisines"`
	OrdersByStatus map[string]int `json:"ordersByStatus"`
	RevenueByDay   []struct {
		Date    string  `json:"date"`
		Revenue float64 `json:"revenue"`
		Orders  int     `json:"orders"`
	} `json:"revenueByDay"`
}

// Notification DTOs
type NotificationResponse struct {
	ID        uuid.UUID  `json:"id"`
	Type      string     `json:"type"`
	Title     string     `json:"title"`
	Message   string     `json:"message"`
	Data      string     `json:"data,omitempty"`
	IsRead    bool       `json:"isRead"`
	CreatedAt time.Time  `json:"createdAt"`
}

func (n *Notification) ToResponse() NotificationResponse {
	return NotificationResponse{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Message:   n.Message,
		Data:      n.Data,
		IsRead:    n.IsRead,
		CreatedAt: n.CreatedAt,
	}
}
