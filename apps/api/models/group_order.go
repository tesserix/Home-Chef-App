package models

import (
	"time"

	"github.com/google/uuid"
)

// group_order.go — group / office orders (#46): a host starts a shared cart from
// one chef, invites others via a link, each participant adds their own items and
// pays their split share, then it consolidates into ONE Order with a single
// delivery to one drop. Two contexts: office (professional/corporate, optional
// company name for the GST invoice) and personal (meetups). Two billing modes:
// split (each pays their share) and host (one payer — the corporate card case).

type GroupOrderType string

const (
	GroupOrderOffice   GroupOrderType = "office"
	GroupOrderPersonal GroupOrderType = "personal"
)

type GroupOrderSplitMode string

const (
	GroupSplitEqualByItems GroupOrderSplitMode = "split" // each pays their own items + pro-rata fees
	GroupSplitHostPays     GroupOrderSplitMode = "host"  // only the host pays the full total
)

type GroupOrderStatus string

const (
	GroupOrderOpen      GroupOrderStatus = "open"      // accepting participants + items
	GroupOrderLocked    GroupOrderStatus = "locked"    // priced; awaiting payment
	GroupOrderPlaced    GroupOrderStatus = "placed"    // consolidated Order created + paid
	GroupOrderConfirmed GroupOrderStatus = "confirmed" // chef accepted (mirrors order)
	GroupOrderDelivered GroupOrderStatus = "delivered"
	GroupOrderCancelled GroupOrderStatus = "cancelled"
	GroupOrderExpired   GroupOrderStatus = "expired"
	// GroupOrderFailed marks a group whose delivery terminally failed (#393/#594). NON-
	// terminal: the payout hold is frozen `disputed` and it stays open until an admin
	// resolves the money outcome (refund participants vs release). Fits varchar(12).
	GroupOrderFailed GroupOrderStatus = "failed"
)

type GroupParticipantRole string

const (
	GroupRoleHost  GroupParticipantRole = "host"
	GroupRoleGuest GroupParticipantRole = "guest"
)

type GroupParticipantPaymentStatus string

const (
	GroupPayPending   GroupParticipantPaymentStatus = "pending"
	GroupPayCompleted GroupParticipantPaymentStatus = "completed"
	GroupPayRefunded  GroupParticipantPaymentStatus = "refunded"
)

// GroupOrderMaxParticipants caps a shared cart (abuse guardrail).
const GroupOrderMaxParticipants = 25

type GroupOrder struct {
	ID        uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	HostID    uuid.UUID           `gorm:"type:uuid;index;not null" json:"hostId"`
	ChefID    uuid.UUID           `gorm:"type:uuid;index;not null" json:"chefId"`
	Type      GroupOrderType      `gorm:"type:varchar(10);not null;default:'personal'" json:"type"`
	SplitMode GroupOrderSplitMode `gorm:"type:varchar(10);not null;default:'split'" json:"splitMode"`
	Title     string              `gorm:"" json:"title,omitempty"`
	// CompanyName — optional, office orders only (GST invoice / corporate billing).
	CompanyName string `gorm:"" json:"companyName,omitempty"`

	// JoinToken backs the invite link. Opaque (crypto/rand hex), never serialized.
	JoinToken string           `gorm:"uniqueIndex;not null" json:"-"`
	Status    GroupOrderStatus `gorm:"type:varchar(12);index;not null;default:'open'" json:"status"`

	// OrderID links the consolidated Order once placed; PayoutTransferID is the
	// chef's held Route transfer (released on delivery).
	OrderID          *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`
	PayoutTransferID string     `gorm:"" json:"-"`
	// CommissionRate freezes the platform commission rate the held transfer was sized at
	// (#547; mirrors Order.CommissionRate). 0 for groups held before this column existed.
	CommissionRate float64 `gorm:"default:0" json:"-"`

	// Payout hold (#456). The group order is a first-class payout-hold aggregate,
	// mirroring Order/MealPlanDay: on delivery the hold parks at
	// awaiting_customer_confirmation (no money moves), the host confirming advances
	// it to release_eligible, which the admin payout queue (#388) consumes to drive
	// the real (flag-gated) Razorpay release. release_eligible itself moves no money.
	PayoutHoldStatus    PayoutHoldStatus `gorm:"type:varchar(32);not null;default:''" json:"payoutHoldStatus,omitempty"`
	CustomerConfirmedAt *time.Time       `gorm:"" json:"customerConfirmedAt,omitempty"`
	DeliveredAt         *time.Time       `gorm:"" json:"deliveredAt,omitempty"`
	// PayoutSettledAt marks the money seam actually completed (#459) — stamped only
	// AFTER releaseMoney/reverseMoney returns nil. PayoutSettleAttempts bounds the
	// payout-reconcile re-drives of a released/reversed row still settled_at NULL.
	PayoutSettledAt      *time.Time `gorm:"" json:"payoutSettledAt,omitempty"`
	PayoutSettleAttempts int        `gorm:"default:0" json:"-"`

	// Single drop — frozen at lock (mirrors Order's delivery columns).
	DeliveryAddressLine1      string  `gorm:"" json:"deliveryAddressLine1,omitempty"`
	DeliveryAddressLine2      string  `gorm:"" json:"deliveryAddressLine2,omitempty"`
	DeliveryAddressCity       string  `gorm:"" json:"deliveryAddressCity,omitempty"`
	DeliveryAddressState      string  `gorm:"" json:"deliveryAddressState,omitempty"`
	DeliveryAddressPostalCode string  `gorm:"" json:"deliveryAddressPostalCode,omitempty"`
	DeliveryAddressCountry    string  `gorm:"type:varchar(2);default:'IN'" json:"deliveryAddressCountry,omitempty"`
	DeliveryLatitude          float64 `gorm:"" json:"deliveryLatitude,omitempty"`
	DeliveryLongitude         float64 `gorm:"" json:"deliveryLongitude,omitempty"`
	DeliveryInstructions      string  `gorm:"" json:"deliveryInstructions,omitempty"`

	// Totals — computed once at lock from the full item set.
	Currency    string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`
	Subtotal    float64 `gorm:"default:0" json:"subtotal"`
	DeliveryFee float64 `gorm:"default:0" json:"deliveryFee"`
	ServiceFee  float64 `gorm:"default:0" json:"serviceFee"`
	Tax         float64 `gorm:"default:0" json:"tax"`
	TaxRate     float64 `gorm:"default:0" json:"taxRate"`
	TaxName     string  `gorm:"" json:"taxName,omitempty"`
	Total       float64 `gorm:"default:0" json:"total"`

	ScheduledFor *time.Time `gorm:"" json:"scheduledFor,omitempty"`
	ExpiresAt    time.Time  `gorm:"index" json:"expiresAt"`
	LockedAt     *time.Time `gorm:"" json:"lockedAt,omitempty"`
	PlacedAt     *time.Time `gorm:"" json:"placedAt,omitempty"`
	CancelledAt  *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason string     `gorm:"" json:"cancelReason,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	// participants/items are NOT omitempty and are normalised to [] before every
	// response (EnsureSlices): a freshly created group has zero items, and an
	// omitted/null array crashes clients that do items.filter/map (mobile #46).
	Participants []GroupOrderParticipant `gorm:"foreignKey:GroupOrderID" json:"participants"`
	Items        []GroupOrderItem        `gorm:"foreignKey:GroupOrderID" json:"items"`
	Chef         *ChefProfile            `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
}

// EnsureSlices guarantees participants/items serialise as [] (never null/omitted)
// so clients can safely call .filter/.map/.length on a freshly created group.
func (g *GroupOrder) EnsureSlices() *GroupOrder {
	if g.Participants == nil {
		g.Participants = []GroupOrderParticipant{}
	}
	if g.Items == nil {
		g.Items = []GroupOrderItem{}
	}
	return g
}

type GroupOrderParticipant struct {
	ID           uuid.UUID            `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	GroupOrderID uuid.UUID            `gorm:"type:uuid;uniqueIndex:idx_group_participant;not null" json:"groupOrderId"`
	UserID       uuid.UUID            `gorm:"type:uuid;uniqueIndex:idx_group_participant;not null" json:"userId"`
	Role         GroupParticipantRole `gorm:"type:varchar(8);not null;default:'guest'" json:"role"`
	DisplayName  string               `gorm:"" json:"displayName,omitempty"`

	ShareAmount       float64                       `gorm:"default:0" json:"shareAmount"`
	PaymentStatus     GroupParticipantPaymentStatus `gorm:"type:varchar(12);index;default:'pending'" json:"paymentStatus"`
	RazorpayOrderID   string                        `gorm:"" json:"razorpayOrderId,omitempty"`
	RazorpayPaymentID string                        `gorm:"" json:"razorpayPaymentId,omitempty"`
	RefundTxnID       *uuid.UUID                    `gorm:"type:uuid" json:"refundTxnId,omitempty"`

	JoinedAt  time.Time `gorm:"autoCreateTime" json:"joinedAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

type GroupOrderItem struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	GroupOrderID  uuid.UUID `gorm:"type:uuid;index;not null" json:"groupOrderId"`
	ParticipantID uuid.UUID `gorm:"type:uuid;index;not null" json:"participantId"`
	MenuItemID    uuid.UUID `gorm:"type:uuid;not null" json:"menuItemId"`
	Name          string    `gorm:"not null" json:"name"`
	Price         float64   `gorm:"not null" json:"price"`
	Quantity      int       `gorm:"not null;default:1" json:"quantity"`
	Subtotal      float64   `gorm:"not null" json:"subtotal"`
	Notes         string    `gorm:"" json:"notes,omitempty"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// SplitShares computes each participant's share for the equal-by-items mode:
// their own item subtotal plus a pro-rata slice of the extras (delivery + service
// + tax), allocated by item-subtotal proportion. The rounding remainder (paise
// drift) is given to the largest contributor so the shares always sum exactly to
// subtotal+extras. Pure + deterministic for unit testing.
func SplitShares(subtotals map[uuid.UUID]float64, extras float64) map[uuid.UUID]float64 {
	out := make(map[uuid.UUID]float64, len(subtotals))
	var groupSubtotal float64
	for _, s := range subtotals {
		groupSubtotal += s
	}
	if len(subtotals) == 0 {
		return out
	}

	round2 := func(f float64) float64 { return float64(int64(f*100+0.5)) / 100 }

	var allocated float64
	var biggestID uuid.UUID
	var biggestSub float64 = -1
	for id, sub := range subtotals {
		var share float64
		if groupSubtotal > 0 {
			share = round2(sub + extras*(sub/groupSubtotal))
		} else {
			// No items anywhere: split the extras evenly.
			share = round2(extras / float64(len(subtotals)))
		}
		out[id] = share
		allocated += share
		if sub > biggestSub {
			biggestSub, biggestID = sub, id
		}
	}

	// Push any rounding drift onto the largest contributor so shares sum exactly.
	total := round2(groupSubtotal + extras)
	if drift := round2(total - allocated); drift != 0 {
		out[biggestID] = round2(out[biggestID] + drift)
	}
	return out
}
