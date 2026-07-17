package models

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// parseOrderItemModifiers decodes the JSON modifier snapshot on an order line,
// returning an empty slice for blank/invalid data (#232).
func parseOrderItemModifiers(s string) []OrderItemModifier {
	out := []OrderItemModifier{}
	if s == "" || s == "[]" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

// ParsedModifiers returns this line's selected add-ons (#232), for surfaces
// outside the models package (e.g. the invoice generator).
func (i *OrderItem) ParsedModifiers() []OrderItemModifier {
	return parseOrderItemModifiers(i.Modifiers)
}

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
	// OrderStatusRejected is a chef declining a pending order — semantically a
	// chef-initiated cancellation (releases reserved capacity + triggers the
	// cancellation/refund path). The vendor app sends this status; without the
	// enum it was stored as an invalid string and skipped the cancel side-effects.
	OrderStatusRejected OrderStatus = "rejected"
	OrderStatusRefunded OrderStatus = "refunded"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentCompleted PaymentStatus = "completed"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
)

// FulfillmentType is how an order reaches the customer.
type FulfillmentType string

const (
	FulfillmentDelivery     FulfillmentType = "delivery"      // 3PL rider (default)
	FulfillmentChefDelivery FulfillmentType = "chef_delivery" // chef delivers (Phase 2)
	FulfillmentPickup       FulfillmentType = "pickup"        // customer collects
)

type Order struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderNumber     string          `gorm:"uniqueIndex;not null" json:"orderNumber"`
	CustomerID      uuid.UUID       `gorm:"type:uuid;not null;index" json:"customerId"`
	ChefID          uuid.UUID       `gorm:"type:uuid;not null;index" json:"chefId"`
	DeliveryID      *uuid.UUID      `gorm:"type:uuid;index" json:"deliveryId,omitempty"`
	Status          OrderStatus     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentStatus   PaymentStatus   `gorm:"type:varchar(20);default:'pending'" json:"paymentStatus"`
	PaymentMethod   string          `gorm:"" json:"paymentMethod"`
	FulfillmentType FulfillmentType `gorm:"type:varchar(16);default:'delivery'" json:"fulfillmentType"`

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
	// CommissionRate freezes the platform commission rate applied when the order
	// was placed, so a later admin retune of the runtime rate cannot make the
	// settlement statement disagree with the Route transfer already sent (#390).
	// 0/unset for legacy rows → callers fall back to the live rate/default.
	CommissionRate float64 `gorm:"default:0" json:"commissionRate"`
	Tip            float64 `gorm:"default:0" json:"tip"`       // Legacy: total tip (kept for backward compat)
	ChefTip        float64 `gorm:"default:0" json:"chefTip"`   // Tip for the chef/kitchen
	DriverTip      float64 `gorm:"default:0" json:"driverTip"` // Tip for the delivery driver
	Discount       float64 `gorm:"default:0" json:"discount"`
	// ChefFundedDiscount is the portion of Discount funded by the chef via a
	// chef-funded promo (#39). It is billed to the chef at settlement (subtracted
	// from their Route payout + earnings); platform-funded promos leave it 0 and
	// the platform absorbs the discount. Always 0 ≤ ChefFundedDiscount ≤ Discount.
	ChefFundedDiscount float64 `gorm:"default:0" json:"chefFundedDiscount"`
	Total              float64 `gorm:"not null" json:"total"`
	// WalletApplied is the store credit applied at checkout (#141). The customer
	// is charged (Total − WalletApplied) at the gateway; the chef/driver splits are
	// still settled in full (the wallet-covered slice is topped up from the platform
	// balance). Recorded at payment-create, debited from the wallet on capture.
	WalletApplied float64 `gorm:"default:0" json:"walletApplied"`
	PromoCode     string  `gorm:"" json:"promoCode,omitempty"`
	// Currency is the 3-letter ISO code the customer is charged in. Frozen
	// at order creation from the chef's settlement currency so later edits
	// on the chef profile don't invalidate an in-flight payment.
	Currency string `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	// Delivery Address
	DeliveryAddressLine1      string `gorm:"" json:"deliveryAddressLine1"`
	DeliveryAddressLine2      string `gorm:"" json:"deliveryAddressLine2"`
	DeliveryAddressCity       string `gorm:"" json:"deliveryAddressCity"`
	DeliveryAddressState      string `gorm:"" json:"deliveryAddressState"`
	DeliveryAddressPostalCode string `gorm:"" json:"deliveryAddressPostalCode"`
	// ISO-3166 alpha-2 country used to pick the tax rule and for invoicing.
	DeliveryAddressCountry string  `gorm:"type:varchar(2);default:'IN'" json:"deliveryAddressCountry"`
	DeliveryLatitude       float64 `gorm:"" json:"deliveryLatitude"`
	DeliveryLongitude      float64 `gorm:"" json:"deliveryLongitude"`
	DeliveryInstructions   string  `gorm:"type:text" json:"deliveryInstructions"`

	// Timing
	EstimatedPrepTime     int        `gorm:"" json:"estimatedPrepTime"` // minutes
	EstimatedDeliveryTime int        `gorm:"" json:"estimatedDeliveryTime"`
	ScheduledFor          *time.Time `gorm:"" json:"scheduledFor,omitempty"`
	// DeliverySlot is the named scheduled-delivery slot this order was placed
	// for (#51): "" (ASAP / unscheduled), "lunch", or "dinner". When set,
	// ScheduledFor holds the slot window start on the chosen IST day and the
	// chef's per-slot daily capacity was reserved at order time.
	DeliverySlot string     `gorm:"type:varchar(8)" json:"deliverySlot,omitempty"`
	AcceptedAt   *time.Time `gorm:"" json:"acceptedAt,omitempty"`
	// AcceptReminderCount / LastAcceptReminderAt track the pre-close nudges (#694):
	// from 2h before the kitchen closes, every 30 minutes, the chef is reminded of
	// an order still not accepted. The count is how many have fired; the sweep uses
	// it to know whether the next scheduled nudge is owed, so a 5-minute sweep tick
	// cannot double-send within one 30-minute slot.
	AcceptReminderCount  int        `gorm:"not null;default:0" json:"acceptReminderCount"`
	LastAcceptReminderAt *time.Time `gorm:"" json:"lastAcceptReminderAt,omitempty"`
	PreparedAt           *time.Time `gorm:"" json:"preparedAt,omitempty"`
	PickedUpAt           *time.Time `gorm:"" json:"pickedUpAt,omitempty"`
	DeliveredAt          *time.Time `gorm:"" json:"deliveredAt,omitempty"`
	CancelledAt          *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason         string     `gorm:"" json:"cancelReason,omitempty"`

	// Special Instructions
	SpecialInstructions string `gorm:"type:text" json:"specialInstructions"`

	// Lifecycle photos (public GCS URLs). The chef must attach these at the
	// matching transition (enforced in the vendor app):
	//   ReadyPhotoURL    — the prepared dish, captured when the chef marks the
	//                      order ready. Shown to the customer for trust/appetite.
	//   HandoverPhotoURL — proof-of-handover at pickup, captured when the chef
	//                      marks a pickup order handed over (dispute evidence).
	ReadyPhotoURL    string `gorm:"type:text" json:"readyPhotoUrl,omitempty"`
	HandoverPhotoURL string `gorm:"type:text" json:"handoverPhotoUrl,omitempty"`

	// Payment gateway. PaymentProvider records which gateway handled this
	// order so refunds, reconciliation, and webhooks read the correct
	// provider-specific ID below. Inherited from ChefProfile.PaymentProvider
	// at order creation time so late-switching a chef doesn't invalidate
	// already-placed orders.
	PaymentProvider string `gorm:"type:varchar(20);default:'razorpay'" json:"paymentProvider"`
	// Gateway ids. Each is covered by a PARTIAL unique index (WHERE col <> '') created in
	// database.go's postMigrate block (#395·1) — a plain GORM uniqueIndex tag can't be used
	// because wallet-only/unpaid/Stripe/meal-plan-day-shell orders share the empty default.
	StripePaymentIntentID string     `gorm:"" json:"-"` // Stripe PaymentIntent ID (unique when set)
	RazorpayOrderID       string     `gorm:"" json:"-"` // Razorpay order ID (unique when set)
	RazorpayPaymentID     string     `gorm:"" json:"-"` // Razorpay payment ID (unique when set)
	RefundID              string     `gorm:"" json:"-"` // Gateway refund ID (if refunded)
	RefundedAt            *time.Time `gorm:"" json:"refundedAt,omitempty"`
	RefundAmount          float64    `gorm:"default:0" json:"refundAmount"`
	RefundReason          string     `gorm:"" json:"refundReason,omitempty"`
	RefundInitiatedBy     string     `gorm:"type:varchar(20)" json:"refundInitiatedBy,omitempty"` // chef, admin, system

	// Payout hold (#387). Independent of Status: on delivery the hold becomes
	// awaiting_customer_confirmation (no money moves); the customer confirming
	// advances it to release_eligible, which the admin payout queue (#388)
	// consumes to drive the real Razorpay release. release_eligible itself moves
	// no money. Empty for meal-plan/consolidated orders (they settle their own).
	PayoutHoldStatus    PayoutHoldStatus `gorm:"type:varchar(32);default:''" json:"payoutHoldStatus,omitempty"`
	CustomerConfirmedAt *time.Time       `gorm:"" json:"customerConfirmedAt,omitempty"`

	// PayoutSettledAt marks that the money seam actually completed (#459) — stamped
	// only AFTER releaseMoney/reverseMoney returns nil. It decouples
	// status-committed (payout_hold_status flipped in the primary tx) from
	// money-confirmed-moved: a released/reversed row with settled_at NULL is drift
	// the payout-reconcile cron re-drives. PayoutSettleAttempts bounds those
	// re-drives (alert + stop at the cap for a permanently-bad transfer).
	PayoutSettledAt      *time.Time `gorm:"" json:"payoutSettledAt,omitempty"`
	PayoutSettleAttempts int        `gorm:"default:0" json:"-"`

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
	// Modifiers is a JSON snapshot of the selected add-ons for this line (#232) —
	// []OrderItemModifier{groupName, optionName, priceDelta}. Subtotal already
	// includes their deltas; this is the immutable breakdown for the order + invoice.
	Modifiers string    `gorm:"type:jsonb;default:'[]'" json:"-"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`

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
	CancelReasonOutOfIngredient  CancelReason = "out_of_ingredient"
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
// OrderSource tags where an order originated so the vendor feed can group it
// (#435). Derived at response time from the reverse links (meal-plan day,
// subscription fulfilment, group order) — not a stored column.
type OrderSource string

const (
	OrderSourceAlacarte     OrderSource = "alacarte"
	OrderSourceMealPlan     OrderSource = "meal_plan"
	OrderSourceSubscription OrderSource = "subscription"
	OrderSourceGroup        OrderSource = "group"
)

type OrderResponse struct {
	ID              uuid.UUID       `json:"id"`
	OrderNumber     string          `json:"orderNumber"`
	Status          OrderStatus     `json:"status"`
	FulfillmentType FulfillmentType `json:"fulfillmentType"`
	PaymentStatus   PaymentStatus   `json:"paymentStatus"`
	PaymentProvider string          `json:"paymentProvider,omitempty"`
	Currency        string          `json:"currency"`
	// Source groups the order in the vendor feed (à-la-carte / meal-plan day /
	// subscription day / group). Set by the chef-orders handler (#435).
	Source OrderSource `json:"source,omitempty"`
	// CustomerName and CustomerPhone are populated by handlers that load the
	// Customer relation (e.g. chef order list, chef order detail). They are
	// intentionally absent from the base DTO so customer-facing endpoints
	// cannot accidentally return chef-side data.
	CustomerName    string              `json:"customerName,omitempty"`
	CustomerPhone   string              `json:"customerPhone,omitempty"`
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
	// Chef is populated when the handler preloads the Chef relation
	// (customer order list/detail). Omitted otherwise so chef-facing
	// endpoints don't carry a redundant self-reference.
	Chef *OrderChefResponse `json:"chef,omitempty"`
	// Scheduled delivery slot (#51) — surfaced so the customer order
	// list/detail can show "Lunch · Mon 12:00" instead of "ASAP". Both omit
	// when the order is unscheduled (ASAP).
	ScheduledFor *time.Time `json:"scheduledFor,omitempty"`
	DeliverySlot string     `json:"deliverySlot,omitempty"`
	// Lifecycle photos (public URLs) — the food-ready photo is shown to the
	// customer; the handover photo is dispute evidence for pickup orders.
	ReadyPhotoURL    string    `json:"readyPhotoUrl,omitempty"`
	HandoverPhotoURL string    `json:"handoverPhotoUrl,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	// Escrow payout-hold state (#387/#617). Surfaced so the customer app can render the
	// "Confirm received" CTA on a delivered order awaiting confirmation, plus the confirmed /
	// disputed states. Empty (and omitted) when there is no hold — i.e. whenever the escrow
	// flags are off — so the customer CTA degrades to nothing pre-launch.
	PayoutHoldStatus    PayoutHoldStatus `json:"payoutHoldStatus,omitempty"`
	CustomerConfirmedAt *time.Time       `json:"customerConfirmedAt,omitempty"`
}

// OrderChefResponse is the minimal chef identity the customer order
// list/detail render (business name + image). Kept small on purpose — the
// full chef profile is fetched separately when a detail view needs it.
type OrderChefResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	ImageURL string    `json:"imageUrl,omitempty"`
}

type OrderItemResponse struct {
	ID         uuid.UUID `json:"id"`
	MenuItemID uuid.UUID `json:"menuItemId"`
	Name       string    `json:"name"`
	Price      float64   `json:"price"`
	Quantity   int       `json:"quantity"`
	Subtotal   float64   `json:"subtotal"`
	Notes      string    `json:"notes,omitempty"`
	// Modifiers (#232) — the selected add-ons for this line.
	Modifiers []OrderItemModifier `json:"modifiers"`
	// SpecialInstructions is an alias for Notes exposed in the vendor detail view.
	SpecialInstructions string `json:"specialInstructions,omitempty"`
	// IsVeg is resolved live from the MenuItem at response time. Omitted when nil
	// (legacy items predating the column, or items with no veg flag).
	IsVeg *bool `json:"isVeg,omitempty"`
	// Per-line cancellation state. Surfaced so the mobile order detail
	// can render the cancelled line with strikethrough + "Refunded ₹X"
	// while the rest of the order continues prep. Mirror of the same-
	// named columns on the OrderItem GORM model.
	IsCancelled     bool       `json:"isCancelled,omitempty"`
	CancelledReason string     `json:"cancelledReason,omitempty"`
	CancelledAt     *time.Time `json:"cancelledAt,omitempty"`
	RefundAmount    float64    `json:"refundAmount,omitempty"`
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
	// OffersSelfDelivery is the chef's self-delivery CAPABILITY (the "I deliver
	// myself" toggle), surfaced so the vendor app can offer the Mark-Ready
	// carrier choice on a delivery order. This is the authoritative gate — NOT
	// the distance fields below, which are 0 when the chef set no radius or
	// coords are missing and must never be used to infer the capability.
	OffersSelfDelivery bool `json:"offersSelfDelivery"`
	// RiderDispatchAvailable is a COMPUTED flag: whether a 3PL provider is
	// currently enabled, i.e. whether "hand to a rider" is a real option. The
	// vendor app gates the Mark-Ready rider button (and the mid-ready "hand to a
	// rider" switch) on this, so the chef is never offered a rider while 3PL is
	// dark. Flipping a provider's is_enabled=true re-enables the rider path with
	// no app change.
	RiderDispatchAvailable bool `json:"riderDispatchAvailable"`
	// Chef self-delivery distance gate (chef_delivery only). DistanceKm is the
	// chef→drop straight-line distance; MaxDistanceKm is the chef's configured
	// comfort radius. The vendor app shows a soft "beyond your range" warning
	// when DistanceKm > MaxDistanceKm > 0. Both 0/omitted for other fulfillment
	// modes (or when coords are missing — distance unknown).
	SelfDeliveryDistanceKm    float64 `json:"selfDeliveryDistanceKm,omitempty"`
	SelfDeliveryMaxDistanceKm float64 `json:"selfDeliveryMaxDistanceKm,omitempty"`
}

func (o *Order) ToResponse() OrderResponse {
	items := make([]OrderItemResponse, len(o.Items))
	for i, item := range o.Items {
		items[i] = OrderItemResponse{
			ID:              item.ID,
			MenuItemID:      item.MenuItemID,
			Name:            item.Name,
			Price:           item.Price,
			Quantity:        item.Quantity,
			Subtotal:        item.Subtotal,
			Notes:           item.Notes,
			Modifiers:       parseOrderItemModifiers(item.Modifiers),
			IsCancelled:     item.IsCancelled,
			CancelledReason: item.CancelledReason,
			CancelledAt:     item.CancelledAt,
			RefundAmount:    item.RefundAmount,
		}
	}

	currency := o.Currency
	if currency == "" {
		currency = "INR"
	}

	// Chef identity — only when the relation was preloaded (Chef.ID set).
	// Prefer the profile image, fall back to the banner.
	var chef *OrderChefResponse
	if o.Chef.ID != uuid.Nil {
		image := o.Chef.ProfileImage
		if image == "" {
			image = o.Chef.BannerImage
		}
		chef = &OrderChefResponse{
			ID:       o.Chef.ID,
			Name:     o.Chef.BusinessName,
			ImageURL: image,
		}
	}

	return OrderResponse{
		ID:          o.ID,
		OrderNumber: o.OrderNumber,
		Status:      o.Status,
		FulfillmentType: func() FulfillmentType {
			if o.FulfillmentType == "" {
				return FulfillmentDelivery
			}
			return o.FulfillmentType
		}(),
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
		Chef:                chef,
		ScheduledFor:        o.ScheduledFor,
		DeliverySlot:        o.DeliverySlot,
		ReadyPhotoURL:       o.ReadyPhotoURL,
		HandoverPhotoURL:    o.HandoverPhotoURL,
		CreatedAt:           o.CreatedAt,
		PayoutHoldStatus:    o.PayoutHoldStatus,
		CustomerConfirmedAt: o.CustomerConfirmedAt,
	}
}

// ToChefResponse builds the order response for the CHEF / vendor view. It starts
// from the customer DTO but redacts customer PII so a chef can never see the
// exact delivery address or contact the customer directly (an off-platform
// bypass risk): the street address is reduced to the delivery AREA (city/state),
// the phone is dropped, and only the customer's first name is kept (enough to
// label the packed order). The 3PL rider still receives the precise address +
// phone server-to-server — they, not the chef, perform the delivery.
//
// Food/special instructions are preserved (the chef needs them to cook
// correctly); navigation/delivery instructions are handled by the caller for
// the detail view, since they can leak building-level address detail.
//
// Requires the Customer relation to be preloaded for the first name; when it
// isn't, CustomerName is simply empty (omitempty), never the full name.
func (o *Order) ToChefResponse() OrderResponse {
	resp := o.ToResponse()
	// chef_delivery: the chef delivers themselves, so they need the customer's
	// FULL address + phone + name to reach the door. ToResponse already populated
	// the full address; keep it and add contact. (Intentional contact exposure
	// scoped to chef_delivery — masked calling, #321, can replace it later.)
	if o.FulfillmentType == FulfillmentChefDelivery {
		resp.CustomerName = strings.TrimSpace(o.Customer.FirstName + " " + o.Customer.LastName)
		resp.CustomerPhone = o.Customer.Phone
		return resp
	}
	// delivery + pickup: area only (city/state), first name, no phone (privacy).
	resp.DeliveryAddress = AddressResponse{
		City:  o.DeliveryAddressCity,
		State: o.DeliveryAddressState,
	}
	resp.CustomerName = o.Customer.FirstName
	resp.CustomerPhone = ""
	return resp
}
