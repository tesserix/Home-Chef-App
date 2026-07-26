package handlers

// apple_signin.go — records the Sign in with Apple grant so account deletion
// can revoke it.
//
// App Review guideline 5.1.1(v): an app offering Sign in with Apple must call
// Apple's token-revocation endpoint when the user deletes their account.
// Revocation needs a refresh token, and that can only be obtained by exchanging
// the single-use authorization code Apple hands the client during sign-in — so
// the mobile apps post it here immediately after a successful Apple sign-in.
//
// This endpoint is deliberately forgiving: it never fails the caller. The apps
// call it as a fire-and-forget step after login, and a user must never be
// bounced out of a working sign-in because Apple's token endpoint was briefly
// unavailable.

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type AppleSignInHandler struct{}

func NewAppleSignInHandler() *AppleSignInHandler { return &AppleSignInHandler{} }

// LinkGrant exchanges the Apple authorization code for a refresh token and
// stores it against the authenticated user.
//
// POST /v1/auth/apple/link   { "authorizationCode": "..." }
//
// Always answers 200 with a status field. The client has nothing useful to do
// with a failure — the sign-in already succeeded — and surfacing an error would
// only tempt a client into retrying with a code Apple has already burned.
func (h *AppleSignInHandler) LinkGrant(c *gin.Context) {
	var req struct {
		AuthorizationCode string `json:"authorizationCode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := middleware.GetUserID(c)
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	err := services.LinkAppleGrant(c.Request.Context(), database.DB, &user, req.AuthorizationCode)
	switch {
	case err == nil:
		c.JSON(http.StatusOK, gin.H{"status": "linked"})
	case errors.Is(err, services.ErrAppleNotConfigured):
		// Nothing to store, and nothing wrong — this deployment has no Apple
		// service account wired up.
		c.JSON(http.StatusOK, gin.H{"status": "not_configured"})
	default:
		// Log the user id only; the authorization code is a credential and the
		// error may quote Apple's request echo.
		log.Printf("apple: could not link grant for user=%s: %v", user.ID, err)
		c.JSON(http.StatusOK, gin.H{"status": "deferred"})
	}
}
