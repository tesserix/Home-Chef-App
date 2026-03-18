package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct{}

func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// RegisterRequest represents the registration payload
type RegisterRequest struct {
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Phone     string `json:"phone"`
}

// LoginRequest represents the login payload
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	User         models.UserResponse `json:"user"`
	AccessToken  string              `json:"accessToken"`
	RefreshToken string              `json:"refreshToken"`
}

// Register creates a new user account
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if email already exists
	var existingUser models.User
	if err := database.DB.Where("email = ?", strings.ToLower(req.Email)).First(&existingUser).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Create user
	user := models.User{
		Email:        strings.ToLower(req.Email),
		Password:     string(hashedPassword),
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		Role:         models.RoleCustomer,
		AuthProvider: models.ProviderEmail,
		IsActive:     true,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusCreated, AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Login authenticates a user and returns tokens
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user
	var user models.User
	if err := database.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if account is active
	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	database.DB.Save(&user)

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// RefreshTokenRequest represents the refresh token payload
type RefreshTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// RefreshToken generates new access token using refresh token
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find refresh token
	var refreshToken models.RefreshToken
	if err := database.DB.Where("token = ? AND revoked_at IS NULL", req.RefreshToken).
		First(&refreshToken).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	// Check if expired
	if refreshToken.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired"})
		return
	}

	// Get user
	var user models.User
	if err := database.DB.First(&user, "id = ?", refreshToken.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
		return
	}

	// Revoke old refresh token
	now := time.Now()
	refreshToken.RevokedAt = &now
	database.DB.Save(&refreshToken)

	// Generate new tokens
	accessToken, newRefreshToken, err := middleware.GenerateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"accessToken":  accessToken,
		"refreshToken": newRefreshToken,
	})
}

// Logout revokes the refresh token
func (h *AuthHandler) Logout(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Revoke refresh token
	now := time.Now()
	database.DB.Model(&models.RefreshToken{}).
		Where("token = ?", req.RefreshToken).
		Update("revoked_at", now)

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	user, exists := middleware.GetUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// UpdateProfileRequest represents the profile update payload
type UpdateProfileRequest struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Phone     string `json:"phone"`
	Avatar    string `json:"avatar"`
}

// UpdateProfile updates the current user's profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	user, exists := middleware.GetUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields if provided
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}

	if err := database.DB.Save(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, user.ToResponse())
}

// ChangePasswordRequest represents the password change payload
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	NewPassword     string `json:"newPassword" binding:"required,min=8"`
}

// ChangePassword changes the user's password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	user, exists := middleware.GetUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.CurrentPassword)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	user.Password = string(hashedPassword)
	if err := database.DB.Save(user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Revoke all refresh tokens for this user
	database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", user.ID).
		Update("revoked_at", time.Now())

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ForgotPasswordRequest represents the forgot password payload
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPassword initiates the password reset process
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find user (don't reveal if email exists)
	var user models.User
	if err := database.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		// Return success even if user doesn't exist (security)
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
		return
	}

	// TODO: Generate reset token and send email
	// For now, just return success
	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset link has been sent"})
}

// OAuthLoginRequest represents the OAuth login payload
type OAuthLoginRequest struct {
	Provider   string `json:"provider" binding:"required"`
	Token      string `json:"token" binding:"required"`
	Email      string `json:"email"`
	FirstName  string `json:"firstName"`
	LastName   string `json:"lastName"`
	ProviderID string `json:"providerId"`
	Avatar     string `json:"avatar"`
}

// OAuthLogin handles OAuth authentication
func (h *AuthHandler) OAuthLogin(c *gin.Context) {
	var req OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// TODO: Verify OAuth token with provider
	// For now, trust the client data (insecure - implement proper verification)

	provider := models.AuthProvider(req.Provider)
	if provider != models.ProviderGoogle && provider != models.ProviderFacebook && provider != models.ProviderApple {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth provider"})
		return
	}

	// Check if user exists
	var user models.User
	err := database.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error

	if err != nil {
		// Create new user
		user = models.User{
			Email:        strings.ToLower(req.Email),
			FirstName:    req.FirstName,
			LastName:     req.LastName,
			Avatar:       req.Avatar,
			Role:         models.RoleCustomer,
			AuthProvider: provider,
			ProviderID:   req.ProviderID,
			IsActive:     true,
			EmailVerified: true, // OAuth emails are verified
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else {
		// Update existing user's OAuth info if needed
		if user.ProviderID == "" {
			user.ProviderID = req.ProviderID
			user.AuthProvider = provider
			database.DB.Save(&user)
		}
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	database.DB.Save(&user)

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokens(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate tokens"})
		return
	}

	c.JSON(http.StatusOK, AuthResponse{
		User:         user.ToResponse(),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

// Helper to get user ID from string parameter
func ParseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}
