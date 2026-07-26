package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/services"
)

// PasswordResetHandler owns self-service password reset.
//
// Both endpoints are unauthenticated by necessity — someone who cannot sign in
// is exactly who needs them — so both are rate limited in the service layer and
// both answer identically whether or not the account exists.
type PasswordResetHandler struct{}

func NewPasswordResetHandler() *PasswordResetHandler { return &PasswordResetHandler{} }

// genericResetAccepted is returned for EVERY outcome of a reset request:
// account found, account missing, wrong app, rate limited. Any variation would
// let a stranger probe which addresses are registered and in which app.
const genericResetAccepted = "If that email has an account, we've sent a link to reset the password. It expires in 15 minutes."

// RequestPasswordReset emails a reset link.
//
// POST /auth/password-reset/request  {"email":"…","app":"customer|vendor|delivery"}
//
// `app` selects the Identity Platform tenant. It matters: accounts are
// tenant-scoped, so a vendor's address genuinely does not exist in the customer
// tenant, and a request from the wrong app can never produce an email.
func (h *PasswordResetHandler) RequestPasswordReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required,email"`
		App   string `json:"app"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		// Even a malformed address gets the generic answer — a validation error
		// distinguishable from success is still an oracle.
		c.JSON(http.StatusOK, gin.H{"message": genericResetAccepted})
		return
	}

	err := services.RequestPasswordReset(c.Request.Context(), req.Email, req.App, c.ClientIP())
	switch {
	case err == nil, errors.Is(err, services.ErrPasswordResetRateLimited):
		// Rate limiting is deliberately indistinguishable from success. Saying
		// "too many requests for this address" would confirm the address exists.
		c.JSON(http.StatusOK, gin.H{"message": genericResetAccepted})
	default:
		// A real infrastructure failure (no Redis, mailer down, GIP unreachable)
		// is worth telling the user about — silently swallowing it would leave
		// them waiting for an email that is never coming.
		//
		// LOG IT. Returning 503 with nothing in the logs makes this class of
		// failure invisible: the user sees a generic apology, the operator sees
		// a bare status code, and nobody can tell whether it was GIP, Redis or
		// the mailer. The error text is safe — the service layer deliberately
		// keeps the address and the reset link out of it.
		log.Printf("password-reset: request failed: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "We couldn't send the reset email just now. Please try again in a few minutes.",
		})
	}
}

// ConsumePasswordResetToken redeems our single-use token and forwards the user
// to Identity Platform's reset page.
//
// GET /auth/password-reset/consume?t=…
//
// The indirection is what gives us a 15-minute, single-use link on our own
// domain: Firebase's own oobCode lives about an hour, cannot be shortened per
// request, and is replayable until it expires.
func (h *PasswordResetHandler) ConsumePasswordResetToken(c *gin.Context) {
	link, err := services.RedeemPasswordResetToken(c.Request.Context(), c.Query("t"))
	if err != nil {
		// One message for expired, unknown and already-used alike.
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(services.PasswordResetExpiredHTML()))
		return
	}
	// 303 so a refresh of the redirected page cannot re-issue the (now consumed)
	// redirect, and browsers switch to GET.
	c.Redirect(http.StatusSeeOther, link)
}
