package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/config"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// EnsureLoginEmailVerified requires the user's account email to be OTP-verified
// before onboarding can complete. No-op when the feature flag is off. Writes the
// response and returns false when the email is unverified. Used by onboarding
// flows (customer, delivery) where the email is the account email, not re-typed.
func EnsureLoginEmailVerified(c *gin.Context, userID uuid.UUID) bool {
	if config.AppConfig == nil || !config.AppConfig.EmailOTPEnabled {
		return true
	}
	var user models.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return false
	}
	email := services.NormalizeEmail(user.Email)
	if !services.IsValidEmailFormat(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Your account email is invalid. Please contact support.", "field": "email"})
		return false
	}
	if !services.IsEmailOTPVerified(c.Request.Context(), userID.String(), email) {
		c.JSON(http.StatusPreconditionRequired, gin.H{"error": "Please verify your email with the 6-digit code we sent before continuing.", "field": "email"})
		return false
	}
	return true
}

type EmailOTPHandler struct{}

func NewEmailOTPHandler() *EmailOTPHandler { return &EmailOTPHandler{} }

type emailOTPRequest struct {
	Email string `json:"email" binding:"required"`
}

type emailOTPVerify struct {
	Email string `json:"email" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

func maskEmailAddr(e string) string {
	at := -1
	for i, c := range e {
		if c == '@' {
			at = i
			break
		}
	}
	if at <= 1 {
		return e
	}
	return string(e[0]) + "***" + e[at:]
}

// RequestOTP emails a fresh 6-digit code to the address the user is onboarding with.
func (h *EmailOTPHandler) RequestOTP(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	var req emailOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter a valid email address", "field": "email"})
		return
	}
	email := services.NormalizeEmail(req.Email)
	if !services.IsValidEmailFormat(email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": services.ErrOTPInvalidEmail.Error(), "field": "email"})
		return
	}

	var existing models.User
	if err := database.DB.Where("email = ? AND id != ?", email, userID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This email is already registered with another account.", "field": "email"})
		return
	}

	var user models.User
	_ = database.DB.Where("id = ?", userID).First(&user).Error

	if err := services.RequestEmailOTP(c.Request.Context(), userID.String(), email, user.FirstName); err != nil {
		c.JSON(otpStatus(err), gin.H{"error": err.Error(), "field": "email"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"sent": true, "email": maskEmailAddr(email), "expiresInSeconds": 600})
}

// VerifyOTP validates the submitted code and marks the email verified for onboarding.
func (h *EmailOTPHandler) VerifyOTP(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	var req emailOTPVerify
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Enter the 6-digit code", "field": "code"})
		return
	}
	if err := services.VerifyEmailOTP(c.Request.Context(), userID.String(), req.Email, req.Code); err != nil {
		c.JSON(otpStatus(err), gin.H{"error": err.Error(), "field": "code"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"verified": true})
}

func otpStatus(err error) int {
	switch {
	case errors.Is(err, services.ErrOTPCooldown), errors.Is(err, services.ErrOTPSendLimit), errors.Is(err, services.ErrOTPAttemptLimit):
		return http.StatusTooManyRequests
	case errors.Is(err, services.ErrOTPUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadRequest
	}
}
