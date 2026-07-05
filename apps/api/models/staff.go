package models

import (
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// StaffRole represents the role a staff member holds in internal portals
type StaffRole string

const (
	StaffRoleSuperAdmin   StaffRole = "super_admin"
	StaffRoleAdmin        StaffRole = "admin"
	StaffRoleFleetManager StaffRole = "fleet_manager"
	StaffRoleDeliveryOps  StaffRole = "delivery_ops"
	StaffRoleSupport      StaffRole = "support"
)

// StaffRoleGrantsPlatformAdmin reports whether accepting an invitation for this
// StaffRole sets User.Role = RoleAdmin — i.e. full admin-portal access, incl.
// money movement. Only fleet_manager / delivery_ops are delivery-portal roles
// (RoleDelivery); everything else is platform-admin. This is the SINGLE source
// of truth shared by AcceptInvitation (the role mapping) and CreateInvitation
// (the security gate) so they can never drift (audit #4).
func StaffRoleGrantsPlatformAdmin(role StaffRole) bool {
	return role != StaffRoleFleetManager && role != StaffRoleDeliveryOps
}

// StaffRoleIsPlatformAdmin reports whether a StaffRole is one of the actual
// platform administrators allowed to invite admin-tier staff.
func StaffRoleIsPlatformAdmin(role StaffRole) bool {
	return role == StaffRoleSuperAdmin || role == StaffRoleAdmin
}

// StaffPermission represents a granular permission
type StaffPermission string

const (
	// Admin portal permissions
	SPViewDashboard StaffPermission = "dashboard:view"
	SPManageUsers   StaffPermission = "users:manage"
	SPViewUsers     StaffPermission = "users:view"
	SPManageChefs   StaffPermission = "chefs:manage"
	SPVerifyChefs   StaffPermission = "chefs:verify"
	SPViewChefs     StaffPermission = "chefs:view"
	SPManageOrders  StaffPermission = "orders:manage"
	SPViewOrders    StaffPermission = "orders:view"
	SPRefundOrders  StaffPermission = "orders:refund"
	// SPManagePayouts gates the escrow payout-release surface
	// (/admin/payouts/*: release, withhold, reverse, bulk, pending queue).
	// Moving vendor payout money is finance-sensitive, so it is granted to
	// super_admin only — NOT to the general admin role (#515 / #461 step 2).
	SPManagePayouts   StaffPermission = "payouts:manage"
	SPViewAnalytics   StaffPermission = "analytics:view"
	SPManageSettings  StaffPermission = "settings:manage"
	SPViewSettings    StaffPermission = "settings:view"
	SPManageApprovals StaffPermission = "approvals:manage"
	SPViewApprovals   StaffPermission = "approvals:view"
	SPManageStaff     StaffPermission = "staff:manage"
	SPInviteStaff     StaffPermission = "staff:invite"
	SPViewStaff       StaffPermission = "staff:view"
	SPModerateContent StaffPermission = "content:moderate"

	// Delivery portal permissions
	SPViewDeliveryDashboard  StaffPermission = "delivery:dashboard"
	SPManageFleet            StaffPermission = "fleet:manage"
	SPViewFleet              StaffPermission = "fleet:view"
	SPManageDeliveryPartners StaffPermission = "delivery_partners:manage"
	SPVerifyDeliveryPartners StaffPermission = "delivery_partners:verify"
	SPViewDeliveryPartners   StaffPermission = "delivery_partners:view"
	SPViewDeliveryOrders     StaffPermission = "delivery_orders:view"
	SPAssignDeliveries       StaffPermission = "deliveries:assign"
	SPViewDeliveryAnalytics  StaffPermission = "delivery_analytics:view"
	SPViewDeliveryEarnings   StaffPermission = "delivery_earnings:view"
	SPManageZones            StaffPermission = "zones:manage"
	SPViewZones              StaffPermission = "zones:view"
)

// DefaultStaffPermissions defines what each staff role can do
var DefaultStaffPermissions = map[StaffRole][]StaffPermission{
	StaffRoleSuperAdmin: {
		// Everything
		SPViewDashboard, SPManageUsers, SPViewUsers,
		SPManageChefs, SPVerifyChefs, SPViewChefs,
		SPManageOrders, SPViewOrders, SPRefundOrders,
		SPManagePayouts,
		SPViewAnalytics, SPManageSettings, SPViewSettings,
		SPManageApprovals, SPViewApprovals,
		SPManageStaff, SPInviteStaff, SPViewStaff,
		SPModerateContent,
		SPViewDeliveryDashboard, SPManageFleet, SPViewFleet,
		SPManageDeliveryPartners, SPVerifyDeliveryPartners, SPViewDeliveryPartners,
		SPViewDeliveryOrders, SPAssignDeliveries,
		SPViewDeliveryAnalytics, SPViewDeliveryEarnings,
		SPManageZones, SPViewZones,
	},
	StaffRoleAdmin: {
		SPViewDashboard, SPManageUsers, SPViewUsers,
		SPManageChefs, SPVerifyChefs, SPViewChefs,
		SPManageOrders, SPViewOrders, SPRefundOrders,
		SPViewAnalytics, SPViewSettings,
		SPManageApprovals, SPViewApprovals,
		SPInviteStaff, SPViewStaff,
		SPModerateContent,
	},
	StaffRoleFleetManager: {
		SPViewDeliveryDashboard, SPManageFleet, SPViewFleet,
		SPManageDeliveryPartners, SPVerifyDeliveryPartners, SPViewDeliveryPartners,
		SPViewDeliveryOrders, SPAssignDeliveries,
		SPViewDeliveryAnalytics, SPViewDeliveryEarnings,
		SPInviteStaff, SPViewStaff,
		SPManageZones, SPViewZones,
	},
	StaffRoleDeliveryOps: {
		SPViewDeliveryDashboard, SPViewFleet,
		SPViewDeliveryPartners,
		SPViewDeliveryOrders, SPAssignDeliveries,
		SPViewDeliveryAnalytics, SPViewDeliveryEarnings,
	},
	StaffRoleSupport: {
		SPViewDashboard, SPViewUsers, SPViewChefs,
		SPViewOrders, SPViewApprovals,
		SPViewDeliveryDashboard, SPViewDeliveryPartners, SPViewDeliveryOrders,
	},
}

// StaffMember represents an internal staff member (admin/delivery portals)
type StaffMember struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID       uuid.UUID      `gorm:"type:uuid;uniqueIndex;not null" json:"userId"`
	StaffRole    StaffRole      `gorm:"type:varchar(30);not null" json:"staffRole"`
	Permissions  []byte         `gorm:"type:jsonb;default:'[]'" json:"-"` // Custom permission overrides (JSONB)
	Department   string         `gorm:"type:varchar(50)" json:"department"`
	Title        string         `gorm:"type:varchar(100)" json:"title"`
	InvitedByID  *uuid.UUID     `gorm:"type:uuid" json:"invitedById,omitempty"`
	IsActive     bool           `gorm:"default:true" json:"isActive"`
	JoinedAt     time.Time      `gorm:"autoCreateTime" json:"joinedAt"`
	LastActiveAt *time.Time     `gorm:"" json:"lastActiveAt,omitempty"`
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// Relationships
	User      User  `gorm:"foreignKey:UserID" json:"user,omitempty"`
	InvitedBy *User `gorm:"foreignKey:InvitedByID" json:"invitedBy,omitempty"`
}

// InvitationStatus tracks the state of a staff invitation
type InvitationStatus string

const (
	InvitationPending  InvitationStatus = "pending"
	InvitationAccepted InvitationStatus = "accepted"
	InvitationExpired  InvitationStatus = "expired"
	InvitationRevoked  InvitationStatus = "revoked"
)

// StaffInvitation represents a pending invitation for someone to join as staff
type StaffInvitation struct {
	ID           uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Email        string           `gorm:"not null;index" json:"email"`
	StaffRole    StaffRole        `gorm:"type:varchar(30);not null" json:"staffRole"`
	Department   string           `gorm:"type:varchar(50)" json:"department"`
	Title        string           `gorm:"type:varchar(100)" json:"title"`
	Token        string           `gorm:"uniqueIndex;not null" json:"-"`
	InvitedByID  uuid.UUID        `gorm:"type:uuid;not null" json:"invitedById"`
	Status       InvitationStatus `gorm:"type:varchar(20);default:'pending'" json:"status"`
	Message      string           `gorm:"type:text" json:"message,omitempty"`
	ExpiresAt    time.Time        `gorm:"not null" json:"expiresAt"`
	AcceptedAt   *time.Time       `gorm:"" json:"acceptedAt,omitempty"`
	AcceptedByID *uuid.UUID       `gorm:"type:uuid" json:"acceptedById,omitempty"`
	CreatedAt    time.Time        `gorm:"autoCreateTime" json:"createdAt"`

	// Relationships
	InvitedBy  User  `gorm:"foreignKey:InvitedByID" json:"invitedBy,omitempty"`
	AcceptedBy *User `gorm:"foreignKey:AcceptedByID" json:"acceptedBy,omitempty"`
}

// DTOs

type StaffMemberResponse struct {
	ID           uuid.UUID             `json:"id"`
	UserID       uuid.UUID             `json:"userId"`
	Email        string                `json:"email"`
	FirstName    string                `json:"firstName"`
	LastName     string                `json:"lastName"`
	Avatar       string                `json:"avatar,omitempty"`
	Phone        string                `json:"phone,omitempty"`
	StaffRole    StaffRole             `json:"staffRole"`
	Department   string                `json:"department"`
	Title        string                `json:"title"`
	IsActive     bool                  `json:"isActive"`
	JoinedAt     time.Time             `json:"joinedAt"`
	LastActiveAt *time.Time            `json:"lastActiveAt,omitempty"`
	Permissions  []StaffPermission     `json:"permissions"`
	InvitedBy    *StaffInviterResponse `json:"invitedBy,omitempty"`
}

type StaffInviterResponse struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

type StaffInvitationResponse struct {
	ID         uuid.UUID            `json:"id"`
	Email      string               `json:"email"`
	StaffRole  StaffRole            `json:"staffRole"`
	Department string               `json:"department"`
	Title      string               `json:"title"`
	Status     InvitationStatus     `json:"status"`
	Message    string               `json:"message,omitempty"`
	InviteURL  string               `json:"inviteUrl,omitempty"`
	ExpiresAt  time.Time            `json:"expiresAt"`
	AcceptedAt *time.Time           `json:"acceptedAt,omitempty"`
	InvitedBy  StaffInviterResponse `json:"invitedBy"`
	CreatedAt  time.Time            `json:"createdAt"`
}

func (s *StaffMember) ToResponse() StaffMemberResponse {
	resp := StaffMemberResponse{
		ID:           s.ID,
		UserID:       s.UserID,
		StaffRole:    s.StaffRole,
		Department:   s.Department,
		Title:        s.Title,
		IsActive:     s.IsActive,
		JoinedAt:     s.JoinedAt,
		LastActiveAt: s.LastActiveAt,
		Permissions:  DefaultStaffPermissions[s.StaffRole],
	}
	if s.User.ID != uuid.Nil {
		resp.Email = s.User.Email
		resp.FirstName = s.User.FirstName
		resp.LastName = s.User.LastName
		resp.Avatar = s.User.Avatar
		resp.Phone = s.User.Phone
	}
	if s.InvitedBy != nil && s.InvitedBy.ID != uuid.Nil {
		resp.InvitedBy = &StaffInviterResponse{
			ID:    s.InvitedBy.ID,
			Name:  s.InvitedBy.FirstName + " " + s.InvitedBy.LastName,
			Email: s.InvitedBy.Email,
		}
	}
	return resp
}

// ToResponse builds the public DTO. The accept URL contains a live token
// and must NOT appear in list responses where any staff reader could
// harvest it; pass includeURL=true only on the immediate-after-create
// response so the inviter can copy the link once.
func (i *StaffInvitation) ToResponse(baseURL string, includeURL bool) StaffInvitationResponse {
	resp := StaffInvitationResponse{
		ID:         i.ID,
		Email:      i.Email,
		StaffRole:  i.StaffRole,
		Department: i.Department,
		Title:      i.Title,
		Status:     i.Status,
		Message:    i.Message,
		ExpiresAt:  i.ExpiresAt,
		AcceptedAt: i.AcceptedAt,
		CreatedAt:  i.CreatedAt,
	}
	if includeURL && i.Status == InvitationPending {
		resp.InviteURL = baseURL + "/invite/accept?token=" + i.Token
	}
	if i.InvitedBy.ID != uuid.Nil {
		resp.InvitedBy = StaffInviterResponse{
			ID:    i.InvitedBy.ID,
			Name:  i.InvitedBy.FirstName + " " + i.InvitedBy.LastName,
			Email: i.InvitedBy.Email,
		}
	}
	return resp
}

// HasPermission checks if a staff member's role includes a permission
func (s *StaffMember) HasPermission(perm StaffPermission) bool {
	perms, ok := DefaultStaffPermissions[s.StaffRole]
	if !ok {
		return false
	}
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}

// CanAccessAdminPortal checks if the staff role grants admin portal access
func (s *StaffMember) CanAccessAdminPortal() bool {
	switch s.StaffRole {
	case StaffRoleSuperAdmin, StaffRoleAdmin, StaffRoleSupport:
		return true
	}
	return false
}

// CanAccessDeliveryPortal checks if the staff role grants delivery portal access
func (s *StaffMember) CanAccessDeliveryPortal() bool {
	switch s.StaffRole {
	case StaffRoleSuperAdmin, StaffRoleFleetManager, StaffRoleDeliveryOps:
		return true
	}
	return false
}

// defaultSuperAdminEmails is the built-in super admin allowlist used when the
// SUPER_ADMIN_EMAILS env var is unset or empty. Preserves the historical behavior.
var defaultSuperAdminEmails = []string{
	"samyak.rout@gmail.com",
	"mahesh.sangawar@gmail.com",
	"unidevidp@gmail.com",
}

// SuperAdminEmails is the effective super admin allowlist (always full access).
// Sourced from the comma-separated SUPER_ADMIN_EMAILS env var at startup,
// falling back to defaultSuperAdminEmails when that env var is unset or empty.
// Entries are trimmed and lowercased so IsSuperAdminEmail compares
// case-insensitively.
var SuperAdminEmails = loadSuperAdminEmails()

// loadSuperAdminEmails reads SUPER_ADMIN_EMAILS (comma-separated), falling back
// to the hardcoded default when unset/empty so today's behavior is preserved.
func loadSuperAdminEmails() []string {
	if out := normalizeEmails(strings.Split(os.Getenv("SUPER_ADMIN_EMAILS"), ",")); len(out) > 0 {
		return out
	}
	return normalizeEmails(defaultSuperAdminEmails)
}

// normalizeEmails trims and lowercases each entry, dropping blanks.
func normalizeEmails(in []string) []string {
	out := make([]string, 0, len(in))
	for _, e := range in {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			out = append(out, e)
		}
	}
	return out
}

// IsSuperAdminEmail checks if an email is a super admin (case-insensitive).
func IsSuperAdminEmail(email string) bool {
	want := strings.ToLower(strings.TrimSpace(email))
	for _, e := range SuperAdminEmails {
		if e == want {
			return true
		}
	}
	return false
}
