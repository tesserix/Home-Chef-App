package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

// weekly_menu.go — chef CRUD + public read for the fixed weekly menu (#192). The
// tiffin meal-plan (#193) resolves each booked (day × slot × variant) to a cell here.

type weeklyMenuCellInput struct {
	DayOfWeek   int      `json:"dayOfWeek"` // 0=Sun..6=Sat
	Slot        string   `json:"slot"`      // lunch|dinner
	Variant     string   `json:"variant"`   // veg|nonveg
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	ImageURL    string   `json:"imageUrl"`
	DietaryTags []string `json:"dietaryTags"` // #41
	Allergens   []string `json:"allergens"`   // #41
	MenuItemID  *string  `json:"menuItemId"`
}

type weeklyMenuUpsertRequest struct {
	IsPublished bool                  `json:"isPublished"`
	Items       []weeklyMenuCellInput `json:"items"`
}

func validSlot(s string) bool {
	return s == string(models.MealSlotLunch) || s == string(models.MealSlotDinner)
}
func validVariant(v string) bool {
	return v == string(models.MealVariantVeg) || v == string(models.MealVariantNonVeg)
}

// GetMyWeeklyMenu returns the authed chef's weekly menu (incl. draft).
func (h *ChefHandler) GetMyWeeklyMenu(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	var menu models.WeeklyMenu
	database.DB.Where("chef_id = ?", chef.ID).FirstOrInit(&menu)
	menu.ChefID = chef.ID
	var items []models.WeeklyMenuItem
	database.DB.Where("chef_id = ?", chef.ID).Order("day_of_week, slot, variant").Find(&items)

	c.JSON(http.StatusOK, gin.H{
		"isPublished": menu.IsPublished,
		"publishedAt": menu.PublishedAt,
		"items":       items,
	})
}

// PutWeeklyMenu replaces the authed chef's weekly menu cells and sets publish
// state. Replace-all keeps the editor simple and the stored set in sync with what
// the chef submits.
func (h *ChefHandler) PutWeeklyMenu(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	var req weeklyMenuUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cells := make([]models.WeeklyMenuItem, 0, len(req.Items))
	for _, in := range req.Items {
		if in.DayOfWeek < 0 || in.DayOfWeek > 6 || !validSlot(in.Slot) || !validVariant(in.Variant) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each cell needs dayOfWeek 0-6, slot lunch|dinner, variant veg|nonveg"})
			return
		}
		if in.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each cell needs a dish name"})
			return
		}
		var menuItemID *uuid.UUID
		if in.MenuItemID != nil && *in.MenuItemID != "" {
			if id, perr := uuid.Parse(*in.MenuItemID); perr == nil {
				menuItemID = &id
			}
		}
		cells = append(cells, models.WeeklyMenuItem{
			ChefID:      chef.ID,
			DayOfWeek:   in.DayOfWeek,
			Slot:        models.MealSlot(in.Slot),
			Variant:     models.MealVariant(in.Variant),
			Name:        in.Name,
			Description: in.Description,
			Price:       in.Price,
			ImageURL:    in.ImageURL,
			DietaryTags: ensureStringArray(in.DietaryTags),
			Allergens:   ensureStringArray(in.Allergens),
			MenuItemID:  menuItemID,
		})
	}

	if req.IsPublished && len(cells) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot publish an empty weekly menu"})
		return
	}

	now := time.Now()
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("chef_id = ?", chef.ID).Delete(&models.WeeklyMenuItem{}).Error; err != nil {
			return err
		}
		if len(cells) > 0 {
			if err := tx.Create(&cells).Error; err != nil {
				return err
			}
		}
		var menu models.WeeklyMenu
		tx.Where("chef_id = ?", chef.ID).FirstOrInit(&menu)
		menu.ChefID = chef.ID
		menu.IsPublished = req.IsPublished
		if req.IsPublished {
			menu.PublishedAt = &now
		}
		return tx.Save(&menu).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save weekly menu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"isPublished": req.IsPublished, "items": cells})
}

// GetPublicWeeklyMenu returns a chef's PUBLISHED weekly menu (by UUID or slug).
func (h *ChefHandler) GetPublicWeeklyMenu(c *gin.Context) {
	chefID, ok := resolveChefID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	var menu models.WeeklyMenu
	if err := database.DB.Where("chef_id = ? AND is_published = ?", chefID, true).First(&menu).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"isPublished": false, "items": []models.WeeklyMenuItem{}})
		return
	}
	var items []models.WeeklyMenuItem
	database.DB.Where("chef_id = ?", chefID).Order("day_of_week, slot, variant").Find(&items)
	c.JSON(http.StatusOK, gin.H{"isPublished": true, "publishedAt": menu.PublishedAt, "items": items})
}
