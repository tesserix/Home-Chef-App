package models

import (
	"time"

	"github.com/google/uuid"
)

// OrderChefPayout is what a chef is paid for one order, persisted once.
//
// The chef used to be shown the CUSTOMER's numbers — subtotal, platform fee,
// combined GST, and a ₹351.97 total on an order the kitchen earns ₹320 for. A
// chef cannot reconcile a payout against that, and admin had no per-order view
// at all, so there was no single figure the two could agree on.
//
//	net = food + delivery + tip − penalty
//
// No commission, no GST, no TDS. The platform's revenue is the customer-paid
// platform fee, and the platform accounts for all GST (CGST s.9(5): the
// e-commerce operator is liable for the tax on restaurant service supplied
// through it, not the kitchen). See .planning/CHEF-PAYOUT-VIEW-DESIGN.md.
type OrderChefPayout struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OrderID  uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"orderId"`
	ChefID   uuid.UUID `gorm:"type:uuid;not null;index" json:"chefId"`
	Currency string    `gorm:"type:varchar(3);not null;default:INR" json:"currency"`

	// FoodAmount is the order's food subtotal — the chef's own price.
	FoodAmount float64 `gorm:"type:numeric(12,2);not null;default:0" json:"foodAmount"`
	// DeliveryFee is the chef's ONLY when they carried the leg and it was
	// charged; 0 otherwise, and the UI then omits the line rather than showing ₹0.
	DeliveryFee float64 `gorm:"type:numeric(12,2);not null;default:0" json:"deliveryFee"`
	// ChefTip is the kitchen's tip. A driver tip is never the chef's.
	ChefTip float64 `gorm:"type:numeric(12,2);not null;default:0" json:"chefTip"`
	// Penalty is the amount attributed to this order (chefcancel:<orderID>),
	// positive meaning deducted. The money still moves at the weekly settlement —
	// the ledger stays the authority for when.
	Penalty float64 `gorm:"type:numeric(12,2);not null;default:0" json:"penalty"`
	// NetPayout is the ONLY figure either surface renders. Clients never recompute.
	NetPayout float64 `gorm:"type:numeric(12,2);not null;default:0" json:"netPayout"`

	Status     string    `gorm:"type:varchar(16);not null;default:pending;index" json:"status"`
	ComputedAt time.Time `gorm:"not null" json:"computedAt"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Chef payout lifecycle.
const (
	// ChefPayoutEstimated — computed from the live order, NOT persisted. What a
	// chef sees before delivery, when there is no row yet. Never written to the
	// table: it is a projection of an order still in flight, and the figures can
	// still move (the chef may lower the delivery fee at accept, a penalty may be
	// raised). The row written at delivery is what freezes them.
	ChefPayoutEstimated = "estimated"
	// ChefPayoutPending — computed and owed, not yet released.
	ChefPayoutPending = "pending"
	// ChefPayoutReleased — settled to the chef.
	ChefPayoutReleased = "released"
	// ChefPayoutReversed — the order was refunded/cancelled after computation.
	ChefPayoutReversed = "reversed"
)

func (OrderChefPayout) TableName() string { return "order_chef_payouts" }

// ChefPayoutResponse is the ONE serialised payout shape. The vendor app and the
// tesserix-home admin view both render this — neither recomputes, so the figure
// a chef sees and the figure staff verify a payout against cannot diverge.
type ChefPayoutResponse struct {
	FoodAmount float64 `json:"foodAmount"`
	// DeliveryFee is 0 unless the chef carried the leg and it was charged; the UI
	// omits the line entirely rather than showing ₹0.
	DeliveryFee float64 `json:"deliveryFee"`
	ChefTip     float64 `json:"chefTip"`
	Penalty     float64 `json:"penalty"`
	// NetPayout is the exact sum of the lines above, to the paise.
	NetPayout  float64   `json:"netPayout"`
	Currency   string    `json:"currency"`
	Status     string    `json:"status"`
	ComputedAt time.Time `json:"computedAt"`
}

// ToResponse serialises a stored payout for both surfaces.
func (p *OrderChefPayout) ToResponse() *ChefPayoutResponse {
	if p == nil {
		return nil
	}
	return &ChefPayoutResponse{
		FoodAmount: p.FoodAmount, DeliveryFee: p.DeliveryFee, ChefTip: p.ChefTip,
		Penalty: p.Penalty, NetPayout: p.NetPayout,
		Currency: p.Currency, Status: p.Status, ComputedAt: p.ComputedAt,
	}
}
