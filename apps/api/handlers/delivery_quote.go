package handlers

// delivery_quote.go — checkout delivery-fee preview (#pickup-incentive).
//
// The customer app used to show "Delivery fee — Free" for every mode, hiding
// both the real fee and the saving from picking up. To show either honestly the
// app needs the fee BEFORE placing the order — and it must be the SAME number
// CreateOrder will charge, or the customer agrees to one total and is billed
// another. This endpoint returns exactly what CreateOrder computes, because both
// call services.QuoteOrderDeliveryFee.

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type deliveryQuoteRequest struct {
	// The drop coordinates + city drive the 3PL / self-delivery distance fee.
	// Optional: without them the flat policy fee is returned, matching
	// CreateOrder's own fallback.
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
}

// QuoteDeliveryFee returns the per-mode delivery fee for a chef + drop address,
// so checkout can show the real delivery cost and pickup's saving.
//
// POST /v1/chefs/:id/delivery-quote
func (h *OrderHandler) QuoteDeliveryFee(c *gin.Context) {
	chefID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid chef id"})
		return
	}
	var req deliveryQuoteRequest
	_ = c.ShouldBindJSON(&req) // all fields optional; a bad body just means "no coords"

	var chef models.ChefProfile
	if err := database.DB.First(&chef, "id = ?", chefID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Chef not found"})
		return
	}

	country := req.Country
	if country == "" {
		country = "IN"
	}

	// A customer's "delivery" request is always created as plain delivery (the
	// chef picks the carrier later at Mark Ready), so the fee it will be charged
	// is the delivery-mode fee — NOT chef_delivery. Quoting chef_delivery here
	// would show a number the order never uses.
	deliveryFee := services.QuoteOrderDeliveryFee(chef, models.FulfillmentDelivery, req.Latitude, req.Longitude, req.City, country)
	// Pickup is always free — this zero is the saving the app advertises.
	const pickupFee = 0.0

	saving := deliveryFee - pickupFee
	if saving < 0 {
		saving = 0
	}

	c.JSON(http.StatusOK, gin.H{
		"deliveryFee": models.RoundAmount(deliveryFee),
		"pickupFee":   pickupFee,
		// pickupSaving is what the customer keeps by collecting. 0 when delivery is
		// itself free — the app then shows no incentive rather than a fake one.
		"pickupSaving": models.RoundAmount(saving),
		"currency":     services.CurrencyForCountry(chef.PayoutCountry),
		"offersPickup": chef.OffersPickup,
	})
}
