package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// dietary.go — dietary & allergen profile endpoints (#41). The taxonomy and the
// conflict matcher live in services/dietary.go so the web/mobile clients and any
// server-side check share one definition.

type DietaryHandler struct{}

func NewDietaryHandler() *DietaryHandler { return &DietaryHandler{} }

// GetDietaryOptions returns the canonical diet + allergen vocabularies so every
// client (profile chips, menu-item form) uses the same tokens. Public.
// GET /dietary/options
func (h *DietaryHandler) GetDietaryOptions(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"diets":     services.DietOptions,
		"allergens": services.AllergenOptions,
	})
}

type dietaryCheckRequest struct {
	MenuItemIDs []uuid.UUID `json:"menuItemIds" binding:"required,min=1"`
}

type dietaryWarning struct {
	MenuItemID uuid.UUID                  `json:"menuItemId"`
	Name       string                     `json:"name"`
	Conflicts  []services.DietaryConflict `json:"conflicts"`
}

// CheckDietary returns, for a set of menu items, any conflicts with the
// authenticated customer's dietary profile (allergens to avoid + diet). Used to
// warn at checkout — non-blocking; the client decides how to surface it.
// POST /dietary/check
func (h *DietaryHandler) CheckDietary(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	var req dietaryCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load the customer's profile (absent profile → no preferences → no warnings).
	var profile models.CustomerProfile
	database.DB.Where("user_id = ?", userID).First(&profile)
	if len(profile.DietaryPreferences) == 0 && len(profile.FoodAllergies) == 0 {
		c.JSON(http.StatusOK, gin.H{"hasConflicts": false, "warnings": []dietaryWarning{}})
		return
	}

	var items []models.MenuItem
	if err := database.DB.Where("id IN ?", req.MenuItemIDs).Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load items"})
		return
	}

	warnings := []dietaryWarning{}
	for i := range items {
		conflicts := services.DietaryConflictsForItem(profile.DietaryPreferences, profile.FoodAllergies, items[i])
		if len(conflicts) > 0 {
			warnings = append(warnings, dietaryWarning{
				MenuItemID: items[i].ID,
				Name:       items[i].Name,
				Conflicts:  conflicts,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"hasConflicts": len(warnings) > 0,
		"warnings":     warnings,
	})
}
