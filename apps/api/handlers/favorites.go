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
