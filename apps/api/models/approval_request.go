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
	CreatedAt     time.Time             `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time             `gorm:"autoUpdateTime" json:"updatedAt"`

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
