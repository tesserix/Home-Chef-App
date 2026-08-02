package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// customer_risk.go — customer-side refund-abuse detection (#937). Every guard on the
// issue-refund path is scoped to ONE order, so a customer who claims a little on many
// orders across many chefs is invisible. These three tables give the platform a
// customer-scoped view: an append-only event ledger, a derived score, and an audit of
// every human decision taken off the back of it.

// CustomerRiskEventKind is why an event was recorded. Money-out kinds raise the score;
// RiskOrderPlaced is the denominator that keeps every rate honest.
type CustomerRiskEventKind string

const (
	RiskOrderPlaced           CustomerRiskEventKind = "order_placed"
	RiskIssueReported         CustomerRiskEventKind = "issue_reported"
	RiskIssueAutoRefunded     CustomerRiskEventKind = "issue_auto_refunded"
	RiskIssueResolved         CustomerRiskEventKind = "issue_resolved"
	RiskIssueRejected         CustomerRiskEventKind = "issue_rejected"
	RiskNoEvidenceClaim       CustomerRiskEventKind = "no_evidence_claim"
	RiskLateWindowClaim       CustomerRiskEventKind = "late_window_claim"
	RiskCancelLate            CustomerRiskEventKind = "cancel_late"
	RiskMealPlanRefund        CustomerRiskEventKind = "mealplan_refund"
	RiskDeliveryFailureClaim  CustomerRiskEventKind = "delivery_failure_claim"
	RiskDeliveryFaultCustomer CustomerRiskEventKind = "delivery_fault_customer"
	RiskChargeback            CustomerRiskEventKind = "chargeback"
)

// RiskBand is the derived severity of a profile. Automatic — computed from the ledger,
// never set by hand.
type RiskBand string

const (
	BandNormal   RiskBand = "normal"
	BandWatch    RiskBand = "watch"
	BandElevated RiskBand = "elevated"
	BandSevere   RiskBand = "severe"
)

// bandRank orders the bands so callers can compare severity without a switch.
func bandRank(b RiskBand) int {
	switch b {
	case BandWatch:
		return 1
	case BandElevated:
		return 2
	case BandSevere:
		return 3
	}
	return 0
}

// AtLeast reports whether b is at or above min in severity.
func (b RiskBand) AtLeast(min RiskBand) bool { return bandRank(b) >= bandRank(min) }

// ValidRiskBand reports whether b is a band this API accepts.
func ValidRiskBand(b RiskBand) bool {
	switch b {
	case BandNormal, BandWatch, BandElevated, BandSevere:
		return true
	}
	return false
}

// RiskStatus is the human-owned disposition of a profile. Everything except
// RiskStatusFlagged is set by an admin; the engine never blocks anyone by itself.
type RiskStatus string

const (
	RiskStatusOK          RiskStatus = "ok"
	RiskStatusFlagged     RiskStatus = "flagged"      // engine raised it for review
	RiskStatusUnderReview RiskStatus = "under_review" // an admin picked it up
	RiskStatusRestricted  RiskStatus = "restricted"   // may order, may not file new claims
	RiskStatusBlocked     RiskStatus = "blocked"      // account suspension applied alongside
	RiskStatusCleared     RiskStatus = "cleared"      // investigated, no abuse found
)

// ValidRiskStatus reports whether s is a status an admin may set.
func ValidRiskStatus(s RiskStatus) bool {
	switch s {
	case RiskStatusOK, RiskStatusFlagged, RiskStatusUnderReview,
		RiskStatusRestricted, RiskStatusBlocked, RiskStatusCleared:
		return true
	}
	return false
}

// CustomerRiskEvent is one risk-bearing act by one customer.
//
// SourceKey is the natural key of the originating event ("issue:<id>", "order:<id>")
// and is UNIQUE, so a retried or concurrent write counts exactly once — the same
// guarantee ChefPenalty.SourceKey gives the chef-side levy. This ledger is the only
// source of truth: CustomerRiskProfile is a cache and is recomputable from these rows.
type CustomerRiskEvent struct {
	ID         uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CustomerID uuid.UUID             `gorm:"type:uuid;not null;index" json:"customerId"`
	Kind       CustomerRiskEventKind `gorm:"type:varchar(32);not null;index" json:"kind"`
	SourceKey  string                `gorm:"uniqueIndex;not null" json:"sourceKey"`

	OrderID *uuid.UUID `gorm:"type:uuid;index" json:"orderId,omitempty"`
	ChefID  *uuid.UUID `gorm:"type:uuid;index" json:"chefId,omitempty"`

	// Amount is money returned to the customer (0 for signal-only kinds); Weight is the
	// scoring weight frozen at write time so retuning the config never restates history.
	Amount float64 `gorm:"default:0" json:"amount"`
	Weight float64 `gorm:"default:0" json:"weight"`
	Reason string  `gorm:"type:text" json:"reason,omitempty"`

	OccurredAt time.Time `gorm:"not null;index" json:"occurredAt"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (CustomerRiskEvent) TableName() string { return "customer_risk_events" }

// BeforeCreate mints the id and defaults OccurredAt — the sqlite test harness has no
// gen_random_uuid(), and an unset OccurredAt would fall outside every scoring window.
func (e *CustomerRiskEvent) BeforeCreate(*gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now()
	}
	return nil
}

// CustomerRiskProfile is the derived per-customer view the admin queue reads, plus the
// human disposition. Counters are denormalised so the queue sorts without touching the
// ledger; Score and Band are always recomputable from customer_risk_events.
type CustomerRiskProfile struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`

	WindowDays int `gorm:"default:90" json:"windowDays"`

	OrdersInWindow        int `gorm:"default:0" json:"ordersInWindow"`
	IssuesReported        int `gorm:"default:0" json:"issuesReported"`
	IssuesAutoRefunded    int `gorm:"default:0" json:"issuesAutoRefunded"`
	IssuesRejected        int `gorm:"default:0" json:"issuesRejected"`
	CancellationsLate     int `gorm:"default:0" json:"cancellationsLate"`
	MealPlanRefunds       int `gorm:"default:0" json:"mealPlanRefunds"`
	DeliveryFailureClaims int `gorm:"default:0" json:"deliveryFailureClaims"`
	NoEvidenceClaims      int `gorm:"default:0" json:"noEvidenceClaims"`

	// Claim spread: how many distinct kitchens the claims hit, against how many were
	// ordered from. A wide spread is the signature of a serial claimer; a narrow one
	// usually means a single bad kitchen, which is a chef problem, not a customer one.
	DistinctChefsClaimed int `gorm:"default:0" json:"distinctChefsClaimed"`
	DistinctChefsOrdered int `gorm:"default:0" json:"distinctChefsOrdered"`

	RefundedAmount float64 `gorm:"default:0" json:"refundedAmount"`
	SpentAmount    float64 `gorm:"default:0" json:"spentAmount"`
	RefundedShare  float64 `gorm:"default:0" json:"refundedShare"`
	ClaimRate      float64 `gorm:"default:0" json:"claimRate"`

	Score float64  `gorm:"default:0;index" json:"score"`
	Band  RiskBand `gorm:"type:varchar(16);not null;default:'normal';index" json:"band"`

	Status       RiskStatus `gorm:"type:varchar(16);not null;default:'ok';index" json:"status"`
	StatusReason string     `gorm:"type:text" json:"statusReason,omitempty"`

	FlaggedAt  *time.Time `gorm:"" json:"flaggedAt,omitempty"`
	ReviewedAt *time.Time `gorm:"" json:"reviewedAt,omitempty"`
	ReviewedBy *uuid.UUID `gorm:"type:uuid" json:"reviewedBy,omitempty"`

	LastEventAt  *time.Time `gorm:"" json:"lastEventAt,omitempty"`
	RecomputedAt time.Time  `gorm:"" json:"recomputedAt"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (CustomerRiskProfile) TableName() string { return "customer_risk_profiles" }

func (p *CustomerRiskProfile) BeforeCreate(*gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// RestrictsClaims reports whether this profile's status bars the customer from filing
// a new order issue. Ordering is deliberately unaffected — a suspected abuser we are
// still happy to sell to is a different question from one we will auto-refund.
func (p *CustomerRiskProfile) RestrictsClaims() bool {
	return p.Status == RiskStatusRestricted || p.Status == RiskStatusBlocked
}

// DisputeSignal is the neutral heads-up a chef gets on an order from a customer with an
// elevated dispute history: a band word and one thing they can do about it. Never a
// score, a count, or a history — enough for a chef to protect themselves with evidence,
// not enough to judge or refuse a customer.
type DisputeSignal struct {
	Band   RiskBand `json:"band"`
	Advice string   `json:"advice"`
}

// CustomerRiskAction records one admin decision, with the score they actually saw.
// This is what answers a customer complaint, an internal dispute, or a regulator
// asking why an account was restricted.
type CustomerRiskAction struct {
	ID      uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID  uuid.UUID `gorm:"type:uuid;not null;index" json:"userId"`
	ActorID uuid.UUID `gorm:"type:uuid;not null;index" json:"actorId"`

	FromStatus RiskStatus `gorm:"type:varchar(16)" json:"fromStatus"`
	ToStatus   RiskStatus `gorm:"type:varchar(16);not null" json:"toStatus"`
	Note       string     `gorm:"type:text" json:"note,omitempty"`

	ScoreSnapshot float64  `gorm:"default:0" json:"scoreSnapshot"`
	BandSnapshot  RiskBand `gorm:"type:varchar(16)" json:"bandSnapshot"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

func (CustomerRiskAction) TableName() string { return "customer_risk_actions" }

func (a *CustomerRiskAction) BeforeCreate(*gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// interface guards — these three are AutoMigrated and rely on their table names.
var (
	_ schema.Tabler = CustomerRiskEvent{}
	_ schema.Tabler = CustomerRiskProfile{}
	_ schema.Tabler = CustomerRiskAction{}
)
