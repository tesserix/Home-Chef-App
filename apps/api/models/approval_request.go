package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type ApprovalRequestType string

const (
	ApprovalKitchenOnboarding    ApprovalRequestType = "kitchen_onboarding"
	ApprovalMenuItemNew          ApprovalRequestType = "menu_item_new"
	ApprovalMenuItemUpdate       ApprovalRequestType = "menu_item_update"
	ApprovalDocumentVerification ApprovalRequestType = "document_verification"
	ApprovalPricingChange        ApprovalRequestType = "pricing_change"
	ApprovalKitchenUpdate        ApprovalRequestType = "kitchen_update"
	ApprovalDriverOnboarding     ApprovalRequestType = "driver_onboarding"
	ApprovalDriverDocument       ApprovalRequestType = "driver_document"
)

type ApprovalRequestStatus string

const (
	ApprovalPending       ApprovalRequestStatus = "pending"
	ApprovalApproved      ApprovalRequestStatus = "approved"
	ApprovalRejected      ApprovalRequestStatus = "rejected"
	ApprovalInfoRequested ApprovalRequestStatus = "info_requested"
	ApprovalCancelled     ApprovalRequestStatus = "cancelled"
)

type ApprovalRequest struct {
	ID            uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Type          ApprovalRequestType   `gorm:"type:varchar(50);not null;index" json:"type"`
	Status        ApprovalRequestStatus `gorm:"type:varchar(30);default:'pending';index" json:"status"`
	Priority      string                `gorm:"type:varchar(20);default:'normal'" json:"priority"` // low, normal, high, urgent
	ChefID        *uuid.UUID            `gorm:"type:uuid;index" json:"chefId,omitempty"`
	PartnerID     *uuid.UUID            `gorm:"type:uuid;index" json:"partnerId,omitempty"`
	SubmittedByID uuid.UUID             `gorm:"type:uuid;not null" json:"submittedById"`
	ReviewedByID  *uuid.UUID            `gorm:"type:uuid" json:"reviewedById,omitempty"`
	EntityType    string                `gorm:"type:varchar(50)" json:"entityType"` // chef_profile, menu_item, chef_document
	EntityID      uuid.UUID             `gorm:"type:uuid" json:"entityId"`
	Title         string                `gorm:"not null" json:"title"`
	Description   string                `gorm:"type:text" json:"description"`
	SubmittedData string                `gorm:"type:jsonb" json:"submittedData"`
	AdminNotes    string                `gorm:"type:text" json:"adminNotes,omitempty"`
	ReviewedAt    *time.Time            `json:"reviewedAt,omitempty"`
	ExpiresAt     *time.Time            `json:"expiresAt,omitempty"`

	// ── Reminders / escalation (#697) ────────────────────────────────────────
	// A chef whose request sits unattended has no lever other than contacting
	// support. These let them bump it themselves, on a cooldown, and escalate it
	// if it keeps being ignored.

	// ReminderCount is how many times the submitter has bumped this request.
	// >= 1 pins it to the top of the admin queue with a warning marker;
	// >= ReminderEscalationThreshold means escalated.
	ReminderCount int `gorm:"not null;default:0;index" json:"reminderCount"`
	// LastRemindedAt is when the most recent bump happened. It — not CreatedAt —
	// is the base for the NEXT cooldown.
	LastRemindedAt *time.Time `json:"lastRemindedAt,omitempty"`
	// EscalatedAt is stamped the moment ReminderCount reaches the threshold, and
	// never cleared: it is the audit fact "this was escalated", not a live flag.
	// The admin escalation panel reads it.
	EscalatedAt *time.Time `gorm:"index" json:"escalatedAt,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updatedAt"`

	// Relations
	Chef        ChefProfile     `gorm:"foreignKey:ChefID" json:"chef,omitempty"`
	Partner     DeliveryPartner `gorm:"foreignKey:PartnerID" json:"partner,omitempty"`
	SubmittedBy User            `gorm:"foreignKey:SubmittedByID" json:"submittedBy,omitempty"`
	ReviewedBy  *User           `gorm:"foreignKey:ReviewedByID" json:"reviewedBy,omitempty"`

	// Computed display fields (not persisted) — populated by the admin
	// handlers from the preloaded relations so every approval, of any type,
	// clearly shows who submitted it and which kitchen it belongs to.
	RequestedByName  string `gorm:"-" json:"requestedByName,omitempty"`
	RequestedByEmail string `gorm:"-" json:"requestedByEmail,omitempty"`
	KitchenName      string `gorm:"-" json:"kitchenName,omitempty"`

	// NextRemindAt is when the submitter may bump this again — computed, not
	// stored. Sent so the app can render a real countdown instead of offering a
	// button that fails, and so the RULE lives in one place rather than being
	// re-implemented (and drifting) in each client.
	NextRemindAt *time.Time `gorm:"-" json:"nextRemindAt,omitempty"`
	// CanRemind is NextRemindAt evaluated for the caller, for clients that would
	// otherwise compare timestamps against their own (possibly skewed) clock.
	CanRemind bool `gorm:"-" json:"canRemind"`
}

// Reminder cadence (#697).
const (
	// ReminderEscalationThreshold — bumps at or beyond this mean the request is
	// escalated and belongs in the admin escalation panel.
	ReminderEscalationThreshold = 3
	// ReminderCooldown is the wait before each of the first
	// ReminderEscalationThreshold bumps: a day is long enough that a bump means
	// "genuinely unattended" rather than impatience.
	ReminderCooldown = 24 * time.Hour
	// ReminderCooldownEscalated is the wait once escalated. Shorter, because by
	// then the request has been ignored for 3+ days and the chef is blocked.
	ReminderCooldownEscalated = 6 * time.Hour
)

// IsEscalated reports whether this request has been bumped enough to be escalated.
func (a *ApprovalRequest) IsEscalated() bool {
	return a.ReminderCount >= ReminderEscalationThreshold
}

// AcceptsReminders reports whether bumping is meaningful. A decided request
// (approved/rejected/cancelled) is nobody's blocker, so reminding is refused
// rather than silently doing nothing. info_requested stays remindable: the ball
// can be back with the admin after the chef replies.
func (a *ApprovalRequest) AcceptsReminders() bool {
	return a.Status == ApprovalPending || a.Status == ApprovalInfoRequested
}

// ReminderCooldownFor returns the wait before the nth bump (1-based).
//
// The first ReminderEscalationThreshold bumps are a day apart; after that the
// request is escalated and the cadence tightens to 6h. n <= 0 is treated as the
// first bump — a caller asking "when is the next one" for a never-reminded
// request is asking about bump #1.
func ReminderCooldownFor(n int) time.Duration {
	if n > ReminderEscalationThreshold {
		return ReminderCooldownEscalated
	}
	return ReminderCooldown
}

// NextReminderAt returns the earliest time the submitter may bump again.
//
// The base is the LAST BUMP if there is one, else creation: the cadence is
// "24h since I last asked", not "24h since it was raised" — otherwise every
// bump after the first would be instantly available.
func (a *ApprovalRequest) NextReminderAt() time.Time {
	base := a.CreatedAt
	if a.LastRemindedAt != nil {
		base = *a.LastRemindedAt
	}
	return base.Add(ReminderCooldownFor(a.ReminderCount + 1))
}

// CanRemindAt reports whether the submitter may bump at time `now`.
func (a *ApprovalRequest) CanRemindAt(now time.Time) bool {
	if !a.AcceptsReminders() {
		return false
	}
	return !now.Before(a.NextReminderAt())
}

// PopulateReminderFields fills the computed NextRemindAt / CanRemind for a
// response. Kept next to the rule so a client never has to re-derive it.
func (a *ApprovalRequest) PopulateReminderFields(now time.Time) {
	if !a.AcceptsReminders() {
		a.NextRemindAt = nil
		a.CanRemind = false
		return
	}
	next := a.NextReminderAt()
	a.NextRemindAt = &next
	a.CanRemind = a.CanRemindAt(now)
}

// PopulateDisplayFields fills RequestedByName / RequestedByEmail / KitchenName
// from the preloaded SubmittedBy, Chef and Partner relations. Prefers the
// submitter's own identity, falling back to the chef's / partner's linked user.
// Safe to call with zero-valued relations (fields just stay empty). Call after
// Preload("Chef.User"), Preload("Partner.User") and Preload("SubmittedBy").
func (a *ApprovalRequest) PopulateDisplayFields() {
	name := strings.TrimSpace(a.SubmittedBy.FirstName + " " + a.SubmittedBy.LastName)
	email := a.SubmittedBy.Email

	if a.ChefID != nil {
		if a.Chef.BusinessName != "" {
			a.KitchenName = a.Chef.BusinessName
		}
		if name == "" {
			name = strings.TrimSpace(a.Chef.User.FirstName + " " + a.Chef.User.LastName)
		}
		if email == "" {
			email = a.Chef.User.Email
		}
	} else if a.PartnerID != nil {
		if name == "" {
			name = strings.TrimSpace(a.Partner.User.FirstName + " " + a.Partner.User.LastName)
		}
		if email == "" {
			email = a.Partner.User.Email
		}
	}

	a.RequestedByName = name
	a.RequestedByEmail = email
}

// ApprovalRequestHistory tracks state transitions
type ApprovalRequestHistory struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ApprovalID  uuid.UUID `gorm:"type:uuid;not null;index" json:"approvalId"`
	FromStatus  string    `gorm:"type:varchar(30)" json:"fromStatus"`
	ToStatus    string    `gorm:"type:varchar(30);not null" json:"toStatus"`
	ChangedByID uuid.UUID `gorm:"type:uuid;not null" json:"changedById"`
	Notes       string    `gorm:"type:text" json:"notes,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`

	ChangedBy User `gorm:"foreignKey:ChangedByID" json:"changedBy,omitempty"`
}
