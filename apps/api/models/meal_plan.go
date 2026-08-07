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
	MealPlanDayCancelled     MealPlanDayStatus = "cancelled"
	MealPlanDayRefunded      MealPlanDayStatus = "refunded"
	// MealPlanDayFailed marks a day whose delivery terminally failed (#393). It is
	// deliberately NON-terminal (excluded from allDaysTerminal): the day's payout hold
	// is frozen to disputed and the plan stays open until an admin resolves the day's
	// money outcome (refund vs release).
	MealPlanDayFailed MealPlanDayStatus = "failed"
)

// MealPlan is one customer's advance booking from one chef.
type MealPlan struct {
	// Live/test data partition. See models.ModePartition.
	ModePartition

	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	MealPlanNumber string         `gorm:"uniqueIndex;not null" json:"mealPlanNumber"`
	CustomerID     uuid.UUID      `gorm:"type:uuid;index;not null" json:"customerId"`
	ChefID         uuid.UUID      `gorm:"type:uuid;index;not null" json:"chefId"`
	Status         MealPlanStatus `gorm:"type:varchar(24);index;default:'pending_chef'" json:"status"`

	StartDate time.Time `gorm:"index" json:"startDate"`
	EndDate   time.Time `gorm:"index" json:"endDate"`

	// Money snapshot for the full REQUESTED set (the advance). The accepted
	// subset's totals are derived from the days; the declined remainder is refunded.
	//
	// Total = Subtotal + PlatformFee + Tax + delivery. Delivery is NOT stored — it is
	// derived as Total − Subtotal − PlatformFee − Tax (planDeliveryTotal), so anything
	// added to Total must also be subtracted there or the delivery share silently
	// absorbs it.
	Subtotal float64 `gorm:"default:0" json:"subtotal"`
	// PlatformFee is the platform's own charge on the food subtotal, snapshotted at
	// booking from PlatformFeePercent. Shown to the customer as "Platform fee" on the
	// receipt. Non-refundable on a customer-initiated skip (it IS returned by the
	// make-whole perDayGross refund when the platform or chef is at fault).
	PlatformFee float64 `gorm:"default:0" json:"platformFee"`
	// TaxRate freezes the GST percent applied at booking so the spawned per-day orders
	// can report a rate alongside the amount. Without it a receipt renders the tax
	// line as "IGST (0%)" against a non-zero figure.
	TaxRate float64 `gorm:"default:0" json:"taxRate"`
	// Tax is the plan's TOTAL tax across every supply.
	//
	// On plans booked before tax was split per supply it is food GST alone, which
	// is why perDayFoodGST falls back to it: the chef day-transfer, the TDS
	// reporting and the credit notes on those plans were all reconciled against
	// that figure and must not be restated.
	Tax float64 `gorm:"default:0" json:"tax"`
	// Per-supply snapshot, mirroring Order. TaxFood is the ONLY one the chef's
	// payout is withheld against — the tax on the platform fee and on delivery is
	// the platform's own output tax.
	TaxFood             float64 `gorm:"default:0" json:"taxFood"`
	TaxService          float64 `gorm:"default:0" json:"taxService"`
	TaxDelivery         float64 `gorm:"default:0" json:"taxDelivery"`
	TaxRateFood         float64 `gorm:"default:0" json:"taxRateFood"`
	TaxRateService      float64 `gorm:"default:0" json:"taxRateService"`
	TaxRateDelivery     float64 `gorm:"default:0" json:"taxRateDelivery"`
	TaxServiceInclusive bool    `gorm:"default:false" json:"taxServiceInclusive"`
	Total               float64 `gorm:"not null" json:"total"`
	Currency            string  `gorm:"type:varchar(3);default:'INR'" json:"currency"`

	// Escrow links (#194): the upfront capture that funds the held chef payouts.
	EscrowPaymentID string `gorm:"" json:"escrowPaymentId,omitempty"`
	// Covered by a PARTIAL unique index (WHERE gateway_order_id <> '') in database.go's
	// postMigrate block (#395·1) — unique when set, empty for an unpaid/handshake plan.
	// Holds the gateway order id whichever rail took the money: Cashfree stamps its
	// own order id here exactly as orders do, so read it with PaymentProvider.
	GatewayOrderID string `gorm:"" json:"gatewayOrderId,omitempty"`
	// Which gateway actually captured the advance. Stored, never inferred: a plan
	// captured on Cashfree must refund on Cashfree, and the reverse strands the money.
	PaymentProvider string `gorm:"type:varchar(20);default:'razorpay'" json:"paymentProvider,omitempty"`

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
	// Live/test data partition. See models.ModePartition.
	ModePartition

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

	// v2 meal-plan refund workflow (docs/meal-plan-refund-flow-design.md), gated by
	// MEALPLAN_REFUND_FLOW_V2_ENABLED. The day stays in `skip_req` status through the flow;
	// these fields track the sub-state (the Status column is varchar(12), too short for the
	// stage names). RefundStage: pending_chef (≤12h, awaiting the chef's Full/Half/None),
	// pending_admin (chef chose, awaiting admin pay), resolved (terminal). ChefRefundChoice is
	// the chef's proportion; RefundDestination is the admin's payout target. Empty when the day
	// is not in the v2 flow (or was auto-approved >12h, which resolves straight to refunded).
	RefundStage      MealPlanRefundStage `gorm:"type:varchar(16);default:''" json:"refundStage,omitempty"`
	ChefRefundChoice RefundProportion    `gorm:"type:varchar(8);default:''" json:"chefRefundChoice,omitempty"`
	// RefundPercent is the AUTHORITATIVE agreed refund percentage, 0–100 (#834 v3).
	// It replaces the three-value ChefRefundChoice enum, which could not express a
	// floor or anything between half and full; that column is still written as a
	// coarse label for legacy readers, but every amount is computed from this.
	// nil ⇒ not yet decided (fall back to the enum for pre-v3 rows — RefundPercentOf).
	RefundPercent *int `gorm:"" json:"refundPercent,omitempty"`
	// RefundFloorPercent pins the tier floor AT THE MOMENT THE REQUEST WAS RAISED, so
	// the chef cannot shrink their own obligation by sitting on the decision until the
	// lead time drops into a lower band. The server rejects any decision below it.
	RefundFloorPercent *int              `gorm:"" json:"refundFloorPercent,omitempty"`
	RefundDestination  RefundDestination `gorm:"type:varchar(8);default:''" json:"refundDestination,omitempty"`
	// RefundDecisionBy is when the chef's window to price this refund runs out. Past it the
	// deadline sweep agrees 100% on the customer's behalf, so an unresponsive kitchen cannot
	// sit on someone's money. Set only while pending_chef.
	RefundDecisionBy *time.Time `gorm:"index" json:"refundDecisionBy,omitempty"`

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

	// RefundAmount is the money that actually went back for this day, computed for the
	// response rather than stored (services.AnnotateMealPlanRefunds). Without it a cancelled
	// plan could only tell the customer THAT a refund happened, never how much (#1041).
	RefundAmount *float64 `gorm:"-" json:"refundAmount,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// RefundProportion is the COARSE LABEL for a day's agreed refund. Under v2 it was the whole
// decision — Full (100%), Half (50%), None (0) — computed against a fee/GST-excluded base.
//
// DEPRECATED as the decision itself since refund policy v3 (#834): the chef now agrees any
// percentage from the tier floor to 100, which this enum cannot express, and the base is the
// full amount the customer paid. MealPlanDay.RefundPercent is authoritative; this is still
// written alongside it (Partial for anything that isn't exactly 100/50/0) so pre-v3 readers
// and historical rows keep working. Empty means no decision yet.
type RefundProportion string

const (
	RefundProportionFull    RefundProportion = "full"
	RefundProportionHalf    RefundProportion = "half"
	RefundProportionNone    RefundProportion = "none"
	RefundProportionPartial RefundProportion = "partial" // v3: any other percentage
)

// RefundProportionLabel is the coarse legacy label for an agreed percentage.
func RefundProportionLabel(percent int) RefundProportion {
	switch percent {
	case 100:
		return RefundProportionFull
	case 50:
		return RefundProportionHalf
	case 0:
		return RefundProportionNone
	default:
		return RefundProportionPartial
	}
}

// LegacyRefundPercent maps a pre-v3 enum value back to its percentage, so a day decided
// before v3 shipped still prices correctly. Anything unrecognised is 0 (refund nothing),
// which is the safe direction — money is never moved off an unreadable decision.
func LegacyRefundPercent(p RefundProportion) int {
	switch p {
	case RefundProportionFull:
		return 100
	case RefundProportionHalf:
		return 50
	default:
		return 0
	}
}

// RefundPercentOf resolves the agreed refund percentage for a day: the v3 column when set,
// otherwise the pre-v3 enum. The single reader every money path goes through.
func (d *MealPlanDay) RefundPercentOf() int {
	if d.RefundPercent != nil {
		return *d.RefundPercent
	}
	return LegacyRefundPercent(d.ChefRefundChoice)
}

// MealPlanRefundStage is the v2 sub-state of a skip_req day (the Status column is too short to
// hold these names, so they live in their own column).
type MealPlanRefundStage string

const (
	MPRefundPendingChef     MealPlanRefundStage = "pending_chef"     // ≤12h, awaiting chef Full/Half/None
	MPRefundPendingCustomer MealPlanRefundStage = "pending_customer" // amount agreed, awaiting the customer's medium choice (RBI)
	MPRefundPendingAdmin    MealPlanRefundStage = "pending_admin"    // customer chose ORIGINAL, awaiting the admin to execute the gateway refund
	MPRefundResolved        MealPlanRefundStage = "resolved"         // terminal (refunded, or no-refund)
)

// RefundDestination is where the admin pays a meal-plan / group-order refund. Wallet is instant
// (closed-loop, spend-on-HomeChef, non-withdrawable); Source reverses to the original card/UPI
// via the gateway (RBI ~5–7 business days).
type RefundDestination string

const (
	RefundDestinationWallet RefundDestination = "wallet"
	RefundDestinationSource RefundDestination = "source"
)

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
