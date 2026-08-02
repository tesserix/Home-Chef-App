package handlers

// chef_verification_gate.go — a chef cannot open their kitchen (switch
// accepting_orders on) until an admin has reviewed and approved the
// application (chef_profiles.is_verified). Documents may be skipped during
// onboarding, so this is the wall between "set up your kitchen and menus"
// (allowed while unverified) and "take customer orders" (not allowed).
//
// Same shape as the payout gate: 409 with a machine-readable reasonCode so
// the vendor app can route the chef to the fix (upload documents vs wait for
// review) instead of a dead end.

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// verificationGateBlocks enforces the gate for a handler about to switch
// accepting_orders on. It writes the error response itself and reports
// whether the caller should stop.
func verificationGateBlocks(c *gin.Context, chef *models.ChefProfile, accepting bool) bool {
	if !accepting || chef.IsVerified {
		return false
	}

	if !services.ChefDocsComplete(database.DB, chef.ID) {
		c.JSON(http.StatusConflict, gin.H{
			"error":      "Upload your ID proof, address proof and FSSAI licence so we can verify your kitchen — you can open as soon as it's approved.",
			"reasonCode": "documents_required",
			"action":     "upload_documents",
		})
		return true
	}

	c.JSON(http.StatusConflict, gin.H{
		"error":      "Your kitchen is being reviewed. You can open as soon as it's approved — most reviews finish within 24-48 hours.",
		"reasonCode": "verification_pending",
		"action":     "await_review",
	})
	return true
}
