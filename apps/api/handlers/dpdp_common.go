package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// dpdp_common.go — the shared scaffolding behind the DPDP Act 2023 data-subject
// endpoints (Right to Access → export, Right to Erasure → delete). The chef,
// customer and driver handlers all reuse these so the export envelope, the
// confirm-email erasure contract, the soft-delete + 30-day retention semantics,
// and the audit trail stay identical across roles. Only the role-specific set of
// related tables (export) and the cascade (delete) differ per handler.

// dpdpRetentionWindow is how long a soft-deleted account is retained before the
// sweeper hard-purges it (legal-hold window per the privacy policy).
const dpdpRetentionWindow = 30 * 24 * time.Hour

// newExportEnvelope builds the common top of every export document: the
// timestamp, the requesting subject, the DPDP notice, and the sanitized user
// row. Role handlers append their own tables to the returned map.
func newExportEnvelope(user models.User) map[string]any {
	return map[string]any{
		"exportedAt": time.Now().UTC().Format(time.RFC3339),
		"requestedBy": map[string]any{
			"userId": user.ID.String(),
			"email":  user.Email,
		},
		"notice": "This export contains all personal data Home Chef holds on your account per DPDP Act 2023 §11.",
		"user":   sanitizeUserForExport(user),
	}
}

// writeExportJSON marshals the dump and returns it as a downloadable attachment.
func writeExportJSON(c *gin.Context, dump map[string]any) {
	filename := fmt.Sprintf("homechef-data-export-%s.json", time.Now().UTC().Format("2006-01-02"))
	body, err := json.MarshalIndent(dump, "", "  ")
	if err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build export"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	c.Data(http.StatusOK, "application/json", body)
}

// loadExportUser loads the authenticated user for an export, writing a 404 and
// returning ok=false when absent.
func loadExportUser(c *gin.Context) (models.User, bool) {
	userID, _ := middleware.GetUserID(c)
	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return models.User{}, false
	}
	return user, true
}

// beginSelfDelete implements the shared front half of the erasure flow: load the
// user (unscoped, so a retried delete on an already-soft-deleted account still
// resolves), require an exact confirmEmail match, and short-circuit an
// already-deleted account with an idempotent 200. It returns proceed=false when
// it has already written the response (error, validation failure, or the
// idempotent already-deleted reply) — the caller must return in that case.
func beginSelfDelete(c *gin.Context) (user models.User, proceed bool) {
	userID, _ := middleware.GetUserID(c)

	if err := database.DB.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return user, false
	}

	var req struct {
		ConfirmEmail string `json:"confirmEmail" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return user, false
	}
	if req.ConfirmEmail != user.Email {
		c.JSON(http.StatusBadRequest, gin.H{"error": "confirmEmail must exactly match your account email"})
		return user, false
	}

	if user.DeletedAt.Valid {
		c.JSON(http.StatusOK, gin.H{
			"status":      "already_deleted",
			"deletedAt":   user.DeletedAt.Time,
			"retainUntil": user.DeletedAt.Time.Add(dpdpRetentionWindow),
		})
		return user, false
	}
	return user, true
}

// finalizeSelfDelete soft-deletes the user, runs the role-specific cascade
// (taking the account's public surfaces offline), records the erasure in the
// audit trail (user id only — never PII), and writes the success response.
// auditAction is the audit action string (e.g. "customer.account.delete").
func finalizeSelfDelete(c *gin.Context, user models.User, auditAction string, cascade func()) {
	if err := database.DB.Delete(&user).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed; please contact support"})
		return
	}
	if cascade != nil {
		cascade()
	}

	now := time.Now().UTC()
	retainUntil := now.Add(dpdpRetentionWindow)

	// Erasure path — log the user id only, never the email.
	log.Printf("DPDP delete: action=%s user=%s at=%s", auditAction, user.ID, now.Format(time.RFC3339))
	services.LogAudit(c, auditAction, "user", user.ID.String(),
		nil, gin.H{"deletedAt": now, "retainUntil": retainUntil})

	c.JSON(http.StatusOK, gin.H{
		"status":      "deleted",
		"deletedAt":   now,
		"retainUntil": retainUntil,
		"notice":      "Your account is now hidden. Data will be permanently erased after the 30-day retention window. Contact support to cancel within this window.",
	})
}
