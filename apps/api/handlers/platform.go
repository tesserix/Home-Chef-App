package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/homechef/api/middleware"
	"github.com/homechef/api/services"
)

// PlatformHandler owns the commerce-side platform policy (commission, fees,
// operating hours) and a public read endpoint that checkouts hit to show
// live fees + closed-hours messaging.
type PlatformHandler struct{}

func NewPlatformHandler() *PlatformHandler { return &PlatformHandler{} }

// publicPlatformConfig is the shape we expose publicly — a safe subset of
// the PlatformPolicy, omitting internal payout percentages.
type publicPlatformConfig struct {
	ServiceFeePercent float64 `json:"serviceFeePercent"`
	TaxPercent        float64 `json:"taxPercent"`
	BaseDeliveryFee   float64 `json:"baseDeliveryFee"`
	Timezone          string  `json:"timezone"`
	OpeningTime       string  `json:"openingTime"`
	ClosingTime       string  `json:"closingTime"`
	OperatingDays     []int   `json:"operatingDays"`
	IsOpen            bool    `json:"isOpen"`
	ClosedMessage     string  `json:"closedMessage,omitempty"`
}

// CheckZoneCoverage lets a checkout preflight whether an address is in a
// supported delivery zone without having to submit the whole order first.
// Query params: lat + lon (floats). Returns { covered: bool, zone?: {...} }.
func (h *PlatformHandler) CheckZoneCoverage(c *gin.Context) {
	if !services.HasActiveZones() {
		// No zones configured → treat everywhere as covered (feature off).
		c.JSON(http.StatusOK, gin.H{"covered": true, "configured": false})
		return
	}
	var req struct {
		Latitude  float64 `form:"lat" binding:"required"`
		Longitude float64 `form:"lon" binding:"required"`
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "lat and lon required"})
		return
	}
	zone := services.FindZoneForAddress(req.Latitude, req.Longitude)
	if zone == nil {
		c.JSON(http.StatusOK, gin.H{"covered": false, "configured": true})
		return
	}
	resp := zone.ToResponse()
	c.JSON(http.StatusOK, gin.H{"covered": true, "configured": true, "zone": resp})
}

// GetPublicConfig serves the subset of PlatformPolicy that customer-facing
// checkouts need so fees are consistent and "we're closed" can render
// without a round-trip that fails at order submit.
func (h *PlatformHandler) GetPublicConfig(c *gin.Context) {
	p := services.GetPlatformPolicy()
	open, msg := services.IsPlatformOpen()
	resp := publicPlatformConfig{
		ServiceFeePercent: p.ServiceFeePercent,
		TaxPercent:        p.TaxPercent,
		BaseDeliveryFee:   p.BaseDeliveryFee,
		Timezone:          p.Timezone,
		OpeningTime:       p.OpeningTime,
		ClosingTime:       p.ClosingTime,
		OperatingDays:     p.OperatingDays,
		IsOpen:            open,
	}
	if !open {
		resp.ClosedMessage = msg
	}
	c.JSON(http.StatusOK, resp)
}

// AdminGetPolicy returns the full policy — admin-only because it includes
// payout percentages a customer shouldn't see.
func (h *PlatformHandler) AdminGetPolicy(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetPlatformPolicy())
}

// AdminUpdatePolicy upserts the policy. Every field is optional; unspecified
// keys keep their current values so the UI can do per-section saves.
func (h *PlatformHandler) AdminUpdatePolicy(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	current := services.GetPlatformPolicy()

	var req map[string]any
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if v, ok := req["serviceFeePercent"].(float64); ok {
		current.ServiceFeePercent = v
	}
	if v, ok := req["taxPercent"].(float64); ok {
		current.TaxPercent = v
	}
	if v, ok := req["baseDeliveryFee"].(float64); ok {
		current.BaseDeliveryFee = v
	}
	if v, ok := req["perKmDeliveryFee"].(float64); ok {
		current.PerKmDeliveryFee = v
	}
	if v, ok := req["chefPayoutPercent"].(float64); ok {
		current.ChefPayoutPercent = v
	}
	if v, ok := req["driverPayoutPercent"].(float64); ok {
		current.DriverPayoutPercent = v
	}
	// Empty timezone would silently break IsPlatformOpen, so require a
	// non-empty value to overwrite. Opening/closing times empty is a valid
	// "disable the hours gate" signal, so those accept "".
	if v, ok := req["timezone"].(string); ok && v != "" {
		current.Timezone = v
	}
	if v, ok := req["openingTime"].(string); ok {
		current.OpeningTime = v
	}
	if v, ok := req["closingTime"].(string); ok {
		current.ClosingTime = v
	}
	if v, ok := req["closedMessage"].(string); ok && v != "" {
		current.ClosedMessage = v
	}
	if v, ok := req["operatingDays"].([]any); ok {
		days := make([]int, 0, len(v))
		for _, d := range v {
			if f, ok := d.(float64); ok {
				days = append(days, int(f))
			}
		}
		current.OperatingDays = days
	}

	if err := services.SavePlatformPolicy(current, &userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	services.LogAudit(c, "platform.policy.update", "platform_policy", "", nil, current)
	c.JSON(http.StatusOK, current)
}
