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
	"github.com/homechef/api/services"
)

// daily_menu.go — chef CRUD + public read for the PER-DATE tiffin menu (#405).
// Unlike the fixed weekly menu (weekly_menu.go), each calendar date can carry
// MULTIPLE dishes per slot, so a home chef cooks different things on different
// days. Mirrors the weekly-menu replace-all/publish/outbox patterns.

const dailyMenuMaxItems = 40 // guardrail on dishes per date

type dailyMenuItemInput struct {
	Slot        string   `json:"slot"`    // lunch|dinner
	Variant     string   `json:"variant"` // veg|nonveg
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       float64  `json:"price"`
	ImageURL    string   `json:"imageUrl"`
	DietaryTags []string `json:"dietaryTags"`
	Allergens   []string `json:"allergens"`
	MenuItemID  *string  `json:"menuItemId"`
	// IsThali + ThaliComponents make this entry a priced bundle (#406).
	IsThali         bool     `json:"isThali"`
	ThaliComponents []string `json:"thaliComponents"`
	SortOrder       int      `json:"sortOrder"`
}

type dailyMenuUpsertRequest struct {
	IsPublished bool                 `json:"isPublished"`
	Items       []dailyMenuItemInput `json:"items"`
}

type dailyMenuResponse struct {
	Date        string                 `json:"date"` // YYYY-MM-DD
	IsPublished bool                   `json:"isPublished"`
	PublishedAt *time.Time             `json:"publishedAt,omitempty"`
	Items       []models.DailyMenuItem `json:"items"`
}

// parseDailyRange resolves the ?from&to query window (YYYY-MM-DD, IST). Defaults
// to today..+14d when unset, matching the customer booking horizon.
func parseDailyRange(c *gin.Context) (time.Time, time.Time, bool) {
	fromStr, toStr := c.Query("from"), c.Query("to")
	if fromStr == "" && toStr == "" {
		now := time.Now().In(istLoc)
		from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, istLoc)
		return from, from.AddDate(0, 0, 14), true
	}
	from, e1 := parsePlanDate(fromStr)
	to, e2 := parsePlanDate(toStr)
	if e1 != nil || e2 != nil {
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

// loadDailyMenus returns the chef's day menus in [from, to], published-only when
// requested (the public read). Items are ordered slot → sortOrder → name.
func loadDailyMenus(chefID uuid.UUID, from, to time.Time, publishedOnly bool) []dailyMenuResponse {
	q := database.DB.Where("chef_id = ? AND date >= ? AND date <= ?", chefID, from, to)
	if publishedOnly {
		q = q.Where("is_published = ?", true)
	}
	var menus []models.DailyMenu
	q.Order("date").Find(&menus)

	out := make([]dailyMenuResponse, 0, len(menus))
	for _, m := range menus {
		var items []models.DailyMenuItem
		database.DB.Where("daily_menu_id = ?", m.ID).Order("slot, sort_order, name").Find(&items)
		out = append(out, dailyMenuResponse{
			Date:        m.Date.Format("2006-01-02"),
			IsPublished: m.IsPublished,
			PublishedAt: m.PublishedAt,
			Items:       items,
		})
	}
	return out
}

// PutDailyMenu replaces the authed chef's dishes for one date and sets its publish
// state. Replace-all keeps the stored set in sync with the editor, exactly like
// PutWeeklyMenu — but a date may hold multiple dishes per slot.
// PUT /chef/daily-menu/:date
func (h *ChefHandler) PutDailyMenu(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	date, derr := parsePlanDate(c.Param("date"))
	if derr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "date must be YYYY-MM-DD"})
		return
	}

	var req dailyMenuUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.Items) > dailyMenuMaxItems {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many dishes for one day"})
		return
	}

	items := make([]models.DailyMenuItem, 0, len(req.Items))
	for _, in := range req.Items {
		if !validSlot(in.Slot) || !validVariant(in.Variant) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each dish needs slot lunch|dinner and variant veg|nonveg"})
			return
		}
		if in.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each dish needs a name"})
			return
		}
		if in.Price < 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "price cannot be negative"})
			return
		}
		// A thali is a bundle: it must list at least two component dishes and
		// carry a set price (customers book it as one priced choice).
		if in.IsThali && (len(in.ThaliComponents) < 2 || in.Price <= 0) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "a thali needs at least two components and a set price"})
			return
		}
		var menuItemID *uuid.UUID
		if in.MenuItemID != nil && *in.MenuItemID != "" {
			if id, perr := uuid.Parse(*in.MenuItemID); perr == nil {
				menuItemID = &id
			}
		}
		items = append(items, models.DailyMenuItem{
			ChefID:          chef.ID,
			Date:            date,
			Slot:            models.MealSlot(in.Slot),
			Variant:         models.MealVariant(in.Variant),
			Name:            in.Name,
			Description:     in.Description,
			Price:           in.Price,
			ImageURL:        in.ImageURL,
			DietaryTags:     ensureStringArray(in.DietaryTags),
			Allergens:       ensureStringArray(in.Allergens),
			MenuItemID:      menuItemID,
			IsThali:         in.IsThali,
			ThaliComponents: ensureStringArray(in.ThaliComponents),
			SortOrder:       in.SortOrder,
		})
	}
	if req.IsPublished && len(items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "add at least one dish before publishing"})
		return
	}

	now := time.Now()
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var menu models.DailyMenu
		tx.Where("chef_id = ? AND date = ?", chef.ID, date).FirstOrInit(&menu)
		wasPublished := menu.IsPublished
		menu.ChefID = chef.ID
		menu.Date = date
		menu.IsPublished = req.IsPublished
		if req.IsPublished {
			menu.PublishedAt = &now
		}
		if err := tx.Save(&menu).Error; err != nil {
			return err
		}
		// Replace-all the dishes for this (chef, date).
		if err := tx.Where("chef_id = ? AND date = ?", chef.ID, date).Delete(&models.DailyMenuItem{}).Error; err != nil {
			return err
		}
		if len(items) > 0 {
			for i := range items {
				items[i].DailyMenuID = menu.ID
			}
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}
		// Notify followers on the unpublished→published transition (menu drop),
		// staged in the same tx via the outbox — mirrors the weekly-menu publish.
		if req.IsPublished && !wasPublished {
			return services.EnqueueEvent(tx, services.SubjectDailyMenuPublished, "daily_menu_published", chef.UserID, map[string]any{
				"chef_id":   chef.ID.String(),
				"chef_name": chef.BusinessName,
				"date":      date.Format("2006-01-02"),
			})
		}
		return nil
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save daily menu"})
		return
	}

	c.JSON(http.StatusOK, dailyMenuResponse{
		Date:        date.Format("2006-01-02"),
		IsPublished: req.IsPublished,
		Items:       items,
	})
}

// GetMyDailyMenu returns the authed chef's per-date menus (incl. drafts) in range.
// GET /chef/daily-menu?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *ChefHandler) GetMyDailyMenu(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	var chef models.ChefProfile
	if err := database.DB.Where("user_id = ?", userID).First(&chef).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	from, to, ok := parseDailyRange(c)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from/to must be YYYY-MM-DD"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": loadDailyMenus(chef.ID, from, to, false)})
}

// GetPublicDailyMenu returns a chef's PUBLISHED per-date menus in range (by UUID
// or slug). Unpublished dates are hidden.
// GET /chefs/:id/daily-menu?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *ChefHandler) GetPublicDailyMenu(c *gin.Context) {
	chefID, ok := resolveChefID(c.Param("id"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}
	from, to, ok2 := parseDailyRange(c)
	if !ok2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from/to must be YYYY-MM-DD"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"days": loadDailyMenus(chefID, from, to, true)})
}
