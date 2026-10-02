package handlers

import (
	"errors"
	"gorm.io/gorm"
	"math"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

// CartAvailability lists saved-cart vendors within the customer's selected address radius.
func (h *AddressHandler) CartAvailability(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}
	var req struct {
		ChefIDs []uuid.UUID `json:"chefIds" binding:"max=100"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid cart vendors"})
		return
	}
	db := database.DB.WithContext(c.Request.Context())
	var address models.Address
	if err := db.Where("id = ? AND user_id = ?", id, userID).First(&address).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to check cart availability"})
		}
		return
	}
	ids := make([]uuid.UUID, 0)
	if !cartCoordinatesKnown(address.Latitude, address.Longitude) || len(req.ChefIDs) == 0 {
		c.JSON(http.StatusOK, gin.H{"chefIds": ids})
		return
	}
	var chefs []models.ChefProfile
	if err := db.Where("id IN ? AND is_active = ? AND is_verified = ?", req.ChefIDs, true, true).Find(&chefs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to check cart availability"})
		return
	}
	for _, chef := range chefs {
		visible, reduced := chefVisibleTo(c, &chef)
		if !visible || reduced || !strings.EqualFold(cartCountry(chef.PayoutCountry), cartCountry(address.Country)) || !cartCoordinatesKnown(chef.Latitude, chef.Longitude) {
			continue
		}
		radius := chef.ServiceRadius
		if radius <= 0 {
			radius = services.EffectiveDeliveryMaxKm(chef)
		}
		if services.PlanarDistanceKm(chef.Latitude, chef.Longitude, address.Latitude, address.Longitude) <= radius {
			ids = append(ids, chef.ID)
		}
	}
	c.JSON(http.StatusOK, gin.H{"chefIds": ids})
}

func cartCountry(country string) string {
	if country == "" {
		return "IN"
	}
	return country
}

func cartCoordinatesKnown(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && math.Abs(lat) <= 90 && math.Abs(lng) <= 180 && (lat != 0 || lng != 0)
}
