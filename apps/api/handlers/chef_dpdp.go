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

// ChefDPDPHandler implements the data-subject access endpoints
// required by India's Digital Personal Data Protection Act 2023:
//   - Right to Access  → GET /chef/me/export   (JSON dump of all data)
//   - Right to Erasure → POST /chef/me/delete  (soft delete + retention queue)
//
// Soft-delete semantics: GORM's DeletedAt column flips, the row stays
// queryable by admin tooling for the retention window (default 30 days
// per the existing privacy policy), then a separate sweeper cron
// purges. Future Wave 4 work adds the sweeper; for now the row sits
// soft-deleted and admins can pull it for legal holds.
type ChefDPDPHandler struct{}

func NewChefDPDPHandler() *ChefDPDPHandler {
	return &ChefDPDPHandler{}
}

// ExportMyData returns a JSON dump of every row associated with the
// authenticated chef's user account. Mirrors the GDPR "data portability"
// concept — the chef should be able to walk away with their data in a
// machine-readable form. Returned as a single JSON document so the
// chef can save it to their device with one tap.
//
// GET /chef/me/export
func (h *ChefDPDPHandler) ExportMyData(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var user models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	dump := map[string]interface{}{
		"exportedAt": time.Now().UTC().Format(time.RFC3339),
		"requestedBy": map[string]interface{}{
			"userId": user.ID.String(),
			"email":  user.Email,
		},
		"notice": "This export contains all personal data Home Chef holds on your account per DPDP Act 2023 §11.",
	}

	dump["user"] = sanitizeUserForExport(user)

	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err == nil {
		dump["chefProfile"] = chef

		// Related tables — pulled per-table so the export survives
		// future schema changes (a new `chef_X` table just needs to be
		// appended here). All filtered by chef_id so other chefs'
		// rows never leak.
		dump["chefDocuments"] = findByChef(chef.ID, &[]models.ChefDocument{})
		dump["chefSchedules"] = findByChef(chef.ID, &[]models.ChefSchedule{})
		dump["chefSettings"] = findByChef(chef.ID, &[]models.ChefSettings{})
		dump["menuItems"] = findByChef(chef.ID, &[]models.MenuItem{})
		dump["orders"] = findByChef(chef.ID, &[]models.Order{})
		dump["reviews"] = findByChef(chef.ID, &[]models.Review{})

		var prefs models.ChefNotificationPreferences
		if err := database.DB.Where("chef_id = ?", chef.ID).First(&prefs).Error; err == nil {
			dump["notificationPreferences"] = prefs
		}
	}

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

// DeleteMyAccount soft-deletes the user + chef profile per the DPDP
// "right to erasure". Hard-delete happens after the retention window
// (separate sweeper cron, Wave 4). Idempotent — re-calling on a
// already-deleted user returns 200 + the same payload so a retry
// from a flaky mobile network is safe.
//
// Required confirmation in the request body — chef must type their
// email exactly to prevent accidental account loss from an
// uncoordinated tap.
//
// POST /chef/me/delete   { "confirmEmail": "<user.email>" }
func (h *ChefDPDPHandler) DeleteMyAccount(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	// Unscoped so an already-soft-deleted account is still found — otherwise
	// GORM's default `deleted_at IS NULL` scope hides it and a retried delete
	// 404s instead of returning the idempotent "already_deleted" below (#106).
	var user models.User
	if err := database.DB.Unscoped().First(&user, "id = ?", userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var req struct {
		ConfirmEmail string `json:"confirmEmail" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.ConfirmEmail != user.Email {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "confirmEmail must exactly match your account email",
		})
		return
	}

	// Already-deleted? Don't error — DPDP retries should be safe.
	if user.DeletedAt.Valid {
		c.JSON(http.StatusOK, gin.H{
			"status":     "already_deleted",
			"deletedAt":  user.DeletedAt.Time,
			"retainUntil": user.DeletedAt.Time.Add(30 * 24 * time.Hour),
		})
		return
	}

	now := time.Now().UTC()

	// Soft delete the user — GORM's DeletedAt mechanism flips the
	// column, list queries auto-filter out, but admin tooling with
	// .Unscoped() can still reach the row during the retention
	// window. Cascade to the chef profile so the kitchen disappears
	// from the marketplace immediately.
	if err := database.DB.Delete(&user).Error; err != nil {
		services.CaptureSentryError(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Delete failed; please contact support"})
		return
	}
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err == nil {
		_ = database.DB.Delete(&chef).Error
		// Take menu items offline so customers don't see a phantom
		// kitchen while the retention window runs out.
		_ = database.DB.Model(&models.MenuItem{}).
			Where("chef_id = ?", chef.ID).
			Update("is_available", false).Error
		// Stop accepting new orders too.
		_ = database.DB.Model(&models.ChefProfile{}).
			Unscoped().
			Where("id = ?", chef.ID).
			Update("accepting_orders", false).Error
	}

	// Log the user id only — never the email. This is the erasure path; the
	// email is PII and the audit row (below) already records the deletion.
	log.Printf("DPDP delete: user=%s at=%s", user.ID, now.Format(time.RFC3339))

	retainUntil := now.Add(30 * 24 * time.Hour)

	// DPDP-significant: record the erasure request (no PII in the row beyond
	// the user id, which is already the audit subject) for the compliance trail.
	services.LogAudit(c, "chef.account.delete", "user", user.ID.String(),
		nil, gin.H{"deletedAt": now, "retainUntil": retainUntil})

	c.JSON(http.StatusOK, gin.H{
		"status":      "deleted",
		"deletedAt":   now,
		"retainUntil": retainUntil,
		"notice":      "Your account is now hidden. Data will be permanently erased after the 30-day retention window. Contact support to cancel within this window.",
	})
}

// findByChef is a tiny generic-ish wrapper around GORM's Find that
// keeps the export function readable. Returns the dest slice via
// reflection of the destination pointer so the caller can drop it
// straight into the JSON map.
func findByChef(chefID interface{}, dest interface{}) interface{} {
	_ = database.DB.Where("chef_id = ?", chefID).Find(dest).Error
	return dest
}

// sanitizeUserForExport strips internal fields (hashed passwords,
// internal flags) before including the User row in the export. The
// data-subject doesn't need their own password hash to exercise
// portability rights, and exporting it adds unnecessary risk.
func sanitizeUserForExport(u models.User) map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID,
		"email":      u.Email,
		"firstName":  u.FirstName,
		"lastName":   u.LastName,
		"phone":      u.Phone,
		"role":       u.Role,
		"createdAt":  u.CreatedAt,
		"updatedAt":  u.UpdatedAt,
	}
}
