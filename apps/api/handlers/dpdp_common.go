package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// dpdp_common.go — the shared scaffolding behind the DPDP Act 2023 Right to
// Access (export) endpoints. The chef, customer and driver handlers reuse these
// so the export envelope stays identical across roles; only the role-specific
// set of related tables differs per handler.
//
// The erasure half used to live here too. It has been superseded by
// services/account_lifecycle.go, which adds the deletion blockers, the restore
// window (services.RestoreWindow) and GIP credential teardown — none of which the old
// soft-delete-and-hope path had.

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
