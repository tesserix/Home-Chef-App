package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"gorm.io/gorm"
)

// staffDB returns the database instance for staff queries
func staffDB() *gorm.DB {
	return database.DB
}

// Permission represents a specific action that can be performed
type Permission string

const (
	// Customer permissions
	PermBrowseChefs    Permission = "browse_chefs"
	PermViewMenu       Permission = "view_menu"
	PermPlaceOrder     Permission = "place_order"
	PermViewOwnOrders  Permission = "view_own_orders"
	PermWriteReview    Permission = "write_review"
	PermManageCart     Permission = "manage_cart"
	PermLikeComment    Permission = "like_comment"
	PermCreateCatering Permission = "create_catering_request"

	// Chef permissions
	PermManageMenu          Permission = "manage_menu"
	PermManageChefOrders    Permission = "manage_chef_orders"
	PermViewChefEarnings    Permission = "view_chef_earnings"
	PermManageChefProfile   Permission = "manage_chef_profile"
	PermRespondToReviews    Permission = "respond_to_reviews"
	PermSubmitCateringQuote Permission = "submit_catering_quote"

	// Delivery permissions
	PermViewDeliveries       Permission = "view_deliveries"
	PermAcceptDelivery       Permission = "accept_delivery"
	PermUpdateDelivery       Permission = "update_delivery"
	PermViewDeliveryEarnings Permission = "view_delivery_earnings"

	// Admin permissions
	PermViewAllUsers    Permission = "view_all_users"
	PermManageUsers     Permission = "manage_users"
	PermVerifyChefs     Permission = "verify_chefs"
	PermViewAllOrders   Permission = "view_all_orders"
	PermViewAnalytics   Permission = "view_analytics"
	PermManageSettings  Permission = "manage_settings"
	PermModerateContent Permission = "moderate_content"
)

// RolePermissions maps roles to their allowed permissions
var RolePermissions = map[models.UserRole][]Permission{
	models.RoleCustomer: {
		PermBrowseChefs,
		PermViewMenu,
		PermPlaceOrder,
		PermViewOwnOrders,
		PermWriteReview,
		PermManageCart,
		PermLikeComment,
		PermCreateCatering,
	},
	models.RoleChef: {
		// Inherit customer permissions
		PermBrowseChefs,
		PermViewMenu,
		// Chef-specific permissions
		PermManageMenu,
		PermManageChefOrders,
		PermViewChefEarnings,
		PermManageChefProfile,
		PermRespondToReviews,
		PermSubmitCateringQuote,
	},
	models.RoleDelivery: {
		PermViewDeliveries,
		PermAcceptDelivery,
		PermUpdateDelivery,
		PermViewDeliveryEarnings,
	},
	models.RoleFleetManager: {
		PermViewDeliveries,
		PermAcceptDelivery,
		PermUpdateDelivery,
		PermViewDeliveryEarnings,
		PermViewAllOrders,
		PermViewAnalytics,
	},
	models.RoleAdmin: {
		// Admin has all permissions
		PermBrowseChefs,
		PermViewMenu,
		PermPlaceOrder,
		PermViewOwnOrders,
		PermWriteReview,
		PermManageCart,
		PermLikeComment,
		PermCreateCatering,
		PermManageMenu,
		PermManageChefOrders,
		PermViewChefEarnings,
		PermManageChefProfile,
		PermRespondToReviews,
		PermSubmitCateringQuote,
		PermViewDeliveries,
		PermAcceptDelivery,
		PermUpdateDelivery,
		PermViewDeliveryEarnings,
		PermViewAllUsers,
		PermManageUsers,
		PermVerifyChefs,
		PermViewAllOrders,
		PermViewAnalytics,
		PermManageSettings,
		PermModerateContent,
	},
}

// HasPermission checks if a role has a specific permission
func HasPermission(role models.UserRole, permission Permission) bool {
	permissions, exists := RolePermissions[role]
	if !exists {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// RequireRole middleware ensures the user has one of the specified roles
func RequireRole(roles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := GetUserRole(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// RequirePermission middleware ensures the user has a specific permission
func RequirePermission(permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := GetUserRole(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		if !HasPermission(userRole, permission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// staffActorEmail returns the acting admin's email address.
//
// Prefers the hydrated users row, then falls back to the signed identity
// header. The fallback exists for admins who have no HomeChef users row at all
// — the tesserix.app console authenticates against its own directory — and is
// safe because X-User-Email is bound into the X-Internal-Auth MAC (#461).
// Returns "" when neither is present, which every caller must treat as "not a
// super admin".
func staffActorEmail(c *gin.Context) string {
	if user, ok := GetUser(c); ok && user.Email != "" {
		return user.Email
	}
	if v, ok := c.Get(CtxUserEmail); ok {
		if email, ok := v.(string); ok {
			return email
		}
	}
	return ""
}

// RequireStaffPermission checks that the authenticated user has a StaffMember
// record with the given permission. This enforces granular staff RBAC beyond
// basic role checks. The StaffMember is loaded from the DB and cached on the
// gin context as "staffMember" for downstream handlers.
func RequireStaffPermission(permission models.StaffPermission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if already loaded (avoid duplicate DB queries in a chain)
		if cached, ok := c.Get("staffMember"); ok {
			if staff, ok := cached.(*models.StaffMember); ok && staff != nil {
				if !staff.HasPermission(permission) {
					c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to perform this action"})
					c.Abort()
					return
				}
				c.Next()
				return
			}
		}

		// A HomeChef-native staff member is keyed by users.id, so the DB lookup
		// is only meaningful when the caller HAS a HomeChef user id.
		//
		// The tesserix.app admin console signs requests with its own OIDC
		// subject as X-User-Id (see lib/api/homechef-admin.ts). That subject is
		// not a HomeChef users.id and frequently is not even a UUID, and
		// applyBFFIdentity only sets the legacy "userID" key when the header
		// parses as one. Demanding a user id up front therefore rejected every
		// external admin with a 401 before any permission was considered —
		// which took out all of Payouts, Payout Setup, Staff and Fleet while
		// the unguarded admin endpoints kept working. Identity is established
		// by the signature, not by this lookup.
		var staff models.StaffMember
		userID, hasUserID := GetUserID(c)
		found := false
		if hasUserID {
			if err := staffDB().Where("user_id = ? AND is_active = ?", userID, true).
				First(&staff).Error; err == nil {
				found = true
			}
		}

		if !found {
			// Auto-provision for default super admins. The email is read from
			// the signed identity when no DB user row backs the caller: it is
			// bound into the X-Internal-Auth MAC (#461), so it cannot be
			// swapped without the signing key, and /admin has already enforced
			// bffAuth + RequirePool(internal) + RequireAdmin above. An empty or
			// unlisted email fails closed here.
			if models.IsSuperAdminEmail(staffActorEmail(c)) {
				staff = models.StaffMember{
					UserID:    userID, // uuid.Nil for a console-only admin
					StaffRole: models.StaffRoleSuperAdmin,
					IsActive:  true,
				}
			} else {
				c.JSON(http.StatusForbidden, gin.H{"error": "Staff access required"})
				c.Abort()
				return
			}
		}

		if !staff.HasPermission(permission) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to perform this action"})
			c.Abort()
			return
		}

		c.Set("staffMember", &staff)
		c.Next()
	}
}

// RequirePool ensures the caller's GIP identity pool is one of the allowed
// pools. This is defense-in-depth on top of role checks: the admin surface is
// gated on RoleAdmin, but a role is a DB column while the pool is bound into the
// BFF signature (X-Auth-Pool). Requiring PoolInternal on /admin means an
// admin-role user who authenticated through the customer/business pool cannot
// reach admin endpoints — the internal Keycloak realm is the only way in.
//
// Fails closed: an absent/empty pool is rejected (an unsigned or pre-GIP request
// would have no pool, and admin is too sensitive to allow that through).
func RequirePool(pools ...models.AuthPool) gin.HandlerFunc {
	return func(c *gin.Context) {
		pool, ok := GetAuthPool(c)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}
		for _, p := range pools {
			if pool == p {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}

// RequireAdmin is a shorthand for RequireRole(RoleAdmin)
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}

// RequireChef is a shorthand for RequireRole(RoleChef)
func RequireChef() gin.HandlerFunc {
	return RequireRole(models.RoleChef, models.RoleAdmin)
}

// RequireDelivery is a shorthand for RequireRole(RoleDelivery)
func RequireDelivery() gin.HandlerFunc {
	return RequireRole(models.RoleDelivery, models.RoleFleetManager, models.RoleAdmin)
}
