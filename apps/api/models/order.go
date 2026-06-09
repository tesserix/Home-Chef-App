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
	// TaxRate / TaxName freeze the rule applied when the order was placed so
	// that later edits to TaxRate rows don't retroactively change historical
	// invoices. TaxName is the label shown on the invoice ("GST", "VAT", ...).
	TaxRate float64 `gorm:"default:0" json:"taxRate"`
	TaxName string  `gorm:"type:varchar(40);default:''" json:"taxName"`
	Tip     float64 `gorm:"default:0" json:"tip"`       // Legacy: total tip (kept for backward compat)
	ChefTip float64 `gorm:"default:0" json:"chefTip"`   // Tip for the chef/kitchen
	DriverTip   float64 `gorm:"default:0" json:"driverTip"`   // Tip for the delivery driver
	Discount    float64 `gorm:"default:0" json:"discount"`
	Total       float64 `gorm:"not null" json:"total"`
	PromoCode   string  `gorm:"" json:"promoCode,omitempty"`
	// Currency is the 3-letter ISO code the customer is charged in. Frozen
	// at order creation from the chef's settlement currency so later edits
	// on the chef profile don't invalidate an in-flight payment.
	Currency string `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	// Delivery Address
	DeliveryAddressLine1      string  `gorm:"" json:"deliveryAddressLine1"`
	DeliveryAddressLine2      string  `gorm:"" json:"deliveryAddressLine2"`
	DeliveryAddressCity       string  `gorm:"" json:"deliveryAddressCity"`
	DeliveryAddressState      string  `gorm:"" json:"deliveryAddressState"`
	DeliveryAddressPostalCode string  `gorm:"" json:"deliveryAddressPostalCode"`
	// ISO-3166 alpha-2 country used to pick the tax rule and for invoicing.
	DeliveryAddressCountry string `gorm:"type:varchar(2);default:'IN'" json:"deliveryAddressCountry"`
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

	// Payment gateway. PaymentProvider records which gateway handled this
	// order so refunds, reconciliation, and webhooks read the correct
	// provider-specific ID below. Inherited from ChefProfile.PaymentProvider
	// at order creation time so late-switching a chef doesn't invalidate
	// already-placed orders.
	PaymentProvider       string `gorm:"type:varchar(20);default:'razorpay'" json:"paymentProvider"`
	StripePaymentIntentID string `gorm:"" json:"-"`          // Stripe PaymentIntent ID
	RazorpayOrderID       string `gorm:"" json:"-"`          // Razorpay order ID
	RazorpayPaymentID     string `gorm:"" json:"-"`          // Razorpay payment ID
	RefundID              string `gorm:"" json:"-"`          // Gateway refund ID (if refunded)
	RefundedAt            *time.Time `gorm:"" json:"refundedAt,omitempty"`
	RefundAmount          float64 `gorm:"default:0" json:"refundAmount"`
	RefundReason          string  `gorm:"" json:"refundReason,omitempty"`
	RefundInitiatedBy     string  `gorm:"type:varchar(20)" json:"refundInitiatedBy,omitempty"` // chef, admin, system

	CreatedAt time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// Customer/Chef are loadable for handlers that need to render names
	// or look up profile fields, but NEVER serialized as raw User/ChefProfile
	// structs. Inlining them previously leaked the customer's
	// EmailVerified/PhoneVerified/TOTPEnabled/AuthProvider/LastLoginAt to
	// any chef who fetched the order, and the chef's profile to the
	// customer. Handlers must add explicit name/email fields they want to
	// expose (the admin orders endpoint already does this via customerName).
	Customer Customer    `gorm:"foreignKey:CustomerID;references:ID" json:"-"`
	Chef     ChefProfile `gorm:"foreignKey:ChefID" json:"-"`
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

	// Per-line cancellation — set when the chef marks a single line item
	// as unfulfillable mid-prep. The order itself stays accepted /
	// preparing so the remaining lines continue; only this line's
	// subtotal (+ proportional tax) is refunded to the customer.
	// RefundID is the gateway refund identifier returned by Razorpay /
	// Stripe; used as the idempotency anchor so retries of the same
	// per-line cancel don't double-refund.
	IsCancelled     bool       `gorm:"default:false" json:"isCancelled"`
	CancelledReason string     `gorm:"type:varchar(40)" json:"cancelledReason,omitempty"`
	CancelledAt     *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	RefundID        string     `gorm:"" json:"-"`
	RefundAmount    float64    `gorm:"default:0" json:"refundAmount,omitempty"`

	Order    Order    `gorm:"foreignKey:OrderID" json:"-"`
	MenuItem MenuItem `gorm:"foreignKey:MenuItemID" json:"menuItem,omitempty"`
}

// CancelReason enumerates the reason values mobile + admin send when
// cancelling an order or per-line item. Kept narrow on purpose — open-
// ended free-text reasons read worse on the customer email + harder
// to bucket in analytics.
type CancelReason string

const (
	CancelReasonOutOfIngredient CancelReason = "out_of_ingredient"
	CancelReasonEquipmentFailure CancelReason = "equipment_failure"
	CancelReasonCustomerRequest  CancelReason = "customer_request"
	CancelReasonOther            CancelReason = "other"
)

// IsValid reports whether r is one of the four allowed cancel reasons.
// Handlers should reject anything else with 400 so the column stays
// queryable for ops dashboards.
func (r CancelReason) IsValid() bool {
	switch r {
	case CancelReasonOutOfIngredient, CancelReasonEquipmentFailure,
		CancelReasonCustomerRequest, CancelReasonOther:
		return true
	}
	return false
}

// DTOs
type OrderResponse struct {
	ID              uuid.UUID           `json:"id"`
	OrderNumber     string              `json:"orderNumber"`
	Status          OrderStatus         `json:"status"`
	PaymentStatus   PaymentStatus       `json:"paymentStatus"`
	PaymentProvider string              `json:"paymentProvider,omitempty"`
	Currency        string              `json:"currency"`
	// CustomerName and CustomerPhone are populated by handlers that load the
	// Customer relation (e.g. chef order list, chef order detail). They are
	// intentionally absent from the base DTO so customer-facing endpoints
	// cannot accidentally return chef-side data.
	CustomerName  string              `json:"customerName,omitempty"`
	CustomerPhone string              `json:"customerPhone,omitempty"`
	Subtotal        float64             `json:"subtotal"`
	DeliveryFee     float64             `json:"deliveryFee"`
	ServiceFee      float64             `json:"serviceFee"`
	Tax             float64             `json:"tax"`
	TaxRate         float64             `json:"taxRate"`
	TaxName         string              `json:"taxName,omitempty"`
	Tip             float64             `json:"tip"`
	ChefTip         float64             `json:"chefTip,omitempty"`
	DriverTip       float64             `json:"driverTip,omitempty"`
	Discount        float64             `json:"discount"`
	Total           float64             `json:"total"`
	Items           []OrderItemResponse `json:"items"`
	DeliveryAddress AddressResponse     `json:"deliveryAddress"`
	CreatedAt       time.Time           `json:"createdAt"`
}

type OrderItemResponse struct {
	ID                   uuid.UUID `json:"id"`
	MenuItemID           uuid.UUID `json:"menuItemId"`
	Name                 string    `json:"name"`
	Price                float64   `json:"price"`
	Quantity             int       `json:"quantity"`
	Subtotal             float64   `json:"subtotal"`
	Notes                string    `json:"notes,omitempty"`
	// SpecialInstructions is an alias for Notes exposed in the vendor detail view.
	SpecialInstructions  string    `json:"specialInstructions,omitempty"`
	// IsVeg is resolved live from the MenuItem at response time. Omitted when nil
	// (legacy items predating the column, or items with no veg flag).
	IsVeg                *bool     `json:"isVeg,omitempty"`
}

type AddressResponse struct {
	Line1      string `json:"line1"`
	Line2      string `json:"line2,omitempty"`
	City       string `json:"city"`
	State      string `json:"state"`
	PostalCode string `json:"postalCode"`
}

// ChefOrderDetailResponse is the full order shape returned by GET /chef/orders/:orderId.
// It extends OrderResponse with fields the vendor needs for kitchen operations.
type ChefOrderDetailResponse struct {
	OrderResponse
	// Timing
	AcceptedAt            *time.Time `json:"acceptedAt,omitempty"`
	PreparedAt            *time.Time `json:"preparedAt,omitempty"`
	PickedUpAt            *time.Time `json:"pickedUpAt,omitempty"`
	DeliveredAt           *time.Time `json:"deliveredAt,omitempty"`
	CancelledAt           *time.Time `json:"cancelledAt,omitempty"`
	CancelReason          string     `json:"cancelReason,omitempty"`
	ScheduledFor          *time.Time `json:"scheduledFor,omitempty"`
	EstimatedPrepTime     int        `json:"estimatedPrepTime,omitempty"`
	EstimatedDeliveryTime int        `json:"estimatedDeliveryTime,omitempty"`
	// Instructions
	SpecialInstructions  string `json:"specialInstructions,omitempty"`
	DeliveryInstructions string `json:"deliveryInstructions,omitempty"`
	// Payment method
	PaymentMethod string `json:"paymentMethod,omitempty"`
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

	currency := o.Currency
	if currency == "" {
		currency = "INR"
	}
	return OrderResponse{
		ID:              o.ID,
		OrderNumber:     o.OrderNumber,
		Status:          o.Status,
		PaymentStatus:   o.PaymentStatus,
		PaymentProvider: o.PaymentProvider,
		Currency:        currency,
		Subtotal:        o.Subtotal,
		DeliveryFee:     o.DeliveryFee,
		ServiceFee:      o.ServiceFee,
		Tax:             o.Tax,
		TaxRate:         o.TaxRate,
		TaxName:         o.TaxName,
		Tip:             o.Tip,
		ChefTip:         o.ChefTip,
		DriverTip:       o.DriverTip,
		Discount:        o.Discount,
		Total:           o.Total,
		Items:           items,
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
