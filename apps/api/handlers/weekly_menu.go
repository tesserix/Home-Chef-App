package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
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
	// Thali/combo: when true the cell is a bundled set (ComboComponents) at the
	// one Price, rather than a single dish.
	IsCombo         bool     `json:"isCombo"`
	ComboComponents []string `json:"comboComponents"`
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
			DietaryTags:     ensureStringArray(in.DietaryTags),
			Allergens:       ensureStringArray(in.Allergens),
			MenuItemID:      menuItemID,
			IsCombo:         in.IsCombo,
			ComboComponents: ensureStringArray(in.ComboComponents),
		})
	}

	// A publishable week must have every offered (day × slot) filled — no holes
	// (#1). Drafts can be saved incomplete; only publishing enforces this.
	if req.IsPublished {
		if err := validatePublishableGrid(cells); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
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
		wasPublished := menu.IsPublished
		menu.ChefID = chef.ID
		menu.IsPublished = req.IsPublished
		if req.IsPublished {
			menu.PublishedAt = &now
		}
		if err := tx.Save(&menu).Error; err != nil {
			return err
		}
		// Notify followers on the publish transition — a "menu drop" (#239).
		// Staged in the SAME tx via the transactional outbox so it's delivered
		// exactly once and never fires on a rolled-back save. Transition-gated
		// (was unpublished → now published) so editing an already-live menu
		// doesn't re-spam followers on every save.
		if req.IsPublished && !wasPublished {
			if err := services.EnqueueEvent(tx, services.SubjectWeeklyMenuPublished, "weekly_menu_published", chef.UserID, map[string]any{
				"chef_id":   chef.ID.String(),
				"chef_name": chef.BusinessName,
			}); err != nil {
				return err
			}
		}
		return nil
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

var weekdayShort = [7]string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}

// validatePublishableGrid enforces the #1 rule: a publishable weekly menu must
// have "every offered (day × slot) filled" — no holes. It treats the days the
// chef actually used and the slots they actually used as the offered grid, then
// requires every (day × slot) in that rectangle to have at least one cell (any
// variant). So a lunch-only or a Mon–Wed menu is fine, but filling Tue lunch
// while leaving Tue dinner blank (when other days have dinner) is rejected. The
// returned error names the first missing cell. Pure + reusable so the vendor
// editors can mirror the same check client-side.
func validatePublishableGrid(cells []models.WeeklyMenuItem) error {
	if len(cells) == 0 {
		return fmt.Errorf("add at least one dish before publishing")
	}

	daySet := map[int]bool{}
	slotSet := map[models.MealSlot]bool{}
	present := map[string]bool{}
	for _, c := range cells {
		daySet[c.DayOfWeek] = true
		slotSet[c.Slot] = true
		present[fmt.Sprintf("%d|%s", c.DayOfWeek, c.Slot)] = true
	}

	// Deterministic ordering for a stable error message: day 0..6, lunch before dinner.
	slotsInOrder := []models.MealSlot{models.MealSlotLunch, models.MealSlotDinner}
	for day := 0; day < 7; day++ {
		if !daySet[day] {
			continue
		}
		for _, slot := range slotsInOrder {
			if !slotSet[slot] {
				continue
			}
			if !present[fmt.Sprintf("%d|%s", day, slot)] {
				return fmt.Errorf("every offered day needs a %s dish — %s is missing its %s dish", slot, weekdayShort[day], slot)
			}
		}
	}
	return nil
}
