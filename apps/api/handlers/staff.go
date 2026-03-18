package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

type StaffHandler struct{}

func NewStaffHandler() *StaffHandler {
	return &StaffHandler{}
}

// GetMyStaffProfile returns the current user's staff profile
func (h *StaffHandler) GetMyStaffProfile(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var staff models.StaffMember
	if err := database.DB.Preload("User").Preload("InvitedBy").
		Where("user_id = ? AND is_active = ?", userID, true).
		First(&staff).Error; err != nil {

		// Check if this is a super admin email that hasn't been provisioned yet
		user, exists := middleware.GetUser(c)
		if exists && models.IsSuperAdminEmail(user.Email) {
			staff = provisionSuperAdmin(user)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"error": "Staff profile not found"})
			return
		}
	}

	// Update last active
	now := time.Now()
	database.DB.Model(&staff).Update("last_active_at", now)

	c.JSON(http.StatusOK, staff.ToResponse())
}

// ListStaff returns all staff members
func (h *StaffHandler) ListStaff(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	roleFilter := c.Query("role")
	search := c.Query("search")
	portal := c.Query("portal") // "admin" or "delivery"

	query := database.DB.Preload("User").Preload("InvitedBy")

	if roleFilter != "" {
		query = query.Where("staff_role = ?", roleFilter)
	}

	if portal == "admin" {
		query = query.Where("staff_role IN ?", []models.StaffRole{
			models.StaffRoleSuperAdmin, models.StaffRoleAdmin, models.StaffRoleSupport,
		})
	} else if portal == "delivery" {
		query = query.Where("staff_role IN ?", []models.StaffRole{
			models.StaffRoleSuperAdmin, models.StaffRoleFleetManager, models.StaffRoleDeliveryOps,
		})
	}

	if search != "" {
		query = query.Joins("JOIN users ON users.id = staff_members.user_id").
			Where("users.email ILIKE ? OR users.first_name ILIKE ? OR users.last_name ILIKE ?",
				"%"+search+"%", "%"+search+"%", "%"+search+"%")
	}

	var total int64
	query.Model(&models.StaffMember{}).Count(&total)

	var staff []models.StaffMember
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch staff"})
		return
	}

	responses := make([]models.StaffMemberResponse, len(staff))
	for i, s := range staff {
		responses[i] = s.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// GetStaffMember returns a single staff member by ID
func (h *StaffHandler) GetStaffMember(c *gin.Context) {
	staffID := c.Param("id")

	var staff models.StaffMember
	if err := database.DB.Preload("User").Preload("InvitedBy").
		Where("id = ?", staffID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found"})
		return
	}

	c.JSON(http.StatusOK, staff.ToResponse())
}

// UpdateStaffRole changes a staff member's role (super_admin only)
func (h *StaffHandler) UpdateStaffRole(c *gin.Context) {
	staffID := c.Param("id")
	currentUserID, _ := middleware.GetUserID(c)

	// Only super admins can change roles
	var currentStaff models.StaffMember
	if err := database.DB.Where("user_id = ?", currentUserID).First(&currentStaff).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a staff member"})
		return
	}
	if currentStaff.StaffRole != models.StaffRoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can change staff roles"})
		return
	}

	var req struct {
		StaffRole  models.StaffRole `json:"staffRole" binding:"required"`
		Department string           `json:"department"`
		Title      string           `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var staff models.StaffMember
	if err := database.DB.Where("id = ?", staffID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found"})
		return
	}

	// Prevent demoting other super admins unless you're changing yourself
	if staff.StaffRole == models.StaffRoleSuperAdmin && staff.UserID != currentUserID {
		// Check if the target is a default super admin email
		var targetUser models.User
		database.DB.First(&targetUser, "id = ?", staff.UserID)
		if models.IsSuperAdminEmail(targetUser.Email) {
			c.JSON(http.StatusForbidden, gin.H{"error": "Cannot change role of a default super admin"})
			return
		}
	}

	staff.StaffRole = req.StaffRole
	if req.Department != "" {
		staff.Department = req.Department
	}
	if req.Title != "" {
		staff.Title = req.Title
	}

	if err := database.DB.Save(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update staff role"})
		return
	}

	database.DB.Preload("User").Preload("InvitedBy").First(&staff, "id = ?", staff.ID)
	c.JSON(http.StatusOK, staff.ToResponse())
}

// DeactivateStaff deactivates a staff member
func (h *StaffHandler) DeactivateStaff(c *gin.Context) {
	staffID := c.Param("id")
	currentUserID, _ := middleware.GetUserID(c)

	var staff models.StaffMember
	if err := database.DB.Preload("User").Where("id = ?", staffID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found"})
		return
	}

	// Prevent deactivating default super admins
	if models.IsSuperAdminEmail(staff.User.Email) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot deactivate a default super admin"})
		return
	}

	// Prevent deactivating yourself
	if staff.UserID == currentUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot deactivate yourself"})
		return
	}

	staff.IsActive = false
	if err := database.DB.Save(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to deactivate staff member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff member deactivated"})
}

// ReactivateStaff reactivates a deactivated staff member
func (h *StaffHandler) ReactivateStaff(c *gin.Context) {
	staffID := c.Param("id")

	var staff models.StaffMember
	if err := database.DB.Where("id = ?", staffID).First(&staff).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Staff member not found"})
		return
	}

	staff.IsActive = true
	if err := database.DB.Save(&staff).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reactivate staff member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Staff member reactivated"})
}

// --- Invitations ---

// CreateInvitation sends a staff invitation
func (h *StaffHandler) CreateInvitation(c *gin.Context) {
	currentUserID, _ := middleware.GetUserID(c)

	// Verify inviter has invite permission
	var inviterStaff models.StaffMember
	if err := database.DB.Where("user_id = ? AND is_active = ?", currentUserID, true).
		First(&inviterStaff).Error; err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Not a staff member"})
		return
	}
	if !inviterStaff.HasPermission(models.SPInviteStaff) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You don't have permission to invite staff"})
		return
	}

	var req struct {
		Email      string           `json:"email" binding:"required,email"`
		StaffRole  models.StaffRole `json:"staffRole" binding:"required"`
		Department string           `json:"department"`
		Title      string           `json:"title"`
		Message    string           `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Only super admins can invite super admins
	if req.StaffRole == models.StaffRoleSuperAdmin && inviterStaff.StaffRole != models.StaffRoleSuperAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only super admins can invite other super admins"})
		return
	}

	// Check if already a staff member
	var existingStaff models.StaffMember
	if err := database.DB.Joins("JOIN users ON users.id = staff_members.user_id").
		Where("users.email = ? AND staff_members.deleted_at IS NULL", req.Email).
		First(&existingStaff).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This person is already a staff member"})
		return
	}

	// Check for existing pending invitation
	var existingInvite models.StaffInvitation
	if err := database.DB.Where("email = ? AND status = ?", req.Email, models.InvitationPending).
		First(&existingInvite).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "A pending invitation already exists for this email"})
		return
	}

	// Generate invitation token
	token, err := generateSecureToken(32)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate invitation token"})
		return
	}

	invitation := models.StaffInvitation{
		Email:       req.Email,
		StaffRole:   req.StaffRole,
		Department:  req.Department,
		Title:       req.Title,
		Token:       token,
		InvitedByID: currentUserID,
		Status:      models.InvitationPending,
		Message:     req.Message,
		ExpiresAt:   time.Now().AddDate(0, 0, 7), // 7-day expiry
	}

	if err := database.DB.Create(&invitation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create invitation"})
		return
	}

	// Load inviter for response
	database.DB.Preload("InvitedBy").First(&invitation, "id = ?", invitation.ID)

	// Determine base URL for invite link
	baseURL := c.GetHeader("Origin")
	if baseURL == "" {
		baseURL = "https://admin.fe3dr.com"
		// Use delivery portal URL for delivery roles
		if req.StaffRole == models.StaffRoleFleetManager || req.StaffRole == models.StaffRoleDeliveryOps {
			baseURL = "https://delivery.fe3dr.com"
		}
	}

	// TODO: Send email via SendGrid when integrated
	// For now, the invite URL is returned in the response

	c.JSON(http.StatusCreated, invitation.ToResponse(baseURL))
}

// ListInvitations returns all staff invitations
func (h *StaffHandler) ListInvitations(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit
	status := c.Query("status")

	query := database.DB.Preload("InvitedBy")

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// Expire old invitations
	database.DB.Model(&models.StaffInvitation{}).
		Where("status = ? AND expires_at < ?", models.InvitationPending, time.Now()).
		Update("status", models.InvitationExpired)

	var total int64
	query.Model(&models.StaffInvitation{}).Count(&total)

	var invitations []models.StaffInvitation
	if err := query.Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&invitations).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch invitations"})
		return
	}

	baseURL := c.GetHeader("Origin")
	if baseURL == "" {
		baseURL = "https://admin.fe3dr.com"
	}

	responses := make([]models.StaffInvitationResponse, len(invitations))
	for i, inv := range invitations {
		responses[i] = inv.ToResponse(baseURL)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": responses,
		"pagination": gin.H{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": (total + int64(limit) - 1) / int64(limit),
			"hasNext":    int64(offset+limit) < total,
			"hasPrev":    page > 1,
		},
	})
}

// RevokeInvitation revokes a pending invitation
func (h *StaffHandler) RevokeInvitation(c *gin.Context) {
	invitationID := c.Param("id")

	var invitation models.StaffInvitation
	if err := database.DB.Where("id = ? AND status = ?", invitationID, models.InvitationPending).
		First(&invitation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Pending invitation not found"})
		return
	}

	invitation.Status = models.InvitationRevoked
	if err := database.DB.Save(&invitation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke invitation"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked"})
}

// ResendInvitation resets expiry and returns new invite URL
func (h *StaffHandler) ResendInvitation(c *gin.Context) {
	invitationID := c.Param("id")

	var invitation models.StaffInvitation
	if err := database.DB.Preload("InvitedBy").
		Where("id = ? AND status IN ?", invitationID,
			[]models.InvitationStatus{models.InvitationPending, models.InvitationExpired}).
		First(&invitation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invitation not found"})
		return
	}

	// Generate new token and reset expiry
	token, _ := generateSecureToken(32)
	invitation.Token = token
	invitation.Status = models.InvitationPending
	invitation.ExpiresAt = time.Now().AddDate(0, 0, 7)

	if err := database.DB.Save(&invitation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resend invitation"})
		return
	}

	baseURL := c.GetHeader("Origin")
	if baseURL == "" {
		baseURL = "https://admin.fe3dr.com"
	}

	c.JSON(http.StatusOK, invitation.ToResponse(baseURL))
}

// AcceptInvitation accepts an invitation (public endpoint, called after login)
func (h *StaffHandler) AcceptInvitation(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invitation token required"})
		return
	}

	var invitation models.StaffInvitation
	if err := database.DB.Where("token = ? AND status = ?", token, models.InvitationPending).
		First(&invitation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid or expired invitation"})
		return
	}

	if time.Now().After(invitation.ExpiresAt) {
		invitation.Status = models.InvitationExpired
		database.DB.Save(&invitation)
		c.JSON(http.StatusGone, gin.H{"error": "Invitation has expired"})
		return
	}

	// Get the authenticated user
	userID, exists := middleware.GetUserID(c)
	if !exists {
		// Return invite details for unauthenticated users so the frontend can show info
		c.JSON(http.StatusOK, gin.H{
			"invitation": gin.H{
				"email":     invitation.Email,
				"staffRole": invitation.StaffRole,
				"title":     invitation.Title,
				"message":   invitation.Message,
				"expiresAt": invitation.ExpiresAt,
			},
			"requiresAuth": true,
		})
		return
	}

	// Verify the authenticated user's email matches
	user, _ := middleware.GetUser(c)
	if user.Email != invitation.Email {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "This invitation was sent to " + invitation.Email + ". Please sign in with that email.",
		})
		return
	}

	// Create staff member
	tx := database.DB.Begin()

	staff := models.StaffMember{
		UserID:      userID,
		StaffRole:   invitation.StaffRole,
		Department:  invitation.Department,
		Title:       invitation.Title,
		InvitedByID: &invitation.InvitedByID,
		IsActive:    true,
	}

	if err := tx.Create(&staff).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create staff profile"})
		return
	}

	// Update user role based on staff role
	newRole := models.RoleAdmin
	if invitation.StaffRole == models.StaffRoleFleetManager || invitation.StaffRole == models.StaffRoleDeliveryOps {
		newRole = models.RoleDelivery
	}
	tx.Model(&models.User{}).Where("id = ?", userID).Update("role", newRole)

	// Mark invitation as accepted
	now := time.Now()
	invitation.Status = models.InvitationAccepted
	invitation.AcceptedAt = &now
	invitation.AcceptedByID = &userID
	tx.Save(&invitation)

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Invitation accepted. Welcome to the team!",
		"staffRole": staff.StaffRole,
	})
}

// ValidateInvitation checks if an invitation token is valid (public endpoint)
func (h *StaffHandler) ValidateInvitation(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token required"})
		return
	}

	var invitation models.StaffInvitation
	if err := database.DB.Where("token = ?", token).First(&invitation).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid invitation token"})
		return
	}

	if invitation.Status != models.InvitationPending {
		c.JSON(http.StatusGone, gin.H{
			"error":  "Invitation is no longer valid",
			"status": invitation.Status,
		})
		return
	}

	if time.Now().After(invitation.ExpiresAt) {
		invitation.Status = models.InvitationExpired
		database.DB.Save(&invitation)
		c.JSON(http.StatusGone, gin.H{"error": "Invitation has expired"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":     true,
		"email":     invitation.Email,
		"staffRole": invitation.StaffRole,
		"title":     invitation.Title,
		"message":   invitation.Message,
		"expiresAt": invitation.ExpiresAt,
	})
}

// GetStaffRoles returns available staff roles and their permissions
func (h *StaffHandler) GetStaffRoles(c *gin.Context) {
	roles := []gin.H{
		{
			"role":        models.StaffRoleSuperAdmin,
			"label":       "Super Admin",
			"description": "Full access to all admin and delivery portal features",
			"portals":     []string{"admin", "delivery"},
			"permissions": models.DefaultStaffPermissions[models.StaffRoleSuperAdmin],
		},
		{
			"role":        models.StaffRoleAdmin,
			"label":       "Admin",
			"description": "Manage users, chefs, orders, and approvals",
			"portals":     []string{"admin"},
			"permissions": models.DefaultStaffPermissions[models.StaffRoleAdmin],
		},
		{
			"role":        models.StaffRoleFleetManager,
			"label":       "Fleet Manager",
			"description": "Manage delivery partners, fleet operations, and delivery analytics",
			"portals":     []string{"delivery"},
			"permissions": models.DefaultStaffPermissions[models.StaffRoleFleetManager],
		},
		{
			"role":        models.StaffRoleDeliveryOps,
			"label":       "Delivery Operations",
			"description": "View and assign deliveries, monitor delivery partners",
			"portals":     []string{"delivery"},
			"permissions": models.DefaultStaffPermissions[models.StaffRoleDeliveryOps],
		},
		{
			"role":        models.StaffRoleSupport,
			"label":       "Support",
			"description": "View-only access for customer support across both portals",
			"portals":     []string{"admin", "delivery"},
			"permissions": models.DefaultStaffPermissions[models.StaffRoleSupport],
		},
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// --- Helpers ---

func provisionSuperAdmin(user *models.User) models.StaffMember {
	staff := models.StaffMember{
		UserID:     user.ID,
		StaffRole:  models.StaffRoleSuperAdmin,
		Department: "Engineering",
		Title:      "Super Admin",
		IsActive:   true,
	}

	if err := database.DB.Create(&staff).Error; err != nil {
		// If already exists (race condition), fetch it
		database.DB.Where("user_id = ?", user.ID).First(&staff)
	}

	// Ensure user role is admin
	database.DB.Model(user).Update("role", models.RoleAdmin)

	staff.User = *user
	return staff
}

func generateSecureToken(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
