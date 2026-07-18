package handlers

// admin_delivery_intelligence.go — the cost/usage read model for the
// delivery-intelligence feature (#699), surfaced in the tesserix-home admin
// console. It answers three operational questions the platform owner asked for:
//
//   - REQUESTS  — how many paid routing/weather calls are we making, and how
//                 often is a cache hit sparing us one?
//   - TIER      — how are delivery zones configured (tier, fares, thresholds)?
//   - EXPENSES  — what has that cost us, live and all-time?
//
// It is read-only: zone CRUD already lives at /admin/delivery/zones. Everything
// here is derived from the live counters (services.GetDeliveryUsageSnapshot) and
// the durable delivery_distance_cache table, so it adds no new writes.

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/homechef/api/database"
	"github.com/homechef/api/models"
	"github.com/homechef/api/services"
)

type zoneTierSummary struct {
	Tier            string  `json:"tier"`
	Count           int64   `json:"count"`
	AvgBaseFare     float64 `json:"avgBaseFare"`
	AvgPerKmRate    float64 `json:"avgPerKmRate"`
	AvgMinimumFare  float64 `json:"avgMinimumFare"`
	AvgSurge        float64 `json:"avgSurgeMultiplier"`
	ActiveZoneCount int64   `json:"activeZoneCount"`
}

type deliveryIntelligenceResponse struct {
	// REQUESTS + EXPENSES (live, since restart) from the in-process counters.
	Usage services.DeliveryUsageSnapshot `json:"usage"`

	// EXPENSES (all-time, durable): every row is one chef→address trip we paid the
	// routing provider for exactly once. Row count × per-call price is the total
	// distance spend to date — and every future order on those trips is free.
	CachedTrips             int64   `json:"cachedTrips"`
	AllTimeDistanceSpendUSD float64 `json:"allTimeDistanceSpendUsd"`

	// TIER config, rolled up so the admin sees the pricing shape at a glance.
	ZoneTiers []zoneTierSummary `json:"zoneTiers"`
}

// GetDeliveryIntelligence returns the delivery-intelligence cost/usage summary
// for the admin console. Read-only; safe to poll.
func (h *AdminHandler) GetDeliveryIntelligence(c *gin.Context) {
	snap := services.GetDeliveryUsageSnapshot()

	var cachedTrips int64
	database.DB.Model(&models.DeliveryDistanceCache{}).Count(&cachedTrips)

	// Roll up zones by tier. GORM aggregate — one grouped query, no N+1.
	var tiers []zoneTierSummary
	database.DB.Model(&models.DeliveryZone{}).
		Select(`tier,
			COUNT(*) as count,
			AVG(base_fare) as avg_base_fare,
			AVG(per_km_rate) as avg_per_km_rate,
			AVG(minimum_fare) as avg_minimum_fare,
			AVG(surge_multiplier) as avg_surge,
			SUM(CASE WHEN is_active THEN 1 ELSE 0 END) as active_zone_count`).
		Group("tier").
		Order("tier").
		Scan(&tiers)

	c.JSON(http.StatusOK, deliveryIntelligenceResponse{
		Usage:                   snap,
		CachedTrips:             cachedTrips,
		AllTimeDistanceSpendUSD: float64(cachedTrips) * snap.DistancePricePerCall,
		ZoneTiers:               tiers,
	})
}
