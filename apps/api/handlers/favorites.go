package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

const maxFavoriteChefs = 7

// Dishes are far more numerous than chefs (a curated max of 7), so the cap is
// generous — high enough to never get in a power user's way, low enough to
// bound the favorites query and per-user row count (#237).
const maxFavoriteDishes = 100

type FavoriteHandler struct{}

func NewFavoriteHandler() *FavoriteHandler {
	return &FavoriteHandler{}
}

// ListFavoriteChefs returns the authenticated user's favorite chefs.
func (h *FavoriteHandler) ListFavoriteChefs(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var favorites []models.FavoriteChef
	if err := database.DB.Preload("Chef").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch favorites"})
		return
	}

	responses := make([]models.FavoriteChefResponse, len(favorites))
	for i, fav := range favorites {
		responses[i] = models.FavoriteChefResponse{
			ID:        fav.ID,
			ChefID:    fav.ChefID,
			Chef:      fav.Chef.ToResponse(),
			CreatedAt: fav.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"count": len(responses),
		"max":   maxFavoriteChefs,
	})
}

// ListFavoriteChefIDs returns just the chef IDs for quick lookup (used by frontend).
func (h *FavoriteHandler) ListFavoriteChefIDs(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var ids []uuid.UUID
	database.DB.Model(&models.FavoriteChef{}).
		Where("user_id = ?", userID).
		Pluck("chef_id", &ids)

	c.JSON(http.StatusOK, gin.H{"chefIds": ids})
}

// AddFavoriteChef adds a chef to the user's favorites.
func (h *FavoriteHandler) AddFavoriteChef(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		ChefID string `json:"chefId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "chefId is required"})
		return
	}

	chefID, err := uuid.Parse(req.ChefID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chefId"})
		return
	}

	// Verify chef exists
	var chefCount int64
	database.DB.Model(&models.ChefProfile{}).Where("id = ?", chefID).Count(&chefCount)
	if chefCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	// Check current count
	var currentCount int64
	database.DB.Model(&models.FavoriteChef{}).Where("user_id = ?", userID).Count(&currentCount)
	if currentCount >= maxFavoriteChefs {
		c.JSON(http.StatusConflict, gin.H{
			"error": "Maximum 7 favorite chefs allowed. Remove one before adding another.",
			"max":   maxFavoriteChefs,
			"count": currentCount,
		})
		return
	}

	// Check for duplicate
	var existing models.FavoriteChef
	if err := database.DB.Where("user_id = ? AND chef_id = ?", userID, chefID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Chef is already in your favorites"})
		return
	}

	fav := models.FavoriteChef{
		UserID: userID,
		ChefID: chefID,
	}
	if err := database.DB.Create(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add favorite"})
		return
	}

	// Load chef for response
	database.DB.Preload("Chef").First(&fav, fav.ID)

	c.JSON(http.StatusCreated, models.FavoriteChefResponse{
		ID:        fav.ID,
		ChefID:    fav.ChefID,
		Chef:      fav.Chef.ToResponse(),
		CreatedAt: fav.CreatedAt,
	})
}

// RemoveFavoriteChef removes a chef from the user's favorites.
func (h *FavoriteHandler) RemoveFavoriteChef(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	chefIDParam := c.Param("chefId")

	chefID, err := uuid.Parse(chefIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chefId"})
		return
	}

	result := database.DB.Where("user_id = ? AND chef_id = ?", userID, chefID).Delete(&models.FavoriteChef{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Favorite not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from favorites"})
}

// ─────────────────────────────────────────────────────────────────────────────
// Favorite dishes (#237) — saved individual menu items. Mirrors the chef
// favorites above; the only differences are the higher cap and that each saved
// dish carries a lightweight chef summary so the Favorites surface can render
// "from <chef>" + link without a second round-trip.
// ─────────────────────────────────────────────────────────────────────────────

// ListFavoriteDishes returns the authenticated user's favorite dishes.
func (h *FavoriteHandler) ListFavoriteDishes(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var favorites []models.FavoriteDish
	if err := database.DB.
		Preload("MenuItem").
		Preload("MenuItem.Chef").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&favorites).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch favorite dishes"})
		return
	}

	responses := make([]models.FavoriteDishResponse, 0, len(favorites))
	for _, fav := range favorites {
		// The Preload skips soft-deleted dishes, leaving a zero-valued MenuItem.
		// Skip those so a deleted dish doesn't render as a blank ₹0 row.
		if fav.MenuItem.ID == uuid.Nil {
			continue
		}
		responses = append(responses, models.FavoriteDishResponse{
			ID:         fav.ID,
			MenuItemID: fav.MenuItemID,
			MenuItem:   fav.MenuItem.ToResponse(),
			Chef: models.FavoriteDishChef{
				ID:           fav.MenuItem.Chef.ID,
				BusinessName: fav.MenuItem.Chef.BusinessName,
				ProfileImage: fav.MenuItem.Chef.ProfileImage,
			},
			CreatedAt: fav.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  responses,
		"count": len(responses),
		"max":   maxFavoriteDishes,
	})
}

// ListFavoriteDishIDs returns just the menu-item IDs for quick lookup (used by
// the client to mark hearts as filled without loading full dish objects).
func (h *FavoriteHandler) ListFavoriteDishIDs(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var ids []uuid.UUID
	database.DB.Model(&models.FavoriteDish{}).
		Where("user_id = ?", userID).
		Pluck("menu_item_id", &ids)

	c.JSON(http.StatusOK, gin.H{"menuItemIds": ids})
}

// AddFavoriteDish saves a menu item to the user's favorites.
func (h *FavoriteHandler) AddFavoriteDish(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req struct {
		MenuItemID string `json:"menuItemId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "menuItemId is required"})
		return
	}

	menuItemID, err := uuid.Parse(req.MenuItemID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menuItemId"})
		return
	}

	// Verify the dish exists (soft-deleted rows are excluded by GORM scope).
	var itemCount int64
	database.DB.Model(&models.MenuItem{}).Where("id = ?", menuItemID).Count(&itemCount)
	if itemCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dish not found"})
		return
	}

	var currentCount int64
	database.DB.Model(&models.FavoriteDish{}).Where("user_id = ?", userID).Count(&currentCount)
	if currentCount >= maxFavoriteDishes {
		c.JSON(http.StatusConflict, gin.H{
			"error": "You've reached the maximum number of saved dishes. Remove one before adding another.",
			"max":   maxFavoriteDishes,
			"count": currentCount,
		})
		return
	}

	var existing models.FavoriteDish
	if err := database.DB.Where("user_id = ? AND menu_item_id = ?", userID, menuItemID).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Dish is already in your favorites"})
		return
	}

	fav := models.FavoriteDish{UserID: userID, MenuItemID: menuItemID}
	if err := database.DB.Create(&fav).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add favorite dish"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"id": fav.ID, "menuItemId": fav.MenuItemID, "createdAt": fav.CreatedAt})
}

// RemoveFavoriteDish removes a menu item from the user's favorites.
func (h *FavoriteHandler) RemoveFavoriteDish(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	menuItemIDParam := c.Param("menuItemId")

	menuItemID, err := uuid.Parse(menuItemIDParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid menuItemId"})
		return
	}

	result := database.DB.Where("user_id = ? AND menu_item_id = ?", userID, menuItemID).Delete(&models.FavoriteDish{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Favorite not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Removed from favorites"})
}
