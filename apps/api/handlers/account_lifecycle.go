package handlers

// account_lifecycle.go — the HTTP surface for deactivate, delete, restore and
// start-fresh, shared by all three roles.
//
// Apple 5.1.1(v) requires deletion to be initiated inside the app, and Google
// Play requires the same plus a web route; these endpoints are what the mobile
// apps call. The deletion half deliberately mirrors the existing DPDP contract
// in dpdp_common.go (confirmEmail, idempotent retry) so the two paths cannot
// drift apart.

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type AccountLifecycleHandler struct{}

func NewAccountLifecycleHandler() *AccountLifecycleHandler { return &AccountLifecycleHandler{} }

// loadLifecycleUser resolves the authenticated user unscoped, so an already
// soft-deleted account still resolves and retried calls stay idempotent.
func loadLifecycleUser(c *gin.Context) (models.User, bool) {
	userID, _ := middleware.GetUserID(c)
	var user models.User
	if err := database.DB.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return models.User{}, false
	}
	return user, true
}

// Deactivate pauses the account reversibly.
//
// POST /me/deactivate   { "reason": "..." }  (reason optional)
func (h *AccountLifecycleHandler) Deactivate(c *gin.Context) {
	user, ok := loadLifecycleUser(c)
	if !ok {
		return
	}
	if user.DeletedAt.Valid {
		c.JSON(http.StatusConflict, gin.H{
			"error": "This account is already scheduled for deletion.",
		})
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req) // body is optional

	if err := services.Deactivate(database.DB, &user, req.Reason); err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not deactivate your account"})
		return
	}

	services.LogAudit(c, "account.deactivate", "user", user.ID.String(), nil, gin.H{
		"deactivatedAt": user.DeactivatedAt,
	})
	c.JSON(http.StatusOK, gin.H{
		"status":        "deactivated",
		"deactivatedAt": user.DeactivatedAt,
		"notice": "Your account is paused. Sign in again any time to reactivate it — " +
			"nothing has been deleted.",
	})
}

// Reactivate lifts a pause.
//
// POST /me/reactivate
func (h *AccountLifecycleHandler) Reactivate(c *gin.Context) {
	user, ok := loadLifecycleUser(c)
	if !ok {
		return
	}
	if user.DeletedAt.Valid {
		c.JSON(http.StatusConflict, gin.H{
			"error": "This account is scheduled for deletion and cannot be reactivated here.",
		})
		return
	}

	if err := services.Reactivate(database.DB, &user); err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not reactivate your account"})
		return
	}

	services.LogAudit(c, "account.reactivate", "user", user.ID.String(), nil, nil)
	c.JSON(http.StatusOK, gin.H{"status": "active"})
}

// DeletionEligibility previews whether deletion would succeed, so the app can
// warn before the confirmation screen rather than after it.
//
// GET /me/deletion-eligibility
func (h *AccountLifecycleHandler) DeletionEligibility(c *gin.Context) {
	user, ok := loadLifecycleUser(c)
	if !ok {
		return
	}
	blockers := services.DeletionBlockers(database.DB, user)
	c.JSON(http.StatusOK, gin.H{
		"deletable":     len(blockers) == 0,
		"blockers":      blockers,
		"retentionDays": int(services.RestoreWindow.Hours() / 24),
	})
}

// DeleteAccount soft-deletes the account and starts the restore window.
//
// POST /me/account   (DELETE-style)   { "confirmEmail": "...", "reason": "..." }
func (h *AccountLifecycleHandler) DeleteAccount(c *gin.Context) {
	user, ok := loadLifecycleUser(c)
	if !ok {
		return
	}

	var req struct {
		ConfirmEmail string `json:"confirmEmail" binding:"required"`
		Reason       string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// Case-insensitive and trimmed, matching dpdp_common.go: the mobile confirm
	// gate normalises the same way, and a stricter server check would let the
	// app enable "Delete" and then reject the request.
	if !strings.EqualFold(strings.TrimSpace(req.ConfirmEmail), strings.TrimSpace(user.Email)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "confirmEmail must match your account email"})
		return
	}

	// Idempotent: a retry over a flaky mobile network returns the same payload
	// rather than a confusing error.
	if user.DeletedAt.Valid {
		c.JSON(http.StatusOK, gin.H{
			"status":     "already_deleted",
			"deletedAt":  user.DeletedAt.Time,
			"purgeAfter": user.PurgeAfter,
		})
		return
	}

	blockers, err := services.RequestDeletion(database.DB, &user, req.Reason)
	if err == services.ErrBlocked {
		c.JSON(http.StatusConflict, gin.H{
			"error":    "Please finish these before deleting your account.",
			"blockers": blockers,
		})
		return
	}
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed; please contact support"})
		return
	}

	// Kill the credential AFTER the erasure is committed. Best-effort: an
	// Identity Platform hiccup must not roll back a deletion the user asked
	// for, and the purge sweeper retries this before erasing.
	if user.GIPUid != "" {
		if err := services.DeleteGIPAccount(c.Request.Context(), user.GIPTenantID, user.GIPUid); err != nil {
			log.Printf("account: GIP credential teardown failed for user=%s: %v", user.ID, err)
		}
	}

	// User id only — never the email. This is the erasure path.
	services.LogAudit(c, "account.delete", "user", user.ID.String(), nil, gin.H{
		"deletedAt":  user.DeletedAt.Time,
		"purgeAfter": user.PurgeAfter,
	})

	c.JSON(http.StatusOK, gin.H{
		"status":     "deleted",
		"deletedAt":  user.DeletedAt.Time,
		"purgeAfter": user.PurgeAfter,
		"notice": "Your account has been deleted. If you sign up again with this email " +
			"within 180 days you can restore your history — after that it is erased permanently.",
	})
}

// Restore brings a deleted account back, authorised by the restore token the
// re-signup handshake issued. Data returns; standing does not.
//
// POST /account/restore   { "restoreToken": "..." }
func (h *AccountLifecycleHandler) Restore(c *gin.Context) {
	var req struct {
		RestoreToken string `json:"restoreToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, gipUid, err := services.ParseRestoreToken(req.RestoreToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "This restore link has expired. Please sign in again to retry.",
		})
		return
	}

	var ghost models.User
	if err := database.DB.Unscoped().First(&ghost, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
		return
	}

	restored, err := services.Restore(database.DB, userID, gipUid, ghost.GIPTenantID, ghost.GIPProvider)
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusConflict, gin.H{
			"error": "This account can no longer be restored.",
		})
		return
	}

	services.LogAudit(c, "account.restore", "user", userID.String(), nil, gin.H{
		"restoredAt": restored.RestoredAt,
	})

	reapproval := restored.Role == models.RoleChef || restored.Role == models.RoleDelivery
	c.JSON(http.StatusOK, gin.H{
		"status":             "restored",
		"reapprovalRequired": reapproval,
		"notice": reapprovalNotice(reapproval),
	})
}

func reapprovalNotice(reapproval bool) string {
	if reapproval {
		return "Welcome back. Your history has been restored, but your account needs to be " +
			"approved again — please re-upload your identity documents to get started."
	}
	return "Welcome back. Your account and order history have been restored."
}

// StartFresh discards the deleted account permanently so the caller can sign up
// clean on the same email. This is also what releases the unique email slot the
// ghost row holds.
//
// POST /account/start-fresh   { "restoreToken": "..." }
func (h *AccountLifecycleHandler) StartFresh(c *gin.Context) {
	var req struct {
		RestoreToken string `json:"restoreToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _, err := services.ParseRestoreToken(req.RestoreToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "This link has expired. Please sign in again to retry.",
		})
		return
	}

	var ghost models.User
	if err := database.DB.Unscoped().First(&ghost, "id = ?", userID).Error; err != nil {
		// Already gone — the caller can just sign up.
		c.JSON(http.StatusOK, gin.H{"status": "purged"})
		return
	}
	if !ghost.DeletedAt.Valid {
		c.JSON(http.StatusConflict, gin.H{"error": "This account is active and cannot be discarded."})
		return
	}

	if err := services.ArchiveAccountFinancials(c.Request.Context(), ghost); err != nil {
		log.Printf("account: archive failed before start-fresh for user=%s: %v", ghost.ID, err)
	}
	if err := services.PurgeUser(database.DB, ghost.ID, ghost.Role); err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not complete the request"})
		return
	}

	services.LogAudit(c, "account.start_fresh", "user", ghost.ID.String(), nil, nil)
	c.JSON(http.StatusOK, gin.H{
		"status": "purged",
		"notice": "Your previous account has been permanently erased. You can now sign up fresh.",
	})
}
