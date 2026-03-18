package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusAccepted   OrderStatus = "accepted"
	OrderStatusPreparing  OrderStatus = "preparing"
	OrderStatusReady      OrderStatus = "ready"
	OrderStatusPickedUp   OrderStatus = "picked_up"
	OrderStatusDelivering OrderStatus = "delivering"
	OrderStatusDelivered  OrderStatus = "delivered"
	OrderStatusCancelled  OrderStatus = "cancelled"
	OrderStatusRefunded   OrderStatus = "refunded"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentCompleted PaymentStatus = "completed"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
)

type Order struct {
	ID            uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderNumber   string        `gorm:"uniqueIndex;not null" json:"orderNumber"`
	CustomerID    uuid.UUID     `gorm:"type:uuid;not null;index" json:"customerId"`
	ChefID        uuid.UUID     `gorm:"type:uuid;not null;index" json:"chefId"`
	DeliveryID    *uuid.UUID    `gorm:"type:uuid;index" json:"deliveryId,omitempty"`
	Status        OrderStatus   `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentStatus PaymentStatus `gorm:"type:varchar(20);default:'pending'" json:"paymentStatus"`
	PaymentMethod string        `gorm:"" json:"paymentMethod"`

	// Pricing
	Subtotal    float64 `gorm:"not null" json:"subtotal"`
	DeliveryFee float64 `gorm:"default:0" json:"deliveryFee"`
	ServiceFee  float64 `gorm:"default:0" json:"serviceFee"`
	Tax         float64 `gorm:"default:0" json:"tax"`
	Tip         float64 `gorm:"default:0" json:"tip"`         // Legacy: total tip (kept for backward compat)
	ChefTip     float64 `gorm:"default:0" json:"chefTip"`     // Tip for the chef/kitchen
	DriverTip   float64 `gorm:"default:0" json:"driverTip"`   // Tip for the delivery driver
	Discount    float64 `gorm:"default:0" json:"discount"`
	Total       float64 `gorm:"not null" json:"total"`
	PromoCode   string  `gorm:"" json:"promoCode,omitempty"`

	// Delivery Address
	DeliveryAddressLine1      string  `gorm:"" json:"deliveryAddressLine1"`
	DeliveryAddressLine2      string  `gorm:"" json:"deliveryAddressLine2"`
	DeliveryAddressCity       string  `gorm:"" json:"deliveryAddressCity"`
	DeliveryAddressState      string  `gorm:"" json:"deliveryAddressState"`
	DeliveryAddressPostalCode string  `gorm:"" json:"deliveryAddressPostalCode"`
	DeliveryLatitude          float64 `gorm:"" json:"deliveryLatitude"`
	DeliveryLongitude         float64 `gorm:"" json:"deliveryLongitude"`
	DeliveryInstructions      string  `gorm:"type:text" json:"deliveryInstructions"`

	// Timing
	EstimatedPrepTime     int        `gorm:"" json:"estimatedPrepTime"` // minutes
	EstimatedDeliveryTime int        `gorm:"" json:"estimatedDeliveryTime"`
	ScheduledFor          *time.Time `gorm:"" json:"scheduledFor,omitempty"`
	AcceptedAt            *time.Time `gorm:"" json:"acceptedAt,omitempty"`
	PreparedAt            *time.Time `gorm:"" json:"preparedAt,omitempty"`
	PickedUpAt            *time.Time `gorm:"" json:"pickedUpAt,omitempty"`
	DeliveredAt           *time.Time `gorm:"" json:"deliveredAt,omitempty"`
	CancelledAt           *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason          string     `gorm:"" json:"cancelReason,omitempty"`

	// Special Instructions
	SpecialInstructions string `gorm:"type:text" json:"specialInstructions"`

	// Payment gateway
	StripePaymentIntentID string `gorm:"" json:"-"`          // Legacy Stripe
	RazorpayOrderID       string `gorm:"" json:"-"`          // Razorpay order ID
	RazorpayPaymentID     string `gorm:"" json:"-"`          // Razorpay payment ID
	RefundID              string `gorm:"" json:"-"`          // Razorpay refund ID (if refunded)
	RefundedAt            *time.Time `gorm:"" json:"refundedAt,omitempty"`
	RefundAmount          float64 `gorm:"default:0" json:"refundAmount"`
	RefundReason          string  `gorm:"" json:"refundReason,omitempty"`
	RefundInitiatedBy     string  `gorm:"type:varchar(20)" json:"refundInitiatedBy,omitempty"` // chef, admin, system

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	Customer Customer    `gorm:"foreignKey:CustomerID;references:ID" json:"customer,omitempty"`
	Chef     ChefProfile `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
	Items    []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"`
	Delivery *Delivery   `gorm:"foreignKey:OrderID" json:"delivery,omitempty"`
	Review   *Review     `gorm:"foreignKey:OrderID" json:"review,omitempty"`
}

// Customer is an alias for User for clarity in Order relationships
type Customer = User

type OrderItem struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID    uuid.UUID `gorm:"type:uuid;not null;index" json:"orderId"`
	MenuItemID uuid.UUID `gorm:"type:uuid;not null" json:"menuItemId"`
	Name       string    `gorm:"not null" json:"name"`
	Price      float64   `gorm:"not null" json:"price"`
	Quantity   int       `gorm:"not null" json:"quantity"`
	Subtotal   float64   `gorm:"not null" json:"subtotal"`
	Notes      string    `gorm:"" json:"notes,omitempty"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`

	Order    Order    `gorm:"foreignKey:OrderID" json:"-"`
	MenuItem MenuItem `gorm:"foreignKey:MenuItemID" json:"menuItem,omitempty"`
}

// DTOs
type OrderResponse struct {
	ID              uuid.UUID              `json:"id"`
	OrderNumber     string                 `json:"orderNumber"`
	Status          OrderStatus            `json:"status"`
	PaymentStatus   PaymentStatus          `json:"paymentStatus"`
	Subtotal        float64                `json:"subtotal"`
	DeliveryFee     float64                `json:"deliveryFee"`
	ServiceFee      float64                `json:"serviceFee"`
	Tax             float64                `json:"tax"`
	Tip             float64                `json:"tip"`
	Discount        float64                `json:"discount"`
	Total           float64                `json:"total"`
	Items           []OrderItemResponse    `json:"items"`
	DeliveryAddress AddressResponse        `json:"deliveryAddress"`
	CreatedAt       time.Time              `json:"createdAt"`
}

type OrderItemResponse struct {
	ID         uuid.UUID `json:"id"`
	MenuItemID uuid.UUID `json:"menuItemId"`
	Name       string    `json:"name"`
	Price      float64   `json:"price"`
	Quantity   int       `json:"quantity"`
	Subtotal   float64   `json:"subtotal"`
	Notes      string    `json:"notes,omitempty"`
}

type AddressResponse struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
}

func (o *Order) ToResponse() OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			ID:         item.ID,
			MenuItemID: item.MenuItemID,
			Name:       item.Name,
			Price:      item.Price,
			Quantity:   item.Quantity,
			Subtotal:   item.Subtotal,
			Notes:      item.Notes,
		}
	}

	return OrderResponse{
		ID:            o.ID,
		OrderNumber:   o.OrderNumber,
		Status:        o.Status,
		PaymentStatus: o.PaymentStatus,
		Subtotal:      o.Subtotal,
		DeliveryFee:   o.DeliveryFee,
		ServiceFee:    o.ServiceFee,
		Tax:           o.Tax,
		Tip:           o.Tip,
		Discount:      o.Discount,
		Total:         o.Total,
		Items:         items,
		DeliveryAddress: AddressResponse{
			Line1:      o.DeliveryAddressLine1,
			Line2:      o.DeliveryAddressLine2,
			City:       o.DeliveryAddressCity,
			State:      o.DeliveryAddressState,
			PostalCode: o.DeliveryAddressPostalCode,
		},
		CreatedAt: o.CreatedAt,
	}
}
