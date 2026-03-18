package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/homechef/api/database"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/models"
)

type AddressHandler struct{}

func NewAddressHandler() *AddressHandler {
	return &AddressHandler{}
}

func (h *AddressHandler) GetAddresses(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var addresses []models.Address
	if err := database.DB.Where("user_id = ?", userID).Order("is_default DESC, created_at ASC").Find(&addresses).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch addresses"})
		return
	}

	c.JSON(http.StatusOK, addresses)
}

type createAddressRequest struct {
	Label      string  `json:"label" binding:"required"`
	Line1      string  `json:"line1" binding:"required"`
	Line2      string  `json:"line2"`
	City       string  `json:"city" binding:"required"`
	State      string  `json:"state" binding:"required"`
	PostalCode string  `json:"postalCode" binding:"required"`
	Country    string  `json:"country"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	IsDefault  bool    `json:"isDefault"`
}

func (h *AddressHandler) CreateAddress(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)

	var req createAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	country := req.Country
	if country == "" {
		country = "IN"
	}

	address := models.Address{
		UserID:     userID,
		Label:      req.Label,
		Line1:      req.Line1,
		Line2:      req.Line2,
		City:       req.City,
		State:      req.State,
		PostalCode: req.PostalCode,
		Country:    country,
		Latitude:   req.Latitude,
		Longitude:  req.Longitude,
		IsDefault:  req.IsDefault,
	}

	// If this is the default, unset other defaults
	if req.IsDefault {
		database.DB.Model(&models.Address{}).Where("user_id = ?", userID).Update("is_default", false)
	}

	if err := database.DB.Create(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create address"})
		return
	}

	c.JSON(http.StatusCreated, address)
}

func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	addressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	var address models.Address
	if err := database.DB.Where("id = ? AND user_id = ?", addressID, userID).First(&address).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}

	var req createAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.IsDefault && !address.IsDefault {
		database.DB.Model(&models.Address{}).Where("user_id = ? AND id != ?", userID, addressID).Update("is_default", false)
	}

	address.Label = req.Label
	address.Line1 = req.Line1
	address.Line2 = req.Line2
	address.City = req.City
	address.State = req.State
	address.PostalCode = req.PostalCode
	if req.Country != "" {
		address.Country = req.Country
	}
	address.Latitude = req.Latitude
	address.Longitude = req.Longitude
	address.IsDefault = req.IsDefault

	if err := database.DB.Save(&address).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	c.JSON(http.StatusOK, address)
}

func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	userID, _ := middleware.GetUserID(c)
	addressID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	result := database.DB.Where("id = ? AND user_id = ?", addressID, userID).Delete(&models.Address{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete address"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address deleted"})
}
