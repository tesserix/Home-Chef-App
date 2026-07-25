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

// ChefDPDPHandler implements the data-subject access endpoints
// required by India's Digital Personal Data Protection Act 2023:
//   - Right to Access  → GET /chef/me/export   (JSON dump of all data)
//
// Deletion is NOT here: it lives in account_lifecycle.go, which adds blocker
// checks, the 180-day restore window and GIP credential teardown.
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
		// Orders are mapped through ToChefResponse so the export carries only the
		// data the chef is already entitled to see — the customer's exact delivery
		// address, geo and delivery instructions are redacted (area/first name
		// only), never dumped raw into the chef's export.
		var chefOrders []models.Order
		database.DB.Where("chef_id = ?", chef.ID).
			Preload("Items").Preload("Customer").Find(&chefOrders)
		orderExports := make([]models.OrderResponse, 0, len(chefOrders))
		for i := range chefOrders {
			orderExports = append(orderExports, chefOrders[i].ToChefResponse())
		}
		dump["orders"] = orderExports
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
		"id":        u.ID,
		"email":     u.Email,
		"firstName": u.FirstName,
		"lastName":  u.LastName,
		"phone":     u.Phone,
		"role":      u.Role,
		"createdAt": u.CreatedAt,
		"updatedAt": u.UpdatedAt,
	}
}
