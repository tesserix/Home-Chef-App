package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
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

	// Enforce the currently configured platform password policy.
	if err := services.ValidatePasswordAgainstPolicy(req.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Send welcome email + verification email asynchronously
	go func() {
		emailSvc := services.GetEmailService()
		emailSvc.SendWelcomeEmail(user.Email, user.FirstName)

		// Generate email verification token (24h expiry)
		verifyToken := make([]byte, 32)
		if _, err := rand.Read(verifyToken); err == nil {
			tokenHex := hex.EncodeToString(verifyToken)
			evToken := models.EmailVerificationToken{
				UserID:    user.ID,
				Token:     tokenHex,
				ExpiresAt: time.Now().Add(24 * time.Hour),
			}
			if database.DB.Create(&evToken).Error == nil {
				baseURL := "https://fe3dr.com"
				verifyURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, tokenHex)
				emailSvc.SendEmailVerification(user.Email, user.FirstName, verifyURL)
			}
		}
	}()

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokensWithContext(&user, c)
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

	// 2FA gating — two branches before we issue real tokens:
	//
	//  (a) The user has TOTP enrolled → they must present a 6-digit code on
	//      /auth/2fa/verify. We return a short-lived challenge token that
	//      proves they cleared the password step.
	//
	//  (b) Platform policy requires 2FA for admins but this admin hasn't
	//      enrolled yet → force them through enrollment before login
	//      completes. Same challenge-token pattern, just a different claim.
	policy := services.GetSecurityPolicy()
	if user.TOTPEnabled {
		challenge, cerr := middleware.GenerateTwoFactorChallenge(&user, middleware.ChallengeVerify)
		if cerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start 2FA challenge"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"twoFactorRequired": true,
			"challengeToken":    challenge,
		})
		return
	}
	if policy.TwoFactorRequiredForAdmins && user.Role == models.RoleAdmin && !policy.IsTwoFactorExempt(user.Email) {
		enroll, cerr := middleware.GenerateTwoFactorChallenge(&user, middleware.ChallengeEnroll)
		if cerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start 2FA enrollment"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"twoFactorEnrollmentRequired": true,
			"enrollmentToken":             enroll,
		})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	database.DB.Save(&user)

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokensWithContext(&user, c)
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
	accessToken, newRefreshToken, err := middleware.GenerateTokensWithContext(&user, c)
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

	// Enforce platform password policy on the new password.
	if err := services.ValidatePasswordAgainstPolicy(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Always return success to avoid email enumeration
	successMsg := gin.H{"message": "If the email exists, a reset link has been sent"}

	// Find user
	var user models.User
	if err := database.DB.Where("email = ?", strings.ToLower(req.Email)).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, successMsg)
		return
	}

	// Invalidate any existing unused reset tokens for this user
	database.DB.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL", user.ID).
		Update("used_at", time.Now())

	// Generate a cryptographically secure token (32 bytes = 64 hex chars)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		log.Printf("ForgotPassword: failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}
	token := hex.EncodeToString(tokenBytes)

	// Store the reset token with 1-hour expiry
	resetToken := models.PasswordResetToken{
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if err := database.DB.Create(&resetToken).Error; err != nil {
		log.Printf("ForgotPassword: failed to save reset token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// Send the reset email asynchronously (don't block response)
	go func() {
		if err := services.GetEmailService().SendPasswordResetEmail(user.Email, token); err != nil {
			log.Printf("ForgotPassword: failed to send email to %s: %v", user.Email, err)
		}
	}()

	c.JSON(http.StatusOK, successMsg)
}

// ResetPasswordRequest represents the password reset payload
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

// ResetPassword validates a reset token and sets a new password
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Find the reset token
	var resetToken models.PasswordResetToken
	if err := database.DB.Where("token = ? AND used_at IS NULL", req.Token).First(&resetToken).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
		return
	}

	// Check expiry
	if resetToken.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Reset token has expired"})
		return
	}

	// Enforce platform password policy on the reset password.
	if err := services.ValidatePasswordAgainstPolicy(req.NewPassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash the new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process password"})
		return
	}

	// Update the user's password
	if err := database.DB.Model(&models.User{}).Where("id = ?", resetToken.UserID).
		Update("password", string(hashedPassword)).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Mark token as used
	now := time.Now()
	resetToken.UsedAt = &now
	database.DB.Save(&resetToken)

	// Revoke all refresh tokens for this user
	database.DB.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", resetToken.UserID).
		Update("revoked_at", now)

	c.JSON(http.StatusOK, gin.H{"message": "Password has been reset successfully"})
}

// VerifyEmail validates an email verification token and marks the user as verified
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification token is required"})
		return
	}

	var evToken models.EmailVerificationToken
	if err := database.DB.Where("token = ?", token).First(&evToken).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid verification token"})
		return
	}

	if !evToken.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification token has expired or already been used"})
		return
	}

	// Mark the user as email-verified
	if err := database.DB.Model(&models.User{}).Where("id = ?", evToken.UserID).
		Update("email_verified", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify email"})
		return
	}

	// Mark token as used
	now := time.Now()
	evToken.UsedAt = &now
	database.DB.Save(&evToken)

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// ResendVerification resends the email verification link
func (h *AuthHandler) ResendVerification(c *gin.Context) {
	userID, exists := c.Get("userId")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	if user.EmailVerified {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is already verified"})
		return
	}

	// Invalidate old tokens
	database.DB.Model(&models.EmailVerificationToken{}).
		Where("user_id = ? AND used_at IS NULL", user.ID).
		Update("used_at", time.Now())

	// Generate new token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	tokenHex := hex.EncodeToString(tokenBytes)
	evToken := models.EmailVerificationToken{
		UserID:    user.ID,
		Token:     tokenHex,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := database.DB.Create(&evToken).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create verification token"})
		return
	}

	baseURL := "https://fe3dr.com"
	verifyURL := fmt.Sprintf("%s/verify-email?token=%s", baseURL, tokenHex)
	go services.GetEmailService().SendEmailVerification(user.Email, user.FirstName, verifyURL)

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
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

// OAuthLogin handles OAuth authentication with server-side token verification
func (h *AuthHandler) OAuthLogin(c *gin.Context) {
	var req OAuthLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	provider := models.AuthProvider(req.Provider)
	if provider != models.ProviderGoogle && provider != models.ProviderFacebook && provider != models.ProviderApple {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid OAuth provider"})
		return
	}

	// Verify the OAuth token with the respective provider
	verifiedEmail, verifiedName, verifiedProviderID, verifiedAvatar, err := verifyOAuthToken(provider, req.Token)
	if err != nil {
		log.Printf("OAuth verification failed for %s: %v", provider, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OAuth token verification failed"})
		return
	}

	// Use verified data, falling back to client data only for non-critical fields
	email := verifiedEmail
	if email == "" {
		email = req.Email
	}
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	providerID := verifiedProviderID
	if providerID == "" {
		providerID = req.ProviderID
	}

	firstName := req.FirstName
	lastName := req.LastName
	if verifiedName != "" && firstName == "" {
		parts := strings.SplitN(verifiedName, " ", 2)
		firstName = parts[0]
		if len(parts) > 1 {
			lastName = parts[1]
		}
	}

	avatar := req.Avatar
	if verifiedAvatar != "" {
		avatar = verifiedAvatar
	}

	// Check if user exists
	var user models.User
	dbErr := database.DB.Where("email = ?", strings.ToLower(email)).First(&user).Error

	if dbErr != nil {
		// Create new user
		user = models.User{
			Email:         strings.ToLower(email),
			FirstName:     firstName,
			LastName:      lastName,
			Avatar:        avatar,
			Role:          models.RoleCustomer,
			AuthProvider:  provider,
			ProviderID:    providerID,
			IsActive:      true,
			EmailVerified: true, // OAuth emails are verified by the provider
		}

		if err := database.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
			return
		}
	} else {
		// Refuse to silently rebind an existing account that registered with
		// a different auth method to a new OAuth provider. Allowing a silent
		// rebind made OAuthLogin a permanent account-takeover primitive: any
		// attacker who controlled the matching Google/Facebook/Apple email
		// could overwrite an email-password admin's AuthProvider/ProviderID
		// and authenticate going forward without ever knowing the password.
		// Linking a different provider must go through an authenticated
		// "link account" flow that re-verifies the original credential.
		if user.AuthProvider != provider {
			log.Printf("OAuthLogin: refused rebind for %s — existing provider=%s, attempted=%s",
				user.Email, user.AuthProvider, provider)
			c.JSON(http.StatusConflict, gin.H{
				"error": "This email is registered with a different sign-in method. Sign in with your original method, then link this provider from your profile.",
			})
			return
		}
		// Same provider — fill in providerID if it wasn't recorded yet (legacy
		// rows). Don't touch AuthProvider; it was already validated above.
		if user.ProviderID == "" {
			user.ProviderID = providerID
			database.DB.Save(&user)
		}
	}

	if !user.IsActive {
		c.JSON(http.StatusForbidden, gin.H{"error": "Account is suspended"})
		return
	}

	// 2FA gate — same as the email/password Login path. The OAuth provider's
	// authentication is not a substitute for the platform's TOTP requirement.
	policy := services.GetSecurityPolicy()
	if user.TOTPEnabled {
		challenge, cerr := middleware.GenerateTwoFactorChallenge(&user, middleware.ChallengeVerify)
		if cerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start 2FA challenge"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"twoFactorRequired": true,
			"challengeToken":    challenge,
		})
		return
	}
	if policy.TwoFactorRequiredForAdmins && user.Role == models.RoleAdmin && !policy.IsTwoFactorExempt(user.Email) {
		enroll, cerr := middleware.GenerateTwoFactorChallenge(&user, middleware.ChallengeEnroll)
		if cerr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start 2FA enrollment"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"twoFactorEnrollmentRequired": true,
			"enrollmentToken":             enroll,
		})
		return
	}

	// Update last login
	now := time.Now()
	user.LastLoginAt = &now
	database.DB.Save(&user)

	// Generate tokens
	accessToken, refreshToken, err := middleware.GenerateTokensWithContext(&user, c)
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

// verifyOAuthToken verifies the token with the respective OAuth provider.
// Returns (email, name, providerID, avatar, error).
func verifyOAuthToken(provider models.AuthProvider, token string) (string, string, string, string, error) {
	switch provider {
	case models.ProviderGoogle:
		return verifyGoogleToken(token)
	case models.ProviderFacebook:
		return verifyFacebookToken(token)
	case models.ProviderApple:
		return verifyAppleToken(token)
	default:
		return "", "", "", "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

// verifyGoogleToken verifies a Google ID token using the tokeninfo endpoint
func verifyGoogleToken(idToken string) (string, string, string, string, error) {
	resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?id_token=" + idToken)
	if err != nil {
		return "", "", "", "", fmt.Errorf("google: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", fmt.Errorf("google: token invalid (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", fmt.Errorf("google: failed to read response: %w", err)
	}

	var result struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Sub     string `json:"sub"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", "", "", fmt.Errorf("google: failed to parse response: %w", err)
	}

	if result.Email == "" {
		return "", "", "", "", fmt.Errorf("google: no email in token")
	}

	return result.Email, result.Name, result.Sub, result.Picture, nil
}

// verifyFacebookToken verifies a Facebook access token using the Graph API
func verifyFacebookToken(accessToken string) (string, string, string, string, error) {
	resp, err := http.Get("https://graph.facebook.com/me?fields=id,name,email,picture.type(large)&access_token=" + accessToken)
	if err != nil {
		return "", "", "", "", fmt.Errorf("facebook: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", "", fmt.Errorf("facebook: token invalid (status %d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", "", "", fmt.Errorf("facebook: failed to read response: %w", err)
	}

	var result struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture struct {
			Data struct {
				URL string `json:"url"`
			} `json:"data"`
		} `json:"picture"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", "", "", fmt.Errorf("facebook: failed to parse response: %w", err)
	}

	if result.ID == "" {
		return "", "", "", "", fmt.Errorf("facebook: no user ID in response")
	}

	return result.Email, result.Name, result.ID, result.Picture.Data.URL, nil
}

// verifyAppleToken performs basic JWT validation for Apple Sign In tokens.
// A full implementation would fetch Apple's public keys from
// https://appleid.apple.com/auth/keys and verify the RS256 signature.
// Here we decode the payload and validate the basic claims.
func verifyAppleToken(idToken string) (string, string, string, string, error) {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return "", "", "", "", fmt.Errorf("apple: invalid JWT format")
	}

	// Decode the payload (part[1])
	decoded, err := base64URLDecode(parts[1])
	if err != nil {
		return "", "", "", "", fmt.Errorf("apple: failed to decode payload: %w", err)
	}

	var claims struct {
		Iss   string `json:"iss"`
		Sub   string `json:"sub"`
		Email string `json:"email"`
		Exp   int64  `json:"exp"`
	}
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return "", "", "", "", fmt.Errorf("apple: failed to parse claims: %w", err)
	}

	// Validate issuer
	if claims.Iss != "https://appleid.apple.com" {
		return "", "", "", "", fmt.Errorf("apple: invalid issuer: %s", claims.Iss)
	}

	// Validate expiry
	if time.Now().Unix() > claims.Exp {
		return "", "", "", "", fmt.Errorf("apple: token expired")
	}

	if claims.Sub == "" {
		return "", "", "", "", fmt.Errorf("apple: no subject in token")
	}

	return claims.Email, "", claims.Sub, "", nil
}

// base64URLDecode decodes a base64url-encoded string (with or without padding)
func base64URLDecode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// DeviceTokenRequest represents the FCM device token registration payload
type DeviceTokenRequest struct {
	Token string `json:"token" binding:"required"`
}

// UpdateDeviceToken registers or updates the FCM device token for the current user
func (h *AuthHandler) UpdateDeviceToken(c *gin.Context) {
	user, exists := middleware.GetUser(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user.FCMToken = req.Token
	if err := database.DB.Model(user).Update("fcm_token", req.Token).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update device token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Device token updated"})
}

// Helper to get user ID from string parameter
func ParseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}
