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
	PermBrowseChefs      Permission = "browse_chefs"
	PermViewMenu         Permission = "view_menu"
	PermPlaceOrder       Permission = "place_order"
	PermViewOwnOrders    Permission = "view_own_orders"
	PermWriteReview      Permission = "write_review"
	PermManageCart       Permission = "manage_cart"
	PermViewSocialFeed   Permission = "view_social_feed"
	PermLikeComment      Permission = "like_comment"
	PermCreateCatering   Permission = "create_catering_request"

	// Chef permissions
	PermManageMenu          Permission = "manage_menu"
	PermManageChefOrders    Permission = "manage_chef_orders"
	PermViewChefEarnings    Permission = "view_chef_earnings"
	PermManageChefProfile   Permission = "manage_chef_profile"
	PermCreatePost          Permission = "create_post"
	PermRespondToReviews    Permission = "respond_to_reviews"
	PermSubmitCateringQuote Permission = "submit_catering_quote"

	// Delivery permissions
	PermViewDeliveries     Permission = "view_deliveries"
	PermAcceptDelivery     Permission = "accept_delivery"
	PermUpdateDelivery     Permission = "update_delivery"
	PermViewDeliveryEarnings Permission = "view_delivery_earnings"

	// Admin permissions
	PermViewAllUsers       Permission = "view_all_users"
	PermManageUsers        Permission = "manage_users"
	PermVerifyChefs        Permission = "verify_chefs"
	PermViewAllOrders      Permission = "view_all_orders"
	PermViewAnalytics      Permission = "view_analytics"
	PermManageSettings     Permission = "manage_settings"
	PermModerateContent    Permission = "moderate_content"
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
		PermViewSocialFeed,
		PermLikeComment,
		PermCreateCatering,
	},
	models.RoleChef: {
		// Inherit customer permissions
		PermBrowseChefs,
		PermViewMenu,
		PermViewSocialFeed,
		// Chef-specific permissions
		PermManageMenu,
		PermManageChefOrders,
		PermViewChefEarnings,
		PermManageChefProfile,
		PermCreatePost,
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
		PermViewSocialFeed,
		PermLikeComment,
		PermCreateCatering,
		PermManageMenu,
		PermManageChefOrders,
		PermViewChefEarnings,
		PermManageChefProfile,
		PermCreatePost,
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

// RequireStaffPermission checks that the authenticated user has a StaffMember
// record with the given permission. This enforces granular staff RBAC beyond
// basic role checks. The StaffMember is loaded from the DB and cached on the
// gin context as "staffMember" for downstream handlers.
func RequireStaffPermission(permission models.StaffPermission) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := GetUserID(c)
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

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

		// Load from DB
		var staff models.StaffMember
		if err := staffDB().Where("user_id = ? AND is_active = ?", userID, true).First(&staff).Error; err != nil {
			// Auto-provision for default super admins
			user, userExists := GetUser(c)
			if userExists && models.IsSuperAdminEmail(user.Email) {
				staff = models.StaffMember{
					UserID:    userID,
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
