package models

import (
	"time"

	"github.com/google/uuid"
)

// meal_plan.go — the tiffin meal-plan: a customer pre-books a calendar of days
// (a week/month ahead) from one chef, each day tagged a slot + veg/nonveg. The
// chef accepts all or cherry-picks a subset; the customer approves any trim; money
// is held in escrow and released per delivered day. Design + flow: issue #1; this
// file is the data model + state machine (#193). Payment/escrow lives in #194.

// MealSlot is the meal a plan day covers.
type MealSlot string

const (
	MealSlotLunch  MealSlot = "lunch"
	MealSlotDinner MealSlot = "dinner"
)

// MealVariant is the customer's per-day veg/nonveg choice (the chef offers both
// variants per slot in their weekly menu, #192).
type MealVariant string

const (
	MealVariantVeg    MealVariant = "veg"
	MealVariantNonVeg MealVariant = "nonveg"
)

// MealPlanStatus is the negotiation + lifecycle state of a whole plan.
type MealPlanStatus string

const (
	MealPlanPendingChef      MealPlanStatus = "pending_chef"       // customer paid the advance, awaiting chef
	MealPlanChefAcceptedFull MealPlanStatus = "chef_accepted_full" // chef took every day → auto-confirms
	MealPlanChefModified     MealPlanStatus = "chef_modified"      // chef trimmed days
	MealPlanAwaitingCustomer MealPlanStatus = "awaiting_customer"  // trimmed plan back to the customer
	MealPlanConfirmed        MealPlanStatus = "confirmed"          // accepted set locked, escrow settled
	MealPlanActive           MealPlanStatus = "active"             // deliveries in progress
	MealPlanCompleted        MealPlanStatus = "completed"
	MealPlanCancelled        MealPlanStatus = "cancelled"
	MealPlanExpired          MealPlanStatus = "expired" // a cutoff lapsed → full refund
)

// MealPlanDayStatus is the per-day state.
type MealPlanDayStatus string

const (
	MealPlanDayRequested MealPlanDayStatus = "requested"
	MealPlanDayAccepted  MealPlanDayStatus = "accepted" // chef will cook this day
	MealPlanDayDeclined  MealPlanDayStatus = "declined" // chef cherry-picked it out → refunded
	MealPlanDayConfirmed MealPlanDayStatus = "confirmed"
	MealPlanDayPrepared  MealPlanDayStatus = "prepared"
	MealPlanDayDelivered MealPlanDayStatus = "delivered" // releases the held payout
	MealPlanDaySkipped   MealPlanDayStatus = "skipped"   // customer skipped before cutoff → refunded
	MealPlanDayCancelled MealPlanDayStatus = "cancelled"
	MealPlanDayRefunded  MealPlanDayStatus = "refunded"
)

// MealPlan is one customer's advance booking from one chef.
type MealPlan struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MealPlanNumber string         `gorm:"uniqueIndex;not null" json:"mealPlanNumber"`
	CustomerID     uuid.UUID      `gorm:"type:uuid;index;not null" json:"customerId"`
	ChefID         uuid.UUID      `gorm:"type:uuid;index;not null" json:"chefId"`
	Status         MealPlanStatus `gorm:"type:varchar(24);index;default:'pending_chef'" json:"status"`

	StartDate time.Time `gorm:"index" json:"startDate"`
	EndDate   time.Time `gorm:"index" json:"endDate"`

	// Money snapshot for the full REQUESTED set (the advance). The accepted
	// subset's totals are derived from the days; the declined remainder is refunded.
	Subtotal float64 `gorm:"default:0" json:"subtotal"`
	Tax      float64 `gorm:"default:0" json:"tax"`
	Total    float64 `gorm:"not null" json:"total"`
	Currency string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	// Escrow links (#194): the upfront capture that funds the held chef payouts.
	EscrowPaymentID string `gorm:"" json:"escrowPaymentId,omitempty"`
	RazorpayOrderID string `gorm:"" json:"razorpayOrderId,omitempty"`

	// Negotiation cutoffs — a lapse auto-cancels + fully refunds.
	ChefRespondBy     *time.Time `gorm:"" json:"chefRespondBy,omitempty"`
	CustomerApproveBy *time.Time `gorm:"" json:"customerApproveBy,omitempty"`

	ConfirmedAt  *time.Time `gorm:"" json:"confirmedAt,omitempty"`
	CancelledAt  *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason string     `gorm:"type:text" json:"cancelReason,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Days     []MealPlanDay `gorm:"foreignKey:MealPlanID" json:"days,omitempty"`
	Customer *User         `gorm:"foreignKey:CustomerID" json:"customer,omitempty"`
	Chef     *ChefProfile  `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
}

// MealPlanDay is one slot on one date within a plan.
type MealPlanDay struct {
	ID         uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MealPlanID uuid.UUID         `gorm:"type:uuid;index;not null" json:"mealPlanId"`
	Date       time.Time         `gorm:"index;not null" json:"date"`
	Slot       MealSlot          `gorm:"type:varchar(10);not null" json:"slot"`
	Variant    MealVariant       `gorm:"type:varchar(10);not null" json:"variant"`
	Status     MealPlanDayStatus `gorm:"type:varchar(12);index;default:'requested'" json:"status"`

	// Resolved from the chef's weekly menu (#192); DishName snapshots the title so
	// later menu edits don't rewrite history.
	WeeklyMenuItemID *uuid.UUID `gorm:"type:uuid" json:"weeklyMenuItemId,omitempty"`
	DishName         string     `gorm:"" json:"dishName,omitempty"`
	Price            float64    `gorm:"default:0" json:"price"`

	// Fulfilment + money links (#194/#197).
	OrderID          *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`
	PayoutTransferID string     `gorm:"" json:"payoutTransferId,omitempty"`
	// PreparedAt is stamped when the chef marks the dish prepared from the prep
	// view (#50) — the "being cooked" signal the customer sees live.
	PreparedAt  *time.Time `gorm:"" json:"preparedAt,omitempty"`
	DeliveredAt *time.Time `gorm:"" json:"deliveredAt,omitempty"`
	RefundTxnID *uuid.UUID `gorm:"type:uuid" json:"refundTxnId,omitempty"`

	// Payout hold (#387). Same semantics as Order: on delivery the day's hold
	// becomes awaiting_customer_confirmation (no release); the customer confirming
	// advances it to release_eligible for the admin payout queue (#388).
	PayoutHoldStatus    PayoutHoldStatus `gorm:"type:varchar(32);default:''" json:"payoutHoldStatus,omitempty"`
	CustomerConfirmedAt *time.Time       `gorm:"" json:"customerConfirmedAt,omitempty"`

	// PayoutSettledAt / PayoutSettleAttempts — same semantics as Order (#459): the
	// money seam (ReleaseDayPayout / ReverseTransfer) is only stamped settled after
	// it returns nil; a released/reversed day with settled_at NULL is drift the
	// payout-reconcile cron re-drives, bounded by the attempt counter.
	PayoutSettledAt      *time.Time `gorm:"" json:"payoutSettledAt,omitempty"`
	PayoutSettleAttempts int        `gorm:"default:0" json:"-"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// AcceptedTotal sums the price of the days the chef accepted/confirmed — the
// amount that stays in escrow (the rest is refunded).
func (p *MealPlan) AcceptedTotal() float64 {
	var sum float64
	for _, d := range p.Days {
		switch d.Status {
		case MealPlanDayDeclined, MealPlanDaySkipped, MealPlanDayCancelled, MealPlanDayRefunded:
			continue
		default:
			sum += d.Price
		}
	}
	return sum
}
