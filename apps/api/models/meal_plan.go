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
	// MealPlanDaySkipRequested marks a day the customer asked to skip. It is a
	// NON-terminal, still-in-scope holding state (fits varchar(12)): the day's payout
	// hold is frozen `disputed` and the plan stays open until an ADMIN approves (→
	// skipped + partial refund) or rejects (→ confirmed). Not auto-refunded — the
	// customer no longer self-credits a skip (#422 policy change).
	MealPlanDaySkipRequested MealPlanDayStatus = "skip_req"
	MealPlanDaySkipped       MealPlanDayStatus = "skipped" // admin approved a skip → partial refund
	MealPlanDayCancelled MealPlanDayStatus = "cancelled"
	MealPlanDayRefunded  MealPlanDayStatus = "refunded"
	// MealPlanDayFailed marks a day whose delivery terminally failed (#393). It is
	// deliberately NON-terminal (excluded from allDaysTerminal): the day's payout hold
	// is frozen to disputed and the plan stays open until an admin resolves the day's
	// money outcome (refund vs release).
	MealPlanDayFailed MealPlanDayStatus = "failed"
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
	// Covered by a PARTIAL unique index (WHERE razorpay_order_id <> '') in database.go's
	// postMigrate block (#395·1) — unique when set, empty for an unpaid/handshake plan.
	RazorpayOrderID string `gorm:"" json:"razorpayOrderId,omitempty"`

	// Negotiation cutoffs — a lapse auto-cancels + fully refunds.
	ChefRespondBy     *time.Time `gorm:"" json:"chefRespondBy,omitempty"`
	CustomerApproveBy *time.Time `gorm:"" json:"customerApproveBy,omitempty"`

	ConfirmedAt  *time.Time `gorm:"" json:"confirmedAt,omitempty"`
	CancelledAt  *time.Time `gorm:"" json:"cancelledAt,omitempty"`
	CancelReason string     `gorm:"type:text" json:"cancelReason,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	Days []MealPlanDay `gorm:"foreignKey:MealPlanID" json:"days,omitempty"`
	// SECURITY (audit H1/M1): the raw relations are json:"-" so they never
	// auto-serialize. A chef must not receive the customer's email/phone (the
	// off-platform-contact bypass the rest of the platform guards), and a customer
	// must not receive the home-chef's street address / lat-long / FSSAI / GSTIN.
	// Handlers populate the minimized, role-appropriate views below via ProjectFor*.
	Customer *User        `gorm:"foreignKey:CustomerID" json:"-"`
	Chef     *ChefProfile `gorm:"foreignKey:ChefID" json:"-"`

	CustomerView *MealPlanCustomerView `gorm:"-" json:"customer,omitempty"`
	ChefView     *MealPlanChefView     `gorm:"-" json:"chef,omitempty"`
}

// MealPlanCustomerView is the customer info a chef (or admin) may see about a
// plan. Email/Phone are populated ONLY for the admin projection.
type MealPlanCustomerView struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email,omitempty"`
	Phone     string `json:"phone,omitempty"`
}

// MealPlanChefView is the chef info a customer may see — business identity only,
// never the home-chef's precise location or licence numbers.
type MealPlanChefView struct {
	BusinessName string `json:"businessName"`
	ProfileImage string `json:"profileImage,omitempty"`
}

// ProjectForChef exposes only the customer's name to the chef (no email/phone).
func (m *MealPlan) ProjectForChef() {
	if m.Customer != nil {
		m.CustomerView = &MealPlanCustomerView{FirstName: m.Customer.FirstName, LastName: m.Customer.LastName}
	}
	m.ChefView = nil
}

// ProjectForCustomer exposes only the chef's business name + image to the customer.
func (m *MealPlan) ProjectForCustomer() {
	if m.Chef != nil {
		m.ChefView = &MealPlanChefView{BusinessName: m.Chef.BusinessName, ProfileImage: m.Chef.ProfileImage}
	}
	m.CustomerView = nil
}

// ProjectForAdmin (RequireAdmin routes only) keeps full customer contact for support.
func (m *MealPlan) ProjectForAdmin() {
	if m.Customer != nil {
		m.CustomerView = &MealPlanCustomerView{
			FirstName: m.Customer.FirstName, LastName: m.Customer.LastName,
			Email: m.Customer.Email, Phone: m.Customer.Phone,
		}
	}
	if m.Chef != nil {
		m.ChefView = &MealPlanChefView{BusinessName: m.Chef.BusinessName, ProfileImage: m.Chef.ProfileImage}
	}
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
	// CommissionRate freezes the platform commission rate the held transfer was sized at
	// (#547), mirroring Order.CommissionRate — so any later recompute (admin queue net,
	// reconciliation) is exact even if the platform flat rate changed while the hold was
	// pending. 0 for days held before this column existed (readers fall back to current).
	CommissionRate float64 `gorm:"default:0" json:"commissionRate,omitempty"`
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

// AcceptedDayCount counts the days that stay in scope (same filter as
// AcceptedTotal) — used to size the per-day delivery fee on the accepted set.
func (p *MealPlan) AcceptedDayCount() int {
	n := 0
	for _, d := range p.Days {
		switch d.Status {
		case MealPlanDayDeclined, MealPlanDaySkipped, MealPlanDayCancelled, MealPlanDayRefunded:
			continue
		default:
			n++
		}
	}
	return n
}
